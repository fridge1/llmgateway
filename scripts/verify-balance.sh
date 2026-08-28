#!/usr/bin/env bash
# 验证 GET /v1/balance — 三类 Key（普通用户 / 租户 / 子用户）+ 无 Key 401
#
# 用法：
#   BASE_URL=http://localhost:9090 \
#   USER_KEY=sk-xxx TENANT_KEY=sk-xxx SUBUSER_KEY=sk-xxx \
#   ./scripts/verify-balance.sh
#
# 只填哪个 Key 就只测哪一类，未提供的自动跳过。
set -uo pipefail

BASE_URL="${BASE_URL:-http://localhost:9090}"
pass=0; fail=0

check() {
  local label="$1" key="$2" want_type="$3"
  local http_code body
  body=$(curl -s -o /tmp/balance_resp.json -w "%{http_code}" \
    -H "Authorization: Bearer $key" \
    "$BASE_URL/v1/balance")
  if [ "$http_code" != "200" ]; then
    echo "FAIL [$label] HTTP $http_code"; ((fail++)); return
  fi
  local got_type
  got_type=$(cat /tmp/balance_resp.json | python3 -c "import sys,json; print(json.load(sys.stdin).get('type',''))" 2>/dev/null || echo "")
  if [ "$got_type" = "$want_type" ]; then
    echo "PASS [$label] type=$got_type"; ((pass++))
  else
    echo "FAIL [$label] want type=$want_type, got=$got_type"; ((fail++))
  fi
}

# 无 Key → 401
http_no_key=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/v1/balance")
if [ "$http_no_key" = "401" ]; then
  echo "PASS [no key → 401]"; ((pass++))
else
  echo "FAIL [no key] HTTP $http_no_key (want 401)"; ((fail++))
fi

[ -n "${USER_KEY:-}"    ] && check "user key"    "$USER_KEY"    "user"
[ -n "${TENANT_KEY:-}"  ] && check "tenant key"  "$TENANT_KEY"  "tenant"
[ -n "${SUBUSER_KEY:-}" ] && check "sub_user key" "$SUBUSER_KEY" "sub_user"

echo ""
echo "Results: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
