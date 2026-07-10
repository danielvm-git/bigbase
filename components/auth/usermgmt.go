package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danielvm/bigbase/kernel"
)

func (a *Auth) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PATCH" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID, _ := UserIDFromContext(r.Context())
	email, _ := UserEmailFromContext(r.Context())

	var req struct {
		Name      *string `json:"name"`
		AvatarURL *string `json:"avatar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.Name != nil {
		if len(*req.Name) > 100 {
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name too long (max 100 chars)"})
			return
		}
		_, _ = a.db.ExecContext(r.Context(), "UPDATE users SET name = ? WHERE id = ?", *req.Name, userID)
	}
	if req.AvatarURL != nil {
		if !strings.HasPrefix(*req.AvatarURL, "http://") && !strings.HasPrefix(*req.AvatarURL, "https://") {
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "avatar_url must be a valid URL"})
			return
		}
		_, _ = a.db.ExecContext(r.Context(), "UPDATE users SET avatar_url = ? WHERE id = ?", *req.AvatarURL, userID)
	}

	// Return updated user.
	var dbName, dbAvatar, dbEmail, dbRole string
	_ = a.db.QueryRowContext(r.Context(), "SELECT COALESCE(name,''), COALESCE(avatar_url,''), COALESCE(email,''), COALESCE(role,'') FROM users WHERE id = ?", userID).Scan(&dbName, &dbAvatar, &dbEmail, &dbRole)

	kernel.WriteJSON(w, http.StatusOK, map[string]any{
		"id":         userID,
		"email":      email,
		"name":       dbName,
		"avatar_url": dbAvatar,
		"role":       dbRole,
	})
}

func (a *Auth) handleListIdentities(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID, _ := UserIDFromContext(r.Context())

	var googleID, name, avatar, createdAt string
	_ = a.db.QueryRowContext(r.Context(), "SELECT COALESCE(google_id,''), COALESCE(name,''), COALESCE(avatar_url,''), COALESCE(created_at,'') FROM users WHERE id = ?", userID).Scan(&googleID, &name, &avatar, &createdAt)

	identities := []map[string]any{}
	if googleID != "" {
		userEmail, _ := UserEmailFromContext(r.Context())
		identities = append(identities, map[string]any{
			"provider":   "google",
			"id":         googleID,
			"email":      userEmail,
			"name":       name,
			"avatar_url": avatar,
			"created_at": createdAt,
		})
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"identities": identities})
}

func (a *Auth) handleLinkIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID, _ := UserIDFromContext(r.Context())

	var req struct {
		Provider string `json:"provider"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Provider != "google" || req.Code == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "provider and code required"})
		return
	}

	verifier := a.googleVerifier
	if verifier == nil {
		redirectURI, ok := a.oauthCallbackRedirectURI()
		if !ok {
			kernel.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "public URL not configured"})
			return
		}
		verifier = &realGoogleVerifier{
			clientID:     a.googleClientID,
			clientSecret: a.googleClientSecret,
			redirectURI:  redirectURI,
		}
	}

	googleUser, err := verifier.Verify(r.Context(), req.Code)
	if err != nil {
		a.logger.Error("link identity verify", "error", err)
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "google auth failed"})
		return
	}

	// Check google_id not already linked to another user.
	var existingID int64
	err = a.db.QueryRowContext(r.Context(), "SELECT id FROM users WHERE google_id = ? AND id != ?", googleUser.GoogleID, userID).Scan(&existingID)
	if err == nil {
		kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "google account already linked to another user"})
		return
	}

	_, _ = a.db.ExecContext(r.Context(), "UPDATE users SET google_id = ?, avatar_url = ? WHERE id = ?", googleUser.GoogleID, googleUser.Avatar, userID)

	kernel.WriteJSON(w, http.StatusOK, map[string]any{
		"provider":   "google",
		"id":         googleUser.GoogleID,
		"email":      googleUser.Email,
		"name":       googleUser.Name,
		"avatar_url": googleUser.Avatar,
	})
}

func (a *Auth) handleUnlinkIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID, _ := UserIDFromContext(r.Context())
	provider := r.PathValue("provider")

	// Can only unlink google provider.
	if provider != "google" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported provider"})
		return
	}

	// Check user has email/password as fallback.
	var email string
	err := a.db.QueryRowContext(r.Context(), "SELECT COALESCE(email,'') FROM users WHERE id = ?", userID).Scan(&email)
	if err != nil || email == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot unlink only identity — add email first"})
		return
	}

	_, _ = a.db.ExecContext(r.Context(), "UPDATE users SET google_id = NULL WHERE id = ?", userID)
	kernel.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
