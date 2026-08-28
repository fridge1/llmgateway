#!/bin/bash
# Claude Code 一键配置脚本 - LLM Gateway
# 使用方法: curl -fsSL https://your-domain.com/setup-claude.sh | bash

set -e

echo "🚀 配置 Claude Code 使用 LLM Gateway..."
echo ""

# 检测操作系统
if [[ "$OSTYPE" == "darwin"* ]]; then
    CONFIG_DIR="$HOME/Library/Application Support/Claude"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    CONFIG_DIR="$HOME/.config/claude"
else
    echo "❌ 不支持的操作系统: $OSTYPE"
    echo "   目前仅支持 macOS 和 Linux"
    exit 1
fi

# 创建配置目录
mkdir -p "$CONFIG_DIR"

# 提示用户输入 API Key
echo "请输入您的 LLM Gateway API Key:"
echo "（在 https://your-domain.com/keys 获取）"
echo ""
read -p "API Key: " API_KEY

if [ -z "$API_KEY" ]; then
    echo ""
    echo "❌ API Key 不能为空"
    exit 1
fi

# 写入配置文件
cat > "$CONFIG_DIR/config.json" <<EOF
{
  "api": {
    "baseURL": "https://your-domain.com/v1",
    "apiKey": "$API_KEY"
  }
}
EOF

echo ""
echo "✅ 配置完成！"
echo ""
echo "配置文件已保存到: $CONFIG_DIR/config.json"
echo ""
echo "现在可以使用 Claude Code 了："
echo "  claude --help"
echo ""
echo "💡 优势："
echo "  ✓ 原生 Anthropic Messages API"
echo "  ✓ 完整支持 Prompt Cache 和 Extended Thinking"
echo "  ✓ 比官方便宜 20%"
echo "  ✓ 国内直连，低延迟"
echo ""
echo "📚 更多文档: https://your-domain.com/docs"
