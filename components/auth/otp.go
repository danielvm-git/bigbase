package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	otpCodeLen     = 6
	otpTTL         = 5 * time.Minute
	maxOTPAttempts = 3
	maxOTPsPerHour = 3
)

// otpRate tracks OTP sends per email for rate limiting (in-memory test helper compatibility).
type otpRate struct {
	count       int
	windowStart time.Time
}

type otpRecord struct {
	key       string
	codeHash  string
	expiresAt time.Time
	attempts  int
}

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
	now := time.Now()
	allowed, err := a.rateLimitStore.Increment(r.Context(), email, now, maxOTPsPerHour)
	if err != nil {
		a.logger.Error("OTP rate limit check failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if !allowed {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		return
	}

	code := generateOTP()
	codeHash := hashCode(code)

	err = a.otpStore.Store(r.Context(), email, codeHash, now.Add(otpTTL))
	if err != nil {
		a.logger.Error("OTP store failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	a.emailSender.SendEmail(email, "Your verification code", fmt.Sprintf("Your code is: %s", code))
	a.recordAudit("auth.otp_sent", 0, email, getIP(r), nil)
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

	rec, err := a.otpStore.Get(r.Context(), email)
	if err != nil {
		a.logger.Error("OTP retrieve failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if rec == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid code"})
		return
	}
	if rec.attempts >= maxOTPAttempts || time.Now().After(rec.expiresAt) {
		_ = a.otpStore.Delete(r.Context(), email)
		writeJSON(w, http.StatusGone, map[string]string{"error": "code expired"})
		return
	}
	inputHash := hashCode(req.Code)
	if subtle.ConstantTimeCompare([]byte(rec.codeHash), []byte(inputHash)) != 1 {
		_ = a.otpStore.RecordAttempt(r.Context(), email)
		a.recordAudit("auth.otp_failed", 0, email, getIP(r), nil)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid code"})
		return
	}
	_ = a.otpStore.Delete(r.Context(), email)

	// Find or create user, return JWT.
	userID, orgID, err := a.findOrCreateEmailUser(r.Context(), email)
	if err != nil {
		a.logger.Error("find or create OTP user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	token, err := createJWT(userID, email, "user", orgID, a.secret, a.accessExpiry)
	if err != nil {
		a.logger.Error("create JWT", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.writeAuthResponse(w, r, http.StatusOK, userID, email, token)
	a.recordAudit("auth.otp_verified", userID, email, getIP(r), nil)
}
