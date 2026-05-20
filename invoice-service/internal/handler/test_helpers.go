package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

func urlencode(values map[string]string) string {
	var parts []string
	for k, v := range values {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, "&")
}

// newFormRequest creates a POST request with form-encoded body and correct Content-Type.
func newFormRequest(url, body string) *http.Request {
	req := httptest.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}
