package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// ManifestHealthCheck defines the health check endpoint probe configuration.
// Zero-valued fields are replaced by defaults via WithDefaults().
type ManifestHealthCheck struct {
	Path                 string `yaml:"path"                  toml:"path"`
	ExpectedStatus       int    `yaml:"expected_status"        toml:"expected_status"`
	ExpectedBodyContains string `yaml:"expected_body_contains"  toml:"expected_body_contains"`
	TimeoutSeconds       int    `yaml:"timeout_seconds"         toml:"timeout_seconds"`
	IntervalSeconds      int    `yaml:"interval_seconds"        toml:"interval_seconds"`
	MaxRetries           int    `yaml:"max_retries"             toml:"max_retries"`
}

// WithDefaults returns a ManifestHealthCheck with zero-valued fields filled
// from the defaults: path=/, expected_status=200, timeout_seconds=30,
// interval_seconds=2, max_retries=5. expected_body_contains defaults to empty
// (no body assertion).
func (hc ManifestHealthCheck) WithDefaults() ManifestHealthCheck {
	if hc.Path == "" {
		hc.Path = "/"
	}
	if hc.ExpectedStatus == 0 {
		hc.ExpectedStatus = 200
	}
	if hc.TimeoutSeconds == 0 {
		hc.TimeoutSeconds = 30
	}
	if hc.IntervalSeconds == 0 {
		hc.IntervalSeconds = 2
	}
	if hc.MaxRetries == 0 {
		hc.MaxRetries = 5
	}
	return hc
}

// ManifestSecurity represents the [security] section of bigbase.yaml / bigbase.toml.
type ManifestSecurity struct {
	CSP string `yaml:"csp" toml:"csp"`
}

// Manifest represents a bigbase.yaml configuration file in a repo root.
type Manifest struct {
	Version     int                 `yaml:"version"              toml:"version"`
	Framework   string              `yaml:"framework"            toml:"framework"`
	Build       ManifestBuild       `yaml:"build"                toml:"build"`
	Start       ManifestStart       `yaml:"start"                toml:"start"`
	Env         map[string]string   `yaml:"env"                  toml:"env"`
	HealthCheck ManifestHealthCheck `yaml:"health_check"          toml:"health_check"`
	Security    ManifestSecurity    `yaml:"security"             toml:"security"`
}

// ManifestBuild represents the build section of bigbase.yaml / bigbase.toml.
type ManifestBuild struct {
	Command string `yaml:"command"  toml:"command"`
	Output  string `yaml:"output"   toml:"output"`
}

// ManifestStart represents the start section of bigbase.yaml / bigbase.toml.
type ManifestStart struct {
	Command    string `yaml:"command"     toml:"command"`
	Port       int    `yaml:"port"        toml:"port"`
	ASGIImport string `yaml:"asgi_import" toml:"asgi_import"`
}

// validFrameworks is the set of framework values accepted by the manifest.
var validFrameworks = map[string]bool{
	"sveltekit":      true,
	"astro":          true,
	"next":           true,
	"vue":            true,
	"react":          true,
	"static":         true,
	"static-sidecar": true,
	"go":             true,
	"python":         true,
	"node":           true,
}

// manifestToAppType converts a manifest framework string to an AppType.
func manifestToAppType(m *Manifest) AppType {
	switch m.Framework {
	case "node", "sveltekit", "astro", "next", "vue", "react":
		return AppNode
	case "go":
		return AppGo
	case "python":
		return AppPython
	case "static":
		return AppStatic
	case "static-sidecar":
		return AppStaticSidecar
	default:
		return AppStatic
	}
}

// LoadManifest reads and validates a bigbase.yaml from the given directory.
// Returns nil, nil if the file does not exist — callers should fall back to
// auto-detection.
func LoadManifest(dir string) (*Manifest, error) {
	return LoadManifestPath(dir, "")
}

// LoadManifestPath reads and validates a manifest file from the given directory using the specified manifest filename.
// If manifestPath is empty, checks for bigbase.toml first, then falls back to bigbase.yaml.
// Returns nil, nil if neither file exists.
func LoadManifestPath(dir, manifestPath string) (*Manifest, error) {
	if manifestPath == "" {
		// Prefer TOML over YAML when both exist
		tomlPath := filepath.Join(dir, "bigbase.toml")
		if _, err := os.Stat(tomlPath); err == nil {
			manifestPath = "bigbase.toml"
		} else {
			manifestPath = "bigbase.yaml"
		}
	}

	// Prevent path traversal (CWE-22): resolve and verify path stays within dir.
	cleanManifestPath := filepath.Clean(manifestPath)
	if filepath.IsAbs(cleanManifestPath) {
		return nil, fmt.Errorf("manifest path must be relative: %s", manifestPath)
	}
	path := filepath.Join(dir, cleanManifestPath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve manifest dir: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve manifest path: %w", err)
	}
	if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) && absPath != absDir {
		return nil, fmt.Errorf("manifest path escapes project directory: %s", manifestPath)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", manifestPath, err)
	}

	var m Manifest
	if strings.HasSuffix(manifestPath, ".toml") {
		if err := toml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parse %s: %w", manifestPath, err)
		}
	} else {
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parse %s: %w", manifestPath, err)
		}
	}

	if err := m.validate(); err != nil {
		return nil, fmt.Errorf("validate %s: %w", manifestPath, err)
	}

	return &m, nil
}

// ValidateManifest parses and validates manifest YAML or TOML content.
func ValidateManifest(data []byte) error {
	var m Manifest
	// Try TOML first (detected by key patterns), fall back to YAML
	if err := toml.Unmarshal(data, &m); err == nil && m.Version > 0 {
		return m.validate()
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	return m.validate()
}

func (m *Manifest) validate() error {
	if m.Version <= 0 {
		return fmt.Errorf("version is required")
	}
	if m.Framework == "" {
		return fmt.Errorf("framework is required")
	}
	if !validFrameworks[m.Framework] {
		return fmt.Errorf("invalid framework %q", m.Framework)
	}
	if m.Build.Command == "" {
		return fmt.Errorf("build.command is required")
	}
	// start.command is required unless Python framework with explicit asgi_import.
	if m.Start.Command == "" {
		if m.Framework != "python" || m.Start.ASGIImport == "" {
			return fmt.Errorf("start.command is required")
		}
	}
	if m.Start.Port < 1 || m.Start.Port > 65535 {
		return fmt.Errorf("start.port must be between 1 and 65535, got %d", m.Start.Port)
	}
	return nil
}

// MergeManifests applies three-layer configuration merge: manifest (bigbase.yaml/toml)
// → site defaults → request overrides. Non-zero values in higher-priority layers
// override lower-priority ones. Returns a new Manifest; does not modify inputs.
func MergeManifests(manifest *Manifest, siteDefaults *SiteDefaults, requestOverrides *Manifest) *Manifest {
	base := Manifest{}
	if manifest != nil {
		base = *manifest
	}

	// Layer 2: site defaults (from deploy_defaults on site record)
	if siteDefaults != nil {
		if siteDefaults.AppType != "" && base.Framework == "" {
			base.Framework = siteDefaults.AppType
		}
		if siteDefaults.BuildCommand != "" && base.Build.Command == "" {
			base.Build.Command = siteDefaults.BuildCommand
		}
		if siteDefaults.StartCommand != "" && base.Start.Command == "" {
			base.Start.Command = siteDefaults.StartCommand
		}
		if siteDefaults.HealthPath != "" && base.HealthCheck.Path == "" {
			base.HealthCheck.Path = siteDefaults.HealthPath
		}
		// Merge env vars (site defaults → manifest)
		if len(siteDefaults.Env) > 0 {
			if base.Env == nil {
				base.Env = make(map[string]string)
			}
			for k, v := range siteDefaults.Env {
				if _, exists := base.Env[k]; !exists {
					base.Env[k] = v
				}
			}
		}
		if siteDefaults.CSPPolicy != "" && base.Security.CSP == "" {
			base.Security.CSP = siteDefaults.CSPPolicy
		}
	}

	// Layer 3: request overrides (explicit values win)
	if requestOverrides != nil {
		if requestOverrides.Framework != "" {
			base.Framework = requestOverrides.Framework
		}
		if requestOverrides.Build.Command != "" {
			base.Build.Command = requestOverrides.Build.Command
		}
		if requestOverrides.Build.Output != "" {
			base.Build.Output = requestOverrides.Build.Output
		}
		if requestOverrides.Start.Command != "" {
			base.Start.Command = requestOverrides.Start.Command
		}
		if requestOverrides.Start.Port > 0 {
			base.Start.Port = requestOverrides.Start.Port
		}
		if requestOverrides.Start.ASGIImport != "" {
			base.Start.ASGIImport = requestOverrides.Start.ASGIImport
		}
		if requestOverrides.HealthCheck.Path != "" {
			base.HealthCheck.Path = requestOverrides.HealthCheck.Path
		}
		if requestOverrides.HealthCheck.ExpectedStatus > 0 {
			base.HealthCheck.ExpectedStatus = requestOverrides.HealthCheck.ExpectedStatus
		}
		if requestOverrides.HealthCheck.TimeoutSeconds > 0 {
			base.HealthCheck.TimeoutSeconds = requestOverrides.HealthCheck.TimeoutSeconds
		}
		// Merge env vars (request overrides both)
		if len(requestOverrides.Env) > 0 {
			if base.Env == nil {
				base.Env = make(map[string]string)
			}
			for k, v := range requestOverrides.Env {
				base.Env[k] = v
			}
		}
	}

	return &base
}

// SiteDefaults is a minimal representation of deploy_defaults from the sites table,
// used as the middle layer in three-layer merge.
type SiteDefaults struct {
	AppType          string            `json:"app_type,omitempty"`
	BuildCommand     string            `json:"build_command,omitempty"`
	StartCommand     string            `json:"start_command,omitempty"`
	PassthroughPaths []string          `json:"passthrough_paths,omitempty"`
	HealthPath       string            `json:"health_path,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	CSPPolicy        string            `json:"csp_policy,omitempty"`
}

// PackageJSON is a helper struct for parsing dependencies and scripts from package.json.
type PackageJSON struct {
	Name            string            `json:"name"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
}

// InitManifest detects the framework in dir and writes a default bigbase.yaml.
func InitManifest(dir string) error {
	manifestPath := filepath.Join(dir, "bigbase.yaml")
	if _, err := os.Stat(manifestPath); err == nil {
		return fmt.Errorf("bigbase.yaml already exists")
	}

	var m Manifest
	m.Version = 1

	packageJSONPath := filepath.Join(dir, "package.json")
	goModPath := filepath.Join(dir, "go.mod")
	pythonMainPath := filepath.Join(dir, "main.py")
	pythonAppPath := filepath.Join(dir, "app.py")

	if _, err := os.Stat(packageJSONPath); err == nil {
		data, err := os.ReadFile(packageJSONPath)
		if err != nil {
			return fmt.Errorf("read package.json: %w", err)
		}
		var pkg PackageJSON
		if err := json.Unmarshal(data, &pkg); err != nil {
			return fmt.Errorf("parse package.json: %w", err)
		}

		isSvelteKit := false
		if pkg.Dependencies != nil && pkg.Dependencies["@sveltejs/kit"] != "" {
			isSvelteKit = true
		}
		if pkg.DevDependencies != nil && pkg.DevDependencies["@sveltejs/kit"] != "" {
			isSvelteKit = true
		}

		if isSvelteKit {
			m.Framework = "sveltekit"
			m.Build.Command = NodePMRunCommand(dir, "build")
			m.Build.Output = "build/"
			m.Start.Command = "node build/index.js"
			m.Start.Port = 3000
		} else {
			m.Framework = "node"
			var buildCmd string
			if pkg.Scripts != nil && pkg.Scripts["build"] != "" {
				buildCmd = pkg.Scripts["build"]
			} else {
				buildCmd = "echo build"
			}
			m.Build.Command = buildCmd
			startCmd := "node index.js"
			if pkg.Scripts != nil && pkg.Scripts["start"] != "" {
				startCmd = NodePMStartCommand(dir)
			}
			m.Start.Command = startCmd
			m.Start.Port = 3000
		}
	} else if _, err := os.Stat(goModPath); err == nil {
		m.Framework = "go"
		m.Build.Command = "go build -o app ."
		m.Start.Command = "./app"
		m.Start.Port = 8080
	} else if fileExists(pythonMainPath) || fileExists(pythonAppPath) {
		m.Framework = "python"
		reqPath := filepath.Join(dir, "requirements.txt")
		if _, err := os.Stat(reqPath); err == nil {
			m.Build.Command = "pip install -r requirements.txt"
		} else {
			m.Build.Command = "echo 'no requirements.txt'"
		}
		if fileExists(pythonMainPath) {
			m.Start.Command = "python main.py"
		} else {
			m.Start.Command = "python app.py"
		}
		m.Start.Port = 8000
	} else {
		m.Framework = "static"
		m.Build.Command = "echo static"
		m.Start.Command = "echo static"
		m.Start.Port = 3000
	}

	yamlData, err := yaml.Marshal(&m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, yamlData, 0644); err != nil {
		return fmt.Errorf("write bigbase.yaml: %w", err)
	}

	return nil
}
