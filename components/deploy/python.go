package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// isCLIScriptEntry returns true when the module/appVar pair looks like a
// PEP 621 console_script entry point rather than an ASGI application import.
// CLI scripts typically end in __main__:main or __main__:cli.
func isCLIScriptEntry(module, appVar string) bool {
	// __main__ modules are CLI entry points, not ASGI apps.
	if strings.HasSuffix(module, "__main__") {
		return true
	}
	// Variable names like "main", "cli", "entrypoint", "run" indicate
	// a CLI function, not an ASGI application instance.
	switch appVar {
	case "main", "cli", "entrypoint", "run":
		return true
	}
	return false
}

// parseASGIImport splits a manifest asgi_import value into the ASGI import
// string and extra uvicorn arguments. For example:
//
//	"grimoire.app:create_app --factory" → ("grimoire.app:create_app", ["--factory"])
//	"myapp.main:app"                   → ("myapp.main:app", nil)
//	"app:app"                          → ("app:app", nil)
func parseASGIImport(raw string) (importStr string, extraArgs []string) {
	// Find the first space to split import string from extra args.
	spaceIdx := strings.IndexByte(raw, ' ')
	if spaceIdx < 0 {
		return raw, nil
	}
	return raw[:spaceIdx], strings.Fields(raw[spaceIdx+1:])
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
//
// When manifest is non-nil and specifies an asgi_import, it takes priority
// over auto-detection from pyproject.toml [project.scripts].
func pythonStartCommand(ctx context.Context, buildDir string, port int, manifest *Manifest) *exec.Cmd {
	pp := ParsePyProjectTOML(buildDir)
	if pp != nil {
		_, uvErr := exec.LookPath("uv")
		useUV := uvErr == nil
		if pp.HasUvicorn() {
			module, appVar := pp.EntryPoint()
			if module == "" {
				module = "app"
			}
			if appVar == "" {
				appVar = "app"
			}
			// Heuristic: if the entry point looks like a CLI script
			// (module ends with __main__ or the variable is "main"), it's
			// a PEP 621 console_script, not an ASGI application. Fall back
			// to the universal FastAPI convention of app:app.
			if isCLIScriptEntry(module, appVar) {
				module = "app"
				appVar = "app"
			}

			var extraArgs []string
			// Manifest ASGI import override takes priority over heuristics.
			if manifest != nil && manifest.Start.ASGIImport != "" {
				asgiImport, extra := parseASGIImport(manifest.Start.ASGIImport)
				module, appVar = splitEntryPoint(asgiImport)
				extraArgs = extra
				if module == "" {
					module = "app"
				}
				if appVar == "" {
					appVar = "app"
				}
			}

			portStr := fmt.Sprintf("%d", port)
			baseArgs := []string{module + ":" + appVar}
			baseArgs = append(baseArgs, extraArgs...)
			baseArgs = append(baseArgs, "--host", "0.0.0.0", "--port", portStr)
			if useUV {
				args := append([]string{"run", "uvicorn"}, baseArgs...)
				cmd := exec.CommandContext(ctx, "uv", args...)
				cmd.Dir = buildDir
				return cmd
			}
			// Fallback: python3 -m uvicorn
			pythonBin := resolvePythonBin()
			args := append([]string{"-m", "uvicorn"}, baseArgs...)
			cmd := exec.CommandContext(ctx, pythonBin, args...)
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
