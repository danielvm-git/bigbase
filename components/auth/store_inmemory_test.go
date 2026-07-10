package auth

import (
	"context"
	"sync"
	"time"
)

// MapOTPStore is an in-memory implementation of OTPStore for testing.
type MapOTPStore struct {
	mu      sync.Mutex
	records map[string]*otpRecord
}

func NewMapOTPStore() *MapOTPStore {
	return &MapOTPStore{
		records: make(map[string]*otpRecord),
	}
}

func (s *MapOTPStore) Store(ctx context.Context, key, codeHash string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[key] = &otpRecord{
		key:       key,
		codeHash:  codeHash,
		expiresAt: expiresAt,
		attempts:  0,
	}
	return nil
}

func (s *MapOTPStore) Get(ctx context.Context, key string) (*otpRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[key]
	if !ok {
		return nil, nil
	}
	// Return a copy to prevent race conditions on mutable data
	return &otpRecord{
		key:       rec.key,
		codeHash:  rec.codeHash,
		expiresAt: rec.expiresAt,
		attempts:  rec.attempts,
	}, nil
}

func (s *MapOTPStore) RecordAttempt(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[key]
	if ok {
		rec.attempts++
	}
	return nil
}

func (s *MapOTPStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, key)
	return nil
}

// MapRateLimitStore is an in-memory implementation of RateLimitStore for testing.
type MapRateLimitStore struct {
	mu     sync.Mutex
	limits map[string]*otpRate
}

func NewMapRateLimitStore() *MapRateLimitStore {
	return &MapRateLimitStore{
		limits: make(map[string]*otpRate),
	}
}

func (s *MapRateLimitStore) Increment(ctx context.Context, key string, windowStart time.Time, max int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rate, ok := s.limits[key]
	now := time.Now()
	if ok && now.Sub(rate.windowStart) < time.Hour {
		if rate.count >= max {
			return false, nil
		}
		rate.count++
		return true, nil
	}
	s.limits[key] = &otpRate{
		count:       1,
		windowStart: windowStart,
	}
	return true, nil
}

// MapLoginLockoutStore is an in-memory LoginLockoutStore for testing.
type MapLoginLockoutStore struct {
	mu    sync.Mutex
	state map[string]*loginLockoutRow
}

func NewMapLoginLockoutStore() *MapLoginLockoutStore {
	return &MapLoginLockoutStore{state: make(map[string]*loginLockoutRow)}
}

func (s *MapLoginLockoutStore) CheckLocked(ctx context.Context, email string) (bool, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.state[email]
	if !ok {
		return false, 0, nil
	}
	now := time.Now()
	if row.hasLocked && now.Before(row.lockedUntil) {
		return true, row.lockedUntil.Sub(now), nil
	}
	return false, 0, nil
}

func (s *MapLoginLockoutStore) RecordFailure(ctx context.Context, email string) (bool, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	row := s.state[email]
	failedCount := 1
	windowStart := now
	if row != nil {
		if row.hasLocked && now.Before(row.lockedUntil) {
			return true, row.lockedUntil.Sub(now), nil
		}
		if now.Sub(row.windowStart) < loginFailureWindow {
			failedCount = row.failedCount + 1
			windowStart = row.windowStart
		}
	}
	newRow := &loginLockoutRow{failedCount: failedCount, windowStart: windowStart}
	if failedCount >= maxFailedLoginAttempts {
		newRow.lockedUntil = now.Add(loginLockoutDuration)
		newRow.hasLocked = true
		s.state[email] = newRow
		return true, loginLockoutDuration, nil
	}
	s.state[email] = newRow
	return false, 0, nil
}

func (s *MapLoginLockoutStore) ClearFailures(ctx context.Context, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.state, email)
	return nil
}
