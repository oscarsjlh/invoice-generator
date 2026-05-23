package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"invoice-app/internal/db"
)

// NewTestDB creates a temporary SQLite database with migrations applied.
// Returns the store and a cleanup function that removes the temp directory.
func NewTestDB(t *testing.T) *db.Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := db.Open(dbPath)
	require.NoError(t, err)

	// Find migrations directory
	migDir := MigrationsDir(t)
	err = store.Migrate(migDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	return store
}

// MigrationsDir returns the path to the migrations directory, searching upward from CWD.
func MigrationsDir(t *testing.T) string {
	t.Helper()
	return findDir(t, "migrations")
}

// AuthMigrationsDir returns the path to the auth-migrations directory, searching upward from CWD.
func AuthMigrationsDir(t *testing.T) string {
	t.Helper()
	return findDir(t, "auth-migrations")
}

func findDir(t *testing.T, name string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		return name
	}
	dir := cwd
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			rel, err := filepath.Rel(cwd, candidate)
			if err == nil {
				return rel
			}
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	for _, candidate := range []string{name, filepath.Join("..", name)} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return name
}

// NewTestAuthDB creates a temporary SQLite auth database with auth migrations applied.
// Returns the AuthDB and registers cleanup.
func NewTestAuthDB(t *testing.T) *db.AuthDB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "auth-test.db")

	adb, err := db.OpenAuthDB(dbPath)
	require.NoError(t, err)

	migDir := AuthMigrationsDir(t)
	err = adb.Migrate(migDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, adb.Close())
	})

	return adb
}

// SampleSettings returns a fully populated db.Settings for use in tests.
func SampleSettings() db.Settings {
	return db.Settings{
		BusinessName:       "Test Business Ltd",
		BusinessAddress:    "123 Test St\nLondon\nSW1A 1AA",
		BankName:           "Test Bank",
		AccountName:        "Test Account",
		AccountNumber:      "12345678",
		SortCode:           "12-34-56",
		PaymentTerms:       "Payment due within 30 days.",
		DefaultDueDays:     30,
		CustomerName:       "Test Customer",
		CustomerTitle:      "Dr",
		CustomerEmail:      "customer@example.com",
		CustomerAddress:    "456 Client Rd",
		CustomerPostalCode: "SW1A 1AA",
		CustomerCity:       "London",
	}
}
