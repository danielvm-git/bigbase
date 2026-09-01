package deploy

import (
	"path/filepath"
	"testing"
)

func TestManifestIntegration(t *testing.T) {
	t.Run("manifest overrides auto-detection for framework", func(t *testing.T) {
		dir := t.TempDir()
		// Create a package.json (would normally trigger AppNode detection)
		writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)
		// But also create bigbase.yaml declaring go framework
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), `version: 1
framework: go
build:
  command: go build -o app .
start:
  command: ./app
  port: 8080
`)

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		if m == nil {
			t.Fatal("expected manifest, got nil")
		}

		// Manifest framework should take precedence
		appType := manifestToAppType(m)
		if appType != AppGo {
			t.Errorf("appType = %s, want %s (go)", appType, AppGo)
		}
	})

	t.Run("manifest provides build and start commands", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), `version: 1
framework: sveltekit
build:
  command: npm run build
  output: build/
start:
  command: node build/index.js
  port: 3000
`)

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}

		buildCmd := m.Build.Command
		if buildCmd != "npm run build" {
			t.Errorf("build command = %s, want 'npm run build'", buildCmd)
		}

		startCmd := m.Start.Command
		if startCmd != "node build/index.js" {
			t.Errorf("start command = %s, want 'node build/index.js'", startCmd)
		}

		port := m.Start.Port
		if port != 3000 {
			t.Errorf("port = %d, want 3000", port)
		}
	})

	t.Run("no manifest falls back to auto-detection", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"name":"test"}`)

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		if m != nil {
			t.Fatal("expected nil manifest when file absent")
		}

		// Auto-detection should still work
		appType := DetectAppType(dir)
		if appType != AppNode {
			t.Errorf("auto-detect appType = %s, want %s", appType, AppNode)
		}
	})

	t.Run("manifest env variables are accessible", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), `version: 1
framework: node
build:
  command: npm run build
start:
  command: npm start
  port: 3000
env:
  NODE_VERSION: "20"
  DATABASE_URL: "postgres://localhost:5432"
`)

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}

		if len(m.Env) != 2 {
			t.Fatalf("env count = %d, want 2", len(m.Env))
		}
		if m.Env["NODE_VERSION"] != "20" {
			t.Errorf("NODE_VERSION = %s, want 20", m.Env["NODE_VERSION"])
		}
		if m.Env["DATABASE_URL"] != "postgres://localhost:5432" {
			t.Errorf("DATABASE_URL = %s, want postgres://localhost:5432", m.Env["DATABASE_URL"])
		}
	})

	t.Run("manifest path persistence and loading", func(t *testing.T) {
		dir := t.TempDir()
		// Write a manifest to a custom file path
		writeFile(t, filepath.Join(dir, "custom-manifest.yaml"), `version: 1
framework: go
build:
  command: go build -o testapp .
start:
  command: ./testapp
  port: 9090
`)

		m, err := LoadManifestPath(dir, "custom-manifest.yaml")
		if err != nil {
			t.Fatalf("LoadManifestPath: %v", err)
		}
		if m == nil {
			t.Fatal("expected manifest, got nil")
			return
		}
		if m.Framework != "go" || m.Build.Command != "go build -o testapp ." {
			t.Errorf("unexpected custom manifest contents: %+v", m)
		}
	})
}

func TestManifestFlags(t *testing.T) {
	// Merge order: CLI flags > manifest > auto-detection
	t.Run("CLI appType override takes precedence over manifest framework", func(t *testing.T) {
		dir := t.TempDir()
		// Auto-detection would see go (due to go.mod)
		writeFile(t, filepath.Join(dir, "go.mod"), "module test")
		// Manifest would declare sveltekit (node)
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), `version: 1
framework: sveltekit
build:
  command: npm run build
start:
  command: node build/index.js
  port: 3000
`)

		// 1. If appType is specified (CLI flags), it wins:
		cliAppType := AppPython
		manifest, _ := LoadManifest(dir)

		appType := cliAppType
		if appType == "" {
			if manifest != nil {
				appType = manifestToAppType(manifest)
			} else {
				appType = DetectAppType(dir)
			}
		}
		if appType != AppPython {
			t.Errorf("expected AppPython from CLI, got %s", appType)
		}

		// 2. If appType is empty, manifest wins:
		cliAppType = ""
		appType = cliAppType
		if appType == "" {
			if manifest != nil {
				appType = manifestToAppType(manifest)
			} else {
				appType = DetectAppType(dir)
			}
		}
		if appType != AppNode { // sveltekit framework maps to AppNode
			t.Errorf("expected AppNode from manifest, got %s", appType)
		}

		// 3. If both empty, auto-detection wins:
		cliAppType = ""
		appType = cliAppType
		if appType == "" {
			// mock no manifest scenario
			appType = DetectAppType(dir)
		}
		if appType != AppGo {
			t.Errorf("expected AppGo from auto-detection, got %s", appType)
		}
	})
}
