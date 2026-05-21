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

func TestSettingsPageReturnsSavedValues(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.SaveSettings(db.Settings{
		BusinessName:    "Test Business Ltd",
		BusinessAddress: "123 Test St\nLondon\nSW1A 1AA",
		DefaultDueDays:  45,
	})

	req := httptest.NewRequest("GET", "/settings", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.settingsPage(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Test Business Ltd")
}

func TestSaveSettingsRedirects(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := newFormRequest("/settings", urlencode(map[string]string{
		"business_name":    "New Business Ltd",
		"bank_name":        "New Bank",
		"default_due_days": "60",
		"customer_name":    "Jane Doe",
		"customer_email":   "jane@example.com",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.saveSettings(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "/settings")
}
