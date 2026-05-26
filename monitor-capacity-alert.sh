#!/bin/bash

# Continuous capacity monitor - alerts when capacity found, you approve deployment
# Usage: ./monitor-capacity-alert.sh [region] [interval_seconds]

REGION="${1:-sa-saopaulo-1}"
INTERVAL="${2:-300}"  # Default: check every 5 minutes
SKILL_DIR="/Users/danielvm/.claude/plugins/cache/vibe-skills/admin-devops/f6491fc8bd7d/skills/oci"

echo "🔍 Starting capacity monitor for $REGION"
echo "   Checking every $INTERVAL seconds..."
echo "   Will ALERT when capacity found (you approve deployment)"
echo ""

ATTEMPT=0
while true; do
  ATTEMPT=$((ATTEMPT + 1))
  TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

  echo "[$TIMESTAMP] Attempt #$ATTEMPT: Checking capacity..."

  # Run capacity check
  CAPACITY_OUTPUT=$("$SKILL_DIR/scripts/check-oci-capacity.sh" "$REGION" 2>&1)

  # Check if capacity found (look for "AVAILABLE" in output)
  if echo "$CAPACITY_OUTPUT" | grep -q "AVAILABLE"; then
    echo ""
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║          🎉 CAPACITY FOUND IN $REGION!                   ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""
    echo "Capacity details:"
    echo "$CAPACITY_OUTPUT" | grep -A 5 "AVAILABLE"
    echo ""
    echo "To deploy, run:"
    echo "  cd /Users/danielvm/Developer/big-appwrite"
    echo "  terraform apply -auto-approve"
    echo ""
    echo "Will continue checking in case deployment is needed for fallback..."
    echo ""
  else
    echo "   ❌ No capacity in $REGION (checked at $TIMESTAMP)"
    echo "   Next check in $INTERVAL seconds..."
    echo ""
  fi

  sleep "$INTERVAL"
done
