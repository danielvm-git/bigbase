package deploy

import (
	"testing"
	"time"
)

func TestSupervisor_HealthPolling(t *testing.T) {
	if healthPollInterval != 10*time.Second {
		t.Errorf("healthPollInterval = %v, want 10s", healthPollInterval)
	}
	if healthFailureThreshold != 3 {
		t.Errorf("healthFailureThreshold = %d, want 3", healthFailureThreshold)
	}
}

func TestSpec_HealthFields(t *testing.T) {
	spec := Spec{
		DeployID:   "test-1",
		HealthPort: 0,
	}
	if spec.HealthPort != 0 {
		t.Error("HealthPort should default to 0 (disabled)")
	}

	spec2 := Spec{
		DeployID:   "test-2",
		HealthPort: 8080,
		HealthPath: "/health",
	}
	if spec2.HealthPort != 8080 {
		t.Error("HealthPort should be set")
	}
	if spec2.HealthPath != "/health" {
		t.Errorf("HealthPath = %s, want /health", spec2.HealthPath)
	}
}

func TestSpec_WritableDir(t *testing.T) {
	spec := Spec{
		DeployID:    "test-1",
		WritableDir: "/data/writable/test-1",
	}
	if spec.WritableDir != "/data/writable/test-1" {
		t.Errorf("WritableDir = %s", spec.WritableDir)
	}
}
