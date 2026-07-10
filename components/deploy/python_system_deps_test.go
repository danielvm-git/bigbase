package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPyProjectTOML_SystemDeps(t *testing.T) {
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\n\n[tools.system_deps]\ndeps = [\"git\", \"curl\", \"ssh\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	pp := ParsePyProjectTOML(dir)
	if pp == nil {
		t.Fatal("expected ParsePyProjectTOML to succeed")
	}
	deps := pp.SystemDeps()
	if len(deps) != 3 {
		t.Errorf("expected 3 system deps, got %d: %v", len(deps), deps)
	}
}

func TestPyProjectTOML_SystemDeps_Allowlist(t *testing.T) {
	dir := t.TempDir()
	content := "[project]\nname = \"testapp\"\n\n[tools.system_deps]\ndeps = [\"git\", \"evil-package\", \"curl\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	pp := ParsePyProjectTOML(dir)
	if pp == nil {
		t.Fatal("expected ParsePyProjectTOML to succeed")
	}
	deps := pp.SystemDeps()
	if len(deps) != 2 {
		t.Errorf("expected 2 allowed deps, got %d: %v", len(deps), deps)
	}
	for _, dep := range deps {
		if dep != "git" && dep != "curl" {
			t.Errorf("unexpected dep: %s", dep)
		}
	}
}

func TestAllowedSystemDep(t *testing.T) {
	if !allowedSystemDep("git") {
		t.Error("git should be allowed")
	}
	if allowedSystemDep("malicious-package") {
		t.Error("malicious-package should not be allowed")
	}
}

func TestSpec_BackgroundProcesses(t *testing.T) {
	spec := Spec{
		DeployID:            "test-1",
		BackgroundProcesses: []string{"python worker.py", "celery -A tasks worker"},
	}
	if len(spec.BackgroundProcesses) != 2 {
		t.Errorf("expected 2 background processes, got %d", len(spec.BackgroundProcesses))
	}
}
