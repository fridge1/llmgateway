import { Terminal, CreditCard, Zap, Building2, ShieldCheck, Layers } from "lucide-react";
import type { ReactNode } from "react";

interface Feature {
  icon: ReactNode;
  title: string;
  desc: string;
  color: string;
}

const FEATURES: Feature[] = [
  {
    icon: <Terminal size={22} className="text-white" />,
    title: "零改造接入",
    desc: "原生兼容 Claude Code、Cursor、Codex CLI、Windsurf，替换 BASE_URL 即可使用，无需重写任何代码。",
    color: "from-slate-600 to-slate-700",
  },
  {
    icon: <CreditCard size={22} className="text-white" />,
    title: "支付宝充值 · 可开发票",
    desc: "人民币支付宝直充，按 Token 实时计费或按月订阅自由切换，余额清晰可查，企业可申请开具发票。",
    color: "from-blue-500 to-indigo-500",
  },
  {
    icon: <Zap size={22} className="text-white" />,
    title: "故障自动切换",
    desc: "内置熔断与负载均衡，单个上游故障时自动切换到健康节点，调用不中断，业务无感知。",
    color: "from-amber-500 to-orange-500",
  },
  {
    icon: <ShieldCheck size={22} className="text-white" />,
    title: "每一笔都可查",
    desc: "每次调用都有可追溯的交易记录，支持按 API Key、模型、时间维度查询，充值消费明明白白。",
    color: "from-rose-500 to-pink-500",
  },
  {
    icon: <Building2 size={22} className="text-white" />,
    title: "团队与子账号",
    desc: "企业账户支持子用户、独立配额、按成员审计用量，团队协作与成本分账一站到位。",
    color: "from-emerald-500 to-teal-500",
  },
  {
    icon: <Layers size={22} className="text-white" />,
    title: "一个 Key 调用所有模型",
    desc: "OpenAI / Anthropic / Gemini 协议互转，一次接入即可调用 Claude、GPT、Gemini 等所有已配置模型。",
    color: "from-indigo-500 to-purple-500",
  },
];

const FeatureCards = () => {
  return (
    <section id="features" className="py-20 bg-muted/30">
      <div className="max-w-6xl mx-auto px-6">
        <div className="text-center mb-12">
          <h2 className="text-3xl md:text-4xl font-bold text-foreground tracking-tight mb-3">
            为开发者与团队设计
          </h2>
          <p className="text-muted-foreground max-w-xl mx-auto">
            从个人开发到企业团队，都能找到合适的接入方式
          </p>
        </div>

        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
          {FEATURES.map((f) => (
            <div
              key={f.title}
              className="group bg-card border border-border rounded-xl p-6 shadow-card hover:shadow-elevated hover:border-primary/40 hover:-translate-y-1 transition-all duration-300 ease-[cubic-bezier(0.22,1,0.36,1)]"
            >
              <div className={`w-11 h-11 rounded-xl bg-gradient-to-br ${f.color} flex items-center justify-center mb-4 shadow-button transition-transform duration-300 group-hover:scale-110`}>
                {f.icon}
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">{f.title}</h3>
              <p className="text-sm text-muted-foreground leading-relaxed">{f.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default FeatureCards;
