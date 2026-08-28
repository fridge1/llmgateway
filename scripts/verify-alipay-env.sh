#!/usr/bin/env bash
# Post-deploy checks for Alipay PC web pay (optional).
# Usage: CONFIG=/path/to/config.yaml ./scripts/verify-alipay-env.sh
set -euo pipefail
CONFIG="${CONFIG:-config.docker.yaml}"
if [[ ! -f "$CONFIG" ]]; then
  echo "Config not found: $CONFIG (set CONFIG=...)"
  exit 1
fi

echo "Checking $CONFIG for payment.alipay..."
# Simple grep-based hints (YAML structure may vary)
if grep -q 'app_id: ""' "$CONFIG" 2>/dev/null || grep -q "app_id: ''" "$CONFIG" 2>/dev/null; then
  echo "WARN: app_id appears empty — set payment.alipay.app_id"
fi
if grep -q 'notify_url: ""' "$CONFIG" 2>/dev/null || grep -q "notify_url: ''" "$CONFIG" 2>/dev/null; then
  echo "WARN: notify_url appears empty — async recharge will not run"
fi
if grep -qE '^[[:space:]]*is_production:[[:space:]]*true' "$CONFIG" 2>/dev/null; then
  echo "OK: is_production=true (openapi.alipay.com)"
else
  echo "INFO: is_production not true — using sandbox gateway unless overridden"
fi

echo "Done. Manual checks: HTTPS reachable /api/payment/alipay/notify, small sandbox payment, then production test."
