package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPasswordReset(t *testing.T) {
	t.Run("forgot_password_sends_email", func(t *testing.T) {
		_, sender, handler, _ := setupAuthWithEmail(t)

		// Register a user first
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"reset@test.com","password":"oldpass1"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)
		if regW.Code != http.StatusCreated {
			t.Fatalf("register: %d %s", regW.Code, regW.Body.String())
		}

		// Request password reset
		fpReq := httptest.NewRequest("POST", "/api/auth/forgot-password",
			strings.NewReader(`{"email":"reset@test.com"}`))
		fpReq.Header.Set("Content-Type", "application/json")
		fpW := httptest.NewRecorder()
		handler.ServeHTTP(fpW, fpReq)
		if fpW.Code != http.StatusOK {
			t.Fatalf("forgot-password: expected 200, got %d: %s", fpW.Code, fpW.Body.String())
		}

		// Should have sent a reset email (verify email + reset email)
		resetEmailCount := 0
		for _, e := range sender.sent {
			if strings.Contains(e.Subject, "reset") || strings.Contains(e.Subject, "Reset") {
				resetEmailCount++
			}
		}
		if resetEmailCount == 0 {
			t.Fatal("expected password reset email to be sent")
		}
	})

	t.Run("forgot_password_unknown_email_returns_ok", func(t *testing.T) {
		// Should return 200 even for unknown emails (prevents user enumeration)
		_, _, handler, _ := setupAuthWithEmail(t)

		fpReq := httptest.NewRequest("POST", "/api/auth/forgot-password",
			strings.NewReader(`{"email":"nobody@test.com"}`))
		fpReq.Header.Set("Content-Type", "application/json")
		fpW := httptest.NewRecorder()
		handler.ServeHTTP(fpW, fpReq)
		if fpW.Code != http.StatusOK {
			t.Fatalf("expected 200 for unknown email, got %d", fpW.Code)
		}
	})

	t.Run("forgot_password_missing_email", func(t *testing.T) {
		_, _, handler, _ := setupAuthWithEmail(t)

		fpReq := httptest.NewRequest("POST", "/api/auth/forgot-password",
			strings.NewReader(`{}`))
		fpReq.Header.Set("Content-Type", "application/json")
		fpW := httptest.NewRecorder()
		handler.ServeHTTP(fpW, fpReq)
		if fpW.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing email, got %d", fpW.Code)
		}
	})

	t.Run("reset_password_success", func(t *testing.T) {
		a, sender, handler, _ := setupAuthWithEmail(t)

		// Register
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"rp@test.com","password":"oldpass1"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)
		if regW.Code != http.StatusCreated {
			t.Fatalf("register: %d", regW.Code)
		}

		// Request reset
		fpReq := httptest.NewRequest("POST", "/api/auth/forgot-password",
			strings.NewReader(`{"email":"rp@test.com"}`))
		fpReq.Header.Set("Content-Type", "application/json")
		fpW := httptest.NewRecorder()
		handler.ServeHTTP(fpW, fpReq)
		if fpW.Code != http.StatusOK {
			t.Fatalf("forgot-password: %d", fpW.Code)
		}

		// Find reset token in sent emails
		resetToken := ""
		for _, e := range sender.sent {
			if strings.Contains(e.Subject, "reset") || strings.Contains(e.Subject, "Reset") {
				resetToken = a.ExtractResetToken(e.Body)
				break
			}
		}
		if resetToken == "" {
			t.Fatalf("could not find reset token in emails: %v", sender.sent)
		}

		// Reset the password
		rpReq := httptest.NewRequest("POST", "/api/auth/reset-password",
			strings.NewReader(`{"token":"`+resetToken+`","new_password":"newpass1"}`))
		rpReq.Header.Set("Content-Type", "application/json")
		rpW := httptest.NewRecorder()
		handler.ServeHTTP(rpW, rpReq)
		if rpW.Code != http.StatusOK {
			t.Fatalf("reset-password: expected 200, got %d: %s", rpW.Code, rpW.Body.String())
		}

		// Login with new password should succeed
		loginReq := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"rp@test.com","password":"newpass1"}`))
		loginReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()
		handler.ServeHTTP(loginW, loginReq)
		if loginW.Code != http.StatusOK {
			t.Fatalf("login with new password: expected 200, got %d: %s", loginW.Code, loginW.Body.String())
		}

		// Login with old password should fail
		oldLoginReq := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"rp@test.com","password":"oldpass1"}`))
		oldLoginReq.Header.Set("Content-Type", "application/json")
		oldLoginW := httptest.NewRecorder()
		handler.ServeHTTP(oldLoginW, oldLoginReq)
		if oldLoginW.Code != http.StatusUnauthorized {
			t.Fatalf("login with old password: expected 401, got %d", oldLoginW.Code)
		}
	})

	t.Run("reset_password_invalid_token", func(t *testing.T) {
		_, _, handler, _ := setupAuthWithEmail(t)

		rpReq := httptest.NewRequest("POST", "/api/auth/reset-password",
			strings.NewReader(`{"token":"badtoken123","new_password":"newpass1"}`))
		rpReq.Header.Set("Content-Type", "application/json")
		rpW := httptest.NewRecorder()
		handler.ServeHTTP(rpW, rpReq)
		if rpW.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid token, got %d", rpW.Code)
		}
	})

	t.Run("reset_password_token_can_only_be_used_once", func(t *testing.T) {
		a, sender, handler, _ := setupAuthWithEmail(t)

		// Register + request reset
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"once@test.com","password":"oldpass1"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)

		fpReq := httptest.NewRequest("POST", "/api/auth/forgot-password",
			strings.NewReader(`{"email":"once@test.com"}`))
		fpReq.Header.Set("Content-Type", "application/json")
		fpW := httptest.NewRecorder()
		handler.ServeHTTP(fpW, fpReq)

		resetToken := ""
		for _, e := range sender.sent {
			if strings.Contains(e.Subject, "reset") || strings.Contains(e.Subject, "Reset") {
				resetToken = a.ExtractResetToken(e.Body)
				break
			}
		}
		if resetToken == "" {
			t.Fatal("no reset token found")
		}

		// First use succeeds
		rp1 := httptest.NewRequest("POST", "/api/auth/reset-password",
			strings.NewReader(`{"token":"`+resetToken+`","new_password":"newpass1"}`))
		rp1.Header.Set("Content-Type", "application/json")
		rp1W := httptest.NewRecorder()
		handler.ServeHTTP(rp1W, rp1)
		if rp1W.Code != http.StatusOK {
			t.Fatalf("first use: expected 200, got %d", rp1W.Code)
		}

		// Second use of same token fails
		rp2 := httptest.NewRequest("POST", "/api/auth/reset-password",
			strings.NewReader(`{"token":"`+resetToken+`","new_password":"newpass2"}`))
		rp2.Header.Set("Content-Type", "application/json")
		rp2W := httptest.NewRecorder()
		handler.ServeHTTP(rp2W, rp2)
		if rp2W.Code != http.StatusBadRequest {
			t.Fatalf("second use: expected 400, got %d", rp2W.Code)
		}
	})

	t.Run("reset_password_short_password_rejected", func(t *testing.T) {
		a, sender, handler, _ := setupAuthWithEmail(t)

		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"short@test.com","password":"oldpass1"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)

		fpReq := httptest.NewRequest("POST", "/api/auth/forgot-password",
			strings.NewReader(`{"email":"short@test.com"}`))
		fpReq.Header.Set("Content-Type", "application/json")
		fpW := httptest.NewRecorder()
		handler.ServeHTTP(fpW, fpReq)

		resetToken := ""
		for _, e := range sender.sent {
			if strings.Contains(e.Subject, "reset") || strings.Contains(e.Subject, "Reset") {
				resetToken = a.ExtractResetToken(e.Body)
				break
			}
		}
		if resetToken == "" {
			t.Fatal("no reset token found")
		}

		rpReq := httptest.NewRequest("POST", "/api/auth/reset-password",
			strings.NewReader(`{"token":"`+resetToken+`","new_password":"ab"}`))
		rpReq.Header.Set("Content-Type", "application/json")
		rpW := httptest.NewRecorder()
		handler.ServeHTTP(rpW, rpReq)
		if rpW.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for short password, got %d", rpW.Code)
		}
	})
}
