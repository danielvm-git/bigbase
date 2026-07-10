package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/danielvm/bigbase/kernel"
)

func (a *Auth) handleAnonymousToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// Only available when configured.
	if a.googleClientID == "" && a.emailSender == nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "anonymous auth not configured"})
		return
	}

	// Generate anonymous JWT with role=anonymous, 1 hour TTL.
	now := time.Now()
	claims := Claims{
		UserID: 0,
		Email:  "",
		Role:   "anonymous",
		OrgID:  0,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "anon-" + generateAnonymousID(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(a.secret)
	if err != nil {
		a.logger.Error("sign anonymous token", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]string{"token": signed})
	a.recordAudit("auth.anonymous_token", 0, "", getIP(r), map[string]any{
		"subject": claims.Subject,
	})
}

func generateAnonymousID() string {
	id, _ := kernel.GenerateID()
	return "anon-" + id[:12]
}

func (a *Auth) handlePopupCallback(w http.ResponseWriter, r *http.Request) {
	// OAuth popup callback: renders minimal HTML that does postMessage to parent.
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "code required"})
		return
	}

	// Validate state (reuse OAuth state validation).
	stateCookie, stateErr := r.Cookie("oauth_state")
	var spaRedirect string
	stateValid := false
	if stateErr == nil {
		if s, rdt, ok := VerifySPAState(stateCookie.Value, a.secret); ok && s == state {
			stateValid = true
			spaRedirect = rdt
		}
	}
	if !stateValid {
		if stateErr != nil || !VerifyState(stateCookie.Value, state, a.secret) {
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid state"})
			return
		}
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

	googleUser, err := verifier.Verify(r.Context(), code)
	if err != nil {
		a.logger.Error("popup verify", "error", err)
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "google auth failed"})
		return
	}

	userID, orgID, err := a.findOrCreateGoogleUser(r.Context(), googleUser)
	if err != nil {
		a.logger.Error("popup find user", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	jwtToken, err := createJWT(userID, googleUser.Email, "user", orgID, a.secret, a.accessExpiry)
	if err != nil {
		a.logger.Error("popup jwt", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Clear state cookie.
	a.clearOAuthStateCookie(w, r)

	// Determine target origin for postMessage.
	targetOrigin := "*"
	if spaRedirect != "" && a.isSPAOriginAllowed(spaRedirect) {
		targetOrigin = spaRedirect[:strings.Index(spaRedirect, "/")]
	}

	// Render HTML page that does window.opener.postMessage.
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><script>
window.opener.postMessage({type:"bigbase-auth:oauth-complete",token:"%s"}, "%s");
window.close();
</script></head><body>Sign in complete. You may close this window.</body></html>`, jwtToken, targetOrigin)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}
