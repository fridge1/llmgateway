import { useState, useEffect } from "react";
import { Users, Activity, DollarSign, UserX, ChevronLeft, ChevronRight, Loader, Search } from "lucide-react";
import { useAdminSubscriptionUsersUsage, useSubscriptionPlans } from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 20;

const STATUS_TABS = [
  { key: "", label: "全部" },
  { key: "active", label: "活跃" },
  { key: "expired", label: "已过期" },
] as const;

const statusLabel = (s: string) => {
  switch (s) {
    case "active": return "活跃";
    case "expired": return "已过期";
    default: return s;
  }
};

const statusBadge = (s: string) => {
  switch (s) {
    case "active": return "bg-emerald-500/10 text-emerald-600 border-emerald-500/20 dark:text-emerald-400";
    case "expired": return "bg-gray-500/10 text-gray-600 border-gray-500/20 dark:text-gray-400";
    default: return "bg-muted text-muted-foreground border-border";
  }
};

const AdminSubscriptionUsage = () => {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [statusFilter, setStatusFilterRaw] = useState("");
  const [planFilter, setPlanFilterRaw] = useState("");

  // Reset to page 1 whenever search debounces to a new value, or filters change.
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch((prev) => {
        if (prev !== search) {
          setPage(1);
          return search;
        }
        return prev;
      });
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const setStatusFilter = (v: string) => {
    setStatusFilterRaw(v);
    setPage(1);
  };
  const setPlanFilter = (v: string) => {
    setPlanFilterRaw(v);
    setPage(1);
  };

  const { data: usageData, isLoading } = useAdminSubscriptionUsersUsage(
    page,
    PAGE_SIZE,
    debouncedSearch,
    statusFilter,
    planFilter
  );
  const { data: plansData } = useSubscriptionPlans();

  const users = usageData?.users ?? [];
  const total = usageData?.total ?? 0;
  const activeCount = usageData?.active_count ?? 0;
  const totalUsage = usageData?.total_usage ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const plans = plansData?.plans ?? [];

  const expiredCount = total - activeCount;

  if (isLoading && page === 1) {
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
    { label: "总订阅用户", value: String(total), icon: Users, iconBg: "bg-primary/10", iconColor: "text-primary" },
    { label: "活跃订阅", value: String(activeCount), icon: Activity, iconBg: "bg-emerald-500/10 dark:bg-emerald-500/15", iconColor: "text-emerald-500" },
    { label: "本月总用量", value: `¥${totalUsage.toFixed(2)}`, icon: DollarSign, iconBg: "bg-violet-500/10 dark:bg-violet-500/15", iconColor: "text-violet-500" },
    { label: "已过期", value: String(expiredCount), icon: UserX, iconBg: "bg-gray-500/10 dark:bg-gray-500/15", iconColor: "text-gray-500" },
  ];

  const getUsageColor = (percent: number) => {
    if (percent >= 85) return "bg-red-500";
    if (percent >= 60) return "bg-amber-500";
    return "bg-emerald-500";
  };

  return (
    <div className="page-container fade-in">
      <div className="mb-6">
        <h1 className="text-lg font-bold text-foreground">订阅用量管理</h1>
        <p className="text-sm text-muted-foreground mt-0.5">查看所有订阅用户的用量和剩余配额</p>
      </div>

      {/* Summary cards */}
      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {statCards.map((card, i) => (
          <div key={card.label} className="bg-card border border-border rounded-xl p-4 shadow-card stagger-item" style={{ animationDelay: `${i * 80}ms` }}>
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

      {/* Filters */}
      <div className="bg-card border border-border rounded-xl shadow-card mb-4">
        <div className="p-4 flex items-center gap-4">
          {/* Search */}
          <div className="flex-1 relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              placeholder="搜索手机号或昵称..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-sm bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20"
            />
          </div>

          {/* Status filter */}
          <div className="flex items-center gap-1 bg-muted rounded-lg p-1">
            {STATUS_TABS.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setStatusFilter(tab.key)}
                className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
                  statusFilter === tab.key
                    ? "bg-card text-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Plan filter */}
          <select
            value={planFilter}
            onChange={(e) => setPlanFilter(e.target.value)}
            className="px-3 py-2 text-sm bg-background border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20"
          >
            <option value="">全部套餐</option>
            {plans.map((plan) => (
              <option key={plan.id} value={plan.id}>
                {plan.display_name}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader size={24} className="animate-spin text-primary" />
          </div>
        ) : users.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <Users size={32} className="mb-2 opacity-30" />
            <span className="text-sm">暂无订阅用户</span>
          </div>
        ) : (
          <Table className="w-full">
            <TableHeader className="bg-muted/30 border-b border-border">
              <TableRow>
                <TableHead className="px-5 py-3 text-left text-xs font-semibold text-muted-foreground">用户</TableHead>
                <TableHead className="px-5 py-3 text-left text-xs font-semibold text-muted-foreground">套餐</TableHead>
                <TableHead className="px-5 py-3 text-right text-xs font-semibold text-muted-foreground">配额</TableHead>
                <TableHead className="px-5 py-3 text-right text-xs font-semibold text-muted-foreground">已用</TableHead>
                <TableHead className="px-5 py-3 text-right text-xs font-semibold text-muted-foreground">剩余</TableHead>
                <TableHead className="px-5 py-3 text-left text-xs font-semibold text-muted-foreground">用量</TableHead>
                <TableHead className="px-5 py-3 text-right text-xs font-semibold text-muted-foreground">请求数</TableHead>
                <TableHead className="px-5 py-3 text-center text-xs font-semibold text-muted-foreground">状态</TableHead>
                <TableHead className="px-5 py-3 text-left text-xs font-semibold text-muted-foreground">到期时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody className="divide-y divide-border">
              {users.map((user) => {
                const isImage = user.plan_category === "image";
                const fmtQuota = (v: number) =>
                  isImage ? `${Math.round(v)} 张` : `¥${v.toFixed(2)}`;
                return (
                <TableRow key={user.subscription_id} className="hover:bg-muted/20 transition-colors">
                  <TableCell className="px-5 py-3">
                    <div className="flex flex-col">
                      <span className="text-sm font-medium text-foreground">{user.user_identifier}</span>
                      {user.user_nickname && (
                        <span className="text-xs text-muted-foreground">{user.user_nickname}</span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="px-5 py-3">
                    <div className="flex flex-col">
                      <span className="text-sm font-medium text-foreground">{user.plan_name}</span>
                      <span className="text-xs text-muted-foreground">¥{user.plan_price_cny.toFixed(2)}</span>
                    </div>
                  </TableCell>
                  <TableCell className="px-5 py-3 text-right text-sm text-foreground tabular-nums">
                    {fmtQuota(user.quota_amount_cny)}
                  </TableCell>
                  <TableCell className="px-5 py-3 text-right text-sm text-foreground tabular-nums">
                    {fmtQuota(user.amount_used)}
                  </TableCell>
                  <TableCell className="px-5 py-3 text-right text-sm text-foreground tabular-nums">
                    {fmtQuota(user.amount_remaining)}
                  </TableCell>
                  <TableCell className="px-5 py-3">
                    <div className="flex items-center gap-2">
                      <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden">
                        <div
                          className={`h-full ${getUsageColor(user.usage_percent)} transition-all`}
                          style={{ width: `${Math.min(user.usage_percent, 100)}%` }}
                        />
                      </div>
                      <span className="text-xs text-muted-foreground tabular-nums w-10 text-right">
                        {user.usage_percent.toFixed(0)}%
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="px-5 py-3 text-right text-sm text-muted-foreground tabular-nums">
                    {user.request_count.toLocaleString()}
                  </TableCell>
                  <TableCell className="px-5 py-3 text-center">
                    <span className={`inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full border ${statusBadge(user.status)}`}>
                      {statusLabel(user.status)}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-3 text-sm text-muted-foreground">
                    {new Date(user.expires_at).toLocaleString("zh-CN")}
                  </TableCell>
                </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}

        {totalPages > 1 && (
          <div className="px-5 py-3 border-t border-border flex items-center justify-between">
            <span className="text-xs text-muted-foreground">
              第 {page} / {totalPages} 页，共 {total} 条
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

export default AdminSubscriptionUsage;
