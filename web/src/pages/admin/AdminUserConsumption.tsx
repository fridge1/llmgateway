import { useState } from "react";
import {
  ChevronLeft, Loader2, TrendingUp, Cpu, Coins, BarChart2, Download, Filter,
  ChevronRight,
} from "lucide-react";
import {
  AreaChart, Area, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from "recharts";
import {
  useAdminUserTransactions,
  useAdminUserConsumptionStats,
  exportAdminUserTransactions,
} from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const COLORS = [
  "var(--primary)", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6",
  "#06b6d4", "#ec4899", "#14b8a6", "#f97316", "#6366f1",
  "#84cc16", "#a855f7", "#0ea5e9", "#e11d48", "#22c55e",
];

interface AdminUserConsumptionProps {
  userId: string;
  userPhone: string;
  onBack: () => void;
}

const AdminUserConsumption = ({ userId, userPhone, onBack }: AdminUserConsumptionProps) => {
  const [days, setDays] = useState(7);
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(20);
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const [exporting, setExporting] = useState(false);

  const { data: stats, isLoading: statsLoading } = useAdminUserConsumptionStats(userId, days);
  const { data: txData, isLoading: txLoading } = useAdminUserTransactions(
    userId, page, size, startDate || undefined, endDate || undefined, typeFilter || undefined
  );

  const transactions = txData?.transactions ?? [];
  const total = txData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  const handleExport = async () => {
    setExporting(true);
    try {
      await exportAdminUserTransactions(userId, startDate || undefined, endDate || undefined);
    } catch {
      // silently fail
    } finally {
      setExporting(false);
    }
  };

  const totalTokens = stats
    ? (stats.models || []).reduce((sum, m) => sum + m.prompt_tokens + m.completion_tokens + m.cache_read_tokens + m.cache_creation_tokens, 0)
    : 0;

  const todayCost = stats
    ? (stats.daily_trend || []).filter(d => d.date === new Date().toISOString().split('T')[0]).reduce((sum, d) => sum + d.cost, 0)
    : 0;

  const monthCost = stats
    ? (stats.daily_trend || []).filter(d => {
        const trendDate = new Date(d.date);
        const now = new Date();
        return trendDate.getMonth() === now.getMonth() && trendDate.getFullYear() === now.getFullYear();
      }).reduce((sum, d) => sum + d.cost, 0)
    : 0;

  const tokenPieData = stats && totalTokens > 0 ? [
    { name: "输入", value: (stats.models || []).reduce((sum, m) => sum + m.prompt_tokens, 0) },
    { name: "输出", value: (stats.models || []).reduce((sum, m) => sum + m.completion_tokens, 0) },
    { name: "缓存命中", value: (stats.models || []).reduce((sum, m) => sum + m.cache_read_tokens, 0) },
    { name: "缓存写入", value: (stats.models || []).reduce((sum, m) => sum + m.cache_creation_tokens, 0) },
  ].filter(d => d.value > 0) : [];

  if (statsLoading) {
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
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            className="p-1.5 hover:bg-muted rounded-lg transition-colors"
          >
            <ChevronLeft size={18} />
          </button>
          <div>
            <div className="flex items-center gap-2">
              <TrendingUp size={20} className="text-primary" />
              <h1 className="text-xl font-bold text-foreground">用户消费详情</h1>
            </div>
            <p className="text-sm text-muted-foreground mt-0.5 ml-7">{userPhone}</p>
          </div>
        </div>
        <div className="flex items-center gap-1 bg-card border border-border rounded-lg p-1 shadow-card">
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
      </div>

      {/* Summary Cards */}
      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[
          { icon: Coins, iconBg: "bg-orange-50 dark:bg-orange-500/10", iconColor: "text-orange-500", label: "今日消费", value: `¥${todayCost.toFixed(4)}` },
          { icon: Coins, iconBg: "bg-emerald-50 dark:bg-emerald-500/10", iconColor: "text-emerald-500", label: "本月消费", value: `¥${monthCost.toFixed(4)}` },
          { icon: TrendingUp, iconBg: "bg-blue-50 dark:bg-blue-500/10", iconColor: "text-blue-500", label: "总请求", value: (stats?.total_requests ?? 0).toLocaleString() },
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

      {/* Cost Trend + Token Pie */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        {/* Cost Trend */}
        <div className="bg-card border border-border rounded-xl p-5 shadow-card">
          <div className="mb-4">
            <h3 className="text-sm font-semibold text-foreground">费用趋势</h3>
            <p className="text-xs text-muted-foreground mt-0.5">近 {days} 天消费变化</p>
          </div>
          <div style={{ height: 200 }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={stats?.daily_trend ?? []} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                <defs>
                  <linearGradient id="colorUserCost" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--primary)" stopOpacity={0.15} />
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
                <Area type="monotone" dataKey="cost" stroke="var(--primary)" strokeWidth={2} fill="url(#colorUserCost)" dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Token Distribution */}
        <div className="bg-card border border-border rounded-xl p-5 shadow-card">
          <div className="mb-4">
            <h3 className="text-sm font-semibold text-foreground">Token 构成</h3>
            <p className="text-xs text-muted-foreground mt-0.5">各类 Token 分布</p>
          </div>
          <div style={{ height: 200 }}>
            {tokenPieData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie data={tokenPieData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={50} outerRadius={80} paddingAngle={2}>
                    {tokenPieData.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                  </Pie>
                  <Tooltip contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "10px", fontSize: 12 }} />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-sm text-muted-foreground">暂无数据</div>
            )}
          </div>
        </div>
      </div>

      {/* Model Breakdown Table */}
      <div className="bg-card border border-border rounded-xl p-5 shadow-card mb-6">
        <div className="mb-4">
          <h3 className="text-sm font-semibold text-foreground">模型消费统计</h3>
          <p className="text-xs text-muted-foreground mt-0.5">按模型聚合的请求次数与费用</p>
        </div>
        {(stats?.models ?? []).length === 0 ? (
          <div className="text-sm text-muted-foreground text-center py-6">暂无模型消费记录</div>
        ) : (
          <Table className="w-full">
              <TableHeader>
                <TableRow className="table-header">
                  <TableHead className="px-4 py-3 text-left">模型</TableHead>
                  <TableHead className="px-4 py-3 text-right">请求次数</TableHead>
                  <TableHead className="px-4 py-3 text-right">输入Token</TableHead>
                  <TableHead className="px-4 py-3 text-right">输出Token</TableHead>
                  <TableHead className="px-4 py-3 text-right">总费用</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(stats?.models ?? []).map((m) => (
                  <TableRow key={m.model} className="border-t border-border hover:bg-muted/30 transition-colors">
                    <TableCell className="px-4 py-3 text-sm font-mono text-foreground">{m.model}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-muted-foreground text-right">{m.request_count.toLocaleString()}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-muted-foreground text-right font-mono">{m.prompt_tokens.toLocaleString()}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-muted-foreground text-right font-mono">{m.completion_tokens.toLocaleString()}</TableCell>
                    <TableCell className="px-4 py-3 text-sm font-medium text-right">¥{m.total_cost.toFixed(4)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
                  )}
      </div>

      {/* Transaction Table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="p-5 border-b border-border">
          <h3 className="text-sm font-semibold text-foreground mb-3">消费流水</h3>
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center gap-2 bg-muted border border-border rounded-lg px-3 py-2 text-sm">
              <Filter size={14} className="text-muted-foreground" />
              <select
                className="bg-transparent text-sm text-foreground outline-none cursor-pointer"
                value={typeFilter}
                onChange={(e) => { setTypeFilter(e.target.value); setPage(1); }}
              >
                <option value="">全部类型</option>
                <option value="consumption">消费</option>
                <option value="recharge">充值</option>
                <option value="subscription_usage">订阅消费</option>
                <option value="refund">退款</option>
              </select>
            </div>
            <input
              type="date"
              value={startDate}
              onChange={(e) => { setStartDate(e.target.value); setPage(1); }}
              className="bg-muted border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none"
            />
            <span className="text-xs text-muted-foreground">至</span>
            <input
              type="date"
              value={endDate}
              onChange={(e) => { setEndDate(e.target.value); setPage(1); }}
              className="bg-muted border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none"
            />
            <button
              onClick={handleExport}
              disabled={exporting}
              className="btn-primary flex items-center gap-1.5 disabled:opacity-50"
            >
              {exporting ? <Loader2 size={14} className="animate-spin" /> : <Download size={14} />}
              导出 Excel
            </button>
            <div className="text-xs text-muted-foreground">共 {total} 条记录</div>
          </div>
        </div>

        {txLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 size={16} className="animate-spin mr-2 text-primary" />
            <span className="text-sm text-muted-foreground">加载中...</span>
          </div>
        ) : transactions.length === 0 ? (
          <div className="empty-state">
            <div className="w-14 h-14 bg-muted rounded-full flex items-center justify-center mb-4">
              <BarChart2 size={24} className="text-muted-foreground" />
            </div>
            <p className="text-sm font-medium text-muted-foreground">暂无交易记录</p>
          </div>
        ) : (
          <>
            <Table className="w-full">
                <TableHeader>
                  <TableRow className="table-header">
                    <TableHead className="px-4 py-3 text-left">时间</TableHead>
                    <TableHead className="px-4 py-3 text-left">类型</TableHead>
                    <TableHead className="px-4 py-3 text-left">模型</TableHead>
                    <TableHead className="px-4 py-3 text-right">输入</TableHead>
                    <TableHead className="px-4 py-3 text-right">输出</TableHead>
                    <TableHead className="px-4 py-3 text-right">缓存命中</TableHead>
                    <TableHead className="px-4 py-3 text-right">缓存写入</TableHead>
                    <TableHead className="px-4 py-3 text-right">金额</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {transactions.map((tx) => (
                    <TableRow key={tx.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                      <TableCell className="px-4 py-3.5 text-sm text-muted-foreground">
                        {new Date(tx.created_at).toLocaleString("zh-CN")}
                      </TableCell>
                      <TableCell className="px-4 py-3.5 text-sm text-foreground">{tx.type}</TableCell>
                      <TableCell className="px-4 py-3.5 text-sm font-mono text-foreground">{tx.model || "—"}</TableCell>
                      <TableCell className="px-4 py-3.5 text-sm text-muted-foreground text-right font-mono">
                        {tx.prompt_tokens != null ? tx.prompt_tokens.toLocaleString() : "—"}
                      </TableCell>
                      <TableCell className="px-4 py-3.5 text-sm text-muted-foreground text-right font-mono">
                        {tx.completion_tokens != null ? tx.completion_tokens.toLocaleString() : "—"}
                      </TableCell>
                      <TableCell className="px-4 py-3.5 text-sm text-muted-foreground text-right font-mono">
                        {tx.cache_read_tokens != null ? tx.cache_read_tokens.toLocaleString() : "—"}
                      </TableCell>
                      <TableCell className="px-4 py-3.5 text-sm text-muted-foreground text-right font-mono">
                        {tx.cache_creation_tokens != null ? tx.cache_creation_tokens.toLocaleString() : "—"}
                      </TableCell>
                      <TableCell className={`px-4 py-3.5 text-sm font-medium text-right ${tx.amount < 0 ? "text-destructive" : "text-emerald-500"}`}>
                        {tx.amount < 0 ? "-" : "+"}¥{Math.abs(tx.amount).toFixed(4)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
                        {/* Pagination */}
            <div className="px-5 py-3.5 border-t border-border flex items-center justify-between">
              <div className="text-xs text-muted-foreground">共 {total} 条记录</div>
              <div className="flex items-center gap-2">
                <button
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                  className="w-7 h-7 rounded-lg flex items-center justify-center text-sm transition-colors hover:bg-muted/60 disabled:text-muted-foreground disabled:cursor-not-allowed"
                >
                  <ChevronLeft size={14} />
                </button>
                <span className="text-sm text-foreground px-2">
                  <span className="font-medium">{page}</span>
                  <span className="text-muted-foreground"> / {totalPages}</span>
                </span>
                <button
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                  className="w-7 h-7 rounded-lg flex items-center justify-center text-sm transition-colors hover:bg-muted/60 disabled:text-muted-foreground disabled:cursor-not-allowed"
                >
                  <ChevronRight size={14} />
                </button>
                <select
                  className="ml-2 bg-muted border border-border rounded-lg px-2 py-1 text-xs text-foreground outline-none cursor-pointer"
                  value={size}
                  onChange={(e) => { setSize(Number(e.target.value)); setPage(1); }}
                >
                  <option value={10}>10条/页</option>
                  <option value={20}>20条/页</option>
                  <option value={50}>50条/页</option>
                </select>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default AdminUserConsumption;
