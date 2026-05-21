package auth

import (
	"time"

	"invoice-app/internal/db"
)

type WebAuthnUser struct {
	*db.User
}

func NewWebAuthnUser(u *db.User) *WebAuthnUser {
	return &WebAuthnUser{User: u}
}

type SessionInfo struct {
	UserID    int64
	Username  string
	Token     []byte
	CreatedAt time.Time
	ExpiresAt time.Time
}

type AuthConfig struct {
	RPID          string
	RPOrigins     []string
	RPDisplayName string
	SessionTTL    time.Duration
}
