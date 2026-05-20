package db

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	migrationsDir := findMigrationsDir(t)
	err = store.Migrate(migrationsDir)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	return store
}

func findMigrationsDir(t *testing.T) string {
	t.Helper()
	dir := "internal/db"
	for i := 0; i < 4; i++ {
		candidate := filepath.Join(dir, "..", "migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Join(dir, "..")
	}
	for _, name := range []string{"invoice-service/migrations", "migrations"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	t.Fatal("migrations directory not found")
	return ""
}

func TestEntryCRUD(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)

	err := store.CreateEntry("2024-03-15", "Consulting", 4.5, "Client meeting")
	if err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	entry, err := store.GetEntry(1)
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if entry.Date != "2024-03-15" || entry.Category != "Consulting" || entry.Hours != 4.5 {
		t.Errorf("GetEntry returned wrong values: %+v", entry)
	}
}

func TestListEntriesSorted(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)

	store.CreateEntry("2024-03-10", "Design", 2.0, "")
	store.CreateEntry("2024-03-15", "Consulting", 4.5, "")
	store.CreateEntry("2024-03-05", "Research", 1.0, "")

	entries, err := store.ListEntries()
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Date != "2024-03-15" {
		t.Errorf("first entry date = %q, want %q", entries[0].Date, "2024-03-15")
	}
	if entries[2].Date != "2024-03-05" {
		t.Errorf("last entry date = %q, want %q", entries[2].Date, "2024-03-05")
	}
}

func TestUpdateEntry(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)

	store.CreateEntry("2024-03-15", "Consulting", 4.5, "Original notes")

	err := store.UpdateEntry(1, "2024-03-16", "Design", 3.0, "Updated notes")
	if err != nil {
		t.Fatalf("UpdateEntry: %v", err)
	}

	entry, err := store.GetEntry(1)
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if entry.Date != "2024-03-16" || entry.Category != "Design" || entry.Hours != 3.0 || entry.Notes != "Updated notes" {
		t.Errorf("GetEntry after update = %+v, expected updated values", entry)
	}
	if entry.ID != 1 {
		t.Errorf("entry ID changed from 1 to %d", entry.ID)
	}
}

func TestDeleteEntry(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)

	store.CreateEntry("2024-03-15", "Consulting", 4.5, "")

	err := store.DeleteEntry(1)
	if err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}

	_, err = store.GetEntry(1)
	if err == nil {
		t.Fatal("expected error for deleted entry, got nil")
	}

	entries, err := store.ListEntries()
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after delete, got %d", len(entries))
	}
}
