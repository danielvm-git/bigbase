package deploy

// Regression tests (e81s06) guarding already-fixed defects so they cannot
// silently return. Each test asserts an existing guard's behavior; no
// production code is exercised beyond the public deploy API.

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// TestAppTypeIsValid guards AppType.IsValid(): every known type is valid and
// unknown/empty/differently-cased inputs are rejected. Prevents a silent
// widening or narrowing of the accepted runtime set.
func TestAppTypeIsValid(t *testing.T) {
	valid := []AppType{AppNode, AppGo, AppPython, AppPHP, AppStatic, AppStaticSidecar}
	for _, at := range valid {
		if !at.IsValid() {
			t.Errorf("AppType %q should be valid", at)
		}
	}

	// AllAppTypes() must stay in sync with IsValid().
	for _, at := range AllAppTypes() {
		if !at.IsValid() {
			t.Errorf("AllAppTypes returned %q which IsValid() rejects", at)
		}
	}
	if got := len(AllAppTypes()); got != len(valid) {
		t.Errorf("AllAppTypes length = %d, want %d", got, len(valid))
	}

	invalid := []AppType{"", "ruby", "rust", "java", "unknown", "NODE", "Static", " node"}
	for _, at := range invalid {
		if at.IsValid() {
			t.Errorf("AppType %q should be invalid", at)
		}
	}
}

// TestErrorSentinels guards the deploy sentinel errors so callers can keep
// using errors.Is instead of fragile string comparisons.
func TestErrorSentinels(t *testing.T) {
	if !errors.Is(fmt.Errorf("lookup: %w", ErrRepoNotFound), ErrRepoNotFound) {
		t.Error("wrapped ErrRepoNotFound must match via errors.Is")
	}
	if !errors.Is(fmt.Errorf("lookup: %w", ErrDeploymentNotFound), ErrDeploymentNotFound) {
		t.Error("wrapped ErrDeploymentNotFound must match via errors.Is")
	}
	// The two sentinels must remain distinct.
	if errors.Is(ErrRepoNotFound, ErrDeploymentNotFound) {
		t.Error("ErrRepoNotFound and ErrDeploymentNotFound must not be identical")
	}
	// A plain error with the same text must NOT match the sentinel — that is
	// exactly the fragile-string-comparison behavior these sentinels replaced.
	if errors.Is(errors.New("repo not found"), ErrRepoNotFound) {
		t.Error("a same-text non-sentinel error must not match ErrRepoNotFound")
	}
}

// TestManifestPathTraversal guards LoadManifestPath against CWE-22: absolute
// paths and paths escaping the project directory must be rejected before any
// file read.
func TestManifestPathTraversal(t *testing.T) {
	dir := t.TempDir()

	traversal := []string{
		"/etc/passwd",         // absolute
		"../../etc/passwd",    // parent escape
		"../" + filepath.Base(dir) + "-sibling/x.yaml", // sibling escape
		"sub/../../escape.yaml",                        // normalized escape
	}
	for _, p := range traversal {
		if _, err := LoadManifestPath(dir, p); err == nil {
			t.Errorf("LoadManifestPath(%q) should reject path traversal, got nil error", p)
		} else if !strings.Contains(err.Error(), "relative") && !strings.Contains(err.Error(), "escapes") {
			t.Errorf("LoadManifestPath(%q) rejected with unexpected error: %v", p, err)
		}
	}

	// A safe relative path that simply doesn't exist must be a clean miss
	// (nil, nil) — callers fall back to auto-detection.
	if m, err := LoadManifestPath(dir, "bigbase.yaml"); err != nil || m != nil {
		t.Errorf("LoadManifestPath(safe, missing) = (%v, %v), want (nil, nil)", m, err)
	}
}
