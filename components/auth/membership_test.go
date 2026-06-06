package auth_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

// setupMembership creates two users and an org owned by user1.
// Returns (authInstance, publicHandler, protectedHandler, ownerToken, memberToken, orgID).
func setupMembership(t *testing.T) (*auth.Auth, http.Handler, http.Handler, string, string, float64) {
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

	pub := a.Handler()
	prot := a.ProtectedHandler()

	// Register owner
	regOwner := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"owner@test.com","password":"secret123"}`))
	regOwner.Header.Set("Content-Type", "application/json")
	regOwnerW := httptest.NewRecorder()
	pub.ServeHTTP(regOwnerW, regOwner)
	if regOwnerW.Code != http.StatusCreated {
		t.Fatalf("register owner: %d %s", regOwnerW.Code, regOwnerW.Body.String())
	}
	ownerToken := parseResponse(t, regOwnerW.Body.Bytes())["token"].(string)

	// Register future member
	regMember := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"member@test.com","password":"secret123"}`))
	regMember.Header.Set("Content-Type", "application/json")
	regMemberW := httptest.NewRecorder()
	pub.ServeHTTP(regMemberW, regMember)
	if regMemberW.Code != http.StatusCreated {
		t.Fatalf("register member: %d %s", regMemberW.Code, regMemberW.Body.String())
	}
	memberToken := parseResponse(t, regMemberW.Body.Bytes())["token"].(string)

	// Create org owned by owner
	createOrg := httptest.NewRequest("POST", "/api/orgs",
		strings.NewReader(`{"name":"Test Org","slug":"test-org"}`))
	createOrg.Header.Set("Content-Type", "application/json")
	createOrg.Header.Set("Authorization", "Bearer "+ownerToken)
	createOrgW := httptest.NewRecorder()
	prot.ServeHTTP(createOrgW, createOrg)
	if createOrgW.Code != http.StatusCreated {
		t.Fatalf("create org: %d %s", createOrgW.Code, createOrgW.Body.String())
	}
	var orgResp map[string]any
	if err := json.NewDecoder(createOrgW.Body).Decode(&orgResp); err != nil {
		t.Fatalf("decode org: %v", err)
	}
	orgID := orgResp["data"].(map[string]any)["id"].(float64)

	return a, pub, prot, ownerToken, memberToken, orgID
}

func TestMembership(t *testing.T) {
	t.Run("invite_by_email", func(t *testing.T) {
		_, _, prot, ownerToken, _, orgID := setupMembership(t)

		body := `{"email":"member@test.com","role":"member"}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/invites", orgID),
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+ownerToken)
		w := httptest.NewRecorder()
		prot.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("invite: expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		data, ok := resp["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data object, got: %v", resp)
		}
		if data["token"] == "" || data["token"] == nil {
			t.Error("expected non-empty invite token in response")
		}
		if data["role"] != "member" {
			t.Errorf("expected role 'member', got %v", data["role"])
		}
	})

	t.Run("invite_invalid_role", func(t *testing.T) {
		_, _, prot, ownerToken, _, orgID := setupMembership(t)

		body := `{"email":"member@test.com","role":"superadmin"}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/invites", orgID),
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+ownerToken)
		w := httptest.NewRecorder()
		prot.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid role, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invite_non_owner_forbidden", func(t *testing.T) {
		_, _, prot, _, memberToken, orgID := setupMembership(t)

		body := `{"email":"other@test.com","role":"member"}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/invites", orgID),
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		w := httptest.NewRecorder()
		prot.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("accept_invite", func(t *testing.T) {
		_, _, prot, ownerToken, memberToken, orgID := setupMembership(t)

		// Create invite
		inviteBody := `{"email":"member@test.com","role":"member"}`
		inviteReq := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/invites", orgID),
			strings.NewReader(inviteBody))
		inviteReq.Header.Set("Content-Type", "application/json")
		inviteReq.Header.Set("Authorization", "Bearer "+ownerToken)
		inviteW := httptest.NewRecorder()
		prot.ServeHTTP(inviteW, inviteReq)
		if inviteW.Code != http.StatusCreated {
			t.Fatalf("invite: %d %s", inviteW.Code, inviteW.Body.String())
		}
		var inviteResp map[string]any
		if err := json.NewDecoder(inviteW.Body).Decode(&inviteResp); err != nil {
			t.Fatalf("decode invite: %v", err)
		}
		inviteToken := inviteResp["data"].(map[string]any)["token"].(string)

		// Accept invite as member
		acceptReq := httptest.NewRequest("POST",
			fmt.Sprintf("/api/orgs/%.0f/invites/%s/accept", orgID, inviteToken),
			nil)
		acceptReq.Header.Set("Authorization", "Bearer "+memberToken)
		acceptW := httptest.NewRecorder()
		prot.ServeHTTP(acceptW, acceptReq)

		if acceptW.Code != http.StatusOK {
			t.Fatalf("accept invite: expected 200, got %d: %s", acceptW.Code, acceptW.Body.String())
		}
	})

	t.Run("accept_invalid_token", func(t *testing.T) {
		_, _, prot, _, memberToken, orgID := setupMembership(t)

		acceptReq := httptest.NewRequest("POST",
			fmt.Sprintf("/api/orgs/%.0f/invites/bad-token/accept", orgID),
			nil)
		acceptReq.Header.Set("Authorization", "Bearer "+memberToken)
		acceptW := httptest.NewRecorder()
		prot.ServeHTTP(acceptW, acceptReq)

		if acceptW.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for invalid token, got %d: %s", acceptW.Code, acceptW.Body.String())
		}
	})

	t.Run("list_members", func(t *testing.T) {
		_, _, prot, ownerToken, memberToken, orgID := setupMembership(t)

		// Invite and accept
		inviteBody := `{"email":"member@test.com","role":"admin"}`
		inviteReq := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/invites", orgID),
			strings.NewReader(inviteBody))
		inviteReq.Header.Set("Content-Type", "application/json")
		inviteReq.Header.Set("Authorization", "Bearer "+ownerToken)
		inviteW := httptest.NewRecorder()
		prot.ServeHTTP(inviteW, inviteReq)
		if inviteW.Code != http.StatusCreated {
			t.Fatalf("invite: %d %s", inviteW.Code, inviteW.Body.String())
		}
		var inviteResp map[string]any
		_ = json.NewDecoder(inviteW.Body).Decode(&inviteResp)
		inviteToken := inviteResp["data"].(map[string]any)["token"].(string)

		acceptReq := httptest.NewRequest("POST",
			fmt.Sprintf("/api/orgs/%.0f/invites/%s/accept", orgID, inviteToken),
			nil)
		acceptReq.Header.Set("Authorization", "Bearer "+memberToken)
		acceptW := httptest.NewRecorder()
		prot.ServeHTTP(acceptW, acceptReq)
		if acceptW.Code != http.StatusOK {
			t.Fatalf("accept: %d %s", acceptW.Code, acceptW.Body.String())
		}

		// List members
		listReq := httptest.NewRequest("GET", fmt.Sprintf("/api/orgs/%.0f/members", orgID), nil)
		listReq.Header.Set("Authorization", "Bearer "+ownerToken)
		listW := httptest.NewRecorder()
		prot.ServeHTTP(listW, listReq)

		if listW.Code != http.StatusOK {
			t.Fatalf("list members: expected 200, got %d: %s", listW.Code, listW.Body.String())
		}
		var listResp map[string]any
		if err := json.NewDecoder(listW.Body).Decode(&listResp); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		members, ok := listResp["data"].([]any)
		if !ok {
			t.Fatalf("expected data array, got: %v", listResp)
		}
		// Should contain the accepted member
		if len(members) < 1 {
			t.Fatalf("expected at least 1 member, got %d", len(members))
		}
	})

	t.Run("invite_missing_email", func(t *testing.T) {
		_, _, prot, ownerToken, _, orgID := setupMembership(t)

		body := `{"role":"member"}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/invites", orgID),
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+ownerToken)
		w := httptest.NewRecorder()
		prot.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("duplicate_invite_conflict", func(t *testing.T) {
		_, _, prot, ownerToken, _, orgID := setupMembership(t)

		body := `{"email":"member@test.com","role":"member"}`
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/orgs/%.0f/invites", orgID),
				strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+ownerToken)
			w := httptest.NewRecorder()
			prot.ServeHTTP(w, req)
			if i == 0 && w.Code != http.StatusCreated {
				t.Fatalf("first invite: expected 201, got %d: %s", w.Code, w.Body.String())
			}
			if i == 1 && w.Code != http.StatusConflict {
				t.Fatalf("duplicate invite: expected 409, got %d: %s", w.Code, w.Body.String())
			}
		}
	})
}
