package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"invoice-app/internal/testutil"
)

func TestInvoicesPageReturns200(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := httptest.NewRequest("GET", "/invoices", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.ih.invoicesPage(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Invoices")
}

func TestGenerateInvoiceSuccess(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00))

	req := newFormRequest("/invoices/generate", urlencode(map[string]string{
		"month":        "2024-03",
		"invoice_date": "2024-04-01",
		"due_days":     "30",
		"category":     "All",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.ih.generateInvoice(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "/invoices/")
}

func TestGenerateInvoiceAcceptsCustomInvoiceNumber(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00))

	req := newFormRequest("/invoices/generate", urlencode(map[string]string{
		"month":          "2024-03",
		"invoice_date":   "2024-04-01",
		"due_days":       "30",
		"category":       "All",
		"invoice_number": "CUSTOM-2024-03",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.ih.generateInvoice(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)

	invoices, err := ta.store.ListInvoices()
	require.NoError(t, err)
	require.Len(t, invoices, 1)
	assert.Equal(t, "CUSTOM-2024-03", invoices[0].InvoiceNumber)
}

func TestGenerateInvoiceFailsWhenUnrated(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))

	req := newFormRequest("/invoices/generate", urlencode(map[string]string{
		"month":        "2024-03",
		"invoice_date": "2024-04-01",
		"due_days":     "30",
		"category":     "All",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.ih.generateInvoice(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "notice=")
}

func TestInvoicePreviewReturns200(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00))
	require.NoError(t, ta.store.SaveSettings(testutil.SampleSettings()))

	id, err := ta.store.GenerateInvoice("2024-03", "All", "2024-04-01", 30, "", testutil.SampleSettings())
	require.NoError(t, err)

	req := httptest.NewRequest("GET", fmt.Sprintf("/invoices/%d", id), nil)
	req.SetPathValue("id", fmt.Sprintf("%d", id))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.ih.invoicePreview(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "INV-")
}

func TestDeleteInvoiceRedirectsAndRemovesInvoice(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00))

	id, err := ta.store.GenerateInvoice("2024-03", "All", "2024-04-01", 30, "", testutil.SampleSettings())
	require.NoError(t, err)

	req := httptest.NewRequest("POST", fmt.Sprintf("/invoices/%d/delete", id), nil)
	req.SetPathValue("id", fmt.Sprintf("%d", id))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.ih.deleteInvoice(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Equal(t, "/invoices?notice=Invoice+deleted", resp.Header.Get("Location"))

	_, err = ta.store.GetInvoice(id)
	require.Error(t, err)
}

func TestInvoicePreview404(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := httptest.NewRequest("GET", "/invoices/99999", nil)
	req.SetPathValue("id", "99999")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.ih.invoicePreview(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDashboardRendersSuccessfully(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00))
	require.NoError(t, ta.store.SaveSettings(testutil.SampleSettings()))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.dashboard(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Dashboard")
}

func TestDashboardShowsMissingRateWarning(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-04-01", "", 150.00))

	req := httptest.NewRequest("GET", "/?year=2024&month=03", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.dashboard(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	assert.Contains(t, html, "You have missing rates on some entries")
	assert.Contains(t, html, "1 affected")
	assert.Contains(t, html, "not included in the totals")
}
