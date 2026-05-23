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

func TestCreateEntryValid(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := newFormRequest("/entries", urlencode(map[string]string{
		"date":     "2024-03-15",
		"category": "Consulting",
		"hours":    "4.5",
		"notes":    "Client meeting",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.createEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Location"), "/entries")
}

func TestCreateEntryMissingCategory(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := newFormRequest("/entries", urlencode(map[string]string{
		"date":  "2024-03-15",
		"hours": "4.5",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.createEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateEntryInvalidHours(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := newFormRequest("/entries", urlencode(map[string]string{
		"date":     "2024-03-15",
		"category": "Consulting",
		"hours":    "abc",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.createEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEntriesTableReturnsPartial(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, "")

	req := httptest.NewRequest("GET", "/entries/table", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.entriesTable(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")
}

func TestDeleteEntryRedirects(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, "")

	req := httptest.NewRequest("POST", "/entries/1/delete", nil)
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.deleteEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Location"), "/entries")
}

func TestEditEntryFormReturnsPartial(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, "Client meeting")

	req := httptest.NewRequest("GET", "/entries/1/edit", nil)
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.editEntryForm(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")
}

func TestEditEntryFormNotFound(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := httptest.NewRequest("GET", "/entries/999/edit", nil)
	req.SetPathValue("id", "999")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.editEntryForm(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestUpdateEntrySuccess(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, "")

	req := newFormRequest("/entries/1", urlencode(map[string]string{
		"date":     "2024-04-01",
		"category": "Design",
		"hours":    "3.0",
		"notes":    "Updated notes",
	}))
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.updateEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Design")
	assert.Contains(t, string(body), "Updated")
}

func TestUpdateEntryMissingCategory(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, "")

	req := newFormRequest("/entries/1", urlencode(map[string]string{
		"date":  "2024-04-01",
		"hours": "3.0",
	}))
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.updateEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

type testApp struct {
	app   *App
	store *db.Store
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	store := testutil.NewTestDB(t)
	cfg := config.Config{
		Address:        ":8080",
		TypstBin:       "",
		OCREnabled:     false,
		DefaultDueDays: 30,
	}
	logger := NewLogger("error", "text")
	multiStore := db.NewMultiStore(t.TempDir(), testutil.MigrationsDir(t))
	multiStore.SetLegacyStore(store)
	return &testApp{app: New(multiStore, nil, nil, nil, cfg, logger), store: store}
}
