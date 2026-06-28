package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PhoneSender interface {
	SendSMS(phone, message string) error
}

func (a *Auth) handleSendPhoneOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if a.emailSender == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "phone auth not configured"})
		return
	}

	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Phone == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone required"})
		return
	}
	phone := strings.TrimSpace(req.Phone)

	// Rate limit: reuse OTP rate limiter with phone as key.
	otpRateMu.Lock()
	rate, ok := otpRates["phone:"+phone]
	now := time.Now()
	if ok && now.Sub(rate.windowStart) < time.Hour {
		if rate.count >= maxOTPsPerHour {
			otpRateMu.Unlock()
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
			return
		}
		rate.count++
	} else {
		otpRates["phone:"+phone] = &otpRate{count: 1, windowStart: now}
	}
	otpRateMu.Unlock()

	code := generateOTP()
	codeHash := hashCode(code)

	otpStoreMu.Lock()
	otpStore["phone:"+phone] = &otpRecord{
		email:     phone,
		codeHash:  codeHash,
		expiresAt: now.Add(otpTTL),
		attempts:  0,
	}
	otpStoreMu.Unlock()

	// Send SMS via configured phone sender or log it (dev mode).
	if a.phoneSender != nil {
		if err := a.phoneSender.SendSMS(phone, fmt.Sprintf("Your BigBase code: %s", code)); err != nil {
			a.logger.Error("send sms", "error", err)
		}
	} else {
		a.logger.Info("phone OTP (dev mode — no phone sender configured)", "phone", phone, "code", code)
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *Auth) handleVerifyPhoneOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Phone == "" || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone and code required"})
		return
	}
	phone := strings.TrimSpace(req.Phone)

	otpStoreMu.Lock()
	rec, ok := otpStore["phone:"+phone]
	if !ok {
		otpStoreMu.Unlock()
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid code"})
		return
	}
	if rec.attempts >= maxOTPAttempts || time.Now().After(rec.expiresAt) {
		delete(otpStore, "phone:"+phone)
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
	delete(otpStore, "phone:"+phone)
	otpStoreMu.Unlock()

	// Find or create user by phone.
	userID, orgID, err := a.findOrCreatePhoneUser(r.Context(), phone)
	if err != nil {
		a.logger.Error("find or create phone user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	email := fmt.Sprintf("phone-%s@bigbase.local", phone)
	token, err := createJWT(userID, email, "user", orgID, a.secret, a.accessExpiry)
	if err != nil {
		a.logger.Error("create JWT", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.writeAuthResponse(w, r, http.StatusOK, userID, email, token)
}
