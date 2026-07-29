// Package verifytoolchain implements the declared-toolchain contract verifier
// for issue #179. It reads toolchain.toml, checks each declared tool is on
// PATH, and compares the installed version against the contract's minimum.
//
// The version parsing and comparison logic (this file) is deliberately pure
// and side-effect free so it can be unit-tested without any real binaries on
// PATH.
package verifytoolchain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version is a parsed semantic-ish version (major.minor.patch), where minor
// and patch are optional. Only the components present in both the parsed
// version and the minimum are compared.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	HasMinor   bool
	HasPatch   bool
	PreRelease string // e.g. "rc.1" for "1.2.3-rc.1"; "" if none
}

// versionRe captures a leading numeric version from arbitrary CLI output.
// Examples it matches (the captured group is the numeric version):
//
//	"v20.19.0"                       -> "20.19.0"
//	"Node.js v22.3.0."               -> "22.3.0"
//	"go version go1.26.3 darwin/arm" -> "1.26.3"
//	"Python 3.13.13"                 -> "3.13.13"
//	"npm 11.16.0"                    -> "11.16.0"
//	"uv 0.11.14 (abc1234 ...)"       -> "0.11.14"
//	"git version 2.54.0"             -> "2.54.0"
//	"pip 26.1.1 from ..."            -> "26.1.1"
//	"pnpm 10.0.0"                    -> "10.0.0"
//	"PHP 8.3.6 (cli)"                -> "8.3.6"
//
// It intentionally tolerates a leading non-digit prefix (e.g. "go", "v",
// "Node.js v") so we can parse a wide variety of --version outputs without a
// per-tool adapter table.
var versionRe = regexp.MustCompile(`(\d+\.\d+(?:\.\d+)?(?:-[0-9A-Za-z.-]+)?)`)

// ParseVersion extracts the first version-like token from raw CLI output and
// parses it into a Version. Returns an error if no version-like token is
// found. Minor is always required (we require at least major.minor so that a
// floor like "20.19" compares meaningfully); patch is optional.
func ParseVersion(raw string) (Version, error) {
	m := versionRe.FindStringSubmatch(raw)
	if m == nil {
		return Version{}, fmt.Errorf("no version-like token in output: %q", raw)
	}
	return parseNumeric(m[1])
}

// parseNumeric parses a string like "23", "1.26", "1.26.3", "1.26.3-rc.1" into
// a Version. A single component (e.g. "23" for pip's floor) is accepted and
// treated as major-only (minor/patch unconstrained). Installed versions from
// CLI output always have at least major.minor because the versionRe extractor
// requires it; floors may be more permissive.
func parseNumeric(s string) (Version, error) {
	preRelease := ""
	main := s
	if i := strings.Index(s, "-"); i >= 0 {
		main = s[:i]
		preRelease = s[i+1:]
	}
	parts := strings.Split(main, ".")
	if len(parts) < 1 || parts[0] == "" {
		return Version{}, fmt.Errorf("version %q needs at least a major component", s)
	}
	v := Version{PreRelease: preRelease}
	var err error
	if v.Major, err = strconv.Atoi(parts[0]); err != nil {
		return Version{}, fmt.Errorf("invalid major in %q: %w", s, err)
	}
	if len(parts) >= 2 {
		if v.Minor, err = strconv.Atoi(parts[1]); err != nil {
			return Version{}, fmt.Errorf("invalid minor in %q: %w", s, err)
		}
		v.HasMinor = true
	}
	if len(parts) >= 3 {
		if v.Patch, err = strconv.Atoi(parts[2]); err != nil {
			return Version{}, fmt.Errorf("invalid patch in %q: %w", s, err)
		}
		v.HasPatch = true
	}
	return v, nil
}

// MustParse is a convenience for tests.
func MustParse(s string) Version {
	v, err := parseNumeric(s)
	if err != nil {
		panic(err)
	}
	return v
}

// Compare returns -1, 0, or +1 as v compares less than, equal to, or greater
// than other. Comparison follows semver precedence rules, scoped to the
// components BOTH sides declare:
//
//  1. Compare major (always).
//  2. Compare minor only if BOTH sides declare it. A floor of "20.19" vs an
//     installed "20.19.5" compares on minor and ignores the floor-absent
//     patch — the contract floor must not be over-constrained by a component
//     it never specified.
//  3. Compare patch only if BOTH sides declare it.
//  4. A version WITHOUT a pre-release tag is GREATER than one WITH a
//     pre-release tag at the same numeric components (1.0.0 > 1.0.0-rc.1).
//     Two pre-releases compare lexically by tag.
//
// Rationale: the contract floor is a minimum bar, so a floor that omits
// minor/patch must not fail an installed version that exceeds it on a later
// component. Using "both declared" (not "either declared") keeps a sparse
// floor permissive in exactly the way a floor should be.
func (v Version) Compare(other Version) int {
	if c := cmpInt(v.Major, other.Major); c != 0 {
		return c
	}
	if v.HasMinor && other.HasMinor {
		if c := cmpInt(v.Minor, other.Minor); c != 0 {
			return c
		}
	}
	if v.HasPatch && other.HasPatch {
		if c := cmpInt(v.Patch, other.Patch); c != 0 {
			return c
		}
	}
	// Pre-release precedence: no prerelease > prerelease.
	switch {
	case v.PreRelease == "" && other.PreRelease != "":
		return 1
	case v.PreRelease != "" && other.PreRelease == "":
		return -1
	case v.PreRelease != other.PreRelease:
		return strings.Compare(v.PreRelease, other.PreRelease)
	}
	return 0
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// AtLeast reports whether v >= floor, using Compare semantics.
func (v Version) AtLeast(floor Version) bool {
	return v.Compare(floor) >= 0
}
