package verifytoolchain

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	cases := []struct {
		raw      string
		wantStr  string // expected parsedString output
		wantErr  bool
	}{
		// Real --version outputs from the tools we declare.
		{raw: "v22.3.0", wantStr: "22.3.0"},
		{raw: "Node.js v22.3.0\n", wantStr: "22.3.0"},
		{raw: "go version go1.26.3 darwin/arm64", wantStr: "1.26.3"},
		{raw: "Python 3.13.13", wantStr: "3.13.13"},
		{raw: "Python 3.13.13 (main, ...)", wantStr: "3.13.13"},
		{raw: "pip 26.1.1 from /usr/lib/python3.13/...", wantStr: "26.1.1"},
		{raw: "npm 11.16.0", wantStr: "11.16.0"},
		{raw: "uv 0.11.14 (3fdfdc7d4 2026-05-12 aarch64-apple-darwin)", wantStr: "0.11.14"},
		{raw: "git version 2.54.0", wantStr: "2.54.0"},
		{raw: "pnpm 10.0.0", wantStr: "10.0.0"},
		{raw: "PHP 8.3.6 (cli) (built: ...)", wantStr: "8.3.6"},
		{raw: "Composer version 2.7.9 2024-09-10 14:30:00", wantStr: "2.7.9"},

		// Floor without patch (contract can declare "20.19").
		{raw: "20.19", wantStr: "20.19"},

		// Pre-release tag is preserved.
		{raw: "1.0.0-rc.1", wantStr: "1.0.0-rc.1"},

		// No version-like token at all.
		{raw: "command not found", wantErr: true},
		{raw: "", wantErr: true},
		// Major-only is NOT enough — we require at least major.minor so a
		// floor like "20.19" can't be satisfied by a bare "20".
		{raw: "go version go1", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			v, err := ParseVersion(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseVersion(%q) = %+v, want error", tc.raw, v)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseVersion(%q) error: %v", tc.raw, err)
			}
			got := parsedString(v)
			if got != tc.wantStr {
				t.Errorf("ParseVersion(%q) parsed = %s, want %s", tc.raw, got, tc.wantStr)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	cases := []struct {
		name     string
		a, b     Version
		wantSign int // -1, 0, +1
	}{
		// Equal numeric versions.
		{"eq", MustParse("20.19.0"), MustParse("20.19.0"), 0},
		// Major bump wins.
		{"major", MustParse("21.0.0"), MustParse("20.19.0"), 1},
		// Minor bump.
		{"minor_higher", MustParse("20.20.0"), MustParse("20.19.0"), 1},
		{"minor_lower", MustParse("20.18.5"), MustParse("20.19.0"), -1},
		// Patch bump.
		{"patch", MustParse("20.19.1"), MustParse("20.19.0"), 1},
		// Floor "20.19" (no patch) vs installed "20.19.5": patch is ignored
		// when the floor omits it — compares equal on major.minor.
		{"floor_no_patch_eq", MustParse("20.19.5"), floorVersion("20.19"), 0},
		// Floor "20.19" vs installed "20.20.0": minor higher => installed wins.
		{"floor_no_patch_higher", MustParse("20.20.0"), floorVersion("20.19"), 1},
		// Floor "20.19" vs installed "20.18.9": installed loses.
		{"floor_no_patch_lower", MustParse("20.18.9"), floorVersion("20.19"), -1},
		// Pre-release: 1.0.0 > 1.0.0-rc.1
		{"prerelease_lt_release", MustParse("1.0.0"), MustParse("1.0.0-rc.1"), 1},
		// Two pre-releases compare lexically.
		{"prerelease_alpha_beta", MustParse("1.0.0-beta"), MustParse("1.0.0-alpha"), 1},
		// Go uses 1.26 (two parts) — must compare against go.mod floor 1.26.
		{"go_style", MustParse("1.26.3"), floorVersion("1.26"), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sign(tc.a.Compare(tc.b))
			if got != tc.wantSign {
				t.Errorf("Compare(%s, %s) sign = %d, want %d", parsedString(tc.a), parsedString(tc.b), got, tc.wantSign)
			}
		})
	}
}

func TestAtLeast(t *testing.T) {
	// Mirrors the real floors from toolchain.toml.
	checks := []struct {
		installed string
		floor     string
		want      bool
	}{
		// node floor 20.19
		{"22.3.0", "20.19", true},
		{"20.19.0", "20.19", true},
		{"20.18.0", "20.19", false},
		{"18.0.0", "20.19", false}, // the Node-18-too-old bug
		// python3 floor 3.11
		{"3.13.13", "3.11", true},
		{"3.11.0", "3.11", true},
		{"3.10.0", "3.11", false},
		// go floor 1.26
		{"1.26.3", "1.26", true},
		{"1.25.0", "1.26", false},
		// uv floor 0.5
		{"0.11.14", "0.5", true},
		{"0.4.0", "0.5", false},
		// pip floor 23
		{"26.1.1", "23", true},
		{"23.0.0", "23", true},
		{"22.0.2", "23", false},
	}
	for _, c := range checks {
		t.Run(c.installed+"_vs_"+c.floor, func(t *testing.T) {
			v := MustParse(c.installed)
			f := floorVersion(c.floor)
			if got := v.AtLeast(f); got != c.want {
				t.Errorf("%s.AtLeast(%s) = %v, want %v", c.installed, c.floor, got, c.want)
			}
		})
	}
}

// sign normalizes a Compare result to -1/0/+1.
func sign(n int) int {
	if n < 0 {
		return -1
	}
	if n > 0 {
		return 1
	}
	return 0
}

// floorVersion parses a contract floor like "20.19" or "1.26" (the strings
// that appear under [tools.required.*].min in toolchain.toml).
func floorVersion(s string) Version {
	v, err := parseNumeric(s)
	if err != nil {
		panic(err)
	}
	return v
}
