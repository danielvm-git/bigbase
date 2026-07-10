package auth

import (
	"context"
	"database/sql"
	"strconv"
	"time"
)

const (
	maxFailedLoginAttempts = 5
	loginLockoutDuration   = 15 * time.Minute
	loginFailureWindow     = 15 * time.Minute
)

const loginLockoutsMigration = `CREATE TABLE IF NOT EXISTS login_lockouts (
	email TEXT PRIMARY KEY,
	failed_count INTEGER NOT NULL DEFAULT 0,
	locked_until TEXT,
	window_start TEXT NOT NULL
)`

// LoginLockoutStore tracks per-email failed login attempts and lockout state.
type LoginLockoutStore interface {
	CheckLocked(ctx context.Context, email string) (locked bool, retryAfter time.Duration, err error)
	RecordFailure(ctx context.Context, email string) (locked bool, retryAfter time.Duration, err error)
	ClearFailures(ctx context.Context, email string) error
}

type dbLoginLockoutStore struct {
	db DBer
}

func NewDBLoginLockoutStore(db DBer) LoginLockoutStore {
	return &dbLoginLockoutStore{db: db}
}

type loginLockoutRow struct {
	failedCount int
	lockedUntil time.Time
	windowStart time.Time
	hasLocked   bool
}

func (s *dbLoginLockoutStore) load(ctx context.Context, email string) (*loginLockoutRow, error) {
	var failedCount int
	var lockedUntilStr, windowStartStr sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT failed_count, locked_until, window_start FROM login_lockouts WHERE email = ?", email).
		Scan(&failedCount, &lockedUntilStr, &windowStartStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row := &loginLockoutRow{failedCount: failedCount}
	if windowStartStr.Valid {
		row.windowStart, err = time.Parse(time.RFC3339, windowStartStr.String)
		if err != nil {
			return nil, err
		}
	}
	if lockedUntilStr.Valid && lockedUntilStr.String != "" {
		row.lockedUntil, err = time.Parse(time.RFC3339, lockedUntilStr.String)
		if err != nil {
			return nil, err
		}
		row.hasLocked = true
	}
	return row, nil
}

func (s *dbLoginLockoutStore) CheckLocked(ctx context.Context, email string) (bool, time.Duration, error) {
	row, err := s.load(ctx, email)
	if err != nil || row == nil {
		return false, 0, err
	}
	now := time.Now()
	if row.hasLocked && now.Before(row.lockedUntil) {
		return true, row.lockedUntil.Sub(now), nil
	}
	return false, 0, nil
}

func (s *dbLoginLockoutStore) RecordFailure(ctx context.Context, email string) (bool, time.Duration, error) {
	now := time.Now()
	row, err := s.load(ctx, email)
	if err != nil {
		return false, 0, err
	}

	failedCount := 1
	windowStart := now
	var lockedUntil time.Time
	applyLock := false

	if row != nil {
		if row.hasLocked && now.Before(row.lockedUntil) {
			return true, row.lockedUntil.Sub(now), nil
		}
		if now.Sub(row.windowStart) < loginFailureWindow {
			failedCount = row.failedCount + 1
			windowStart = row.windowStart
		}
	}

	if failedCount >= maxFailedLoginAttempts {
		lockedUntil = now.Add(loginLockoutDuration)
		applyLock = true
	}

	windowStr := windowStart.Format(time.RFC3339)
	if row == nil {
		var lockedStr any
		if applyLock {
			lockedStr = lockedUntil.Format(time.RFC3339)
		}
		_, err = s.db.ExecContext(ctx,
			"INSERT INTO login_lockouts (email, failed_count, locked_until, window_start) VALUES (?, ?, ?, ?)",
			email, failedCount, lockedStr, windowStr)
	} else {
		if applyLock {
			_, err = s.db.ExecContext(ctx,
				"UPDATE login_lockouts SET failed_count = ?, locked_until = ?, window_start = ? WHERE email = ?",
				failedCount, lockedUntil.Format(time.RFC3339), windowStr, email)
		} else {
			_, err = s.db.ExecContext(ctx,
				"UPDATE login_lockouts SET failed_count = ?, locked_until = NULL, window_start = ? WHERE email = ?",
				failedCount, windowStr, email)
		}
	}
	if err != nil {
		return false, 0, err
	}
	if applyLock {
		return true, loginLockoutDuration, nil
	}
	return false, 0, nil
}

func (s *dbLoginLockoutStore) ClearFailures(ctx context.Context, email string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM login_lockouts WHERE email = ?", email)
	return err
}

func retryAfterHeader(d time.Duration) string {
	secs := int(d.Seconds())
	if secs < 1 {
		secs = 1
	}
	return strconv.Itoa(secs)
}
