package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// PyProjectTOML represents the relevant sections of a pyproject.toml for
// BigBase detection.
type PyProjectTOML struct {
	Project struct {
		Name         string            `toml:"name"`
		Dependencies []string          `toml:"dependencies"`
		Scripts      map[string]string `toml:"scripts"`
	} `toml:"project"`
	Tool struct {
		UV any `toml:"uv"`
	} `toml:"tool"`
	// Tools holds the [tools] section (e.g. [tools.system_deps]).
	Tools struct {
		SystemDeps struct {
			Deps []string `toml:"deps"`
		} `toml:"system_deps"`
	} `toml:"tools"`
}

// HasPyProjectTOML checks whether a pyproject.toml exists and is parseable
// as a Python project (must have [project] section).
func HasPyProjectTOML(buildDir string) bool {
	path := filepath.Join(buildDir, "pyproject.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var pp PyProjectTOML
	if err := toml.Unmarshal(data, &pp); err != nil {
		return false
	}
	return pp.Project.Name != "" || len(pp.Project.Scripts) > 0 || pp.Tool.UV != nil
}

// ParsePyProjectTOML reads and parses pyproject.toml from the build directory.
// Returns nil if the file doesn't exist or is unparseable.
func ParsePyProjectTOML(buildDir string) *PyProjectTOML {
	path := filepath.Join(buildDir, "pyproject.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var pp PyProjectTOML
	if err := toml.Unmarshal(data, &pp); err != nil {
		return nil
	}
	return &pp
}

// HasUvicorn checks whether uvicorn is listed in the project dependencies.
func (pp *PyProjectTOML) HasUvicorn() bool {
	for _, dep := range pp.Project.Dependencies {
		if isUvicornDep(dep) {
			return true
		}
	}
	return false
}

// EntryPoint returns the app module and variable from [project.scripts],
// e.g. "myapp.main:app" → ("myapp.main", "app"). Returns empty strings
// if no scripts section is defined.
func (pp *PyProjectTOML) EntryPoint() (module, appVar string) {
	if len(pp.Project.Scripts) == 0 {
		return "", ""
	}
	for _, val := range pp.Project.Scripts {
		return splitEntryPoint(val)
	}
	return "", ""
}

// isUvicornDep checks whether a dependency string (e.g. "uvicorn>=0.30")
// refers to the uvicorn package.
func isUvicornDep(dep string) bool {
	name := dep
	for i, c := range dep {
		if c == '>' || c == '<' || c == '=' || c == '~' || c == '!' || c == '[' || c == ';' {
			name = dep[:i]
			break
		}
	}
	return name == "uvicorn"
}

// splitEntryPoint splits "module.path:app_var" into ("module.path", "app_var").
func splitEntryPoint(entry string) (module, appVar string) {
	for i := len(entry) - 1; i >= 0; i-- {
		if entry[i] == ':' {
			return entry[:i], entry[i+1:]
		}
	}
	return entry, "app"
}

// SystemDeps returns the system dependencies declared in [tools.system_deps],
// filtered to only allowlisted packages.
func (pp *PyProjectTOML) SystemDeps() []string {
	var filtered []string
	for _, dep := range pp.Tools.SystemDeps.Deps {
		if allowedSystemDep(dep) {
			filtered = append(filtered, dep)
		}
	}
	return filtered
}

// allowedSystemDeps is the allowlist of apt packages that can be installed
// in the build sandbox. Expanding this list is a deliberate security decision.
var allowedSystemDeps = map[string]bool{
	"git":             true,
	"curl":            true,
	"ssh":             true,
	"libpq-dev":       true,
	"libssl-dev":      true,
	"build-essential": true,
	"ffmpeg":          true,
	"imagemagick":     true,
	"wget":            true,
	"unzip":           true,
	"ca-certificates": true,
}

func allowedSystemDep(name string) bool {
	return allowedSystemDeps[name]
}

// resolvePythonBin returns the preferred Python binary available on PATH.
func resolvePythonBin() string {
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}
	return "python"
}

// pythonStartCommand returns the exec.Cmd for starting a Python deployment.
// Uses uv run when pyproject.toml is present and uv is available; falls back
// to python3 when uv is not installed.
func pythonStartCommand(ctx context.Context, buildDir string, port int) *exec.Cmd {
	pp := ParsePyProjectTOML(buildDir)
	if pp != nil {
		_, uvErr := exec.LookPath("uv")
		useUV := uvErr == nil
		if pp.HasUvicorn() {
			module, appVar := pp.EntryPoint()
			if module == "" {
				module = "app"
			}
			portStr := fmt.Sprintf("%d", port)
			if useUV {
				cmd := exec.CommandContext(ctx, "uv", "run", "uvicorn",
					module+":"+appVar, "--host", "0.0.0.0", "--port", portStr)
				cmd.Dir = buildDir
				return cmd
			}
			// Fallback: python3 -m uvicorn
			pythonBin := resolvePythonBin()
			cmd := exec.CommandContext(ctx, pythonBin, "-m", "uvicorn",
				module+":"+appVar, "--host", "0.0.0.0", "--port", portStr)
			cmd.Dir = buildDir
			return cmd
		}
		// Fallback: run the app module directly with uv.
		scriptModule, _ := pp.EntryPoint()
		if scriptModule != "" {
			if useUV {
				cmd := exec.CommandContext(ctx, "uv", "run", "python", "-m", scriptModule)
				cmd.Dir = buildDir
				return cmd
			}
			pythonBin := resolvePythonBin()
			cmd := exec.CommandContext(ctx, pythonBin, "-m", scriptModule)
			cmd.Dir = buildDir
			return cmd
		}
		if useUV {
			cmd := exec.CommandContext(ctx, "uv", "run", "python", "app.py")
			cmd.Dir = buildDir
			return cmd
		}
		pythonBin := resolvePythonBin()
		cmd := exec.CommandContext(ctx, pythonBin, "app.py")
		cmd.Dir = buildDir
		return cmd
	}
	// Legacy Python: no pyproject.toml, use python3 with app.py.
	pythonBin := resolvePythonBin()
	cmd := exec.CommandContext(ctx, pythonBin, "app.py")
	cmd.Dir = buildDir
	return cmd
}
