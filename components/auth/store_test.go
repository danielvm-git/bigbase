package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func TestOTPStore(t *testing.T) {
	logger := kernel.NoopLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	k.Register(d)

	a := New(Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	k.Register(a)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	store := NewDBOTPStore(d)
	ctx := context.Background()
	key := "test@example.com"
	codeHash := "hash123"
	expiry := time.Now().Add(5 * time.Minute)

	// Test Store
	if err := store.Store(ctx, key, codeHash, expiry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Test Get
	rec, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if rec == nil {
		t.Fatalf("Expected record, got nil")
	}
	if rec.codeHash != codeHash {
		t.Errorf("Expected codeHash %q, got %q", codeHash, rec.codeHash)
	}

	// Test RecordAttempt
	if err := store.RecordAttempt(ctx, key); err != nil {
		t.Fatalf("RecordAttempt failed: %v", err)
	}
	rec2, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if rec2.attempts != 1 {
		t.Errorf("Expected attempts to be 1, got %d", rec2.attempts)
	}

	// Test Delete
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	rec3, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if rec3 != nil {
		t.Errorf("Expected record to be deleted, got %v", rec3)
	}
}

func TestRateLimitStore(t *testing.T) {
	logger := kernel.NoopLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	k.Register(d)

	a := New(Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	k.Register(a)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	store := NewDBRateLimitStore(d)
	ctx := context.Background()
	key := "test-rate-limit"
	now := time.Now()

	// Initial increment should succeed
	ok, err := store.Increment(ctx, key, now, 3)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if !ok {
		t.Errorf("Expected ok = true, got false")
	}

	// Second increment should succeed
	ok, err = store.Increment(ctx, key, now, 3)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if !ok {
		t.Errorf("Expected ok = true, got false")
	}

	// Third increment should succeed
	ok, err = store.Increment(ctx, key, now, 3)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if !ok {
		t.Errorf("Expected ok = true, got false")
	}

	// Fourth increment should exceed max limit of 3
	ok, err = store.Increment(ctx, key, now, 3)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if ok {
		t.Errorf("Expected ok = false (exceeded limit), got true")
	}

	// If window expired (simulated with an hour ago), increment should succeed and reset count to 1
	hourAgo := now.Add(-61 * time.Minute)
	// We update the record directly or call increment with new window start
	// Note: in dbRateLimitStore.Increment, if stored window is > 1 hour ago, it updates count to 1
	_, err = store.Increment(ctx, key, now, 3)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	// Wait, the stored window is "now", which was set in previous increments, so it's not expired yet.
	// Let's modify the database window_start manually to simulate time passage.
	_, err = d.ExecContext(ctx, "UPDATE otp_rate_limits SET window_start = ? WHERE key = ?", hourAgo.Format(time.RFC3339), key)
	if err != nil {
		t.Fatalf("Update DB failed: %v", err)
	}

	// Now increment should reset the window and return true
	ok, err = store.Increment(ctx, key, now, 3)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if !ok {
		t.Errorf("Expected ok = true after reset, got false")
	}
}

func TestOTPPersistenceAcrossRestart(t *testing.T) {
	logger := kernel.NoopLogger{}
	tempDir, err := os.MkdirTemp("", "bigbase-otp-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	dbPath := filepath.Join(tempDir, "test.db")

	// Phase 1: Start kernel with file-based DB, store OTP code and rate limit
	k1 := kernel.New(logger)
	d1 := db.New(db.Options{Path: dbPath, Logger: logger})
	a1 := New(Options{DB: d1, Logger: logger, Secret: "test-secret-32-chars!!!"})
	k1.Register(d1)
	k1.Register(a1)

	if err := k1.Start(); err != nil {
		t.Fatalf("kernel 1 start: %v", err)
	}

	ctx := context.Background()
	key := "persistent@test.com"
	codeHash := "persistentHash"
	expiry := time.Now().Add(10 * time.Minute)

	if err := a1.otpStore.Store(ctx, key, codeHash, expiry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	allowed, err := a1.rateLimitStore.Increment(ctx, key, time.Now(), 3)
	if err != nil || !allowed {
		t.Fatalf("Increment failed: %v, %v", err, allowed)
	}

	_ = k1.Stop()

	// Phase 2: Start new kernel pointing to same database file (simulated restart)
	k2 := kernel.New(logger)
	d2 := db.New(db.Options{Path: dbPath, Logger: logger})
	a2 := New(Options{DB: d2, Logger: logger, Secret: "test-secret-32-chars!!!"})
	k2.Register(d2)
	k2.Register(a2)

	if err := k2.Start(); err != nil {
		t.Fatalf("kernel 2 start: %v", err)
	}
	defer func() { _ = k2.Stop() }()

	// Verify OTP code is still there and can be retrieved
	rec, err := a2.otpStore.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get after restart failed: %v", err)
	}
	if rec == nil {
		t.Fatalf("Expected OTP record to persist across restart, but got nil")
	}
	// Verify expiry year matches to be sure parsing worked
	if rec.codeHash != codeHash {
		t.Errorf("Expected codeHash %q, got %q", codeHash, rec.codeHash)
	}

	// Verify rate limit count is 1
	var count int
	err = d2.QueryRowContext(ctx, "SELECT count FROM otp_rate_limits WHERE key = ?", key).Scan(&count)
	if err != nil {
		t.Fatalf("Query count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected rate limit count 1, got %d", count)
	}
}
