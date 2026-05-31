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
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

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
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-10", "Design", 2.0, ""); err != nil {
		t.Fatalf("CreateEntry 1: %v", err)
	}
	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry 2: %v", err)
	}
	if err := store.CreateEntry("2024-03-05", "Research", 1.0, ""); err != nil {
		t.Fatalf("CreateEntry 3: %v", err)
	}

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

func TestListEntriesFilteredByYearAndMonth(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-10", "Design", 2.0, ""); err != nil {
		t.Fatalf("CreateEntry 1: %v", err)
	}
	if err := store.CreateEntry("2024-04-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry 2: %v", err)
	}
	if err := store.CreateEntry("2023-03-05", "Research", 1.0, ""); err != nil {
		t.Fatalf("CreateEntry 3: %v", err)
	}

	entries, err := store.ListEntriesFiltered("2024", "03", "")
	if err != nil {
		t.Fatalf("ListEntriesFiltered: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Date != "2024-03-10" || entries[0].Category != "Design" {
		t.Errorf("filtered entry = %+v, want 2024-03-10 Design", entries[0])
	}

	yearEntries, err := store.ListEntriesFiltered("2024", "", "")
	if err != nil {
		t.Fatalf("ListEntriesFiltered year: %v", err)
	}
	if len(yearEntries) != 2 {
		t.Fatalf("expected 2 entries for 2024, got %d", len(yearEntries))
	}
}

func TestListEntriesFilteredByRate(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-10", "Design", 2.0, ""); err != nil {
		t.Fatalf("CreateEntry 1: %v", err)
	}
	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry 2: %v", err)
	}
	if err := store.CreateEntry("2024-04-15", "Consulting", 1.5, ""); err != nil {
		t.Fatalf("CreateEntry 3: %v", err)
	}

	entries, err := store.ListEntriesFiltered("2024", "03", "Consulting")
	if err != nil {
		t.Fatalf("ListEntriesFiltered: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Date != "2024-03-15" || entries[0].Category != "Consulting" {
		t.Errorf("filtered entry = %+v, want 2024-03-15 Consulting", entries[0])
	}
}

func TestListEntriesMarksMissingRates(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry missing rate: %v", err)
	}
	if err := store.CreateEntry("2024-04-15", "Consulting", 2, ""); err != nil {
		t.Fatalf("CreateEntry rated: %v", err)
	}
	if err := store.CreateRate("Consulting", "2024-04-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	entries, err := store.ListEntries()
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].MissingRate {
		t.Errorf("newer rated entry MissingRate = true, want false")
	}
	if !entries[1].MissingRate {
		t.Errorf("older unrated entry MissingRate = false, want true")
	}
}

func TestCountUnratedEntriesFiltered(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry 1: %v", err)
	}
	if err := store.CreateEntry("2024-04-15", "Consulting", 2, ""); err != nil {
		t.Fatalf("CreateEntry 2: %v", err)
	}
	if err := store.CreateRate("Consulting", "2024-04-01", "", 150.00); err != nil {
		t.Fatalf("CreateRate: %v", err)
	}

	count, err := store.CountUnratedEntriesFiltered("2024", "03")
	if err != nil {
		t.Fatalf("CountUnratedEntriesFiltered: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	count, err = store.CountUnratedEntriesFiltered("2024", "04")
	if err != nil {
		t.Fatalf("CountUnratedEntriesFiltered: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestUpdateEntry(t *testing.T) {
	t.Parallel()
	store := setupTestDB(t)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, "Original notes"); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

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
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := store.CreateEntry("2024-03-15", "Consulting", 4.5, ""); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

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
