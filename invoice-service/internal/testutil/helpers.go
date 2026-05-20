package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"invoice-app/internal/db"
)

// SampleEntry returns a valid db.Entry for tests.
func SampleEntry() map[string]any {
	return map[string]any{
		"id":       1,
		"date":     "2024-03-15",
		"category": "Consulting",
		"hours":    4.5,
		"notes":    "Client meeting",
	}
}

// SampleRate returns a valid db.Rate for tests.
func SampleRate() map[string]any {
	return map[string]any{
		"id":         1,
		"category":   "Consulting",
		"start_date": "2024-01-01",
		"end_date":   "",
		"rate":       150.00,
	}
}

// AssertHTMLContains checks that the response body contains the expected substring.
func AssertHTMLContains(t *testing.T, body []byte, substr string) {
	t.Helper()
	require.Contains(t, string(body), substr, "expected response body to contain %q", substr)
}

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
		store.Close()
	})

	return store
}

// MigrationsDir returns the path to the migrations directory, searching upward from CWD.
func MigrationsDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		return "migrations"
	}
	dir := cwd
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "migrations")
		if _, err := os.Stat(candidate); err == nil {
			// Return path relative to CWD for consistency
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
	// Fallback: check common relative paths from CWD
	for _, name := range []string{"migrations", "../migrations"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return "migrations"
}
