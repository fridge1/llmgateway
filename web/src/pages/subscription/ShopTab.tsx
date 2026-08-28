import { useState } from "react";
import { Zap, ChevronDown } from "lucide-react";
import type { SubscriptionPlan, SubscriptionWithUsage } from "@/types/api";
import {
  tierColors,
  isImagePlan,
  isTrialPlan,
  discountMap,
} from "./shared";

interface ShopTabProps {
  plans: SubscriptionPlan[] | undefined;
  activeSubs: SubscriptionWithUsage[];
  onBuyClick: (planId: number) => void;
  purchaseDisabled?: boolean;
  disabledReason?: string;
}

const ShopTab = ({ plans, activeSubs, onBuyClick, purchaseDisabled, disabledReason }: ShopTabProps) => {
  const [estimateHours, setEstimateHours] = useState(4);
  const [collapsedBrands, setCollapsedBrands] = useState<Record<string, boolean>>({});

  const estimatedMonthlyCost = estimateHours * 2.5 * 30;
  const estimateRecommendation =
    estimatedMonthlyCost < 100
      ? "建议：按量付费"
      : estimatedMonthlyCost < 200
        ? "建议：Claude 开发者版（¥99/月）"
        : estimatedMonthlyCost < 400
          ? "建议：Claude 专业版（¥299/月）"
          : estimatedMonthlyCost < 700
            ? "建议：Claude 团队版（¥599/月）"
            : "建议：Claude 无限版（¥999/月）";

  const allPlans = plans ?? [];
  const claudePlans = allPlans.filter(
    (p) => !p.name.startsWith("openai-") && !p.name.startsWith("image-"),
  );
  const openaiPlans = allPlans.filter((p) => p.name.startsWith("openai-"));
  const imagePlans = allPlans.filter((p) => p.name.startsWith("image-"));

  const toggleBrand = (brand: string) => {
    setCollapsedBrands((prev) => ({ ...prev, [brand]: !prev[brand] }));
  };

  const renderBuyButton = (plan: SubscriptionPlan) => {
    if (purchaseDisabled) {
      return (
        <div className="w-full py-2.5 rounded-lg text-sm font-semibold text-center text-orange-600 dark:text-orange-400 bg-orange-50 dark:bg-orange-500/10 border border-orange-200/60 dark:border-orange-500/30">
          {disabledReason || "租户已配置专属定价，无法购买套餐"}
        </div>
      );
    }
    if (isTrialPlan(plan.name)) {
      const hasActiveTrial = activeSubs.some(
        (item) => item.subscription.plan_id === plan.id,
      );
      if (hasActiveTrial) {
        return (
          <div className="w-full py-2.5 rounded-lg text-sm font-semibold text-center text-primary bg-primary/8 dark:bg-primary/15 border border-primary/20">
            已购买
          </div>
        );
      }
    }
    return (
      <button
        onClick={() => onBuyClick(plan.id)}
        className="w-full btn-primary py-2.5 text-sm font-semibold"
      >
        立即购买
      </button>
    );
  };

  const renderPlanCard = (plan: SubscriptionPlan, i: number) => {
    const colors = tierColors[plan.name] ?? tierColors.pro;
    const discount = discountMap[plan.name];
    const isOpenAI = plan.name.startsWith("openai-");

    return (
      <div
        key={plan.id}
        className="bg-card border border-border rounded-xl shadow-card p-5 flex flex-col transition-all duration-300 relative overflow-hidden stagger-item hover:shadow-elevated hover:-translate-y-0.5"
        style={{ animationDelay: `${i * 80}ms` }}
      >
        <div className={`absolute top-0 left-0 right-0 h-1 bg-gradient-to-r ${colors.topBar}`} />
        {discount && (
          <div
            className="absolute top-3 right-3 px-2 py-0.5 text-white text-xs font-semibold rounded"
            style={{
              background: isOpenAI
                ? "linear-gradient(135deg, #10B981, #059669)"
                : "linear-gradient(135deg, #4F46E5, #7C3AED)",
            }}
          >
            {discount}
          </div>
        )}
        <div className="flex items-center justify-between mb-3 mt-1">
          <div className={`px-2.5 py-1 rounded-full text-xs font-semibold border ${colors.badge}`}>
            {plan.display_name}
          </div>
        </div>
        <div className="mb-1">
          <span className="text-2xl font-bold text-foreground">¥{plan.monthly_price_cny}</span>
          <span className="text-sm text-muted-foreground">/{plan.duration_days <= 7 ? "周" : "月"}</span>
        </div>
        <div className="flex items-center gap-1.5 text-sm text-muted-foreground mb-4">
          <Zap size={13} className="text-primary" />
          {isImagePlan(plan.name) ? (
            <>包含 {Math.round(plan.quota_amount_cny)} 张图片生成</>
          ) : (
            <>包含 ¥{plan.quota_amount_cny} 使用额度</>
          )}
        </div>
        {plan.description && (
          <p className="text-xs text-muted-foreground mb-4 leading-relaxed">{plan.description}</p>
        )}
        {plan.models && plan.models.length > 0 && (
          <div className="mb-5 flex-1">
            <div className="text-xs font-medium text-muted-foreground mb-2">覆盖模型</div>
            <div className="flex flex-wrap gap-1.5">
              {plan.models.map((m) => (
                <span key={m} className="text-[11px] px-2 py-0.5 rounded-full bg-muted text-foreground border border-border">
                  {m}
                </span>
              ))}
            </div>
          </div>
        )}
        <div className="mt-auto">{renderBuyButton(plan)}</div>
      </div>
    );
  };

  const renderBrandSection = (
    key: string,
    title: string,
    accentClass: string,
    brandPlans: SubscriptionPlan[],
    cols: string,
  ) => {
    if (brandPlans.length === 0) return null;
    const collapsed = collapsedBrands[key];
    return (
      <div key={key}>
        <button
          onClick={() => toggleBrand(key)}
          className="w-full flex items-center justify-between mb-4 cursor-pointer group"
        >
          <h2 className="text-base font-semibold text-foreground flex items-center gap-2">
            <span className={`inline-block w-1 h-5 rounded-full bg-gradient-to-b ${accentClass}`} />
            {title}
            <span className="text-xs text-muted-foreground font-normal ml-1">{brandPlans.length} 个套餐</span>
          </h2>
          <ChevronDown
            size={18}
            className={`text-muted-foreground transition-transform duration-200 ${collapsed ? "-rotate-90" : ""}`}
          />
        </button>
        {!collapsed && (
          <div className={`grid ${cols} gap-5`}>
            {brandPlans.map((plan, i) => renderPlanCard(plan, i))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="space-y-6">
      <div className="bg-gradient-to-br from-indigo-50 to-violet-50 dark:from-indigo-500/5 dark:to-violet-500/5 border border-indigo-200/60 dark:border-indigo-500/15 rounded-xl shadow-card p-6">
        <h3 className="text-base font-bold text-foreground mb-3">用量估算器</h3>
        <p className="text-sm text-muted-foreground mb-4">
          根据您的使用习惯，估算每月 Claude Code 消费，帮助选择合适套餐。
        </p>
        <div className="bg-card rounded-lg p-4 border border-border">
          <div className="flex items-center gap-4 mb-3">
            <label className="text-sm font-medium text-foreground whitespace-nowrap">每天使用时长：</label>
            <input
              type="range"
              min="1"
              max="12"
              value={estimateHours}
              className="flex-1 accent-primary"
              onChange={(e) => setEstimateHours(parseInt(e.target.value))}
            />
            <span className="text-sm font-bold text-foreground whitespace-nowrap">{estimateHours}</span>
            <span className="text-sm text-muted-foreground">小时/天</span>
          </div>
          <div className="text-sm space-y-1">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">预估月消费：</span>
              <span className="text-lg font-bold text-foreground">¥{estimatedMonthlyCost.toFixed(0)}</span>
            </div>
            <div className="text-xs text-primary font-medium">{estimateRecommendation}</div>
          </div>
        </div>
      </div>

      {renderBrandSection(
        "claude",
        "Claude 套餐",
        "from-indigo-500 to-violet-500",
        claudePlans,
        "grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5",
      )}
      {renderBrandSection(
        "openai",
        "OpenAI 套餐",
        "from-green-500 to-emerald-500",
        openaiPlans,
        "grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5",
      )}
      {renderBrandSection(
        "image",
        "图片生成套餐",
        "from-orange-500 to-fuchsia-500",
        imagePlans,
        "grid-cols-1 md:grid-cols-2 lg:grid-cols-3",
      )}
    </div>
  );
};

export default ShopTab;
