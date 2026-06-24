package deploy

import (
	"context"
	"errors"
	"testing"
	"time"
)

// supervisorRegistry is a minimal DeploymentHostRegistry spy for the Supervisor tests.
type supervisorRegistry struct {
	registered   map[string]int // host → port
	unregistered []string       // hosts that were unregistered
}

func (r *supervisorRegistry) RegisterDeploymentHost(host string, port int, _ string, _ []string, _ map[string]string) error {
	if r.registered == nil {
		r.registered = make(map[string]int)
	}
	r.registered[host] = port
	return nil
}

func (r *supervisorRegistry) UnregisterDeploymentHost(host string) {
	r.unregistered = append(r.unregistered, host)
	delete(r.registered, host)
}

// supervisorHarness wires a Supervisor under test with injectable fakes.
type supervisorHarness struct {
	runner   *FakeRunner
	clock    *FakeClock
	registry *supervisorRegistry
	sup      *Supervisor
	failed   []string // deploy IDs passed to the onFailed callback
	events   []string // event names passed to the onEvent callback
}

func newHarness(instances ...*FakeInstance) *supervisorHarness {
	r := &FakeRunner{queue: instances}
	c := &FakeClock{now: time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)}
	reg := &supervisorRegistry{}
	h := &supervisorHarness{runner: r, clock: c, registry: reg}
	h.sup = NewSupervisor(r, c, reg,
		func(id string) { h.failed = append(h.failed, id) },
		func(name string, _ Spec) { h.events = append(h.events, name) },
	)
	return h
}

// TestSupervisorRespawnsAfterBackoff: crash → Supervisor sleeps a backoff interval
// → re-spawns a new Instance (not the dead one). Asserts spawn count and sleep count.
func TestSupervisorRespawnsAfterBackoff(t *testing.T) {
	crash := errors.New("exit status 1")
	h := newHarness(
		newFakeInstance(crash), // attempt 0 crashes
		newFakeInstance(nil),   // attempt 1 runs until Stop
	)
	spec := Spec{DeployID: "dep-test", Host: "app.bigbase.click", Port: 10001}

	done := make(chan struct{})
	go func() {
		h.sup.Run(context.Background(), spec)
		close(done)
	}()

	// Let the loop run: crash → sleep → respawn → second instance healthy (blocks).
	// Signal that the second instance should stop.
	eventually(t, func() bool { return h.runner.calls >= 2 })
	h.sup.Stop(spec.DeployID)
	<-done

	if h.runner.calls != 2 {
		t.Errorf("Spawn calls = %d, want 2", h.runner.calls)
	}
	if len(h.clock.Sleeps) == 0 {
		t.Error("Supervisor did not sleep between crash and respawn")
	}
	// First sleep should be within [0, backoffBase*factor^0] = [0, 1s]
	if h.clock.Sleeps[0] > backoffBase {
		t.Errorf("first backoff = %v, want ≤ %v", h.clock.Sleeps[0], backoffBase)
	}
}

// TestSupervisorCrashLoopDeregistersHost: after crashLoopBurst crashes within
// crashLoopWindow the Supervisor must mark the deployment failed, de-register
// the proxy host, and emit deploy.crash_looped.
func TestSupervisorCrashLoopDeregistersHost(t *testing.T) {
	crash := errors.New("exit status 1")
	instances := make([]*FakeInstance, crashLoopBurst+2)
	for i := range instances {
		instances[i] = newFakeInstance(crash)
	}
	h := newHarness(instances...)

	spec := Spec{DeployID: "dep-loop", Host: "loop.bigbase.click", Port: 10002}

	h.sup.Run(context.Background(), spec)

	// Host must be de-registered after crash-loop.
	if _, ok := h.registry.registered[spec.Host]; ok {
		t.Error("host still registered after crash-loop; want UnregisterDeploymentHost called")
	}
	// Event must be emitted.
	if len(h.events) == 0 || h.events[len(h.events)-1] != "deploy.crash_looped" {
		t.Errorf("events = %v; want last event deploy.crash_looped", h.events)
	}
	// onFailed must be called with the deploy ID.
	if len(h.failed) == 0 || h.failed[0] != spec.DeployID {
		t.Errorf("failed = %v; want [%s]", h.failed, spec.DeployID)
	}
}

// TestSupervisorNoRespawnAfterStop: an intentional Stop must not trigger a respawn.
func TestSupervisorNoRespawnAfterStop(t *testing.T) {
	h := newHarness(
		newFakeInstance(nil), // healthy; blocks until Stop
	)
	spec := Spec{DeployID: "dep-stop", Host: "stop.bigbase.click", Port: 10003}

	done := make(chan struct{})
	go func() {
		h.sup.Run(context.Background(), spec)
		close(done)
	}()

	eventually(t, func() bool { return h.runner.calls >= 1 })
	h.sup.Stop(spec.DeployID)
	<-done

	if h.runner.calls != 1 {
		t.Errorf("Spawn calls after intentional Stop = %d, want 1 (no respawn)", h.runner.calls)
	}
}

// eventually polls cond until it returns true or 2s elapses.
func eventually(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within 2s")
}
