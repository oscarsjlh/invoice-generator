package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/testutil"
)

func TestRecoverMiddlewareCatchesPanic(t *testing.T) {
	t.Parallel()

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recover middleware
	handler := recoverMiddleware(panicHandler)

	// First request should panic-recover to 500
	req1 := httptest.NewRequest("GET", "/", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	resp1 := w1.Result()
	assert.Equal(t, http.StatusInternalServerError, resp1.StatusCode)
	body1, _ := io.ReadAll(resp1.Body)
	assert.Contains(t, string(body1), "internal server error")

	// Second request should still work (server continues)
	req2 := httptest.NewRequest("GET", "/", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	resp2 := w2.Result()
	assert.Equal(t, http.StatusInternalServerError, resp2.StatusCode)
}

func TestRequestLoggingMiddleware(t *testing.T) {
	t.Parallel()

	// Create a simple handler
	helloHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	// Wrap with logger and request logging middleware
	handler := LoggerMiddleware(NewLogger("error", "text", false))(RequestLoggingMiddleware()(helloHandler))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "hello")
}

func TestCSRFMiddlewareAllowsGET(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware()(inner)

	req := httptest.NewRequest("GET", "/entries", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestCSRFMiddlewareAllowsHEAD(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware()(inner)

	req := httptest.NewRequest("HEAD", "/entries", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestCSRFMiddlewareAllowsOPTIONS(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware()(inner)

	req := httptest.NewRequest("OPTIONS", "/entries", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestCSRFMiddlewareRejectsPOSTWithoutCookie(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware()(inner)

	req := httptest.NewRequest("POST", "/entries", strings.NewReader(""))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestCSRFMiddlewarePOSTWithMatchingHeader(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware()(inner)

	req := httptest.NewRequest("POST", "/entries", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-token"})
	req.Header.Set("X-CSRF-Token", "test-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestCSRFMiddlewarePOSTWithMatchingFormField(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware()(inner)

	body := "csrf_token=test-token&other_field=value"
	req := httptest.NewRequest("POST", "/entries", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestCSRFMiddlewarePOSTWithMismatchedToken(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware()(inner)

	req := httptest.NewRequest("POST", "/entries", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "correct-token"})
	req.Header.Set("X-CSRF-Token", "wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestCSRFMiddlewareAllowsPublicPaths(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware()(inner)

	for _, path := range []string{"/login", "/register", "/static/css/style.css", "/health"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("POST", path, strings.NewReader(""))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		})
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := SecurityHeadersMiddleware()(inner)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "default-src 'self'")
	assert.Contains(t, resp.Header.Get("Permissions-Policy"), "geolocation=()")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "ok")
}

func TestIsSafeMethod(t *testing.T) {
	t.Parallel()
	assert.True(t, isSafeMethod("GET"))
	assert.True(t, isSafeMethod("HEAD"))
	assert.True(t, isSafeMethod("OPTIONS"))
	assert.False(t, isSafeMethod("POST"))
	assert.False(t, isSafeMethod("PUT"))
	assert.False(t, isSafeMethod("PATCH"))
	assert.False(t, isSafeMethod("DELETE"))
}

func TestExtractCSRFTokenFromHeader(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	req.Header.Set("X-CSRF-Token", "header-token")
	assert.Equal(t, "header-token", extractCSRFToken(req))
}

func TestExtractCSRFTokenFromForm(t *testing.T) {
	t.Parallel()
	body := "csrf_token=form-token"
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	assert.Equal(t, "form-token", extractCSRFToken(req))
}

func TestExtractCSRFTokenEmpty(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	assert.Equal(t, "", extractCSRFToken(req))
}

func TestIsPublicPath(t *testing.T) {
	t.Parallel()
	assert.True(t, isPublicPath("/login"))
	assert.True(t, isPublicPath("/login/begin"))
	assert.True(t, isPublicPath("/register"))
	assert.True(t, isPublicPath("/register/begin"))
	assert.True(t, isPublicPath("/static/"))
	assert.True(t, isPublicPath("/static/js/app.js"))
	assert.True(t, isPublicPath("/health"))
	assert.False(t, isPublicPath("/login-extra"))
	assert.False(t, isPublicPath("/register-extra"))
	assert.False(t, isPublicPath("/entries"))
	assert.False(t, isPublicPath("/invoices"))
	assert.False(t, isPublicPath("/settings"))
	assert.False(t, isPublicPath("/"))
}

func TestAppPublicPathRemovesRegisterWhenDisabled(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.RegistrationEnabled = false

	assert.True(t, ta.app.isPublicPath("/login"))
	assert.False(t, ta.app.isPublicPath("/register"))
	assert.False(t, ta.app.isPublicPath("/register/begin"))
}

func TestAuthMiddlewarePublicPathPassesThrough(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := ta.app.authMiddleware(inner)

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestAuthMiddlewareNoSessionRedirects(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := ta.app.authMiddleware(inner)

	req := httptest.NewRequest("GET", "/entries", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
	assert.Equal(t, "/login?next=%2Fentries&notice=signin_required", w.Result().Header.Get("Location"))
}

func TestAuthMiddlewareNoSessionPostRedirectsWithoutNext(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := ta.app.authMiddleware(inner)

	req := httptest.NewRequest("POST", "/entries", strings.NewReader(""))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
	assert.Equal(t, "/login?notice=signin_required", w.Result().Header.Get("Location"))
}

func TestAuthMiddlewareRegisterDisabledReturns404(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)
	ta.app.cfg.RegistrationEnabled = false

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := ta.app.authMiddleware(inner)

	req := httptest.NewRequest("GET", "/register", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestAuthMiddlewareValidSessionPassesThrough(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	id, _ := ta.adb.CreateUser("testuser", "Test User")

	sessW := httptest.NewRecorder()
	sessReq := httptest.NewRequest("GET", "/", nil)
	assert.NoError(t, ta.sc.Set(sessW, sessReq, id))
	sessW.Flush()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		store := StoreFromContext(r.Context())
		user := UserFromContext(r.Context())
		assert.NotNil(t, store)
		assert.NotNil(t, user)
		assert.Equal(t, "testuser", user.Username)
		w.WriteHeader(http.StatusOK)
	})
	handler := ta.app.authMiddleware(inner)

	req := httptest.NewRequest("GET", "/entries", nil)
	for _, c := range sessW.Result().Cookies() {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestAuthMiddlewareDisabledWithLegacyStore(t *testing.T) {
	t.Parallel()
	store := testutil.NewTestDB(t)
	cfg := config.Config{
		Address:     ":8080",
		AuthEnabled: false,
	}
	logger := NewLogger("error", "text", false)

	migDir := testutil.MigrationsDir(t)
	multiStore := db.NewMultiStore(t.TempDir(), migDir)
	multiStore.SetLegacyStore(store)

	app := New(multiStore, nil, nil, nil, cfg, logger)
	app.SetLegacyStore(store)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := StoreFromContext(r.Context())
		u := UserFromContext(r.Context())
		assert.NotNil(t, s)
		assert.NotNil(t, u)
		assert.Equal(t, "anonymous", u.Username)
		w.WriteHeader(http.StatusOK)
	})
	handler := app.authMiddleware(inner)

	req := httptest.NewRequest("GET", "/entries", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}
