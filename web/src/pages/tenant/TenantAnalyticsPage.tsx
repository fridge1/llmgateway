import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Loader2, TrendingUp, Cpu, Users, Coins,
} from "lucide-react";
import {
  AreaChart, Area, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from "recharts";
import { useTenantDetail, useTenantStats } from "@/hooks/use-tenant";
import { TenantPageHeader } from "@/components/tenant/TenantPageHeader";

const COLORS = [
  "var(--primary)", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6",
  "#06b6d4", "#ec4899", "#14b8a6", "#f97316", "#6366f1",
  "#84cc16", "#a855f7", "#0ea5e9", "#e11d48", "#22c55e",
];

const TenantAnalyticsPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [days, setDays] = useState(7);

  const { data: tenant } = useTenantDetail(id!);
  const { data: stats, isLoading } = useTenantStats(id!, days);

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

  const totalTokens = stats
    ? stats.token_stats.total_prompt + stats.token_stats.total_completion +
      stats.token_stats.total_cache_read + stats.token_stats.total_cache_creation
    : 0;

  const tokenPieData = stats ? [
    { name: "输入", value: stats.token_stats.total_prompt },
    { name: "输出", value: stats.token_stats.total_completion },
    { name: "缓存命中", value: stats.token_stats.total_cache_read },
    { name: "缓存写入", value: stats.token_stats.total_cache_creation },
  ].filter(d => d.value > 0) : [];

  return (
    <div className="page-container fade-in">
      <TenantPageHeader
        title="数据分析"
        description="查看费用趋势、模型分布与 Token 构成"
        tenantName={tenant?.name}
        icon={TrendingUp}
        onBack={() => navigate(`/dashboard/tenants/${id}`)}
        actions={
          <div className="flex items-center gap-1 bg-background border border-border rounded-lg p-1">
          {[7, 14, 30].map((d) => (
            <button
              key={d}
              onClick={() => setDays(d)}
              className={`px-3 py-1.5 text-xs rounded-md transition-colors ${
                days === d
                  ? "bg-primary text-primary-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted"
              }`}
            >
              {d}天
            </button>
          ))}
          </div>
        }
      />

      {/* Summary Cards */}
      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {[
          { icon: Coins, iconBg: "bg-orange-50 dark:bg-orange-500/10", iconColor: "text-orange-500", label: "今日消费", value: `¥${(stats?.today_cost ?? 0).toFixed(4)}` },
          { icon: Coins, iconBg: "bg-emerald-50 dark:bg-emerald-500/10", iconColor: "text-emerald-500", label: "本月消费", value: `¥${(stats?.month_cost ?? 0).toFixed(4)}` },
          { icon: TrendingUp, iconBg: "bg-blue-50 dark:bg-blue-500/10", iconColor: "text-blue-500", label: "日均消费", value: `¥${(stats?.daily_average ?? 0).toFixed(4)}` },
          { icon: Cpu, iconBg: "bg-violet-50 dark:bg-violet-500/10", iconColor: "text-violet-500", label: "总Token", value: totalTokens.toLocaleString() },
        ].map((card, i) => {
          const Icon = card.icon;
          return (
            <div key={card.label} className="bg-card border border-border rounded-xl p-4 shadow-card stagger-item" style={{ animationDelay: `${i * 80}ms` }}>
              <div className="flex items-center gap-2 mb-2">
                <div className={`w-7 h-7 rounded-lg flex items-center justify-center ${card.iconBg}`}>
                  <Icon size={14} className={card.iconColor} />
                </div>
                <span className="text-xs text-muted-foreground">{card.label}</span>
              </div>
              <p className="text-xl font-bold text-foreground">{card.value}</p>
            </div>
          );
        })}
      </div>

      {/* Cost Trend */}
      <div className="bg-card border border-border rounded-xl p-5 shadow-card mb-6">
        <div className="mb-4">
          <h3 className="text-sm font-semibold text-foreground">费用趋势</h3>
          <p className="text-xs text-muted-foreground mt-0.5">近 {days} 天消费变化</p>
        </div>
        <div style={{ height: 220 }}>
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={stats?.daily_trend ?? []} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
              <defs>
                <linearGradient id="colorTenantCost" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--primary)" stopOpacity={0.15} />
                  <stop offset="95%" stopColor="var(--primary)" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
              <XAxis dataKey="date" tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} />
              <Tooltip
                contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "10px", fontSize: 12, boxShadow: "0 4px 6px -1px rgba(0,0,0,0.05)" }}
                formatter={(v: number) => [`¥${v.toFixed(4)}`, "消费"]}
              />
              <Area type="monotone" dataKey="cost" stroke="var(--primary)" strokeWidth={2} fill="url(#colorTenantCost)" dot={false} />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Model Distribution + Token Breakdown */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        {/* Model Distribution */}
        <div className="bg-card border border-border rounded-xl p-5 shadow-card">
          <div className="mb-4">
            <h3 className="text-sm font-semibold text-foreground">模型消费分布</h3>
            <p className="text-xs text-muted-foreground mt-0.5">各模型消费占比</p>
          </div>
          <div style={{ height: 240 }}>
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={stats?.model_breakdown ?? []}
                  dataKey="cost"
                  nameKey="model"
                  cx="50%"
                  cy="50%"
                  outerRadius={80}
                  label={({ model, percent }) => `${model.length > 12 ? model.slice(0, 12) + "..." : model} ${(percent * 100).toFixed(0)}%`}
                  labelLine={false}
                >
                  {(stats?.model_breakdown ?? []).map((_, i) => (
                    <Cell key={i} fill={COLORS[i % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "10px", fontSize: 12, boxShadow: "0 4px 6px -1px rgba(0,0,0,0.05)" }}
                  formatter={(v: number) => [`¥${v.toFixed(4)}`, "消费"]}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Token Breakdown */}
        <div className="bg-card border border-border rounded-xl p-5 shadow-card">
          <div className="mb-4">
            <h3 className="text-sm font-semibold text-foreground">Token 构成</h3>
            <p className="text-xs text-muted-foreground mt-0.5">各类型 Token 占比</p>
          </div>
          <div style={{ height: 240 }}>
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={tokenPieData}
                  dataKey="value"
                  nameKey="name"
                  cx="50%"
                  cy="50%"
                  outerRadius={80}
                  label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                  labelLine={false}
                >
                  {tokenPieData.map((_, i) => (
                    <Cell key={i} fill={COLORS[i % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "10px", fontSize: 12, boxShadow: "0 4px 6px -1px rgba(0,0,0,0.05)" }}
                  formatter={(v: number) => [v.toLocaleString(), "Token"]}
                />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      {/* Sub-user Ranking */}
      <div className="bg-card border border-border rounded-xl p-5 shadow-card">
        <div className="mb-4">
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-lg bg-blue-50 dark:bg-blue-500/10 flex items-center justify-center">
              <Users size={14} className="text-blue-500" />
            </div>
            <h3 className="text-sm font-semibold text-foreground">子用户消费排行</h3>
          </div>
          <p className="text-xs text-muted-foreground mt-0.5 ml-9">TOP 10 子用户消费对比</p>
        </div>
        {(stats?.sub_user_ranking ?? []).length === 0 ? (
          <div className="text-center py-8 text-muted-foreground text-sm">暂无数据</div>
        ) : (
          <div style={{ height: 280 }}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={stats?.sub_user_ranking ?? []}
                layout="vertical"
                margin={{ top: 5, right: 30, bottom: 5, left: 80 }}
              >
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" horizontal={false} />
                <XAxis type="number" tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} />
                <YAxis
                  type="category"
                  dataKey="sub_user_username"
                  tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                  axisLine={false}
                  tickLine={false}
                  width={75}
                />
                <Tooltip
                  contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "10px", fontSize: 12, boxShadow: "0 4px 6px -1px rgba(0,0,0,0.05)" }}
                  formatter={(v: number) => [`¥${v.toFixed(4)}`, "消费"]}
                />
                <Bar dataKey="total_cost" fill="var(--primary)" radius={[0, 4, 4, 0]} fillOpacity={0.8} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>
    </div>
  );
};

export default TenantAnalyticsPage;
