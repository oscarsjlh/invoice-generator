//go:build !e2e

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE2EHelperRoutesAbsentFromNormalBuild(t *testing.T) {
	t.Parallel()
	ta := newTestAppWithAuth(t)

	req := httptest.NewRequest("POST", "/__e2e/session", strings.NewReader(`{"username":"alice"}`))
	w := httptest.NewRecorder()

	ta.app.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Result().StatusCode)
	assert.Contains(t, w.Result().Header.Get("Location"), "/login")
	assert.False(t, isPublicPath("/__e2e/session"))
}
