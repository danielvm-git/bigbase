package deploy

import (
	"context"
	"time"
)

// Spec is the immutable description of what to run. Built once from the
// deployment row + manifest; handed to Runner.Spawn on every attempt.
// It is the only state that survives a restart — restart history lives
// in the Supervisor, not in the Instance.
type Spec struct {
	DeployID string
	Host     string   // public hostname; "" for localhost-only
	Port     int
	Dir      string   // serve/build output directory
	AppType  AppType
	Env      []string // resolved env vars (PORT=N + manifest vars)
	StartCmd []string // resolved start command; nil for static apps
}

// Instance is a single-use live run of one deployment — a subprocess or an
// in-process static server. Wait blocks until the run ends; the Instance is
// spent after Wait returns. Restart is achieved by spawning a fresh Instance
// via Runner.Spawn, not by re-calling anything on a dead Instance.
type Instance interface {
	// Wait blocks until the instance exits (process death or server stop).
	Wait() error
	// Stop signals a graceful shutdown within the given grace period.
	Stop(grace time.Duration) error
	// Health returns nil when the instance is ready to serve traffic.
	// Stub this epic; e43 makes it real.
	Health(ctx context.Context) error
}

// Runner spawns Instances. The production adapter (added in e53s03) contains a
// process arm and a static arm; FakeRunner (supervisor_fakes_test.go) returns
// scripted Instances so supervision logic is testable without real processes.
type Runner interface {
	Spawn(ctx context.Context, spec Spec) (Instance, error)
}

// Clock abstracts time for the Supervisor's backoff and crash-loop window.
// FakeClock (supervisor_fakes_test.go) records Sleep durations for assertions.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}
