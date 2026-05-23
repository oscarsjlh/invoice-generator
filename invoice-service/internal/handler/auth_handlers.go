package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
)

func (a *App) loginPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.sessions.GetUserFromRequest(r)
	if user != nil {
		a.redirect(w, r, "/", "")
		return
	}
	data := LoginPageData{Notice: noticeFromRequest(r)}
	a.renderPage(w, r, http.StatusOK, "login.html", data)
}

func (a *App) registerPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.sessions.GetUserFromRequest(r)
	if user != nil {
		a.redirect(w, r, "/", "")
		return
	}
	data := RegisterPageData{Notice: noticeFromRequest(r)}
	a.renderPage(w, r, http.StatusOK, "register.html", data)
}

type beginRegisterRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

func (a *App) beginRegistration(w http.ResponseWriter, r *http.Request) {
	logger := LoggerFromContext(r.Context())
	var req beginRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
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
	logger := LoggerFromContext(r.Context())
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
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

	sid, options, userID, err := a.webAuthn.BeginLogin(req.Username)
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
	r.Body.Close()
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
	a.redirect(w, r, "/login", "Signed out")
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

var _ = protocol.CredentialCreation{}
