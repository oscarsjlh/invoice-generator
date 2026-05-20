package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
		w.Write([]byte("hello"))
	})

	// Wrap with request logging middleware
	handler := RequestLoggingMiddleware()(helloHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "hello")
}
