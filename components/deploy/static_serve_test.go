package deploy_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielvm/bigbase/components/deploy"
)

func TestResolvePureStaticServeDir_PublicIndex(t *testing.T) {
	dir := t.TempDir()
	pub := filepath.Join(dir, "public")
	if err := os.MkdirAll(pub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pub, "index.html"), []byte("<h1>firebase</h1>"), 0644); err != nil {
		t.Fatal(err)
	}
	// Firebase-style: no root index.html
	got, err := deploy.ResolvePureStaticServeDir(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != pub {
		t.Fatalf("want public/, got %s", got)
	}
}

func TestResolvePureStaticServeDir_RootIndex(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>root</h1>"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := deploy.ResolvePureStaticServeDir(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("want appRoot, got %s", got)
	}
}

func TestResolvePureStaticServeDir_OutDirHint(t *testing.T) {
	dir := t.TempDir()
	dist := filepath.Join(dir, "dist")
	if err := os.MkdirAll(dist, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<h1>dist</h1>"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(dir, "public"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "public", "index.html"), []byte("<h1>pub</h1>"), 0644)

	got, err := deploy.ResolvePureStaticServeDir(dir, "dist")
	if err != nil {
		t.Fatal(err)
	}
	if got != dist {
		t.Fatalf("want dist/, got %s", got)
	}
}

func TestResolvePureStaticServeDir_FailClosedWithoutIndex(t *testing.T) {
	dir := t.TempDir()
	// Source-tree checkout: package files but no built entrypoint
	_ = os.WriteFile(filepath.Join(dir, "astro.config.mjs"), []byte("export default {}"), 0644)
	_ = os.MkdirAll(filepath.Join(dir, "public"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "public", "favicon.svg"), []byte("<svg/>"), 0644)

	_, err := deploy.ResolvePureStaticServeDir(dir, "dist")
	if err == nil {
		t.Fatal("expected static_output_missing")
	}
	var coded *deploy.CodedError
	if !errors.As(err, &coded) || coded.Code != "static_output_missing" {
		t.Fatalf("want static_output_missing, got %v", err)
	}
}

func TestRequireStaticIndex_FailClosed(t *testing.T) {
	dir := t.TempDir()
	err := deploy.RequireStaticIndex(dir, dir+"/dist", dir+"/build")
	if err == nil {
		t.Fatal("expected static_output_missing")
	}
	var coded *deploy.CodedError
	if !errors.As(err, &coded) || coded.Code != "static_output_missing" {
		t.Fatalf("want static_output_missing, got %v", err)
	}
}

func TestRequireStaticIndex_OK(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>ok</h1>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := deploy.RequireStaticIndex(dir); err != nil {
		t.Fatal(err)
	}
}

func TestGetStartCommand_AdapterNodeBuildIndex(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, `{"name":"web","type":"module"}`)
	if err := os.MkdirAll(filepath.Join(dir, "build"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "build", "index.js"), []byte("export {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := deploy.GetStartCommand(dir); got != "node build/index.js" {
		t.Fatalf("want node build/index.js, got %q", got)
	}
}

func TestFindStaticServeDirAfterNodeBuild_RequiresIndex(t *testing.T) {
	dir := t.TempDir()
	// SSR adapter-node style: build/ exists but no index.html
	if err := os.MkdirAll(filepath.Join(dir, "build"), 0755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "build", "index.js"), []byte("export {}"), 0644)
	if _, ok := deploy.FindStaticServeDirAfterNodeBuild(dir, "build"); ok {
		t.Fatal("adapter-node build/ without index.html must not promote to static")
	}

	dist := filepath.Join(dir, "dist")
	if err := os.MkdirAll(dist, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<h1>ok</h1>"), 0644); err != nil {
		t.Fatal(err)
	}
	got, ok := deploy.FindStaticServeDirAfterNodeBuild(dir, "build")
	if !ok || got != dist {
		t.Fatalf("want dist with index.html, got %q ok=%v", got, ok)
	}
}

func TestFrameworkMode_RootPathWebNotMonorepoListing(t *testing.T) {
	root := t.TempDir()
	web := filepath.Join(root, "web")
	if err := os.MkdirAll(web, 0755); err != nil {
		t.Fatal(err)
	}
	writePkg(t, root, `{"name":"mono","scripts":{"build":"pnpm -C web run build"}}`)
	writePkg(t, web, `{
		"dependencies": {
			"@sveltejs/kit": "^2.0.0",
			"@sveltejs/adapter-node": "^5.0.0"
		}
	}`)
	_ = os.WriteFile(filepath.Join(web, "svelte.config.js"), []byte(`import adapter from '@sveltejs/adapter-node';
export default { kit: { adapter: adapter() } };`), 0644)

	appRoot := deploy.ResolveAppRoot(root, "web")
	got, outDir, err := deploy.ResolveDeployAppType(appRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != deploy.AppNode {
		t.Fatalf("root_path=web must detect node (adapter-node), got %s out=%s", got, outDir)
	}
	// Monorepo root alone must not be treated as a ready-to-serve static site
	rootType, _, err := deploy.ResolveDeployAppType(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if rootType == deploy.AppStatic {
		t.Fatal("monorepo root must not resolve to static (would FileServer-list the checkout)")
	}
}
