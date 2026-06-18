package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	otpCodeLen  = 6
	otpTTL      = 5 * time.Minute
	maxOTPAttempts = 3
	maxOTPsPerHour = 3
)

// otpRate tracks OTP sends per email for rate limiting.
type otpRate struct {
	count     int
	windowStart time.Time
}

// otpCodeStore is an in-memory store for OTP codes (used when no DB table yet).
// In production this should be a database table, but for testing we use in-memory.
var (
	otpStoreMu sync.Mutex
	otpStore   = map[string]*otpRecord{}
)

type otpRecord struct {
	email     string
	codeHash  string
	expiresAt time.Time
	attempts  int
}

// otpRateStore tracks rate limits.
var (
	otpRateMu   sync.Mutex
	otpRates    = map[string]*otpRate{}
)

// generateOTP generates a 6-digit numeric OTP code.
func generateOTP() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "000000"
	}
	n := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if n < 0 {
		n = -n
	}
	code := fmt.Sprintf("%06d", n%1000000)
	return code
}

// hashCode returns the SHA-256 hex of a code (never store plaintext).
func hashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}

func (a *Auth) handleSendOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if a.emailSender == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "OTP not configured"})
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}
	email := strings.ToLower(req.Email)

	// Rate limit: max 3 codes per email per hour.
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

	code := generateOTP()
	codeHash := hashCode(code)

	otpStoreMu.Lock()
	otpStore[email] = &otpRecord{
		email:     email,
		codeHash:  codeHash,
		expiresAt: now.Add(otpTTL),
		attempts:  0,
	}
	otpStoreMu.Unlock()

	a.emailSender.SendEmail(email, "Your verification code", fmt.Sprintf("Your code is: %s", code))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *Auth) handleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and code required"})
		return
	}
	email := strings.ToLower(req.Email)

	otpStoreMu.Lock()
	rec, ok := otpStore[email]
	if !ok {
		otpStoreMu.Unlock()
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid code"})
		return
	}
	if rec.attempts >= maxOTPAttempts || time.Now().After(rec.expiresAt) {
		delete(otpStore, email)
		otpStoreMu.Unlock()
		writeJSON(w, http.StatusGone, map[string]string{"error": "code expired"})
		return
	}
	if rec.codeHash != hashCode(req.Code) {
		rec.attempts++
		otpStoreMu.Unlock()
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid code"})
		return
	}
	delete(otpStore, email)
	otpStoreMu.Unlock()

	// Find or create user, return JWT.
	userID, orgID, err := a.findOrCreateEmailUser(r.Context(), email)
	if err != nil {
		a.logger.Error("find or create OTP user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	token, err := createJWT(userID, email, "user", orgID, a.secret)
	if err != nil {
		a.logger.Error("create JWT", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.writeAuthResponse(w, r, http.StatusOK, userID, email, token)
}
