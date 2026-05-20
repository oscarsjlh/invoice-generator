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

func newTestApp(t *testing.T) *App {
	t.Helper()
	store := testutil.NewTestDB(t)
	cfg := config.Config{
		Address:        ":8080",
		TypstBin:       "", // disable PDF for handler tests
		OCREnabled:     false,
		DefaultDueDays: 30,
	}
	logger := NewLogger("info", "text")
	return New(store, cfg, logger)
}

func TestCreateEntryValid(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	req := newFormRequest("/entries", urlencode(map[string]string{
		"date":     "2024-03-15",
		"category": "Consulting",
		"hours":    "4.5",
		"notes":    "Client meeting",
	}))
	w := httptest.NewRecorder()

	app.createEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Location"), "/entries")
}

func TestCreateEntryMissingCategory(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	req := newFormRequest("/entries", urlencode(map[string]string{
		"date":  "2024-03-15",
		"hours": "4.5",
	}))
	w := httptest.NewRecorder()

	app.createEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateEntryInvalidHours(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	req := newFormRequest("/entries", urlencode(map[string]string{
		"date":     "2024-03-15",
		"category": "Consulting",
		"hours":    "abc",
	}))
	w := httptest.NewRecorder()

	app.createEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEntriesTableReturnsPartial(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	// Create an entry first via the store directly
	store := testutil.NewTestDB(t)
	store.CreateEntry("2024-03-15", "Consulting", 4.5, "")
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app = New(store, cfg, logger)

	req := httptest.NewRequest("GET", "/entries/table", nil)
	w := httptest.NewRecorder()

	app.entriesTable(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")
}

func TestDeleteEntryRedirects(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	// Create an entry first via the store directly
	store := testutil.NewTestDB(t)
	store.CreateEntry("2024-03-15", "Consulting", 4.5, "")
	cfg := config.Config{Address: ":8080", OCREnabled: false}
	logger := NewLogger("info", "text")
	app = New(store, cfg, logger)

	req := httptest.NewRequest("POST", "/entries/1/delete", nil)
	w := httptest.NewRecorder()

	app.Routes().ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Location"), "/entries")
}
