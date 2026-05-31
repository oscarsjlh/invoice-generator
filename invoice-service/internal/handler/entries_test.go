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
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-01-01", "", 150))

	req := newFormRequest("/entries", urlencode(map[string]string{
		"date":     "2024-03-15",
		"category": "Consulting",
		"hours":    "4.5",
		"notes":    "Client meeting",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.createEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Location"), "/entries")
}

func TestCreateEntryUnknownCategoryWarns(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-01-01", "", 150))

	req := newFormRequest("/entries", urlencode(map[string]string{
		"date":     "2024-03-15",
		"category": "Typo",
		"hours":    "4.5",
		"year":     "2024",
		"month":    "03",
	}))
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.createEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "/entries?")
	assert.Contains(t, location, "year=2024")
	assert.Contains(t, location, "month=03")
	assert.Contains(t, location, "does+not+exist")

	entries, err := ta.store.ListEntries()
	require.NoError(t, err)
	assert.Empty(t, entries)
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

	ta.eh.createEntry(w, req)

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

	ta.eh.createEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEntriesTableReturnsPartial(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))

	req := httptest.NewRequest("GET", "/entries/table", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.entriesTable(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")
}

func TestEntriesTableShowsMissingRateWarning(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-04-01", "", 150.00))

	req := httptest.NewRequest("GET", "/entries/table", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.entriesTable(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	assert.Contains(t, html, "missing-rate-cell")
	assert.Contains(t, html, "Rate does not exist")
}

func TestEntriesTableRespectsYearMonthFilters(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateEntry("2024-04-15", "Design", 2.0, ""))
	require.NoError(t, ta.store.CreateEntry("2023-03-15", "Research", 1.0, ""))

	req := httptest.NewRequest("GET", "/entries/table?year=2024&month=03", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.entriesTable(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	assert.Contains(t, html, "Consulting")
	assert.NotContains(t, html, "15 Apr 2024")
	assert.NotContains(t, html, "15 Mar 2023")
	assert.Contains(t, html, `name="year"`)
	assert.Contains(t, html, `value="2024" selected`)
	assert.Contains(t, html, `name="month"`)
	assert.Contains(t, html, `value="03" selected`)
	assert.Contains(t, html, `hx-get="/entries/1/edit?month=03&amp;year=2024"`)
	assert.Contains(t, html, `hx-post="/entries/1/delete?month=03&amp;year=2024"`)
}

func TestEntriesTableRespectsRateFilterWithYearMonth(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateRate("Consulting", "2024-01-01", "", 150.00))
	require.NoError(t, ta.store.CreateRate("Design", "2024-01-01", "", 120.00))
	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateEntry("2024-03-16", "Design", 2.0, ""))
	require.NoError(t, ta.store.CreateEntry("2024-04-15", "Consulting", 1.0, ""))

	req := httptest.NewRequest("GET", "/entries/table?year=2024&month=03&rate=Consulting", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.entriesTable(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	assert.Contains(t, html, "Consulting")
	assert.NotContains(t, html, "16 Mar 2024")
	assert.NotContains(t, html, "15 Apr 2024")
	assert.Contains(t, html, `name="rate"`)
	assert.Contains(t, html, `value="Consulting" selected`)
	assert.Contains(t, html, `hx-get="/entries/1/edit?month=03&amp;rate=Consulting&amp;year=2024"`)
	assert.Contains(t, html, `hx-post="/entries/1/delete?month=03&amp;rate=Consulting&amp;year=2024"`)
}

func TestDeleteEntryRedirects(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))

	req := httptest.NewRequest("POST", "/entries/1/delete", nil)
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.deleteEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Location"), "/entries")
}

func TestEditEntryFormReturnsPartial(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, "Client meeting"))
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-01-01", "", 150))

	req := httptest.NewRequest("GET", "/entries/1/edit?year=2024&month=03", nil)
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.editEntryForm(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Consulting")
	assert.Contains(t, string(body), `hx-post="/entries/1?month=03&amp;year=2024"`)
	assert.Contains(t, string(body), `hx-get="/entries/table?month=03&amp;year=2024"`)
}

func TestEditEntryFormNotFound(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := httptest.NewRequest("GET", "/entries/999/edit", nil)
	req.SetPathValue("id", "999")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.editEntryForm(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestUpdateEntrySuccess(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateRate("Design", "2024-01-01", "", 120))

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

	ta.eh.updateEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Design")
	assert.Contains(t, string(body), "Updated")
}

func TestUpdateEntryUnknownCategoryWarns(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))
	require.NoError(t, ta.store.CreateRate("Consulting", "2024-01-01", "", 150))

	req := newFormRequest("/entries/1?year=2024&month=03", urlencode(map[string]string{
		"date":     "2024-03-16",
		"category": "Typo",
		"hours":    "3.0",
	}))
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.updateEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	assert.Contains(t, html, `Category &#34;Typo&#34; does not exist`)
	assert.Contains(t, html, "Consulting")
	assert.NotContains(t, html, ">Typo<")

	entry, err := ta.store.GetEntry(1)
	require.NoError(t, err)
	assert.Equal(t, "Consulting", entry.Category)
}

func TestUpdateEntryMissingCategory(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	require.NoError(t, ta.store.CreateEntry("2024-03-15", "Consulting", 4.5, ""))

	req := newFormRequest("/entries/1", urlencode(map[string]string{
		"date":  "2024-04-01",
		"hours": "3.0",
	}))
	req.SetPathValue("id", "1")
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.eh.updateEntry(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

type testApp struct {
	app   *App
	store *db.Store
	eh    *EntryHandlers
	rh    *RateHandlers
	ih    *InvoiceHandlers
	sh    *SettingsHandlers
	oh    *OCRHandlers
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
	logger := NewLogger("error", "text", false)
	multiStore := db.NewMultiStore(t.TempDir(), testutil.MigrationsDir(t))
	multiStore.SetLegacyStore(store)
	app := New(multiStore, nil, nil, nil, cfg, logger)
	return &testApp{
		app:   app,
		store: store,
		eh:    NewEntryHandlers(app.renderer),
		rh:    NewRateHandlers(app.renderer),
		ih:    NewInvoiceHandlers(app.renderer, cfg),
		sh:    NewSettingsHandlers(app.renderer),
		oh:    NewOCRHandlers(app.renderer, nil, cfg),
	}
}
