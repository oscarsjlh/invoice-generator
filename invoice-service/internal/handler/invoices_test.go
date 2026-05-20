package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/testutil"
)

func TestInvoicesPageReturns200(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	req := httptest.NewRequest("GET", "/invoices", nil)
	w := httptest.NewRecorder()

	app.invoicesPage(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Invoices")
}

func TestGenerateInvoiceSuccess(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)

	// Set up entries and rates for March 2024
	store.CreateEntry("2024-03-15", "Consulting", 4.5, "")
	store.CreateRate("Consulting", "2024-01-01", "", 150.00)

	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	req := newFormRequest("/invoices/generate", urlencode(map[string]string{
		"month":        "2024-03",
		"invoice_date": "2024-04-01",
		"due_days":     "30",
		"category":     "All",
	}))
	w := httptest.NewRecorder()

	app.generateInvoice(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "/invoices/")
}

func TestGenerateInvoiceFailsWhenUnrated(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)

	// Create entry without rate
	store.CreateEntry("2024-03-15", "Consulting", 4.5, "")

	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	req := newFormRequest("/invoices/generate", urlencode(map[string]string{
		"month":        "2024-03",
		"invoice_date": "2024-04-01",
		"due_days":     "30",
		"category":     "All",
	}))
	w := httptest.NewRecorder()

	app.generateInvoice(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "notice=")
}

func TestInvoicePreviewReturns200(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)

	// Create an invoice first
	store.CreateEntry("2024-03-15", "Consulting", 4.5, "")
	store.CreateRate("Consulting", "2024-01-01", "", 150.00)
	store.SaveSettings(db.Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		BankName:        "Test Bank",
		AccountName:     "Test Account",
		AccountNumber:   "12345678",
		SortCode:        "12-34-56",
		PaymentTerms:    "Payment due within 30 days.",
	})

	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	// Generate an invoice first
	id, err := store.GenerateInvoice("2024-03", "All", "2024-04-01", 30, db.Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		BankName:        "Test Bank",
		AccountName:     "Test Account",
		AccountNumber:   "12345678",
		SortCode:        "12-34-56",
		PaymentTerms:    "Payment due within 30 days.",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", fmt.Sprintf("/invoices/%d", id), nil)
	w := httptest.NewRecorder()

	app.Routes().ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "INV-")
}

func TestInvoicePreview404(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	req := httptest.NewRequest("GET", "/invoices/99999", nil)
	w := httptest.NewRecorder()

	app.Routes().ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
