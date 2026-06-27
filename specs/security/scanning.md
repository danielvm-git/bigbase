# Security Scanning — BigBase

## Overview

BigBase uses a layered security scanning approach combining SAST (static analysis),
SCA (supply chain), secrets detection, and DAST (dynamic analysis).

| Layer | Tool | Scope | Frequency |
|-------|------|-------|-----------|
| SAST | [gosec](https://github.com/securego/gosec) | Go source code | Every commit (preflight) |
| SCA | [govulncheck](https://golang.org/x/vuln/cmd/govulncheck) | Go module dependencies | Every commit (preflight) |
| Secrets | [gitleaks](https://github.com/gitleaks/gitleaks) | Git history | Every commit (preflight) |
| DAST | [OWASP ZAP](https://www.zaproxy.org/) | Live deployment | Scheduled / on-demand |
| Headers | observatory-check.sh | Live deployment | On-demand |

## Prerequisites

```bash
# SAST
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Secrets scanning
go install github.com/zricethezav/gitleaks/v8@latest

# DAST (requires Docker)
docker pull ghcr.io/zaproxy/zaproxy:stable
```

## Preflight (CI gate)

The `npm run preflight` meta-script runs all static checks:

```bash
# Run all preflight checks
npm run preflight

# Individual steps:
npm run preflight:go      # go vet + tests + gosec
npm run preflight:build   # go build
npm run preflight:secrets # gitleaks
npm run preflight:ui      # UI build
```

Preflight runs on every push via GitHub Actions. All steps must pass
before a PR can merge.

## DAST — OWASP ZAP Baseline

The ZAP baseline scan runs passive + light active scanning against a live
deployment. It does NOT perform intrusive attacks.

### Local Run

```bash
# Start BigBase
go run . serve --port 9999 &

# Run ZAP baseline scan
TARGET_URL=http://localhost:9999 bash scripts/zap-baseline.sh
```

The scan generates `zap-report.html` in the current directory. The script
exits non-zero if HIGH-severity alerts are detected.

### Staging / Production

```bash
TARGET_URL=https://staging.bigbase.click bash scripts/zap-baseline.sh
```

### Scheduled Automation

ZAP baseline scans can be scheduled via cron or GitHub Actions:

```yaml
# .github/workflows/dast-scan.yml (example)
on:
  schedule:
    - cron: '0 6 * * 1'  # Every Monday at 6 AM
  workflow_dispatch:      # Manual trigger

jobs:
  zap-baseline:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: TARGET_URL=https://bigbase.click bash scripts/zap-baseline.sh
      - uses: actions/upload-artifact@v4
        with:
          name: zap-report
          path: zap-report.html
```

See [GitHub Actions docs](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions)
for complete workflow syntax.

## Header Audit — Observatory Check

The observatory-check.sh script audits HTTP response headers against
Mozilla Observatory-style best practices.

```bash
TARGET_URL=http://localhost:9999 bash scripts/observatory-check.sh
```

Expected header set:
| Header | Value | Purpose |
|--------|-------|---------|
| Content-Security-Policy | `default-src 'self'` / permissive for UI | XSS prevention |
| Strict-Transport-Security | `max-age=63072000; includeSubDomains` | HTTPS enforcement |
| X-Frame-Options | `DENY` | Clickjacking prevention |
| X-Content-Type-Options | `nosniff` | MIME sniffing prevention |
| Referrer-Policy | `strict-origin-when-cross-origin` | Referrer leakage prevention |
| Permissions-Policy | Restrictive defaults | Feature restriction |
| Cache-Control | `no-store` on API routes | Sensitive data caching |

## gosec Exclusion Rules

Certain gosec rules are excluded from the preflight due to intentional patterns
in the BigBase codebase. See [gosec-exclusions.md](gosec-exclusions.md) for the
full rationale. These should be revisited periodically as part of security
maintenance.

## Security Posture

### Current Coverage

- ✅ All Go code scanned for common vulnerabilities (gosec)
- ✅ Go dependencies checked for known CVEs (govulncheck)
- ✅ Git history scanned for secrets (gitleaks)
- ✅ Security headers audited on live deployment
- ⏳ DAST baseline (ZAP) — manual/on-demand
- ⏳ GitHub Actions scheduled DAST — not yet automated

### Roadmap

| Item | Status | Epic |
|------|--------|------|
| SAST (gosec) integration | ✅ Done | e48s02 |
| SCA (govulncheck) integration | ✅ Done | e48s02 |
| Secrets scanning (gitleaks) | ✅ Done | e48s02 |
| Preflight CI scripts | ✅ Done | e48s02 |
| ZAP baseline script | ✅ Done | e48s03 |
| Header audit script | ✅ Done | e48s03 |
| Scheduled DAST workflow | 📋 Planned | Future |
| Authenticated ZAP scanning | 📋 Planned | Future |
| Nonce-based CSP | 📋 Planned | e60 |
| Cookie hardening | 📋 Planned | e49 |

## References

- [OWASP ZAP User Guide](https://www.zaproxy.org/docs/)
- [Mozilla Observatory](https://observatory.mozilla.org/)
- [gosec docs](https://github.com/securego/gosec)
- [govulncheck docs](https://go.dev/doc/security/vuln/)
- [gitleaks docs](https://github.com/gitleaks/gitleaks)
