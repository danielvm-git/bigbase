package deploy

import (
	"context"
	"net/http"
	"os/exec"
	"time"
)

// Spec is the immutable description of what to run. Built once from the
// deployment row + manifest; handed to Runner.Spawn on every attempt.
// It is the only state that survives a restart — restart history lives
// in the Supervisor, not in the Instance.
type Spec struct {
	DeployID string
	Host     string // public hostname; "" for localhost-only
	Port     int
	Dir      string // serve/build output directory
	AppType  AppType
	Env      []string // resolved env vars (PORT=N + manifest vars)
	StartCmd []string // resolved start command; nil for static apps
	// WritableDir is a deployment-scoped persistent directory for runtime
	// data (logs, SQLite DBs, uploads). Created during build and passed
	// as WRITABLE_DIR env var.
	WritableDir string
	// HealthPort is the port to poll for health checks. 0 disables polling.
	HealthPort int
	// HealthPath is the HTTP path to poll (default: /health).
	HealthPath string
	// BackgroundProcesses is a list of shell commands to run as background
	// subprocesses alongside the main deployment process.
	BackgroundProcesses []string
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

// processInstance wraps exec.Cmd. Single-use: Wait calls cmd.Wait; Stop kills
// the process. Production log streaming is driven by the deployRunner goroutines.
type processInstance struct {
	cmd    *exec.Cmd
	bgCmds []*exec.Cmd
}

func (p *processInstance) Wait() error { return p.cmd.Wait() }

func (p *processInstance) Stop(_ time.Duration) error {
	for _, bg := range p.bgCmds {
		if bg.Process != nil {
			killProcessGroup(bg.Process.Pid)
		}
	}
	if p.cmd.Process == nil {
		return nil
	}
	killProcessGroup(p.cmd.Process.Pid)
	return nil
}

func (p *processInstance) Health(_ context.Context) error { return nil }

// staticInstance wraps http.Server. Wait calls ListenAndServe (blocks until
// the server closes); Stop calls Shutdown with the given grace period.
type staticInstance struct {
	srv *http.Server
}

func (s *staticInstance) Wait() error {
	err := s.srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *staticInstance) Stop(grace time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

func (s *staticInstance) Health(_ context.Context) error { return nil }
