export const SITE_URL = "https://your-domain.com";
export const SITE_NAME = "LLM Gateway";

export interface PageMeta {
  /** 完整标题（含品牌后缀） */
  title: string;
  description: string;
}

/**
 * 公开页 TDK 唯一数据源。
 * 两处消费：页面组件（<Seo path="...">，运行时 head 提升）
 * 与预渲染脚本（scripts/prerender.mjs，构建期替换 index.html 的静态标签）。
 * 新增公开页时：此处加一条 + public/sitemap.xml 加 <url> + App.tsx 的 PUBLIC_PATHS。
 */
export const PUBLIC_PAGE_META: Record<string, PageMeta> = {
  "/": {
    title: "LLM Gateway · 统一的大模型 API 网关",
    description:
      "一个网关对接所有大模型。统一接入 OpenAI、Anthropic、Gemini，原生协议透传，按量计费，故障自动切换。兼容 Claude Code、Cursor、Codex CLI 等主流 AI 编码工具。",
  },
  "/claude": {
    title: "国内用 Claude Code · Claude API 中国直连 · LLM Gateway",
    description:
      "国内 Claude 专用网关：原生 Anthropic API，完整支持 Prompt Cache 和 Extended Thinking，比官方便宜 20%，支付宝直充，国内直连低延迟。Claude Code 3 分钟一键配置。",
  },
  "/docs": {
    title: "接入文档 · Claude Code / Cursor / Codex CLI 配置教程 · LLM Gateway",
    description:
      "LLM Gateway 接入文档：Claude Code、Cursor、Codex CLI、Windsurf 等工具的配置教程，OpenAI / Anthropic / Gemini 三协议 API 说明，认证方式与错误处理。",
  },
  "/download": {
    title: "下载桌面端 · LLM Gateway",
    description:
      "下载 LLM Gateway 桌面客户端（macOS / Windows），本地管理 API Key、查看用量与账单，一键配置 Claude Code 等 AI 编码工具。",
  },
  "/enterprise": {
    title: "企业版 · 团队协作 · 子账号与分账 · 可开发票 · LLM Gateway",
    description:
      "LLM Gateway 企业版：企业租户与子账号、独立配额与按成员用量审计、专属定价、支付宝充值可开增值税发票、Docker 私有化部署。为 5 人以上团队与需要分账、合规的组织设计。",
  },
  "/qualifications": {
    title: "企业资质 · LLM Gateway",
    description: "企业资质与增值电信业务经营许可证信息。",
  },
};

export const ORGANIZATION_JSONLD = {
  "@context": "https://schema.org",
  "@type": "Organization",
  name: SITE_NAME,
  url: SITE_URL,
  logo: `${SITE_URL}/favicon.svg`,
};

export const WEBSITE_JSONLD = {
  "@context": "https://schema.org",
  "@type": "WebSite",
  name: SITE_NAME,
  url: SITE_URL,
};
