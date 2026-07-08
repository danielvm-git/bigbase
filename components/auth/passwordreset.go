package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	resetTokenLen    = 32              // hex chars in password reset token
	resetTokenExpiry = time.Hour       // 1-hour token TTL
)

// migratePasswordReset creates the password_reset_tokens table.
func (a *Auth) migratePasswordReset(ctx context.Context) error {
	return a.db.Migrate(`CREATE TABLE IF NOT EXISTS password_reset_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL UNIQUE,
		expires_at TEXT NOT NULL,
		used INTEGER NOT NULL DEFAULT 0
	)`)
}

// generateResetToken creates a cryptographically random 32-hex-char token.
func generateResetToken() (string, error) {
	b := make([]byte, resetTokenLen/2)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// storeResetToken inserts a new password reset token for the given user.
func (a *Auth) storeResetToken(ctx context.Context, userID int64, token string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	expiresAt := time.Now().UTC().Add(resetTokenExpiry).Format(time.RFC3339)
	_, err := a.db.ExecContext(ctx,
		"INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, token, expiresAt)
	return err
}

// sendResetEmail dispatches the password reset email via the injected EmailSender.
func (a *Auth) sendResetEmail(email, token string) {
	if a.emailSender == nil {
		return
	}
	subject := "Reset your BigBase password"
	body := fmt.Sprintf("Click the link to reset your password:\n/api/auth/reset-password?token=%s\n\nOr use the token: %s", token, token)
	a.emailSender.SendEmail(email, subject, body)
}

// ExtractResetToken parses the reset token from an email body.
// Used in tests to extract the token from a captured email body.
func (a *Auth) ExtractResetToken(body string) string {
	const prefix = "reset-password?token="
	idx := strings.Index(body, prefix)
	if idx == -1 {
		return ""
	}
	token := body[idx+len(prefix):]
	// Trim trailing newline or space.
	if nl := strings.IndexByte(token, '\n'); nl >= 0 {
		token = token[:nl]
	}
	return strings.TrimSpace(token)
}

// handleForgotPassword handles POST /api/auth/forgot-password.
func (a *Auth) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}

	email := strings.ToLower(req.Email)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Look up the user — return 200 even if not found (prevents user enumeration).
	var userID int64
	err := a.db.QueryRowContext(ctx,
		"SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err != nil {
		// Unknown email — respond OK silently.
		writeJSON(w, http.StatusOK, map[string]string{"status": "if that email exists, a reset link has been sent"})
		return
	}

	token, err := generateResetToken()
	if err != nil {
		a.logger.Error("generate reset token", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if err := a.storeResetToken(ctx, userID, token); err != nil {
		a.logger.Error("store reset token", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.sendResetEmail(email, token)

	a.recordAudit("auth.password_reset_requested", userID, email, getIP(r), nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "if that email exists, a reset link has been sent"})
}

// handleResetPassword handles POST /api/auth/reset-password.
func (a *Auth) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token required"})
		return
	}
	if len(req.NewPassword) < minPasswordLen {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Fetch the token record.
	var tokenID, userID int64
	var expiresAt string
	var used int
	err := a.db.QueryRowContext(ctx,
		"SELECT id, user_id, expires_at, used FROM password_reset_tokens WHERE token = ?",
		req.Token).Scan(&tokenID, &userID, &expiresAt, &used)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired token"})
		return
	}

	// Check used flag.
	if used != 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired token"})
		return
	}

	// Check expiry.
	exp, parseErr := time.Parse(time.RFC3339, expiresAt)
	if parseErr != nil || time.Now().After(exp) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired token"})
		return
	}

	// Hash the new password.
	hash, err := hashPassword(req.NewPassword)
	if err != nil {
		a.logger.Error("hash new password", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Update the password.
	_, err = a.db.ExecContext(ctx,
		"UPDATE users SET password_hash = ? WHERE id = ?", hash, userID)
	if err != nil {
		a.logger.Error("update password", "user_id", userID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Invalidate the token.
	_, err = a.db.ExecContext(ctx,
		"UPDATE password_reset_tokens SET used = 1 WHERE id = ?", tokenID)
	if err != nil {
		a.logger.Error("invalidate reset token", "token_id", tokenID, "error", err)
		// Non-fatal: password already updated.
	}

	var email string
	_ = a.db.QueryRowContext(ctx, "SELECT email FROM users WHERE id = ?", userID).Scan(&email)

	a.recordAudit("auth.password_reset_completed", userID, email, getIP(r), nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "password updated"})
}

