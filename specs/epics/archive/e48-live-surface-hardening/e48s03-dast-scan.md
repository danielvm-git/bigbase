# e48s03: DAST baseline scan and scheduled automation

## 1. Story ID
e48s03

## 2. Epic
e48 — Security: Live Surface Hardening

## 3. Status
planned

## 4. BCPs
2

## 5. Type
feat

## 6. Context
infra

## 7. Summary
Create an OWASP ZAP baseline DAST scan script that runs against the staging deployment, produce an HTML report, and document the security scanning posture. Add a Mozilla Observatory check script to audit HTTP security headers on the live domain.

## 8. Problem Statement
The project has no dynamic application security testing (DAST). After e48s01 and e48s02 add static defenses (path blocking, health auth, security headers, SAST/SCA), we need runtime verification that the live surface is actually hardened:
1. **No DAST baseline**: No automated scan verifies that deployed endpoints don't expose vulnerabilities like XSS, injection, or misconfiguration.
2. **No header audit**: No automated check verifies that security headers are correctly served on the live domain.

## 9. Proposed Solution
- Create `scripts/zap-baseline.sh`: wraps the OWASP ZAP Docker image (`ghcr.io/zaproxy/zaproxy:stable`) with baseline scan against the target URL. Generates `zap-report.html` and exits non-zero on HIGH alerts (matching Seal's vulnerability methodology gate: CRITICAL and MAJOR/HIGH blocked).
- Create `scripts/observatory-check.sh`: curls the Mozilla Observatory API or uses a local header comparison to document current HTTP security header score.
- Create `specs/security/scanning.md`: documents the DAST and header audit setup, how to run scans, and the expected baseline.

## 10. Affected Modules
| Module | Purpose | Callers | Contracts |
|--------|---------|---------|-----------|
| `scripts/zap-baseline.sh` (new) | OWASP ZAP DAST baseline scan | CI pipeline, developer CLI | Requires Docker; exits 0 on pass, non-zero on HIGH alerts |
| `scripts/observatory-check.sh` (new) | HTTP security header audit | CI pipeline, developer CLI | Exits 0; outputs current header score |
| `specs/security/scanning.md` (new) | Security scanning documentation | Developers, auditors | Markdown doc |

## 11. Dependencies
- **OWASP ZAP** `[OK]` — Apache 2.0 licensed, industry-standard DAST tool. Run via Docker: `ghcr.io/zaproxy/zaproxy:stable`
- **Docker** `[OK]` — Required to run ZAP container. Already available in CI and on developer machines.
- No new Go dependencies.

## 12. Implementation Steps

### Story e48s03: DAST baseline scan and scheduled automation — Implementation Steps

**type:** feat
**context:** infra
**Context**: Create DAST (Dynamic Application Security Testing) scripts for OWASP ZAP baseline scanning and Mozilla Observatory HTTP header auditing. Document the scanning posture in specs/security/scanning.md. The ZAP script runs against a live deployment, produces an HTML report, and fails on HIGH alerts.

## Steps

1. Create `scripts/zap-baseline.sh` — Docker-based OWASP ZAP baseline scan, accepts TARGET_URL env var, outputs `zap-report.html`, exits non-zero on HIGH alerts → verify: `bash scripts/zap-baseline.sh --help 2>&1 | grep -i 'usage\|TARGET_URL'`

2. Test ZAP scan against local running instance (optional, requires Docker) → verify: `TARGET_URL=http://localhost:9999 bash scripts/zap-baseline.sh && test -f zap-report.html || echo 'Docker not available, script syntax OK'`

3. Create `scripts/observatory-check.sh` — HTTP header audit using curl against TARGET_URL → verify: `TARGET_URL=http://localhost:9999 bash scripts/observatory-check.sh`

4. Create `specs/security/scanning.md` — document DAST setup, header audit, scheduled scanning strategy → verify: `test -f specs/security/scanning.md`

5. Verify all scripts are executable → verify: `test -x scripts/zap-baseline.sh && test -x scripts/observatory-check.sh`

## Verification Script (Step-by-Step)

1. **Verify ZAP script exists and is executable**:
   ```bash
   ls -la scripts/zap-baseline.sh
   # Expected: -rwxr-xr-x
   ```
2. **Start BigBase locally for testing**:
   ```bash
   go run . serve --port 9999 &
   sleep 3
   ```
3. **Run Observatory check against local instance**:
   ```bash
   TARGET_URL=http://localhost:9999 bash scripts/observatory-check.sh
   # Expected: output showing header presence/absence
   ```
4. **Run ZAP baseline scan (if Docker available)**:
   ```bash
   TARGET_URL=http://localhost:9999 bash scripts/zap-baseline.sh
   # Expected: zap-report.html generated
   # If Docker unavailable: script exits gracefully with message
   ```
5. **Verify documentation exists**:
   ```bash
   cat specs/security/scanning.md | head -20
   # Expected: document describing scanning setup
   ```
6. **Stop local server**:
   ```bash
   kill %1
   ```

## Out of scope

- GitHub Actions scheduled cron workflow (can be added in a follow-up; the scripts are CI-ready)
- Authenticated ZAP scanning (baseline is unauthenticated spider-only)
- ZAP full/active scan (baseline scan is passive + light active; full scans require more setup)
- OWASP Dependency-Check (handled by govulncheck in e48s02)
- Integration with Seal's DAST HTTP module (scripts are standalone; Seal integration is separate)

## Risks

- **Docker availability**: ZAP requires Docker. On machines without Docker, the scan is skipped. Mitigation: script checks for `docker` and exits gracefully with a message.
- **ZAP scan duration**: Baseline scan takes 2-5 minutes depending on application size. Mitigation: run asynchronously in CI or on schedule, not as a preflight gate.
- **False positives**: ZAP may flag issues that aren't exploitable (e.g., missing anti-CSRF tokens on a stateless API). Mitigation: document expected findings and use ZAP context files for future tuning.
- **Rate limiting interference**: If e47's rate limiter is active, ZAP's spider may trigger rate limits. Mitigation: configure ZAP to use a reasonable request rate or whitelist the scanner IP.

## 13. Definition of Done
- [x] `scripts/zap-baseline.sh` exists and is executable
- [x] ZAP script accepts `TARGET_URL` env var and generates `zap-report.html`
- [x] `scripts/observatory-check.sh` exists and is executable
- [x] Observatory script audits HTTP security headers
- [x] `specs/security/scanning.md` documents the scanning setup
- [x] Both scripts use `bash` (not zsh-specific features)

## 14. Acceptance Criteria (Gherkin)
```gherkin
Feature: DAST baseline scan and header audit

  Scenario: ZAP baseline scan script is executable
    Given scripts/zap-baseline.sh exists
    When I run bash scripts/zap-baseline.sh --help
    Then the output describes usage

  Scenario: ZAP scan runs against a target
    Given Docker is available
    And BigBase is running at http://localhost:9999
    When I run TARGET_URL=http://localhost:9999 bash scripts/zap-baseline.sh
    Then a zap-report.html file is generated
    And the script exits 0 if no HIGH alerts found

  Scenario: ZAP scan handles missing Docker gracefully
    Given Docker is not available
    When I run TARGET_URL=http://localhost:9999 bash scripts/zap-baseline.sh
    Then the script exits 0 with a "Docker not available" message

  Scenario: Observatory check audits headers
    Given BigBase is running at http://localhost:9999
    When I run TARGET_URL=http://localhost:9999 bash scripts/observatory-check.sh
    Then the output lists security headers present/missing

  Scenario: Scanning documentation exists
    When I check specs/security/scanning.md
    Then the file exists and describes the scanning setup
```

## 15. Non-Functional Requirements
- ZAP baseline scan: <10 minutes per target
- Both scripts: pure bash (#!/usr/bin/env bash), no external dependencies beyond Docker (ZAP) and curl (Observatory)
- Scripts must be idempotent — running twice produces same results

## 16. Test Strategy
- **Syntax**: `bash -n scripts/zap-baseline.sh` and `bash -n scripts/observatory-check.sh`
- **Dry-run**: ZAP script accepts `--help` flag
- **Integration**: Run against local instance and verify output files
- **Error handling**: Test with Docker unavailable (graceful exit)

## 17. Rollback Plan
- Remove `scripts/zap-baseline.sh` and `scripts/observatory-check.sh`
- Archive `specs/security/scanning.md`
- No other artifacts to clean up
