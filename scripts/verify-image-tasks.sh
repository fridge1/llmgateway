#!/usr/bin/env bash
# 验证 /v1/images/tasks 异步图像接口：三类 Key（普通用户 / 租户 / 租户子用户）
# 各跑 生成 + 编辑 两种任务，轮询至完成，校验：
#   1) result_urls 张数正确且公网可访问
#   2) cost 已记录（订阅覆盖时 cost=0，单独提示）
#   3)（可选 DB_CHECK=1）反查对应账本确认扣费真正落账：
#        用户Key   -> transactions          (request_id LIKE 'img[-edit]-{id}-%')
#        租户Key   -> tenant_transactions    (sub_user_id IS NULL)
#        子用户Key -> tenant_transactions    (sub_user_id IS NOT NULL) + owner transactions
#
# 用法：
#   BASE_URL=http://localhost \
#   USER_KEY=sk-xxx TENANT_KEY=sk-xxx SUBUSER_KEY=sk-xxx \
#   MODEL=gpt-image-2 N=2 \
#   EDIT_IMAGE_URL=https://.../foo.png \   # 提供则跑编辑用例，否则跳过编辑
#   DB_CHECK=1 \                            # 开启查库核对（需本机 psql + .env 密码）
#   ./scripts/verify-image-tasks.sh
#
# 只填哪个 Key 就只测哪一类，未提供的自动跳过。
set -uo pipefail

BASE_URL="${BASE_URL:-http://localhost}"
MODEL="${MODEL:-gpt-image-2}"
PROMPT="${PROMPT:-a red apple on a wooden table, studio lighting}"
EDIT_PROMPT="${EDIT_PROMPT:-make the background pure white}"
SIZE="${SIZE:-1024x1024}"
N="${N:-1}"
POLL_INTERVAL="${POLL_INTERVAL:-3}"
POLL_TIMEOUT="${POLL_TIMEOUT:-600}"   # 秒，尾部模型最长 ~7 分钟
EDIT_IMAGE_URL="${EDIT_IMAGE_URL:-}"
DB_CHECK="${DB_CHECK:-0}"

pass=0; fail=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---- DB 连接（仅 DB_CHECK=1 时使用）-------------------------------------------
DB_DSN=""
if [ "$DB_CHECK" = "1" ]; then
  if [ -n "${DATABASE_DSN:-}" ]; then
    DB_DSN="$DATABASE_DSN"
  else
    PW=$(grep -E '^POSTGRES_PASSWORD=' "$SCRIPT_DIR/../.env" 2>/dev/null | head -1 | cut -d= -f2-)
    DB_HOST="${DB_HOST:-${DB_HOST}}"
    if [ -n "$PW" ]; then
      DB_DSN="postgres://gateway:${PW}@${DB_HOST}:5432/gateway?sslmode=disable"
    fi
  fi
  if [ -z "$DB_DSN" ]; then
    echo "⚠ DB_CHECK=1 但无法构造 DSN（缺 DATABASE_DSN 且 .env 无 POSTGRES_PASSWORD），跳过查库。"
    DB_CHECK=0
  fi
fi

# psql_q <sql> -> 单值输出
psql_q() { psql "$DB_DSN" -t -A -c "$1" 2>/dev/null | tr -d '[:space:]'; }

# verify_db_charge <label> <class:user|tenant|subuser> <task_id> <mode:generate|edit>
verify_db_charge() {
  [ "$DB_CHECK" = "1" ] || return 0
  local label="$1" cls="$2" id="$3" mode="$4"
  local prefix="img-${id}-"
  [ "$mode" = "edit" ] && prefix="img-edit-${id}-"
  local like="${prefix}%"

  local cnt sum
  case "$cls" in
    user)
      cnt=$(psql_q "SELECT count(*) FROM transactions WHERE request_id LIKE '${like}' AND model='${MODEL}';")
      sum=$(psql_q "SELECT COALESCE(SUM(amount),0) FROM transactions WHERE request_id LIKE '${like}' AND model='${MODEL}';")
      echo "[$label]   DB transactions: 行数=$cnt 金额合计=$sum"
      ;;
    tenant)
      cnt=$(psql_q "SELECT count(*) FROM tenant_transactions WHERE request_id LIKE '${like}' AND sub_user_id IS NULL;")
      sum=$(psql_q "SELECT COALESCE(SUM(amount),0) FROM tenant_transactions WHERE request_id LIKE '${like}' AND sub_user_id IS NULL;")
      echo "[$label]   DB tenant_transactions(租户): 行数=$cnt 金额合计=$sum"
      ;;
    subuser)
      cnt=$(psql_q "SELECT count(*) FROM tenant_transactions WHERE request_id LIKE '${like}' AND sub_user_id IS NOT NULL;")
      sum=$(psql_q "SELECT COALESCE(SUM(amount),0) FROM tenant_transactions WHERE request_id LIKE '${like}' AND sub_user_id IS NOT NULL;")
      local ocnt
      ocnt=$(psql_q "SELECT count(*) FROM transactions WHERE request_id LIKE '${like}';")
      echo "[$label]   DB tenant_transactions(子用户): 行数=$cnt 金额合计=${sum}；owner transactions 行数=$ocnt"
      ;;
  esac
  if [ "${cnt:-0}" -ge 1 ]; then
    echo "[$label]   ✓ 账本已落账"
  else
    echo "[$label]   ⚠ 未查到扣费行（可能命中订阅覆盖，或结算尚在异步处理中）"
  fi
}

# poll_task <label> <api_key> <task_id> -> 设置全局 LAST_TASK_JSON / LAST_STATUS
poll_task() {
  local label="$1" key="$2" id="$3" waited=0 task status
  while :; do
    task=$(curl -s "$BASE_URL/v1/images/tasks/$id" -H "Authorization: Bearer $key")
    status=$(echo "$task" | jq -r '.status // "unknown"')
    printf '\r[%s] 轮询中… status=%-10s 已等待 %ds' "$label" "$status" "$waited"
    case "$status" in completed|failed) echo ""; break ;; esac
    if [ "$waited" -ge "$POLL_TIMEOUT" ]; then
      echo ""; echo "[$label] ✗ 轮询超时（${POLL_TIMEOUT}s）"; LAST_STATUS="timeout"; return
    fi
    sleep "$POLL_INTERVAL"; waited=$((waited+POLL_INTERVAL))
  done
  LAST_TASK_JSON="$task"; LAST_STATUS="$status"
}

# check_result <label> -> 校验 result_urls + cost，使用全局 LAST_TASK_JSON。返回 0/1 给 ok
check_result() {
  local label="$1" task="$LAST_TASK_JSON" urls cost n_urls ok=1 i=0 ucode u
  urls=$(echo "$task" | jq -r '.result_urls[]? // empty')
  cost=$(echo "$task" | jq -r '.cost // 0')
  n_urls=$(echo "$task" | jq -r '(.result_urls // []) | length')
  echo "[$label] 完成：image_count=$(echo "$task" | jq -r '.image_count') result_urls=$n_urls cost=$cost"
  [ "$n_urls" -lt 1 ] && { echo "[$label] ✗ result_urls 为空"; ok=0; }
  [ "$n_urls" != "$N" ] && echo "[$label] ⚠ 期望 $N 张，实际 $n_urls 张"
  while IFS= read -r u; do
    [ -z "$u" ] && continue
    i=$((i+1)); ucode=$(curl -s -o /dev/null -w '%{http_code}' -I "$u")
    if [ "$ucode" = "200" ]; then echo "[$label]   图$i 可访问 (200) $u"
    else echo "[$label]   图$i ✗ 不可访问 ($ucode) $u"; ok=0; fi
  done <<< "$urls"
  if awk "BEGIN{exit !($cost > 0)}"; then echo "[$label] ✓ cost=${cost}（已计费）"
  else echo "[$label] ⚠ cost=0（可能命中订阅配额覆盖）"; fi
  CHECK_OK="$ok"
}

# try_delete <label> <key> <task_id> —— DELETE_CHECK=1 时删除已完成任务并校验 204 + 后续 GET 404
try_delete() {
  [ "${DELETE_CHECK:-0}" = "1" ] || return 0
  local label="$1" key="$2" id="$3" dcode gcode
  dcode=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE "$BASE_URL/v1/images/tasks/$id" -H "Authorization: Bearer $key")
  if [ "$dcode" = "204" ]; then echo "[$label]   ✓ 删除成功 (204) task_id=$id"
  else echo "[$label]   ✗ 删除返回 $dcode（期望 204）"; CHECK_OK=0; return; fi
  gcode=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/v1/images/tasks/$id" -H "Authorization: Bearer $key")
  if [ "$gcode" = "404" ]; then echo "[$label]   ✓ 删除后 GET 返回 404"
  else echo "[$label]   ⚠ 删除后 GET 返回 $gcode（期望 404）"; fi
}

# run_generate <label> <key> <class>
run_generate() {
  local label="$1" key="$2" cls="$3" body resp http id
  echo "==================================================================="
  echo "[$label] 生成任务  model=$MODEL n=$N size=$SIZE"
  echo "-------------------------------------------------------------------"
  body=$(jq -nc --arg m "$MODEL" --arg p "$PROMPT" --arg s "$SIZE" --argjson n "$N" \
    '{model:$m, prompt:$p, size:$s, n:$n}')
  resp=$(curl -s -w $'\n%{http_code}' -X POST "$BASE_URL/v1/images/tasks" \
    -H "Authorization: Bearer $key" -H 'Content-Type: application/json' -d "$body")
  http=$(echo "$resp" | tail -1); resp=$(echo "$resp" | sed '$d')
  if [ "$http" != "202" ]; then echo "[$label] ✗ 提交失败 HTTP=$http resp=$resp"; fail=$((fail+1)); return; fi
  id=$(echo "$resp" | jq -r '.id'); echo "[$label] ✓ 已受理 task_id=$id"
  poll_task "$label" "$key" "$id"
  [ "$LAST_STATUS" = "completed" ] || { echo "[$label] ✗ status=$LAST_STATUS error=$(echo "$LAST_TASK_JSON" | jq -r '.error_message // ""')"; fail=$((fail+1)); return; }
  check_result "$label"
  verify_db_charge "$label" "$cls" "$id" generate
  try_delete "$label" "$key" "$id"
  if [ "$CHECK_OK" = 1 ]; then echo "[$label] ✓ 生成通过"; pass=$((pass+1)); else echo "[$label] ✗ 生成未通过"; fail=$((fail+1)); fi
}

# run_edit <label> <key> <class>
run_edit() {
  local label="$1" key="$2" cls="$3" body resp http id
  [ -z "$EDIT_IMAGE_URL" ] && { echo "（[$label] 跳过 编辑：未设置 EDIT_IMAGE_URL）"; return; }
  echo "==================================================================="
  echo "[$label] 编辑任务  model=$MODEL n=$N url=$EDIT_IMAGE_URL"
  echo "-------------------------------------------------------------------"
  body=$(jq -nc --arg m "$MODEL" --arg p "$EDIT_PROMPT" --arg s "$SIZE" --argjson n "$N" --arg u "$EDIT_IMAGE_URL" \
    '{model:$m, prompt:$p, size:$s, n:$n, image_urls:[$u]}')
  resp=$(curl -s -w $'\n%{http_code}' -X POST "$BASE_URL/v1/images/tasks/edits" \
    -H "Authorization: Bearer $key" -H 'Content-Type: application/json' -d "$body")
  http=$(echo "$resp" | tail -1); resp=$(echo "$resp" | sed '$d')
  if [ "$http" != "202" ]; then echo "[$label] ✗ 提交失败 HTTP=$http resp=$resp"; fail=$((fail+1)); return; fi
  id=$(echo "$resp" | jq -r '.id'); echo "[$label] ✓ 已受理 task_id=$id"
  poll_task "$label" "$key" "$id"
  [ "$LAST_STATUS" = "completed" ] || { echo "[$label] ✗ status=$LAST_STATUS error=$(echo "$LAST_TASK_JSON" | jq -r '.error_message // ""')"; fail=$((fail+1)); return; }
  check_result "$label"
  verify_db_charge "$label" "$cls" "$id" edit
  try_delete "$label" "$key" "$id"
  if [ "$CHECK_OK" = 1 ]; then echo "[$label] ✓ 编辑通过"; pass=$((pass+1)); else echo "[$label] ✗ 编辑未通过"; fail=$((fail+1)); fi
}

run_class() { run_generate "$1" "$2" "$3"; run_edit "$1" "$2" "$3"; }

echo "BASE_URL=$BASE_URL  MODEL=$MODEL  N=$N  DB_CHECK=$DB_CHECK  EDIT=${EDIT_IMAGE_URL:+on}"
[ -n "${USER_KEY:-}" ]    && run_class "用户Key"   "$USER_KEY"    user    || echo "（跳过 用户Key：未设置 USER_KEY）"
[ -n "${TENANT_KEY:-}" ]  && run_class "租户Key"   "$TENANT_KEY"  tenant  || echo "（跳过 租户Key：未设置 TENANT_KEY）"
[ -n "${SUBUSER_KEY:-}" ] && run_class "子用户Key" "$SUBUSER_KEY" subuser || echo "（跳过 子用户Key：未设置 SUBUSER_KEY）"

echo "==================================================================="
echo "结果：通过 ${pass}，失败 $fail"
[ "$fail" -eq 0 ]
