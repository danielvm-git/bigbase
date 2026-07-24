package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectNodePackageManager(t *testing.T) {
	t.Run("defaults to npm", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0644); err != nil {
			t.Fatal(err)
		}
		if got := DetectNodePackageManager(dir); got != "npm" {
			t.Fatalf("got %q, want npm", got)
		}
	})

	t.Run("packageManager field", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"packageManager":"pnpm@9.0.0"}`), 0644); err != nil {
			t.Fatal(err)
		}
		if got := DetectNodePackageManager(dir); got != "pnpm" {
			t.Fatalf("got %q, want pnpm", got)
		}
	})

	lockfileCases := []struct {
		file string
		want string
	}{
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
		{"bun.lockb", "bun"},
		{"bun.lock", "bun"},
	}
	for _, tc := range lockfileCases {
		t.Run("lockfile "+tc.file, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, tc.file), []byte("lock"), 0644); err != nil {
				t.Fatal(err)
			}
			if got := DetectNodePackageManager(dir); got != tc.want {
				t.Fatalf("got %q, want %s", got, tc.want)
			}
		})
	}

	t.Run("packageManager overrides lockfile", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"packageManager":"yarn@4.0.0"}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lock"), 0644); err != nil {
			t.Fatal(err)
		}
		if got := DetectNodePackageManager(dir); got != "yarn" {
			t.Fatalf("got %q, want yarn", got)
		}
	})
}

func TestNodeInstallCommand_UsesPnpmWhenLockfilePresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: 6"), 0644); err != nil {
		t.Fatal(err)
	}
	pm := DetectNodePackageManager(dir)
	if pm != "pnpm" {
		t.Fatalf("detected %q, want pnpm", pm)
	}
	name, args := NodeInstallCommand(pm)
	if name != "pnpm" || len(args) != 1 || args[0] != "install" {
		t.Fatalf("got %q %v, want pnpm [install]", name, args)
	}
}

func TestNodeBuildCommand(t *testing.T) {
	name, args := NodeBuildCommand("pnpm")
	if name != "pnpm" || len(args) != 2 || args[0] != "run" || args[1] != "build" {
		t.Fatalf("got %q %v", name, args)
	}
}

func TestEnsureNodePackageManager_MissingBun(t *testing.T) {
	if _, err := exec.LookPath("bun"); err == nil {
		t.Skip("bun is installed")
	}
	err := ensureNodePackageManager("bun")
	if err == nil {
		t.Fatal("expected error when bun missing")
	}
	if !strings.Contains(err.Error(), "bun") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureNodePackageManager_NpmPresent(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not installed")
	}
	if err := ensureNodePackageManager("npm"); err != nil {
		t.Fatalf("npm should be present: %v", err)
	}
}
