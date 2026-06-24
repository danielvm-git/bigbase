package deploy

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// wallClock is the production Clock implementation.
type wallClock struct{}

func (c *wallClock) Now() time.Time        { return time.Now() }
func (c *wallClock) Sleep(d time.Duration) { time.Sleep(d) }

// deployRunner spawns Instances from Specs for the production deploy component.
// It is used by resumeCandidates (resumed deployments). New deployments go
// through startApp/serveStatic directly for now (wired in a future story).
type deployRunner struct {
	db DBer
}

func (r *deployRunner) Spawn(ctx context.Context, spec Spec) (Instance, error) {
	if spec.AppType == AppStatic {
		return r.spawnStatic(spec)
	}
	return r.spawnProcess(ctx, spec)
}

func (r *deployRunner) spawnStatic(spec Spec) (Instance, error) {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(spec.Dir)))
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", spec.Port),
		Handler: mux,
	}
	return &staticInstance{srv: srv}, nil
}

func (r *deployRunner) spawnProcess(ctx context.Context, spec Spec) (Instance, error) {
	cmd := r.buildCmd(ctx, spec)
	if cmd == nil {
		return nil, fmt.Errorf("no start command for app type %s", spec.AppType)
	}
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start process: %w", err)
	}
	if pid := cmd.Process.Pid; pid > 0 {
		_, _ = r.db.ExecContext(ctx,
			"UPDATE deployments SET pid = ? WHERE id = ?", pid, spec.DeployID)
	}
	return &processInstance{cmd: cmd}, nil
}

func (r *deployRunner) buildCmd(ctx context.Context, spec Spec) *exec.Cmd {
	if len(spec.StartCmd) > 0 {
		return exec.CommandContext(ctx, spec.StartCmd[0], spec.StartCmd[1:]...)
	}
	switch spec.AppType {
	case AppGo:
		return exec.CommandContext(ctx, filepath.Join(spec.Dir, "app"))
	case AppPython:
		python := "python3"
		if _, err := exec.LookPath(python); err != nil {
			python = "python"
		}
		return exec.CommandContext(ctx, python, "app.py")
	case AppNode:
		startCmd := GetStartCommand(spec.Dir)
		parts := strings.Fields(startCmd)
		if len(parts) == 0 {
			parts = []string{"node", "index.js"}
		}
		args := append([]string{"exec", "--"}, parts...)
		return exec.CommandContext(ctx, "npm", args...)
	}
	return nil
}
