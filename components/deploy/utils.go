package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var pickPortMu sync.Mutex
var pickPortCounter int64

func pickPort(base int) int {
	pickPortMu.Lock()
	defer pickPortMu.Unlock()
	pickPortCounter++
	return base + int(pickPortCounter)
}

func marshalPassthroughPaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	data, _ := json.Marshal(paths)
	return string(data)
}
func parsePassthroughPaths(raw string) []string {
	if raw == "" {
		return nil
	}
	var paths []string
	if err := json.Unmarshal([]byte(raw), &paths); err != nil {
		return nil
	}
	return paths
}
func FormatBuildCommand(name string, args ...string) string {
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}
func DetectAppType(buildDir string) AppType {
	if fileExists(filepath.Join(buildDir, "package.json")) {
		return AppNode
	}
	if fileExists(filepath.Join(buildDir, "go.mod")) {
		return AppGo
	}
	// Python: pyproject.toml (PEP 518/621) is the primary signal for modern
	// Python projects. Fall back to app.py/main.py for legacy projects.
	if HasPyProjectTOML(buildDir) {
		return AppPython
	}
	if fileExists(filepath.Join(buildDir, "app.py")) ||
		fileExists(filepath.Join(buildDir, "main.py")) {
		return AppPython
	}
	if fileExists(filepath.Join(buildDir, "index.html")) {
		return AppStatic
	}
	return AppStatic
}
func GetStartCommand(buildDir string) string {
	data, err := os.ReadFile(filepath.Join(buildDir, "package.json"))
	if err != nil {
		return "node index.js"
	}

	var pkg struct {
		Scripts struct {
			Start string `json:"start"`
		} `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "node index.js"
	}

	if pkg.Scripts.Start != "" {
		return pkg.Scripts.Start
	}
	return "node index.js"
}
func ValidateNodeBuildScript(dir string) error {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return fmt.Errorf("cannot read package.json: %w", err)
	}
	var pkg struct {
		Scripts struct {
			Build string `json:"build"`
		} `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("invalid package.json: %w", err)
	}
	if pkg.Scripts.Build == "" {
		return fmt.Errorf("no build script found in package.json: add a \"build\" entry to \"scripts\" or set app_type to static")
	}
	return nil
}
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
