export interface FaqItem {
  q: string;
  a: string;
}

/**
 * Source of truth for the public FAQ. Both the FAQ accordion (UI) and the
 * FAQPage JSON-LD (SEO) read from this list, so the answers stay in sync.
 */
export const FAQ_ITEMS: FaqItem[] = [
  {
    q: "和直接调官方 API 有什么区别？",
    a: "我们做了三件官方不直接提供的事：协议互转（用 OpenAI 客户端调 Anthropic 模型）、统一计费与配额管理（一份账单看所有模型）、上游故障自动切换。同时支持支付宝充值、可开发票，无需海外信用卡或境外网络环境。",
  },
  {
    q: "充值余额安全吗？会不会跑路？",
    a: "支持按量计费，余额随充随用、可随时在仪表盘查询每一笔消费明细。所有充值走支付宝官方通道，可申请开具发票，遇到问题可通过工单联系客服处理。",
  },
  {
    q: "如何计费？最小消费是多少？",
    a: "支持两种模式：按 Token 实时计费（精确到每次调用，无最低消费）和包月订阅（固定额度，超出后回退到按量）。账户余额可随时在仪表盘查看，所有交易都有记录。",
  },
  {
    q: "数据是否安全？请求会被记录吗？",
    a: "我们仅记录元数据（模型名、Token 数、时间戳），用于计费和用量审计。不存储请求内容和响应内容。所有 API Key 以 SHA-256 哈希存储，传输全程 TLS。",
  },
  {
    q: "支持哪些 AI 编码工具？",
    a: "原生兼容 Claude Code、Cursor、Codex CLI、Windsurf、Aider 等主流工具。配置时只需把 BASE_URL 指到我们的网关，API Key 换成我们颁发的即可，其余设置完全不变。",
  },
  {
    q: "企业账户和个人账户有什么不同？",
    a: "企业账户支持创建子用户、为子用户分配独立配额、按成员审计用量、定制定价、独立计费。适合 5 人以上团队或需要分账的组织使用。",
  },
  {
    q: "支持私有化部署吗？",
    a: "支持。我们提供基于 Docker Compose 的私有部署方案，你可以在自己的服务器上运行完整网关，自带 PostgreSQL，零外部依赖。请联系客服获取部署文档。",
  },
];
