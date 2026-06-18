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
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if a.emailSender == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "magic link not configured"})
		return
	}

	var req struct {
		Email      string `json:"email"`
		RedirectTo string `json:"redirect_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}
	email := strings.ToLower(req.Email)

	// Rate limit: reuse OTP rate limit (3 per email per hour).
	otpRateMu.Lock()
	rate, ok := otpRates[email]
	now := time.Now()
	if ok && now.Sub(rate.windowStart) < time.Hour {
		if rate.count >= maxOTPsPerHour {
			otpRateMu.Unlock()
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
			return
		}
		rate.count++
	} else {
		otpRates[email] = &otpRate{count: 1, windowStart: now}
	}
	otpRateMu.Unlock()

	token, err := generateMagicToken()
	if err != nil {
		a.logger.Error("generate magic token", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
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
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *Auth) handleVerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	redirectTo := r.URL.Query().Get("redirect_to")

	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token required"})
		return
	}

	mlStoreMu.Lock()
	rec, ok := mlStore[token]
	if !ok || rec.used || time.Now().After(rec.expiresAt) {
		mlStoreMu.Unlock()
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
		return
	}

	// Verify HMAC.
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(token))
	expectedHash := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(rec.tokenHash), []byte(expectedHash)) {
		mlStoreMu.Unlock()
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return
	}

	rec.used = true
	mlStoreMu.Unlock()

	email := rec.email

	// Find or create user, generate JWT.
	userID, orgID, err := a.findOrCreateEmailUser(r.Context(), email)
	if err != nil {
		a.logger.Error("find or create magic link user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	jwt, err := createJWT(userID, email, "user", orgID, a.secret)
	if err != nil {
		a.logger.Error("create JWT", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// SPA redirect with #token= if redirect_to is allowed.
	if redirectTo != "" && a.isSPAOriginAllowed(redirectTo) {
		spaURL := redirectTo + "#token=" + url.QueryEscape(jwt)
		http.Redirect(w, r, spaURL, http.StatusFound)
		return
	}

	// Standard flow: set cookie and redirect.
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    jwt,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400,
	})
	http.Redirect(w, r, a.postLoginRedirect, http.StatusFound)
}
