package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manifest represents a bigbase.yaml configuration file in a repo root.
type Manifest struct {
	Version   int               `yaml:"version"`
	Framework string            `yaml:"framework"`
	Build     ManifestBuild     `yaml:"build"`
	Start     ManifestStart     `yaml:"start"`
	Env       map[string]string `yaml:"env"`
}

// ManifestBuild represents the build section of bigbase.yaml.
type ManifestBuild struct {
	Command string `yaml:"command"`
	Output  string `yaml:"output"`
}

// ManifestStart represents the start section of bigbase.yaml.
type ManifestStart struct {
	Command string `yaml:"command"`
	Port    int    `yaml:"port"`
}

// validFrameworks is the set of framework values accepted by the manifest.
var validFrameworks = map[string]bool{
	"sveltekit": true,
	"astro":     true,
	"next":      true,
	"vue":       true,
	"react":     true,
	"static":    true,
	"go":        true,
	"python":    true,
	"node":      true,
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
// If manifestPath is empty, defaults to "bigbase.yaml".
// Returns nil, nil if the file does not exist.
func LoadManifestPath(dir, manifestPath string) (*Manifest, error) {
	if manifestPath == "" {
		manifestPath = "bigbase.yaml"
	}
	path := filepath.Join(dir, manifestPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", manifestPath, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", manifestPath, err)
	}

	if err := m.validate(); err != nil {
		return nil, fmt.Errorf("validate %s: %w", manifestPath, err)
	}

	return &m, nil
}

// ValidateManifest parses and validates manifest YAML content.
func ValidateManifest(data []byte) error {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
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
	if m.Start.Command == "" {
		return fmt.Errorf("start.command is required")
	}
	if m.Start.Port < 1 || m.Start.Port > 65535 {
		return fmt.Errorf("start.port must be between 1 and 65535, got %d", m.Start.Port)
	}
	return nil
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
			m.Build.Command = "npm run build"
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
				startCmd = "npm start"
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
