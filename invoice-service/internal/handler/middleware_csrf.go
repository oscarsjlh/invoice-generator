package handler

import "net/http"

func CSRFMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			expected, err := r.Cookie("csrf_token")
			if err != nil || expected.Value == "" {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			provided := extractCSRFToken(r)
			if provided == "" || provided != expected.Value {
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

	if err := r.ParseForm(); err == nil {
		if t := r.FormValue("csrf_token"); t != "" {
			return t
		}
	}

	if err := r.ParseMultipartForm(32 << 20); err == nil {
		if t := r.FormValue("csrf_token"); t != "" {
			return t
		}
	}

	return ""
}
