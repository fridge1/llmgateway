import { useNavigate } from "react-router-dom";
import { Check, Sparkles } from "lucide-react";
import { usePublicPlans, type PublicPlan } from "@/hooks/use-public";
import { useAuth } from "@/contexts/AuthContext";
import { Skeleton } from "@/components/ui/skeleton";

function describeQuota(p: PublicPlan): string[] {
  const lines: string[] = [];
  if (p.quota_amount_cny > 0) {
    lines.push(`包含 ¥${p.quota_amount_cny.toFixed(0)} 使用额度`);
  }
  if (p.duration_days > 0 && p.duration_days !== 30) {
    lines.push(`${p.duration_days} 天有效期`);
  }
  return lines;
}

const PlanCards = () => {
  const navigate = useNavigate();
  const auth = useAuth();
  const { data, isLoading } = usePublicPlans();
  const plans = data?.plans ?? [];

  const handleSelect = () => {
    if (auth.isAuthenticated) {
      navigate("/dashboard/subscription");
    } else {
      navigate("/auth?tab=register");
    }
  };

  return (
    <section id="plans" className="py-20">
      <div className="max-w-6xl mx-auto px-6">
        <div className="text-center mb-12">
          <h2 className="text-3xl md:text-4xl font-bold text-foreground tracking-tight mb-3">
            灵活的订阅套餐
          </h2>
          <p className="text-muted-foreground">
            按团队规模选择，随时升级，按比例退款
          </p>
        </div>

        {isLoading ? (
          <div className="grid md:grid-cols-3 gap-5">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-80 rounded-xl" />
            ))}
          </div>
        ) : plans.length === 0 ? (
          <p className="text-center text-muted-foreground">暂无可用套餐</p>
        ) : (
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {plans.slice(0, 6).map((p) => (
              <div
                key={p.id}
                className={`relative bg-card rounded-xl p-6 transition-all ${
                  p.recommended
                    ? "border-2 border-primary shadow-elevated md:scale-[1.02]"
                    : "border border-border hover:shadow-card"
                }`}
              >
                {p.recommended && (
                  <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-1 brand-gradient text-white text-xs font-semibold rounded-full flex items-center gap-1 shadow-button">
                    <Sparkles size={12} />
                    推荐
                  </div>
                )}

                <h3 className="text-lg font-bold text-foreground mb-2">{p.display_name}</h3>
                {p.description && (
                  <p className="text-sm text-muted-foreground mb-4 line-clamp-2">{p.description}</p>
                )}

                <div className="mb-5">
                  <span className="text-4xl font-bold text-foreground">¥{p.monthly_price_cny.toFixed(0)}</span>
                  <span className="text-muted-foreground ml-1">
                    /{p.duration_days === 7 ? "周" : "月"}
                  </span>
                </div>

                <ul className="space-y-2.5 mb-6 min-h-[120px]">
                  {describeQuota(p).map((line) => (
                    <li key={line} className="flex items-start gap-2 text-sm text-foreground">
                      <Check size={16} className="text-primary mt-0.5 flex-shrink-0" />
                      <span>{line}</span>
                    </li>
                  ))}
                  <li className="flex items-start gap-2 text-sm text-foreground">
                    <Check size={16} className="text-primary mt-0.5 flex-shrink-0" />
                    <span>无限次 API 调用</span>
                  </li>
                  <li className="flex items-start gap-2 text-sm text-foreground">
                    <Check size={16} className="text-primary mt-0.5 flex-shrink-0" />
                    <span>邮件 / 微信技术支持</span>
                  </li>
                </ul>

                <button
                  onClick={handleSelect}
                  className={`w-full py-2.5 font-semibold rounded-lg transition-colors ${
                    p.recommended
                      ? "btn-primary"
                      : "bg-muted text-foreground hover:bg-muted/70 active:scale-[0.97] transition-all"
                  }`}
                >
                  {auth.isAuthenticated ? "立即订阅" : "免费注册并订阅"}
                </button>
              </div>
            ))}
          </div>
        )}

        {auth.isAuthenticated && (
          <div className="mt-8 text-center">
            <button
              onClick={() => navigate("/dashboard/subscription")}
              className="text-primary hover:underline font-medium text-sm"
            >
              查看完整套餐对比 →
            </button>
          </div>
        )}
      </div>
    </section>
  );
};

export default PlanCards;
