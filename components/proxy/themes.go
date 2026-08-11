package proxy

import (
	"fmt"
	"html/template"
	"strings"
)

// landingAccent is one entry of the landing-page accent ramp. It mirrors the
// admin console's canonical source ui/src/context/accentThemes.ts. Drift
// between the two is caught by TestAccentRampParity. Brand fields hold the
// "r, g, b" triple (without the rgb() wrapper) so the inline script can compose
// rgb()/rgba() the same way applyAccentToDocument does in the React app.
type landingAccent struct {
	ID       string
	Brand500 string
	Brand600 string
	Brand700 string
	Rainbow  bool
}

// landingAccents is the single Go source of truth for the landing page's accent
// ramp. It MUST stay byte-for-byte equal to ACCENT_THEMES in
// ui/src/context/accentThemes.ts (enforced by TestAccentRampParity).
var landingAccents = []landingAccent{
	{ID: "default", Brand500: "79, 70, 229", Brand600: "67, 56, 202", Brand700: "55, 48, 163"},
	{ID: "january", Brand500: "13, 148, 136", Brand600: "15, 118, 110", Brand700: "17, 94, 89"},
	{ID: "february", Brand500: "234, 88, 12", Brand600: "194, 65, 12", Brand700: "154, 52, 18"},
	{ID: "march", Brand500: "124, 58, 237", Brand600: "109, 40, 217", Brand700: "91, 33, 182"},
	{ID: "april", Brand500: "22, 163, 74", Brand600: "21, 128, 61", Brand700: "22, 101, 52"},
	{ID: "may", Brand500: "167, 139, 250", Brand600: "139, 92, 246", Brand700: "124, 58, 237"},
	{ID: "june", Brand500: "79, 70, 229", Brand600: "67, 56, 202", Brand700: "55, 48, 163", Rainbow: true},
	{ID: "july", Brand500: "253, 186, 116", Brand600: "251, 146, 60", Brand700: "234, 88, 12"},
	{ID: "august", Brand500: "156, 163, 175", Brand600: "107, 114, 128", Brand700: "75, 85, 99"},
	{ID: "september", Brand500: "234, 179, 8", Brand600: "202, 138, 4", Brand700: "161, 98, 7"},
	{ID: "october", Brand500: "236, 72, 153", Brand600: "219, 39, 119", Brand700: "190, 24, 93"},
	{ID: "november", Brand500: "37, 99, 235", Brand600: "29, 78, 216", Brand700: "30, 64, 175"},
	{ID: "december", Brand500: "220, 38, 38", Brand600: "185, 28, 28", Brand700: "153, 27, 27"},
}

// landingThemeScript returns the blocking inline <head> script that gives the
// server-rendered landing page full theme parity with the React admin console.
//
// Because the landing page and /admin/ are same-origin, they share localStorage.
// The script reads the admin's bigbase-theme (light|dark) and bigbase-accent
// (one of landingAccents) keys, resolves a final theme (falling back to the OS
// prefers-color-scheme for first-time visitors), and applies it before first
// paint: it sets <html data-theme> and overrides the brand CSS custom
// properties — the same set and rgb()/rgba() formulas as the admin's
// applyAccentToDocument (ui/src/context/ThemeContext.tsx). It toggles
// data-accent-rainbow for the June accent.
//
// Security: both localStorage reads are enum-validated before any DOM write.
// No value is ever written to an HTML sink — only attributes and CSS custom
// properties, all from a hardcoded data table. See
// specs/security/epics/e85/THREAT_MODEL.md.
func landingThemeScript() template.HTML {
	var b strings.Builder
	b.WriteString("<script>(function(){var A={")
	for i, a := range landingAccents {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%s:{b:%q,c:%q,d:%q,r:%t}", a.ID, a.Brand500, a.Brand600, a.Brand700, a.Rainbow)
	}
	b.WriteString(`};var t;try{t=localStorage.getItem('bigbase-theme')}catch(e){}
if(t!=='light'&&t!=='dark'){t=(window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches)?'dark':'light'}
var a;try{a=localStorage.getItem('bigbase-accent')}catch(e){}
if(!A[a]){a='default'}
var r=document.documentElement,v=A[a],s=r.style;
r.setAttribute('data-theme',t);
s.setProperty('--brand-500','rgb('+v.b+')');
s.setProperty('--brand-600','rgb('+v.c+')');
s.setProperty('--brand-700','rgb('+v.d+')');
s.setProperty('--border-accent','rgb('+v.b+')');
s.setProperty('--bg-accent','rgb('+v.b+')');
s.setProperty('--bg-accent-hover','rgb('+v.c+')');
s.setProperty('--bg-accent-active','rgb('+v.d+')');
s.setProperty('--fg-accent','rgb('+v.b+')');
s.setProperty('--brand-tint','rgba('+v.b+',0.10)');
s.setProperty('--focus-ring','0 0 0 3px rgba('+v.b+',0.18)');
if(v.r){r.setAttribute('data-accent-rainbow','true')}else{r.removeAttribute('data-accent-rainbow')}
	})();</script>`)
	// #nosec G203 -- verified false positive: b.String() is built solely from a
	// hardcoded Go data table (landingAccents) + static literals; no attacker-
	// controlled input reaches it. Both localStorage reads are enum-validated
	// before any DOM write, and the script contains no HTML sink. See
	// specs/security/epics/e85/THREAT_MODEL.md (CWE-79, confidence < 8).
	return template.HTML(b.String())
}
