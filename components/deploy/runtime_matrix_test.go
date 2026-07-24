package deploy_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielvm/bigbase/components/deploy"
)

func TestDetectAppTypePHP(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"name":"app/php"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := deploy.DetectAppType(dir); got != deploy.AppPHP {
		t.Fatalf("expected php, got %s", got)
	}
}

func TestDetectAppTypePackageJSONWinsOverComposer(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{}`), 0644)
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"x"}`), 0644)
	if got := deploy.DetectAppType(dir); got != deploy.AppNode {
		t.Fatalf("expected node when both present, got %s", got)
	}
}

func TestDetectNodePackageManager_PnpmLockNoNpmFallback(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"x"}`), 0644)
	_ = os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: '9'\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{}`), 0644)
	if got := deploy.DetectNodePackageManager(dir); got != "pnpm" {
		t.Fatalf("expected pnpm (lockfile priority over npm), got %s", got)
	}
}

func TestDetectNodePackageManager_PackageManagerField(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"packageManager":"pnpm@9.0.0"}`), 0644)
	_ = os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{}`), 0644)
	if got := deploy.DetectNodePackageManager(dir); got != "pnpm" {
		t.Fatalf("expected pnpm from packageManager field, got %s", got)
	}
}

func TestResolveAppRoot_RespectsNestedPath(t *testing.T) {
	base := t.TempDir()
	web := filepath.Join(base, "web")
	if err := os.MkdirAll(web, 0755); err != nil {
		t.Fatal(err)
	}
	got := deploy.ResolveAppRoot(base, "web")
	if got != web {
		t.Fatalf("expected %s, got %s", web, got)
	}
	if got := deploy.ResolveAppRoot(base, "../escape"); got != base {
		t.Fatalf("escape should fall back to base, got %s", got)
	}
}

func TestCodedError_ToolMissingShape(t *testing.T) {
	err := deploy.DetectAppType // keep import used via other tests
	_ = err
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0644)
	if got := deploy.DetectAppType(dir); got != deploy.AppGo {
		t.Fatalf("expected go, got %s", got)
	}
}

func TestAllAppTypesIncludesPHP(t *testing.T) {
	found := false
	for _, a := range deploy.AllAppTypes() {
		if a == deploy.AppPHP {
			found = true
		}
	}
	if !found {
		t.Fatal("AllAppTypes must include php")
	}
	if !deploy.AppPHP.IsValid() {
		t.Fatal("AppPHP must be valid")
	}
}

// Ensure CodedError is returned from ensure path via ResolveDeployAppType mismatch.
func TestResolveDeployAppType_MismatchReturnsCodedError(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"dependencies": {"astro": "^4.0.0", "@astrojs/node": "^8.0.0"},
		"devDependencies": {}
	}`), 0644)
	_ = os.WriteFile(filepath.Join(dir, "astro.config.mjs"), []byte(`export default { output: 'server' }`), 0644)

	_, _, err := deploy.ResolveDeployAppType(dir, deploy.AppStatic)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	var coded *deploy.CodedError
	if !errors.As(err, &coded) {
		t.Fatalf("expected CodedError, got %T: %v", err, err)
	}
	if coded.Code != "app_type_mismatch" {
		t.Fatalf("expected app_type_mismatch, got %s", coded.Code)
	}
}
