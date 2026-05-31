package db

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	_ "modernc.org/sqlite"
)

type AuthDB struct {
	db *sql.DB
}

type User struct {
	ID             int64
	Username       string
	DisplayName    string
	WebAuthnUserID string
	CreatedAt      string
	creds          []webauthn.Credential
}

func (u *User) Credentials() []webauthn.Credential {
	if u.creds == nil {
		return []webauthn.Credential{}
	}
	return u.creds
}

func (u *User) WebAuthnID() []byte {
	if u.WebAuthnUserID != "" {
		return []byte(u.WebAuthnUserID)
	}
	return []byte(fmt.Sprintf("%d", u.ID))
}

func (u *User) WebAuthnName() string {
	return u.Username
}

func (u *User) WebAuthnDisplayName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Username
}

func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials()
}

func (u *User) WebAuthnIcon() string {
	return ""
}

type CredentialRow struct {
	ID              int64
	UserID          int64
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	Transports      string
	Flags           []byte
	Authenticator   []byte
	CreatedAt       string
}

func OpenAuthDB(path string) (*AuthDB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create auth database directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open auth database: %w", err)
	}
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply pragma %q for auth database %s: %w", pragma, path, err)
		}
	}
	return &AuthDB{db: db}, nil
}

func (a *AuthDB) Close() error {
	return a.db.Close()
}

func (a *AuthDB) Migrate(dir string) error {
	if _, err := a.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (filename TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		filename := filepath.Base(file)

		var count int
		err := a.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE filename = ?`, filename).Scan(&count)
		if err != nil || count > 0 {
			continue
		}

		contents, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		statements := splitStatements(string(contents))

		tx, err := a.db.Begin()
		if err != nil {
			return fmt.Errorf("begin transaction for migration %s: %w", filename, err)
		}

		txErr := func() error {
			for i, stmt := range statements {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				if _, err := tx.Exec(stmt); err != nil {
					if isDuplicateColumnError(err) && strings.HasPrefix(strings.ToUpper(stmt), "ALTER TABLE") {
						continue
					}
					return fmt.Errorf("run migration %s statement %d: %w", filename, i+1, err)
				}
			}
			return nil
		}()

		if txErr != nil {
			_ = tx.Rollback()
			return txErr
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", filename, err)
		}

		if _, err := a.db.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, filename); err != nil {
			return fmt.Errorf("record migration %s: %w", filename, err)
		}
	}
	return nil
}

func (a *AuthDB) CreateUser(username, displayName string) (int64, error) {
	webAuthnUserID, err := generateWebAuthnUserID()
	if err != nil {
		return 0, err
	}
	result, err := a.db.Exec(`INSERT INTO users (username, display_name, webauthn_user_id) VALUES (?, ?, ?)`, username, displayName, webAuthnUserID)
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	return result.LastInsertId()
}

func (a *AuthDB) CreateUserWithWebAuthnID(username, displayName, webAuthnUserID string) (int64, error) {
	if webAuthnUserID == "" {
		return 0, fmt.Errorf("webauthn user id is required")
	}
	result, err := a.db.Exec(`INSERT INTO users (username, display_name, webauthn_user_id) VALUES (?, ?, ?)`, username, displayName, webAuthnUserID)
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	return result.LastInsertId()
}

func generateWebAuthnUserID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate webauthn user id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (a *AuthDB) GetUserByID(id int64) (*User, error) {
	row := a.db.QueryRow(`SELECT id, username, display_name, COALESCE(webauthn_user_id, CAST(id AS TEXT)), created_at FROM users WHERE id = ?`, id)
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.WebAuthnUserID, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	creds, err := a.GetCredentials(id)
	if err != nil {
		return nil, err
	}
	u.creds = creds
	return &u, nil
}

func (a *AuthDB) GetUserByUsernameForAuth(username string) (*User, error) {
	row := a.db.QueryRow(`SELECT id, username, display_name, COALESCE(webauthn_user_id, CAST(id AS TEXT)), created_at FROM users WHERE username = ?`, username)
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.WebAuthnUserID, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	creds, err := a.GetCredentials(u.ID)
	if err != nil {
		return nil, err
	}
	u.creds = creds
	return &u, nil
}

func (a *AuthDB) GetUserByUsernameFold(username string) (*User, error) {
	row := a.db.QueryRow(`SELECT id, username, display_name, COALESCE(webauthn_user_id, CAST(id AS TEXT)), created_at FROM users WHERE lower(username) = lower(?) ORDER BY id LIMIT 1`, username)
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.WebAuthnUserID, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username fold: %w", err)
	}

	creds, err := a.GetCredentials(u.ID)
	if err != nil {
		return nil, err
	}
	u.creds = creds
	return &u, nil
}

func (a *AuthDB) SaveCredential(c *webauthn.Credential, userID int64) error {
	transports, err := json.Marshal(c.Transport)
	if err != nil {
		return fmt.Errorf("marshal transports: %w", err)
	}
	flags, err := json.Marshal(c.Flags)
	if err != nil {
		return fmt.Errorf("marshal flags: %w", err)
	}
	authData, err := json.Marshal(c.Authenticator)
	if err != nil {
		return fmt.Errorf("marshal authenticator: %w", err)
	}

	_, err = a.db.Exec(`INSERT INTO webauthn_credentials (user_id, credential_id, public_key, attestation_type, transports, flags, authenticator) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, c.ID, c.PublicKey, c.AttestationType, string(transports), flags, authData)
	if err != nil {
		return fmt.Errorf("save credential: %w", err)
	}
	return nil
}

func (a *AuthDB) GetCredentials(userID int64) ([]webauthn.Credential, error) {
	rows, err := a.db.Query(`SELECT credential_id, public_key, attestation_type, transports, flags, authenticator FROM webauthn_credentials WHERE user_id = ? ORDER BY id`, userID)
	if err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var creds []webauthn.Credential
	for rows.Next() {
		var c webauthn.Credential
		var transportsRaw, flagsRaw, authRaw string
		err := rows.Scan(&c.ID, &c.PublicKey, &c.AttestationType, &transportsRaw, &flagsRaw, &authRaw)
		if err != nil {
			return nil, fmt.Errorf("scan credential: %w", err)
		}
		if err := json.Unmarshal([]byte(transportsRaw), &c.Transport); err != nil {
			return nil, fmt.Errorf("unmarshal transports: %w", err)
		}
		if err := json.Unmarshal([]byte(flagsRaw), &c.Flags); err != nil {
			return nil, fmt.Errorf("unmarshal flags: %w", err)
		}
		if err := json.Unmarshal([]byte(authRaw), &c.Authenticator); err != nil {
			return nil, fmt.Errorf("unmarshal authenticator: %w", err)
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (a *AuthDB) CreateSession(userID int64, duration time.Duration) ([]byte, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	hash := sha256.Sum256(token)
	expiresAt := time.Now().UTC().Add(duration).Format(time.RFC3339)

	_, err := a.db.Exec(`INSERT INTO sessions (user_id, token_hash, expires_at) VALUES (?, ?, ?)`, userID, hash[:], expiresAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

func (a *AuthDB) ValidateSessionToken(token []byte) (*User, error) {
	hash := sha256.Sum256(token)

	var user User
	var expiresAtStr string
	err := a.db.QueryRow(`SELECT u.id, u.username, u.display_name, COALESCE(u.webauthn_user_id, CAST(u.id AS TEXT)), u.created_at, s.expires_at FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash = ?`, hash[:]).Scan(&user.ID, &user.Username, &user.DisplayName, &user.WebAuthnUserID, &user.CreatedAt, &expiresAtStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("validate session: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse expires_at: %w", err)
	}

	if time.Now().UTC().After(expiresAt) {
		_, _ = a.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hash[:])
		return nil, nil
	}

	creds, err := a.GetCredentials(user.ID)
	if err != nil {
		return nil, err
	}
	user.creds = creds
	return &user, nil
}

func (a *AuthDB) DeleteSession(token []byte) error {
	hash := sha256.Sum256(token)
	_, err := a.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hash[:])
	return err
}

// DeleteUser removes a user and cascades to related rows (credentials, sessions)
func (a *AuthDB) DeleteUser(userID int64) error {
	_, err := a.db.Exec(`DELETE FROM users WHERE id = ?`, userID)
	return err
}

func (a *AuthDB) CleanupExpiredSessions() error {
	_, err := a.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339))
	return err
}
