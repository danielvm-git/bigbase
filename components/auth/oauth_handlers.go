package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

func (a *Auth) handleGoogleOAuth(w http.ResponseWriter, r *http.Request) {
	if a.googleClientID == "" {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	rawState, _ := kernel.GenerateID()

	// Accept optional ?redirect= for SPA token delivery.
	spaRedirect := r.URL.Query().Get("redirect")
	var googleState string
	var cookieValue string

	if spaRedirect != "" && len(a.spaOriginAllowlist) > 0 {
		if a.isSPAOriginAllowed(spaRedirect) {
			// Encode the SPA redirect into the signed state.
			cookieValue = SignSPAState(rawState, spaRedirect, a.secret)
			googleState = rawState // Google sees only the raw state.
		} else {
			// Redirect not allowed — fall back to standard flow.
			cookieValue = SignState(rawState, a.secret)
			googleState = rawState
		}
	} else {
		cookieValue = SignState(rawState, a.secret)
		googleState = rawState
	}

	// Set a signed state cookie so the callback can verify this flow originated here.
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    cookieValue,
		HttpOnly: true,
		Secure:   a.cookieSecure(r),
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   600,
	})

	redirectURI, ok := a.oauthCallbackRedirectURI()
	if !ok {
		kernel.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "public URL not configured"})
		return
	}
	url := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid+email+profile&state=%s",
		a.googleClientID, url.QueryEscape(redirectURI), googleState)
	http.Redirect(w, r, url, http.StatusFound)
}

func (a *Auth) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if a.googleClientID == "" {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "code required"})
		return
	}

	// Validate OAuth state to prevent CSRF.
	queryState := r.URL.Query().Get("state")
	stateCookie, stateErr := r.Cookie("oauth_state")

	// Try SPA state first (contains redirect URL), then fall back to standard state.
	var spaRedirect string
	stateValid := false
	if stateErr == nil {
		if s, r, ok := VerifySPAState(stateCookie.Value, a.secret); ok && s == queryState {
			stateValid = true
			spaRedirect = r
		}
	}
	if !stateValid {
		if stateErr != nil || !VerifyState(stateCookie.Value, queryState, a.secret) {
			a.clearOAuthStateCookie(w, r)
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid state"})
			return
		}
	}

	verifier := a.googleVerifier
	if verifier == nil {
		redirectURI, ok := a.oauthCallbackRedirectURI()
		if !ok {
			a.clearOAuthStateCookie(w, r)
			kernel.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "public URL not configured"})
			return
		}
		verifier = &realGoogleVerifier{
			clientID:     a.googleClientID,
			clientSecret: a.googleClientSecret,
			redirectURI:  redirectURI,
		}
	}

	googleUser, err := verifier.Verify(r.Context(), code)
	if err != nil {
		a.clearOAuthStateCookie(w, r)
		a.logger.Error("google token verification failed", "error", err)
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "google auth failed"})
		return
	}

	userID, orgID, err := a.findOrCreateGoogleUser(r.Context(), googleUser)
	if err != nil {
		a.clearOAuthStateCookie(w, r)
		a.logger.Error("find or create google user", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	token, err := createJWT(userID, googleUser.Email, "user", orgID, a.secret, a.accessExpiry)
	if err != nil {
		a.clearOAuthStateCookie(w, r)
		a.logger.Error("create jwt", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// SPA token delivery: redirect to SPA with #token=... instead of setting cookie.
	if spaRedirect != "" && a.isSPAOriginAllowed(spaRedirect) {
		a.clearOAuthStateCookie(w, r)
		spaURL := spaRedirect + "#token=" + url.QueryEscape(token)
		a.recordAudit("auth.oauth_callback", userID, googleUser.Email, getIP(r), nil)
		http.Redirect(w, r, spaURL, http.StatusFound)
		return
	}

	// Clear the OAuth state cookie now that the flow is complete.
	a.clearOAuthStateCookie(w, r)

	// Standard flow: set HttpOnly cookie and redirect to postLoginRedirect.
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		MaxAge:   86400,
	})
	a.recordAudit("auth.oauth_callback", userID, googleUser.Email, getIP(r), nil)
	http.Redirect(w, r, a.postLoginRedirect, http.StatusFound)
}

// handleLogout clears the authentication token cookie. This endpoint is for
// cookie-based clients. Bearer-token / localStorage clients should log out by
// dropping the token on the client. The server has no token blacklist.
func (a *Auth) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		MaxAge:   -1,
	})
	userID, _ := UserIDFromContext(r.Context())
	email, _ := UserEmailFromContext(r.Context())
	a.recordAudit("auth.logout", userID, email, getIP(r), nil)
	kernel.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleLogoutAll invalidates all refresh tokens for the authenticated user.
func (a *Auth) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization required"})
		return
	}
	if err := a.invalidateAllUserTokens(r.Context(), userID); err != nil {
		a.logger.Error("logout-all", "user_id", userID, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// findOrCreatePhoneUser finds or creates a user by phone number.
func (a *Auth) findOrCreatePhoneUser(ctx context.Context, phone string) (int64, int64, error) {
	email := fmt.Sprintf("phone-%s@bigbase.local", phone)
	var existingID, defaultOrgID int64
	err := a.db.QueryRowContext(ctx, "SELECT id, COALESCE(default_org_id, 0) FROM users WHERE phone = ?", phone).Scan(&existingID, &defaultOrgID)
	if err == nil {
		return existingID, defaultOrgID, nil
	}

	passwordHash, _ := hashPassword(generateTempPass())
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.db.ExecContext(ctx,
		"INSERT INTO users (email, phone, password_hash, role, created_at) VALUES (?, ?, ?, 'user', ?)",
		email, phone, passwordHash, now)
	if err != nil {
		return 0, 0, fmt.Errorf("insert phone user: %w", err)
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return 0, 0, fmt.Errorf("phone user last insert id: %w", err)
	}

	org, err := a.CreateOrg(ctx, email, email, userID)
	if err != nil {
		a.logger.Error("create personal org for phone user", "error", err)
		return userID, 0, nil
	}
	_, _ = a.db.ExecContext(ctx, "UPDATE users SET default_org_id = ? WHERE id = ?", org.ID, userID)

	return userID, org.ID, nil
}

func (a *Auth) clearOAuthStateCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		MaxAge:   -1,
		Secure:   a.cookieSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *Auth) findOrCreateGoogleUser(ctx context.Context, gu *GoogleUser) (int64, int64, error) {
	if gu.GoogleID != "" {
		var existingID, defaultOrgID int64
		err := a.db.QueryRowContext(ctx, "SELECT id, COALESCE(default_org_id, 0) FROM users WHERE google_id = ?", gu.GoogleID).Scan(&existingID, &defaultOrgID)
		if err == nil {
			return existingID, defaultOrgID, nil
		}
	}

	if gu.Email != "" {
		var existingID, defaultOrgID int64
		var existingGoogleID string
		err := a.db.QueryRowContext(ctx, "SELECT id, COALESCE(google_id,''), COALESCE(default_org_id, 0) FROM users WHERE email = ?", gu.Email).Scan(&existingID, &existingGoogleID, &defaultOrgID)
		if err == nil {
			if existingGoogleID == "" || existingGoogleID != gu.GoogleID {
				_, _ = a.db.ExecContext(ctx, "UPDATE users SET google_id = ?, avatar_url = ? WHERE id = ?", gu.GoogleID, gu.Avatar, existingID)
			}
			return existingID, defaultOrgID, nil
		}
	}

	passwordHash, _ := hashPassword(generateTempPass())
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.db.ExecContext(ctx,
		"INSERT INTO users (email, password_hash, google_id, avatar_url, role, created_at) VALUES (?, ?, ?, ?, 'user', ?)",
		gu.Email, passwordHash, gu.GoogleID, gu.Avatar, now)
	if err != nil {
		return 0, 0, fmt.Errorf("insert google user: %w", err)
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return 0, 0, fmt.Errorf("google user last insert id: %w", err)
	}

	// Auto-create personal org for Google OAuth users
	org, err := a.CreateOrg(ctx, gu.Email, gu.Email, userID)
	if err != nil {
		a.logger.Error("create personal org for google user", "error", err)
		return userID, 0, nil
	}
	_, _ = a.db.ExecContext(ctx, "UPDATE users SET default_org_id = ? WHERE id = ?", org.ID, userID)

	return userID, org.ID, nil
}

// findOrCreateEmailUser finds an existing user by email or creates a passwordless user.
func (a *Auth) findOrCreateEmailUser(ctx context.Context, email string) (int64, int64, error) {
	var existingID, defaultOrgID int64
	err := a.db.QueryRowContext(ctx, "SELECT id, COALESCE(default_org_id, 0) FROM users WHERE email = ?", email).Scan(&existingID, &defaultOrgID)
	if err == nil {
		return existingID, defaultOrgID, nil
	}

	// Create new passwordless user.
	passwordHash, _ := hashPassword(generateTempPass())
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.db.ExecContext(ctx,
		"INSERT INTO users (email, password_hash, role, created_at) VALUES (?, ?, 'user', ?)",
		email, passwordHash, now)
	if err != nil {
		return 0, 0, fmt.Errorf("insert email user: %w", err)
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return 0, 0, fmt.Errorf("email user last insert id: %w", err)
	}

	org, err := a.CreateOrg(ctx, email, email, userID)
	if err != nil {
		a.logger.Error("create personal org", "error", err)
		return userID, 0, nil
	}
	_, _ = a.db.ExecContext(ctx, "UPDATE users SET default_org_id = ? WHERE id = ?", org.ID, userID)

	return userID, org.ID, nil
}

type realGoogleVerifier struct {
	clientID     string
	clientSecret string
	redirectURI  string
}

func (v *realGoogleVerifier) Verify(ctx context.Context, code string) (*GoogleUser, error) {
	body := fmt.Sprintf("code=%s&client_id=%s&client_secret=%s&redirect_uri=%s&grant_type=authorization_code",
		code, v.clientID, v.clientSecret, v.redirectURI)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token error: status %d", resp.StatusCode)
	}

	var tokenResp struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	claims, err := decodeGoogleIDToken(tokenResp.IDToken)
	if err != nil {
		return nil, err
	}

	return &GoogleUser{
		GoogleID: claims.Sub,
		Email:    claims.Email,
		Name:     claims.Name,
		Avatar:   claims.Picture,
	}, nil
}

type googleIDClaims struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func decodeGoogleIDToken(token string) (*googleIDClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id_token: expected 3 parts, got %d", len(parts))
	}

	payload, err := jwtDecodePart(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode id_token payload: %w", err)
	}

	var claims googleIDClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse id_token claims: %w", err)
	}
	return &claims, nil
}

// jwtDecodePart decodes a base64url-encoded JWT part without padding
func jwtDecodePart(part string) ([]byte, error) {
	// Add padding if needed
	switch len(part) % 4 {
	case 2:
		part += "=="
	case 3:
		part += "="
	}
	return base64.RawURLEncoding.DecodeString(part)
}

// SignSPAState signs a state + spaRedirect pair with the server secret.
// Format: "rawState|base64(spaRedirect).hex(signature)"
func SignSPAState(state, spaRedirect string, secret []byte) string {
	payload := state + "|" + base64.RawURLEncoding.EncodeToString([]byte(spaRedirect))
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return payload + "." + hex.EncodeToString(mac.Sum(nil))
}

// VerifySPAState verifies a signed SPA state and extracts the original state and redirect.
func VerifySPAState(signed string, secret []byte) (state, redirect string, ok bool) {
	lastDot := strings.LastIndex(signed, ".")
	if lastDot < 1 || lastDot == len(signed)-1 {
		return "", "", false
	}
	payload := signed[:lastDot]
	expectedMAC := signed[lastDot+1:]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedMAC), []byte(expected)) {
		return "", "", false
	}
	parts := strings.SplitN(payload, "|", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	redirectBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", false
	}
	return parts[0], string(redirectBytes), true
}

// signState signs a random state string with the server secret using HMAC-SHA256.
// Returns "state.hex(signature)" — a tamper-evident token bound to this server instance.
func SignState(state string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(state))
	return state + "." + hex.EncodeToString(mac.Sum(nil))
}

// verifyState checks that value (a signed state from signState) matches the
// original state and was signed with the given secret. Uses constant-time comparison.
func VerifyState(value, state string, secret []byte) bool {
	lastDot := strings.LastIndex(value, ".")
	if lastDot < 1 || lastDot == len(value)-1 {
		return false
	}
	gotState := value[:lastDot]
	if gotState != state {
		return false
	}
	expectedMAC := value[lastDot+1:]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(state))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(expected))
}
