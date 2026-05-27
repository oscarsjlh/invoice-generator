package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"invoice-app/internal/auth"
	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/ocr"
	"invoice-app/static"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	multiStore   *db.MultiStore
	stores       *RequestStoreProvider
	authDB       *db.AuthDB
	webAuthn     *auth.WebAuthnManager
	sessionCookie *auth.SessionCookie
	cfg          config.Config
	logger       *slog.Logger
	renderer     *Renderer
	ocrJobs      *OCRJobRunner
	authEnabled  bool
	authLimiter  *AuthRateLimiter
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

func New(multiStore *db.MultiStore, authDB *db.AuthDB, webAuthn *auth.WebAuthnManager, sessionCookie *auth.SessionCookie, cfg config.Config, logger *slog.Logger) *App {
	app := &App{
		multiStore:    multiStore,
		stores:        NewRequestStoreProvider(multiStore),
		authDB:        authDB,
		webAuthn:      webAuthn,
		sessionCookie: sessionCookie,
		cfg:           cfg,
		logger:        logger,
		authEnabled:   cfg.AuthEnabled,
	}
	app.renderer = NewRenderer(logger, func() bool { return app.authEnabled }, func() bool { return app.cfg.RegistrationEnabled }, func() bool { return app.cfg.OCREnabled })
	app.ocrJobs = NewOCRJobRunner(app.stores, &app.cfg)
	app.authLimiter = NewAuthRateLimiter(cfg.TrustedProxy)
	return app
}

func (a *App) WaitForOCR() {
	a.ocrJobs.Wait()
}

func (a *App) SetLegacyStore(store *db.Store) {
	a.stores.SetLegacyStore(store)
}

// SetOCRClient sets the OCR extractor for testing.
func (a *App) SetOCRClient(client ocr.Extractor) {
	a.ocrJobs.SetExtractor(client)
}

func (a *App) renderPage(w http.ResponseWriter, r *http.Request, status int, page string, data any, extra ...string) {
	a.renderer.Page(w, r, status, page, data, extra...)
}

func (a *App) renderPartial(w http.ResponseWriter, status int, name string, data any, files ...string) {
	a.renderer.Partial(w, status, name, data, files...)
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()

	authHandlers := NewAuthHandlers(a.webAuthn, a.sessionCookie, a.renderer, a.authLimiter, a.cfg.AuthEnabled, a.cfg.RegistrationEnabled)
	authHandlers.Routes(mux)

	entryHandlers := NewEntryHandlers(a.renderer)
	entryHandlers.Routes(mux)

	rateHandlers := NewRateHandlers(a.renderer)
	rateHandlers.Routes(mux)

	invoiceHandlers := NewInvoiceHandlers(a.renderer, a.cfg)
	invoiceHandlers.Routes(mux)

	settingsHandlers := NewSettingsHandlers(a.renderer)
	settingsHandlers.Routes(mux)

	ocrHandlers := NewOCRHandlers(a.renderer, a.ocrJobs, a.cfg)
	ocrHandlers.Routes(mux)

	mux.HandleFunc("GET /health", a.health)
	a.registerE2EHelperRoutes(mux)
	mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/static/", http.FileServerFS(static.FS)).ServeHTTP(w, r)
	})
	mux.HandleFunc("GET /", a.dashboard)

	handler := recoverMiddleware(SecurityHeadersMiddleware()(LoggerMiddleware(a.logger)(RequestLoggingMiddleware()(a.authMiddleware(CSRFMiddlewareWithPublic(a.isPublicPath)(mux))))))
	return otelhttp.NewHandler(handler, "http.server")
}

// publicPaths are paths that do not require authentication.
func (a *App) isPublicPath(path string) bool {
	if path == "/login" || strings.HasPrefix(path, "/login/") || path == "/health" || strings.HasPrefix(path, "/static/") || path == "/auth/username-policy" {
		return true
	}
	if a.cfg.RegistrationEnabled && (path == "/register" || strings.HasPrefix(path, "/register/")) {
		return true
	}
	return isE2EPublicPath(path)
}

func isPublicPath(path string) bool {
	if path == "/login" || strings.HasPrefix(path, "/login/") || path == "/health" || strings.HasPrefix(path, "/static/") || path == "/auth/username-policy" {
		return true
	}
	if path == "/register" || strings.HasPrefix(path, "/register/") {
		return true
	}
	return isE2EPublicPath(path)
}

func (a *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.authEnabled {
			secure := r.TLS != nil
			auth.EnsureCSRFCookie(w, r, secure)
			store, user, err := a.stores.ForRequest(r, nil)
			if err != nil {
				http.Error(w, "auth disabled but no legacy store configured", http.StatusInternalServerError)
				return
			}
			ctx := context.WithValue(r.Context(), contextKeyStore, newTracedStore(r.Context(), store))
			ctx = context.WithValue(ctx, contextKeyUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if !a.cfg.RegistrationEnabled && (r.URL.Path == "/register" || strings.HasPrefix(r.URL.Path, "/register/")) {
			http.NotFound(w, r)
			return
		}

		if a.isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		user, err := a.sessionCookie.Get(r)
		if err != nil || user == nil {
			redirect(w, r, loginRedirectPath(r), noticeSignInRequired)
			return
		}

		a.sessionCookie.EnsureCSRF(w, r)

		store, user, err := a.stores.ForRequest(r, user)
		if err != nil {
			a.logger.Error("open user database", "user_id", user.ID, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyStore, newTracedStore(r.Context(), store))
		ctx = context.WithValue(ctx, contextKeyUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func redirect(w http.ResponseWriter, r *http.Request, path string, notice string) {
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

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
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
