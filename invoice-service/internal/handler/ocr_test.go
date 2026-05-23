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

	ta.app.ocrUploadPage(w, req)

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
	logger := NewLogger("error", "text")
	migDir := testutil.MigrationsDir(t)
	multiStore := db.NewMultiStore(t.TempDir(), migDir)
	multiStore.SetLegacyStore(store)
	app := New(multiStore, nil, nil, nil, cfg, logger)
	app.legacyStore = store

	req := httptest.NewRequest("GET", "/ocr/import", nil)
	ctx := WithTestStore(req.Context(), store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	app.ocrUploadPage(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "OCR")
}

func TestOCRSessionStatusHandler(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.OCREnabled = true

	store := ta.store
	sessionID, err := store.CreateOCRSession()
	require.NoError(t, err)
	store.UpdateOCRSessionState(sessionID, "processing", "")

	idStr := strconv.FormatInt(sessionID, 10)
	req := httptest.NewRequest("GET", "/ocr/import/"+idStr, nil)
	req.SetPathValue("id", idStr)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.ocrSessionStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "OCR")
}

func TestOCRDeleteSessionHandler(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.OCREnabled = true

	store := ta.store
	sessionID, err := store.CreateOCRSession()
	require.NoError(t, err)

	idStr := strconv.FormatInt(sessionID, 10)
	req := httptest.NewRequest("POST", "/ocr/import/"+idStr+"/delete", nil)
	req.SetPathValue("id", idStr)
	ctx := WithTestStore(req.Context(), ta.store)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ta.app.ocrDeleteSession(w, req)
	assert.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
}
