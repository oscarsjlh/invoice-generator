package handler

import (
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/static"
	"invoice-app/templates"
)

type App struct {
	store  *db.Store
	cfg    config.Config
	logger *slog.Logger
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
	return &App{store: store, cfg: cfg, logger: logger}
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
	return a.recoverMiddleware(RequestLoggingMiddleware()(LoggerMiddleware(a.logger)(mux)))
}

func (a *App) renderPage(w http.ResponseWriter, status int, page string, data any, extra ...string) {
	files := append([]string{"layout.html"}, extra...)
	files = append(files, page)
	tmpl, err := template.New("layout.html").Funcs(template.FuncMap{
		"money":     money,
		"numfmt":    numfmt,
		"dateLabel": dateLabel,
		"selected":  selected,
		"monthName": monthName,
		"mul":       func(a float64, b float64) float64 { return a * b },
		"divf":      func(a float64, b float64) float64 { return a / b },
		"div":       func(a, b int64) int64 { return a / b },
	}).ParseFS(templates.FS, files...)
	if err != nil {
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
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"money":     money,
		"numfmt":    numfmt,
		"dateLabel": dateLabel,
		"selected":  selected,
		"monthName": monthName,
		"mul":       func(a float64, b float64) float64 { return a * b },
		"divf":      func(a float64, b float64) float64 { return a / b },
		"div":       func(a, b int64) int64 { return a / b },
	}).ParseFS(templates.FS, files...)
	if err != nil {
		http.Error(w, fmt.Sprintf("parse partial: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("execute partial: %v", err), http.StatusInternalServerError)
	}
}

func (a *App) redirect(w http.ResponseWriter, r *http.Request, path string, notice string) {
	if notice != "" {
		separator := "?"
		if strings.Contains(path, "?") {
			separator = "&"
		}
		path += separator + "notice=" + urlQueryEscape(notice)
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

func urlQueryEscape(value string) string {
	replacer := strings.NewReplacer("%", "%25", " ", "%20", "&", "%26", "?", "%3F", "=", "%3D", "+", "%2B")
	return replacer.Replace(value)
}

func (a *App) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				http.Error(w, fmt.Sprintf("internal server error: %v", recovered), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func notFoundIfNoRows(w http.ResponseWriter, err error) bool {
	if err == sql.ErrNoRows {
		http.NotFound(w, nil)
		return true
	}
	return false
}
