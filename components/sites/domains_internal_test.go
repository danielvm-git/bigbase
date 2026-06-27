package sites

import (
	"testing"
	"time"
)

func TestCertStatus(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		notAfter   time.Time
		haveCert   bool
		wantStatus string
		wantExpiry bool
	}{
		{"no cert yet is provisioning", time.Time{}, false, "provisioning", false},
		{"far future is valid", now.Add(90 * 24 * time.Hour), true, "valid", true},
		{"within 30 days is expiring_soon", now.Add(10 * 24 * time.Hour), true, "expiring_soon", true},
		{"just under window is expiring_soon", now.Add(certWarnWindow - time.Hour), true, "expiring_soon", true},
		{"past expiry is expired", now.Add(-time.Hour), true, "expired", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, expiry := certStatus(tt.notAfter, tt.haveCert, now)
			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
			if (expiry != nil) != tt.wantExpiry {
				t.Errorf("expiry present = %v, want %v", expiry != nil, tt.wantExpiry)
			}
		})
	}
}

func TestVerifyLimiter(t *testing.T) {
	lim := newVerifyLimiter(5 * time.Second)
	base := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)

	if !lim.allow("site|a.com", base) {
		t.Fatal("first call should be allowed")
	}
	if lim.allow("site|a.com", base.Add(time.Second)) {
		t.Error("second call within window should be throttled")
	}
	if !lim.allow("site|a.com", base.Add(6*time.Second)) {
		t.Error("call after window should be allowed")
	}
	if !lim.allow("site|b.com", base.Add(time.Second)) {
		t.Error("different key should have its own bucket")
	}
}
