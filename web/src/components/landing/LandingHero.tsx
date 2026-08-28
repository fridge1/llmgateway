import { useNavigate } from "react-router-dom";
import { ArrowRight, Sparkles, BookOpen } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";

const LandingHero = () => {
  const navigate = useNavigate();
  const auth = useAuth();

  const primaryCta = auth.isAuthenticated
    ? { label: "进入控制台", path: "/dashboard" }
    : { label: "免费注册", path: "/auth?tab=register" };

  return (
    <section className="relative overflow-hidden -mt-14 pt-14" style={{ background: "var(--hero-gradient)" }}>
      <div className="absolute inset-0 bg-grid-white/[0.05] bg-[size:60px_60px]" />
      <div className="absolute top-0 right-0 w-96 h-96 bg-indigo-500/30 rounded-full blur-3xl floating-shape" />
      <div className="absolute bottom-0 left-0 w-96 h-96 bg-purple-500/20 rounded-full blur-3xl floating-shape" style={{ animationDelay: "2s" }} />

      <div className="relative max-w-6xl mx-auto px-6 py-24 md:py-32">
        <div className="text-center max-w-3xl mx-auto slide-up">
          <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-white/10 border border-white/15 text-indigo-100 text-sm font-medium mb-6 backdrop-blur-sm">
            <Sparkles size={16} />
            支付宝充值 · 可开发票 · 国内直连低延迟
          </div>

          <h1 className="text-4xl md:text-6xl font-bold text-white mb-6 leading-tight tracking-tight">
            一个网关，调用
            <br />
            <span className="text-transparent bg-clip-text" style={{ backgroundImage: "linear-gradient(135deg, #A5B4FC, #C4B5FD)" }}>
              所有大模型
            </span>
          </h1>

          <p className="text-lg md:text-xl text-slate-300 mb-10 leading-relaxed">
            国内外主流模型统一接入。原生兼容 Claude Code、Cursor、Codex CLI，替换 BASE_URL 即可使用。
            <br className="hidden md:block" />
            按量计费，故障自动切换，每一笔调用都有可查账单。
          </p>

          <div className="flex flex-col sm:flex-row gap-3 justify-center">
            <button
              onClick={() => navigate(primaryCta.path)}
              className="px-8 py-3.5 brand-gradient text-white font-semibold rounded-xl shadow-elevated hover:brightness-110 active:scale-[0.97] transition-all duration-200 flex items-center gap-2 justify-center"
            >
              {primaryCta.label}
              <ArrowRight size={18} />
            </button>
            <button
              onClick={() => navigate("/docs")}
              className="px-8 py-3.5 bg-white/10 hover:bg-white/20 text-white font-semibold rounded-xl border border-white/20 active:scale-[0.97] transition-all duration-200 flex items-center gap-2 justify-center backdrop-blur-sm"
            >
              <BookOpen size={18} />
              查看文档
            </button>
          </div>

          {!auth.isAuthenticated && (
            <p className="mt-6 text-sm text-slate-400">
              注册即送试用额度，支付宝直充，5 分钟完成第一次调用
            </p>
          )}
        </div>
      </div>
    </section>
  );
};

export default LandingHero;
