package deploy

import (
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

// LoadManifest reads and validates a bigbase.yaml from the given directory.
// Returns nil, nil if the file does not exist — callers should fall back to
// auto-detection.
func LoadManifest(dir string) (*Manifest, error) {
	path := filepath.Join(dir, "bigbase.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read bigbase.yaml: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse bigbase.yaml: %w", err)
	}

	if err := m.validate(); err != nil {
		return nil, fmt.Errorf("validate bigbase.yaml: %w", err)
	}

	return &m, nil
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
