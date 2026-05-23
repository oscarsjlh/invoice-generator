package db

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewMultiStore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	migDir := findMigrationsDir(t)
	ms := NewMultiStore(dir, migDir)
	if ms == nil {
		t.Fatal("expected non-nil MultiStore")
	}
	if err := ms.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestMultiStoreForUserCreatesDB(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	migDir := findMigrationsDir(t)
	ms := NewMultiStore(dir, migDir)
	defer func() {
		if err := ms.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	store, err := ms.ForUser(1)
	if err != nil {
		t.Fatalf("ForUser: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}

	dbPath := filepath.Join(dir, "1", "invoices.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("database file not created at %s", dbPath)
	}
}

func TestMultiStoreForUserReturnsCached(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	migDir := findMigrationsDir(t)
	ms := NewMultiStore(dir, migDir)
	defer func() {
		if err := ms.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	store1, err := ms.ForUser(1)
	if err != nil {
		t.Fatalf("ForUser first call: %v", err)
	}

	store2, err := ms.ForUser(1)
	if err != nil {
		t.Fatalf("ForUser second call: %v", err)
	}

	if store1 != store2 {
		t.Errorf("expected same store instance on second call")
	}
}

func TestMultiStoreExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	migDir := findMigrationsDir(t)
	ms := NewMultiStore(dir, migDir)
	defer func() {
		if err := ms.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	if ms.Exists(1) {
		t.Error("Exists should be false before ForUser")
	}

	_, err := ms.ForUser(1)
	if err != nil {
		t.Fatalf("ForUser: %v", err)
	}

	if !ms.Exists(1) {
		t.Error("Exists should be true after ForUser")
	}

	if ms.Exists(999) {
		t.Error("Exists should be false for nonexistent user")
	}
}

func TestMultiStoreSetLegacyStore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	migDir := findMigrationsDir(t)
	ms := NewMultiStore(dir, migDir)
	defer func() {
		if err := ms.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	legacyStore := setupTestDB(t)
	ms.SetLegacyStore(legacyStore)

	store, err := ms.ForUser(99)
	if err != nil {
		t.Fatalf("ForUser with legacy: %v", err)
	}
	if store != legacyStore {
		t.Error("ForUser should return legacy store when set")
	}
}

func TestMultiStoreLRUEviction(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	migDir := findMigrationsDir(t)
	ms := NewMultiStoreWithLimits(dir, migDir, 2)
	defer func() {
		if err := ms.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	_, err := ms.ForUser(1)
	if err != nil {
		t.Fatalf("ForUser 1: %v", err)
	}
	_, err = ms.ForUser(2)
	if err != nil {
		t.Fatalf("ForUser 2: %v", err)
	}
	_, err = ms.ForUser(3)
	if err != nil {
		t.Fatalf("ForUser 3: %v", err)
	}

	count := ms.cache.len()

	if count > 2 {
		t.Errorf("expected at most 2 stores cached, got %d", count)
	}
}

func TestMultiStoreIdleSweep(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	migDir := findMigrationsDir(t)
	ms := &MultiStore{
		factory: userStoreFactory{
			dir:           dir,
			migrationsDir: migDir,
		},
		cache:    newUserStoreCache(64),
		closeCtx: make(chan struct{}),
	}
	ms.cache.idleMin = 1 * time.Millisecond
	defer func() {
		if err := ms.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	_, err := ms.ForUser(1)
	if err != nil {
		t.Fatalf("ForUser: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	ms.sweepIdle()

	count := ms.cache.len()

	if count != 0 {
		t.Errorf("expected 0 stores after idle sweep, got %d", count)
	}
}

func TestMultiStoreClose(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	migDir := findMigrationsDir(t)
	ms := NewMultiStore(dir, migDir)

	_, err := ms.ForUser(1)
	if err != nil {
		t.Fatalf("ForUser: %v", err)
	}

	closeErr := ms.Close()
	if closeErr != nil {
		t.Logf("Close returned: %v", closeErr)
	}
}

func TestMultiStoreConcurrentAccess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	migDir := findMigrationsDir(t)
	ms := NewMultiStore(dir, migDir)
	defer func() {
		if err := ms.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	for i := int64(0); i < goroutines; i++ {
		wg.Add(1)
		go func(userID int64) {
			defer wg.Done()
			_, err := ms.ForUser(userID)
			errs <- err
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("ForUser failed: %v", err)
		}
	}
}
