package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestParse(t *testing.T) {
	t.Run("parses valid bigbase.yaml", func(t *testing.T) {
		dir := t.TempDir()
		yaml := `version: 1
framework: sveltekit
build:
  command: npm run build
  output: build/
start:
  command: node build/index.js
  port: 3000
env:
  NODE_VERSION: "20"
`
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), yaml)

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: unexpected error: %v", err)
		}
		if m == nil {
			t.Fatal("expected manifest, got nil")
		}
		if m.Version != 1 {
			t.Errorf("version = %d, want 1", m.Version)
		}
		if m.Framework != "sveltekit" {
			t.Errorf("framework = %s, want sveltekit", m.Framework)
		}
		if m.Build.Command != "npm run build" {
			t.Errorf("build.command = %s, want 'npm run build'", m.Build.Command)
		}
		if m.Build.Output != "build/" {
			t.Errorf("build.output = %s, want build/", m.Build.Output)
		}
		if m.Start.Command != "node build/index.js" {
			t.Errorf("start.command = %s, want 'node build/index.js'", m.Start.Command)
		}
		if m.Start.Port != 3000 {
			t.Errorf("start.port = %d, want 3000", m.Start.Port)
		}
		if len(m.Env) != 1 || m.Env["NODE_VERSION"] != "20" {
			t.Errorf("env = %v, want {NODE_VERSION: 20}", m.Env)
		}
	})

	t.Run("returns nil when bigbase.yaml does not exist", func(t *testing.T) {
		dir := t.TempDir()
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: unexpected error: %v", err)
		}
		if m != nil {
			t.Fatal("expected nil, got manifest")
		}
	})

	t.Run("rejects invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), "version: [this is not valid yaml!!{{{")
		_, err := LoadManifest(dir)
		if err == nil {
			t.Fatal("expected error for invalid YAML, got nil")
		}
	})

	t.Run("rejects invalid framework value", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), `version: 1
framework: invalid_framework
build:
  command: make
start:
  command: ./app
  port: 8080
`)
		_, err := LoadManifest(dir)
		if err == nil {
			t.Fatal("expected error for invalid framework, got nil")
		}
	})

	t.Run("rejects port out of range", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), `version: 1
framework: static
build:
  command: echo hi
start:
  command: echo hi
  port: 70000
`)
		_, err := LoadManifest(dir)
		if err == nil {
			t.Fatal("expected error for port out of range, got nil")
		}
	})

	t.Run("rejects missing required fields", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), `version: 1
framework: node
`)
		_, err := LoadManifest(dir)
		if err == nil {
			t.Fatal("expected error for missing required fields, got nil")
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
}
