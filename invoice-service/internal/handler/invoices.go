package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"invoice-app/internal/config"
	"invoice-app/internal/invoicedelivery"
)

type InvoiceHandlers struct {
	renderer *Renderer
	cfg      config.Config
}

func NewInvoiceHandlers(renderer *Renderer, cfg config.Config) *InvoiceHandlers {
	return &InvoiceHandlers{renderer: renderer, cfg: cfg}
}

func (h *InvoiceHandlers) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /invoices", h.invoicesPage)
	mux.HandleFunc("POST /invoices/generate", h.generateInvoice)
	mux.HandleFunc("GET /invoices/{id}", h.invoicePreview)
	mux.HandleFunc("GET /invoices/{id}/pdf", h.invoicePDF)
	mux.HandleFunc("POST /invoices/{id}/send", h.sendInvoice)
}

func (h *InvoiceHandlers) invoicesPage(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	invoices, err := store.ListInvoices()
	if err != nil {
		LoggerFromContext(r.Context()).Error("list invoices", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	categories, err := store.ListCategories()
	if err != nil {
		LoggerFromContext(r.Context()).Error("list categories", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	settings, err := store.LoadSettings()
	if err != nil {
		LoggerFromContext(r.Context()).Error("load settings", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	now := time.Now()
	h.renderer.Page(w, r, http.StatusOK, "invoices.html", InvoicesPageData{
		Invoices:           invoices,
		Categories:         categories,
		Notice:             noticeFromRequest(r),
		DefaultMonth:       now.Format("2006-01"),
		DefaultInvoiceDate: now.Format("2006-01-02"),
		DefaultDueDays:     settings.DefaultDueDays,
		SelectedCategory:   "All",
	}, "invoice_list.html", "invoices.html")
}

func (h *InvoiceHandlers) generateInvoice(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
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

	settings, err := store.LoadSettings()
	if err != nil {
		LoggerFromContext(r.Context()).Error("load settings for invoice generation", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	category := normalizeCategory(r.FormValue("category"))
	invoiceID, err := store.GenerateInvoice(month, category, invoiceDate, dueDays, settings)
	if err != nil {
		LoggerFromContext(r.Context()).Warn("generate invoice", "month", month, "category", category, "error", err)
		redirect(w, r, "/invoices", fmt.Sprintf("Could not generate invoice: %v", err))
		return
	}

	LoggerFromContext(r.Context()).Info("invoice created", "invoice_id", invoiceID, "month", month, "category", category)
	http.Redirect(w, r, fmt.Sprintf("/invoices/%d?notice=%s", invoiceID, url.QueryEscape("Invoice created")), http.StatusSeeOther)
}

func (h *InvoiceHandlers) invoicePreview(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid invoice id", http.StatusBadRequest)
		return
	}
	invoice, err := store.GetInvoice(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		LoggerFromContext(r.Context()).Error("get invoice for preview", "invoice_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	h.renderer.Page(w, r, http.StatusOK, "invoice_preview_page.html", InvoicePreviewPageData{
		Invoice: invoice,
		Notice:  noticeFromRequest(r),
	}, "invoice_preview_fragment.html", "invoice_preview_page.html")
}

func (h *InvoiceHandlers) invoicePDF(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid invoice id", http.StatusBadRequest)
		return
	}
	invoice, err := store.GetInvoice(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		LoggerFromContext(r.Context()).Error("get invoice for pdf", "invoice_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	delivery := invoicedelivery.New(store, h.cfg)
	result, err := delivery.RenderPDF(r.Context(), id)
	if err != nil {
		LoggerFromContext(r.Context()).Error("generate pdf", "invoice_id", id, "invoice_number", invoice.InvoiceNumber, "error", err)
		http.Error(w, fmt.Sprintf("generate pdf: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, sanitizeHeaderValue(invoice.InvoiceNumber)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(result.Data)))
	_, _ = w.Write(result.Data)
}

func (h *InvoiceHandlers) sendInvoice(w http.ResponseWriter, r *http.Request) {
	store := StoreFromContext(r.Context())
	id, err := parseInt64Path(r, "id")
	if err != nil {
		http.Error(w, "invalid invoice id", http.StatusBadRequest)
		return
	}
	invoice, err := store.GetInvoice(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		LoggerFromContext(r.Context()).Error("get invoice for send", "invoice_id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	delivery := invoicedelivery.New(store, h.cfg)
	result, err := delivery.Send(r.Context(), id)
	if errors.Is(err, invoicedelivery.ErrCustomerEmailMissing) {
		redirect(w, r, fmt.Sprintf("/invoices/%d", invoice.ID), "Customer email not configured — set it in Settings")
		return
	}
	if errors.Is(err, invoicedelivery.ErrSMTPNotConfigured) {
		redirect(w, r, fmt.Sprintf("/invoices/%d", invoice.ID), "SMTP not configured — set SMTP_HOST and SMTP_FROM environment variables")
		return
	}
	if errors.Is(err, invoicedelivery.ErrRenderPDF) {
		LoggerFromContext(r.Context()).Error("generate pdf for send", "invoice_id", id, "invoice_number", invoice.InvoiceNumber, "error", err)
		redirect(w, r, fmt.Sprintf("/invoices/%d", invoice.ID), fmt.Sprintf("Generate PDF failed: %v", err))
		return
	}
	if err != nil {
		LoggerFromContext(r.Context()).Error("send invoice email", "invoice_id", id, "invoice_number", invoice.InvoiceNumber, "error", err)
		redirect(w, r, fmt.Sprintf("/invoices/%d", invoice.ID), fmt.Sprintf("Send failed: %v", err))
		return
	}

	redirect(w, r, fmt.Sprintf("/invoices/%d", invoice.ID), fmt.Sprintf("Invoice sent to %s", result.Recipient))
}
