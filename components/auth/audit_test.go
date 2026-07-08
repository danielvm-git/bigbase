package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

type auditRow struct {
	id        int64
	eventType string
	userID    sql.NullInt64
	email     sql.NullString
	ipAddress sql.NullString
	metadata  sql.NullString
	createdAt string
}

func queryAuditEvents(t *testing.T, d DBer) []auditRow {
	t.Helper()
	rows, err := d.QueryContext(context.Background(), "SELECT id, event_type, user_id, email, ip_address, metadata, created_at FROM audit_events ORDER BY id ASC")
	if err != nil {
		t.Fatalf("query audit events: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var events []auditRow
	for rows.Next() {
		var r auditRow
		if err := rows.Scan(&r.id, &r.eventType, &r.userID, &r.email, &r.ipAddress, &r.metadata, &r.createdAt); err != nil {
			t.Fatalf("scan audit row: %v", err)
		}
		events = append(events, r)
	}
	return events
}

func TestAuditEventsTable(t *testing.T) {
	logger := noopLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	k.Register(d)
	a := New(Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	k.Register(a)

	if err := k.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	a.recordAudit("test.event", 42, "user@test.com", "127.0.0.1", map[string]any{"foo": "bar"})

	// Wait for fire-and-forget goroutine to complete.
	time.Sleep(100 * time.Millisecond)

	events := queryAuditEvents(t, d)
	if len(events) != 1 {
		t.Fatalf("Expected 1 audit event, got %d", len(events))
	}
	e := events[0]
	if e.eventType != "test.event" {
		t.Errorf("Expected event_type='test.event', got %q", e.eventType)
	}
	if !e.userID.Valid || e.userID.Int64 != 42 {
		t.Errorf("Expected user_id=42, got %v", e.userID)
	}
	if !e.email.Valid || e.email.String != "user@test.com" {
		t.Errorf("Expected email='user@test.com', got %v", e.email)
	}
	if !e.ipAddress.Valid || e.ipAddress.String != "127.0.0.1" {
		t.Errorf("Expected ip='127.0.0.1', got %v", e.ipAddress)
	}
	if !e.metadata.Valid || !strings.Contains(e.metadata.String, `"foo":"bar"`) {
		t.Errorf("Expected metadata to contain foo:bar, got %q", e.metadata.String)
	}
}

func TestAuditFireAndForget(t *testing.T) {
	logger := noopLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	k.Register(d)
	a := New(Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	k.Register(a)

	if err := k.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	_, err := d.Exec("DROP TABLE audit_events")
	if err != nil {
		t.Fatalf("failed to drop table: %v", err)
	}

	a.recordAudit("test.failing.event", 1, "test@test.com", "1.1.1.1", nil)
}

type mockEmailSender struct{}

func (mockEmailSender) SendEmail(to, subject, body string) {}

func TestAuditIntegration(t *testing.T) {
	logger := noopLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	k.Register(d)
	a := New(Options{
		DB:          d,
		Logger:      logger,
		Secret:      "test-secret-32-chars!!!",
		EmailSender: mockEmailSender{},
	})
	k.Register(a)

	if err := k.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	// 1. auth.register
	regBody := `{"email":"audit@test.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBufferString(regBody))
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	a.handleRegister(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Register failed: %d, %s", w.Code, w.Body.String())
	}

	// 2. auth.login_failed (wrong password)
	loginFailBody := `{"email":"audit@test.com","password":"wrongpassword"}`
	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(loginFailBody))
	req.RemoteAddr = "127.0.0.2:1234"
	w = httptest.NewRecorder()
	a.handleLogin(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected unauthorized, got %d", w.Code)
	}

	// 3. auth.login (success)
	loginSuccessBody := `{"email":"audit@test.com","password":"password123"}`
	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(loginSuccessBody))
	req.RemoteAddr = "127.0.0.3:1234"
	w = httptest.NewRecorder()
	a.handleLogin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Login failed: %d", w.Code)
	}

	// Decode token to get userID and orgID for later tests
	var loginResp struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	userID := loginResp.User.ID
	if userID == 0 {
		t.Fatalf("Expected non-zero user ID in login response, got 0")
	}

	// 4. auth.logout
	req = httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.RemoteAddr = "127.0.0.4:1234"
	// Set the context user info to simulate logged-in user
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxUserEmail, "audit@test.com")
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	a.handleLogout(w, req)

	// 5. auth.anonymous_token
	req = httptest.NewRequest("POST", "/api/auth/anonymous", nil)
	req.RemoteAddr = "127.0.0.5:1234"
	w = httptest.NewRecorder()
	a.handleAnonymousToken(w, req)

	// 6. auth.otp_sent
	otpBody := `{"email":"otp@test.com"}`
	req = httptest.NewRequest("POST", "/api/auth/otp/send", bytes.NewBufferString(otpBody))
	req.RemoteAddr = "127.0.0.6:1234"
	w = httptest.NewRecorder()
	a.handleSendOTP(w, req)

	// 7. auth.otp_failed
	otpVerifyBody := `{"email":"otp@test.com","code":"000000"}` // bad code
	req = httptest.NewRequest("POST", "/api/auth/otp/verify", bytes.NewBufferString(otpVerifyBody))
	req.RemoteAddr = "127.0.0.7:1234"
	w = httptest.NewRecorder()
	a.handleVerifyOTP(w, req)

	// 8. auth.password_reset_requested
	forgotBody := `{"email":"audit@test.com"}`
	req = httptest.NewRequest("POST", "/api/auth/forgot-password", bytes.NewBufferString(forgotBody))
	req.RemoteAddr = "127.0.0.8:1234"
	w = httptest.NewRecorder()
	a.handleForgotPassword(w, req)

	// 9. auth.user_deleted
	req = httptest.NewRequest("DELETE", "/api/users/"+fmt.Sprint(userID), nil)
	req.SetPathValue("id", fmt.Sprint(userID))
	req.RemoteAddr = "127.0.0.9:1234"
	ctx = context.WithValue(req.Context(), ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxUserRole, "admin") // bypass requester check
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	a.handleUserByID(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete user failed: %d, %s", w.Code, w.Body.String())
	}

	// Fire-and-forget audit writes run in background goroutines; wait for them.
	expectedCount := 9
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		if events := queryAuditEvents(t, d); len(events) >= expectedCount {
			break
		}
	}

	// Query events and verify
	events := queryAuditEvents(t, d)
	expectedEvents := []struct {
		eventType string
		ipAddress string
	}{
		{"auth.register", "127.0.0.1"},
		{"auth.login_failed", "127.0.0.2"},
		{"auth.login", "127.0.0.3"},
		{"auth.logout", "127.0.0.4"},
		{"auth.anonymous_token", "127.0.0.5"},
		{"auth.otp_sent", "127.0.0.6"},
		{"auth.otp_failed", "127.0.0.7"},
		{"auth.password_reset_requested", "127.0.0.8"},
		{"auth.user_deleted", "127.0.0.9"},
	}

	if len(events) != len(expectedEvents) {
		t.Fatalf("Expected %d events, got %d", len(expectedEvents), len(events))
	}

	// Fire-and-forget goroutines may insert in any order — verify by set, not position.
	seen := make(map[string]int)
	for _, e := range events {
		seen[e.eventType+"|"+e.ipAddress.String]++
	}
	for _, exp := range expectedEvents {
		key := exp.eventType + "|" + exp.ipAddress
		if seen[key] != 1 {
			t.Errorf("Expected event %q, got %d occurrences", key, seen[key])
		}
	}
}
