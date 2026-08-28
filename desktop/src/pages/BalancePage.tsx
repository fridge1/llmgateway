import { useState } from "react";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { useBalance, useBillingStats, useCreatePayment, usePromotionRules, useOrders, useRepayOrder, useTransactions } from "@/hooks/use-api";
import { Loader2, AlertCircle } from "../components/icons";

const presets = [
  { value: 10, label: "¥10", sub: "基础" },
  { value: 50, label: "¥50", sub: "标准" },
  { value: 100, label: "¥100", sub: "热门", hot: true },
  { value: 200, label: "¥200", sub: "进阶" },
  { value: 500, label: "¥500", sub: "专业" },
  { value: -1, label: "···", sub: "自定义" },
];

function formatTime(iso: string) {
  return new Date(iso).toLocaleString("zh-CN");
}

export default function BalancePage() {
  const [selectedAmount, setSelectedAmount] = useState(100);
  const [customAmount, setCustomAmount] = useState("");
  const [payMessage, setPayMessage] = useState<string | null>(null);
  const [payError, setPayError] = useState<string | null>(null);

  const balance = useBalance();
  const stats = useBillingStats(7);
  const { data: ordersData } = useOrders(1, 10);
  const txns = useTransactions(1, 50, "recharge");
  const createPayment = useCreatePayment();
  const repayOrder = useRepayOrder();
  const { data: promotionData } = usePromotionRules();
  const promotionRules = promotionData?.rules ?? [];

  const isLoading = balance.isLoading || stats.isLoading;
  const finalAmount = selectedAmount === -1 ? (parseFloat(customAmount) || 0) : selectedAmount;
  const balanceVal = balance.data?.balance ?? 0;
  const frozenVal = balance.data?.frozen ?? 0;
  const todayCost = stats.data?.today_cost ?? 0;
  const monthCost = stats.data?.month_cost ?? 0;
  const chartData = stats.data?.daily_trend ?? [];

  const dailyTrend = stats.data?.daily_trend ?? [];
  const avgDailyCost = dailyTrend.length > 0
    ? dailyTrend.reduce((sum, d) => sum + d.cost, 0) / dailyTrend.length
    : 0;
  const estimatedDays = avgDailyCost > 0 ? Math.floor(balanceVal / avgDailyCost) : "—";

  const pendingOrders = (ordersData?.orders ?? []).filter(
    (o) => o.status === "pending" && new Date(o.expired_at) > new Date()
  );

  const rechargeRecords = txns.data?.transactions ?? [];

  const handlePay = async () => {
    if (finalAmount <= 0) return;
    setPayError(null);
    try {
      const res = await createPayment.mutateAsync({ amount: finalAmount, client_type: "desktop" });
      const { openUrl } = await import("@tauri-apps/plugin-opener");
      await openUrl(res.pay_url);
      setPayMessage(`订单 ${res.order_no} 已创建，请在浏览器中完成支付。支付完成后点击"刷新余额"。`);
    } catch (err) {
      setPayError(String(err));
    }
  };

  const handleRepay = async (orderNo: string) => {
    setPayError(null);
    try {
      const res = await repayOrder.mutateAsync({ order_no: orderNo, client_type: "desktop" });
      const { openUrl } = await import("@tauri-apps/plugin-opener");
      await openUrl(res.pay_url);
    } catch (err) {
      setPayError(String(err));
    }
  };

  const handleRefreshBalance = () => {
    balance.refetch();
    setPayMessage(null);
  };

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}>
        <Loader2 size={24} className="animate-spin text-amber-400" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-5">
        <h1 className="text-lg font-semibold text-obsidian-50">余额充值</h1>
        <p className="text-xs text-obsidian-400 mt-0.5">管理您的账户余额</p>
      </div>

      {/* Balance overview */}
      <div className="bg-gradient-to-br from-amber-500/10 to-amber-600/5 border border-amber-500/20 rounded-xl p-5 mb-5">
        <div className="text-xs text-obsidian-400 mb-1">可用余额</div>
        <div className="text-3xl font-bold text-obsidian-50 mb-3">¥{balanceVal.toFixed(4)}</div>
        <div className="grid grid-cols-4 gap-3">
          {[
            { label: "冻结金额", value: `¥${frozenVal.toFixed(4)}` },
            { label: "今日消费", value: `¥${todayCost.toFixed(4)}` },
            { label: "本月消费", value: `¥${monthCost.toFixed(4)}` },
            { label: "预计可用", value: typeof estimatedDays === "number" ? `${estimatedDays} 天` : "— 天" },
          ].map((item) => (
            <div key={item.label} className="bg-obsidian-900/60 rounded-lg p-2.5">
              <div className="text-[10px] text-obsidian-400 mb-0.5">{item.label}</div>
              <div className="text-sm font-semibold text-obsidian-200">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Promotion rules */}
      {promotionRules.length > 0 && (
        <div className="bg-emerald-500/5 border border-emerald-500/20 rounded-xl p-4 mb-5">
          <div className="text-sm font-semibold text-emerald-400 mb-2">充值优惠</div>
          <div className="space-y-1.5">
            {promotionRules.map((rule, i) => (
              <div key={i} className="text-xs text-obsidian-300">
                {rule.description ?? `充值 ¥${rule.threshold ?? 0} 起，赠送 ${rule.bonus_percent ? `${rule.bonus_percent}%` : `¥${rule.bonus_amount ?? 0}`}`}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Recharge */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-5 mb-5">
        <div className="text-sm font-semibold text-obsidian-50 mb-3">选择充值金额</div>
        <div className="grid grid-cols-6 gap-2 mb-4">
          {presets.map((p) => (
            <button
              key={p.value}
              onClick={() => setSelectedAmount(p.value)}
              className={`py-3 rounded-lg text-center transition-all duration-200 border relative ${
                selectedAmount === p.value
                  ? "bg-amber-500 border-amber-400 text-obsidian-950"
                  : "bg-obsidian-800 border-obsidian-700 text-obsidian-200 hover:border-obsidian-600"
              }`}
            >
              <div className="text-sm font-semibold">{p.label}</div>
              <div className={`text-[10px] mt-0.5 ${selectedAmount === p.value ? "text-obsidian-950/60" : "text-obsidian-500"}`}>
                {p.sub}
              </div>
              {p.hot && (
                <span className="absolute -top-1.5 -right-1.5 text-[9px] px-1.5 py-0.5 rounded-full bg-red-500 text-white">推荐</span>
              )}
            </button>
          ))}
        </div>

        {selectedAmount === -1 && (
          <input
            type="number"
            className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50 mb-4"
            placeholder="输入自定义金额"
            value={customAmount}
            onChange={(e) => setCustomAmount(e.target.value)}
          />
        )}

        <div className="flex items-center gap-3">
          <button
            onClick={handlePay}
            disabled={finalAmount <= 0 || createPayment.isPending}
            className="px-6 py-2.5 bg-amber-500 hover:bg-amber-400 hover:shadow-glow-amber disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200"
          >
            {createPayment.isPending ? "处理中..." : `支付 ¥${finalAmount > 0 ? finalAmount : 0}`}
          </button>
          <span className="text-xs text-obsidian-500">支持支付宝</span>
        </div>
      </div>

      {/* Payment message */}
      {payMessage && (
        <div className="bg-amber-500/10 border border-amber-500/20 rounded-xl p-4 mb-5 flex items-center justify-between">
          <span className="text-sm text-obsidian-200">{payMessage}</span>
          <button onClick={handleRefreshBalance} className="px-3 py-1.5 bg-amber-500 text-obsidian-950 text-xs font-semibold rounded-lg hover:bg-amber-400 transition-colors">
            刷新余额
          </button>
        </div>
      )}

      {/* Payment error */}
      {payError && (
        <div className="bg-red-500/10 border border-red-500/20 rounded-xl p-4 mb-5 flex items-center gap-2">
          <AlertCircle size={16} className="text-red-400 flex-shrink-0" />
          <span className="text-sm text-red-300 flex-1">{payError}</span>
          <button onClick={() => setPayError(null)} className="text-xs text-obsidian-400 hover:text-obsidian-200">关闭</button>
        </div>
      )}

      {/* 7-day spending trend + recharge history */}
      <div className="grid grid-cols-2 gap-4">
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-sm font-semibold text-obsidian-50 mb-3">近 7 天消费趋势</div>
          <div style={{ height: 150 }}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={chartData} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a38" vertical={false} />
                <XAxis dataKey="date" tick={{ fontSize: 10, fill: "#8888a0" }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 10, fill: "#8888a0" }} axisLine={false} tickLine={false} />
                <Tooltip
                  contentStyle={{ background: "#111118", border: "1px solid #2a2a38", borderRadius: "8px", fontSize: 12 }}
                  formatter={(v) => [`¥${Number(v).toFixed(4)}`, "消费"]}
                />
                <Bar dataKey="cost" fill="#f59e0b" radius={[4, 4, 0, 0]} fillOpacity={0.8} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
          <div className="px-4 py-3 border-b border-obsidian-700 text-sm font-semibold text-obsidian-50">最近充值记录</div>
          {pendingOrders.length === 0 && rechargeRecords.length === 0 ? (
            <div className="py-8 text-center text-xs text-obsidian-500">暂无充值记录</div>
          ) : (
            <div className="divide-y divide-obsidian-800 max-h-[200px] overflow-y-auto">
              {pendingOrders.map((order) => (
                <div key={order.id} className="px-4 py-2.5 flex items-center justify-between">
                  <div>
                    <div className="text-sm text-obsidian-200 flex items-center gap-1.5">
                      ¥{order.amount.toFixed(2)}
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-amber-500/10 text-amber-400">待支付</span>
                    </div>
                    <div className="text-xs text-obsidian-500">{order.order_no}</div>
                  </div>
                  <button
                    onClick={() => handleRepay(order.order_no)}
                    className="px-2.5 py-1 bg-amber-500/10 text-amber-400 text-xs font-medium rounded-lg hover:bg-amber-500/20 transition-colors"
                  >
                    继续支付
                  </button>
                </div>
              ))}
              {rechargeRecords.map((t) => (
                <div key={t.id} className="px-4 py-2.5 flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium text-emerald-400">+¥{t.amount.toFixed(2)}</div>
                    <div className="text-xs text-obsidian-500">{t.description || "充值"} · {formatTime(t.created_at)}</div>
                  </div>
                  <div className="text-xs text-obsidian-400">余额 ¥{t.balance_after.toFixed(2)}</div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
