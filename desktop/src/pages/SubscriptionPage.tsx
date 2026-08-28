import { useState } from "react";
import {
  useSubscriptionPlans, useSubscriptionCurrent, useSubscribe,
  useSubscriptionHistory, useCreateSubscriptionPayment,
} from "@/hooks/use-api";
import { Loader2, AlertCircle, ChevronDown } from "../components/icons";
import type { SubscriptionPlan, SubscriptionWithUsage, SubscribeResponse } from "@/lib/types-api";

function getBrand(name: string): string {
  if (name.startsWith("openai-")) return "openai";
  if (name.startsWith("image-")) return "image";
  return "claude";
}

const isTrialPlan = (name: string) => name === "trial" || name === "openai-trial";
const isImagePlan = (name: string) => name.startsWith("image-");
const IMAGE_UNIT_PRICE = 0.08;
const cnyToImageCount = (cny: number) => Math.floor(cny / IMAGE_UNIT_PRICE);

function HistoryTab() {
  const { data: items = [], isLoading } = useSubscriptionHistory();
  if (isLoading) return <div className="py-10 text-center"><Loader2 size={16} className="animate-spin text-amber-400 inline" /></div>;
  if (items.length === 0) return <div className="py-10 text-center text-xs text-obsidian-500">暂无购买记录</div>;
  return (
    <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
      <table className="w-full">
        <thead>
          <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
            <th className="text-left px-4 py-2 font-medium">套餐</th>
            <th className="text-left px-4 py-2 font-medium">状态</th>
            <th className="text-left px-4 py-2 font-medium">到期时间</th>
            <th className="text-left px-4 py-2 font-medium">购买时间</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
              <td className="px-4 py-2.5 text-sm font-medium text-obsidian-100">
                {item.plan?.display_name ?? item.plan?.name ?? `套餐 ${item.plan_id}`}
              </td>
              <td className="px-4 py-2.5">
                <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                  item.status === "active" ? "bg-emerald-500/10 text-emerald-400" :
                  item.status === "expired" ? "bg-obsidian-700 text-obsidian-400" :
                  "bg-amber-500/10 text-amber-400"
                }`}>
                  {item.status === "active" ? "生效中" : item.status === "expired" ? "已过期" : item.status}
                </span>
              </td>
              <td className="px-4 py-2.5 text-xs text-obsidian-400">
                {new Date(item.expires_at).toLocaleDateString("zh-CN")}
              </td>
              <td className="px-4 py-2.5 text-xs text-obsidian-400">
                {new Date(item.created_at).toLocaleString("zh-CN")}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function SubscriptionPage() {
  const { data: plansData, isLoading: plansLoading } = useSubscriptionPlans();
  const { data: currentData, isLoading: currentLoading } = useSubscriptionCurrent();
  const subscribe = useSubscribe();
  const createPayment = useCreateSubscriptionPayment();

  const plans = plansData?.plans ?? [];
  const activeSubs: SubscriptionWithUsage[] = currentData?.subscriptions ?? [];
  const hasAnySub = activeSubs.length > 0;
  const isLoading = plansLoading || currentLoading;

  const [activeTab, setActiveTab] = useState<"shop" | "current" | "history">("shop");
  const [confirmPlanId, setConfirmPlanId] = useState<number | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [paymentInfo, setPaymentInfo] = useState<SubscribeResponse | null>(null);
  const [estimateHours, setEstimateHours] = useState(4);
  const [collapsedBrands, setCollapsedBrands] = useState<Record<string, boolean>>({});

  const estimatedMonthlyCost = estimateHours * 2.5 * 30;
  const estimateRecommendation = estimatedMonthlyCost < 100
    ? "建议：按量付费"
    : estimatedMonthlyCost < 200
      ? "建议：Claude 开发者版（¥99/月）"
      : estimatedMonthlyCost < 400
        ? "建议：Claude 专业版（¥299/月）"
        : estimatedMonthlyCost < 700
          ? "建议：Claude 团队版（¥599/月）"
          : "建议：Claude 无限版（¥999/月）";

  const handleSubscribe = async () => {
    if (!confirmPlanId) return;
    setActionError(null);
    setPaymentInfo(null);
    try {
      const result = await subscribe.mutateAsync({ planId: confirmPlanId, clientType: "desktop" });
      if (result.need_payment) {
        setPaymentInfo(result);
      } else {
        setConfirmPlanId(null);
      }
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : "操作失败，请重试");
    }
  };

  const handleGoToPay = async () => {
    if (!confirmPlanId) return;
    setActionError(null);
    try {
      const result = await createPayment.mutateAsync({ planId: confirmPlanId, clientType: "desktop" });
      if (result.pay_url) {
        const { openUrl } = await import("@tauri-apps/plugin-opener");
        await openUrl(result.pay_url);
      }
      setConfirmPlanId(null);
      setPaymentInfo(null);
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : "创建支付订单失败");
    }
  };

  const closeConfirm = () => {
    setConfirmPlanId(null);
    setActionError(null);
    setPaymentInfo(null);
  };

  const toggleBrand = (key: string) => {
    setCollapsedBrands((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  const getButtonForPlan = (plan: SubscriptionPlan) => {
    if (isTrialPlan(plan.name)) {
      const hasActiveTrial = activeSubs.some((item) => item.subscription.plan_id === plan.id);
      if (hasActiveTrial) {
        return (
          <div className="w-full py-2 rounded-lg text-xs font-semibold text-center text-amber-400 bg-amber-500/10 border border-amber-500/20">
            已购买
          </div>
        );
      }
    }
    const disabled = plansData?.purchase_disabled ?? false;
    return (
      <button
        onClick={() => !disabled && setConfirmPlanId(plan.id)}
        disabled={disabled}
        className="w-full py-2 rounded-lg text-xs font-semibold bg-amber-500 hover:bg-amber-400 text-obsidian-950 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
      >
        立即购买
      </button>
    );
  };

  const renderPlanCard = (plan: SubscriptionPlan) => (
    <div
      key={plan.id}
      className="bg-obsidian-900 border border-obsidian-700 hover:border-obsidian-600 rounded-xl p-4 flex flex-col transition-all duration-200"
    >
      <div className="text-sm font-semibold text-obsidian-50 mb-1">{plan.display_name || plan.name}</div>
      <div className="text-2xl font-bold text-amber-400 mb-1">
        ¥{plan.monthly_price_cny.toFixed(0)}
        <span className="text-xs text-obsidian-400 font-normal">/{plan.duration_days <= 7 ? "周" : "月"}</span>
      </div>
      <div className="text-xs text-obsidian-400 mb-3">
        {isImagePlan(plan.name)
          ? `包含 ${cnyToImageCount(plan.quota_amount_cny)} 张图片生成`
          : `包含 ¥${plan.quota_amount_cny.toFixed(0)} 使用额度`}
      </div>
      {plan.description && <div className="text-xs text-obsidian-500 mb-3">{plan.description}</div>}
      {plan.models && plan.models.length > 0 && (
        <div className="mb-3 flex-1">
          <div className="text-[10px] text-obsidian-500 mb-1.5">覆盖模型</div>
          <div className="flex flex-wrap gap-1">
            {plan.models.map((m) => (
              <span key={m} className="px-1.5 py-0.5 rounded text-[10px] bg-obsidian-800 text-obsidian-300">{m}</span>
            ))}
          </div>
        </div>
      )}
      <div className="mt-auto">{getButtonForPlan(plan)}</div>
    </div>
  );

  const renderActiveSubscriptions = () => {
    if (activeSubs.length === 0) return null;
    const grouped: Record<string, SubscriptionWithUsage[]> = {};
    for (const item of activeSubs) {
      const plan = plans.find(p => p.id === item.subscription.plan_id);
      const brand = plan ? getBrand(plan.name) : "claude";
      (grouped[brand] ??= []).push(item);
    }
    const brandColors: Record<string, string> = {
      claude: "from-indigo-500 to-violet-500",
      openai: "from-green-500 to-emerald-500",
      image: "from-orange-500 to-fuchsia-500",
    };
    const brandLabels: Record<string, string> = { claude: "Claude", openai: "OpenAI", image: "图片生成" };

    return Object.entries(grouped).map(([brand, items]) => (
      <div key={brand} className="mb-4">
        <button onClick={() => toggleBrand(`sub_${brand}`)} className="w-full flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            <span className={`inline-block w-1 h-4 rounded-full bg-gradient-to-b ${brandColors[brand] ?? brandColors.claude}`} />
            <span className="text-sm font-semibold text-obsidian-200">{brandLabels[brand] ?? brand} 订阅</span>
            <span className="text-xs text-obsidian-500">{items.length} 个</span>
          </div>
          <ChevronDown size={14} className={`text-obsidian-500 transition-transform duration-200 ${collapsedBrands[`sub_${brand}`] ? "-rotate-90" : ""}`} />
        </button>
        {!collapsedBrands[`sub_${brand}`] && items.map((item) => {
          const sub = item.subscription;
          const usage = item.usage;
          const plan = plans.find(p => p.id === sub.plan_id);
          const usagePct = usage?.usage_percent ?? 0;
          return (
            <div key={sub.id} className="bg-obsidian-900 border border-amber-500/20 rounded-xl p-4 mb-3">
              <div className="flex items-center gap-3 mb-3">
                <div className="text-sm font-semibold text-obsidian-50">{plan?.display_name ?? plan?.name ?? "—"}</div>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-emerald-500/10 text-emerald-400">生效中</span>
                <span className="text-xs text-obsidian-400">到期：{new Date(sub.expires_at).toLocaleDateString("zh-CN")}</span>
              </div>
              {usage && (
                <div>
                  <div className="flex items-center justify-between text-xs text-obsidian-400 mb-1">
                    <span>本期用量</span>
                    <span className="text-obsidian-200">
                      {isImagePlan(plan?.name ?? "")
                        ? `${cnyToImageCount(usage.total_amount_used)} / ${cnyToImageCount(usage.quota_amount_cny)} 张图片`
                        : `¥${usage.total_amount_used.toFixed(2)} / ¥${usage.quota_amount_cny.toFixed(2)}`}
                    </span>
                  </div>
                  <div className="h-2 rounded-full bg-obsidian-800">
                    <div
                      className="h-full rounded-full transition-all duration-300"
                      style={{
                        width: `${Math.min(usagePct, 100)}%`,
                        background: usagePct > 90 ? "#ef4444" : usagePct > 70 ? "#f59e0b" : "linear-gradient(90deg, #f59e0b, #d97706)",
                      }}
                    />
                  </div>
                  <div className="text-[10px] text-obsidian-500 mt-1">
                    已使用 {usagePct.toFixed(1)}%{usagePct >= 100 && "，超出部分将从余额按量扣费"}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    ));
  };

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}>
        <Loader2 size={24} className="animate-spin text-amber-400" />
      </div>
    );
  }

  const claudePlans = plans.filter(p => !p.name.startsWith("openai-") && !p.name.startsWith("image-"));
  const openaiPlans = plans.filter(p => p.name.startsWith("openai-"));
  const imagePlans = plans.filter(p => p.name.startsWith("image-"));
  const confirmPlan = confirmPlanId ? plans.find(p => p.id === confirmPlanId) : null;

  return (
    <div className="p-6">
      <div className="mb-5">
        <h1 className="text-lg font-semibold text-obsidian-50">订阅套餐</h1>
        <p className="text-xs text-obsidian-400 mt-0.5">选择适合您的订阅计划</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-obsidian-800 rounded-lg p-0.5 mb-5 w-fit">
        {([
          { key: "shop", label: "购买套餐" },
          { key: "current", label: "我的套餐" },
          { key: "history", label: "购买记录" },
        ] as const).map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={`px-4 py-1.5 rounded-md text-xs font-medium transition-all duration-200 ${
              activeTab === tab.key ? "bg-amber-500 text-obsidian-950" : "text-obsidian-400 hover:text-obsidian-200"
            }`}
          >
            {tab.label}
            {tab.key === "current" && hasAnySub && (
              <span className="ml-1 text-[10px] px-1 py-0.5 rounded-full bg-amber-500/20 text-amber-300">{activeSubs.length}</span>
            )}
          </button>
        ))}
      </div>

      {/* Current tab */}
      {activeTab === "current" && (
        <>
          {activeSubs.length === 0 ? (
            <div className="py-12 text-center text-sm text-obsidian-500">暂无生效套餐</div>
          ) : (
            renderActiveSubscriptions()
          )}
        </>
      )}

      {/* History tab */}
      {activeTab === "history" && <HistoryTab />}

      {/* Shop tab */}
      {activeTab === "shop" && (
        <>
          {/* Usage Estimator (desktop-only) */}
          {!hasAnySub && (
            <div className="bg-obsidian-900 border border-amber-500/20 rounded-xl p-5 mb-6">
              <h3 className="text-sm font-semibold text-obsidian-50 mb-2">用量估算器</h3>
              <p className="text-xs text-obsidian-400 mb-3">根据您的使用习惯，估算每月 Claude Code 消费，帮助选择合适套餐。</p>
              <div className="bg-obsidian-800/50 rounded-lg p-3">
                <div className="flex items-center gap-3 mb-2">
                  <label className="text-xs text-obsidian-300 whitespace-nowrap">每天使用时长：</label>
                  <input
                    type="range" min="1" max="12" value={estimateHours}
                    className="flex-1 accent-amber-500"
                    onChange={(e) => setEstimateHours(parseInt(e.target.value))}
                  />
                  <span className="text-xs font-bold text-obsidian-50 whitespace-nowrap">{estimateHours} 小时/天</span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-obsidian-400">预估月消费：</span>
                  <span className="text-base font-bold text-amber-400">¥{estimatedMonthlyCost.toFixed(0)}</span>
                </div>
                <div className="text-[10px] text-amber-400/80 mt-1">{estimateRecommendation}</div>
              </div>
            </div>
          )}

          {plansData?.purchase_disabled && plansData.disabled_reason && (
            <div className="flex items-center gap-2 bg-amber-500/10 border border-amber-500/20 rounded-xl px-4 py-3 mb-4 text-xs text-amber-400">
              <AlertCircle size={14} className="shrink-0" />
              {plansData.disabled_reason}
            </div>
          )}

          <div className="space-y-8">
            {claudePlans.length > 0 && (
              <div>
                <button onClick={() => toggleBrand("claude")} className="w-full flex items-center justify-between mb-3">
                  <h2 className="text-sm font-semibold text-obsidian-200 flex items-center gap-2">
                    <span className="inline-block w-1 h-4 rounded-full bg-gradient-to-b from-indigo-500 to-violet-500" />
                    Claude 套餐
                    <span className="text-xs text-obsidian-500 font-normal ml-1">{claudePlans.length} 个</span>
                  </h2>
                  <ChevronDown size={14} className={`text-obsidian-500 transition-transform duration-200 ${collapsedBrands["claude"] ? "-rotate-90" : ""}`} />
                </button>
                {!collapsedBrands["claude"] && (
                  <div className="grid grid-cols-3 gap-3">{claudePlans.map(plan => renderPlanCard(plan))}</div>
                )}
              </div>
            )}
            {openaiPlans.length > 0 && (
              <div>
                <button onClick={() => toggleBrand("openai")} className="w-full flex items-center justify-between mb-3">
                  <h2 className="text-sm font-semibold text-obsidian-200 flex items-center gap-2">
                    <span className="inline-block w-1 h-4 rounded-full bg-gradient-to-b from-green-500 to-emerald-500" />
                    OpenAI 套餐
                    <span className="text-xs text-obsidian-500 font-normal ml-1">{openaiPlans.length} 个</span>
                  </h2>
                  <ChevronDown size={14} className={`text-obsidian-500 transition-transform duration-200 ${collapsedBrands["openai"] ? "-rotate-90" : ""}`} />
                </button>
                {!collapsedBrands["openai"] && (
                  <div className="grid grid-cols-3 gap-3">{openaiPlans.map(plan => renderPlanCard(plan))}</div>
                )}
              </div>
            )}
            {imagePlans.length > 0 && (
              <div>
                <button onClick={() => toggleBrand("image")} className="w-full flex items-center justify-between mb-3">
                  <h2 className="text-sm font-semibold text-obsidian-200 flex items-center gap-2">
                    <span className="inline-block w-1 h-4 rounded-full bg-gradient-to-b from-orange-500 to-fuchsia-500" />
                    图片生成套餐
                    <span className="text-xs text-obsidian-500 font-normal ml-1">{imagePlans.length} 个</span>
                  </h2>
                  <ChevronDown size={14} className={`text-obsidian-500 transition-transform duration-200 ${collapsedBrands["image"] ? "-rotate-90" : ""}`} />
                </button>
                {!collapsedBrands["image"] && (
                  <div className="grid grid-cols-3 gap-3">{imagePlans.map(plan => renderPlanCard(plan))}</div>
                )}
              </div>
            )}
            {plans.length === 0 && (
              <div className="py-16 text-center text-sm text-obsidian-500">暂无可用套餐</div>
            )}
          </div>
        </>
      )}

      {/* Confirm purchase modal */}
      {confirmPlanId && confirmPlan && !paymentInfo && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm" onClick={closeConfirm}>
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[400px] p-6" onClick={e => e.stopPropagation()}>
            <h3 className="text-base font-semibold text-obsidian-50 mb-2">确认购买</h3>
            <p className="text-sm text-obsidian-400 mb-5">
              确认购买 {confirmPlan.display_name} 套餐？将从余额扣除 ¥{confirmPlan.monthly_price_cny}，有效期 {confirmPlan.duration_days} 天，从购买日起计算。
            </p>
            {actionError && (
              <div className="flex items-center gap-2 text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2.5 mb-4">
                <AlertCircle size={14} className="flex-shrink-0" />
                <span>{actionError}</span>
              </div>
            )}
            <div className="flex gap-3">
              <button onClick={closeConfirm} className="flex-1 py-2 text-sm text-obsidian-300 bg-obsidian-800 rounded-lg hover:bg-obsidian-700 transition-colors">取消</button>
              <button
                onClick={handleSubscribe}
                disabled={subscribe.isPending}
                className="flex-1 py-2 rounded-lg text-sm font-semibold bg-amber-500 hover:bg-amber-400 text-obsidian-950 transition-colors flex items-center justify-center gap-1.5 disabled:opacity-50"
              >
                {subscribe.isPending && <Loader2 size={14} className="animate-spin" />}
                确认购买
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Insufficient balance modal */}
      {paymentInfo && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm" onClick={closeConfirm}>
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[400px] p-6" onClick={e => e.stopPropagation()}>
            <h3 className="text-base font-semibold text-obsidian-50 mb-4">余额不足</h3>
            <div className="space-y-2.5 mb-5">
              <div className="flex justify-between text-sm">
                <span className="text-obsidian-400">套餐价格</span>
                <span className="font-medium text-obsidian-100">¥{paymentInfo.plan_price?.toFixed(2)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-obsidian-400">当前余额</span>
                <span className="font-medium text-obsidian-100">¥{paymentInfo.balance?.toFixed(2)}</span>
              </div>
              <div className="border-t border-obsidian-800 my-2" />
              <div className="flex justify-between text-sm">
                <span className="text-obsidian-400">需要支付</span>
                <span className="font-semibold text-amber-400">¥{paymentInfo.shortfall?.toFixed(2)}</span>
              </div>
            </div>
            <p className="text-xs text-obsidian-500 mb-4">支付完成后将自动完成订阅，无需再次操作。</p>
            {actionError && (
              <div className="flex items-center gap-2 text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2.5 mb-4">
                <AlertCircle size={14} className="flex-shrink-0" />
                <span>{actionError}</span>
              </div>
            )}
            <div className="flex gap-3">
              <button onClick={closeConfirm} className="flex-1 py-2 text-sm text-obsidian-300 bg-obsidian-800 rounded-lg hover:bg-obsidian-700 transition-colors">取消</button>
              <button
                onClick={handleGoToPay}
                disabled={createPayment.isPending}
                className="flex-1 py-2 rounded-lg text-sm font-semibold bg-amber-500 hover:bg-amber-400 text-obsidian-950 transition-colors flex items-center justify-center gap-1.5 disabled:opacity-50"
              >
                {createPayment.isPending && <Loader2 size={14} className="animate-spin" />}
                去支付
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
