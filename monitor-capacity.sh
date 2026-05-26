#!/bin/bash

# Continuous capacity monitor that auto-deploys when capacity found
# Usage: ./monitor-capacity.sh [region] [interval_seconds]

REGION="${1:-sa-saopaulo-1}"
INTERVAL="${2:-300}"  # Default: check every 5 minutes
SKILL_DIR="/Users/danielvm/.claude/plugins/cache/vibe-skills/admin-devops/f6491fc8bd7d/skills/oci"

echo "🔍 Starting capacity monitor for $REGION"
echo "   Checking every $INTERVAL seconds..."
echo "   Will auto-deploy when capacity found"
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
    echo "🎉 CAPACITY FOUND! Deploying..."
    echo ""

    # Auto-deploy
    terraform apply -auto-approve

    if [ $? -eq 0 ]; then
      echo ""
      echo "✅ Deployment successful!"
      echo "📋 Outputs:"
      terraform output
      exit 0
    else
      echo ""
      echo "❌ Deployment failed. Retrying in $INTERVAL seconds..."
    fi
  else
    echo "   ❌ No capacity in $REGION (checked at $TIMESTAMP)"
    echo "   Waiting $INTERVAL seconds before next check..."
    echo ""
  fi

  sleep "$INTERVAL"
done
