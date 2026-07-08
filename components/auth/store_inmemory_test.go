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
