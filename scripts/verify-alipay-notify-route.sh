#!/usr/bin/env bash
# 在「跑 Docker 的那台服务器」上执行，用于确认异步通知路径可达网关。
# 期望：HTTP 400（验签失败）——说明路由已到 llm-gateway；000/502/504 说明 Nginx/网络有问题。

set -euo pipefail

BASE="${1:-http://127.0.0.1}"
PATH_NOTIFY="/api/payment/alipay/notify"
URL="${BASE%/}${PATH_NOTIFY}"

echo "==> POST ${URL}"
code=$(curl -sS -o /dev/null -w "%{http_code}" -X POST "$URL" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "" --connect-timeout 5) || code="000"

echo "HTTP ${code}"
case "$code" in
  400) echo "OK: 网关已收到请求（空表单无法验签，属预期）。" ;;
  401|403) echo "若出现鉴权，检查是否误加了仅 JWT 的中间件。" ;;
  404) echo "路径未到达网关：检查 Nginx location、docker 端口映射。" ;;
  502|503|504) echo "反代到上游失败：检查 llm-gateway 容器是否 healthy。" ;;
  000) echo "连接失败：检查本机 80/443 是否监听、防火墙。" ;;
  *) echo "请结合 docker logs llm-gateway 查看是否有 POST ${PATH_NOTIFY}。" ;;
esac

echo ""
echo "公网自检（在能访问你域名的机器上）："
echo "  curl -sS -o /dev/null -w '%{http_code}\\n' -X POST \"https://你的域名${PATH_NOTIFY}\" \\"
echo "    -H 'Content-Type: application/x-www-form-urlencoded' -d ''"
echo ""
echo "支付宝开放平台：应用 → 开发设置 → 接口加签方式 / 授权回调地址，须与 config 中 notify_url 一致。"
