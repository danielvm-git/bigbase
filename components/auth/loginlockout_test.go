package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginLockoutAfterMaxFailures(t *testing.T) {
	_, handler, _ := setupAuth(t)

	registerReq := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"lockout@example.com","password":"correct-pass"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	handler.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register: got %d", registerRec.Code)
	}

	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"lockout@example.com","password":"wrong-pass"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if i < 5 {
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("attempt %d: expected 401, got %d", i, rec.Code)
			}
		} else if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d: expected 429 lockout, got %d body=%s", i, rec.Code, rec.Body.String())
		}
	}

	// 6th attempt while locked — still 429 even with correct password
	req := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"lockout@example.com","password":"correct-pass"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("locked login with correct password: expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on lockout response")
	}
}

func TestLoginClearsLockoutOnSuccess(t *testing.T) {
	_, handler, _ := setupAuth(t)

	registerReq := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"clear@example.com","password":"good-pass"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	handler.ServeHTTP(registerRec, registerReq)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"clear@example.com","password":"bad-pass"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("failed attempt %d: got %d", i+1, rec.Code)
		}
	}

	loginReq := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"clear@example.com","password":"good-pass"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("successful login after failures: expected 200, got %d body=%s", loginRec.Code, loginRec.Body.String())
	}
}
