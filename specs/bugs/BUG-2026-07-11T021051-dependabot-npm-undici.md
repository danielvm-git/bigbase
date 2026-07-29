# undici (npm) — Multiple HTTP client vulnerabilities

**Source:** GHS Dependabot
**Severity:** MAJOR / NORMAL / MINOR
**Ecosystem:** npm (ui/)

## Vulnerabilities

### MAJOR (2)
| CVE | GHSA | Score | Description |
|-----|------|-------|-------------|
| CVE-2026-12151 | GHSA-vxpw-j846-p89q | 7.5 | WebSocket client DoS via fragment count bypass |
| CVE-2026-9697 | GHSA-vmh5-mc38-953g | 7.4 | TLS certificate validation bypass via dropped requestTls in SOCKS5 ProxyAgent |

### NORMAL (3)
| CVE | GHSA | Score | Description |
|-----|------|-------|-------------|
| CVE-2026-9678 | GHSA-pr7r-676h-xcf6 | 5.9 | Cross-user information disclosure via shared cache whitespace bypass |
| CVE-2026-9679 | GHSA-p88m-4jfj-68fv | 5.9 | HTTP header injection via Set-Cookie percent-decoding |
| CVE-2026-39365 (deps) | — | 0.0 | Path traversal in optimized deps .map handling (transitive) |

### MINOR (2)
| CVE | GHSA | Score | Description |
|-----|------|-------|-------------|
| CVE-2026-11525 | GHSA-g8m3-5g58-fq7m | 3.7 | Set-Cookie SameSite attribute downgrade via permissive substring matching |
| CVE-2026-6733 | GHSA-35p6-xmwp-9g52 | 3.7 | HTTP response queue poisoning via keep-alive socket reuse |

## Recommendation
Update undici to the latest version in ui/ package.json.

## Status
fixed

## Source
seal.github_dependabot

## Discovered
2026-07-11
