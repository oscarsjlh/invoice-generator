package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRatesPageReturns200(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := httptest.NewRequest("GET", "/rates", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.ratesPage(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Rates")
}

func TestCreateRateValid(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := newFormRequest("/rates", urlencode(map[string]string{
		"category":   "Consulting",
		"start_date": "2024-01-01",
		"end_date":   "",
		"rate":       "150.00",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.createRate(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusSeeOther {
		t.Logf("createRate response: status=%d body=%s", resp.StatusCode, string(body))
	}
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
}

func TestCreateRateMissingCategory(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := newFormRequest("/rates", urlencode(map[string]string{
		"start_date": "2024-01-01",
		"end_date":   "",
		"rate":       "150.00",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.createRate(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteRate(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00)

	req := httptest.NewRequest("POST", "/rates/1/delete", nil)
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.deleteRate(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)

	rates, err := ta.store.ListRates()
	require.NoError(t, err)
	assert.Empty(t, rates)
}

func TestEditRateFormReturnsPartial(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00)

	req := httptest.NewRequest("GET", "/rates/1/edit", nil)
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.editRateForm(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")
}

func TestEditRateFormNotFound(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := httptest.NewRequest("GET", "/rates/999/edit", nil)
	req.SetPathValue("id", "999")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.editRateForm(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestUpdateRateSuccess(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00)

	req := newFormRequest("/rates/1", urlencode(map[string]string{
		"category":   "Design",
		"start_date": "2024-02-01",
		"end_date":   "",
		"rate":       "200.00",
	}))
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.updateRate(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Design")
}

func TestUpdateRateMissingCategory(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00)

	req := newFormRequest("/rates/1", urlencode(map[string]string{
		"start_date": "2024-02-01",
		"rate":       "200.00",
	}))
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.updateRate(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
