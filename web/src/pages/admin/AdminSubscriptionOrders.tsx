import { useState } from "react";
import { DollarSign, Hash, TrendingUp, Clock, ChevronLeft, ChevronRight, Loader, CreditCard } from "lucide-react";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from "recharts";
import { useAdminSubscriptionOrderStats, useAdminSubscriptionOrders } from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 20;

const dayOptions = [
  { label: "7 天", value: 7 },
  { label: "30 天", value: 30 },
  { label: "90 天", value: 90 },
  { label: "365 天", value: 365 },
];

const STATUS_TABS = [
  { key: "", label: "全部" },
  { key: "paid", label: "已支付" },
  { key: "pending", label: "待支付" },
] as const;

const TYPE_TABS = [
  { key: "", label: "全部类型" },
  { key: "new", label: "新订阅" },
  { key: "upgrade", label: "升级" },
  { key: "renew", label: "续购" },
] as const;

const statusLabel = (s: string) => {
  switch (s) {
    case "paid": return "已支付";
    case "pending": return "待支付";
    default: return s;
  }
};

const statusBadge = (s: string) => {
  switch (s) {
    case "paid": return "bg-emerald-500/10 text-emerald-600 border-emerald-500/20 dark:text-emerald-400";
    case "pending": return "bg-amber-500/10 text-amber-600 border-amber-500/20 dark:text-amber-400";
    default: return "bg-muted text-muted-foreground border-border";
  }
};

const typeLabel = (t: string) => {
  switch (t) {
    case "new": return "新订阅";
    case "upgrade": return "升级";
    case "renew": return "续购";
    default: return t;
  }
};

const AdminSubscriptionOrders = () => {
  const [days, setDays] = useState(30);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState("");

  const { data: stats, isLoading: statsLoading } = useAdminSubscriptionOrderStats(days);
  const { data: ordersData, isLoading: ordersLoading } = useAdminSubscriptionOrders(page, PAGE_SIZE, statusFilter, typeFilter);

  const orders = ordersData?.orders ?? [];
  const total = ordersData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const dailyTrend = stats?.daily_trend ?? [];
  const planBreakdown = stats?.plan_breakdown ?? [];

  if (statsLoading) {
    return (
      <div className="page-container fade-in flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader size={28} className="animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">加载中...</span>
        </div>
      </div>
    );
  }

  const statCards = [
    { label: "总收入", value: `¥${(stats?.total_revenue ?? 0).toFixed(2)}`, icon: DollarSign, iconBg: "bg-emerald-500/10 dark:bg-emerald-500/15", iconColor: "text-emerald-500" },
    { label: "总订单数", value: String(stats?.total_orders ?? 0), icon: Hash, iconBg: "bg-primary/10", iconColor: "text-primary" },
    { label: "已支付", value: String(stats?.paid_orders ?? 0), icon: CreditCard, iconBg: "bg-violet-500/10 dark:bg-violet-500/15", iconColor: "text-violet-500" },
    { label: "待支付", value: String(stats?.pending_orders ?? 0), icon: Clock, iconBg: "bg-amber-500/10 dark:bg-amber-500/15", iconColor: "text-amber-500" },
    { label: "平均订单金额", value: `¥${(stats?.avg_order_value ?? 0).toFixed(2)}`, icon: TrendingUp, iconBg: "bg-orange-500/10 dark:bg-orange-500/15", iconColor: "text-orange-500" },
  ];

  return (
    <div className="page-container fade-in">
      {/* Header + time selector */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-lg font-bold text-foreground">订阅订单统计</h1>
          <p className="text-sm text-muted-foreground mt-0.5">查看订阅订单的收入和趋势数据</p>
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
      <div className="grid grid-cols-1 gap-4 mb-6 sm:grid-cols-2 xl:grid-cols-4">
        {statCards.map((card, i) => (
          <div key={card.label} className="flex-1 bg-card border border-border rounded-xl p-4 shadow-card stagger-item" style={{ animationDelay: `${i * 80}ms` }}>
            <div className="flex items-center gap-3 mb-2">
              <div className={`w-8 h-8 ${card.iconBg} rounded-lg flex items-center justify-center`}>
                <card.icon size={15} className={card.iconColor} />
              </div>
              <span className="text-xs text-muted-foreground font-medium">{card.label}</span>
            </div>
            <div className="text-xl font-bold text-foreground tabular-nums">{card.value}</div>
          </div>
        ))}
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 gap-4 mb-6 lg:grid-cols-2">
        {/* Daily trend */}
        <div className="bg-card border border-border rounded-xl shadow-card p-5">
          <h2 className="text-sm font-semibold text-foreground mb-4">每日订阅收入趋势</h2>
          {dailyTrend.length === 0 ? (
            <div className="flex items-center justify-center h-48 text-sm text-muted-foreground">暂无数据</div>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={dailyTrend}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                <XAxis dataKey="date" tick={{ fontSize: 11 }} tickFormatter={(v) => v.slice(5)} />
                <YAxis tick={{ fontSize: 11 }} tickFormatter={(v) => `¥${v}`} />
                <Tooltip formatter={(v: number) => [`¥${v.toFixed(2)}`, "收入"]} labelFormatter={(l) => `日期: ${l}`} />
                <Area type="monotone" dataKey="amount" stroke="hsl(var(--primary))" fill="hsl(var(--primary) / 0.15)" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </div>

        {/* Plan breakdown */}
        <div className="bg-card border border-border rounded-xl shadow-card p-5">
          <h2 className="text-sm font-semibold text-foreground mb-4">按套餐分布</h2>
          {planBreakdown.length === 0 ? (
            <div className="flex items-center justify-center h-48 text-sm text-muted-foreground">暂无数据</div>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={planBreakdown}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                <XAxis dataKey="plan_name" tick={{ fontSize: 11 }} />
                <YAxis tick={{ fontSize: 11 }} tickFormatter={(v) => `¥${v}`} />
                <Tooltip formatter={(v: number) => [`¥${v.toFixed(2)}`, "收入"]} />
                <Bar dataKey="amount" fill="hsl(var(--primary))" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      {/* Orders table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-foreground">订单明细</span>
            <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
              {ordersLoading ? "..." : `${total} 条`}
            </span>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex gap-1">
              {STATUS_TABS.map((tab) => (
                <button
                  key={tab.key}
                  onClick={() => { setStatusFilter(tab.key); setPage(1); }}
                  className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
                    statusFilter === tab.key
                      ? "bg-primary text-primary-foreground"
                      : "text-muted-foreground hover:text-foreground hover:bg-muted"
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
            <div className="w-px h-4 bg-border" />
            <div className="flex gap-1">
              {TYPE_TABS.map((tab) => (
                <button
                  key={tab.key}
                  onClick={() => { setTypeFilter(tab.key); setPage(1); }}
                  className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
                    typeFilter === tab.key
                      ? "bg-primary text-primary-foreground"
                      : "text-muted-foreground hover:text-foreground hover:bg-muted"
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>
        </div>

        {ordersLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : orders.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20">
            <div className="w-12 h-12 bg-muted rounded-xl flex items-center justify-center mb-3">
              <CreditCard size={18} className="text-muted-foreground/50" />
            </div>
            <div className="text-sm text-muted-foreground">暂无订阅订单</div>
          </div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3 font-semibold">订单ID</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">用户</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">套餐</TableHead>
                <TableHead className="text-right px-5 py-3 font-semibold">金额</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">类型</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">状态</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">创建时间</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">支付时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.map((order, i) => (
                <TableRow
                  key={order.id}
                  className={`border-t border-border hover:bg-muted/40 transition-colors ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                >
                  <TableCell className="px-5 py-3 text-sm font-mono text-foreground">{order.id.slice(0, 8)}…</TableCell>
                  <TableCell className="px-5 py-3 text-sm text-muted-foreground">{order.user_identifier || "—"}</TableCell>
                  <TableCell className="px-5 py-3 text-sm text-foreground">{order.plan_name}</TableCell>
                  <TableCell className="px-5 py-3 text-sm text-foreground text-right tabular-nums">¥{order.amount_cny.toFixed(2)}</TableCell>
                  <TableCell className="px-5 py-3 text-sm text-muted-foreground">{typeLabel(order.type)}</TableCell>
                  <TableCell className="px-5 py-3">
                    <span className={`inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full border ${statusBadge(order.status)}`}>
                      {statusLabel(order.status)}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-3 text-sm text-muted-foreground">
                    {new Date(order.created_at).toLocaleString("zh-CN")}
                  </TableCell>
                  <TableCell className="px-5 py-3 text-sm text-muted-foreground">
                    {order.paid_at ? new Date(order.paid_at).toLocaleString("zh-CN") : "—"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {totalPages > 1 && (
          <div className="px-5 py-3 border-t border-border flex items-center justify-between">
            <span className="text-xs text-muted-foreground">
              第 {page} / {totalPages} 页
            </span>
            <div className="flex items-center gap-1">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted disabled:opacity-30 disabled:pointer-events-none transition-colors"
              >
                <ChevronLeft size={14} />
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted disabled:opacity-30 disabled:pointer-events-none transition-colors"
              >
                <ChevronRight size={14} />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default AdminSubscriptionOrders;
