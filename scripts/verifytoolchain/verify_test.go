package verifytoolchain

import (
	"strings"
	"testing"
)

func TestVerifyRequiredMissingFails(t *testing.T) {
	c := &Contract{
		Tools: ContractTools{
			Required: map[string]ToolSpec{
				"node":   {Min: "20.19"},
				"python3": {Min: "3.11"},
			},
			Optional: map[string]ToolSpec{},
		},
	}
	// node present+ok, python3 MISSING.
	lookup := func(name string) (string, string, error) {
		switch name {
		case "node":
			return "/usr/bin/node", "v22.3.0", nil
		default:
			return "", "", errNotFound
		}
	}
	results, verr := Verify(c, lookup)
	if verr == nil {
		t.Fatal("Verify: expected error for missing required tool, got nil")
	}
	if !strings.Contains(verr.Error(), "TOOLCHAIN_MISSING") {
		t.Errorf("Verify error = %q, want it to contain TOOLCHAIN_MISSING", verr.Error())
	}
	if !strings.Contains(verr.Error(), "python3") {
		t.Errorf("Verify error = %q, want it to name python3", verr.Error())
	}
	// node should be marked OK.
	var nodeRes *Result
	for i := range results {
		if results[i].Tool == "node" {
			nodeRes = &results[i]
		}
	}
	if nodeRes == nil || !nodeRes.OK {
		t.Errorf("expected node OK, got %+v", nodeRes)
	}
}

func TestVerifyVersionTooLowFails(t *testing.T) {
	c := &Contract{
		Tools: ContractTools{
			Required: map[string]ToolSpec{"node": {Min: "20.19"}},
		},
	}
	lookup := func(name string) (string, string, error) {
		return "/usr/bin/node", "v18.0.0", nil // Node 18 — too old (the recurring bug)
	}
	_, verr := Verify(c, lookup)
	if verr == nil {
		t.Fatal("Verify: expected error for below-floor version, got nil")
	}
	if !strings.Contains(verr.Error(), "TOOLCHAIN_VERSION_TOO_LOW") {
		t.Errorf("Verify error = %q, want TOOLCHAIN_VERSION_TOO_LOW", verr.Error())
	}
}

func TestVerifyOptionalMissingIsOK(t *testing.T) {
	c := &Contract{
		Tools: ContractTools{
			Required: map[string]ToolSpec{"node": {Min: "20.19"}},
			Optional: map[string]ToolSpec{"corepack": {}, "yarn": {}},
		},
	}
	lookup := func(name string) (string, string, error) {
		if name == "node" {
			return "/usr/bin/node", "v22.3.0", nil
		}
		return "", "", errNotFound // optional tools absent
	}
	results, verr := Verify(c, lookup)
	if verr != nil {
		t.Fatalf("Verify: optional missing must not fail; got %v", verr)
	}
	for _, r := range results {
		if r.Required {
			continue
		}
		if !r.OK || r.Skip == "" {
			t.Errorf("optional tool %s should be OK+skipped, got %+v", r.Tool, r)
		}
	}
}

func TestVerifyPresenceOnlyContract(t *testing.T) {
	// min = "" means presence only — a tool present with an unparseable
	// --version output still passes.
	c := &Contract{
		Tools: ContractTools{
			Required: map[string]ToolSpec{"git": {}}, // no min
		},
	}
	lookup := func(name string) (string, string, error) {
		return "/usr/bin/git", "git version some-weird-output-no-numbers", nil
	}
	results, verr := Verify(c, lookup)
	if verr != nil {
		t.Fatalf("presence-only contract with unparseable version should pass; got %v", verr)
	}
	if !results[0].OK {
		t.Errorf("expected OK for presence-only, got %+v", results[0])
	}
}

func TestVerifyPresentButVersionCmdErrorsStillCountsAsPresent(t *testing.T) {
	// Regression guard for the `go` bug: the tool IS on PATH (path != "") but
	// its version subcommand returns a non-nil error (go --version is invalid
	// syntax). Presence must be decided by LookPath (path != ""), not by the
	// version subcommand's exit status. Here the version output is still
	// parseable, so the floor is checked normally and the tool passes.
	c := &Contract{
		Tools: ContractTools{
			Required: map[string]ToolSpec{"go": {Min: "1.26"}},
		},
	}
	lookup := func(name string) (string, string, error) {
		// path set (present), but versionErr non-nil — mirrors `go --version`
		// before we taught defaultLookup to use `go version`.
		return "/usr/local/go/bin/go", "go version go1.26.3 darwin/arm64", errVersionCmdFailed
	}
	results, verr := Verify(c, lookup)
	if verr != nil {
		t.Fatalf("present tool with failing version subcommand should still pass; got %v", verr)
	}
	if !results[0].Present {
		t.Errorf("expected Present=true when path != \"\", got %+v", results[0])
	}
}

func TestVersionArgsRegistry(t *testing.T) {
	// `go` is the one tool whose --version flag is invalid; it must use
	// `go version`. Everything else defaults to --version.
	if got := versionArgs["go"]; len(got) != 1 || got[0] != "version" {
		t.Errorf("versionArgs[go] = %v, want [version]", got)
	}
	if _, ok := versionArgs["node"]; ok {
		t.Errorf("node should fall through to the --version default, not be in the map")
	}
}

func TestLoadContractRealFile(t *testing.T) {
	// Loads the actual toolchain.toml shipped with the repo and asserts the
	// required set matches what the deploy runtime invokes. This is the
	// regression guard: if someone adds an exec.Command in components/deploy
	// but forgets to add the tool here, a future assertion (below) will catch
	// it once extended — for now we assert the known-required set is stable.
	c, err := LoadContract("../../toolchain.toml")
	if err != nil {
		t.Fatalf("LoadContract: %v", err)
	}
	want := []string{"git", "go", "node", "npm", "pip", "pnpm", "python3", "uv"}
	got := c.RequiredTools()
	if len(got) != len(want) {
		t.Fatalf("required tools = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("required[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
