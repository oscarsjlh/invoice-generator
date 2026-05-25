//go:build e2e

package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const e2eHelperPrefix = "/__e2e/"

type e2eSessionRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type e2eSessionResponse struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

func (a *App) registerE2EHelperRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /__e2e/session", a.e2eCreateSession)
}

func isE2EPublicPath(path string) bool {
	return strings.HasPrefix(path, e2eHelperPrefix)
}

func (a *App) e2eCreateSession(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("E2E_TEST_HELPERS") != "true" {
		http.Error(w, "e2e helpers disabled", http.StatusForbidden)
		return
	}
	if !a.authEnabled || a.authDB == nil || a.sessions == nil {
		http.Error(w, "auth helpers require auth-enabled app", http.StatusBadRequest)
		return
	}

	var req e2eSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, http.ErrBodyReadAfterClose) {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.Username == "" {
		req.Username = fmt.Sprintf("e2e-user-%d", os.Getpid())
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Username
	}

	user, err := a.authDB.GetUserByUsernameForAuth(req.Username)
	if err != nil {
		a.logger.Error("e2e lookup user", "username", req.Username, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		id, err := a.authDB.CreateUser(req.Username, req.DisplayName)
		if err != nil {
			a.logger.Error("e2e create user", "username", req.Username, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		user, err = a.authDB.GetUserByID(id)
		if err != nil || user == nil {
			a.logger.Error("e2e reload user", "user_id", id, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	if err := a.sessions.CreateSession(w, r, user.ID); err != nil {
		a.logger.Error("e2e create session", "user_id", user.ID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(e2eSessionResponse{
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	})
}
