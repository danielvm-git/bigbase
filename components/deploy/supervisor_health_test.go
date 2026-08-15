package deploy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
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

// e73s03 coverage: the Supervisor's health-poll loop. After
// healthFailureThreshold consecutive unhealthy responses it must emit the
// deploy.health_failed event and stop the instance (which triggers the
// restart loop). The loop had no direct unit test before this.
func TestSupervisorHealthLoop_RestartsAfterThreshold(t *testing.T) {
	// Tighten the poll interval so three failures happen in milliseconds.
	orig := healthPollInterval
	healthPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { healthPollInterval = orig })

	// An always-unhealthy endpoint (500 >= 400 counts as a failure).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	port, err := strconv.Atoi(srv.URL[strings.LastIndex(srv.URL, ":")+1:])
	if err != nil {
		t.Fatalf("parse test server port from %q: %v", srv.URL, err)
	}

	var mu sync.Mutex
	var events []string
	s := NewSupervisor(nil, nil, nil, nil, func(name string, _ Spec) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})

	inst := newFakeInstance(nil)
	spec := Spec{DeployID: "health-1", HealthPort: port, HealthPath: "/health"}

	done := make(chan struct{})
	go s.healthLoop(context.Background(), spec, inst, done)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("healthLoop did not stop the instance within 5s")
	}

	if !inst.stopped {
		t.Error("instance should be stopped after healthFailureThreshold failures")
	}
	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, e := range events {
		if e == "deploy.health_failed" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected deploy.health_failed event, got %v", events)
	}
}

// A healthy endpoint must NOT trigger a restart; the loop keeps polling until
// its context is cancelled.
func TestSupervisorHealthLoop_HealthyNeverRestarts(t *testing.T) {
	orig := healthPollInterval
	healthPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { healthPollInterval = orig })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	port, _ := strconv.Atoi(srv.URL[strings.LastIndex(srv.URL, ":")+1:])

	var mu sync.Mutex
	var events []string
	s := NewSupervisor(nil, nil, nil, nil, func(name string, _ Spec) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})

	inst := newFakeInstance(nil)
	spec := Spec{DeployID: "health-ok", HealthPort: port, HealthPath: "/health"}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go s.healthLoop(ctx, spec, inst, done)

	time.Sleep(60 * time.Millisecond) // ~12 poll cycles, all healthy
	cancel()
	<-done

	if inst.stopped {
		t.Error("healthy instance must not be stopped by the health loop")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(events) != 0 {
		t.Errorf("healthy instance must emit no health events, got %v", events)
	}
}
