//go:build e2e

package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"invoice-app/internal/db"
)

func TestFullInvoiceWorkflow(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.SaveSettings(db.Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		BankName:        "Test Bank",
		AccountName:     "Test Account",
		AccountNumber:   "12345678",
		SortCode:        "12-34-56",
		PaymentTerms:    "Payment due within 30 days.",
		DefaultDueDays:  30,
	})

	createEntryViaHTTP(t, ta.app, ta.store, "2024-03-15", "Consulting", "4.5", "Client meeting")
	createEntryViaHTTP(t, ta.app, ta.store, "2024-03-16", "Design", "2.0", "UI review")

	createRateViaHTTP(t, ta.app, ta.store, "Consulting", "2024-01-01", "", "150.00")
	createRateViaHTTP(t, ta.app, ta.store, "Design", "2024-01-01", "", "120.00")

	generateInvoiceViaHTTP(t, ta.app, ta.store, "2024-03", "All")

	req := httptest.NewRequest("GET", "/invoices", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	ta.app.invoicesPage(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "INV-")
}

func TestEntryCRUDViaHTTP(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	createEntryViaHTTP(t, ta.app, ta.store, "2024-03-15", "Consulting", "4.5", "")

	req := httptest.NewRequest("GET", "/entries/table", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	ta.app.entriesTable(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")

	deleteReq := httptest.NewRequest("POST", "/entries/1/delete", nil)
	deleteReq.SetPathValue("id", "1")
	ctx2 := WithTestStore(deleteReq.Context(), ta.store)
	deleteReq = deleteReq.WithContext(ctx2)
	deleteW := httptest.NewRecorder()
	ta.app.deleteEntry(deleteW, deleteReq)
	require.Equal(t, http.StatusSeeOther, deleteW.Result().StatusCode)

	req2 := httptest.NewRequest("GET", "/entries/table", nil)
	ctx3 := WithTestStore(req2.Context(), ta.store)
	req2 = req2.WithContext(ctx3)
	w2 := httptest.NewRecorder()
	ta.app.entriesTable(w2, req2)
	body2, _ := io.ReadAll(w2.Result().Body)
	assert.NotContains(t, string(body2), "Consulting")
}

func TestRateManagementViaHTTP(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	createRateViaHTTP(t, ta.app, ta.store, "Consulting", "2024-01-01", "", "150.00")

	req := httptest.NewRequest("GET", "/rates/table", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	ta.app.ratesTable(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")

	updateReq := newFormRequest("/rates/1", urlencode(map[string]string{
		"category":   "Consulting",
		"start_date": "2024-01-01",
		"end_date":   "",
		"rate":       "175.00",
	}))
	updateReq.SetPathValue("id", "1")
	ctx2 := WithTestStore(updateReq.Context(), ta.store)
	updateReq = updateReq.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	ta.app.updateRate(w2, updateReq)

	resp2 := w2.Result()
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	body2, _ := io.ReadAll(resp2.Body)
	assert.Contains(t, string(body2), "175.00")
}

func TestSettingsPersistenceViaHTTP(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	saveReq := newFormRequest("/settings", urlencode(map[string]string{
		"business_name":    "Acme Corp",
		"bank_name":        "High Street Bank",
		"default_due_days": "45",
		"customer_name":    "Bob Smith",
		"customer_email":   "bob@acme.com",
	}))
	ctx := WithTestStore(saveReq.Context(), ta.store)
	saveReq = saveReq.WithContext(ctx)
	w := httptest.NewRecorder()
	ta.app.saveSettings(w, saveReq)
	require.Equal(t, http.StatusSeeOther, w.Result().StatusCode)

	req := httptest.NewRequest("GET", "/settings", nil)
	ctx2 := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	ta.app.settingsPage(w2, req)

	resp := w2.Result()
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Acme Corp")
	assert.Contains(t, string(body), "High Street Bank")
	assert.Contains(t, string(body), "Bob Smith")
}

func TestDashboardShowsSummary(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, "")
	ta.store.CreateEntry("2024-03-16", "Design", 2.0, "")
	ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00)
	ta.store.CreateRate("Design", "2024-01-01", "", 120.00)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	ta.app.dashboard(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")
	assert.Contains(t, string(body), "Design")
}

func createEntryViaHTTP(t *testing.T, app *App, store *db.Store, date, category, hours, notes string) {
	t.Helper()
	req := newFormRequest("/entries", urlencode(map[string]string{
		"date":     date,
		"category": category,
		"hours":    hours,
		"notes":    notes,
	}))
	ctx := WithTestStore(req.Context(), store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	app.createEntry(w, req)
	require.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
}

func createRateViaHTTP(t *testing.T, app *App, store *db.Store, category, startDate, endDate, rate string) {
	t.Helper()
	req := newFormRequest("/rates", urlencode(map[string]string{
		"category":   category,
		"start_date": startDate,
		"end_date":   endDate,
		"rate":       rate,
	}))
	ctx := WithTestStore(req.Context(), store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	app.createRate(w, req)
	require.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
}

func generateInvoiceViaHTTP(t *testing.T, app *App, store *db.Store, month, category string) {
	t.Helper()
	req := newFormRequest("/invoices/generate", urlencode(map[string]string{
		"month":        month,
		"invoice_date": "2024-04-01",
		"due_days":     "30",
		"category":     category,
	}))
	ctx := WithTestStore(req.Context(), store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	app.generateInvoice(w, req)
	require.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
}
