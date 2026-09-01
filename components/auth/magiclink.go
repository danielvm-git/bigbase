package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

// magicLinkStore is an in-memory store for magic link tokens.
var (
	mlStoreMu sync.Mutex
	mlStore   = map[string]*magicLinkRecord{}
)

type magicLinkRecord struct {
	email     string
	tokenHash string
	expiresAt time.Time
	used      bool
}

func generateMagicToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (a *Auth) handleSendMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if a.emailSender == nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "magic link not configured"})
		return
	}

	var req struct {
		Email      string `json:"email"`
		RedirectTo string `json:"redirect_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}
	email := strings.ToLower(req.Email)

	// Rate limit: reuse OTP rate limit (3 per email per hour).
	now := time.Now()
	allowed, err := a.rateLimitStore.Increment(r.Context(), email, now, maxOTPsPerHour)
	if err != nil {
		a.logger.Error("magic link rate limit check failed", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if !allowed {
		kernel.WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		return
	}

	token, err := generateMagicToken()
	if err != nil {
		a.logger.Error("generate magic token", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(token))
	tokenHash := hex.EncodeToString(mac.Sum(nil))

	mlStoreMu.Lock()
	mlStore[token] = &magicLinkRecord{
		email:     email,
		tokenHash: tokenHash,
		expiresAt: now.Add(15 * time.Minute),
		used:      false,
	}
	mlStoreMu.Unlock()

	// Build the magic link.
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	link := fmt.Sprintf("%s://%s/api/auth/magic-link/verify?token=%s", scheme, r.Host, token)
	if req.RedirectTo != "" {
		link += "&redirect_to=" + url.QueryEscape(req.RedirectTo)
	}

	a.emailSender.SendEmail(email, "Sign in to BigBase", fmt.Sprintf("Click here to sign in: %s", link))
	kernel.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *Auth) handleVerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	redirectTo := r.URL.Query().Get("redirect_to")

	if token == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "token required"})
		return
	}

	mlStoreMu.Lock()
	rec, ok := mlStore[token]
	if !ok || rec.used || time.Now().After(rec.expiresAt) {
		mlStoreMu.Unlock()
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
		return
	}

	// Verify HMAC.
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(token))
	expectedHash := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(rec.tokenHash), []byte(expectedHash)) {
		mlStoreMu.Unlock()
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return
	}

	rec.used = true
	mlStoreMu.Unlock()

	email := rec.email

	// Find or create user, generate JWT.
	userID, orgID, err := a.findOrCreateEmailUser(r.Context(), email)
	if err != nil {
		a.logger.Error("find or create magic link user", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	jwt, err := createJWT(userID, email, "user", orgID, a.secret, a.accessExpiry)
	if err != nil {
		a.logger.Error("create JWT", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// SPA redirect with #token= if redirect_to is allowed.
	if redirectTo != "" && a.isSPAOriginAllowed(redirectTo) {
		target, err := url.Parse(strings.ReplaceAll(redirectTo, "\\", "/"))
		if err == nil && target != nil {
			target.Fragment = "token=" + url.QueryEscape(jwt)
			http.Redirect(w, r, target.String(), http.StatusFound)
			return
		}
	}

	// Standard flow: set cookie and redirect.
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    jwt,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		MaxAge:   86400,
	})
	http.Redirect(w, r, a.postLoginRedirect, http.StatusFound)
}
