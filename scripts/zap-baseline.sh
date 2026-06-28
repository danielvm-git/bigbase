#!/usr/bin/env bash
set -euo pipefail

# zap-baseline.sh — OWASP ZAP DAST baseline scan
#
# Runs a ZAP baseline scan against TARGET_URL using the official Docker image.
# Generates zap-report.html and exits non-zero on HIGH-severity alerts.
#
# Usage:
#   TARGET_URL=https://example.com bash scripts/zap-baseline.sh
#   TARGET_URL=http://localhost:9999 bash scripts/zap-baseline.sh
#
# Environment:
#   TARGET_URL   — Required. URL to scan (e.g. http://localhost:9999)
#   ZAP_IMAGE    — ZAP Docker image tag (default: ghcr.io/zaproxy/zaproxy:stable)
#   REPORT_DIR   — Output directory for report (default: .)
#
# Requires: Docker

usage() {
  cat <<EOF
Usage: TARGET_URL=<url> bash $0

Scans TARGET_URL with OWASP ZAP baseline and writes zap-report.html.

Environment:
  TARGET_URL   Required. URL to scan.
  ZAP_IMAGE    ZAP Docker image (default: ghcr.io/zaproxy/zaproxy:stable)
  REPORT_DIR   Output directory (default: current directory)

Exit codes:
  0 — No HIGH alerts found (or Docker unavailable)
  1 — HIGH alerts detected
EOF
  exit 0
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
fi

if [[ -z "${TARGET_URL:-}" ]]; then
  echo "ERROR: TARGET_URL is required."
  echo "Usage: TARGET_URL=<url> bash $0"
  exit 1
fi

ZAP_IMAGE="${ZAP_IMAGE:-ghcr.io/zaproxy/zaproxy:stable}"
REPORT_DIR="${REPORT_DIR:-.}"

# Check Docker availability
if ! command -v docker &>/dev/null; then
  echo "WARN: Docker is not available. Skipping ZAP baseline scan."
  echo "Install Docker to run DAST scans: https://docs.docker.com/get-docker/"
  exit 0
fi

if ! docker info &>/dev/null; then
  echo "WARN: Docker daemon is not running. Skipping ZAP baseline scan."
  exit 0
fi

echo "=== OWASP ZAP Baseline Scan ==="
echo "Target: $TARGET_URL"
echo "Image:  $ZAP_IMAGE"
echo "Report: $REPORT_DIR/zap-report.html"
echo ""

# Pull the image if not cached
docker pull --quiet "$ZAP_IMAGE" 2>/dev/null || true

# Run the ZAP baseline scan
# --auto: run spider + passive scan
# -I: do not generate GUI
# -r: generate HTML report
set +e
docker run --rm \
  -v "$(pwd):/zap/wrk" \
  -e "HOME=/zap" \
  "$ZAP_IMAGE" \
  zap-baseline.py \
    -t "$TARGET_URL" \
    -r zap-report.html \
    -I \
    -j \
    -m 2
ZAP_EXIT=$?
set -e

if [[ $ZAP_EXIT -ne 0 && $ZAP_EXIT -ne 1 ]]; then
  echo "ERROR: ZAP baseline scan failed to execute (exit code: $ZAP_EXIT)."
  exit 1
fi

# Check if report was generated
if [[ -f "zap-report.html" ]]; then
  if [[ "$REPORT_DIR" != "." ]]; then
    mkdir -p "$REPORT_DIR"
    mv zap-report.html "$REPORT_DIR/zap-report.html"
  fi

  echo ""
  echo "✅ ZAP baseline scan complete."
  echo "   Report: $REPORT_DIR/zap-report.html"

  # Check for HIGH alerts in the report (simple heuristic)
  if grep -q '"riskdesc":"High"' "$REPORT_DIR/zap-report.html" 2>/dev/null; then
    echo "⚠️  WARNING: ZAP found HIGH-severity alerts. Review the report."
    exit 1
  fi

  exit 0
else
  echo ""
  echo "⚠️  ZAP scan completed but no report was generated."
  echo "   Check Docker logs for details."
  exit 1
fi
