package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"invoice-app/internal/config"
	"invoice-app/internal/testutil"
)

func TestRatesPageReturns200(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	req := httptest.NewRequest("GET", "/rates", nil)
	w := httptest.NewRecorder()

	app.ratesPage(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Rates")
}

func TestCreateRateValid(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	req := newFormRequest("/rates", urlencode(map[string]string{
		"category":   "Consulting",
		"start_date": "2024-01-01",
		"end_date":   "",
		"rate":       "150.00",
	}))
	w := httptest.NewRecorder()

	app.createRate(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusSeeOther {
		t.Logf("createRate response: status=%d body=%s", resp.StatusCode, string(body))
	}
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
}

func TestCreateRateMissingCategory(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	req := newFormRequest("/rates", urlencode(map[string]string{
		"start_date": "2024-01-01",
		"end_date":   "",
		"rate":       "150.00",
	}))
	w := httptest.NewRecorder()

	app.createRate(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteRate(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	// Create a rate first
	store.CreateRate("Consulting", "2024-01-01", "", 150.00)

	req := httptest.NewRequest("POST", "/rates/1/delete", nil)
	w := httptest.NewRecorder()

	app.Routes().ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)

	// Verify rate was deleted
	rates, err := store.ListRates()
	require.NoError(t, err)
	assert.Empty(t, rates)
}
