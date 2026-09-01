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
			return
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

func TestInitManifest(t *testing.T) {
	t.Run("detects SvelteKit framework", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"devDependencies":{"@sveltejs/kit":"^2.0.0"}}`)
		err := InitManifest(dir)
		if err != nil {
			t.Fatalf("InitManifest failed: %v", err)
		}
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest failed: %v", err)
		}
		if m.Framework != "sveltekit" || m.Build.Command != "npm run build" || m.Start.Command != "node build/index.js" {
			t.Errorf("unexpected sveltekit manifest: %+v", m)
		}
	})

	t.Run("detects generic Node framework with scripts", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"scripts":{"build":"npm run compile","start":"node server.js"}}`)
		err := InitManifest(dir)
		if err != nil {
			t.Fatalf("InitManifest failed: %v", err)
		}
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest failed: %v", err)
		}
		if m.Framework != "node" || m.Build.Command != "npm run compile" || m.Start.Command != "npm start" {
			t.Errorf("unexpected node manifest: %+v", m)
		}
	})

	t.Run("detects generic Node framework without scripts", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{}`)
		err := InitManifest(dir)
		if err != nil {
			t.Fatalf("InitManifest failed: %v", err)
		}
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest failed: %v", err)
		}
		if m.Framework != "node" || m.Build.Command != "echo build" || m.Start.Command != "node index.js" {
			t.Errorf("unexpected node manifest: %+v", m)
		}
	})

	t.Run("detects Go framework", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "go.mod"), "module test")
		err := InitManifest(dir)
		if err != nil {
			t.Fatalf("InitManifest failed: %v", err)
		}
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest failed: %v", err)
		}
		if m.Framework != "go" || m.Build.Command != "go build -o app ." || m.Start.Command != "./app" {
			t.Errorf("unexpected go manifest: %+v", m)
		}
	})

	t.Run("detects Python framework", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.py"), "print('hi')")
		writeFile(t, filepath.Join(dir, "requirements.txt"), "flask")
		err := InitManifest(dir)
		if err != nil {
			t.Fatalf("InitManifest failed: %v", err)
		}
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest failed: %v", err)
		}
		if m.Framework != "python" || m.Build.Command != "pip install -r requirements.txt" || m.Start.Command != "python main.py" {
			t.Errorf("unexpected python manifest: %+v", m)
		}
	})

	t.Run("falls back to Static framework", func(t *testing.T) {
		dir := t.TempDir()
		err := InitManifest(dir)
		if err != nil {
			t.Fatalf("InitManifest failed: %v", err)
		}
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest failed: %v", err)
		}
		if m.Framework != "static" || m.Build.Command != "echo static" || m.Start.Command != "echo static" {
			t.Errorf("unexpected static manifest: %+v", m)
		}
	})

	t.Run("fails if bigbase.yaml already exists", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), "exists")
		err := InitManifest(dir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestValidateManifest(t *testing.T) {
	t.Run("valid manifest passes", func(t *testing.T) {
		data := []byte(`version: 1
framework: static
build:
  command: echo hi
start:
  command: serve
  port: 3000
`)
		if err := ValidateManifest(data); err != nil {
			t.Fatalf("ValidateManifest: unexpected error: %v", err)
		}
	})

	t.Run("invalid YAML fails", func(t *testing.T) {
		data := []byte(`version: [broken yaml{{{`)
		if err := ValidateManifest(data); err == nil {
			t.Fatal("expected error for invalid YAML, got nil")
		}
	})

	t.Run("missing version fails", func(t *testing.T) {
		data := []byte(`framework: static
build:
  command: echo
start:
  command: serve
  port: 3000
`)
		if err := ValidateManifest(data); err == nil {
			t.Fatal("expected error for missing version, got nil")
		}
	})

	t.Run("invalid framework fails", func(t *testing.T) {
		data := []byte(`version: 1
framework: unknown-framework
build:
  command: echo
start:
  command: serve
  port: 3000
`)
		if err := ValidateManifest(data); err == nil {
			t.Fatal("expected error for invalid framework, got nil")
		}
	})

	t.Run("port out of range fails", func(t *testing.T) {
		data := []byte(`version: 1
framework: static
build:
  command: echo
start:
  command: serve
  port: 99999
`)
		if err := ValidateManifest(data); err == nil {
			t.Fatal("expected error for port out of range, got nil")
		}
	})

	t.Run("missing build.command fails", func(t *testing.T) {
		data := []byte(`version: 1
framework: static
build: {}
start:
  command: serve
  port: 3000
`)
		if err := ValidateManifest(data); err == nil {
			t.Fatal("expected error for missing build.command, got nil")
		}
	})
}

func TestManifestHealthCheck(t *testing.T) {
	t.Run("defaults when health_check omitted", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), `version: 1
framework: node
build:
  command: npm run build
start:
  command: node index.js
  port: 3000
`)
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		if m == nil {
			t.Fatal("expected manifest")
			return
		}
		hc := m.HealthCheck.WithDefaults()
		if hc.Path != "/" {
			t.Errorf("default path = %q, want /", hc.Path)
		}
		if hc.ExpectedStatus != 200 {
			t.Errorf("default expected_status = %d, want 200", hc.ExpectedStatus)
		}
		if hc.TimeoutSeconds != 30 {
			t.Errorf("default timeout_seconds = %d, want 30", hc.TimeoutSeconds)
		}
		if hc.IntervalSeconds != 2 {
			t.Errorf("default interval_seconds = %d, want 2", hc.IntervalSeconds)
		}
		if hc.MaxRetries != 5 {
			t.Errorf("default max_retries = %d, want 5", hc.MaxRetries)
		}
		if hc.ExpectedBodyContains != "" {
			t.Errorf("default expected_body_contains = %q, want empty", hc.ExpectedBodyContains)
		}
	})

	t.Run("partial health_check overrides only set fields", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), `version: 1
framework: node
build:
  command: npm run build
start:
  command: node index.js
  port: 3000
health_check:
  path: /api/health
  expected_status: 204
`)
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		if m == nil {
			t.Fatal("expected manifest")
			return
		}
		hc := m.HealthCheck.WithDefaults()
		if hc.Path != "/api/health" {
			t.Errorf("path = %q, want /api/health", hc.Path)
		}
		if hc.ExpectedStatus != 204 {
			t.Errorf("expected_status = %d, want 204", hc.ExpectedStatus)
		}
		// Non-overridden fields should get defaults
		if hc.TimeoutSeconds != 30 {
			t.Errorf("timeout_seconds = %d, want 30", hc.TimeoutSeconds)
		}
		if hc.IntervalSeconds != 2 {
			t.Errorf("interval_seconds = %d, want 2", hc.IntervalSeconds)
		}
		if hc.MaxRetries != 5 {
			t.Errorf("max_retries = %d, want 5", hc.MaxRetries)
		}
		if hc.ExpectedBodyContains != "" {
			t.Errorf("expected_body_contains = %q, want empty", hc.ExpectedBodyContains)
		}
	})

	t.Run("full health_check overrides all defaults", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), `version: 1
framework: node
build:
  command: npm run build
start:
  command: node index.js
  port: 3000
health_check:
  path: /healthz
  expected_status: 200
  expected_body_contains: ok
  timeout_seconds: 60
  interval_seconds: 5
  max_retries: 10
`)
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		if m == nil {
			t.Fatal("expected manifest")
			return
		}
		hc := m.HealthCheck.WithDefaults()
		if hc.Path != "/healthz" {
			t.Errorf("path = %q, want /healthz", hc.Path)
		}
		if hc.ExpectedStatus != 200 {
			t.Errorf("expected_status = %d, want 200", hc.ExpectedStatus)
		}
		if hc.ExpectedBodyContains != "ok" {
			t.Errorf("expected_body_contains = %q, want ok", hc.ExpectedBodyContains)
		}
		if hc.TimeoutSeconds != 60 {
			t.Errorf("timeout_seconds = %d, want 60", hc.TimeoutSeconds)
		}
		if hc.IntervalSeconds != 5 {
			t.Errorf("interval_seconds = %d, want 5", hc.IntervalSeconds)
		}
		if hc.MaxRetries != 10 {
			t.Errorf("max_retries = %d, want 10", hc.MaxRetries)
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
}

func TestValidateManifest_ASGIImportRelaxesStartCommand(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "Python with asgi_import and no start.command is valid",
			yaml: `version: 1
framework: python
build:
  command: uv sync --frozen
start:
  asgi_import: grimoire.app:create_app --factory
  port: 8000
`,
			wantErr: false,
		},
		{
			name: "Python without asgi_import and no start.command is invalid",
			yaml: `version: 1
framework: python
build:
  command: uv sync --frozen
start:
  port: 8000
`,
			wantErr: true,
		},
		{
			name: "Node without start.command is invalid (asgi_import ignored)",
			yaml: `version: 1
framework: node
build:
  command: npm run build
start:
  asgi_import: something:app
  port: 3000
`,
			wantErr: true,
		},
		{
			name: "Python with both asgi_import and start.command is valid",
			yaml: `version: 1
framework: python
build:
  command: uv sync --frozen
start:
  command: uvicorn myapp.main:app
  asgi_import: grimoire.app:create_app
  port: 8000
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateManifest([]byte(tt.yaml))
			if tt.wantErr && err == nil {
				t.Errorf("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestManifestParseTOML(t *testing.T) {
	t.Run("parses valid bigbase.toml", func(t *testing.T) {
		dir := t.TempDir()
		tomlContent := `version = 1
framework = "sveltekit"

[build]
command = "npm run build"
output = "build/"

[start]
command = "node build/index.js"
port = 3000

[env]
NODE_VERSION = "20"
`
		writeFile(t, filepath.Join(dir, "bigbase.toml"), tomlContent)

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: unexpected error: %v", err)
		}
		if m == nil {
			t.Fatal("expected manifest, got nil")
			return
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

	t.Run("prefers bigbase.toml over bigbase.yaml", func(t *testing.T) {
		dir := t.TempDir()
		tomlContent := `version = 1
framework = "go"

[build]
command = "go build -o app ."

[start]
command = "./app"
port = 8080
`
		yamlContent := `version: 1
framework: static
build:
  command: echo
start:
  command: echo
  port: 3000
`
		writeFile(t, filepath.Join(dir, "bigbase.toml"), tomlContent)
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), yamlContent)

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		if m.Framework != "go" {
			t.Errorf("framework = %s, want go (TOML should take precedence)", m.Framework)
		}
	})

	t.Run("falls back to YAML when no TOML exists", func(t *testing.T) {
		dir := t.TempDir()
		yamlContent := `version: 1
framework: python
build:
  command: pip install .
start:
  command: python app.py
  port: 8000
`
		writeFile(t, filepath.Join(dir, "bigbase.yaml"), yamlContent)

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		if m.Framework != "python" {
			t.Errorf("framework = %s, want python", m.Framework)
		}
	})
}

func TestMergeManifests(t *testing.T) {
	t.Run("manifest alone", func(t *testing.T) {
		manifest := &Manifest{
			Version:   1,
			Framework: "node",
			Build:     ManifestBuild{Command: "npm run build"},
			Start:     ManifestStart{Command: "npm start", Port: 3000},
		}
		merged := MergeManifests(manifest, nil, nil)
		if merged.Framework != "node" {
			t.Errorf("framework = %s, want node", merged.Framework)
		}
	})

	t.Run("site defaults fill gaps", func(t *testing.T) {
		manifest := &Manifest{
			Version:   1,
			Framework: "node",
			Build:     ManifestBuild{},
			Start:     ManifestStart{Port: 3000},
		}
		siteDefaults := &SiteDefaults{
			BuildCommand: "npm run build",
			StartCommand: "npm start",
			HealthPath:   "/api/health",
		}
		merged := MergeManifests(manifest, siteDefaults, nil)
		if merged.Build.Command != "npm run build" {
			t.Errorf("build.command = %s, want 'npm run build'", merged.Build.Command)
		}
		if merged.Start.Command != "npm start" {
			t.Errorf("start.command = %s, want 'npm start'", merged.Start.Command)
		}
		if merged.HealthCheck.Path != "/api/health" {
			t.Errorf("health_check.path = %s, want '/api/health'", merged.HealthCheck.Path)
		}
	})

	t.Run("request overrides win", func(t *testing.T) {
		manifest := &Manifest{
			Version:   1,
			Framework: "node",
			Build:     ManifestBuild{Command: "npm run build"},
			Start:     ManifestStart{Command: "npm start", Port: 3000},
		}
		siteDefaults := &SiteDefaults{
			BuildCommand: "yarn build",
			StartCommand: "yarn start",
		}
		overrides := &Manifest{
			Build: ManifestBuild{Command: "pnpm run build"},
		}
		merged := MergeManifests(manifest, siteDefaults, overrides)
		// Request override wins over manifest
		if merged.Build.Command != "pnpm run build" {
			t.Errorf("build.command = %s, want 'pnpm run build' (request override wins)", merged.Build.Command)
		}
		// Request override wins
		merged2 := MergeManifests(&Manifest{Build: ManifestBuild{}}, siteDefaults, overrides)
		if merged2.Build.Command != "pnpm run build" {
			t.Errorf("build.command with override = %s, want 'pnpm run build'", merged2.Build.Command)
		}
	})

	t.Run("env merge preserves manifest, adds site, overrides with request", func(t *testing.T) {
		manifest := &Manifest{
			Env: map[string]string{"A": "1", "B": "2"},
		}
		siteDefaults := &SiteDefaults{
			Env: map[string]string{"B": "site", "C": "3"},
		}
		overrides := &Manifest{
			Env: map[string]string{"C": "req"},
		}
		merged := MergeManifests(manifest, siteDefaults, overrides)
		if merged.Env["A"] != "1" {
			t.Errorf("env.A = %s, want 1", merged.Env["A"])
		}
		if merged.Env["B"] != "2" {
			t.Errorf("env.B = %s, want 2 (manifest wins over site)", merged.Env["B"])
		}
		if merged.Env["C"] != "req" {
			t.Errorf("env.C = %s, want req (request wins)", merged.Env["C"])
		}
	})

	t.Run("manifest security.csp wins over site default", func(t *testing.T) {
		manifest := &Manifest{
			Security: ManifestSecurity{CSP: "default-src 'self'"},
		}
		siteDefaults := &SiteDefaults{
			CSPPolicy: "default-src 'self' https://cdn.example.com",
		}
		merged := MergeManifests(manifest, siteDefaults, nil)
		if merged.Security.CSP != "default-src 'self'" {
			t.Errorf("Security.CSP = %q, want manifest value 'default-src 'self''", merged.Security.CSP)
		}
	})

	t.Run("site default csp fills in when manifest has none", func(t *testing.T) {
		manifest := &Manifest{
			Version:   1,
			Framework: "static",
		}
		siteDefaults := &SiteDefaults{
			CSPPolicy: "default-src 'self' https://cdn.example.com",
		}
		merged := MergeManifests(manifest, siteDefaults, nil)
		if merged.Security.CSP != "default-src 'self' https://cdn.example.com" {
			t.Errorf("Security.CSP = %q, want site default", merged.Security.CSP)
		}
	})

	t.Run("no csp set — security.csp is empty", func(t *testing.T) {
		manifest := &Manifest{Version: 1, Framework: "node"}
		merged := MergeManifests(manifest, &SiteDefaults{}, nil)
		if merged.Security.CSP != "" {
			t.Errorf("Security.CSP = %q, want empty string when none configured", merged.Security.CSP)
		}
	})
}
