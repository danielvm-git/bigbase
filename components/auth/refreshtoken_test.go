package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func setupAuthFresh(t *testing.T) (*auth.Auth, http.Handler, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})

	k.Register(a)
	k.Register(d)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return a, a.Handler(), a.ProtectedHandler()
}

func registerAndLogin(t *testing.T, handler http.Handler, email, password string) (accessToken, refreshToken string) {
	t.Helper()

	// Register
	regReq := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"`+email+`","password":"`+password+`"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", regW.Code, regW.Body.String())
	}

	// Login
	loginReq := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"`+email+`","password":"`+password+`"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusOK {
		t.Fatalf("login: %d %s", loginW.Code, loginW.Body.String())
	}

	resp := parseResponse(t, loginW.Body.Bytes())
	accessToken, _ = resp["token"].(string)
	refreshToken, _ = resp["refresh_token"].(string)
	return accessToken, refreshToken
}

func TestRefreshToken(t *testing.T) {
	t.Run("login_returns_refresh_token", func(t *testing.T) {
		_, handler, _ := setupAuthFresh(t)

		_, refreshToken := registerAndLogin(t, handler, "rt1@test.com", "secret123")
		if refreshToken == "" {
			t.Fatal("expected non-empty refresh_token in login response")
		}
	})

	t.Run("refresh_issues_new_tokens", func(t *testing.T) {
		_, handler, _ := setupAuthFresh(t)

		_, refreshToken := registerAndLogin(t, handler, "rt2@test.com", "secret123")

		// Use refresh token to get new tokens
		rfReq := httptest.NewRequest("POST", "/api/auth/refresh",
			strings.NewReader(`{"refresh_token":"`+refreshToken+`"}`))
		rfReq.Header.Set("Content-Type", "application/json")
		rfW := httptest.NewRecorder()
		handler.ServeHTTP(rfW, rfReq)
		if rfW.Code != http.StatusOK {
			t.Fatalf("refresh: expected 200, got %d: %s", rfW.Code, rfW.Body.String())
		}

		var rfResp map[string]any
		if err := json.NewDecoder(rfW.Body).Decode(&rfResp); err != nil {
			t.Fatalf("decode refresh: %v", err)
		}
		newAccess, _ := rfResp["token"].(string)
		newRefresh, _ := rfResp["refresh_token"].(string)

		if newAccess == "" {
			t.Fatal("expected new access token")
		}
		if newRefresh == "" {
			t.Fatal("expected new refresh token")
		}
		if newRefresh == refreshToken {
			t.Fatal("expected new refresh token to differ from old one (rotation)")
		}
	})

	t.Run("old_refresh_token_invalidated_after_rotation", func(t *testing.T) {
		_, handler, _ := setupAuthFresh(t)

		_, refreshToken := registerAndLogin(t, handler, "rt3@test.com", "secret123")

		// Rotate once
		rfReq := httptest.NewRequest("POST", "/api/auth/refresh",
			strings.NewReader(`{"refresh_token":"`+refreshToken+`"}`))
		rfReq.Header.Set("Content-Type", "application/json")
		rfW := httptest.NewRecorder()
		handler.ServeHTTP(rfW, rfReq)
		if rfW.Code != http.StatusOK {
			t.Fatalf("rotate: %d", rfW.Code)
		}

		// Replay the old token — should be rejected
		rfReq2 := httptest.NewRequest("POST", "/api/auth/refresh",
			strings.NewReader(`{"refresh_token":"`+refreshToken+`"}`))
		rfReq2.Header.Set("Content-Type", "application/json")
		rfW2 := httptest.NewRecorder()
		handler.ServeHTTP(rfW2, rfReq2)
		if rfW2.Code != http.StatusUnauthorized {
			t.Fatalf("replay old token: expected 401, got %d: %s", rfW2.Code, rfW2.Body.String())
		}
	})

	t.Run("family_invalidation_on_replay", func(t *testing.T) {
		_, handler, _ := setupAuthFresh(t)

		_, rt0 := registerAndLogin(t, handler, "rt4@test.com", "secret123")

		// Rotate to get rt1
		rfReq := httptest.NewRequest("POST", "/api/auth/refresh",
			strings.NewReader(`{"refresh_token":"`+rt0+`"}`))
		rfReq.Header.Set("Content-Type", "application/json")
		rfW := httptest.NewRecorder()
		handler.ServeHTTP(rfW, rfReq)
		if rfW.Code != http.StatusOK {
			t.Fatalf("first rotation: %d", rfW.Code)
		}

		var rfResp map[string]any
		if err := json.NewDecoder(rfW.Body).Decode(&rfResp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		rt1, _ := rfResp["refresh_token"].(string)

		// Rotate rt1 to get rt2
		rfReq2 := httptest.NewRequest("POST", "/api/auth/refresh",
			strings.NewReader(`{"refresh_token":"`+rt1+`"}`))
		rfReq2.Header.Set("Content-Type", "application/json")
		rfW2 := httptest.NewRecorder()
		handler.ServeHTTP(rfW2, rfReq2)
		if rfW2.Code != http.StatusOK {
			t.Fatalf("second rotation: %d", rfW2.Code)
		}

		// Replay rt0 (already used) — should invalidate ALL tokens for this user
		replayReq := httptest.NewRequest("POST", "/api/auth/refresh",
			strings.NewReader(`{"refresh_token":"`+rt0+`"}`))
		replayReq.Header.Set("Content-Type", "application/json")
		replayW := httptest.NewRecorder()
		handler.ServeHTTP(replayW, replayReq)
		if replayW.Code != http.StatusUnauthorized {
			t.Fatalf("replay rt0: expected 401, got %d", replayW.Code)
		}

		var rfResp2Body map[string]any
		_ = json.NewDecoder(rfW2.Body).Decode(&rfResp2Body)
		rt2, _ := rfResp2Body["refresh_token"].(string)
		if rt2 == "" {
			t.Fatal("expected rt2 from second rotation")
		}

		// rt1 should also be invalidated (family invalidation)
		rfResp2 := httptest.NewRequest("POST", "/api/auth/refresh",
			strings.NewReader(`{"refresh_token":"`+rt1+`"}`))
		rfResp2.Header.Set("Content-Type", "application/json")
		rfRespW2 := httptest.NewRecorder()
		handler.ServeHTTP(rfRespW2, rfResp2)
		if rfRespW2.Code != http.StatusUnauthorized {
			t.Fatalf("rt1 after family invalidation: expected 401, got %d", rfRespW2.Code)
		}

		// Current tip of family (rt2) must also be revoked after replay detection
		rfResp3 := httptest.NewRequest("POST", "/api/auth/refresh",
			strings.NewReader(`{"refresh_token":"`+rt2+`"}`))
		rfResp3.Header.Set("Content-Type", "application/json")
		rfRespW3 := httptest.NewRecorder()
		handler.ServeHTTP(rfRespW3, rfResp3)
		if rfRespW3.Code != http.StatusUnauthorized {
			t.Fatalf("rt2 after family invalidation: expected 401, got %d body=%s", rfRespW3.Code, rfRespW3.Body.String())
		}
	})

	t.Run("invalid_refresh_token_rejected", func(t *testing.T) {
		_, handler, _ := setupAuthFresh(t)

		rfReq := httptest.NewRequest("POST", "/api/auth/refresh",
			strings.NewReader(`{"refresh_token":"bad.token.here"}`))
		rfReq.Header.Set("Content-Type", "application/json")
		rfW := httptest.NewRecorder()
		handler.ServeHTTP(rfW, rfReq)
		if rfW.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for invalid token, got %d", rfW.Code)
		}
	})

	t.Run("missing_refresh_token_rejected", func(t *testing.T) {
		_, handler, _ := setupAuthFresh(t)

		rfReq := httptest.NewRequest("POST", "/api/auth/refresh",
			strings.NewReader(`{}`))
		rfReq.Header.Set("Content-Type", "application/json")
		rfW := httptest.NewRecorder()
		handler.ServeHTTP(rfW, rfReq)
		if rfW.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing token, got %d", rfW.Code)
		}
	})
}
