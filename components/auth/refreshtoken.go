package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	refreshTokenLen    = 64              // hex chars (32 bytes of entropy)
	refreshTokenExpiry = 30 * 24 * time.Hour // 30-day TTL
)

// migrateRefreshTokens creates the refresh_tokens table.
func (a *Auth) migrateRefreshTokens(ctx context.Context) error {
	return a.db.Migrate(`CREATE TABLE IF NOT EXISTS refresh_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL UNIQUE,
		family TEXT NOT NULL,
		used INTEGER NOT NULL DEFAULT 0,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`)
}

// generateRefreshToken creates a cryptographically random 64-hex-char token.
func generateRefreshToken() (string, error) {
	b := make([]byte, refreshTokenLen/2)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// generateFamilyID creates a new token family identifier.
func generateFamilyID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate family id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// issueRefreshToken inserts a new refresh token for the given user and family.
func (a *Auth) issueRefreshToken(ctx context.Context, userID int64, family string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	token, err := generateRefreshToken()
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(refreshTokenExpiry).Format(time.RFC3339)
	createdAt := now.Format(time.RFC3339)

	_, err = a.db.ExecContext(ctx,
		"INSERT INTO refresh_tokens (user_id, token, family, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		userID, token, family, expiresAt, createdAt)
	if err != nil {
		return "", fmt.Errorf("insert refresh token: %w", err)
	}
	return token, nil
}

// invalidateFamily marks all refresh tokens in the given family as used.
// Called on replay detection to protect the entire token lineage.
func (a *Auth) invalidateFamily(ctx context.Context, family string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := a.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET used = 1 WHERE family = ?", family)
	return err
}

// handleRefresh handles POST /api/auth/refresh.
func (a *Auth) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh_token required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Fetch token record.
	var tokenID, userID int64
	var family, expiresAt string
	var used int
	err := a.db.QueryRowContext(ctx,
		"SELECT id, user_id, family, expires_at, used FROM refresh_tokens WHERE token = ?",
		req.RefreshToken).Scan(&tokenID, &userID, &family, &expiresAt, &used)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
		return
	}

	// Replay detection: if already used, invalidate the entire family.
	if used != 0 {
		_ = a.invalidateFamily(ctx, family)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "refresh token already used"})
		return
	}

	// Check expiry.
	exp, parseErr := time.Parse(time.RFC3339, expiresAt)
	if parseErr != nil || time.Now().After(exp) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "refresh token expired"})
		return
	}

	// Mark old token as used (rotation — single use).
	_, err = a.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET used = 1 WHERE id = ?", tokenID)
	if err != nil {
		a.logger.Error("invalidate old refresh token", "id", tokenID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Fetch user info to mint new access token.
	var email, role string
	var defaultOrgID int64
	err = a.db.QueryRowContext(ctx,
		"SELECT email, role, COALESCE(default_org_id, 0) FROM users WHERE id = ?", userID).Scan(&email, &role, &defaultOrgID)
	if err != nil {
		a.logger.Error("fetch user for refresh", "user_id", userID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Issue new access token.
	accessToken, err := createJWT(userID, email, role, defaultOrgID, a.secret)
	if err != nil {
		a.logger.Error("create access token on refresh", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Issue new refresh token in the same family.
	newRefreshToken, err := a.issueRefreshToken(ctx, userID, family)
	if err != nil {
		a.logger.Error("issue new refresh token", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    accessToken,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   86400,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"token":         accessToken,
		"refresh_token": newRefreshToken,
	})
}
