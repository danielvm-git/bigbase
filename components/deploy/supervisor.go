package deploy

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

// Restart policy constants — the in-process equivalent of Docker
// `restart: unless-stopped` (ADR 0004). Named constants, not magic numbers (G25).
const (
	backoffBase   = 1 * time.Second
	backoffFactor = 2.0
	backoffCap    = 60 * time.Second

	crashLoopBurst  = 5
	crashLoopWindow = 60 * time.Second

	// healthPollInterval is how often the Supervisor polls the deployment's
	// health endpoint during runtime.
	healthPollInterval = 10 * time.Second

	// healthFailureThreshold is the number of consecutive failures before
	// the Supervisor forces a restart.
	healthFailureThreshold = 3
)

// Supervisor owns the fleet of running deployments: restart policy (backoff +
// crash-loop detection), resume-on-boot, and guaranteed proxy host
// (re)registration. It implements the in-process equivalent of Docker
// `restart: unless-stopped` (ADR 0004).
type Supervisor struct {
	runner   Runner
	clock    Clock
	registry DeploymentHostRegistry
	onFailed func(deployID string)
	onEvent  func(name string, spec Spec)

	mu        sync.Mutex
	stopping  map[string]bool     // deployID → true when Stop() has been called
	instances map[string]Instance // deployID → currently live Instance
}

// NewSupervisor constructs a Supervisor with injected dependencies.
// onFailed is called when a crash-loop trips (caller updates DB status to "failed").
// onEvent is called for observability events (e.g. "deploy.crash_looped").
func NewSupervisor(r Runner, c Clock, reg DeploymentHostRegistry, onFailed func(string), onEvent func(string, Spec)) *Supervisor {
	return &Supervisor{
		runner:   r,
		clock:    c,
		registry: reg,
		onFailed: onFailed,
		onEvent:  onEvent,
		stopping:  make(map[string]bool),
		instances: make(map[string]Instance),
	}
}

// Run spawns and supervises one Instance for spec, restarting on crash until a
// crash-loop is detected or Stop is called. Blocks until the Instance exits
// permanently (crash-loop or intentional Stop). Safe to call from a goroutine.
func (s *Supervisor) Run(ctx context.Context, spec Spec) {
	var history []time.Time

	for attempt := 0; ; attempt++ {
		if s.isStopping(spec.DeployID) {
			return
		}

		inst, err := s.runner.Spawn(ctx, spec)
		if err != nil {
			// Spawn failure counts as a crash for backoff purposes.
			history = append(history, s.clock.Now())
			if isCrashLooping(history, s.clock.Now()) {
				s.tripCrashLoop(spec)
				return
			}
			s.clock.Sleep(nextBackoff(attempt, rand.Float64()))
			continue
		}

		if s.registry != nil && spec.Host != "" {
			_ = s.registry.RegisterDeploymentHost(spec.Host, spec.Port, spec.DeployID, nil, nil)
		}
		s.setInstance(spec.DeployID, inst)

		// Start health polling if configured.
		healthCtx, healthCancel := context.WithCancel(ctx)
		healthDone := make(chan struct{})
		if spec.HealthPort > 0 {
			go s.healthLoop(healthCtx, spec, inst, healthDone)
		}

		waitErr := inst.Wait()
		healthCancel()
		if spec.HealthPort > 0 {
			<-healthDone // wait for health goroutine to exit
		}
		s.setInstance(spec.DeployID, nil)
		_ = waitErr // any exit without Stop() is treated as a crash

		if s.isStopping(spec.DeployID) {
			return
		}
		history = append(history, s.clock.Now())
		if isCrashLooping(history, s.clock.Now()) {
			s.tripCrashLoop(spec)
			return
		}
		s.clock.Sleep(nextBackoff(attempt, rand.Float64()))
	}
}

// Stop signals that the deployment is being intentionally stopped (not a crash)
// and immediately stops the live Instance if one is running.
func (s *Supervisor) Stop(deployID string) {
	s.mu.Lock()
	s.stopping[deployID] = true
	inst := s.instances[deployID]
	s.mu.Unlock()
	if inst != nil {
		_ = inst.Stop(5 * time.Second)
	}
}

func (s *Supervisor) isStopping(deployID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopping[deployID]
}

func (s *Supervisor) setInstance(deployID string, inst Instance) {
	s.mu.Lock()
	if inst == nil {
		delete(s.instances, deployID)
	} else {
		s.instances[deployID] = inst
	}
	s.mu.Unlock()
}

func (s *Supervisor) tripCrashLoop(spec Spec) {
	if s.registry != nil && spec.Host != "" {
		s.registry.UnregisterDeploymentHost(spec.Host)
	}
	if s.onFailed != nil {
		s.onFailed(spec.DeployID)
	}
	if s.onEvent != nil {
		s.onEvent("deploy.crash_looped", spec)
	}
}

// healthLoop polls the deployment's health endpoint at a fixed interval.
// After healthFailureThreshold consecutive failures, it stops the instance,
// which causes the Supervisor's restart loop to kick in.
func (s *Supervisor) healthLoop(ctx context.Context, spec Spec, inst Instance, done chan<- struct{}) {
	defer close(done)

	path := spec.HealthPath
	if path == "" {
		path = "/health"
	}
	url := fmt.Sprintf("http://localhost:%d%s", spec.HealthPort, path)
	client := &http.Client{Timeout: 5 * time.Second}

	failures := 0
	ticker := time.NewTicker(healthPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resp, err := client.Get(url)
			if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 400 {
				failures++
				if failures >= healthFailureThreshold {
					if s.onEvent != nil {
						s.onEvent("deploy.health_failed", spec)
					}
					_ = inst.Stop(5 * time.Second)
					return
				}
			} else {
				failures = 0
			}
			if resp != nil {
				_ = resp.Body.Close()
			}
		}
	}
}

// nextBackoff returns the delay before restart `attempt` (0-indexed) using full
// jitter: a value in [0, min(cap, base*factor^attempt)]. The jitter fraction
// `frac` (in [0,1]) is injected by the caller so this function stays pure and
// deterministic; production passes rand.Float64().
//
// Full jitter is load-bearing, not cosmetic: resume re-spawns the whole fleet at
// once, so un-jittered backoff would crash-and-retry in lockstep (thundering herd
// on the VPS). Jitter makes that failure mode unreachable by construction.
func nextBackoff(attempt int, frac float64) time.Duration {
	ceiling := float64(backoffBase) * math.Pow(backoffFactor, float64(attempt))
	if ceiling > float64(backoffCap) {
		ceiling = float64(backoffCap)
	}
	return time.Duration(frac * ceiling)
}

// isCrashLooping reports whether at least crashLoopBurst restarts fall within the
// trailing crashLoopWindow ending at now. Mirrors systemd StartLimitBurst /
// StartLimitIntervalSec (consistent with the systemd Isolator). Named predicate (G28):
// on trip the Supervisor marks the deployment failed, de-registers its host, and
// emits deploy.crash_looped.
func isCrashLooping(history []time.Time, now time.Time) bool {
	cutoff := now.Add(-crashLoopWindow)
	count := 0
	for _, t := range history {
		if !t.Before(cutoff) && !t.After(now) {
			count++
		}
	}
	return count >= crashLoopBurst
}
