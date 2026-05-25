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
	Session        webauthn.SessionData
	Kind           string // "reg" or "login"
	UserID         int64  // user id for login flows
	Username       string
	DisplayName    string
	WebAuthnUserID string
	CreatedAt      time.Time
}

func (m *WebAuthnManager) genSessionID() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (m *WebAuthnManager) genWebAuthnUserID() (string, error) {
	b := make([]byte, 32)
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
				delete(m.sessions, k)
			}
		}
		m.mu.Unlock()
	}
}

func (m *WebAuthnManager) BeginRegistration(username, displayName string) (string, *protocol.CredentialCreation, error) {
	existing, err := m.adb.GetUserByUsernameFold(username)
	if err != nil {
		return "", nil, fmt.Errorf("check existing user: %w", err)
	}
	if existing != nil {
		return "", nil, fmt.Errorf("username already taken")
	}

	webAuthnUserID, err := m.genWebAuthnUserID()
	if err != nil {
		return "", nil, fmt.Errorf("gen webauthn user id: %w", err)
	}

	user := &db.User{Username: username, DisplayName: displayName, WebAuthnUserID: webAuthnUserID}
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
	m.sessions[sid] = pendingSession{Session: *session, Kind: "reg", Username: username, DisplayName: displayName, WebAuthnUserID: webAuthnUserID, CreatedAt: time.Now()}
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

	existing, err := m.adb.GetUserByUsernameFold(ps.Username)
	if err != nil {
		return 0, nil, fmt.Errorf("check existing user: %w", err)
	}
	if existing != nil {
		return 0, nil, fmt.Errorf("username already taken")
	}

	user := &db.User{Username: ps.Username, DisplayName: ps.DisplayName, WebAuthnUserID: ps.WebAuthnUserID}
	wuser := NewWebAuthnUser(user)

	credential, err := m.web.FinishRegistration(wuser, ps.Session, r)
	if err != nil {
		return 0, nil, fmt.Errorf("finish registration: %w", err)
	}

	userID, err := m.adb.CreateUserWithWebAuthnID(ps.Username, ps.DisplayName, ps.WebAuthnUserID)
	if err != nil {
		return 0, nil, fmt.Errorf("create user: %w", err)
	}

	if err := m.adb.SaveCredential(credential, userID); err != nil {
		return 0, nil, fmt.Errorf("save credential: %w", err)
	}

	return userID, credential, nil
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
