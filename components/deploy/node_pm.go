package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var corepackEnableOnce sync.Once
var corepackEnableErr error

// DetectNodePackageManager returns npm, pnpm, yarn, or bun for a Node project directory.
// Priority: package.json packageManager field, then lockfiles, then npm.
func DetectNodePackageManager(dir string) string {
	if pm := packageManagerFromPackageJSON(dir); pm != "" {
		return pm
	}

	lockfiles := []struct {
		file string
		pm   string
	}{
		{"bun.lockb", "bun"},
		{"bun.lock", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	}
	for _, lf := range lockfiles {
		if fileExists(filepath.Join(dir, lf.file)) {
			return lf.pm
		}
	}
	return "npm"
}

func packageManagerFromPackageJSON(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		PackageManager string `json:"packageManager"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil || pkg.PackageManager == "" {
		return ""
	}
	name := strings.SplitN(pkg.PackageManager, "@", 2)[0]
	switch name {
	case "npm", "pnpm", "yarn", "bun":
		return name
	default:
		return ""
	}
}

// ensureNodePackageManager verifies the package manager binary is on PATH.
// For pnpm and yarn, runs corepack enable once when missing.
// Never falls back to a different package manager (pnpm must not become npm).
func ensureNodePackageManager(pm string) error {
	if _, err := exec.LookPath(pm); err == nil {
		return nil
	}
	if pm == "pnpm" || pm == "yarn" {
		if err := tryCorepackEnable(); err != nil {
			return codedErr("tool_missing",
				fmt.Sprintf("%s not found on PATH", pm),
				fmt.Sprintf("Install %s or run `corepack enable` on the deploy host. Do not substitute npm when the lockfile requires %s.", pm, pm))
		}
		if _, err := exec.LookPath(pm); err == nil {
			return nil
		}
	}
	if pm == "bun" {
		return codedErr("tool_missing", "bun not found on PATH",
			"Install bun on the deploy host, or switch the project to npm/pnpm/yarn with a matching lockfile.")
	}
	return codedErr("tool_missing",
		fmt.Sprintf("%s not found on PATH", pm),
		fmt.Sprintf("Install %s (or enable Corepack for pnpm/yarn) on the deploy host. No silent package-manager fallback.", pm))
}

func tryCorepackEnable() error {
	corepackEnableOnce.Do(func() {
		corepackEnableErr = exec.Command("corepack", "enable").Run()
	})
	return corepackEnableErr
}

// NodeInstallCommand returns the install command for the given package manager.
func NodeInstallCommand(pm string) (string, []string) {
	switch pm {
	case "pnpm":
		return "pnpm", []string{"install"}
	case "yarn":
		return "yarn", []string{"install", "--frozen-lockfile"}
	case "bun":
		return "bun", []string{"install"}
	default:
		return "npm", []string{"install"}
	}
}

// NodeBuildCommand returns the build command for the given package manager.
func NodeBuildCommand(pm string) (string, []string) {
	return pm, []string{"run", "build"}
}

// NodePMRunCommand returns "<pm> run <script>" for manifest defaults.
func NodePMRunCommand(dir, script string) string {
	pm := DetectNodePackageManager(dir)
	return fmt.Sprintf("%s run %s", pm, script)
}

// NodePMStartCommand returns "<pm> start" for manifest defaults when a start script exists.
func NodePMStartCommand(dir string) string {
	pm := DetectNodePackageManager(dir)
	return fmt.Sprintf("%s start", pm)
}

// NodeStartCommand returns the process command to start a Node app.
func NodeStartCommand(buildDir string) (string, []string) {
	pm := DetectNodePackageManager(buildDir)
	startCmd := GetStartCommand(buildDir)
	parts := strings.Fields(startCmd)
	if len(parts) == 0 {
		parts = []string{"node", "index.js"}
	}
	switch pm {
	case "bun":
		return "bun", parts
	case "npm", "pnpm", "yarn":
		return pm, append([]string{"exec", "--"}, parts...)
	default:
		return "npm", append([]string{"exec", "--"}, parts...)
	}
}

// ensureManifestPackageManager checks the first token of a manifest command when it is a known PM.
func ensureManifestPackageManager(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}
	switch parts[0] {
	case "pnpm", "yarn", "bun", "npm":
		return ensureNodePackageManager(parts[0])
	default:
		return nil
	}
}
