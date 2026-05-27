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

	id, err := ta.store.GenerateInvoice("2024-03", "All", "2024-04-01", 30, testutil.SampleSettings())
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
