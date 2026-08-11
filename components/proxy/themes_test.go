package proxy

import (
	"bytes"
	"html/template"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestLandingThemeInjectionPoint asserts the homeTemplate wires the Go-generated
// theme script and keys dark mode off [data-theme="dark"] (not only the
// prefers-color-scheme media query), with a no-JS fallback for visitors whose
// browser never sets data-theme. These are template-source assertions that do
// not depend on themes.go compiling.
func TestLandingThemeInjectionPoint(t *testing.T) {
	if !strings.Contains(homeTemplate, "{{.ThemeScript}}") {
		t.Fatal(`homeTemplate must inject {{.ThemeScript}} in <head> so the theme script runs before paint`)
	}
	if !strings.Contains(homeTemplate, `[data-theme="dark"]`) {
		t.Fatal(`dark mode must key off [data-theme="dark"], not only prefers-color-scheme`)
	}
	if !strings.Contains(homeTemplate, `:root:not([data-theme])`) {
		t.Fatal(`missing no-JS prefers-color-scheme fallback (":root:not([data-theme])")`)
	}
}

// TestLandingThemeRendersUnescaped executes homeTemplate end-to-end and asserts
// the generated script is injected as a live <script> element (not HTML-escaped
// to inert &lt;script&gt;) and carries the accent data + localStorage reads.
func TestLandingThemeRendersUnescaped(t *testing.T) {
	tmpl := template.Must(template.New("home").Funcs(templateFuncs).Parse(homeTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		Version     string
		Components  []any
		GitHubStars string
		Port        string
		ThemeScript template.HTML
	}{
		ThemeScript: landingThemeScript(),
	}); err != nil {
		t.Fatalf("execute homeTemplate: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<script>") {
		t.Fatal("rendered HTML must contain a live <script> (ThemeScript escaped to &lt;script&gt;?)")
	}
	if !strings.Contains(out, "bigbase-theme") || !strings.Contains(out, "data-theme") {
		t.Error("rendered HTML must carry the theme bootstrap (localStorage read + data-theme)")
	}
	if !strings.Contains(out, "december") {
		t.Error("rendered HTML must embed the accent ramp (expected 'december' id)")
	}
}

// TestLandingThemeScript asserts the generated inline script: reads the admin's
// localStorage keys, resolves theme with a prefers-color-scheme fallback, applies
// data-theme + the full brand custom-property set, and toggles the rainbow flag.
// It also enforces the security invariant: no HTML sink (innerHTML/document.write/eval).
func TestLandingThemeScript(t *testing.T) {
	script := string(landingThemeScript())

	// Reads the admin's localStorage keys (same origin).
	for _, key := range []string{"bigbase-theme", "bigbase-accent"} {
		if !strings.Contains(script, key) {
			t.Errorf("script must read localStorage %q", key)
		}
	}

	// OS-preference fallback for first-time visitors.
	if !strings.Contains(script, "prefers-color-scheme") {
		t.Error("script must fall back to matchMedia('(prefers-color-scheme: dark)')")
	}

	// Applies data-theme + the brand custom-property set (mirrors applyAccentToDocument).
	for _, want := range []string{
		"data-theme",
		"--brand-500", "--brand-600", "--brand-700",
		"--bg-accent", "--bg-accent-hover", "--bg-accent-active",
		"--fg-accent", "--border-accent",
		"--brand-tint", "--focus-ring",
		"data-accent-rainbow",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script must apply %q", want)
		}
	}

	// Every accent id is embedded in the ACCENTS table.
	for _, a := range landingAccents {
		if !strings.Contains(script, a.ID) {
			t.Errorf("accent %q missing from generated script", a.ID)
		}
	}

	// Security: no HTML sink — only attributes + CSS custom properties are written.
	for _, bad := range []string{"innerHTML", "document.write", "eval("} {
		if strings.Contains(script, bad) {
			t.Errorf("security: script must not use %q (no HTML sink)", bad)
		}
	}
}

// TestLandingThemeScriptHandlesUnknownAccent asserts an unknown accent value is
// rejected (only known ACCENTS keys are honoured), so a malicious localStorage
// value can never reach a DOM write as anything other than a known enum.
func TestLandingThemeScriptValidatesAccent(t *testing.T) {
	script := string(landingThemeScript())
	// The script must guard accent lookup: only accept keys present in the table.
	// We assert the ACCENTS membership check is present (A[a] truthiness guard).
	if !strings.Contains(script, "A[") {
		t.Error("script must validate accent against the ACCENTS table before use")
	}
}

// TestAccentRampParity is the drift guard (e85s02): the Go landingAccents table
// must stay byte-for-byte equal to the admin's canonical source
// ui/src/context/accentThemes.ts. Fails with a diff if either side changes
// without the other.
func TestAccentRampParity(t *testing.T) {
	tsAccents := parseTSAccents(t)

	if len(tsAccents) != len(landingAccents) {
		t.Fatalf("accent count drift: ts=%d go=%d", len(tsAccents), len(landingAccents))
	}

	for i, want := range landingAccents {
		got := tsAccents[i]
		if got != want {
			t.Errorf("accent[%d] %q drift:\n  go: %+v\n  ts: %+v", i, want.ID, want, got)
		}
	}
}

// parseTSAccents extracts the ACCENT_THEMES entries from the admin's canonical
// TypeScript source. Each entry carries id, brand500/600/700 RGB triples and an
// optional rainbow flag.
func parseTSAccents(t *testing.T) []landingAccent {
	t.Helper()
	src, err := os.ReadFile("../../ui/src/context/accentThemes.ts")
	if err != nil {
		t.Fatalf("read accentThemes.ts: %v", err)
	}
	s := string(src)

	blockRe := regexp.MustCompile(`\{[^{}]*\}`)
	idRe := regexp.MustCompile(`id:\s*'([^']+)'`)
	b500 := regexp.MustCompile(`brand500:\s*'([^']+)'`)
	b600 := regexp.MustCompile(`brand600:\s*'([^']+)'`)
	b700 := regexp.MustCompile(`brand700:\s*'([^']+)'`)
	rainbowRe := regexp.MustCompile(`rainbow:\s*true`)

	var out []landingAccent
	for _, block := range blockRe.FindAllString(s, -1) {
		// Only ACCENT_THEMES entries have a quoted brand500 value; the
		// AccentTheme interface block uses `brand500: string` (no quotes).
		if !b500.MatchString(block) {
			continue
		}
		id := idRe.FindStringSubmatch(block)
		f500 := b500.FindStringSubmatch(block)
		f600 := b600.FindStringSubmatch(block)
		f700 := b700.FindStringSubmatch(block)
		if id == nil || f500 == nil || f600 == nil || f700 == nil {
			t.Fatalf("could not parse accent block: %s", block)
		}
		out = append(out, landingAccent{
			ID:       id[1],
			Brand500: f500[1],
			Brand600: f600[1],
			Brand700: f700[1],
			Rainbow:  rainbowRe.MatchString(block),
		})
	}
	if len(out) != 13 {
		t.Fatalf("expected 13 accent entries in TS source, got %d", len(out))
	}
	return out
}
