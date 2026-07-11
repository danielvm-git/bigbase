# golang.org/x/crypto — Multiple Critical and High Severity Vulnerabilities

**Source:** GHS Dependabot
**Seal Query:** business_unit=BigBase, source=github_dependabot
**Severity:** CRITICAL / MAJOR / NORMAL
**Ecosystem:** Go (gomod)

## Vulnerabilities

### CRITICAL (7)
| CVE | GHSA | Score | Description |
|-----|------|-------|-------------|
| CVE-2026-39830 | GHSA-vgwf-h737-ff37 | 9.1 | Invoking client can cause server deadlock on unexpected responses |
| CVE-2026-39831 | GHSA-89gr-r52h-f8rx | 9.1 | FIDO/U2F security key physical presence check bypass |
| CVE-2026-39832 | GHSA-f5wc-c3c7-36mc | 9.1 | Doesn't drop invoking agent constraints when forwarding keys |
| CVE-2026-39833 | GHSA-jppx-rxg9-jmrx | 9.1 | Doesn't enforce invoking key constraints |
| CVE-2026-39834 | GHSA-rm3j-f69w-wqmq | 9.1 | Vulnerable to infinite loop on large channel writes |
| CVE-2026-42508 | GHSA-5cgq-3rg8-m6cv | 9.1 | Auth bypass via unenforced @revoked status |
| CVE-2026-46595 | GHSA-x527-x647-q7gg | 10.0 | Invoking VerifiedPublicKeyCallback permissions skip enforcement |

### MAJOR (1)
| CVE | GHSA | Score | Description |
|-----|------|-------|-------------|
| CVE-2026-39829 | GHSA-w879-237q-wc7r | 7.5 | Pathological RSA/DSA parameters may cause DoS |

### NORMAL (4)
| CVE | GHSA | Score | Description |
|-----|------|-------|-------------|
| CVE-2026-39827 | GHSA-qpw4-5x99-6vjp | 6.5 | Memory leak when rejecting channels can lead to DoS |
| CVE-2026-39828 | GHSA-45gg-vh54-h5m9 | 6.3 | Bypass of certificate restrictions |
| CVE-2026-39835 | GHSA-78mq-xcr3-xm33 | 5.3 | Server panic during CheckHostKey/Authenticate flow |
| CVE-2026-46598 | GHSA-9m57-25v3-79x9 | 5.3 | Pathological inputs can lead to client panic |

## Exploit Scenario
These vulnerabilities affect SSH key authentication (golang.org/x/crypto/ssh). An attacker with network access could:
- Bypass certificate restrictions and authentication
- Trigger DoS via deadlock, infinite loops, or panics
- Escalate privileges via unenforced key constraints
- Exploit FIDO/U2F bypass for physical key impersonation

## Recommendation
Update golang.org/x/crypto to the latest version that patches CVE-2026-46595, CVE-2026-42508, CVE-2026-39827-39835. Run:
```
go get golang.org/x/crypto@latest
go mod tidy
```

## Status
triage

## Source
seal.github_dependabot

## Discovered
2026-07-11
