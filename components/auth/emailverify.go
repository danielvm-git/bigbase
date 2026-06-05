package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	verifyTokenLen        = 6  // hex chars in verification token
	verifyTokenExpirySecs = 86400 // 24 hours
)

// EmailSender is the interface the auth component uses to dispatch emails.
// It intentionally does not import the messaging package — dependency flows
// through this interface, keeping components decoupled.
type EmailSender interface {
	SendEmail(to, subject, body string)
}

// generateVerifyToken creates a 6-char hex verification token.
func generateVerifyToken() (string, error) {
	b := make([]byte, verifyTokenLen/2)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate verify token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// migrateEmailVerify adds the email verification columns to the users table.
func (a *Auth) migrateEmailVerify(ctx context.Context) error {
	_ = a.db.Migrate("ALTER TABLE users ADD COLUMN email_verified INTEGER NOT NULL DEFAULT 0")
	_ = a.db.Migrate("ALTER TABLE users ADD COLUMN email_verify_token TEXT DEFAULT ''")
	_ = a.db.Migrate("ALTER TABLE users ADD COLUMN email_verify_expires_at TEXT DEFAULT ''")
	return nil
}

// storeVerifyToken persists the verification token and its expiry for the given user.
func (a *Auth) storeVerifyToken(ctx context.Context, userID int64, token string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	expiresAt := time.Now().UTC().Add(verifyTokenExpirySecs * time.Second).Format(time.RFC3339)
	_, err := a.db.ExecContext(ctx,
		"UPDATE users SET email_verify_token = ?, email_verify_expires_at = ? WHERE id = ?",
		token, expiresAt, userID)
	return err
}

// sendVerificationEmail fires a verification email via the injected EmailSender.
func (a *Auth) sendVerificationEmail(email, token string) {
	if a.emailSender == nil {
		return
	}
	subject := "Verify your BigBase email"
	body := fmt.Sprintf("Your verification token is: %s\n\nUse it at: /api/auth/verify-email?token=%s", token, token)
	a.emailSender.SendEmail(email, subject, body)
}

// ExtractVerifyToken parses the verification token from an email body string.
// Used in tests to extract the token from a captured email body.
func (a *Auth) ExtractVerifyToken(body string) string {
	const prefix = "verify-email?token="
	idx := strings.Index(body, prefix)
	if idx == -1 {
		return ""
	}
	token := body[idx+len(prefix):]
	// Trim any trailing whitespace/newlines.
	token = strings.TrimSpace(token)
	return token
}

// isEmailVerified returns true if the user with the given ID has verified their email,
// OR if the auth component has no emailSender configured (verification is disabled).
func (a *Auth) isEmailVerified(ctx context.Context, userID int64) (bool, error) {
	if a.emailSender == nil {
		return true, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var verified int
	err := a.db.QueryRowContext(ctx,
		"SELECT email_verified FROM users WHERE id = ?", userID).Scan(&verified)
	if err != nil {
		return false, err
	}
	return verified == 1, nil
}

// handleVerifyEmail handles GET /api/auth/verify-email?token=...
func (a *Auth) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var userID int64
	var storedToken, expiresAt string
	err := a.db.QueryRowContext(ctx,
		"SELECT id, email_verify_token, email_verify_expires_at FROM users WHERE email_verify_token = ?",
		token).Scan(&userID, &storedToken, &expiresAt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired token"})
		return
	}

	// Check expiry.
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().After(exp) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired token"})
		return
	}

	// Mark verified and clear the token.
	_, err = a.db.ExecContext(ctx,
		"UPDATE users SET email_verified = 1, email_verify_token = '', email_verify_expires_at = '' WHERE id = ?",
		userID)
	if err != nil {
		a.logger.Error("mark email verified", "user_id", userID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "email verified"})
}
