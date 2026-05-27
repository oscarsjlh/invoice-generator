package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"invoice-app/internal/auth"

	"invoice-app/internal/db"

	"github.com/go-webauthn/webauthn/protocol"
)

type LoginPageData struct {
	Notice     string
	NoticeKind string
	Next       string
	User       *db.User
}

type RegisterPageData struct {
	Notice     string
	NoticeKind string
	User       *db.User
}

const (
	noticeSignInRequired = "signin_required"
	noticeSignedOut      = "signed_out"
)

type AuthHandlers struct {
	webAuthn           *auth.WebAuthnManager
	sessionCookie      *auth.SessionCookie
	renderer           *Renderer
	authLimiter        *AuthRateLimiter
	authEnabled        bool
	registrationEnabled bool
}

func NewAuthHandlers(
	webAuthn *auth.WebAuthnManager,
	sessionCookie *auth.SessionCookie,
	renderer *Renderer,
	authLimiter *AuthRateLimiter,
	authEnabled bool,
	registrationEnabled bool,
) *AuthHandlers {
	return &AuthHandlers{
		webAuthn:            webAuthn,
		sessionCookie:       sessionCookie,
		renderer:            renderer,
		authLimiter:         authLimiter,
		authEnabled:         authEnabled,
		registrationEnabled: registrationEnabled,
	}
}

func (h *AuthHandlers) Routes(mux *http.ServeMux) {
	if h.authEnabled {
		mux.HandleFunc("GET /login", h.loginPage)
		mux.HandleFunc("GET /auth/username-policy", h.usernamePolicyHandler)
		mux.Handle("POST /login/begin", h.authLimiter.Middleware(authLimitLoginBegin)(http.HandlerFunc(h.beginLogin)))
		mux.Handle("POST /login/finish", h.authLimiter.Middleware(authLimitLoginFinish)(http.HandlerFunc(h.finishLogin)))
		if h.registrationEnabled {
			mux.HandleFunc("GET /register", h.registerPage)
			mux.Handle("POST /register/begin", h.authLimiter.Middleware(authLimitRegisterBegin)(http.HandlerFunc(h.beginRegistration)))
			mux.Handle("POST /register/finish", h.authLimiter.Middleware(authLimitRegisterFinish)(http.HandlerFunc(h.finishRegistration)))
		}
		mux.HandleFunc("POST /logout", h.logout)
	}
}

func (h *AuthHandlers) loginPage(w http.ResponseWriter, r *http.Request) {
	user, _ := h.sessionCookie.Get(r)
	if user != nil {
		redirect(w, r, "/", "")
		return
	}
	notice, kind := authNoticeFromRequest(r)
	data := LoginPageData{Notice: notice, NoticeKind: kind, Next: safeNextFromRequest(r)}
	h.renderer.Page(w, r, http.StatusOK, "login.html", data)
}

func (h *AuthHandlers) registerPage(w http.ResponseWriter, r *http.Request) {
	if !h.registrationEnabled {
		http.NotFound(w, r)
		return
	}
	user, _ := h.sessionCookie.Get(r)
	if user != nil {
		redirect(w, r, "/", "")
		return
	}
	notice, kind := authNoticeFromRequest(r)
	data := RegisterPageData{Notice: notice, NoticeKind: kind}
	h.renderer.Page(w, r, http.StatusOK, "register.html", data)
}

type beginRegisterRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

func (h *AuthHandlers) beginRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.registrationEnabled {
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

	sid, options, err := h.webAuthn.BeginRegistration(req.Username, req.DisplayName)
	if err != nil {
		logger.Warn("begin_registration_failed", "event", "begin_registration_failed", "component", "auth", "operation", "begin_registration", "username_hash", RedactEmail(req.Username), "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "registration failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": sid,
		"options":    options,
	})
}

func (h *AuthHandlers) finishRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.registrationEnabled {
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

	userID, credential, err := h.webAuthn.FinishRegistration(req.SessionID, r)
	if err != nil {
		logger.Warn("finish_registration_failed", "event", "finish_registration_failed", "component", "auth", "operation", "finish_registration", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "registration failed"})
		return
	}

	_ = credential

	if err := h.sessionCookie.Set(w, r, userID); err != nil {
		logger.Error("create_session_after_registration_failed", "event", "create_session_after_registration_failed", "component", "auth", "operation", "create_session", "user_id", userID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type beginLoginRequest struct {
	Username string `json:"username"`
}

func (h *AuthHandlers) beginLogin(w http.ResponseWriter, r *http.Request) {
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
		sid, options, userID, err = h.webAuthn.BeginLogin(username)
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

func (h *AuthHandlers) finishLogin(w http.ResponseWriter, r *http.Request) {
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

	userID, err := h.webAuthn.FinishLogin(req.SessionID, r)
	if err != nil {
		logger.Warn("finish_login_failed", "event", "finish_login_failed", "component", "auth", "operation", "finish_login", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "login failed"})
		return
	}

	if err := h.sessionCookie.Set(w, r, userID); err != nil {
		logger.Error("create_session_after_login_failed", "event", "create_session_after_login_failed", "component", "auth", "operation", "create_session", "user_id", userID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandlers) logout(w http.ResponseWriter, r *http.Request) {
	_ = h.sessionCookie.Clear(w, r)
	redirect(w, r, "/login", noticeSignedOut)
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

func (h *AuthHandlers) usernamePolicyHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, auth.DefaultUsernamePolicy)
}
