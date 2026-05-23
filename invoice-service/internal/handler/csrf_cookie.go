package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request, secure bool) {
	if !isSafeMethod(r.Method) {
		return
	}
	if _, err := r.Cookie("csrf_token"); err == nil {
		return
	}

	csrf := make([]byte, 16)
	if _, err := rand.Read(csrf); err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    base64.RawURLEncoding.EncodeToString(csrf),
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
