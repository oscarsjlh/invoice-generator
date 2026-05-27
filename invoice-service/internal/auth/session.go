package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"invoice-app/internal/db"
)

const sessionCookieName = "invoice_session"

type SessionCookie struct {
	authDB       *db.AuthDB
	ttl          time.Duration
	trustedProxy bool
}

func NewSessionCookie(authDB *db.AuthDB, ttl time.Duration, trustedProxy bool) *SessionCookie {
	return &SessionCookie{authDB: authDB, ttl: ttl, trustedProxy: trustedProxy}
}

func (sc *SessionCookie) Set(w http.ResponseWriter, r *http.Request, userID int64) error {
	token, err := sc.authDB.CreateSession(userID, sc.ttl)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	secure := sc.IsSecure(r)
	encoded := base64.RawURLEncoding.EncodeToString(token)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sc.ttl.Seconds()),
	})

	csrf := generateCSRFToken()
	if csrf != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    base64.RawURLEncoding.EncodeToString(csrf),
			Path:     "/",
			HttpOnly: false,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(sc.ttl.Seconds()),
		})
	}
	return nil
}

func (sc *SessionCookie) Get(r *http.Request) (*db.User, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, nil
	}

	token, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, nil
	}

	user, err := sc.authDB.ValidateSessionToken(token)
	if err != nil {
		return nil, nil
	}
	return user, nil
}

func (sc *SessionCookie) Clear(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		token, decErr := base64.RawURLEncoding.DecodeString(cookie.Value)
		if decErr == nil {
			_ = sc.authDB.DeleteSession(token)
		}
	}

	secure := sc.IsSecure(r)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	return nil
}

func ValidateCSRFToken(r *http.Request, formValue string) bool {
	cookie, err := r.Cookie("csrf_token")
	if err != nil || cookie.Value == "" {
		return false
	}
	return cookie.Value == formValue
}

func (sc *SessionCookie) EnsureCSRF(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("csrf_token"); err == nil {
		return
	}

	csrf := generateCSRFToken()
	if csrf == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    base64.RawURLEncoding.EncodeToString(csrf),
		Path:     "/",
		HttpOnly: false,
		Secure:   sc.IsSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (sc *SessionCookie) IsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if sc.trustedProxy && r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	return false
}

func EnsureCSRFCookie(w http.ResponseWriter, r *http.Request, secure bool) {
	if _, err := r.Cookie("csrf_token"); err == nil {
		return
	}

	csrf := generateCSRFToken()
	if csrf == nil {
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

func generateCSRFToken() []byte {
	csrf := make([]byte, 16)
	if _, err := rand.Read(csrf); err != nil {
		return nil
	}
	return csrf
}
