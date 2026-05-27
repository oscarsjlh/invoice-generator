package handler

import (
	"net/http"

	"invoice-app/internal/auth"
)

func CSRFMiddleware() func(http.Handler) http.Handler {
	return CSRFMiddlewareWithPublic(isPublicPath)
}

func CSRFMiddlewareWithPublic(publicPath func(string) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if publicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			provided := extractCSRFToken(r)
			if provided == "" || !auth.ValidateCSRFToken(r, provided) {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func extractCSRFToken(r *http.Request) string {
	if t := r.Header.Get("X-CSRF-Token"); t != "" {
		return t
	}

	if t := r.Header.Get("X-Csrf-Token"); t != "" {
		return t
	}

	if err := r.ParseMultipartForm(32 << 20); err == nil {
		if t := r.FormValue("csrf_token"); t != "" {
			return t
		}
	}

	if err := r.ParseForm(); err == nil {
		if t := r.FormValue("csrf_token"); t != "" {
			return t
		}
	}

	return ""
}
