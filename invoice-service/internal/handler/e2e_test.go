//go:build e2e

package handler

import (
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

func TestFullInvoiceWorkflow(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)

	// Save settings needed for invoice generation
	store.SaveSettings(db.Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		BankName:        "Test Bank",
		AccountName:     "Test Account",
		AccountNumber:   "12345678",
		SortCode:        "12-34-56",
		PaymentTerms:    "Payment due within 30 days.",
		DefaultDueDays:  30,
	})

	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	// Step 1: Create entries via HTTP
	createEntryViaHTTP(t, app, "2024-03-15", "Consulting", "4.5", "Client meeting")
	createEntryViaHTTP(t, app, "2024-03-16", "Design", "2.0", "UI review")

	// Step 2: Set rates via HTTP
	createRateViaHTTP(t, app, "Consulting", "2024-01-01", "", "150.00")
	createRateViaHTTP(t, app, "Design", "2024-01-01", "", "120.00")

	// Step 3: Generate invoice via HTTP
	generateInvoiceViaHTTP(t, app, "2024-03", "All")

	// Step 4: Verify invoice was created by checking the invoices page
	req := httptest.NewRequest("GET", "/invoices", nil)
	w := httptest.NewRecorder()
	app.invoicesPage(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "INV-")
}

func TestEntryCRUDViaHTTP(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	// Create entry
	createEntryViaHTTP(t, app, "2024-03-15", "Consulting", "4.5", "")

	// Verify it appears in the table
	req := httptest.NewRequest("GET", "/entries/table", nil)
	w := httptest.NewRecorder()
	app.entriesTable(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")

	// Delete entry
	deleteReq := httptest.NewRequest("POST", "/entries/1/delete", nil)
	deleteW := httptest.NewRecorder()
	app.deleteEntry(deleteW, deleteReq)
	require.Equal(t, http.StatusSeeOther, deleteW.Result().StatusCode)

	// Verify it's gone
	req2 := httptest.NewRequest("GET", "/entries/table", nil)
	w2 := httptest.NewRecorder()
	app.entriesTable(w2, req2)
	body2, _ := io.ReadAll(w2.Result().Body)
	assert.NotContains(t, string(body2), "Consulting")
}

func TestRateManagementViaHTTP(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	// Create rate
	createRateViaHTTP(t, app, "Consulting", "2024-01-01", "", "150.00")

	// Verify it appears in the table
	req := httptest.NewRequest("GET", "/rates/table", nil)
	w := httptest.NewRecorder()
	app.ratesTable(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")

	// Update rate
	updateReq := newFormRequest("/rates/1", urlencode(map[string]string{
		"category":   "Consulting",
		"start_date": "2024-01-01",
		"end_date":   "",
		"rate":       "175.00",
	}))
	w2 := httptest.NewRecorder()
	app.updateRate(w2, updateReq)

	resp2 := w2.Result()
	require.Equal(t, http.StatusOK, resp2.StatusCode) // HTMX partial for POST
	body2, _ := io.ReadAll(resp2.Body)
	assert.Contains(t, string(body2), "175.00")
}

func TestSettingsPersistenceViaHTTP(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	// Save settings
	saveReq := newFormRequest("/settings", urlencode(map[string]string{
		"business_name":    "Acme Corp",
		"bank_name":        "High Street Bank",
		"default_due_days": "45",
		"customer_name":    "Bob Smith",
		"customer_email":   "bob@acme.com",
	}))
	w := httptest.NewRecorder()
	app.saveSettings(w, saveReq)
	require.Equal(t, http.StatusSeeOther, w.Result().StatusCode)

	// Reload settings page and verify values appear
	req := httptest.NewRequest("GET", "/settings", nil)
	w2 := httptest.NewRecorder()
	app.settingsPage(w2, req)

	resp := w2.Result()
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Acme Corp")
	assert.Contains(t, string(body), "High Street Bank")
	assert.Contains(t, string(body), "Bob Smith")
}

func TestDashboardShowsSummary(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)

	// Set up entries and rates
	store.CreateEntry("2024-03-15", "Consulting", 4.5, "")
	store.CreateEntry("2024-03-16", "Design", 2.0, "")
	store.CreateRate("Consulting", "2024-01-01", "", 150.00)
	store.CreateRate("Design", "2024-01-01", "", 120.00)

	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.dashboard(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	// Should show Consulting and Design categories in the summary
	assert.Contains(t, string(body), "Consulting")
	assert.Contains(t, string(body), "Design")
}

func createEntryViaHTTP(t *testing.T, app *App, date, category, hours, notes string) {
	t.Helper()
	req := newFormRequest("/entries", urlencode(map[string]string{
		"date":     date,
		"category": category,
		"hours":    hours,
		"notes":    notes,
	}))
	w := httptest.NewRecorder()
	app.createEntry(w, req)
	require.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
}

func createRateViaHTTP(t *testing.T, app *App, category, startDate, endDate, rate string) {
	t.Helper()
	req := newFormRequest("/rates", urlencode(map[string]string{
		"category":   category,
		"start_date": startDate,
		"end_date":   endDate,
		"rate":       rate,
	}))
	w := httptest.NewRecorder()
	app.createRate(w, req)
	require.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
}

func generateInvoiceViaHTTP(t *testing.T, app *App, month, category string) {
	t.Helper()
	req := newFormRequest("/invoices/generate", urlencode(map[string]string{
		"month":        month,
		"invoice_date": "2024-04-01",
		"due_days":     "30",
		"category":     category,
	}))
	w := httptest.NewRecorder()
	app.generateInvoice(w, req)
	require.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
}
