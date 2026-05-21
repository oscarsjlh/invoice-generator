package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type MultiStore struct {
	dir           string
	migrationsDir string
	stores        map[int64]*Store
	mu            sync.RWMutex
	legacyStore   *Store
}

// NewMultiStore creates a MultiStore that manages per-user SQLite databases.
// dir is the base directory for user DBs (e.g. "data/users").
// migrationsDir is the path to SQL migration files.
func NewMultiStore(dir, migrationsDir string) *MultiStore {
	return &MultiStore{
		dir:           dir,
		migrationsDir: migrationsDir,
		stores:        make(map[int64]*Store),
	}
}

// SetLegacyStore sets the store for non-authenticated/legacy access.
func (m *MultiStore) SetLegacyStore(s *Store) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.legacyStore = s
}

// ForUser returns a Store for the given user, creating and migrating the DB if needed.
func (m *MultiStore) ForUser(userID int64) (*Store, error) {
	m.mu.RLock()
	if s, ok := m.stores[userID]; ok {
		m.mu.RUnlock()
		return s, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if s, ok := m.stores[userID]; ok {
		return s, nil
	}

	userDir := filepath.Join(m.dir, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return nil, fmt.Errorf("create user directory: %w", err)
	}

	dbPath := filepath.Join(userDir, "invoices.db")
	store, err := Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open user database: %w", err)
	}

	if err := store.Migrate(m.migrationsDir); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("migrate user database: %w", err)
	}

	m.stores[userID] = store
	return store, nil
}

// Exists checks if a user database directory already exists.
func (m *MultiStore) Exists(userID int64) bool {
	userDir := filepath.Join(m.dir, fmt.Sprintf("%d", userID))
	_, err := os.Stat(userDir)
	return err == nil
}

// Close closes all open user stores.
func (m *MultiStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for id, store := range m.stores {
		if err := store.Close(); err != nil {
			lastErr = fmt.Errorf("close store for user %d: %w", id, err)
		}
		delete(m.stores, id)
	}
	return lastErr
}
