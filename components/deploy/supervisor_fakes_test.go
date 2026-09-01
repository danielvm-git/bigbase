package deploy

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// FakeInstance is a single-use scripted Instance for tests.
// Each call to Wait() returns the next error from errSeq;
// once exhausted it blocks until Stop is called.
type FakeInstance struct {
	mu      sync.Mutex
	waitErr error
	stopCh  chan struct{}
	stopped bool
}

func newFakeInstance(waitErr error) *FakeInstance {
	return &FakeInstance{waitErr: waitErr, stopCh: make(chan struct{})}
}

func (f *FakeInstance) Wait() error {
	if f.waitErr != nil {
		return f.waitErr
	}
	// Block until Stop is called (simulates a healthy long-running process).
	<-f.stopCh
	return nil
}

func (f *FakeInstance) Stop(grace time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.stopped {
		f.stopped = true
		close(f.stopCh)
	}
	return nil
}

func (f *FakeInstance) Health(ctx context.Context) error { return nil }

// FakeRunner pops scripted Instances from a queue on each Spawn call.
// When the queue is empty, Spawn returns ErrRunnerExhausted (test sentinel).
var ErrRunnerExhausted = errors.New("FakeRunner: no more scripted instances")

type FakeRunner struct {
	mu    sync.Mutex
	queue []*FakeInstance
	calls int // how many times Spawn was called
}

func (r *FakeRunner) Spawn(_ context.Context, _ Spec) (Instance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if len(r.queue) == 0 {
		return nil, ErrRunnerExhausted
	}
	inst := r.queue[0]
	r.queue = r.queue[1:]
	return inst, nil
}

func (r *FakeRunner) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// FakeClock records durations passed to Sleep so tests can assert backoff behaviour.
type FakeClock struct {
	now    time.Time
	Sleeps []time.Duration
}

func (c *FakeClock) Now() time.Time { return c.now }
func (c *FakeClock) Sleep(d time.Duration) {
	c.Sleeps = append(c.Sleeps, d)
	c.now = c.now.Add(d)
}

// TestFakeRunnerSpawnsScriptedInstances drives the fake Runner/Instance contract:
// each Spawn pops the next scripted instance, Wait returns its error, and
// the call count matches the number of items consumed.
func TestFakeRunnerSpawnsScriptedInstances(t *testing.T) {
	crash := errors.New("process exited: signal killed")

	runner := &FakeRunner{
		queue: []*FakeInstance{
			newFakeInstance(crash), // attempt 0 → crash
			newFakeInstance(crash), // attempt 1 → crash
			newFakeInstance(nil),   // attempt 2 → ok (blocks until Stop)
		},
	}

	ctx := context.Background()
	spec := Spec{DeployID: "dep-1", Host: "app.bigbase.click", Port: 10001}

	// First two instances crash immediately.
	for i := 0; i < 2; i++ {
		inst, err := runner.Spawn(ctx, spec)
		if err != nil {
			t.Fatalf("Spawn[%d]: unexpected error %v", i, err)
		}
		if got := inst.Wait(); !errors.Is(got, crash) {
			t.Errorf("Wait[%d] = %v, want crash sentinel", i, got)
		}
	}

	// Third instance is healthy; Stop causes Wait to return nil.
	inst, err := runner.Spawn(ctx, spec)
	if err != nil {
		t.Fatalf("Spawn[2]: unexpected error %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- inst.Wait() }()
	if err := inst.Stop(0); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if got := <-done; got != nil {
		t.Errorf("Wait after Stop = %v, want nil", got)
	}

	// Exhausted queue returns sentinel.
	if _, err := runner.Spawn(ctx, spec); !errors.Is(err, ErrRunnerExhausted) {
		t.Errorf("Spawn on empty queue = %v, want ErrRunnerExhausted", err)
	}

	if runner.Calls() != 4 {
		t.Errorf("Spawn call count = %d, want 4", runner.Calls())
	}
}
