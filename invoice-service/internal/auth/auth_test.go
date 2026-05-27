package auth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"invoice-app/internal/db"
	"invoice-app/internal/testutil"
)

func TestNewWebAuthnUser(t *testing.T) {
	t.Parallel()
	user := &db.User{
		ID:          42,
		Username:    "testuser",
		DisplayName: "Test User",
	}
	wu := NewWebAuthnUser(user)
	assert.Equal(t, []byte("42"), wu.WebAuthnID())
	assert.Equal(t, "testuser", wu.WebAuthnName())
	assert.Equal(t, "Test User", wu.WebAuthnDisplayName())
	assert.NotNil(t, wu.WebAuthnCredentials())
	assert.Empty(t, wu.WebAuthnCredentials())
	assert.Equal(t, "", wu.WebAuthnIcon())
}

func TestWebAuthnUserDisplayNameFallsBackToUsername(t *testing.T) {
	t.Parallel()
	user := &db.User{
		ID:          1,
		Username:    "nobody",
		DisplayName: "",
	}
	wu := NewWebAuthnUser(user)
	assert.Equal(t, "nobody", wu.WebAuthnDisplayName())
}

func TestWebAuthnUserCredentialsEmptyByDefault(t *testing.T) {
	t.Parallel()
	user := &db.User{
		ID:       1,
		Username: "test",
	}
	wu := NewWebAuthnUser(user)
	creds := wu.WebAuthnCredentials()
	assert.Empty(t, creds)
	assert.Equal(t, creds, wu.Credentials())
}

func TestBeginRegistrationUsernameTaken(t *testing.T) {
	t.Parallel()
	adb := testutil.NewTestAuthDB(t)
	m, err := NewWebAuthnManager(adb, AuthConfig{
		RPDisplayName: "Test",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
		SessionTTL:    24 * time.Hour,
	})
	require.NoError(t, err)

	_, err = adb.CreateUser("alice", "Alice")
	require.NoError(t, err)

	_, _, err = m.BeginRegistration("alice", "Alice Duplicate")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username already taken")
}

func TestBeginLoginUserNotFound(t *testing.T) {
	t.Parallel()
	adb := testutil.NewTestAuthDB(t)
	m, err := NewWebAuthnManager(adb, AuthConfig{
		RPDisplayName: "Test",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
		SessionTTL:    24 * time.Hour,
	})
	require.NoError(t, err)

	_, _, _, err = m.BeginLogin("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestFinishRegistrationInvalidSession(t *testing.T) {
	t.Parallel()
	adb := testutil.NewTestAuthDB(t)
	m, err := NewWebAuthnManager(adb, AuthConfig{
		RPDisplayName: "Test",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
		SessionTTL:    24 * time.Hour,
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/register", nil)
	_, _, err = m.FinishRegistration("nonexistent-session", req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no registration session found")
}

func TestFinishLoginInvalidSession(t *testing.T) {
	t.Parallel()
	adb := testutil.NewTestAuthDB(t)
	m, err := NewWebAuthnManager(adb, AuthConfig{
		RPDisplayName: "Test",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
		SessionTTL:    24 * time.Hour,
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/login", nil)
	_, err = m.FinishLogin("nonexistent-session", req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no login session found")
}

func TestSessionCookieCreateAndClear(t *testing.T) {
	t.Parallel()
	adb := testutil.NewTestAuthDB(t)
	sc := NewSessionCookie(adb, 1*time.Hour, false)

	id, _ := adb.CreateUser("sessionuser", "Session User")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	err := sc.Set(w, req, id)
	require.NoError(t, err)

	cookies := w.Result().Cookies()
	var sessionCookie, csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "invoice_session" {
			sessionCookie = c
		}
		if c.Name == "csrf_token" {
			csrfCookie = c
		}
	}
	require.NotNil(t, sessionCookie, "invoice_session cookie not set")
	require.NotNil(t, csrfCookie, "csrf_token cookie not set")
	assert.True(t, sessionCookie.HttpOnly)
	assert.False(t, csrfCookie.HttpOnly)
	assert.Equal(t, "/", sessionCookie.Path)
	assert.NotEmpty(t, sessionCookie.Value)

	w2 := httptest.NewRecorder()
	err = sc.Clear(w2, req)
	require.NoError(t, err)

	cookies2 := w2.Result().Cookies()
	for _, c := range cookies2 {
		if c.Name == "invoice_session" || c.Name == "csrf_token" {
			assert.Equal(t, -1, c.MaxAge)
		}
	}
}

func TestSessionCookieGetNoCookie(t *testing.T) {
	t.Parallel()
	adb := testutil.NewTestAuthDB(t)
	sc := NewSessionCookie(adb, 1*time.Hour, false)

	req := httptest.NewRequest("GET", "/", nil)
	user, err := sc.Get(req)
	require.NoError(t, err)
	assert.Nil(t, user)
}

func TestSessionCookieGetInvalidToken(t *testing.T) {
	t.Parallel()
	adb := testutil.NewTestAuthDB(t)
	sc := NewSessionCookie(adb, 1*time.Hour, false)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "invoice_session",
		Value: "invalid-base64!!!",
	})
	user, err := sc.Get(req)
	require.NoError(t, err)
	assert.Nil(t, user)
}

func TestSessionCookieGetUnknownToken(t *testing.T) {
	t.Parallel()
	adb := testutil.NewTestAuthDB(t)
	sc := NewSessionCookie(adb, 1*time.Hour, false)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "invoice_session",
		Value: "dGVzdF90b2tlbl92YWx1ZV8xMjM0NQ", // valid base64 but unknown token
	})
	user, err := sc.Get(req)
	require.NoError(t, err)
	assert.Nil(t, user)
}

func TestSessionCookieIsSecure(t *testing.T) {
	t.Parallel()
	adb := testutil.NewTestAuthDB(t)

	t.Run("non-TLS without proxy", func(t *testing.T) {
		sc := NewSessionCookie(adb, 1*time.Hour, false)
		req := httptest.NewRequest("GET", "/", nil)
		assert.False(t, sc.IsSecure(req))
	})

	t.Run("TLS connection", func(t *testing.T) {
		sc := NewSessionCookie(adb, 1*time.Hour, false)
		req := httptest.NewRequest("GET", "/", nil)
		req.TLS = &tls.ConnectionState{}
		assert.True(t, sc.IsSecure(req))
	})

	t.Run("trusted proxy with X-Forwarded-Proto", func(t *testing.T) {
		sc := NewSessionCookie(adb, 1*time.Hour, true)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		assert.True(t, sc.IsSecure(req))
	})
}

func TestAuthConfig(t *testing.T) {
	t.Parallel()
	cfg := AuthConfig{
		RPID:          "example.com",
		RPOrigins:     []string{"https://example.com"},
		RPDisplayName: "My App",
		SessionTTL:    12 * time.Hour,
	}
	assert.Equal(t, "example.com", cfg.RPID)
	assert.Equal(t, "My App", cfg.RPDisplayName)
	assert.Equal(t, 12*time.Hour, cfg.SessionTTL)
}
