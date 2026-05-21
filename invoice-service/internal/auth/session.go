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

type SessionManager struct {
	authDB *db.AuthDB
	ttl    time.Duration
	secure bool
}

func NewSessionManager(authDB *db.AuthDB, ttl time.Duration, secure bool) *SessionManager {
	return &SessionManager{authDB: authDB, ttl: ttl, secure: secure}
}

func (sm *SessionManager) CreateSession(w http.ResponseWriter, userID int64) error {
	token, err := sm.authDB.CreateSession(userID, sm.ttl)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(token)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sm.ttl.Seconds()),
	})

	// set a non-HttpOnly CSRF cookie for double-submit validation on forms
	csrf := make([]byte, 16)
	if _, err := rand.Read(csrf); err == nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    base64.RawURLEncoding.EncodeToString(csrf),
			Path:     "/",
			HttpOnly: false,
			Secure:   sm.secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(sm.ttl.Seconds()),
		})
	}
	return nil
}

func (sm *SessionManager) GetUserFromRequest(r *http.Request) (*db.User, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, nil
	}

	token, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, nil
	}

	user, err := sm.authDB.ValidateSessionToken(token)
	if err != nil {
		return nil, nil
	}
	return user, nil
}

func (sm *SessionManager) DestroySession(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		token, decErr := base64.RawURLEncoding.DecodeString(cookie.Value)
		if decErr == nil {
			_ = sm.authDB.DeleteSession(token)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	// clear csrf cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	return nil
}
