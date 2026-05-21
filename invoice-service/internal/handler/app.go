package handler

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"invoice-app/internal/auth"
	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/static"
	"invoice-app/templates"
)

const (
	contextKeyStore contextKey = "store"
	contextKeyUser  contextKey = "user"
)

func StoreFromContext(ctx context.Context) db.Storer {
	s, _ := ctx.Value(contextKeyStore).(db.Storer)
	return s
}

func UserFromContext(ctx context.Context) *db.User {
	u, _ := ctx.Value(contextKeyUser).(*db.User)
	return u
}

// WithTestStore returns a context with the given Storer attached, for use in tests.
func WithTestStore(ctx context.Context, store db.Storer) context.Context {
	return context.WithValue(ctx, contextKeyStore, store)
}

// StoreForTest returns the multi-store for test setup.
func (a *App) StoreForTest() *db.MultiStore {
	return a.multiStore
}

type App struct {
	multiStore  *db.MultiStore
	authDB      *db.AuthDB
	webAuthn    *auth.WebAuthnManager
	sessions    *auth.SessionManager
	cfg         config.Config
	logger      *slog.Logger
	baseTmpl    *template.Template
	ocrWg       sync.WaitGroup
	authEnabled bool
	legacyStore *db.Store
}

type DashboardPageData struct {
	Summary        []db.MonthlySummary
	RecentInvoices []db.InvoiceSummary
	Years          []string
	Months         []string
	SelectedYear   string
	SelectedMonth  string
	TotalHours     float64
	TotalAmount    float64
	Notice         string
	User           *db.User
}

type EntriesPageData struct {
	Entries    []db.Entry
	Categories []string
	Notice     string
	Form       db.Entry
	User       *db.User
}

type RatesPageData struct {
	Rates  []db.Rate
	Notice string
	Form   db.Rate
	User   *db.User
}

type InvoicesPageData struct {
	Invoices           []db.InvoiceSummary
	Categories         []string
	Notice             string
	DefaultMonth       string
	DefaultInvoiceDate string
	DefaultDueDays     int
	SelectedCategory   string
	User               *db.User
}

type SettingsPageData struct {
	Settings db.Settings
	Notice   string
	User     *db.User
}

type InvoicePreviewPageData struct {
	Invoice db.Invoice
	Notice  string
	User    *db.User
}

type LoginPageData struct {
	Notice string
	User   *db.User
}

type RegisterPageData struct {
	Notice string
	User   *db.User
}

func New(multiStore *db.MultiStore, authDB *db.AuthDB, webAuthn *auth.WebAuthnManager, sessions *auth.SessionManager, cfg config.Config, logger *slog.Logger) *App {
	app := &App{
		multiStore:  multiStore,
		authDB:      authDB,
		webAuthn:    webAuthn,
		sessions:    sessions,
		cfg:         cfg,
		logger:      logger,
		authEnabled: cfg.AuthEnabled,
	}
	app.baseTmpl = app.compileTemplates()
	return app
}

func (a *App) compileTemplates() *template.Template {
	funcMap := template.FuncMap{
		"money":       money,
		"numfmt":      numfmt,
		"dateLabel":   dateLabel,
		"selected":    selected,
		"monthName":   monthName,
		"mul":         func(a float64, b float64) float64 { return a * b },
		"divf":        func(a float64, b float64) float64 { return a / b },
		"div":         func(a, b int64) int64 { return a / b },
		"authEnabled": func() bool { return a.authEnabled },
		"ocrEnabled":  func() bool { return a.cfg.OCREnabled },
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templates.FS, "layout.html")
	if err != nil {
		a.logger.Error("compile base template", "error", err)
		panic(fmt.Sprintf("failed to compile base template: %v", err))
	}
	return tmpl
}

func (a *App) WaitForOCR() {
	a.ocrWg.Wait()
}

func (a *App) SetLegacyStore(store *db.Store) {
	a.legacyStore = store
}

func (a *App) renderPage(w http.ResponseWriter, r *http.Request, status int, page string, data any, extra ...string) {
	files := append([]string{"layout.html"}, extra...)
	files = append(files, page)

	tmpl, err := a.baseTmpl.Clone()
	if err != nil {
		http.Error(w, fmt.Sprintf("clone template: %v", err), http.StatusInternalServerError)
		return
	}
	if _, err := tmpl.ParseFS(templates.FS, files...); err != nil {
		http.Error(w, fmt.Sprintf("parse template: %v", err), http.StatusInternalServerError)
		return
	}

	data = injectUser(data, UserFromContext(r.Context()))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, fmt.Sprintf("execute template: %v", err), http.StatusInternalServerError)
	}
}

func injectUser(data any, user *db.User) any {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return data
		}
		elem := v.Elem()
		if elem.Kind() == reflect.Struct {
			f := elem.FieldByName("User")
			if f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf(user) {
				f.Set(reflect.ValueOf(user))
			}
		}
		return data
	}

	if v.Kind() == reflect.Struct {
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(v)
		f := ptr.Elem().FieldByName("User")
		if f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf(user) {
			f.Set(reflect.ValueOf(user))
		}
		return ptr.Interface()
	}
	return data
}

func (a *App) renderPartial(w http.ResponseWriter, status int, name string, data any, files ...string) {
	tmpl, err := a.baseTmpl.Clone()
	if err != nil {
		http.Error(w, fmt.Sprintf("clone template: %v", err), http.StatusInternalServerError)
		return
	}
	if _, err := tmpl.ParseFS(templates.FS, files...); err != nil {
		http.Error(w, fmt.Sprintf("parse partial: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("execute partial: %v", err), http.StatusInternalServerError)
	}
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()

	if a.authEnabled {
		mux.HandleFunc("GET /login", a.loginPage)
		mux.HandleFunc("POST /login/begin", a.beginLogin)
		mux.HandleFunc("POST /login/finish", a.finishLogin)
		mux.HandleFunc("GET /register", a.registerPage)
		mux.HandleFunc("POST /register/begin", a.beginRegistration)
		mux.HandleFunc("POST /register/finish", a.finishRegistration)
		mux.HandleFunc("POST /logout", a.logout)
	}
	mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/static/", http.FileServerFS(static.FS)).ServeHTTP(w, r)
	})
	mux.HandleFunc("GET /", a.dashboard)
	mux.HandleFunc("GET /entries", a.entriesPage)
	mux.HandleFunc("GET /entries/table", a.entriesTable)
	mux.HandleFunc("POST /entries", a.createEntry)
	mux.HandleFunc("GET /entries/{id}/edit", a.editEntryForm)
	mux.HandleFunc("POST /entries/{id}", a.updateEntry)
	mux.HandleFunc("POST /entries/{id}/delete", a.deleteEntry)
	mux.HandleFunc("GET /rates", a.ratesPage)
	mux.HandleFunc("GET /rates/table", a.ratesTable)
	mux.HandleFunc("POST /rates", a.createRate)
	mux.HandleFunc("POST /rates/{id}/delete", a.deleteRate)
	mux.HandleFunc("GET /rates/{id}/edit", a.editRateForm)
	mux.HandleFunc("POST /rates/{id}", a.updateRate)
	mux.HandleFunc("GET /invoices", a.invoicesPage)
	mux.HandleFunc("POST /invoices/generate", a.generateInvoice)
	mux.HandleFunc("GET /invoices/{id}", a.invoicePreview)
	mux.HandleFunc("GET /invoices/{id}/pdf", a.invoicePDF)
	mux.HandleFunc("POST /invoices/{id}/send", a.sendInvoice)
	mux.HandleFunc("GET /settings", a.settingsPage)
	mux.HandleFunc("POST /settings", a.saveSettings)
	mux.HandleFunc("GET /ocr/import", a.ocrUploadPage)
	mux.HandleFunc("POST /ocr/import", a.ocrStartSession)
	mux.HandleFunc("GET /ocr/import/{id}", a.ocrSessionStatus)
	mux.HandleFunc("POST /ocr/import/{id}/confirm", a.ocrConfirmDrafts)
	mux.HandleFunc("POST /ocr/import/{id}/delete", a.ocrDeleteSession)

	return recoverMiddleware(RequestLoggingMiddleware()(LoggerMiddleware(a.logger)(a.authMiddleware(mux))))
}

// publicPaths are paths that do not require authentication.
var publicPaths = []string{"/login", "/register", "/static/"}

func isPublicPath(path string) bool {
	for _, p := range publicPaths {
		if path == p || strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func (a *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.authEnabled {
			if a.legacyStore == nil {
				http.Error(w, "auth disabled but no legacy store configured", http.StatusInternalServerError)
				return
			}
			ctx := context.WithValue(r.Context(), contextKeyStore, a.legacyStore)
			ctx = context.WithValue(ctx, contextKeyUser, &db.User{ID: 0, Username: "anonymous", DisplayName: "Anonymous"})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		user, err := a.sessions.GetUserFromRequest(r)
		if err != nil || user == nil {
			a.redirect(w, r, "/login", "Please sign in")
			return
		}

		store, err := a.multiStore.ForUser(user.ID)
		if err != nil {
			a.logger.Error("open user database", "user_id", user.ID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyStore, store)
		ctx = context.WithValue(ctx, contextKeyUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) redirect(w http.ResponseWriter, r *http.Request, path string, notice string) {
	if notice != "" {
		separator := "?"
		if strings.Contains(path, "?") {
			separator = "&"
		}
		path += separator + "notice=" + url.QueryEscape(notice)
	}
	http.Redirect(w, r, path, http.StatusSeeOther)
}

func noticeFromRequest(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("notice"))
}

func parseInt64Path(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(r.PathValue(key), 10, 64)
}

func parsePositiveFloat(value string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("must be greater than zero")
	}
	return parsed, nil
}

func parsePositiveInt(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("must be greater than zero")
	}
	return parsed, nil
}

func validateDate(value string) (string, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	return parsed.Format("2006-01-02"), nil
}

func validateMonth(value string) (string, error) {
	value = strings.TrimSpace(value)
	if _, err := time.Parse("2006-01", value); err != nil {
		return "", err
	}
	return value, nil
}

func normalizeCategory(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "All"
	}
	return value
}

func isHTMX(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

func money(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func numfmt(value float64) string {
	intPart := int64(value)
	decPart := value - float64(intPart)
	result := formatWithCommas(intPart)
	if decPart < 0 {
		decPart = -decPart
	}
	return result + fmt.Sprintf("%.2f", decPart)[1:]
}

func formatWithCommas(n int64) string {
	if n < 0 {
		return "-" + formatWithCommas(-n)
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func dateLabel(value string) string {
	for _, format := range []string{"2006-01-02", time.RFC3339} {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed.Format("02 Jan 2006")
		}
	}
	return value
}

func selected(current, candidate string) string {
	if current == candidate {
		return "selected"
	}
	return ""
}

func monthName(value string) string {
	months := map[string]string{
		"01": "January", "02": "February", "03": "March",
		"04": "April", "05": "May", "06": "June",
		"07": "July", "08": "August", "09": "September",
		"10": "October", "11": "November", "12": "December",
	}
	if name, ok := months[value]; ok {
		return name
	}
	return value
}

func sanitizeHeaderValue(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == 0 {
			return -1
		}
		return r
	}, value)
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				http.Error(w, fmt.Sprintf("internal server error: %v", recovered), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
