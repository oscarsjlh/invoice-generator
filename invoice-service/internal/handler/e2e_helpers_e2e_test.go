//go:build e2e

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2ECreateSessionRequiresGuard(t *testing.T) {
	ta := newTestAppWithAuth(t)

	req := httptest.NewRequest("POST", "/__e2e/session", strings.NewReader(`{"username":"guarded"}`))
	w := httptest.NewRecorder()

	ta.app.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode)
}

func TestE2ECreateSessionSeedsAuthenticatedUser(t *testing.T) {
	t.Setenv("E2E_TEST_HELPERS", "true")
	ta := newTestAppWithAuth(t)

	req := httptest.NewRequest("POST", "/__e2e/session", strings.NewReader(`{"username":"seeded","display_name":"Seeded User"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ta.app.Routes().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Result().StatusCode)
	require.NotEmpty(t, w.Result().Cookies())

	var body e2eSessionResponse
	require.NoError(t, json.NewDecoder(w.Result().Body).Decode(&body))
	assert.Equal(t, "seeded", body.Username)

	protectedReq := httptest.NewRequest("GET", "/entries", nil)
	for _, cookie := range w.Result().Cookies() {
		protectedReq.AddCookie(cookie)
	}
	protectedW := httptest.NewRecorder()
	ta.app.Routes().ServeHTTP(protectedW, protectedReq)

	assert.Equal(t, http.StatusOK, protectedW.Result().StatusCode)
}
