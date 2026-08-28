import { useState } from "react";
import { Wallet, Snowflake, Flame, CalendarDays, TrendingUp, Gift, Loader2, Info, Clock } from "lucide-react";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { useBalance, useBillingStats, useTransactions, useCreatePayment, usePromotionRules, useOrders, useRepayOrder } from "@/hooks/use-api";
import { useAuth } from "@/contexts/AuthContext";
import TenantDiscountBanner from "@/components/TenantDiscountBanner";
import { PageHeader } from "@/components/ui/page-header";

const presets = [
  { value: 10, label: "\u00A510", sub: "基础" },
  { value: 50, label: "\u00A550", sub: "标准" },
  { value: 100, label: "\u00A5100", sub: "热门", hot: true },
  { value: 200, label: "\u00A5200", sub: "进阶" },
  { value: 500, label: "\u00A5500", sub: "专业" },
  { value: -1, label: "\u00B7\u00B7\u00B7", sub: "自定义" },
];

const BalancePage = () => {
  const [selectedAmount, setSelectedAmount] = useState(100);
  const [customAmount, setCustomAmount] = useState("");
  const [payMethod] = useState("alipay");
  const [payMessage, setPayMessage] = useState<string | null>(null);

  const balance = useBalance();
  const [trendDays, setTrendDays] = useState(7);
  const stats = useBillingStats(trendDays);
  const rechargeTxns = useTransactions(1, 8, "recharge");
  const { data: ordersData } = useOrders(1, 10);
  const createPayment = useCreatePayment();
  const repayOrder = useRepayOrder();
  const { user } = useAuth();
  const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
    navigator.userAgent
  );
  const { data: promotionData } = usePromotionRules();
  const promotionRules = promotionData?.rules ?? [];
  const rechargeBonusRule = promotionRules.find((r) => r.type === "recharge_bonus");
  const otherPromotions = promotionRules.filter((r) => r.type !== "recharge_bonus");

  const formatPromoDate = (iso?: string) => {
    if (!iso) return "";
    const d = new Date(iso);
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${d.getMonth() + 1}月${d.getDate()}日 ${pad(d.getHours())}:${pad(d.getMinutes())}`;
  };

  const bonusAmount = user?.first_recharge_bonus_cny ?? 0;
  const bonusEligible = bonusAmount > 0 && !user?.first_recharge_bonus_granted;

  const isLoading = balance.isLoading || stats.isLoading || rechargeTxns.isLoading;

  const finalAmount = selectedAmount === -1
    ? (parseFloat(customAmount) || 0)
    : selectedAmount;

  const balanceVal = balance.data?.balance ?? 0;
  const frozenVal = balance.data?.frozen ?? 0;
  const todayCost = stats.data?.today_cost ?? 0;
  const monthCost = stats.data?.month_cost ?? 0;
  const chartData = stats.data?.daily_trend ?? [];

  const dailyTrend = stats.data?.daily_trend ?? [];
  const avgDailyCost = dailyTrend.length > 0
    ? dailyTrend.reduce((sum, d) => sum + d.cost, 0) / dailyTrend.length
    : 0;
  const estimatedDays = avgDailyCost > 0
    ? Math.floor(balanceVal / avgDailyCost)
    : "\u2014";

  const rechargeRecords = rechargeTxns.data?.transactions ?? [];

  const pendingOrders = (ordersData?.orders ?? []).filter(
    (o) => o.status === "pending" && new Date(o.expired_at) > new Date()
  );

  const handlePay = async () => {
    if (finalAmount <= 0) return;
    try {
      const res = await createPayment.mutateAsync({
        amount: finalAmount,
        client_type: isMobile ? "mobile" : undefined,
      });
      if (isMobile) {
        window.location.href = res.pay_url;
      } else {
        window.open(res.pay_url, "_blank");
        setPayMessage(`订单已创建：${res.order_no}，请在新窗口中完成支付。`);
        setTimeout(() => setPayMessage(null), 8000);
      }
    } catch {
      // error handled by React Query
    }
  };

  const handleRepay = async (orderNo: string) => {
    try {
      const res = await repayOrder.mutateAsync({
        order_no: orderNo,
        client_type: isMobile ? "mobile" : undefined,
      });
      if (isMobile) {
        // eslint-disable-next-line react-hooks/immutability
        window.location.href = res.pay_url;
      } else {
        window.open(res.pay_url, "_blank");
      }
    } catch {
      // error handled by React Query
    }
  };

  const formatTime = (iso: string) => new Date(iso).toLocaleString("zh-CN");

  const isBonus = (desc?: string) =>
    !!desc && !desc.startsWith("Alipay recharge");

  if (isLoading) {
    return (
      <div className="page-container fade-in flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={28} className="animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">加载中...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="page-container fade-in">
      <PageHeader eyebrow="资金" title="余额" description="查看余额、消费与充值进度，快速完成账户补给。" />

      <TenantDiscountBanner />

      <div className="grid grid-cols-1 gap-5 mb-5 lg:grid-cols-2">
        {/* Balance card */}
        <div className="rounded-2xl p-6 relative overflow-hidden stagger-item balance-card-gradient text-white" style={{ minHeight: 220 }}>
          {/* Background decoration */}
          <div className="absolute top-0 right-0 w-48 h-48 rounded-full pointer-events-none"
            style={{ background: "radial-gradient(circle, rgba(129,140,248,0.25) 0%, transparent 70%)", transform: "translate(30%, -30%)" }} />
          <div className="absolute bottom-0 left-0 w-32 h-32 rounded-full pointer-events-none"
            style={{ background: "radial-gradient(circle, rgba(167,139,250,0.2) 0%, transparent 70%)", transform: "translate(-30%, 30%)" }} />

          <div className="relative z-10">
            <div className="flex items-center gap-2 mb-4">
              <Wallet size={16} className="text-indigo-300/70" />
              <span className="text-sm font-medium text-indigo-300/70">可用余额</span>
            </div>
            <div className="text-4xl font-bold text-white mb-5">{"\u00A5"}{balanceVal.toFixed(4)}</div>

            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {[
                { icon: Snowflake, label: "冻结金额", value: `\u00A5${frozenVal.toFixed(4)}`, color: "text-cyan-300" },
                { icon: Flame, label: "今日消费", value: `\u00A5${todayCost.toFixed(4)}`, color: "text-orange-300" },
                { icon: CalendarDays, label: "本月消费", value: `\u00A5${monthCost.toFixed(4)}`, color: "text-emerald-300" },
                { icon: TrendingUp, label: "预计可用天数", value: typeof estimatedDays === "number" ? `${estimatedDays} 天` : "\u2014 天", color: "text-violet-300" },
              ].map((item) => {
                const Icon = item.icon;
                return (
                  <div key={item.label} className="rounded-xl p-3" style={{ background: "rgba(255,255,255,0.06)" }}>
                    <Icon size={14} className={`${item.color} mb-2`} />
                    <div className="text-indigo-200/60 text-xs mb-1">{item.label}</div>
                    <div className="text-white font-semibold text-sm">{item.value}</div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {/* Recharge panel */}
        <div className="bg-card border border-border rounded-2xl p-5 shadow-card stagger-item" style={{ animationDelay: "100ms" }}>
          <div className="text-sm font-semibold text-foreground mb-4">快速充值</div>

          <div className="flex flex-wrap gap-2.5 mb-4">
            {presets.map((p) => (
              <button
                key={p.value}
                onClick={() => setSelectedAmount(p.value)}
                className={`relative flex-1 min-w-[calc(33.33%-7px)] rounded-xl border py-3 text-center transition-all duration-200 ${
                  selectedAmount === p.value
                    ? "border-primary bg-primary/5 dark:bg-primary/10 ring-1 ring-primary/30"
                    : "border-border hover:border-primary/40 bg-card"
                }`}
              >
                {p.hot && (
                  <span className="absolute -top-1.5 -right-1.5 text-white text-[9px] font-bold px-1.5 py-0.5 rounded-full brand-gradient shadow-button">
                    热门
                  </span>
                )}
                <div className={`text-sm font-bold ${selectedAmount === p.value ? "text-primary" : "text-foreground"}`}>
                  {p.label}
                </div>
                <div className="text-xs text-muted-foreground mt-0.5">{p.sub}</div>
              </button>
            ))}
          </div>

          {selectedAmount === -1 && (
            <div className="mb-4">
              <input
                className="input-field"
                placeholder="请输入充值金额（元）"
                value={customAmount}
                onChange={(e) => setCustomAmount(e.target.value)}
                type="number"
                min="1"
              />
            </div>
          )}

          {/* Payment methods */}
          <div className="flex items-center justify-between mb-4 text-sm">
            <span className="text-muted-foreground">支付金额</span>
            <div className="flex items-center gap-3">
              <span className="text-base font-bold text-foreground">
                {"\u00A5"}{finalAmount > 0 ? finalAmount.toFixed(2) : "0.00"}
              </span>
              <div className="flex items-center gap-1.5 px-2.5 py-1 bg-blue-50 dark:bg-blue-500/10 border border-blue-200 dark:border-blue-500/20 rounded-lg">
                <div className="w-4 h-4 bg-blue-600 rounded-sm flex items-center justify-center">
                  <span className="text-white text-[8px] font-bold">A</span>
                </div>
                <span className="text-xs font-medium text-blue-700 dark:text-blue-400">支付宝</span>
              </div>
            </div>
          </div>

          {bonusEligible && (
            <div className="mb-3 flex items-center gap-1.5 text-xs text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-500/10 border border-amber-200 dark:border-amber-500/20 rounded-lg px-3 py-2">
              <Gift size={14} className="text-amber-500 shrink-0" />
              <span>首次充值额外赠送 <strong>{(bonusAmount * 100).toFixed(0)}%</strong>（充多少送多少）</span>
            </div>
          )}

          {payMessage && (
            <div className="mb-3 text-xs text-success bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 rounded-lg px-3 py-2">
              {payMessage}
            </div>
          )}

          <button
            disabled={finalAmount <= 0 || createPayment.isPending}
            onClick={handlePay}
            className={`w-full py-3 rounded-xl text-sm font-semibold transition-all duration-200 ${
              finalAmount > 0 && !createPayment.isPending
                ? "btn-primary"
                : "bg-muted text-muted-foreground cursor-not-allowed"
            }`}
          >
            {createPayment.isPending ? "创建订单中..." : `确认支付 ${finalAmount > 0 ? `\u00A5${finalAmount.toFixed(2)}` : ""}`}
          </button>
        </div>
      </div>

      {/* Recharge promotion banner */}
      {rechargeBonusRule && (
        <div className="mb-5 rounded-2xl px-5 py-4 flex items-center gap-3 border border-amber-200/60 dark:border-amber-500/30 banner-amber">
          <div className="w-10 h-10 rounded-xl bg-amber-500/15 dark:bg-amber-400/20 flex items-center justify-center shrink-0">
            <Gift size={20} className="text-amber-600 dark:text-amber-400" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="text-sm font-semibold text-foreground">
              限时活动 · {rechargeBonusRule.title}
            </div>
            <div className="text-xs text-muted-foreground mt-0.5">
              {rechargeBonusRule.description}
              {rechargeBonusRule.starts_at && rechargeBonusRule.ends_at && (
                <span className="ml-2 text-amber-700/80 dark:text-amber-400/90">
                  {formatPromoDate(rechargeBonusRule.starts_at)} - {formatPromoDate(rechargeBonusRule.ends_at)}
                </span>
              )}
            </div>
          </div>
          {typeof rechargeBonusRule.bonus_ratio === "number" && (
            <div className="text-right">
              <div className="text-2xl font-bold text-amber-600 dark:text-amber-400">
                +{(rechargeBonusRule.bonus_ratio * 100).toFixed(0)}%
              </div>
              <div className="text-[10px] text-muted-foreground">充值赠送</div>
            </div>
          )}
        </div>
      )}

      {/* Promotion rules */}
      {otherPromotions.length > 0 && (
        <div className="mb-5 bg-primary/3 dark:bg-primary/5 border border-primary/10 rounded-xl px-5 py-3.5 flex items-start gap-3">
          <Info size={14} className="text-primary mt-0.5 shrink-0" />
          <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-muted-foreground">
            {otherPromotions.map((rule) => (
              <span key={rule.type}>{rule.description}</span>
            ))}
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 gap-5 mb-5 lg:grid-cols-2">
        {/* Spending trend */}
        <div className="bg-card border border-border rounded-xl p-5 shadow-card flex flex-col">
          <div className="flex items-center justify-between mb-4">
            <div className="text-sm font-semibold text-foreground">近 {trendDays} 天消费趋势</div>
            <div className="flex items-center gap-1 bg-background border border-border rounded-lg p-1">
              {[7, 14, 30].map((d) => (
                <button
                  key={d}
                  onClick={() => setTrendDays(d)}
                  className={`px-2.5 py-1 text-xs rounded-md transition-colors ${
                    trendDays === d
                      ? "bg-primary text-primary-foreground font-medium"
                      : "text-muted-foreground hover:text-foreground hover:bg-muted"
                  }`}
                >
                  {d}天
                </button>
              ))}
            </div>
          </div>
          <div className="flex-1" style={{ minHeight: 160 }}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={chartData} margin={{ top: 5, right: 5, bottom: 4, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
                <XAxis dataKey="date" tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} tickLine={false} tickFormatter={(d: string) => { const parts = d.split("-"); return parts.length === 3 ? `${parts[1]}-${parts[2]}` : d; }} interval={trendDays <= 7 ? 0 : trendDays <= 14 ? 1 : 2} />
                <YAxis tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} />
                <Tooltip
                  contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "10px", fontSize: 12, boxShadow: "0 4px 6px -1px rgba(0,0,0,0.05)" }}
                  formatter={(v: number) => [`\u00A5${v.toFixed(4)}`, "消费"]}
                />
                <Bar dataKey="cost" fill="var(--primary)" radius={[4, 4, 0, 0]} fillOpacity={0.8} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Recent recharge */}
        <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
          <div className="px-5 py-4 border-b border-border">
            <div className="text-sm font-semibold text-foreground">最近充值记录</div>
          </div>
          {rechargeTxns.isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 size={20} className="animate-spin text-primary" />
            </div>
          ) : pendingOrders.length === 0 && rechargeRecords.length === 0 ? (
            <div className="empty-state py-12">
              <div className="text-sm text-muted-foreground">暂无记录</div>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {pendingOrders.map((order) => (
                <div key={order.id} className="px-5 py-3.5 flex items-center justify-between hover:bg-muted/30 transition-colors">
                  <div>
                    <div className="text-sm font-medium text-foreground flex items-center gap-1.5">
                      {"¥"}{order.amount.toFixed(2)}
                      <span className="inline-flex items-center gap-0.5 text-[10px] font-semibold text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-500/10 border border-amber-200 dark:border-amber-500/20 rounded-full px-1.5 py-0.5">
                        <Clock size={10} />待支付
                      </span>
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5">订单 {order.order_no} · {formatTime(order.created_at)}</div>
                  </div>
                  <button
                    disabled={repayOrder.isPending}
                    onClick={() => handleRepay(order.order_no)}
                    className="text-xs font-medium text-primary hover:text-primary/80 transition-colors disabled:opacity-50 px-3 py-1.5 rounded-lg hover:bg-primary/5"
                  >
                    {repayOrder.isPending ? "处理中..." : "继续支付"}
                  </button>
                </div>
              ))}
              {rechargeRecords.map((t) => (
                <div key={t.id} className="px-5 py-3.5 flex items-center justify-between hover:bg-muted/30 transition-colors">
                  <div>
                    <div className="text-sm font-medium text-foreground flex items-center gap-1.5">
                      +{"\u00A5"}{t.amount.toFixed(2)}
                      {isBonus(t.description) && (
                        <span className="inline-flex items-center gap-0.5 text-[10px] font-semibold text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-500/10 border border-amber-200 dark:border-amber-500/20 rounded-full px-1.5 py-0.5">
                          <Gift size={10} />赠送
                        </span>
                      )}
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5">{t.description || "充值"} · {formatTime(t.created_at)}</div>
                  </div>
                  <div className="text-xs text-muted-foreground">余额 {"\u00A5"}{t.balance_after.toFixed(2)}</div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default BalancePage;
