package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FrameworkMode is the detected web-framework output model for a project dir.
type FrameworkMode struct {
	Framework  string  // "astro" | "sveltekit" | ""
	Mode       string  // "static" | "server" | "hybrid" | "ambiguous" | ""
	AppType    AppType // static | node when resolved
	OutDir     string
	Ambiguous  bool
	Reason     string
	PackageMgr string // npm | pnpm | yarn | bun when package.json present
}

var (
	reAstroOutput = regexp.MustCompile(`(?m)output\s*:\s*['"](static|server|hybrid)['"]`)
	reSveltePages = regexp.MustCompile(`(?m)pages\s*:\s*['"]([^'"]+)['"]`)
)

// DetectFrameworkMode classifies Astro / SvelteKit output models.
// It does not invent new AppType values — it maps to static vs node (or ambiguous).
func DetectFrameworkMode(dir string) FrameworkMode {
	pkgPath := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return FrameworkMode{}
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return FrameworkMode{}
	}
	deps := mergeDeps(pkg.Dependencies, pkg.DevDependencies)
	pm := DetectNodePackageManager(dir)

	if _, ok := deps["astro"]; ok {
		return detectAstroMode(dir, deps, pm)
	}
	if hasSvelteKit(deps) {
		return detectSvelteKitMode(dir, deps, pm)
	}
	return FrameworkMode{PackageMgr: pm}
}

func mergeDeps(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func hasSvelteKit(deps map[string]string) bool {
	_, ok := deps["@sveltejs/kit"]
	return ok
}

func hasDepPrefix(deps map[string]string, prefix string) bool {
	for k := range deps {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

func detectAstroMode(dir string, deps map[string]string, pm string) FrameworkMode {
	fm := FrameworkMode{Framework: "astro", PackageMgr: pm, OutDir: "dist"}
	cfg := readFirstFile(dir, "astro.config.mjs", "astro.config.js", "astro.config.ts", "astro.config.mts")
	output := "static"
	if m := reAstroOutput.FindStringSubmatch(cfg); len(m) == 2 {
		output = m[1]
	}
	fm.Mode = output
	hasNodeAdapter := hasDepPrefix(deps, "@astrojs/node") || strings.Contains(cfg, "@astrojs/node")
	switch output {
	case "static":
		fm.AppType = AppStatic
		fm.Reason = "astro output static (or unset)"
	case "server", "hybrid":
		if !hasNodeAdapter {
			fm.Ambiguous = true
			fm.Mode = "ambiguous"
			fm.Reason = "astro " + output + " without @astrojs/node — set app_type explicitly or add the node adapter"
			return fm
		}
		fm.AppType = AppNode
		fm.Reason = "astro output " + output + " with @astrojs/node"
	default:
		fm.Ambiguous = true
		fm.Mode = "ambiguous"
		fm.Reason = "unrecognized astro output"
	}
	return fm
}

func detectSvelteKitMode(dir string, deps map[string]string, pm string) FrameworkMode {
	fm := FrameworkMode{Framework: "sveltekit", PackageMgr: pm, OutDir: "build"}
	cfg := readFirstFile(dir, "svelte.config.js", "svelte.config.ts", "svelte.config.mjs")
	hasStatic := hasDep(deps, "@sveltejs/adapter-static") || strings.Contains(cfg, "adapter-static")
	hasNode := hasDep(deps, "@sveltejs/adapter-node") || strings.Contains(cfg, "adapter-node")
	hasAuto := hasDep(deps, "@sveltejs/adapter-auto") || strings.Contains(cfg, "adapter-auto")

	if m := reSveltePages.FindStringSubmatch(cfg); len(m) == 2 && m[1] != "" {
		fm.OutDir = strings.TrimPrefix(m[1], "./")
	}

	switch {
	case hasStatic && !hasNode:
		fm.Mode = "static"
		fm.AppType = AppStatic
		fm.Reason = "sveltekit adapter-static"
	case hasNode && !hasStatic:
		fm.Mode = "server"
		fm.AppType = AppNode
		fm.Reason = "sveltekit adapter-node"
	case hasStatic && hasNode:
		fm.Ambiguous = true
		fm.Mode = "ambiguous"
		fm.Reason = "both adapter-static and adapter-node present — set app_type explicitly"
	case hasAuto && !hasStatic && !hasNode:
		fm.Ambiguous = true
		fm.Mode = "ambiguous"
		fm.Reason = "sveltekit adapter-auto only — set app_type=static or app_type=node (or pin adapter-static/node)"
	default:
		// kit without a recognized adapter
		if _, ok := deps["@sveltejs/kit"]; ok {
			fm.Ambiguous = true
			fm.Mode = "ambiguous"
			fm.Reason = "sveltekit without a resolved adapter — set app_type explicitly"
		}
	}
	return fm
}

func hasDep(deps map[string]string, name string) bool {
	_, ok := deps[name]
	return ok
}

func readFirstFile(dir string, names ...string) string {
	for _, n := range names {
		data, err := os.ReadFile(filepath.Join(dir, n))
		if err == nil {
			return string(data)
		}
	}
	return ""
}

// ResolveDeployAppType combines explicit app_type with framework/runtime detection.
// Returns resolved type, preferred static outDir (may be empty), or a CodedError.
func ResolveDeployAppType(dir string, explicit AppType) (AppType, string, error) {
	fm := DetectFrameworkMode(dir)
	if fm.Ambiguous && explicit == "" {
		return "", "", codedErr("framework_mode_ambiguous",
			"framework output mode is ambiguous",
			fm.Reason+"; pass app_type=static or app_type=node in the deploy action")
	}
	detected := DetectAppType(dir)
	if fm.Framework != "" && !fm.Ambiguous && fm.AppType != "" {
		detected = fm.AppType
	}
	if explicit != "" {
		if !explicit.IsValid() {
			return "", "", codedErr("app_type_mismatch",
				"invalid app_type",
				"Use one of: static, node, go, python, php")
		}
		if fm.Framework != "" && !fm.Ambiguous && fm.AppType != "" && explicit != fm.AppType {
			return "", "", codedErr("app_type_mismatch",
				"explicit app_type does not match detected framework mode",
				"detected "+string(fm.AppType)+" for "+fm.Framework+"/"+fm.Mode+"; got app_type="+string(explicit)+". Align CI app_type or adapters.")
		}
		return explicit, fm.OutDir, nil
	}
	if detected == "" {
		detected = AppStatic
	}
	return detected, fm.OutDir, nil
}
