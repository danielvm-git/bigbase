package deploy

import (
	"path/filepath"
	"strings"
)

// ResolvePureStaticServeDir picks a directory that contains index.html for
// pure-static deploys (no Node build). Preference order:
//  1. outDirHint (framework/manifest output) when it has index.html
//  2. public/ when public/index.html exists (Firebase / classic static)
//  3. appRoot when it has index.html
//
// Returns codedErr static_output_missing when none qualify — callers must not
// FileServer a source checkout (directory listing).
func ResolvePureStaticServeDir(appRoot, outDirHint string) (string, error) {
	var candidates []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		candidates = append(candidates, p)
	}
	if outDirHint != "" {
		add(filepath.Join(appRoot, outDirHint))
	}
	add(filepath.Join(appRoot, "public"))
	add(appRoot)

	tried := make([]string, 0, len(candidates))
	for _, c := range candidates {
		tried = append(tried, c)
		if fileExists(filepath.Join(c, "index.html")) {
			return c, nil
		}
	}
	return "", staticOutputMissing(tried)
}

// RequireStaticIndex fails closed when serveDir has no index.html.
// triedPaths are included in the operator hint (outDir candidates already checked).
func RequireStaticIndex(serveDir string, triedPaths ...string) error {
	if fileExists(filepath.Join(serveDir, "index.html")) {
		return nil
	}
	tried := append([]string{}, triedPaths...)
	if serveDir != "" {
		found := false
		for _, p := range tried {
			if p == serveDir {
				found = true
				break
			}
		}
		if !found {
			tried = append(tried, serveDir)
		}
	}
	return staticOutputMissing(tried)
}

func staticOutputMissing(tried []string) error {
	hintPaths := tried
	if len(hintPaths) == 0 {
		hintPaths = []string{"(none)"}
	}
	return codedErr(
		"static_output_missing",
		"static site has no index.html under the serve directory",
		"Tried: "+strings.Join(hintPaths, ", ")+
			". Run a framework build that emits index.html, set sites.root_path to the app package, or place public/index.html.",
	)
}
