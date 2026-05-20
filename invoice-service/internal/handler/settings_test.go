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

func TestSettingsPageReturnsSavedValues(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)

	// Save some settings first
	store.SaveSettings(db.Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		DefaultDueDays:  45,
	})

	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	req := httptest.NewRequest("GET", "/settings", nil)
	w := httptest.NewRecorder()

	app.settingsPage(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Test Business Ltd")
}

func TestSaveSettingsRedirects(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)

	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app := New(store, cfg, logger)

	req := newFormRequest("/settings", urlencode(map[string]string{
		"business_name":    "New Business Ltd",
		"bank_name":        "New Bank",
		"default_due_days": "60",
		"customer_name":    "Jane Doe",
		"customer_email":   "jane@example.com",
	}))
	w := httptest.NewRecorder()

	app.saveSettings(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "/settings")
}
