import { useNavigate } from "react-router-dom";
import {
  ArrowRight,
  Building2,
  Users,
  FileText,
  ShieldCheck,
  BarChart3,
  Server,
  Wallet,
  Check,
} from "lucide-react";
import LandingFooter from "@/components/landing/LandingFooter";
import { Seo } from "@/components/Seo";

interface Capability {
  icon: typeof Building2;
  title: string;
  desc: string;
}

// 企业版卖点全部对应代码中已上线的能力（租户/子用户/发票/审计/私有部署），
// 不含任何未确认的价格或 SLA 数字承诺。
const CAPABILITIES: Capability[] = [
  {
    icon: Users,
    title: "组织与子账号",
    desc: "创建企业组织，邀请成员并分配角色。每个子账号独立 API Key、独立配额，互不影响。",
  },
  {
    icon: BarChart3,
    title: "按成员用量分账",
    desc: "查看每位成员、每个 Key、每个模型的用量与费用明细，团队成本一目了然，方便内部分摊。",
  },
  {
    icon: FileText,
    title: "增值税发票",
    desc: "支持维护发票抬头、在线提交开票申请，充值消费可开具发票，满足企业报销与财务合规。",
  },
  {
    icon: Wallet,
    title: "统一账户与专属定价",
    desc: "组织统一充值、统一结算，可为租户配置专属定价与折扣，一份账单覆盖全部模型消费。",
  },
  {
    icon: ShieldCheck,
    title: "全链路用量审计",
    desc: "每次调用都有可追溯的交易记录，支持按成员、API Key、模型、时间维度查询与导出。",
  },
  {
    icon: Server,
    title: "私有化部署",
    desc: "提供基于 Docker Compose 的私有部署方案，在自有服务器上运行完整网关，数据不出内网。",
  },
];

const EnterprisePage = () => {
  const navigate = useNavigate();

  return (
    <div className="w-full min-h-screen bg-background">
      <Seo path="/enterprise" />

      {/* Hero */}
      <div className="relative overflow-hidden bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900">
        <div className="absolute inset-0 bg-grid-white/[0.05] bg-[size:60px_60px]" />
        <div className="absolute top-0 right-0 w-96 h-96 bg-blue-500/30 rounded-full blur-3xl" />
        <div className="absolute bottom-0 left-0 w-96 h-96 bg-emerald-500/20 rounded-full blur-3xl" />

        <div className="relative max-w-6xl mx-auto px-6 py-20 md:py-28">
          <div className="text-center max-w-3xl mx-auto">
            <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 text-sm font-medium mb-6">
              <Building2 size={16} />
              为团队与企业设计
            </div>
            <h1 className="text-4xl md:text-6xl font-bold text-white mb-6 leading-tight tracking-tight">
              让团队统一、安全地
              <br />
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-emerald-400">
                用上主流大模型
              </span>
            </h1>
            <p className="text-lg md:text-xl text-slate-300 mb-8 leading-relaxed">
              子账号与配额、按成员分账、增值税发票、用量审计、私有化部署，
              <br className="hidden md:block" />
              企业采购和财务合规需要的能力，一站备齐。
            </p>
            <div className="flex flex-col sm:flex-row gap-3 justify-center">
              <button
                onClick={() => navigate("/auth?tab=register")}
                className="px-8 py-4 bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white font-semibold rounded-xl shadow-lg hover:shadow-xl transition-all duration-200 inline-flex items-center gap-2 justify-center"
              >
                免费注册
                <ArrowRight size={18} />
              </button>
              <button
                onClick={() => navigate("/dashboard/tickets")}
                className="px-8 py-4 bg-white/10 hover:bg-white/20 text-white font-semibold rounded-xl border border-white/20 transition-all duration-200"
              >
                联系我们
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Capabilities */}
      <div className="max-w-6xl mx-auto px-6 py-20">
        <div className="text-center mb-12">
          <h2 className="text-3xl md:text-4xl font-bold text-foreground tracking-tight mb-3">
            企业需要的，不只是 API
          </h2>
          <p className="text-muted-foreground max-w-xl mx-auto">
            从成员管理到财务合规，把团队用模型的每一环都管起来
          </p>
        </div>

        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
          {CAPABILITIES.map((c) => {
            const Icon = c.icon;
            return (
              <div
                key={c.title}
                className="bg-card border border-border rounded-xl p-6 hover:shadow-elevated hover:border-primary/40 transition-all"
              >
                <div className="w-11 h-11 rounded-xl bg-gradient-to-br from-blue-500 to-emerald-500 flex items-center justify-center mb-4 shadow-button">
                  <Icon size={22} className="text-white" />
                </div>
                <h3 className="text-lg font-semibold text-foreground mb-2">{c.title}</h3>
                <p className="text-sm text-muted-foreground leading-relaxed">{c.desc}</p>
              </div>
            );
          })}
        </div>
      </div>

      {/* Why us — 信任要点 */}
      <div className="bg-muted/30 py-20">
        <div className="max-w-4xl mx-auto px-6">
          <div className="text-center mb-12">
            <h2 className="text-3xl md:text-4xl font-bold text-foreground tracking-tight mb-3">
              为什么企业选择我们
            </h2>
          </div>
          <div className="grid sm:grid-cols-2 gap-4">
            {[
              "支付宝对公充值，可开具增值税发票，报销合规无障碍",
              "多协议原生兼容，团队现有的 Claude Code / Cursor / Codex 工具零改造接入",
              "上游故障自动切换，一个模型出问题不影响团队整体使用",
              "每笔调用可追溯、可导出，用量与成本对财务透明",
              "支持私有化部署，敏感场景数据不出内网",
              "工单与专属对接，问题有人跟进",
            ].map((line) => (
              <div key={line} className="flex items-start gap-3 bg-card border border-border rounded-xl p-5">
                <Check size={20} className="text-emerald-600 shrink-0 mt-0.5" />
                <span className="text-sm text-foreground leading-relaxed">{line}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* CTA */}
      <div className="max-w-4xl mx-auto px-6 py-20 text-center">
        <h2 className="text-3xl font-bold text-foreground mb-4">为你的团队开通企业账户</h2>
        <p className="text-lg text-muted-foreground mb-8">
          注册后即可创建组织、邀请成员，需要专属定价或私有部署方案可随时联系我们
        </p>
        <div className="flex flex-col sm:flex-row gap-3 justify-center">
          <button
            onClick={() => navigate("/auth?tab=register")}
            className="px-10 py-4 bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white font-semibold rounded-xl shadow-lg hover:shadow-xl transition-all duration-200 inline-flex items-center gap-2 justify-center"
          >
            免费注册
            <ArrowRight size={20} />
          </button>
          <button
            onClick={() => navigate("/docs")}
            className="px-10 py-4 bg-muted hover:bg-muted/80 text-foreground font-semibold rounded-xl border border-border transition-all"
          >
            查看接入文档
          </button>
        </div>
      </div>

      <LandingFooter />
    </div>
  );
};

export default EnterprisePage;
