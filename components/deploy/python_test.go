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
	cmd := pythonStartCommand(context.Background(), dir)
	if cmd == nil {
		t.Fatal("expected start command")
	}
	if filepath.Base(cmd.Path) != "uv" {
		t.Errorf("cmd.Base = %s, want uv (full: %s)", filepath.Base(cmd.Path), cmd.Path)
	}
	if len(cmd.Args) < 2 || cmd.Args[1] != "run" {
		t.Errorf("expected uv run, got %v", cmd.Args)
	}
}

func TestPythonStartCommand_Fallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.py"), []byte("print(\"hello\")"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := pythonStartCommand(context.Background(), dir)
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
	cmd := pythonStartCommand(context.Background(), dir)
	if cmd == nil {
		t.Fatal("expected start command")
	}
	baseName := filepath.Base(cmd.Path)
	if baseName != "uv" {
		t.Errorf("base name = %s, want uv (full path: %s)", baseName, cmd.Path)
	}
	if len(cmd.Args) < 2 || cmd.Args[1] != "run" {
		t.Errorf("expected uv run, got %v", cmd.Args)
	}
}
