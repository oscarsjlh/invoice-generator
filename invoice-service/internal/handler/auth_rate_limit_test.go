package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthRateLimiterReturns429(t *testing.T) {
	t.Parallel()
	limiter := NewAuthRateLimiter(false)
	handler := limiter.Middleware(authLimitRegisterBegin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/register/begin", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	}

	req := httptest.NewRequest("POST", "/register/begin", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Result().StatusCode)
	assert.NotEmpty(t, w.Result().Header.Get("Retry-After"))
}

func TestAuthRateLimiterUsesTrustedForwardedFor(t *testing.T) {
	t.Parallel()
	limiter := NewAuthRateLimiter(true)
	handler := limiter.Middleware(authLimitRegisterBegin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/register/begin", nil)
		req.RemoteAddr = "198.51.100.1:1234"
		req.Header.Set("X-Forwarded-For", "203.0.113.20, 198.51.100.1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	}

	req := httptest.NewRequest("POST", "/register/begin", nil)
	req.RemoteAddr = "198.51.100.2:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.20, 198.51.100.2")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Result().StatusCode)
}
