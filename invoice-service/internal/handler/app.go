package handler

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/static"
	"invoice-app/templates"
)

type App struct {
	store    db.Storer
	cfg      config.Config
	logger   *slog.Logger
	baseTmpl *template.Template
	ocrWg    sync.WaitGroup
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
}

type EntriesPageData struct {
	Entries    []db.Entry
	Categories []string
	Notice     string
	Form       db.Entry
}

type RatesPageData struct {
	Rates  []db.Rate
	Notice string
	Form   db.Rate
}

type InvoicesPageData struct {
	Invoices           []db.InvoiceSummary
	Categories         []string
	Notice             string
	DefaultMonth       string
	DefaultInvoiceDate string
	DefaultDueDays     int
	SelectedCategory   string
}

type SettingsPageData struct {
	Settings db.Settings
	Notice   string
}

type InvoicePreviewPageData struct {
	Invoice db.Invoice
	Notice  string
}

func New(store *db.Store, cfg config.Config, logger *slog.Logger) *App {
	app := &App{store: store, cfg: cfg, logger: logger}
	app.baseTmpl = app.compileTemplates()
	return app
}

func (a *App) compileTemplates() *template.Template {
	funcMap := template.FuncMap{
		"money":     money,
		"numfmt":    numfmt,
		"dateLabel": dateLabel,
		"selected":  selected,
		"monthName": monthName,
		"mul":       func(a float64, b float64) float64 { return a * b },
		"divf":      func(a float64, b float64) float64 { return a / b },
		"div":       func(a, b int64) int64 { return a / b },
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templates.FS, "layout.html")
	if err != nil {
		a.logger.Error("compile base template", "error", err)
		panic(fmt.Sprintf("failed to compile base template: %v", err))
	}
	return tmpl
}

// WaitForOCR blocks until all background OCR processing goroutines complete.
func (a *App) WaitForOCR() {
	a.ocrWg.Wait()
}

func (a *App) renderPage(w http.ResponseWriter, status int, page string, data any, extra ...string) {
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, fmt.Sprintf("execute template: %v", err), http.StatusInternalServerError)
	}
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

// StoreForTest returns the underlying store for testing purposes.
func (a *App) StoreForTest() db.Storer {
	return a.store
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
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
	mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/static/", http.FileServerFS(static.FS)).ServeHTTP(w, r)
	})
	return recoverMiddleware(RequestLoggingMiddleware()(LoggerMiddleware(a.logger)(mux)))
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

// sanitizeHeaderValue strips control characters (CR, LF, NUL) from a string
// to prevent HTTP header injection.
func sanitizeHeaderValue(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == 0 {
			return -1
		}
		return r
	}, value)
}

// recoverMiddleware recovers from panics and returns a 500 error.
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
