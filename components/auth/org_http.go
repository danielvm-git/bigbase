package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielvm/bigbase/kernel"
)

func (a *Auth) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" || req.Slug == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name and slug required"})
		return
	}
	if !isValidSlug(req.Slug) {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "slug must be lowercase letters, numbers, and hyphens"})
		return
	}

	org, err := a.CreateOrg(r.Context(), req.Name, req.Slug, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "slug already exists"})
			return
		}
		a.logger.Error("create org", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusCreated, map[string]any{"data": org})
}

func (a *Auth) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())

	orgs, err := a.ListOrgsByOwner(r.Context(), userID)
	if err != nil {
		a.logger.Error("list orgs", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": orgs})
}

func (a *Auth) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	id, err := parseOrgID(idStr)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	org, orgErr := a.lookupOwnedOrg(r.Context(), id, userID)
	if orgErr != nil {
		kernel.WriteJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": org})
}

func (a *Auth) handleUpdateOrg(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	id, err := parseOrgID(idStr)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if _, orgErr := a.lookupOwnedOrg(r.Context(), id, userID); orgErr != nil {
		kernel.WriteJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" && req.Slug == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name or slug required"})
		return
	}
	if req.Slug != "" && !isValidSlug(req.Slug) {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "slug must be lowercase letters, numbers, and hyphens"})
		return
	}

	updatedOrg, err := a.UpdateOrg(r.Context(), id, req.Name, req.Slug)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "slug already exists"})
			return
		}
		a.logger.Error("update org", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": updatedOrg})
}

func (a *Auth) handleDeleteOrg(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	id, err := parseOrgID(idStr)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if _, orgErr := a.lookupOwnedOrg(r.Context(), id, userID); orgErr != nil {
		kernel.WriteJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	if err := a.DeleteOrg(r.Context(), id); err != nil {
		a.logger.Error("delete org", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// orgError is a structured error returned by lookupOwnedOrg.
type orgError struct {
	code    int
	message string
}

// lookupOwnedOrg fetches an org by ID and ensures the caller is the owner.
// Returns the org on success, or an orgError when not found/forbidden/errored.
func (a *Auth) lookupOwnedOrg(ctx context.Context, orgID, userID int64) (*Organization, *orgError) {
	org, err := a.GetOrgByID(ctx, orgID)
	if err != nil {
		a.logger.Error("lookup org", "error", err)
		return nil, &orgError{http.StatusInternalServerError, "internal error"}
	}
	if org == nil {
		return nil, &orgError{http.StatusNotFound, "org not found"}
	}
	if org.OwnerID != userID {
		return nil, &orgError{http.StatusForbidden, "insufficient permissions"}
	}
	return org, nil
}

func parseOrgID(idStr string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id: %s", idStr)
	}
	return id, nil
}

// isValidSlug returns true if s contains only lowercase letters, digits, and hyphens.
func isValidSlug(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return len(s) > 0
}

func generateTempPass() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *Auth) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if _, orgErr := a.lookupOwnedOrg(r.Context(), orgID, userID); orgErr != nil {
		kernel.WriteJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "valid email required"})
		return
	}
	if !validMemberRoles[req.Role] {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "role must be admin or member"})
		return
	}

	invite, err := a.CreateInvite(r.Context(), orgID, strings.ToLower(req.Email), req.Role)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "invite already exists for this email"})
			return
		}
		a.logger.Error("create invite", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusCreated, map[string]any{"data": invite})
}

func (a *Auth) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")
	token := r.PathValue("token")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	invite, err := a.GetInviteByToken(r.Context(), orgID, token)
	if err != nil {
		a.logger.Error("get invite by token", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if invite == nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "invite not found or already accepted"})
		return
	}

	if err := a.AcceptInvite(r.Context(), invite, userID); err != nil {
		a.logger.Error("accept invite", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (a *Auth) handleListMembers(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if _, orgErr := a.lookupOwnedOrg(r.Context(), orgID, userID); orgErr != nil {
		kernel.WriteJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	members, err := a.ListOrgMembers(r.Context(), orgID)
	if err != nil {
		a.logger.Error("list org members", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": members})
}

func (a *Auth) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if _, orgErr := a.lookupOwnedOrg(r.Context(), orgID, userID); orgErr != nil {
		kernel.WriteJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	var req struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}

	created, err := a.CreateAPIKey(r.Context(), orgID, req.Name, req.Scopes)
	if err != nil {
		a.logger.Error("create api key", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusCreated, map[string]any{"data": created})
	email, _ := UserEmailFromContext(r.Context())
	a.recordAudit("auth.api_key_created", userID, email, getIP(r), map[string]any{
		"org_id": orgID,
		"name":   req.Name,
	})
}

func (a *Auth) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if _, orgErr := a.lookupOwnedOrg(r.Context(), orgID, userID); orgErr != nil {
		kernel.WriteJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	keys, err := a.ListAPIKeys(r.Context(), orgID)
	if err != nil {
		a.logger.Error("list api keys", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": keys})
}

func (a *Auth) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")
	keyIDStr := r.PathValue("keyID")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if _, orgErr := a.lookupOwnedOrg(r.Context(), orgID, userID); orgErr != nil {
		kernel.WriteJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	keyID, err := parseOrgID(keyIDStr)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid key id"})
		return
	}

	if err := a.DeleteAPIKey(r.Context(), orgID, keyID); err != nil {
		a.logger.Error("delete api key", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	email, _ := UserEmailFromContext(r.Context())
	a.recordAudit("auth.api_key_deleted", userID, email, getIP(r), map[string]any{
		"org_id": orgID,
		"key_id": keyID,
	})
}

func (a *Auth) handleCreateSiteKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization required"})
		return
	}
	siteID := r.PathValue("id")

	var req struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Validate name length and characters
	if req.Name != "" {
		if len(req.Name) > 100 {
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name must be 100 characters or fewer"})
			return
		}
		for _, ch := range req.Name {
			if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' && ch != '_' && ch != ' ' {
				kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name must be alphanumeric, hyphens, underscores, or spaces"})
				return
			}
		}
	}

	// Validate scopes if provided
	if len(req.Scopes) > 0 {
		for _, s := range req.Scopes {
			if s != "deploy" {
				kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown scope: " + s})
				return
			}
		}
	}

	rawToken, keyID, err := a.CreateSiteKey(r.Context(), siteID, req.Name, req.Scopes)
	if err != nil {
		a.logger.Error("create site key", "key_id", keyID, "site_id", siteID, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]any{
			"key_id":  keyID,
			"site_id": siteID,
			"name":    req.Name,
			"key":     rawToken,
		},
	})
	email, _ := UserEmailFromContext(r.Context())
	a.recordAudit("auth.site_key_created", userID, email, getIP(r), map[string]any{
		"site_id": siteID,
		"key_id":  keyID,
	})
}

func (a *Auth) handleListSiteKeys(w http.ResponseWriter, r *http.Request) {
	if _, ok := UserIDFromContext(r.Context()); !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization required"})
		return
	}
	siteID := r.PathValue("id")

	keys, err := a.ListSiteKeys(r.Context(), siteID)
	if err != nil {
		a.logger.Error("list site keys", "site_id", siteID, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Convert to response format — metadata only, no raw tokens
	type keyEntry struct {
		KeyID      string  `json:"key_id"`
		Name       string  `json:"name"`
		Prefix     string  `json:"prefix"`
		CreatedAt  string  `json:"created_at"`
		LastUsedAt *string `json:"last_used_at,omitempty"`
	}
	entries := make([]keyEntry, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, keyEntry{
			KeyID:      k.KeyID,
			Name:       k.Name,
			Prefix:     k.Prefix,
			CreatedAt:  k.CreatedAt,
			LastUsedAt: k.LastUsedAt,
		})
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": entries})
}

func (a *Auth) handleRevokeSiteKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization required"})
		return
	}
	siteID := r.PathValue("id")
	keyID := r.PathValue("keyID")

	if err := a.RevokeSiteKey(r.Context(), keyID); err != nil {
		a.logger.Error("revoke site key", "key_id", keyID, "site_id", siteID, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
	email, _ := UserEmailFromContext(r.Context())
	a.recordAudit("auth.site_key_revoked", userID, email, getIP(r), map[string]any{
		"site_id": siteID,
		"key_id":  keyID,
	})
}

// SetOTPStore sets the OTP store. Used for testing.
