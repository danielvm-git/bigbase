package verifytoolchain

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// errNotFound is the sentinel returned by a LookupVersion when the binary is
// not on PATH. The default lookup returns the exec.ErrNotFound wrapped error;
// tests use this bare sentinel via the seam.
var errNotFound = errors.New("tool not found on PATH")

// errVersionCmdFailed is the sentinel a test fake returns to model "tool is
// present on PATH but its version subcommand exited non-zero" (e.g. the
// historical `go --version` invalid-flag case). Production defaultLookup
// returns the real exec.ExitError here.
var errVersionCmdFailed = errors.New("version subcommand failed")

// Result is the outcome of checking a single tool.
type Result struct {
	Tool       string // e.g. "node"
	Required   bool   // true = required, false = optional
	Present    bool   // LookPath succeeded
	Path       string // resolved binary path (when present)
	FoundStr   string // raw --version output (when present)
	Found      *Version
	FloorStr   string // raw min from contract ("" = presence only)
	Floor      *Version
	OK         bool   // passes contract (present + meets floor)
	Skip       string // non-empty => skipped with reason (e.g. optional missing)
	Diagnostic string // human-readable detail for failures
}

// LookupVersion is the seam for locating a binary and capturing its version
// output. The contract:
//
//	path  == ""           => tool NOT on PATH (truly missing)
//	path  != ""           => tool IS present; output may still be empty or
//	                         versionErr non-nil if --version misbehaved
//	                         (presence is decided by LookPath, not --version)
//	output                => raw --version (or equivalent) stdout+stderr
//	versionErr            => non-nil only when the version subcommand itself
//	                         failed to run/exit 0 (e.g. go uses `go version`)
//
// Decoupling presence (LookPath) from the version subcommand is deliberate:
// `go --version` is invalid syntax (Go uses `go version`), and we must not
// report a present tool as missing just because its version flag differs.
type LookupVersion func(name string) (path string, output string, versionErr error)

// versionArgs maps a tool to the args that print its version. Most CLIs accept
// `--version`; the exceptions are listed explicitly. A tool absent from this
// map defaults to []string{"--version"}.
var versionArgs = map[string][]string{
	// `go --version` is invalid (Go's flag parser rejects it). `go version`
	// prints "go version go1.26.3 darwin/arm64".
	"go": {"version"},
}

// defaultLookup is the production lookup: LookPath then run the tool's version
// subcommand. Returns path even when the version subcommand errors, so callers
// decide presence from path != "" alone.
func defaultLookup(name string) (string, string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		// Truly missing — empty path is the only signal checkTool needs.
		return "", "", err
	}
	args := versionArgs[name]
	if args == nil {
		args = []string{"--version"}
	}
	// Combine stdout+stderr: some CLIs (e.g. older php) print version to stderr
	// and some print a banner to stderr alongside the version on stdout.
	out, runErr := exec.Command(name, args...).CombinedOutput()
	return path, string(out), runErr
}

// Verify checks the contract against the host using lookup. Returns one Result
// per declared tool plus an aggregated error if any REQUIRED tool failed
// (optional tools that are missing never cause an error).
func Verify(c *Contract, lookup LookupVersion) ([]Result, error) {
	if lookup == nil {
		lookup = defaultLookup
	}
	var results []Result

	// Required tools first (so failures surface first), then optional.
	for _, name := range c.RequiredTools() {
		spec := c.Tools.Required[name]
		results = append(results, checkTool(name, spec, true, lookup))
	}
	for _, name := range c.OptionalTools() {
		spec := c.Tools.Optional[name]
		results = append(results, checkTool(name, spec, false, lookup))
	}

	var failed []string
	var missing []string
	for _, r := range results {
		if !r.Required {
			continue
		}
		if !r.OK && r.Skip == "" {
			failed = append(failed, r.Tool)
			if !r.Present {
				missing = append(missing, r.Tool)
			}
		}
	}
	if len(failed) == 0 {
		return results, nil
	}
	if len(missing) > 0 {
		return results, fmt.Errorf("TOOLCHAIN_MISSING: required tools not on PATH: %s", strings.Join(missing, ", "))
	}
	return results, fmt.Errorf("TOOLCHAIN_VERSION_TOO_LOW: required tools below declared floor: %s", strings.Join(failed, ", "))
}

// checkTool runs the lookup + version comparison for one tool entry.
func checkTool(name string, spec ToolSpec, required bool, lookup LookupVersion) Result {
	r := Result{Tool: name, Required: required, FloorStr: spec.Min}
	if spec.Min != "" {
		if fv, err := parseNumeric(spec.Min); err == nil {
			r.Floor = &fv
		}
	}

	path, output, _ := lookup(name)
	if path == "" {
		// LookPath failed — the binary is genuinely not on PATH.
		if !required {
			// Optional tool missing is fine — report as skipped, never a failure.
			r.Skip = "optional tool not installed"
			r.OK = true
			return r
		}
		r.Diagnostic = fmt.Sprintf("not found on PATH (install %s on the deploy host)", name)
		return r
	}
	// Present on PATH. Version subcommand may still have errored or produced
	// unparseable output; that is handled below without changing presence.
	r.Present = true
	r.Path = path
	r.FoundStr = strings.TrimSpace(strings.SplitN(output, "\n", 2)[0])

	parsed, perr := ParseVersion(output)
	if perr != nil {
		// Present but version unparseable. If the contract has no floor
		// (presence-only), that's OK. Otherwise we can't prove the floor is
		// met, so fail with a clear diagnostic.
		if r.Floor == nil {
			r.OK = true
			return r
		}
		r.Diagnostic = fmt.Sprintf("present at %s but version could not be parsed from %q (min=%s)", path, r.FoundStr, spec.Min)
		return r
	}
	r.Found = &parsed

	if r.Floor == nil {
		// Presence-only contract — present is enough.
		r.OK = true
		return r
	}
	if parsed.AtLeast(*r.Floor) {
		r.OK = true
		return r
	}
	r.Diagnostic = fmt.Sprintf("%s < min %s (at %s)", parsedString(parsed), spec.Min, path)
	return r
}

func parsedString(v Version) string {
	s := fmt.Sprintf("%d.%d", v.Major, v.Minor)
	if v.HasPatch {
		s += fmt.Sprintf(".%d", v.Patch)
	}
	if v.PreRelease != "" {
		s += "-" + v.PreRelease
	}
	return s
}

// sortStrings is a local helper to avoid importing sort at the top of multiple
// files inconsistently; keeps the package import list tidy.
func sortStrings(s []string) {
	sort.Strings(s)
}
