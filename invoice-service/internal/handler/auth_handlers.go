package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"invoice-app/internal/auth"

	"github.com/go-webauthn/webauthn/protocol"
)

const (
	noticeSignInRequired = "signin_required"
	noticeSignedOut      = "signed_out"
)

func (a *App) loginPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.sessions.GetUserFromRequest(r)
	if user != nil {
		a.redirect(w, r, "/", "")
		return
	}
	notice, kind := authNoticeFromRequest(r)
	data := LoginPageData{Notice: notice, NoticeKind: kind, Next: safeNextFromRequest(r)}
	a.renderPage(w, r, http.StatusOK, "login.html", data)
}

func (a *App) registerPage(w http.ResponseWriter, r *http.Request) {
	if !a.cfg.RegistrationEnabled {
		http.NotFound(w, r)
		return
	}
	user, _ := a.sessions.GetUserFromRequest(r)
	if user != nil {
		a.redirect(w, r, "/", "")
		return
	}
	notice, kind := authNoticeFromRequest(r)
	data := RegisterPageData{Notice: notice, NoticeKind: kind}
	a.renderPage(w, r, http.StatusOK, "register.html", data)
}

type beginRegisterRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

func (a *App) beginRegistration(w http.ResponseWriter, r *http.Request) {
	if !a.cfg.RegistrationEnabled {
		http.NotFound(w, r)
		return
	}
	logger := LoggerFromContext(r.Context())
	var req beginRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	req.Username = auth.NormalizeUsername(req.Username)
	req.DisplayName = req.Username
	if !auth.ValidateUsername(req.Username) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_username", "message": auth.UsernameValidationMessage})
		return
	}

	sid, options, err := a.webAuthn.BeginRegistration(req.Username, req.DisplayName)
	if err != nil {
		logger.Warn("begin_registration_failed", "event", "begin_registration_failed", "component", "auth", "operation", "begin_registration", "username_hash", RedactEmail(req.Username), "error", err)
		// avoid leaking details to client
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "registration failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": sid,
		"options":    options,
	})
}

func (a *App) finishRegistration(w http.ResponseWriter, r *http.Request) {
	if !a.cfg.RegistrationEnabled {
		http.NotFound(w, r)
		return
	}
	logger := LoggerFromContext(r.Context())
	body, err := io.ReadAll(r.Body)
	if closeErr := r.Body.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(body))

	userID, credential, err := a.webAuthn.FinishRegistration(req.SessionID, r)
	if err != nil {
		logger.Warn("finish_registration_failed", "event", "finish_registration_failed", "component", "auth", "operation", "finish_registration", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "registration failed"})
		return
	}

	_ = credential

	if err := a.sessions.CreateSession(w, r, userID); err != nil {
		logger.Error("create_session_after_registration_failed", "event", "create_session_after_registration_failed", "component", "auth", "operation", "create_session", "user_id", userID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type beginLoginRequest struct {
	Username string `json:"username"`
}

func (a *App) beginLogin(w http.ResponseWriter, r *http.Request) {
	logger := LoggerFromContext(r.Context())
	var req beginLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	rawUsername := req.Username
	normalizedUsername := auth.NormalizeUsername(rawUsername)
	if !auth.ValidateUsername(normalizedUsername) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_username", "message": auth.UsernameValidationMessage})
		return
	}

	var sid string
	var options any
	var userID int64
	var err error
	for _, username := range auth.LoginUsernameCandidates(rawUsername) {
		sid, options, userID, err = a.webAuthn.BeginLogin(username)
		if err == nil {
			break
		}
	}
	if err != nil {
		logger.Warn("begin_login_failed", "event", "begin_login_failed", "component", "auth", "operation", "begin_login", "username_hash", RedactEmail(req.Username), "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "login failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": sid,
		"user_id":    userID,
		"options":    options,
	})
}

func (a *App) finishLogin(w http.ResponseWriter, r *http.Request) {
	logger := LoggerFromContext(r.Context())
	body, err := io.ReadAll(r.Body)
	if closeErr := r.Body.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(body))

	userID, err := a.webAuthn.FinishLogin(req.SessionID, r)
	if err != nil {
		logger.Warn("finish_login_failed", "event", "finish_login_failed", "component", "auth", "operation", "finish_login", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "login failed"})
		return
	}

	if err := a.sessions.CreateSession(w, r, userID); err != nil {
		logger.Error("create_session_after_login_failed", "event", "create_session_after_login_failed", "component", "auth", "operation", "create_session", "user_id", userID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	logger := LoggerFromContext(r.Context())
	// Simple CSRF double-submit check: compare csrf cookie to form value
	if err := r.ParseForm(); err == nil {
		formToken := r.FormValue("csrf_token")
		cookie, err := r.Cookie("csrf_token")
		if err != nil || cookie.Value == "" || cookie.Value != formToken {
			logger.Warn("logout_csrf_mismatch", "event", "logout_csrf_mismatch", "component", "auth", "operation", "logout", "error", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
	}

	_ = a.sessions.DestroySession(w, r)
	a.redirect(w, r, "/login", noticeSignedOut)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

var _ = protocol.CredentialCreation{}

func authNoticeFromRequest(r *http.Request) (string, string) {
	switch strings.TrimSpace(r.URL.Query().Get("notice")) {
	case noticeSignInRequired:
		return "Please sign in", "info"
	case noticeSignedOut:
		return "Signed out", "success"
	default:
		return "", ""
	}
}

func loginRedirectPath(r *http.Request) string {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return "/login"
	}
	next := safeNextPath(r.URL.RequestURI())
	if next == "" {
		return "/login"
	}
	return "/login?next=" + url.QueryEscape(next)
}

func safeNextFromRequest(r *http.Request) string {
	return safeNextPath(r.URL.Query().Get("next"))
}

func safeNextPath(next string) string {
	next = strings.TrimSpace(next)
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.Contains(next, "\\") {
		return ""
	}
	return next
}
