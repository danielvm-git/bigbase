# gosec Exclusion Rationale

The following gosec rules are excluded from `npm run preflight:go` because they
produce false positives in the BigBase codebase. Each exclusion is documented
with the pattern that triggers it and why it's safe.

| Rule | CWE | Pattern Trigger | Rationale |
|------|-----|----------------|-----------|
| G104 | 703 | `w.Write()` errors unhandled | Standard Go pattern; writing to ResponseWriter rarely fails |
| G115 | 190 | uint64→int64 conversion in monitoring | System metrics never exceed int64 range |
| G404 | 338 | `math/rand` for backoff jitter | Not used for cryptographic purposes; performance matters |
| G702 | 78 | `exec.Command("git", ...)` with controlled args | All git args are hardcoded or system-controlled |
| G703 | 22 | `filepath.Join(s.dir, id)` for delete ops | id is validated UUID; path scoped to component dir |
| G704 | 918 | HTTP requests to configured URLs | URLs are system-configured (GitHub API, webhook targets) |
| G112 | 400 | `http.Server` without ReadHeaderTimeout | Mitigated by proxy layer; tracked in e48 (G112) |
| G114 | 676 | `http.ListenAndServe` without timeouts | MCP server; low-risk internal service |
| G118 | 400 | `context.Background()` in goroutines | Goroutines need independent lifecycle from HTTP request |
| G120 | 400 | `ParseMultipartForm` with MaxBytesReader | MaxBytesReader already limits body size before parse |
| G124 | 614 | Cookies without Secure/HttpOnly/SameSite | Pre-existing; to be addressed in auth hardening epic (e49) |
| G204 | 78 | `exec.Command` with variable args | All commands use hardcoded programs with variable data args |
| G301 | 276 | `os.MkdirAll` with 0755 | Directories need to be readable by deployed apps |
| G304 | 22 | `os.ReadFile` with joined path | Paths are scoped under component-specific directories |
| G306 | 276 | `os.WriteFile` with 0644 | Files need to be readable by deployment processes |
| G705 | 79 | `w.Write([]byte(html))` | HTML is hardcoded template, not user input |
| G710 | — | HTTP redirect with host | Host is validated deployment host; safe redirect |

## Action Items

These exclusions should be revisited as part of future security epics:
- G124 (cookie security) → e49 (Auth Hardening)
- G112 (Slowloris) → e48 (already tracked)
