# ADR 004 — No containers (Docker / Kubernetes): in-process Go Supervisor with systemd isolation

## Status

Accepted

## Principle

**BigBase will never use Docker, Kubernetes, or any container runtime / orchestrator.**
This is a permanent, non-negotiable constraint, not a current-phase preference. The
product's entire reason to exist is to be a *single lightweight Go binary* on a modest
VPS — containers and orchestration are exactly the weight (and operational surface) it
rejects. Any future capability that Docker/K8s would normally provide (supervision,
isolation, scaling, scheduling, service discovery, rolling deploys) must be solved with
in-process Go plus the host's existing init system (systemd), never by adopting a
container runtime or orchestrator.

## Context

Appwrite runs deployed apps and function runtimes as Docker containers (51 services
in `docker-compose.yml` for 1.9.x). Docker silently provides four things BigBase needs:

1. **Restart on crash / reboot** — `restart: unless-stopped`.
2. **Health checks** — `healthcheck:` gates traffic on readiness.
3. **Resource isolation** — cgroups cap memory/CPU per container and OOM-kill offenders.
4. **Log capture** — `docker logs`.

BigBase was created **specifically to avoid Docker's weight** on a single Contabo VPS.
It runs each deployment as a child process via `exec.Cmd` (process apps) or an in-process
`http.Server` (static). It already has port allocation, a reverse proxy (`deployHosts`),
and log capture — but it has **no process supervision and no resource isolation**.

A 2026-06-23 audit of the live fleet found 5 of 6 sites at `*.bigbase.click` broken,
all from these missing mechanisms:
- `docklocker` → 502: subprocess crashed, never restarted.
- `docklock` → served the BigBase landing page: host not in `deployHosts`.
- `add-tutorial-requests-site`, `big-dock-locker-site`, `cleaninstallguide` → TLS
  `internal error`: host not registered, so Caddy on-demand TLS refused a cert.

(See `specs/bugs/BUG-2026-06-21T153000-process-apps-not-resumed-after-restart.md`.)

## Decision

**BigBase will never use Docker or Kubernetes.** Instead, port the subset of Docker's
design that the deploy path needs into a lightweight, in-process Go module — the
**Supervisor**. (Kubernetes — and any other orchestrator — is rejected for the same
reason as Docker: it reintroduces the operational and resource weight BigBase exists to
avoid. Horizontal scaling, if ever needed, will be solved without an orchestrator.)

- **Supervision stays in-process (Go).** The Supervisor owns a fleet of **Instances**
  (one per running deployment), each spawned through a **Runner** seam (issue #40). It
  provides:
  - **Restart policy** — `unless-stopped` equivalent: crash → exponential backoff →
    respawn, with max-retries and crash-loop detection before marking `failed`.
  - **Health probe** — `Health(ctx)` per Instance (epic e43); a host is registered in
    the proxy **only while healthy**, and de-registered on failure.
  - **Resume on boot** — rehydrate Instances from the `deployments` table on startup.
  - **Guaranteed host (re)registration** — a running Instance can never be missing from
    `deployHosts`. This closes the TLS / landing-page failure class.
- **Resource isolation is delegated to systemd** (not reimplemented). Each app is spawned
  inside a **systemd transient scope** (`systemd-run --scope --slice=bigbase.slice`),
  so systemd's cgroup layer enforces `MemoryMax`, `CPUQuota`, and `TasksMax` and performs
  OOM-kills. This gives Docker-grade isolation with no cgroup bookkeeping code in BigBase.
- **Isolation is a seam (`Isolator`)**, so the strategy is swappable and degrades to a
  **no-op adapter on macOS** (developer machines) where systemd is absent.
- **Connection draining lives at the proxy/router seam** (`DeploymentHostRegistry`), not
  on the Instance, because all deployment traffic is reverse-proxied — the proxy is the
  only layer that can count in-flight connections (epic e45).

### Division of responsibility

| Concern | Owner |
|---------|-------|
| restart, health, resume, host (re)registration, drain orchestration | BigBase Supervisor (Go) |
| memory/CPU/task caps, OOM-kill | systemd scope (cgroups) |
| connection counting + drain | proxy / `DeploymentHostRegistry` |

Supervision is **not** delegated to systemd (no `Restart=always` units), because BigBase
must re-register the proxy host route each time an app restarts — that coupling has to
live in BigBase.

### Restart policy

The in-process equivalent of Docker `restart: unless-stopped`, designed against the
bigpowers Defensive Code triad (retry-with-backoff · timeout · graceful degradation) and
Ousterhout's *Define Errors Out of Existence*. All thresholds are named constants exposed
as flags (no magic numbers, G25); the three decisions are pure, clock-injected functions
tested through the public Supervisor interface with a `FakeRunner` + `FakeClock`.

1. **Backoff — exponential with full jitter, capped.** `nextBackoff(attempt) =
   jitter(min(cap, base · factor^attempt))`. Defaults: base `1s`, factor `2`, cap `60s`.
   Full jitter is load-bearing, not cosmetic: resume re-spawns the whole fleet at once, so
   synchronized backoff would crash-and-retry in lockstep (thundering herd on the VPS).
   Jitter makes that failure mode unreachable by construction.

2. **Crash-loop ceiling — N restarts within window T → `failed`.** Mirrors systemd's
   `StartLimitBurst` / `StartLimitIntervalSec` (consistent with the systemd `Isolator`).
   Defaults: burst `5`, window `60s`; `isCrashLooping(history, now)` is a named predicate
   (G28). On trip: (a) status → `failed`; (b) **de-register the host from `deployHosts`**;
   (c) emit a structured `deploy.crash_looped` event with a remediation hint (Akita
   observability).

3. **Counter reset — sustained health, never optimism.** The restart counter resets to 0
   only after `Health()` has passed continuously for `restartResetThreshold` (default
   `60s`). The rolling window also ages old failures out. A deployment is registered in the
   proxy **only after its first successful health probe**, so flapping never exposes a
   half-up app.

**Health gates registration.** Because a host is registered only while healthy and
de-registered on `failed`, the live failure classes found in the 2026-06-23 audit —
registered-but-dead (502), registered-stale (wrong page / landing fallthrough), and
unregistered (on-demand-TLS refuses a cert) — become *unreachable states*, not merely
recoverable ones. This is the primary correctness argument for the policy.

## Configuration

```
--deploy-isolation          systemd | none   (default: systemd on linux, none on darwin)
--deploy-mem-max            per-app MemoryMax (e.g. 512M)
--deploy-cpu-quota          per-app CPUQuota  (e.g. 50%)
--deploy-restart-max        crash-loop burst ceiling     (default 5)
--deploy-restart-window     crash-loop window            (default 60s)
--deploy-restart-backoff    base backoff between restarts (default 1s; factor 2, cap 60s)
--deploy-restart-reset      sustained-health reset threshold (default 60s)
```

## Consequences

- BigBase stays a single Go binary; no Docker daemon, no per-app containers.
- On Linux, the BigBase process must be able to call `systemd-run` (run under systemd
  with an owned slice, or with appropriate delegation).
- On macOS dev, isolation is a no-op; resource caps are not enforced locally.
- The Supervisor is testable through a `FakeRunner` / `FakeIsolator`, so restart, health,
  and resume logic run in-memory without real processes or systemd.
- This ADR supersedes the implicit "raw `exec.Cmd`, no supervision" status quo and is the
  architectural home for epics e43 (health), e44 (rollback), e45 (drain), e51 (preview).
