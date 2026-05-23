package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"invoice-app/internal/auth"
	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/testutil"
)

func newTestAppWithAuth(t *testing.T) *testAppWithAuth {
	t.Helper()
	store := testutil.NewTestDB(t)
	adb := testutil.NewTestAuthDB(t)
	migDir := testutil.MigrationsDir(t)
	multiStore := db.NewMultiStore(t.TempDir(), migDir)
	multiStore.SetLegacyStore(store)

	cfg := config.Config{
		Address:        ":8080",
		AuthEnabled:    true,
		DefaultDueDays: 30,
	}
	logger := NewLogger("error", "text", true)

	wm, err := auth.NewWebAuthnManager(adb, auth.AuthConfig{
		RPDisplayName: "Test App",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
		SessionTTL:    24 * time.Hour,
	})
	require.NoError(t, err)

	sm := auth.NewSessionManager(adb, 24*time.Hour, false)
	app := New(multiStore, adb, wm, sm, cfg, logger)

	return &testAppWithAuth{
		app:   app,
		adb:   adb,
		store: store,
		sm:    sm,
	}
}

type testAppWithAuth struct {
	app   *App
	adb   *db.AuthDB
	store *db.Store
	sm    *auth.SessionManager
}

func TestLoginPageRedirectsIfAlreadyLoggedIn(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	id, _ := ta.adb.CreateUser("alice", "Alice")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/login", nil)
	require.NoError(t, ta.sm.CreateSession(w, req, id))
	w.Flush()

	req2 := httptest.NewRequest("GET", "/login", nil)
	for _, c := range w.Result().Cookies() {
		req2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	ta.app.loginPage(w2, req2)
	assert.Equal(t, http.StatusSeeOther, w2.Result().StatusCode)
}

func TestLoginPageRendersForm(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	ta.app.loginPage(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "login")
}

func TestRegisterPageRedirectsIfAlreadyLoggedIn(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	id, _ := ta.adb.CreateUser("bob", "Bob")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/register", nil)
	require.NoError(t, ta.sm.CreateSession(w, req, id))
	w.Flush()

	req2 := httptest.NewRequest("GET", "/register", nil)
	for _, c := range w.Result().Cookies() {
		req2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	ta.app.registerPage(w2, req2)
	assert.Equal(t, http.StatusSeeOther, w2.Result().StatusCode)
}

func TestRegisterPageRendersForm(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	req := httptest.NewRequest("GET", "/register", nil)
	w := httptest.NewRecorder()
	ta.app.registerPage(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "register")
}

func TestBeginRegistrationSuccess(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"username":"eve","displayName":"Eve"}`))
	req := httptest.NewRequest("POST", "/api/register/begin", body)
	w := httptest.NewRecorder()
	ta.app.beginRegistration(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Result().Body).Decode(&resp))
	assert.NotEmpty(t, resp["session_id"])
	assert.NotNil(t, resp["options"])
}

func TestBeginRegistrationDuplicate(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	_, err := ta.adb.CreateUser("frank", "Frank")
	require.NoError(t, err)

	body := bytes.NewReader([]byte(`{"username":"frank","displayName":"Frank Dup"}`))
	req := httptest.NewRequest("POST", "/api/register/begin", body)
	w := httptest.NewRecorder()
	ta.app.beginRegistration(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestBeginLoginNoCredentials(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	_, err := ta.adb.CreateUser("grace", "Grace")
	require.NoError(t, err)

	body := bytes.NewReader([]byte(`{"username":"grace"}`))
	req := httptest.NewRequest("POST", "/api/login/begin", body)
	w := httptest.NewRecorder()
	ta.app.beginLogin(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestBeginLoginUserNotFound(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"username":"nobody"}`))
	req := httptest.NewRequest("POST", "/api/login/begin", body)
	w := httptest.NewRecorder()
	ta.app.beginLogin(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestFinishRegistrationInvalidSession(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"session_id":"nonexistent"}`))
	req := httptest.NewRequest("POST", "/api/register/finish", body)
	w := httptest.NewRecorder()
	ta.app.finishRegistration(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestFinishLoginInvalidSession(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"session_id":"nonexistent"}`))
	req := httptest.NewRequest("POST", "/api/login/finish", body)
	w := httptest.NewRecorder()
	ta.app.finishLogin(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestLogoutRedirects(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	id, _ := ta.adb.CreateUser("heidi", "Heidi")
	sessW := httptest.NewRecorder()
	sessReq := httptest.NewRequest("GET", "/", nil)
	require.NoError(t, ta.sm.CreateSession(sessW, sessReq, id))
	sessW.Flush()

	body := bytes.NewReader([]byte("csrf_token="))
	req := httptest.NewRequest("POST", "/logout", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range sessW.Result().Cookies() {
		req.AddCookie(c)
		if c.Name == "csrf_token" {
			body = bytes.NewReader([]byte("csrf_token=" + c.Value))
			req.Body = io.NopCloser(body)
			req.ContentLength = int64(len("csrf_token=" + c.Value))
		}
	}
	w := httptest.NewRecorder()
	ta.app.logout(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
}

func TestLogoutWithoutCSRFCookie(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte("csrf_token=someval"))
	req := httptest.NewRequest("POST", "/logout", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	ta.app.logout(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}
