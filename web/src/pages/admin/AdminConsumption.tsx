import { useState } from "react";
import { Coins, Hash, Layers, Loader } from "lucide-react";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from "recharts";
import { useAdminConsumptionStats } from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
function fmtNum(n: number): string {
  if (n >= 1_000_000_000) return (n / 1_000_000_000).toFixed(2) + "B";
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(2) + "M";
  if (n >= 1_000) return (n / 1_000).toFixed(1) + "K";
  return String(n);
}

const dayOptions = [
  { label: "7 天", value: 7 },
  { label: "30 天", value: 30 },
  { label: "90 天", value: 90 },
  { label: "365 天", value: 365 },
];

const AdminConsumption = () => {
  const [days, setDays] = useState(30);
  const { data, isLoading } = useAdminConsumptionStats(days);

  const models = data?.models ?? [];
  const dailyTrend = data?.daily_trend ?? [];

  const totalTokens = models.reduce(
    (sum, m) => sum + m.prompt_tokens + m.completion_tokens + m.cache_read_tokens + m.cache_creation_tokens,
    0,
  );

  // Bar chart data: top 10 models by cost
  const barData = models.slice(0, 10).map((m) => ({
    model: m.model.length > 20 ? m.model.slice(0, 18) + "…" : m.model,
    cost: m.total_cost,
  }));

  if (isLoading) {
    return (
      <div className="page-container fade-in flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader size={28} className="animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">加载中...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="page-container fade-in">
      {/* Header + time selector */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-lg font-bold text-foreground">消耗统计</h1>
          <p className="text-sm text-muted-foreground mt-0.5">按模型维度查看 Token 消耗和费用明细</p>
        </div>
        <div className="flex items-center gap-1 bg-muted rounded-lg p-1">
          {dayOptions.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setDays(opt.value)}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
                days === opt.value
                  ? "bg-card text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-1 gap-4 mb-6 sm:grid-cols-3">
        {[
          { label: "总消耗金额", value: `¥${(data?.total_cost ?? 0).toFixed(4)}`, icon: Coins, iconBg: "bg-emerald-500/10 dark:bg-emerald-500/15", iconColor: "text-emerald-500" },
          { label: "总请求数", value: fmtNum(data?.total_requests ?? 0), icon: Hash, iconBg: "bg-primary/10", iconColor: "text-primary" },
          { label: "总 Token 数", value: fmtNum(totalTokens), icon: Layers, iconBg: "bg-violet-500/10 dark:bg-violet-500/15", iconColor: "text-violet-500" },
        ].map((card, i) => {
          const Icon = card.icon;
          return (
            <div key={card.label} className="stagger-item flex-1 bg-card border border-border rounded-xl p-5 shadow-card hover:shadow-elevated hover:-translate-y-0.5 transition-all duration-200" style={{ animationDelay: `${i * 80}ms` }}>
              <div className="flex items-start justify-between mb-3">
                <div className={`w-9 h-9 ${card.iconBg} rounded-lg flex items-center justify-center`}>
                  <Icon size={17} className={card.iconColor} />
                </div>
                <span className="text-xs text-muted-foreground">{card.label}</span>
              </div>
              <div className="text-2xl font-bold text-foreground">{card.value}</div>
            </div>
          );
        })}
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-1 gap-4 mb-6 lg:grid-cols-5">
        {/* Daily trend */}
        <div className="lg:col-span-3 bg-card border border-border rounded-xl p-5 shadow-card">
          <div className="text-sm font-semibold text-foreground mb-1">每日消耗趋势</div>
          <div className="text-xs text-muted-foreground mb-4">近 {days} 天消费变化</div>
          <div style={{ height: 200 }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={dailyTrend} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                <defs>
                  <linearGradient id="colorCostAdmin" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--primary)" stopOpacity={0.2} />
                    <stop offset="95%" stopColor="var(--primary)" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
                <XAxis dataKey="date" tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} />
                <Tooltip
                  contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "10px", fontSize: 12 }}
                  formatter={(v: number) => [`¥${v.toFixed(4)}`, "消费"]}
                />
                <Area type="monotone" dataKey="cost" stroke="var(--primary)" strokeWidth={2} fill="url(#colorCostAdmin)" dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Model cost bar chart */}
        <div className="lg:col-span-2 bg-card border border-border rounded-xl p-5 shadow-card">
          <div className="text-sm font-semibold text-foreground mb-1">模型费用排行</div>
          <div className="text-xs text-muted-foreground mb-4">Top 10 模型消费分布</div>
          <div style={{ height: 200 }}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={barData} margin={{ top: 5, right: 5, bottom: 0, left: -20 }} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" horizontal={false} />
                <XAxis type="number" tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} />
                <YAxis type="category" dataKey="model" tick={{ fontSize: 10, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} width={120} />
                <Tooltip
                  contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "10px", fontSize: 12 }}
                  formatter={(v: number) => [`¥${v.toFixed(4)}`, "消费"]}
                />
                <Bar dataKey="cost" fill="var(--primary)" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      {/* Token detail table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="text-sm font-semibold text-foreground">模型 Token 消耗明细</div>
          <div className="text-xs text-muted-foreground mt-0.5">各模型的 Token 用量和对应费用。标 ≈ 的分项费用按当前价目估算，总费用为实际扣费金额。</div>
        </div>
        <div className="overflow-auto">
          <Table className="w-full">
            <TableHeader className="sticky top-0 z-10">
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3 font-semibold">模型</TableHead>
                <TableHead className="text-right px-4 py-3 font-semibold">输入 Token</TableHead>
                <TableHead className="text-right px-4 py-3 font-semibold" title="按当前价目估算，仅供参考">输入费用 ≈</TableHead>
                <TableHead className="text-right px-4 py-3 font-semibold">输出 Token</TableHead>
                <TableHead className="text-right px-4 py-3 font-semibold" title="按当前价目估算，仅供参考">输出费用 ≈</TableHead>
                <TableHead className="text-right px-4 py-3 font-semibold">缓存读取</TableHead>
                <TableHead className="text-right px-4 py-3 font-semibold" title="按当前价目估算，仅供参考">缓存读取费用 ≈</TableHead>
                <TableHead className="text-right px-4 py-3 font-semibold">缓存创建</TableHead>
                <TableHead className="text-right px-4 py-3 font-semibold" title="按当前价目估算，仅供参考">缓存创建费用 ≈</TableHead>
                <TableHead className="text-right px-4 py-3 font-semibold" title="实际扣费金额">总费用（实扣）</TableHead>
                <TableHead className="text-right px-5 py-3 font-semibold">请求数</TableHead>
                <TableHead className="text-right px-5 py-3 font-semibold" style={{ minWidth: 160 }}>成功率</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {models.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={12} className="px-5 py-12 text-center text-sm text-muted-foreground">
                    暂无消耗数据
                  </TableCell>
                </TableRow>
              ) : (
                models.map((m, i) => (
                  <TableRow key={m.model} className={`border-t border-border hover:bg-muted/40 transition-colors ${i % 2 === 0 ? "" : "bg-muted/10"}`}>
                    <TableCell className="px-5 py-3 text-sm font-mono text-foreground whitespace-nowrap">{m.model}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-right text-muted-foreground tabular-nums">{fmtNum(m.prompt_tokens)}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-right text-foreground tabular-nums">¥{m.prompt_cost.toFixed(4)}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-right text-muted-foreground tabular-nums">{fmtNum(m.completion_tokens)}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-right text-foreground tabular-nums">¥{m.completion_cost.toFixed(4)}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-right text-muted-foreground tabular-nums">{fmtNum(m.cache_read_tokens)}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-right text-foreground tabular-nums">¥{m.cache_read_cost.toFixed(4)}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-right text-muted-foreground tabular-nums">{fmtNum(m.cache_creation_tokens)}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-right text-foreground tabular-nums">¥{m.cache_creation_cost.toFixed(4)}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-right font-semibold text-foreground tabular-nums">¥{m.total_cost.toFixed(4)}</TableCell>
                    <TableCell className="px-5 py-3 text-sm text-right text-muted-foreground tabular-nums">{fmtNum(m.request_count)}</TableCell>
                    <TableCell className="px-5 py-3 text-right">
                      {(() => {
                        const rate = m.success_rate ?? 100;
                        const barColor = rate >= 99 ? "bg-emerald-500"
                          : rate >= 95 ? "bg-amber-400"
                          : "bg-red-500";
                        const textColor = rate >= 99 ? "text-emerald-600 dark:text-emerald-400"
                          : rate >= 95 ? "text-amber-600 dark:text-amber-400"
                          : "text-red-600 dark:text-red-400";
                        return (
                          <div className="flex items-center justify-end gap-2">
                            <div className="w-16 h-1.5 rounded-full bg-muted overflow-hidden shrink-0">
                              <div
                                className={`h-full rounded-full transition-all duration-700 ${barColor}`}
                                style={{ width: `${Math.min(100, rate)}%` }}
                              />
                            </div>
                            <span className={`text-xs font-semibold tabular-nums w-12 text-right ${textColor}`}>
                              {rate.toFixed(1)}%
                            </span>
                          </div>
                        );
                      })()}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
};

export default AdminConsumption;
