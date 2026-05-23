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

type userStoreFactory struct {
	dir           string
	migrationsDir string
}

func (f userStoreFactory) open(userID int64) (*Store, error) {
	userDir := filepath.Join(f.dir, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return nil, fmt.Errorf("create user directory: %w", err)
	}

	dbPath := filepath.Join(userDir, "invoices.db")
	store, err := Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open user database: %w", err)
	}

	if err := store.Migrate(f.migrationsDir); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("migrate user database: %w", err)
	}
	return store, nil
}

type userStoreCache struct {
	stores    map[int64]*storeEntry
	maxStores int
	idleMin   time.Duration
	mu        sync.Mutex
}

func newUserStoreCache(maxStores int) *userStoreCache {
	return &userStoreCache{
		stores:    make(map[int64]*storeEntry),
		maxStores: maxStores,
		idleMin:   defaultIdleMin * time.Minute,
	}
}

func (c *userStoreCache) get(userID int64) (*Store, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.stores[userID]; ok {
		entry.lastUsed = time.Now()
		return entry.store, true
	}
	return nil, false
}

func (c *userStoreCache) put(userID int64, store *Store) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.evictIfNeeded()
	c.stores[userID] = &storeEntry{store: store, lastUsed: time.Now()}
}

func (c *userStoreCache) evictIfNeeded() {
	if len(c.stores) < c.maxStores {
		return
	}

	var oldestID int64
	var oldestTime time.Time
	first := true
	for id, entry := range c.stores {
		if first || entry.lastUsed.Before(oldestTime) {
			oldestTime = entry.lastUsed
			oldestID = id
			first = false
		}
	}

	if !first {
		if entry, ok := c.stores[oldestID]; ok {
			_ = entry.store.Close()
			delete(c.stores, oldestID)
		}
	}
}

func (c *userStoreCache) sweepIdle() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-c.idleMin)
	for id, entry := range c.stores {
		if entry.lastUsed.Before(cutoff) {
			_ = entry.store.Close()
			delete(c.stores, id)
		}
	}
}

func (c *userStoreCache) closeAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	for id, entry := range c.stores {
		if err := entry.store.Close(); err != nil {
			lastErr = fmt.Errorf("close store for user %d: %w", id, err)
		}
		delete(c.stores, id)
	}
	return lastErr
}

func (c *userStoreCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.stores)
}

type MultiStore struct {
	factory     userStoreFactory
	cache       *userStoreCache
	legacyStore *Store
	mu          sync.Mutex
	closeCtx    chan struct{}
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
		factory: userStoreFactory{
			dir:           dir,
			migrationsDir: migrationsDir,
		},
		cache:    newUserStoreCache(maxStores),
		closeCtx: make(chan struct{}),
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

	if store, ok := m.cache.get(userID); ok {
		return store, nil
	}

	store, err := m.factory.open(userID)
	if err != nil {
		return nil, err
	}

	m.cache.put(userID, store)
	return store, nil
}

func (m *MultiStore) idleSweepLoop() {
	ticker := time.NewTicker(idleCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.cache.sweepIdle()
		case <-m.closeCtx:
			return
		}
	}
}

func (m *MultiStore) sweepIdle() {
	m.cache.sweepIdle()
}

func (m *MultiStore) Exists(userID int64) bool {
	userDir := filepath.Join(m.factory.dir, fmt.Sprintf("%d", userID))
	_, err := os.Stat(userDir)
	return err == nil
}

func (m *MultiStore) Close() error {
	close(m.closeCtx)

	return m.cache.closeAll()
}
