package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"invoice-app/internal/db"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type WebAuthnManager struct {
	web      *webauthn.WebAuthn
	adb      *db.AuthDB
	mu       sync.Mutex
	sessions map[string]pendingSession
}

func NewWebAuthnManager(authDB *db.AuthDB, cfg AuthConfig) (*WebAuthnManager, error) {
	wconfig := &webauthn.Config{
		RPDisplayName: cfg.RPDisplayName,
		RPID:          cfg.RPID,
		RPOrigins:     cfg.RPOrigins,
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("create webauthn: %w", err)
	}

	m := &WebAuthnManager{
		web:      w,
		adb:      authDB,
		sessions: make(map[string]pendingSession),
	}

	// Start cleanup goroutine to purge stale pending sessions (and orphaned users)
	go m.cleanupLoop()

	return m, nil
}

type pendingSession struct {
	Session   webauthn.SessionData
	Kind      string // "reg" or "login"
	UserID    int64  // user id for reg/login flows (DB id created at begin)
	CreatedAt time.Time
}

func (m *WebAuthnManager) genSessionID() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// cleanupLoop runs periodically to remove expired pending sessions.
func (m *WebAuthnManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	ttl := 5 * time.Minute
	for range ticker.C {
		cutoff := time.Now().Add(-ttl)
		m.mu.Lock()
		for k, v := range m.sessions {
			if v.CreatedAt.Before(cutoff) {
				// if a pending registration left an orphaned user (no creds), delete it
				if v.Kind == "reg" && v.UserID != 0 {
					// attempt to delete user if they have no credentials
					if creds, err := m.adb.GetCredentials(v.UserID); err == nil {
						if len(creds) == 0 {
							// ignore error
							_ = m.adb.DeleteUser(v.UserID)
						}
					}
				}
				delete(m.sessions, k)
			}
		}
		m.mu.Unlock()
	}
}

func (m *WebAuthnManager) BeginRegistration(username, displayName string) (string, *protocol.CredentialCreation, error) {
	existing, err := m.adb.GetUserByUsernameForAuth(username)
	if err != nil {
		return "", nil, fmt.Errorf("check existing user: %w", err)
	}
	if existing != nil {
		return "", nil, fmt.Errorf("username already taken")
	}

	userID, err := m.adb.CreateUser(username, displayName)
	if err != nil {
		return "", nil, fmt.Errorf("create user: %w", err)
	}

	user, err := m.adb.GetUserByID(userID)
	if err != nil {
		return "", nil, fmt.Errorf("get new user: %w", err)
	}

	wuser := NewWebAuthnUser(user)

	options, session, err := m.web.BeginRegistration(wuser)
	if err != nil {
		return "", nil, fmt.Errorf("begin registration: %w", err)
	}

	sid, err := m.genSessionID()
	if err != nil {
		return "", nil, fmt.Errorf("gen session id: %w", err)
	}
	m.mu.Lock()
	m.sessions[sid] = pendingSession{Session: *session, Kind: "reg", UserID: userID, CreatedAt: time.Now()}
	m.mu.Unlock()

	return sid, options, nil
}

func (m *WebAuthnManager) FinishRegistration(sessionID string, r *http.Request) (int64, *webauthn.Credential, error) {
	m.mu.Lock()
	ps, ok := m.sessions[sessionID]
	if !ok {
		m.mu.Unlock()
		return 0, nil, fmt.Errorf("no registration session found")
	}
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	user, err := m.adb.GetUserByID(ps.UserID)
	if err != nil {
		return 0, nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return 0, nil, fmt.Errorf("user not found")
	}

	wuser := NewWebAuthnUser(user)

	credential, err := m.web.FinishRegistration(wuser, ps.Session, r)
	if err != nil {
		return 0, nil, fmt.Errorf("finish registration: %w", err)
	}

	if err := m.adb.SaveCredential(credential, ps.UserID); err != nil {
		return 0, nil, fmt.Errorf("save credential: %w", err)
	}

	return ps.UserID, credential, nil
}

func (m *WebAuthnManager) BeginLogin(username string) (string, *protocol.CredentialAssertion, int64, error) {
	user, err := m.adb.GetUserByUsernameForAuth(username)
	if err != nil {
		return "", nil, 0, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return "", nil, 0, fmt.Errorf("user not found")
	}

	wuser := NewWebAuthnUser(user)

	options, session, err := m.web.BeginLogin(wuser)
	if err != nil {
		return "", nil, 0, fmt.Errorf("begin login: %w", err)
	}

	sid, err := m.genSessionID()
	if err != nil {
		return "", nil, 0, fmt.Errorf("gen session id: %w", err)
	}
	m.mu.Lock()
	m.sessions[sid] = pendingSession{Session: *session, Kind: "login", UserID: user.ID, CreatedAt: time.Now()}
	m.mu.Unlock()

	return sid, options, user.ID, nil
}

func (m *WebAuthnManager) FinishLogin(sessionID string, r *http.Request) (int64, error) {
	m.mu.Lock()
	ps, ok := m.sessions[sessionID]
	if !ok {
		m.mu.Unlock()
		return 0, fmt.Errorf("no login session found")
	}
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	user, err := m.adb.GetUserByID(ps.UserID)
	if err != nil {
		return 0, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return 0, fmt.Errorf("user not found")
	}

	wuser := NewWebAuthnUser(user)

	_, err = m.web.FinishLogin(wuser, ps.Session, r)
	if err != nil {
		return 0, fmt.Errorf("finish login: %w", err)
	}

	return ps.UserID, nil
}
