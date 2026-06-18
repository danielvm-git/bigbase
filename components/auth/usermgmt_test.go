package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateMe(t *testing.T) {
	_, h, ph := setupAuth(t) // from auth_test.go
	token, _ := registerAndLogin(t, h, "user@test.com", "abcdef")

	// Update name.
	body := `{"name":"Alice"}`
	req := httptest.NewRequest("PATCH", "/api/auth/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	ph.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["name"] != "Alice" {
		t.Errorf("expected name Alice, got %v", resp["name"])
	}
}

func TestUpdateMeUnauthenticated(t *testing.T) {
	_, _, ph := setupAuth(t)
	body := `{"name":"Alice"}`
	req := httptest.NewRequest("PATCH", "/api/auth/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ph.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestListIdentities(t *testing.T) {
	_, h, ph := setupAuth(t)
	token, _ := registerAndLogin(t, h, "ident@test.com", "abcdef")

	req := httptest.NewRequest("GET", "/api/auth/me/identities", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	ph.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Identities []map[string]any `json:"identities"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Identities) != 0 {
		t.Errorf("expected 0 identities for email-only user, got %d", len(resp.Identities))
	}
}
