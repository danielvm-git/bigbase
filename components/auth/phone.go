package auth

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

type PhoneSender interface {
	SendSMS(phone, message string) error
}

func (a *Auth) handleSendPhoneOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if a.emailSender == nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "phone auth not configured"})
		return
	}

	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Phone == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "phone required"})
		return
	}
	phone := strings.TrimSpace(req.Phone)
	key := "phone:" + phone

	// Rate limit: reuse OTP rate limiter with phone as key.
	now := time.Now()
	allowed, err := a.rateLimitStore.Increment(r.Context(), key, now, maxOTPsPerHour)
	if err != nil {
		a.logger.Error("phone OTP rate limit check failed", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if !allowed {
		kernel.WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		return
	}

	code := generateOTP()
	codeHash := hashCode(code)

	err = a.otpStore.Store(r.Context(), key, codeHash, now.Add(otpTTL))
	if err != nil {
		a.logger.Error("phone OTP store failed", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	// Send SMS via configured phone sender or log it (dev mode).
	if a.phoneSender != nil {
		if err := a.phoneSender.SendSMS(phone, fmt.Sprintf("Your BigBase code: %s", code)); err != nil {
			a.logger.Error("send sms", "error", err)
		}
	} else {
		a.logger.Info("phone OTP (dev mode — no phone sender configured)", "phone", phone, "code", code)
	}

	a.recordAudit("auth.phone_otp_sent", 0, phone, getIP(r), nil)
	kernel.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *Auth) handleVerifyPhoneOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Phone == "" || req.Code == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "phone and code required"})
		return
	}
	phone := strings.TrimSpace(req.Phone)
	key := "phone:" + phone

	rec, err := a.otpStore.Get(r.Context(), key)
	if err != nil {
		a.logger.Error("phone OTP retrieve failed", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if rec == nil {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid code"})
		return
	}
	if rec.attempts >= maxOTPAttempts || time.Now().After(rec.expiresAt) {
		_ = a.otpStore.Delete(r.Context(), key)
		kernel.WriteJSON(w, http.StatusGone, map[string]string{"error": "code expired"})
		return
	}
	inputHash := hashCode(req.Code)
	if subtle.ConstantTimeCompare([]byte(rec.codeHash), []byte(inputHash)) != 1 {
		_ = a.otpStore.RecordAttempt(r.Context(), key)
		a.recordAudit("auth.phone_otp_failed", 0, phone, getIP(r), nil)
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid code"})
		return
	}
	_ = a.otpStore.Delete(r.Context(), key)

	// Find or create user by phone.
	userID, orgID, err := a.findOrCreatePhoneUser(r.Context(), phone)
	if err != nil {
		a.logger.Error("find or create phone user", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	email := fmt.Sprintf("phone-%s@bigbase.local", phone)
	token, err := createJWT(userID, email, "user", orgID, a.secret, a.accessExpiry)
	if err != nil {
		a.logger.Error("create JWT", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.writeAuthResponse(w, r, http.StatusOK, userID, email, token)
	a.recordAudit("auth.phone_otp_verified", userID, phone, getIP(r), nil)
}
