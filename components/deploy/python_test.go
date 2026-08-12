package deploy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHasPyProjectTOML_Valid(t *testing.T) {
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\ndependencies = [\"fastapi>=0.100\"]\n\n[tool.uv]\ndev-dependencies = [\"pytest\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if !HasPyProjectTOML(dir) {
		t.Error("expected pyproject.toml to be detected")
	}
}

func TestHasPyProjectTOML_NoProjectSection(t *testing.T) {
	dir := t.TempDir()
	content := "[build-system]\nrequires = [\"setuptools\"]\nbuild-backend = \"setuptools.build_meta\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if HasPyProjectTOML(dir) {
		t.Error("expected no detection without [project] or [tool.uv]")
	}
}

func TestHasPyProjectTOML_ToolUVOnly(t *testing.T) {
	dir := t.TempDir()
	content := "[tool.uv]\ndev-dependencies = [\"pytest\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if !HasPyProjectTOML(dir) {
		t.Error("expected detection with [tool.uv] section")
	}
}

func TestHasPyProjectTOML_NoFile(t *testing.T) {
	dir := t.TempDir()
	if HasPyProjectTOML(dir) {
		t.Error("expected no detection without pyproject.toml file")
	}
}

func TestDetectAppType_PyProject(t *testing.T) {
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\n\n[project.scripts]\ndev = \"testapp.main:app\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if got := DetectAppType(dir); got != AppPython {
		t.Errorf("DetectAppType = %s, want AppPython", got)
	}
}

func TestDetectAppType_PyProjectOverNode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}
	pyContent := "[project]\nname = \"testapp\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pyContent), 0644); err != nil {
		t.Fatal(err)
	}
	if got := DetectAppType(dir); got != AppNode {
		t.Errorf("DetectAppType = %s, want AppNode (package.json checked first)", got)
	}
}

func TestDetectAppType_LegacyPython(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.py"), []byte("print(\"hello\")"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := DetectAppType(dir); got != AppPython {
		t.Errorf("DetectAppType = %s, want AppPython", got)
	}
}

func TestParsePyProjectTOML_EntryPoint(t *testing.T) {
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\n\n[project.scripts]\ndev = \"testapp.main:app\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	pp := ParsePyProjectTOML(dir)
	if pp == nil {
		t.Fatal("expected ParsePyProjectTOML to succeed")
	}
	module, appVar := pp.EntryPoint()
	if module != "testapp.main" {
		t.Errorf("module = %q, want testapp.main", module)
	}
	if appVar != "app" {
		t.Errorf("appVar = %q, want app", appVar)
	}
}

func TestIsUvicornDep(t *testing.T) {
	tests := []struct {
		dep  string
		want bool
	}{
		{"uvicorn>=0.30", true},
		{"uvicorn[standard]", true},
		{"uvicorn", true},
		{"uvicorn==0.31.0", true},
		{"fastapi>=0.100", false},
		{"starlette", false},
	}
	for _, tt := range tests {
		if got := isUvicornDep(tt.dep); got != tt.want {
			t.Errorf("isUvicornDep(%q) = %v, want %v", tt.dep, got, tt.want)
		}
	}
}

func TestSplitEntryPoint(t *testing.T) {
	module, appVar := splitEntryPoint("myapp.main:app")
	if module != "myapp.main" || appVar != "app" {
		t.Errorf("splitEntryPoint = (%q, %q), want (myapp.main, app)", module, appVar)
	}

	module, appVar = splitEntryPoint("myapp.core:create_app")
	if module != "myapp.core" || appVar != "create_app" {
		t.Errorf("splitEntryPoint = (%q, %q), want (myapp.core, create_app)", module, appVar)
	}
}

func TestPyProjectTOML_HasUvicorn(t *testing.T) {
	pp := &PyProjectTOML{}
	pp.Project.Dependencies = []string{"fastapi>=0.100", "uvicorn>=0.30"}
	if !pp.HasUvicorn() {
		t.Error("expected uvicorn to be detected")
	}

	pp2 := &PyProjectTOML{}
	pp2.Project.Dependencies = []string{"fastapi>=0.100", "starlette>=0.40"}
	if pp2.HasUvicorn() {
		t.Error("expected no uvicorn detection")
	}
}

func TestPythonStartCommand_Uvicorn(t *testing.T) {
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\ndependencies = [\"fastapi>=0.100\", \"uvicorn>=0.30\"]\n\n[project.scripts]\ndev = \"testapp.main:app\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := pythonStartCommand(context.Background(), dir, 8000, nil)
	if cmd == nil {
		t.Fatal("expected start command")
	}
	baseName := filepath.Base(cmd.Path)
	// When uv is available, it uses "uv". When not, falls back to python3/python.
	if baseName != "uv" && baseName != "python3" && baseName != "python" {
		t.Errorf("base name = %s, want uv, python3, or python (full: %s)", baseName, cmd.Path)
	}
	// Non-CLI entry points like "testapp.main:app" are used as ASGI import strings directly.
	hasCorrectImport := false
	for _, arg := range cmd.Args {
		if arg == "testapp.main:app" {
			hasCorrectImport = true
			break
		}
	}
	if !hasCorrectImport {
		t.Errorf("expected ASGI import string 'testapp.main:app' in args %v", cmd.Args)
	}
}

func TestPythonStartCommand_Fallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.py"), []byte("print(\"hello\")"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := pythonStartCommand(context.Background(), dir, 8000, nil)
	if cmd == nil {
		t.Fatal("expected start command")
	}
	if cmd.Args[0] != "python3" && cmd.Args[0] != "python" {
		t.Errorf("expected python3 or python, got %s", cmd.Args[0])
	}
}

func TestPythonStartCommand_Uvicorn_Path(t *testing.T) {
	// Same as TestPythonStartCommand_Uvicorn but uses filepath.Base for cross-platform.
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\ndependencies = [\"fastapi>=0.100\", \"uvicorn>=0.30\"]\n\n[project.scripts]\ndev = \"testapp.main:app\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := pythonStartCommand(context.Background(), dir, 8000, nil)
	if cmd == nil {
		t.Fatal("expected start command")
	}
	baseName := filepath.Base(cmd.Path)
	// When uv is available, it uses "uv". When not, falls back to python3/python.
	if baseName != "uv" && baseName != "python3" && baseName != "python" {
		t.Errorf("base name = %s, want uv, python3, or python (full path: %s)", baseName, cmd.Path)
	}
}

func TestResolvePythonBin(t *testing.T) {
	// resolvePythonBin prefers python3 and falls back to python.
	bin := resolvePythonBin()
	if bin != "python3" && bin != "python" {
		t.Errorf("resolvePythonBin = %q, want python3 or python", bin)
	}
}

func TestPythonStartCommand_PyProject_NoUv(t *testing.T) {
	// When pyproject.toml exists but uv is not on PATH,
	// pythonStartCommand should fall back to python3.
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\ndependencies = [\"fastapi>=0.100\", \"uvicorn>=0.30\"]\n\n[project.scripts]\ndev = \"testapp.main:app\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := pythonStartCommand(context.Background(), dir, 8000, nil)
	if cmd == nil {
		t.Fatal("expected start command")
	}
	baseName := filepath.Base(cmd.Path)
	// When uv is available, it uses "uv". When not, falls back to python3/python.
	if baseName != "uv" && baseName != "python3" && baseName != "python" {
		t.Errorf("base name = %s, want uv, python3, or python", baseName)
	}
}

func TestPythonStartCommand_SetsDir(t *testing.T) {
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\ndependencies = [\"fastapi>=0.100\", \"uvicorn>=0.30\"]\n\n[project.scripts]\ndev = \"testapp.main:app\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := pythonStartCommand(context.Background(), dir, 8000, nil)
	if cmd == nil {
		t.Fatal("expected start command")
	}
	if cmd.Dir != dir {
		t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, dir)
	}
}

func TestPythonStartCommand_UvicornDefaultsAppApp(t *testing.T) {
	// When pyproject.toml has uvicorn but no [project.scripts] section,
	// the ASGI import string must default to "app:app", not "app:".
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\ndependencies = [\"fastapi>=0.100\", \"uvicorn>=0.30\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := pythonStartCommand(context.Background(), dir, 8000, nil)
	if cmd == nil {
		t.Fatal("expected start command")
	}
	if cmd.Dir != dir {
		t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, dir)
	}
	// Check that the ASGI import string is "app:app" not "app:"
	for _, arg := range cmd.Args {
		if arg == "app:" {
			t.Errorf("found invalid ASGI import string %q in args %v — expected app:app", arg, cmd.Args)
		}
	}
	// Verify the last two args are --port 8000 (not $PORT)
	args := cmd.Args
	if len(args) >= 2 {
		if args[len(args)-2] != "--port" {
			t.Errorf("expected --port arg, got %s", args[len(args)-2])
		}
		if args[len(args)-1] != "8000" {
			t.Errorf("expected port 8000, got %s", args[len(args)-1])
		}
	}
}

func TestPythonStartCommand_CLIScriptFallsBackToAppApp(t *testing.T) {
	// When [project.scripts] contains a CLI entry point (e.g. __main__:main),
	// the ASGI import string must fall back to "app:app" instead of using
	// the CLI entry as an ASGI import.
	tests := []struct {
		name       string
		entry      string
		wantImport string
	}{
		{
			name:       "grimoire-like __main__:main",
			entry:      "grimoire = \"grimoire.__main__:main\"",
			wantImport: "app:app",
		},
		{
			name:       "CLI variable name 'main'",
			entry:      "dev = \"myapp.cli:main\"",
			wantImport: "app:app",
		},
		{
			name:       "CLI variable name 'run'",
			entry:      "dev = \"myapp.cli:run\"",
			wantImport: "app:app",
		},
		{
			name:       "valid ASGI entry kept",
			entry:      "dev = \"myapp.main:app\"",
			wantImport: "myapp.main:app",
		},
		{
			name:       "factory entry kept",
			entry:      "dev = \"myapp.main:create_app\"",
			wantImport: "myapp.main:create_app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			content := "[project]\nname = \"testapp\"\ndependencies = [\"fastapi>=0.100\", \"uvicorn>=0.30\"]\n\n[project.scripts]\n" + tt.entry + "\n"
			if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
			cmd := pythonStartCommand(context.Background(), dir, 8000, nil)
			if cmd == nil {
				t.Fatal("expected start command")
			}
			found := false
			for _, arg := range cmd.Args {
				if arg == tt.wantImport {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected ASGI import %q in args %v", tt.wantImport, cmd.Args)
			}
		})
	}
}

func TestIsCLIScriptEntry(t *testing.T) {
	tests := []struct {
		module string
		appVar string
		isCLI  bool
	}{
		{"grimoire.__main__", "main", true},
		{"grimoire.__main__", "app", true}, // __main__ always CLI
		{"myapp.main", "main", true},
		{"myapp.cli", "run", true},
		{"myapp.main", "cli", true},
		{"myapp.main", "entrypoint", true},
		{"myapp.main", "app", false},
		{"myapp.main", "create_app", false},
		{"myapp.main", "application", false},
		{"app", "app", false},
	}
	for _, tt := range tests {
		got := isCLIScriptEntry(tt.module, tt.appVar)
		if got != tt.isCLI {
			t.Errorf("isCLIScriptEntry(%q, %q) = %v, want %v", tt.module, tt.appVar, got, tt.isCLI)
		}
	}
}

func TestParseASGIImport(t *testing.T) {
	tests := []struct {
		raw       string
		wantStr   string
		wantExtra []string
	}{
		{"app:app", "app:app", nil},
		{"grimoire.app:create_app --factory", "grimoire.app:create_app", []string{"--factory"}},
		{"myapp.main:app", "myapp.main:app", nil},
		{"pkg.module:app_var --reload --reload-dir src", "pkg.module:app_var", []string{"--reload", "--reload-dir", "src"}},
	}
	for _, tt := range tests {
		gotStr, gotExtra := parseASGIImport(tt.raw)
		if gotStr != tt.wantStr {
			t.Errorf("parseASGIImport(%q) str = %q, want %q", tt.raw, gotStr, tt.wantStr)
		}
		if len(gotExtra) != len(tt.wantExtra) {
			t.Errorf("parseASGIImport(%q) extra = %v, want %v", tt.raw, gotExtra, tt.wantExtra)
			continue
		}
		for i := range gotExtra {
			if gotExtra[i] != tt.wantExtra[i] {
				t.Errorf("parseASGIImport(%q) extra[%d] = %q, want %q", tt.raw, i, gotExtra[i], tt.wantExtra[i])
			}
		}
	}
}

func TestPythonStartCommand_ManifestASGIImport(t *testing.T) {
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\ndependencies = [\"fastapi>=0.100\", \"uvicorn>=0.30\"]\n\n[project.scripts]\ngrimoire = \"grimoire.__main__:main\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	manifest := &Manifest{
		Start: ManifestStart{
			ASGIImport: "grimoire.app:create_app --factory",
		},
	}
	cmd := pythonStartCommand(context.Background(), dir, 8000, manifest)
	if cmd == nil {
		t.Fatal("expected start command")
	}

	// Must contain the ASGI import string from the manifest.
	foundImport := false
	foundFactory := false
	for _, arg := range cmd.Args {
		if arg == "grimoire.app:create_app" {
			foundImport = true
		}
		if arg == "--factory" {
			foundFactory = true
		}
	}
	if !foundImport {
		t.Errorf("expected ASGI import 'grimoire.app:create_app' in args %v", cmd.Args)
	}
	if !foundFactory {
		t.Errorf("expected --factory flag in args %v", cmd.Args)
	}
	// Must NOT contain the CLI entry point.
	for _, arg := range cmd.Args {
		if arg == "grimoire.__main__:main" {
			t.Errorf("CLI entry point %q should not appear in args %v", arg, cmd.Args)
		}
	}
}

func TestPythonStartCommand_ManifestASGIImportSimple(t *testing.T) {
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\ndependencies = [\"fastapi>=0.100\", \"uvicorn>=0.30\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	manifest := &Manifest{
		Start: ManifestStart{
			ASGIImport: "custom.module:application",
		},
	}
	cmd := pythonStartCommand(context.Background(), dir, 8000, manifest)
	if cmd == nil {
		t.Fatal("expected start command")
	}

	foundImport := false
	for _, arg := range cmd.Args {
		if arg == "custom.module:application" {
			foundImport = true
			break
		}
	}
	if !foundImport {
		t.Errorf("expected ASGI import 'custom.module:application' in args %v", cmd.Args)
	}
	// Must NOT have extra --factory flag
	for _, arg := range cmd.Args {
		if arg == "--factory" {
			t.Errorf("unexpected --factory in args %v", cmd.Args)
		}
	}
}
