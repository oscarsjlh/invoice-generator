package handler

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"invoice-app/internal/db"
	"invoice-app/internal/email"
	"invoice-app/internal/pdf"
	"invoice-app/templates"
)

func (a *App) invoicesPage(w http.ResponseWriter, r *http.Request) {
	invoices, err := a.store.ListInvoices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	categories, err := a.store.ListCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	settings, err := a.store.LoadSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	a.renderPage(w, http.StatusOK, "invoices.html", InvoicesPageData{
		Invoices:           invoices,
		Categories:         categories,
		Notice:             noticeFromRequest(r),
		DefaultMonth:       now.Format("2006-01"),
		DefaultInvoiceDate: now.Format("2006-01-02"),
		DefaultDueDays:     settings.DefaultDueDays,
		SelectedCategory:   "All",
	}, "invoice_list.html", "invoices.html")
}

func (a *App) generateInvoice(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	month, err := validateMonth(r.FormValue("month"))
	if err != nil {
		http.Error(w, "month must use YYYY-MM", http.StatusBadRequest)
		return
	}
	invoiceDate, err := validateDate(r.FormValue("invoice_date"))
	if err != nil {
		http.Error(w, "invoice date must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	dueDays, err := parsePositiveInt(r.FormValue("due_days"))
	if err != nil {
		http.Error(w, "due days must be a positive integer", http.StatusBadRequest)
		return
	}

	settings, err := a.store.LoadSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	category := normalizeCategory(r.FormValue("category"))
	invoiceID, err := a.store.GenerateInvoice(month, category, invoiceDate, dueDays, settings)
	if err != nil {
		a.redirect(w, r, "/invoices", fmt.Sprintf("Could not generate invoice: %v", err))
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/invoices/%d?notice=%s", invoiceID, urlQueryEscape("Invoice created")), http.StatusSeeOther)
}

func (a *App) invoicePreview(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid invoice id", http.StatusBadRequest)
		return
	}
	invoice, err := a.store.GetInvoice(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.renderPage(w, http.StatusOK, "invoice_preview_page.html", InvoicePreviewPageData{
		Invoice: invoice,
		Notice:  noticeFromRequest(r),
	}, "invoice_preview_fragment.html", "invoice_preview_page.html")
}

func (a *App) invoicePDF(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid invoice id", http.StatusBadRequest)
		return
	}
	invoice, err := a.store.GetInvoice(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pdfData, err := a.generateInvoicePDF(invoice)
	if err != nil {
		http.Error(w, fmt.Sprintf("generate pdf: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, invoice.InvoiceNumber))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))
	w.Write(pdfData)
}

func (a *App) sendInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid invoice id", http.StatusBadRequest)
		return
	}
	invoice, err := a.store.GetInvoice(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	settings, err := a.store.LoadSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if settings.CustomerEmail == "" {
		a.redirect(w, r, fmt.Sprintf("/invoices/%d", invoice.ID), "Customer email not configured — set it in Settings")
		return
	}

	cfg := email.Config{
		SMTPHost: a.cfg.SMTPHost,
		SMTPPort: a.cfg.SMTPPort,
		SMTPUser: a.cfg.SMTPUser,
		SMTPPass: a.cfg.SMTPPass,
		SMTPFrom: a.cfg.SMTPFrom,
	}

	if cfg.SMTPHost == "" || cfg.SMTPFrom == "" {
		a.redirect(w, r, fmt.Sprintf("/invoices/%d", invoice.ID), "SMTP not configured — set SMTP_HOST and SMTP_FROM environment variables")
		return
	}

	pdfData, err := a.generateInvoicePDF(invoice)
	if err != nil {
		a.redirect(w, r, fmt.Sprintf("/invoices/%d", invoice.ID), fmt.Sprintf("Generate PDF failed: %v", err))
		return
	}

	customerName := settings.CustomerName
	if customerName == "" {
		customerName = "Customer"
	}

	if err := email.SendInvoice(
		settings.CustomerEmail,
		customerName,
		cfg.SMTPFrom,
		invoice.InvoiceNumber,
		pdfData,
		cfg,
	); err != nil {
		a.redirect(w, r, fmt.Sprintf("/invoices/%d", invoice.ID), fmt.Sprintf("Send failed: %v", err))
		return
	}

	a.redirect(w, r, fmt.Sprintf("/invoices/%d", invoice.ID), fmt.Sprintf("Invoice sent to %s", settings.CustomerEmail))
}

func (a *App) generateInvoicePDF(invoice db.Invoice) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "typst-invoice-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	templateBytes, err := templates.FS.ReadFile("invoice-maker.typ")
	if err != nil {
		return nil, fmt.Errorf("read typst template: %w", err)
	}

	// Parse business address into components
	street, city, postalCode := parseAddress(invoice.BusinessAddress)

	// Load settings for customer info
	settings, _ := a.store.LoadSettings()

	customerStreet, customerCity, customerPostalCode := parseAddress(settings.CustomerAddress)
	if customerStreet == "" {
		customerStreet = street
	}
	if customerCity == "" {
		customerCity = city
	}
	if customerPostalCode == "" {
		customerPostalCode = postalCode
	}

	data := pdf.InvoiceData{
		InvoiceNumber:      invoice.InvoiceNumber,
		InvoiceDate:        invoice.InvoiceDate,
		DueDate:            invoice.DueDate,
		Month:              invoice.Month,
		BusinessName:       invoice.BusinessName,
		Street:             street,
		City:               city,
		PostalCode:         postalCode,
		BankName:           invoice.BankName,
		AccountName:        invoice.AccountName,
		SortCode:           invoice.SortCode,
		AccountNumber:      invoice.AccountNumber,
		PaymentTerms:       invoice.PaymentTerms,
		CustomerName:       settings.CustomerName,
		CustomerTitle:      settings.CustomerTitle,
		CustomerStreet:     customerStreet,
		CustomerCity:       customerCity,
		CustomerPostalCode: customerPostalCode,
	}

	for _, line := range invoice.Lines {
		hoursMinutes := int(math.Round(line.Hours * 60))
		data.Items = append(data.Items, pdf.ItemData{
			Description: line.Category,
			DurMin:      hoursMinutes,
			HourlyRate:  line.Rate,
		})
	}

	typContent := pdf.FormatInvoiceTyp(data)
	return pdf.GenerateInvoicePDF(tmpDir, templateBytes, typContent)
}

func parseAddress(address string) (street, city, postalCode string) {
	parts := strings.Split(address, "\n")
	nonEmpty := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	switch len(nonEmpty) {
	case 0:
		return address, "", ""
	case 1:
		return nonEmpty[0], "", ""
	case 2:
		return nonEmpty[0], nonEmpty[1], ""
	default:
		last := nonEmpty[len(nonEmpty)-1]
		city := nonEmpty[len(nonEmpty)-2]
		street := strings.Join(nonEmpty[:len(nonEmpty)-2], ", ")
		return street, city, last
	}
}
