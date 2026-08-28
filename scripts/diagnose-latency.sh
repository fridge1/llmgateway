#!/bin/bash
# 诊断 LLM 网关延迟问题

echo "=== LLM Gateway 延迟诊断工具 ==="
echo ""

# 1. 检查数据库连接延迟
echo "1. 数据库连接延迟："
source /root/llm-gateway/.env
start=$(date +%s%N)
psql "postgres://gateway:${POSTGRES_PASSWORD}@${DB_HOST}:5432/gateway?sslmode=disable" -c "SELECT 1;" > /dev/null 2>&1
end=$(date +%s%N)
db_latency=$(( (end - start) / 1000000 ))
echo "   数据库响应时间: ${db_latency}ms"
if [ $db_latency -gt 100 ]; then
    echo "   ⚠️  数据库延迟较高（>100ms）"
fi
echo ""

# 2. 检查上游配置
echo "2. 上游配置统计："
psql "postgres://gateway:${POSTGRES_PASSWORD}@${DB_HOST}:5432/gateway?sslmode=disable" -t -c "
SELECT
    protocol,
    COUNT(*) as count,
    COUNT(DISTINCT model_id) as models
FROM upstreams
GROUP BY protocol
ORDER BY count DESC;
" 2>/dev/null
echo ""

# 3. 检查最慢的上游 base_url（按字符串特征分组）
echo "3. 上游分布（按域名）："
psql "postgres://gateway:${POSTGRES_PASSWORD}@${DB_HOST}:5432/gateway?sslmode=disable" -t -c "
SELECT
    SUBSTRING(base_url FROM 'https?://([^/]+)') as domain,
    COUNT(*) as count
FROM upstreams
GROUP BY domain
ORDER BY count DESC
LIMIT 10;
" 2>/dev/null
echo ""

# 4. 检查网关进程状态
echo "4. 网关进程资源占用："
docker stats --no-stream llm-gateway | tail -n 1
echo ""

# 5. 检查网络连接数
echo "5. 网关网络连接统计："
docker exec llm-gateway sh -c "netstat -an | grep ESTABLISHED | wc -l" 2>/dev/null | xargs echo "   ESTABLISHED 连接数:"
docker exec llm-gateway sh -c "netstat -an | grep TIME_WAIT | wc -l" 2>/dev/null | xargs echo "   TIME_WAIT 连接数:"
echo ""

# 6. 测试上游连通性（示例：测试 OpenAI）
echo "6. 上游连通性测试："
echo "   正在测试 api.openai.com..."
time_output=$(docker exec llm-gateway sh -c "time curl -s -o /dev/null -w '%{time_total}' https://api.openai.com/v1/models -H 'Authorization: Bearer invalid' 2>&1" 2>/dev/null)
echo "   OpenAI API 响应时间: ${time_output}s"
echo ""

# 7. 检查最近的错误日志
echo "7. 最近的网关错误日志（最近10条）："
docker logs llm-gateway --since 1h 2>&1 | grep -i "error\|timeout\|fail" | tail -n 10
echo ""

echo "=== 诊断完成 ==="
echo ""
echo "优化建议："
echo "1. 如果数据库延迟 >50ms，考虑优化数据库连接或增加连接池"
echo "2. 如果上游域名响应慢，考虑更换上游或使用 CDN"
echo "3. 如果 TIME_WAIT 连接数过多（>1000），说明连接复用不足"
echo "4. 检查上游 API 配额和限流状态"
