package deploy_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielvm/bigbase/components/deploy"
)

func writePkg(t *testing.T, dir string, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFrameworkMode_SvelteKitStatic(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, `{
		"dependencies": {
			"@sveltejs/kit": "^2.0.0",
			"@sveltejs/adapter-static": "^3.0.0"
		}
	}`)
	_ = os.WriteFile(filepath.Join(dir, "svelte.config.js"), []byte(`import adapter from '@sveltejs/adapter-static';
export default { kit: { adapter: adapter() } };`), 0644)
	_ = os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: '9'\n"), 0644)

	fm := deploy.DetectFrameworkMode(dir)
	if fm.Framework != "sveltekit" || fm.Mode != "static" || fm.AppType != deploy.AppStatic {
		t.Fatalf("unexpected mode: %+v", fm)
	}
	if fm.OutDir != "build" {
		t.Fatalf("expected outDir build, got %q", fm.OutDir)
	}
	if fm.PackageMgr != "pnpm" {
		t.Fatalf("expected pnpm, got %s", fm.PackageMgr)
	}
	if got := deploy.DetectAppType(dir); got != deploy.AppStatic {
		t.Fatalf("DetectAppType want static, got %s", got)
	}
}

func TestFrameworkMode_SvelteKitNode(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, `{
		"dependencies": {
			"@sveltejs/kit": "^2.0.0",
			"@sveltejs/adapter-node": "^5.0.0"
		}
	}`)
	_ = os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{"lockfileVersion":3}`), 0644)
	fm := deploy.DetectFrameworkMode(dir)
	if fm.Mode != "server" || fm.AppType != deploy.AppNode || fm.PackageMgr != "npm" {
		t.Fatalf("unexpected: %+v", fm)
	}
}

func TestFrameworkMode_SvelteKitAdapterAutoAmbiguous(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, `{
		"dependencies": {
			"@sveltejs/kit": "^2.0.0",
			"@sveltejs/adapter-auto": "^3.0.0"
		}
	}`)
	fm := deploy.DetectFrameworkMode(dir)
	if !fm.Ambiguous || fm.Mode != "ambiguous" {
		t.Fatalf("expected ambiguous, got %+v", fm)
	}
	_, _, err := deploy.ResolveDeployAppType(dir, "")
	if err == nil {
		t.Fatal("expected framework_mode_ambiguous")
	}
	var coded *deploy.CodedError
	if !errors.As(err, &coded) || coded.Code != "framework_mode_ambiguous" {
		t.Fatalf("want framework_mode_ambiguous, got %v", err)
	}
}

func TestFrameworkMode_AstroStaticDefault(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, `{"dependencies":{"astro":"^4.0.0"}}`)
	fm := deploy.DetectFrameworkMode(dir)
	if fm.Framework != "astro" || fm.Mode != "static" || fm.AppType != deploy.AppStatic || fm.OutDir != "dist" {
		t.Fatalf("unexpected: %+v", fm)
	}
}

func TestFrameworkMode_AstroServer(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, `{"dependencies":{"astro":"^4.0.0","@astrojs/node":"^8.0.0"}}`)
	_ = os.WriteFile(filepath.Join(dir, "astro.config.mjs"), []byte(`export default { output: 'server' }`), 0644)
	fm := deploy.DetectFrameworkMode(dir)
	if fm.Mode != "server" || fm.AppType != deploy.AppNode {
		t.Fatalf("unexpected: %+v", fm)
	}
}

func TestFrameworkMode_AstroHybrid(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, `{"dependencies":{"astro":"^4.0.0","@astrojs/node":"^8.0.0"}}`)
	_ = os.WriteFile(filepath.Join(dir, "astro.config.mjs"), []byte(`export default { output: 'hybrid' }`), 0644)
	_ = os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: '9'\n"), 0644)
	fm := deploy.DetectFrameworkMode(dir)
	if fm.Mode != "hybrid" || fm.AppType != deploy.AppNode || fm.PackageMgr != "pnpm" {
		t.Fatalf("unexpected: %+v", fm)
	}
}

func TestFrameworkMode_RootPathWorkspace(t *testing.T) {
	root := t.TempDir()
	web := filepath.Join(root, "web")
	if err := os.MkdirAll(web, 0755); err != nil {
		t.Fatal(err)
	}
	writePkg(t, web, `{
		"dependencies": {
			"@sveltejs/kit": "^2.0.0",
			"@sveltejs/adapter-static": "^3.0.0"
		}
	}`)
	_ = os.WriteFile(filepath.Join(web, "pnpm-lock.yaml"), []byte("lockfileVersion: '9'\n"), 0644)
	appRoot := deploy.ResolveAppRoot(root, "./web")
	fm := deploy.DetectFrameworkMode(appRoot)
	if fm.AppType != deploy.AppStatic || fm.PackageMgr != "pnpm" {
		t.Fatalf("root_path web detection failed: %+v", fm)
	}
}

func TestResolveDeployAppType_ExplicitMatches(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, `{"dependencies":{"astro":"^4.0.0"}}`)
	got, outDir, err := deploy.ResolveDeployAppType(dir, deploy.AppStatic)
	if err != nil {
		t.Fatal(err)
	}
	if got != deploy.AppStatic || outDir != "dist" {
		t.Fatalf("got %s out=%s", got, outDir)
	}
}
