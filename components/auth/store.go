package auth

import (
	"context"
	"database/sql"
	"time"
)

// OTPStore defines the interface for persisting OTP codes.
type OTPStore interface {
	Store(ctx context.Context, key, codeHash string, expiresAt time.Time) error
	Get(ctx context.Context, key string) (*otpRecord, error)
	RecordAttempt(ctx context.Context, key string) error
	Delete(ctx context.Context, key string) error
}

// RateLimitStore defines the interface for tracking OTP rate limits.
type RateLimitStore interface {
	Increment(ctx context.Context, key string, windowStart time.Time, max int) (bool, error)
}

type dbOTPStore struct {
	db DBer
}

func NewDBOTPStore(db DBer) OTPStore {
	return &dbOTPStore{db: db}
}

func (s *dbOTPStore) Store(ctx context.Context, key, codeHash string, expiresAt time.Time) error {
	expiresStr := expiresAt.Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO otp_codes (key, code_hash, expires_at, attempts) VALUES (?, ?, ?, 0) "+
			"ON CONFLICT(key) DO UPDATE SET code_hash = excluded.code_hash, expires_at = excluded.expires_at, attempts = 0",
		key, codeHash, expiresStr)
	return err
}

func (s *dbOTPStore) Get(ctx context.Context, key string) (*otpRecord, error) {
	var codeHash, expiresStr string
	var attempts int
	err := s.db.QueryRowContext(ctx, "SELECT code_hash, expires_at, attempts FROM otp_codes WHERE key = ?", key).
		Scan(&codeHash, &expiresStr, &attempts)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresStr)
	if err != nil {
		return nil, err
	}
	return &otpRecord{
		key:       key,
		codeHash:  codeHash,
		expiresAt: expiresAt,
		attempts:  attempts,
	}, nil
}

func (s *dbOTPStore) RecordAttempt(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE otp_codes SET attempts = attempts + 1 WHERE key = ?", key)
	return err
}

func (s *dbOTPStore) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM otp_codes WHERE key = ?", key)
	return err
}

type dbRateLimitStore struct {
	db DBer
}

func NewDBRateLimitStore(db DBer) RateLimitStore {
	return &dbRateLimitStore{db: db}
}

func (s *dbRateLimitStore) Increment(ctx context.Context, key string, windowStart time.Time, max int) (bool, error) {
	var windowStartStr string
	err := s.db.QueryRowContext(ctx, "SELECT window_start FROM otp_rate_limits WHERE key = ?", key).
		Scan(&windowStartStr)

	if err == sql.ErrNoRows {
		windowStr := windowStart.Format(time.RFC3339)
		_, err := s.db.ExecContext(ctx, "INSERT INTO otp_rate_limits (key, count, window_start) VALUES (?, 1, ?)", key, windowStr)
		if err != nil {
			return false, err
		}
		return true, nil
	} else if err != nil {
		return false, err
	}

	storedWindow, err := time.Parse(time.RFC3339, windowStartStr)
	if err != nil {
		return false, err
	}

	now := time.Now()
	if now.Sub(storedWindow) < time.Hour {
		// Within window: atomically increment only if still under limit.
		result, err := s.db.ExecContext(ctx,
			"UPDATE otp_rate_limits SET count = count + 1 WHERE key = ? AND count < ?", key, max)
		if err != nil {
			return false, err
		}
		rows, _ := result.RowsAffected()
		return rows > 0, nil
	} else {
		windowStr := windowStart.Format(time.RFC3339)
		_, err = s.db.ExecContext(ctx, "UPDATE otp_rate_limits SET count = 1, window_start = ? WHERE key = ?", windowStr, key)
		if err != nil {
			return false, err
		}
		return true, nil
	}
}
