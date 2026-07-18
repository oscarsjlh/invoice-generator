package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/testutil"
)

func TestOCRUploadPageWhenDisabled(t *testing.T) {
	t.Parallel()
	ta := newTestApp(t)

	req := httptest.NewRequest("GET", "/ocr/import", nil)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.oh.ocrUploadPage(w, req)

	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestOCRUploadPageWhenEnabled(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{
		Address:     ":8080",
		AuthEnabled: false,
		OCREnabled:  true,
	}
	logger := NewLogger("error", "text", true)
	migDir := testutil.MigrationsDir(t)
	multiStore := db.NewMultiStore(t.TempDir(), migDir)
	multiStore.SetLegacyStore(store)
	app := New(multiStore, nil, nil, nil, cfg, logger)
	app.SetLegacyStore(store)
	oh := NewOCRHandlers(app.renderer, app.ocrJobs, cfg)

	req := httptest.NewRequest("GET", "/ocr/import", nil)
	ctx := WithTestStore(req.Context(), store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	oh.ocrUploadPage(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "OCR")
}

func TestOCRSessionStatusHandler(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.OCREnabled = true
	ta.oh = NewOCRHandlers(ta.app.renderer, ta.app.ocrJobs, ta.app.cfg)

	store := ta.store
	sessionID, err := store.CreateOCRSession()
	require.NoError(t, err)
	require.NoError(t, store.UpdateOCRSessionState(sessionID, "processing", ""))

	idStr := strconv.FormatInt(sessionID, 10)
	req := httptest.NewRequest("GET", "/ocr/import/"+idStr, nil)
	req.SetPathValue("id", idStr)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.oh.ocrSessionStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "OCR")
}

func TestOCRSessionStatusIncludesDraftCategories(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.OCREnabled = true
	ta.oh = NewOCRHandlers(ta.app.renderer, ta.app.ocrJobs, ta.app.cfg)

	store := ta.store
	sessionID, err := store.CreateOCRSession()
	require.NoError(t, err)
	require.NoError(t, store.UpdateOCRSessionState(sessionID, "review_ready", ""))
	require.NoError(t, store.SaveDraftEntries(sessionID, []db.OCRDraftEntry{
		{
			DateNormalized:     "2026-05-24",
			CategoryRaw:        "Consultng",
			CategoryNormalized: "Consulting",
			HoursRaw:           "2",
			HoursNormalized:    2,
			Confidence:         0.8,
		},
	}))

	idStr := strconv.FormatInt(sessionID, 10)
	req := httptest.NewRequest("GET", "/ocr/import/"+idStr, nil)
	req.SetPathValue("id", idStr)
	req = req.WithContext(WithTestStore(req.Context(), ta.store))
	w := httptest.NewRecorder()

	ta.oh.ocrSessionStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), `<option value="Consulting" selected>Consulting</option>`)
}

func TestOCRSessionStatusIncludesDraftDeleteAction(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.OCREnabled = true
	ta.oh = NewOCRHandlers(ta.app.renderer, ta.app.ocrJobs, ta.app.cfg)

	store := ta.store
	sessionID, err := store.CreateOCRSession()
	require.NoError(t, err)
	require.NoError(t, store.UpdateOCRSessionState(sessionID, "review_ready", ""))
	require.NoError(t, store.SaveDraftEntries(sessionID, []db.OCRDraftEntry{
		{
			DateNormalized:     "2026-05-24",
			CategoryNormalized: "Consulting",
			HoursNormalized:    2,
			Confidence:         0.8,
		},
	}))
	drafts, err := store.GetDraftEntries(sessionID)
	require.NoError(t, err)
	require.Len(t, drafts, 1)

	idStr := strconv.FormatInt(sessionID, 10)
	req := httptest.NewRequest("GET", "/ocr/import/"+idStr, nil)
	req.SetPathValue("id", idStr)
	req = req.WithContext(WithTestStore(req.Context(), ta.store))
	w := httptest.NewRecorder()

	ta.oh.ocrSessionStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "/ocr/import/"+idStr+"/drafts/"+strconv.FormatInt(drafts[0].ID, 10)+"/delete")
}

func TestOCRDeleteDraftHandlerRemovesDraftFromSession(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.OCREnabled = true
	ta.oh = NewOCRHandlers(ta.app.renderer, ta.app.ocrJobs, ta.app.cfg)

	store := ta.store
	sessionID, err := store.CreateOCRSession()
	require.NoError(t, err)
	require.NoError(t, store.UpdateOCRSessionState(sessionID, "review_ready", ""))
	require.NoError(t, store.SaveDraftEntries(sessionID, []db.OCRDraftEntry{
		{
			DateNormalized:     "2026-05-24",
			CategoryNormalized: "Consulting",
			HoursNormalized:    2,
			Confidence:         0.8,
		},
		{
			DateNormalized:     "2026-05-25",
			CategoryNormalized: "Admin",
			HoursNormalized:    1,
			Confidence:         0.7,
		},
	}))
	drafts, err := store.GetDraftEntries(sessionID)
	require.NoError(t, err)
	require.Len(t, drafts, 2)

	idStr := strconv.FormatInt(sessionID, 10)
	draftIDStr := strconv.FormatInt(drafts[0].ID, 10)
	req := httptest.NewRequest("POST", "/ocr/import/"+idStr+"/drafts/"+draftIDStr+"/delete", nil)
	req.SetPathValue("id", idStr)
	req.SetPathValue("draftID", draftIDStr)
	req = req.WithContext(WithTestStore(req.Context(), ta.store))
	w := httptest.NewRecorder()

	ta.oh.ocrDeleteDraft(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
	assert.Equal(t, "/ocr/import/"+idStr+"?notice=Draft+entry+deleted", w.Result().Header.Get("Location"))
	remaining, err := store.GetDraftEntries(sessionID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, drafts[1].ID, remaining[0].ID)
}

func TestOCRDeleteDraftHandlerReturnsNoContentForHTMX(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.OCREnabled = true
	ta.oh = NewOCRHandlers(ta.app.renderer, ta.app.ocrJobs, ta.app.cfg)

	store := ta.store
	sessionID, err := store.CreateOCRSession()
	require.NoError(t, err)
	require.NoError(t, store.UpdateOCRSessionState(sessionID, "review_ready", ""))
	require.NoError(t, store.SaveDraftEntries(sessionID, []db.OCRDraftEntry{
		{
			DateNormalized:     "2026-05-24",
			CategoryNormalized: "Consulting",
			HoursNormalized:    2,
			Confidence:         0.8,
		},
		{
			DateNormalized:     "2026-05-25",
			CategoryNormalized: "Admin",
			HoursNormalized:    1,
			Confidence:         0.7,
		},
	}))
	drafts, err := store.GetDraftEntries(sessionID)
	require.NoError(t, err)
	require.Len(t, drafts, 2)

	idStr := strconv.FormatInt(sessionID, 10)
	draftIDStr := strconv.FormatInt(drafts[0].ID, 10)
	req := httptest.NewRequest("POST", "/ocr/import/"+idStr+"/drafts/"+draftIDStr+"/delete", nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", idStr)
	req.SetPathValue("draftID", draftIDStr)
	req = req.WithContext(WithTestStore(req.Context(), ta.store))
	w := httptest.NewRecorder()

	ta.oh.ocrDeleteDraft(w, req)

	assert.Equal(t, http.StatusNoContent, w.Result().StatusCode)
	remaining, err := store.GetDraftEntries(sessionID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, drafts[1].ID, remaining[0].ID)
}

func TestOCRDeleteSessionHandler(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.OCREnabled = true
	ta.oh = NewOCRHandlers(ta.app.renderer, ta.app.ocrJobs, ta.app.cfg)

	store := ta.store
	sessionID, err := store.CreateOCRSession()
	require.NoError(t, err)

	idStr := strconv.FormatInt(sessionID, 10)
	req := httptest.NewRequest("POST", "/ocr/import/"+idStr+"/delete", nil)
	req.SetPathValue("id", idStr)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.oh.ocrDeleteSession(w, req)
	assert.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
}
