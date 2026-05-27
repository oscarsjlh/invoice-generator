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
		Address:             ":8080",
		AuthEnabled:         true,
		RegistrationEnabled: true,
		DefaultDueDays:      30,
	}
	logger := NewLogger("error", "text", true)

	wm, err := auth.NewWebAuthnManager(adb, auth.AuthConfig{
		RPDisplayName: "Test App",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
		SessionTTL:    24 * time.Hour,
	})
	require.NoError(t, err)

	sc := auth.NewSessionCookie(adb, 24*time.Hour, false)
	app := New(multiStore, adb, wm, sc, cfg, logger)

	ah := NewAuthHandlers(wm, sc, app.renderer, app.authLimiter, cfg.AuthEnabled, cfg.RegistrationEnabled)

	return &testAppWithAuth{
		app:   app,
		adb:   adb,
		store: store,
		sc:    sc,
		ah:    ah,
		eh:    NewEntryHandlers(app.renderer),
		rh:    NewRateHandlers(app.renderer),
		ih:    NewInvoiceHandlers(app.renderer, cfg),
		sh:    NewSettingsHandlers(app.renderer),
		oh:    NewOCRHandlers(app.renderer, app.ocrJobs, cfg),
	}
}

type testAppWithAuth struct {
	app   *App
	adb   *db.AuthDB
	store *db.Store
	sc    *auth.SessionCookie
	ah    *AuthHandlers
	eh    *EntryHandlers
	rh    *RateHandlers
	ih    *InvoiceHandlers
	sh    *SettingsHandlers
	oh    *OCRHandlers
}

func TestLoginPageRedirectsIfAlreadyLoggedIn(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	id, _ := ta.adb.CreateUser("alice", "Alice")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/login", nil)
	require.NoError(t, ta.sc.Set(w, req, id))
	w.Flush()

	req2 := httptest.NewRequest("GET", "/login", nil)
	for _, c := range w.Result().Cookies() {
		req2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	ta.ah.loginPage(w2, req2)
	assert.Equal(t, http.StatusSeeOther, w2.Result().StatusCode)
}

func TestLoginPageRendersForm(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	ta.ah.loginPage(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "login")
	assert.Contains(t, string(body), "/static/auth.js")
}

func TestLoginPageHidesRegisterLinkWhenRegistrationDisabled(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.RegistrationEnabled = false

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	ta.ah.loginPage(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.NotContains(t, string(body), "Create one")
	assert.NotContains(t, string(body), `href="/register"`)
}

func TestRegisterPageRedirectsIfAlreadyLoggedIn(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	id, _ := ta.adb.CreateUser("bob", "Bob")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/register", nil)
	require.NoError(t, ta.sc.Set(w, req, id))
	w.Flush()

	req2 := httptest.NewRequest("GET", "/register", nil)
	for _, c := range w.Result().Cookies() {
		req2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	ta.ah.registerPage(w2, req2)
	assert.Equal(t, http.StatusSeeOther, w2.Result().StatusCode)
}

func TestRegisterPageRendersForm(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	req := httptest.NewRequest("GET", "/register", nil)
	w := httptest.NewRecorder()
	ta.ah.registerPage(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "register")
}

func TestRegisterPageDisabledReturns404(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.RegistrationEnabled = false

	req := httptest.NewRequest("GET", "/register", nil)
	w := httptest.NewRecorder()
	ta.app.Routes().ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestRegisterRoutesDisabledReturn404(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.RegistrationEnabled = false
	handler := ta.app.Routes()

	for _, tc := range []struct {
		method string
		path   string
		body   io.Reader
	}{
		{method: http.MethodGet, path: "/register"},
		{method: http.MethodPost, path: "/register/begin", body: bytes.NewReader([]byte(`{"username":"eve"}`))},
		{method: http.MethodPost, path: "/register/finish", body: bytes.NewReader([]byte(`{"session_id":"x"}`))},
	} {
		req := httptest.NewRequest(tc.method, tc.path, tc.body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
	}
}

func TestBeginRegistrationSuccess(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"username":"eve","displayName":"Eve"}`))
	req := httptest.NewRequest("POST", "/api/register/begin", body)
	w := httptest.NewRecorder()
	ta.ah.beginRegistration(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Result().Body).Decode(&resp))
	assert.NotEmpty(t, resp["session_id"])
	assert.NotNil(t, resp["options"])
}

func TestBeginRegistrationInvalidUsername(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"username":"no spaces"}`))
	req := httptest.NewRequest("POST", "/api/register/begin", body)
	w := httptest.NewRecorder()
	ta.ah.beginRegistration(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Result().Body).Decode(&resp))
	assert.Equal(t, "invalid_username", resp["error"])
	assert.NotEmpty(t, resp["message"])
}

func TestBeginRegistrationNormalizesUsername(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"username":"  Eve_User  ","displayName":"Ignored"}`))
	req := httptest.NewRequest("POST", "/api/register/begin", body)
	w := httptest.NewRecorder()
	ta.ah.beginRegistration(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	existing, err := ta.adb.GetUserByUsernameFold("eve_user")
	require.NoError(t, err)
	assert.Nil(t, existing, "begin registration should not create the user before WebAuthn finishes")
}

func TestBeginRegistrationDisabledReturns404(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.RegistrationEnabled = false
	ta.ah = NewAuthHandlers(ta.app.webAuthn, ta.sc, ta.app.renderer, ta.app.authLimiter, ta.app.cfg.AuthEnabled, ta.app.cfg.RegistrationEnabled)

	body := bytes.NewReader([]byte(`{"username":"eve"}`))
	req := httptest.NewRequest("POST", "/api/register/begin", body)
	w := httptest.NewRecorder()
	ta.ah.beginRegistration(w, req)

	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestBeginRegistrationDuplicate(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	_, err := ta.adb.CreateUser("frank", "Frank")
	require.NoError(t, err)

	body := bytes.NewReader([]byte(`{"username":"frank","displayName":"Frank Dup"}`))
	req := httptest.NewRequest("POST", "/api/register/begin", body)
	w := httptest.NewRecorder()
	ta.ah.beginRegistration(w, req)

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
	ta.ah.beginLogin(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestBeginLoginUserNotFound(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"username":"nobody"}`))
	req := httptest.NewRequest("POST", "/api/login/begin", body)
	w := httptest.NewRecorder()
	ta.ah.beginLogin(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestBeginLoginInvalidUsername(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"username":"bad username"}`))
	req := httptest.NewRequest("POST", "/api/login/begin", body)
	w := httptest.NewRecorder()
	ta.ah.beginLogin(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Result().Body).Decode(&resp))
	assert.Equal(t, "invalid_username", resp["error"])
}

func TestFinishRegistrationInvalidSession(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"session_id":"nonexistent"}`))
	req := httptest.NewRequest("POST", "/api/register/finish", body)
	w := httptest.NewRecorder()
	ta.ah.finishRegistration(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestFinishRegistrationDisabledReturns404(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.RegistrationEnabled = false
	ta.ah = NewAuthHandlers(ta.app.webAuthn, ta.sc, ta.app.renderer, ta.app.authLimiter, ta.app.cfg.AuthEnabled, ta.app.cfg.RegistrationEnabled)

	body := bytes.NewReader([]byte(`{"session_id":"nonexistent"}`))
	req := httptest.NewRequest("POST", "/api/register/finish", body)
	w := httptest.NewRecorder()
	ta.ah.finishRegistration(w, req)

	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestFinishLoginInvalidSession(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	body := bytes.NewReader([]byte(`{"session_id":"nonexistent"}`))
	req := httptest.NewRequest("POST", "/api/login/finish", body)
	w := httptest.NewRecorder()
	ta.ah.finishLogin(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestLogoutRedirects(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	id, _ := ta.adb.CreateUser("heidi", "Heidi")
	sessW := httptest.NewRecorder()
	sessReq := httptest.NewRequest("GET", "/", nil)
	require.NoError(t, ta.sc.Set(sessW, sessReq, id))
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
	ta.ah.logout(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
}

func TestLogoutCSRFRejectedByMiddleware(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	handler := CSRFMiddleware()(http.HandlerFunc(ta.ah.logout))

	body := bytes.NewReader([]byte("csrf_token=someval"))
	req := httptest.NewRequest("POST", "/logout", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestSafeNextPath(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/entries?month=2026-05", safeNextPath("/entries?month=2026-05"))
	assert.Equal(t, "", safeNextPath("https://example.test/entries"))
	assert.Equal(t, "", safeNextPath("//example.test/entries"))
	assert.Equal(t, "", safeNextPath(`/\evil`))
}

func TestAuthNoticeFromRequestIgnoresUnknownCodes(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/login?notice=hello", nil)
	notice, kind := authNoticeFromRequest(req)
	assert.Empty(t, notice)
	assert.Empty(t, kind)
}

func TestUsernamePolicyEndpoint(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	req := httptest.NewRequest("GET", "/auth/username-policy", nil)
	w := httptest.NewRecorder()
	ta.ah.usernamePolicyHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.Equal(t, "application/json", w.Result().Header.Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Result().Body).Decode(&body))
	assert.Equal(t, auth.DefaultUsernamePolicy.Pattern, body["pattern"])
	assert.Equal(t, auth.UsernameValidationMessage, body["message"])
	assert.NotEmpty(t, body["normalization"])
}

func TestUsernamePolicyEndpointIsPublic(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := ta.app.authMiddleware(inner)

	req := httptest.NewRequest("GET", "/auth/username-policy", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}
