package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

func setupTestAuthDB(t *testing.T) *AuthDB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "auth-test.db")

	adb, err := OpenAuthDB(dbPath)
	if err != nil {
		t.Fatalf("OpenAuthDB: %v", err)
	}

	migDir := findAuthMigrationsDir(t)
	err = adb.Migrate(migDir)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	t.Cleanup(func() {
		adb.Close()
	})

	return adb
}

func findAuthMigrationsDir(t *testing.T) string {
	t.Helper()
	dir := "internal/db"
	for i := 0; i < 4; i++ {
		candidate := filepath.Join(dir, "..", "auth-migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Join(dir, "..")
	}
	for _, name := range []string{"invoice-service/auth-migrations", "auth-migrations"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	t.Fatal("auth-migrations directory not found")
	return ""
}

func TestAuthDBCreateUser(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	id1, err := adb.CreateUser("alice", "Alice Wonder")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if id1 <= 0 {
		t.Errorf("expected positive user ID, got %d", id1)
	}
}

func TestAuthDBGetUserByID(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	id, _ := adb.CreateUser("bob", "Bob Builder")
	user, err := adb.GetUserByID(id)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Username != "bob" {
		t.Errorf("Username = %q, want bob", user.Username)
	}
	if user.DisplayName != "Bob Builder" {
		t.Errorf("DisplayName = %q, want Bob Builder", user.DisplayName)
	}
}

func TestAuthDBGetUserByIDNotFound(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	user, err := adb.GetUserByID(999)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil for nonexistent user, got %+v", user)
	}
}

func TestAuthDBGetUserByUsername(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	adb.CreateUser("charlie", "Charlie Chaplin")
	user, err := adb.GetUserByUsernameForAuth("charlie")
	if err != nil {
		t.Fatalf("GetUserByUsernameForAuth: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Username != "charlie" {
		t.Errorf("Username = %q, want charlie", user.Username)
	}
}

func TestAuthDBGetUserByUsernameNotFound(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	user, err := adb.GetUserByUsernameForAuth("nobody")
	if err != nil {
		t.Fatalf("GetUserByUsernameForAuth: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil for nonexistent username, got %+v", user)
	}
}

func TestAuthDBSaveAndGetCredentials(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	id, _ := adb.CreateUser("dave", "Dave Developer")
	cred := &webauthn.Credential{
		ID:              []byte("credential-id-123"),
		PublicKey:       []byte("public-key-data"),
		AttestationType: "none",
		Transport:       []protocol.AuthenticatorTransport{protocol.USB},
	}
	err := adb.SaveCredential(cred, id)
	if err != nil {
		t.Fatalf("SaveCredential: %v", err)
	}

	creds, err := adb.GetCredentials(id)
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if string(creds[0].ID) != "credential-id-123" {
		t.Errorf("credential ID = %s, want credential-id-123", creds[0].ID)
	}
}

func TestAuthDBGetCredentialsEmpty(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	id, _ := adb.CreateUser("eve", "Eve Empty")
	creds, err := adb.GetCredentials(id)
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected empty credentials, got %d", len(creds))
	}
}

func TestAuthDBCreateAndValidateSession(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	id, _ := adb.CreateUser("frank", "Frank Session")
	token, err := adb.CreateSession(id, 1*time.Hour)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if len(token) != 32 {
		t.Errorf("token length = %d, want 32", len(token))
	}

	user, err := adb.ValidateSessionToken(token)
	if err != nil {
		t.Fatalf("ValidateSessionToken: %v", err)
	}
	if user == nil {
		t.Fatal("expected user from valid session, got nil")
	}
	if user.Username != "frank" {
		t.Errorf("Username = %q, want frank", user.Username)
	}
}

func TestAuthDBValidateInvalidToken(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	token := make([]byte, 32)
	for i := range token {
		token[i] = 0xFF
	}
	user, err := adb.ValidateSessionToken(token)
	if err != nil {
		t.Fatalf("ValidateSessionToken: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil for invalid token, got user %+v", user)
	}
}

func TestAuthDBValidateExpiredSession(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	id, _ := adb.CreateUser("grace", "Grace Expired")
	token, err := adb.CreateSession(id, -1*time.Hour)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	user, err := adb.ValidateSessionToken(token)
	if err != nil {
		t.Fatalf("ValidateSessionToken: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil for expired session, got user %+v", user)
	}
}

func TestAuthDBDeleteSession(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	id, _ := adb.CreateUser("heidi", "Heidi Delete")
	token, _ := adb.CreateSession(id, 1*time.Hour)

	err := adb.DeleteSession(token)
	if err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	user, err := adb.ValidateSessionToken(token)
	if err != nil {
		t.Fatalf("ValidateSessionToken after delete: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil after delete, got user %+v", user)
	}
}

func TestAuthDBDeleteUser(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	id, _ := adb.CreateUser("ivan", "Ivan Delete")
	err := adb.DeleteUser(id)
	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	user, err := adb.GetUserByID(id)
	if err != nil {
		t.Fatalf("GetUserByID after delete: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil after delete, got user %+v", user)
	}
}

func TestAuthDBCleanupExpiredSessions(t *testing.T) {
	t.Parallel()
	adb := setupTestAuthDB(t)

	id, _ := adb.CreateUser("jack", "Jack Cleanup")
	_, _ = adb.CreateSession(id, -1*time.Hour)
	_, _ = adb.CreateSession(id, 1*time.Hour)

	err := adb.CleanupExpiredSessions()
	if err != nil {
		t.Fatalf("CleanupExpiredSessions: %v", err)
	}
}
