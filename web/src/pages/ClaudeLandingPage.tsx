import { useNavigate } from "react-router-dom";
import { Check, ArrowRight, Code, Sparkles, DollarSign, Download } from "lucide-react";
import LandingFooter from "@/components/landing/LandingFooter";
import { Seo } from "@/components/Seo";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const ClaudeLandingPage = () => {
  const navigate = useNavigate();

  return (
    <div className="w-full min-h-screen bg-background">
      <Seo path="/claude" />
      {/* Hero Section */}
      <div className="relative overflow-hidden bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900">
        <div className="absolute inset-0 bg-grid-white/[0.05] bg-[size:60px_60px]" />
        <div className="absolute top-0 right-0 w-96 h-96 bg-blue-500/30 rounded-full blur-3xl" />
        <div className="absolute bottom-0 left-0 w-96 h-96 bg-orange-500/20 rounded-full blur-3xl" />

        <div className="relative max-w-6xl mx-auto px-6 py-20">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-orange-500/10 border border-orange-500/20 text-orange-300 text-sm font-medium mb-6">
              <Sparkles size={16} />
              国内 Claude 专用网关
            </div>
            <h1 className="text-5xl md:text-6xl font-bold text-white mb-6 leading-tight">
              国内用 Claude<br />
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-orange-400">
                就用 LLM Gateway
              </span>
            </h1>
            <p className="text-xl text-slate-300 mb-8 max-w-2xl mx-auto">
              原生 Anthropic API，完整支持 Prompt Cache 和 Extended Thinking<br />
              比官方便宜 20%，支付宝直充，国内直连低延迟
            </p>
            <div className="flex gap-4 justify-center">
              <button
                onClick={() => navigate("/auth?tab=register")}
                className="px-8 py-4 bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white font-semibold rounded-xl shadow-lg hover:shadow-xl transition-all duration-200 flex items-center gap-2"
              >
                立即注册
                <ArrowRight size={18} />
              </button>
              <button
                onClick={() => navigate("/docs")}
                className="px-8 py-4 bg-white/10 hover:bg-white/20 text-white font-semibold rounded-xl border border-white/20 transition-all duration-200"
              >
                查看文档
              </button>
              <button
                onClick={() => navigate("/download")}
                className="px-8 py-4 bg-white/10 hover:bg-white/20 text-white font-semibold rounded-xl border border-white/20 transition-all duration-200 flex items-center gap-2"
              >
                <Download size={18} />
                下载桌面端
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Features Section */}
      <div className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-center text-foreground mb-12">核心优势</h2>
        <div className="grid md:grid-cols-3 gap-8">
          <div className="bg-card border border-border rounded-xl p-6 shadow-card hover:shadow-elevated transition-all duration-200">
            <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center mb-4">
              <Code size={24} className="text-blue-600" />
            </div>
            <h3 className="text-xl font-bold text-foreground mb-3">原生 API 透传</h3>
            <p className="text-sm text-muted-foreground leading-relaxed">
              直接透传 Anthropic Messages API，不做格式转换。完整支持 system prompts、tool use、thinking 等所有原生特性。
            </p>
          </div>

          <div className="bg-card border border-border rounded-xl p-6 shadow-card hover:shadow-elevated transition-all duration-200">
            <div className="w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center mb-4">
              <Sparkles size={24} className="text-purple-600" />
            </div>
            <h3 className="text-xl font-bold text-foreground mb-3">Prompt Cache 支持</h3>
            <p className="text-sm text-muted-foreground leading-relaxed">
              精确支持 5min 和 1h TTL 的 Prompt Cache，缓存读取价格仅为正常输入的 10%，大幅降低长上下文成本。
            </p>
          </div>

          <div className="bg-card border border-border rounded-xl p-6 shadow-card hover:shadow-elevated transition-all duration-200">
            <div className="w-12 h-12 bg-orange-100 rounded-lg flex items-center justify-center mb-4">
              <DollarSign size={24} className="text-orange-600" />
            </div>
            <h3 className="text-xl font-bold text-foreground mb-3">便宜 20%</h3>
            <p className="text-sm text-muted-foreground leading-relaxed">
              所有 Claude 模型价格比 Anthropic 官方便宜约 20%，支持支付宝充值，无需国际信用卡和 VPN。
            </p>
          </div>
        </div>
      </div>

      {/* Price Comparison Section */}
      <div className="bg-slate-50 py-20">
        <div className="max-w-6xl mx-auto px-6">
          <h2 className="text-3xl font-bold text-center text-foreground mb-4">价格对比</h2>
          <p className="text-center text-muted-foreground mb-12">所有 Claude 模型均比官方便宜约 20%</p>

          <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
            <Table className="w-full">
              <TableHeader>
                <TableRow className="border-b border-border bg-muted/30">
                  <TableHead className="text-left px-6 py-4 font-semibold text-foreground">模型</TableHead>
                  <TableHead className="text-left px-6 py-4 font-semibold text-foreground">Anthropic 官方</TableHead>
                  <TableHead className="text-left px-6 py-4 font-semibold text-foreground">LLM Gateway</TableHead>
                  <TableHead className="text-left px-6 py-4 font-semibold text-foreground">节省</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow className="border-b border-border">
                  <TableCell className="px-6 py-4 font-medium text-foreground">Claude Haiku 4.5</TableCell>
                  <TableCell className="px-6 py-4 text-muted-foreground">$0.80 / $4.00 per M</TableCell>
                  <TableCell className="px-6 py-4 text-foreground font-medium">¥0.80 / ¥4.00 per M</TableCell>
                  <TableCell className="px-6 py-4 text-green-600 font-semibold">~20%</TableCell>
                </TableRow>
                <TableRow className="border-b border-border bg-orange-50/50">
                  <TableCell className="px-6 py-4 font-medium text-foreground">Claude Sonnet 4.6</TableCell>
                  <TableCell className="px-6 py-4 text-muted-foreground">$3.00 / $15.00 per M</TableCell>
                  <TableCell className="px-6 py-4 text-foreground font-medium">¥2.40 / ¥12.00 per M</TableCell>
                  <TableCell className="px-6 py-4 text-green-600 font-semibold">~20%</TableCell>
                </TableRow>
                <TableRow>
                  <TableCell className="px-6 py-4 font-medium text-foreground">Claude Opus 4.6</TableCell>
                  <TableCell className="px-6 py-4 text-muted-foreground">$5.00 / $25.00 per M</TableCell>
                  <TableCell className="px-6 py-4 text-foreground font-medium">¥4.00 / ¥20.00 per M</TableCell>
                  <TableCell className="px-6 py-4 text-green-600 font-semibold">~20%</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>

          <div className="mt-6 text-center text-sm text-muted-foreground">
            * 价格为输入 / 输出 token 单价（每百万 tokens）
          </div>
        </div>
      </div>

      {/* Setup Demo Section */}
      <div className="max-w-6xl mx-auto px-6 py-20">
        <h2 className="text-3xl font-bold text-center text-foreground mb-4">一键配置</h2>
        <p className="text-center text-muted-foreground mb-12">3 分钟完成 Claude Code 配置</p>

        <div className="bg-slate-900 rounded-xl p-8 shadow-xl">
          <div className="flex items-center gap-2 mb-4">
            <div className="w-3 h-3 rounded-full bg-red-500" />
            <div className="w-3 h-3 rounded-full bg-yellow-500" />
            <div className="w-3 h-3 rounded-full bg-green-500" />
            <span className="ml-4 text-sm text-slate-400">Terminal</span>
          </div>
          <pre className="text-green-400 font-mono text-sm overflow-x-auto">
            <code>$ curl -fsSL https://your-domain.com/setup-claude.sh | bash</code>
          </pre>
        </div>

        <div className="mt-8 grid md:grid-cols-3 gap-6">
          <div className="text-center">
            <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-3 text-blue-600 font-bold text-lg">
              1
            </div>
            <h3 className="font-semibold text-foreground mb-2">注册账号</h3>
            <p className="text-sm text-muted-foreground">手机号注册，获取 API Key</p>
          </div>
          <div className="text-center">
            <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-3 text-blue-600 font-bold text-lg">
              2
            </div>
            <h3 className="font-semibold text-foreground mb-2">运行脚本</h3>
            <p className="text-sm text-muted-foreground">一键配置 Claude Code</p>
          </div>
          <div className="text-center">
            <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-3 text-blue-600 font-bold text-lg">
              3
            </div>
            <h3 className="font-semibold text-foreground mb-2">开始编码</h3>
            <p className="text-sm text-muted-foreground">立即使用 Claude Code</p>
          </div>
        </div>
      </div>

      {/* Pricing Plans Section */}
      <div className="bg-slate-50 py-20">
        <div className="max-w-6xl mx-auto px-6">
          <h2 className="text-3xl font-bold text-center text-foreground mb-4">订阅套餐</h2>
          <p className="text-center text-muted-foreground mb-12">选择适合您的套餐，开始使用 Claude</p>

          <div className="grid md:grid-cols-3 gap-6">
            <div className="bg-card border border-border rounded-xl p-6 shadow-card">
              <h3 className="text-xl font-bold text-foreground mb-2">Claude 开发者版</h3>
              <div className="mb-4">
                <span className="text-3xl font-bold text-foreground">¥99</span>
                <span className="text-muted-foreground">/月</span>
              </div>
              <p className="text-sm text-muted-foreground mb-4">包含 ¥168 使用额度</p>
              <ul className="space-y-2 mb-6">
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  覆盖所有 Claude 模型
                </li>
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  支持 Prompt Cache
                </li>
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  支持 Extended Thinking
                </li>
              </ul>
              <button
                onClick={() => navigate("/subscription")}
                className="w-full py-3 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold rounded-lg transition-colors"
              >
                选择套餐
              </button>
            </div>

            <div className="bg-card border-2 border-orange-400 rounded-xl p-6 shadow-card relative">
              <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-1 bg-orange-400 text-white text-xs font-semibold rounded-full">
                推荐
              </div>
              <h3 className="text-xl font-bold text-foreground mb-2">Claude 专业版</h3>
              <div className="mb-4">
                <span className="text-3xl font-bold text-foreground">¥299</span>
                <span className="text-muted-foreground">/月</span>
              </div>
              <p className="text-sm text-muted-foreground mb-4">包含 ¥499 使用额度</p>
              <ul className="space-y-2 mb-6">
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  覆盖所有 Claude 模型
                </li>
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  支持 Prompt Cache
                </li>
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  支持 Extended Thinking
                </li>
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  适合每天 4-8 小时编码
                </li>
              </ul>
              <button
                onClick={() => navigate("/subscription")}
                className="w-full py-3 bg-orange-500 hover:bg-orange-600 text-white font-semibold rounded-lg transition-colors"
              >
                选择套餐
              </button>
            </div>

            <div className="bg-card border border-border rounded-xl p-6 shadow-card">
              <h3 className="text-xl font-bold text-foreground mb-2">Claude 无限版</h3>
              <div className="mb-4">
                <span className="text-3xl font-bold text-foreground">¥999</span>
                <span className="text-muted-foreground">/月</span>
              </div>
              <p className="text-sm text-muted-foreground mb-4">包含 ¥1998 使用额度</p>
              <ul className="space-y-2 mb-6">
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  覆盖所有 Claude 模型
                </li>
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  支持 Prompt Cache
                </li>
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  支持 Extended Thinking
                </li>
                <li className="flex items-center gap-2 text-sm text-foreground">
                  <Check size={16} className="text-green-600" />
                  不限量使用
                </li>
              </ul>
              <button
                onClick={() => navigate("/subscription")}
                className="w-full py-3 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold rounded-lg transition-colors"
              >
                选择套餐
              </button>
            </div>
          </div>

          <div className="mt-8 text-center">
            <button
              onClick={() => navigate("/subscription")}
              className="text-primary hover:underline font-medium"
            >
              查看所有套餐 →
            </button>
          </div>
        </div>
      </div>

      {/* CTA Section */}
      <div className="max-w-4xl mx-auto px-6 py-20 text-center">
        <h2 className="text-3xl font-bold text-foreground mb-4">立即开始使用 Claude</h2>
        <p className="text-lg text-muted-foreground mb-8">
          注册即送 ¥5 试用额度，无需信用卡
        </p>
        <button
          onClick={() => navigate("/auth?tab=register")}
          className="px-10 py-4 bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white font-semibold rounded-xl shadow-lg hover:shadow-xl transition-all duration-200 inline-flex items-center gap-2"
        >
          免费注册
          <ArrowRight size={20} />
        </button>
      </div>

      {/* Footer */}
      <LandingFooter />
    </div>
  );
};

export default ClaudeLandingPage;