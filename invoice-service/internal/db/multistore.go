package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultMaxStores  = 64
	defaultIdleMin    = 10
	idleCheckInterval = 5 * time.Minute
)

type storeEntry struct {
	store    *Store
	lastUsed time.Time
}

type MultiStore struct {
	dir           string
	migrationsDir string
	stores        map[int64]*storeEntry
	maxStores     int
	idleMin       time.Duration
	mu            sync.Mutex
	legacyStore   *Store
	closeCtx      chan struct{}
}

func NewMultiStore(dir, migrationsDir string) *MultiStore {
	return newMultiStore(dir, migrationsDir, defaultMaxStores)
}

func NewMultiStoreWithLimits(dir, migrationsDir string, maxStores int) *MultiStore {
	if maxStores <= 0 {
		maxStores = defaultMaxStores
	}
	return newMultiStore(dir, migrationsDir, maxStores)
}

func newMultiStore(dir, migrationsDir string, maxStores int) *MultiStore {
	m := &MultiStore{
		dir:           dir,
		migrationsDir: migrationsDir,
		stores:        make(map[int64]*storeEntry),
		maxStores:     maxStores,
		idleMin:       defaultIdleMin * time.Minute,
		closeCtx:      make(chan struct{}),
	}
	go m.idleSweepLoop()
	return m
}

func (m *MultiStore) SetLegacyStore(s *Store) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.legacyStore = s
}

func (m *MultiStore) ForUser(userID int64) (*Store, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.legacyStore != nil {
		return m.legacyStore, nil
	}

	if entry, ok := m.stores[userID]; ok {
		entry.lastUsed = time.Now()
		return entry.store, nil
	}

	m.evictIfNeeded()

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

	m.stores[userID] = &storeEntry{store: store, lastUsed: time.Now()}
	return store, nil
}

func (m *MultiStore) evictIfNeeded() {
	if len(m.stores) < m.maxStores {
		return
	}

	var oldestID int64
	var oldestTime time.Time
	first := true
	for id, entry := range m.stores {
		if first || entry.lastUsed.Before(oldestTime) {
			oldestTime = entry.lastUsed
			oldestID = id
			first = false
		}
	}

	if !first {
		if entry, ok := m.stores[oldestID]; ok {
			_ = entry.store.Close()
			delete(m.stores, oldestID)
		}
	}
}

func (m *MultiStore) idleSweepLoop() {
	ticker := time.NewTicker(idleCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.sweepIdle()
		case <-m.closeCtx:
			return
		}
	}
}

func (m *MultiStore) sweepIdle() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-m.idleMin)
	for id, entry := range m.stores {
		if entry.lastUsed.Before(cutoff) {
			_ = entry.store.Close()
			delete(m.stores, id)
		}
	}
}

func (m *MultiStore) Exists(userID int64) bool {
	userDir := filepath.Join(m.dir, fmt.Sprintf("%d", userID))
	_, err := os.Stat(userDir)
	return err == nil
}

func (m *MultiStore) Close() error {
	close(m.closeCtx)

	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for id, entry := range m.stores {
		if err := entry.store.Close(); err != nil {
			lastErr = fmt.Errorf("close store for user %d: %w", id, err)
		}
		delete(m.stores, id)
	}
	return lastErr
}
