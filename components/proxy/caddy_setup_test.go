package proxy_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupVPSUsesOnDemandTLS(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "setup-vps.sh")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read setup-vps.sh: %v", err)
	}
	s := string(data)
	for _, needle := range []string{
		"on_demand_tls",
		"on_demand",
		"/api/internal/caddy-allow",
	} {
		if !strings.Contains(s, needle) {
			t.Fatalf("setup-vps.sh missing %q", needle)
		}
	}
}
