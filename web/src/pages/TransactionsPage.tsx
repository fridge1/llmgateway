import { useState } from "react";
import { BarChart2, ArrowDownLeft, ArrowUpRight, DollarSign, Sparkles, Crown, Calendar, ChevronLeft, ChevronRight, Loader2, Download } from "lucide-react";
import { useTransactions } from "@/hooks/use-api";
import TenantDiscountBanner from "@/components/TenantDiscountBanner";
import { PageHeader } from "@/components/ui/page-header";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const tabs = [
  { key: "all", label: "全部" },
  { key: "consumption", label: "余额消费" },
  { key: "subscription_usage", label: "订阅消费" },
  { key: "sub_purchase", label: "套餐购买" },
  { key: "recharge", label: "充值" },
  { key: "reward", label: "奖励收入" },
];

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("zh-CN");
}

function TypeBadge({ type }: { type: string }) {
  if (type === "consumption") return <span className="badge-danger">余额消费</span>;
  if (type === "subscription_usage") return <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-violet-50 text-violet-700 border border-violet-200 dark:bg-violet-500/10 dark:text-violet-400 dark:border-violet-500/30">订阅消费</span>;
  if (type === "sub_purchase") return <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-50 text-indigo-700 border border-indigo-200 dark:bg-indigo-500/10 dark:text-indigo-400 dark:border-indigo-500/30">套餐购买</span>;
  if (type === "recharge") return <span className="badge-success">充值</span>;
  if (type === "checkin") return <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-orange-50 text-orange-700 border border-orange-200 dark:bg-orange-500/10 dark:text-orange-400 dark:border-orange-500/30">签到奖励</span>;
  if (type === "task_reward") return <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/30">任务奖励</span>;
  return <span className="badge-secondary">{type}</span>;
}

const TransactionsPage = () => {
  const [activeTab, setActiveTab] = useState("all");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(20);

  const typeFilter = activeTab === "all" ? undefined : activeTab === "reward" ? "checkin,task_reward" : activeTab;
  const { data, isLoading } = useTransactions(page, size, typeFilter, startDate || undefined, endDate || undefined);
  const transactions = data?.transactions ?? [];
  const total = data?.total ?? 0;
  const totalConsumption = data?.total_consumption ?? 0;
  const totalRecharge = data?.total_recharge ?? 0;
  const totalSubscriptionUsage = data?.total_subscription_usage ?? 0;
  const totalSubscriptionPurchase = data?.total_sub_purchase ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  const net = totalRecharge - totalConsumption - totalSubscriptionPurchase;

  const stats = [
    { label: "总记录", value: String(total), icon: BarChart2, iconBg: "bg-blue-50 dark:bg-blue-500/10", iconColor: "text-blue-500" },
    { label: "余额消费", value: `-\u00A5${totalConsumption.toFixed(4)}`, icon: ArrowDownLeft, iconBg: "bg-red-50 dark:bg-red-500/10", iconColor: "text-red-500", valueColor: "text-destructive" },
    { label: "订阅消费", value: `\u00A5${totalSubscriptionUsage.toFixed(4)}`, icon: Sparkles, iconBg: "bg-violet-50 dark:bg-violet-500/10", iconColor: "text-violet-500", valueColor: "text-violet-600 dark:text-violet-400" },
    { label: "套餐购买", value: `-\u00A5${totalSubscriptionPurchase.toFixed(4)}`, icon: Crown, iconBg: "bg-indigo-50 dark:bg-indigo-500/10", iconColor: "text-indigo-500", valueColor: "text-indigo-600 dark:text-indigo-400" },
    { label: "总充值", value: `+\u00A5${totalRecharge.toFixed(4)}`, icon: ArrowUpRight, iconBg: "bg-emerald-50 dark:bg-emerald-500/10", iconColor: "text-emerald-500", valueColor: "text-success" },
    { label: "净额", value: `\u00A5${net.toFixed(4)}`, icon: DollarSign, iconBg: "bg-amber-50 dark:bg-amber-500/10", iconColor: "text-amber-500", valueColor: "text-amber-600 dark:text-amber-400" },
  ];

  const handleTabChange = (key: string) => {
    setActiveTab(key);
    setPage(1);
  };

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
      <PageHeader
        eyebrow="资金"
        title="交易记录"
        description="追踪余额消费、订阅消费、充值与奖励明细。"
      />

      {/* Stats */}
      <TenantDiscountBanner />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4 mb-6">
        {stats.map((s, idx) => {
          const Icon = s.icon;
          return (
            <div key={s.label} className="flex-1 stat-card stagger-item" style={{ animationDelay: `${idx * 80}ms` }}>
              <div className="flex items-start justify-between mb-3">
                <div className={`w-9 h-9 ${s.iconBg} rounded-lg flex items-center justify-center`}>
                  <Icon size={18} className={s.iconColor} />
                </div>
              </div>
              <div className={`text-xl font-bold mb-1 ${s.valueColor || "text-foreground"}`}>{s.value}</div>
              <div className="text-xs text-muted-foreground">{s.label}</div>
            </div>
          );
        })}
      </div>

      {/* Filter bar */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-1 bg-card border border-border rounded-xl p-1 shadow-card">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => handleTabChange(tab.key)}
              className={`px-4 py-1.5 rounded-lg text-sm font-medium transition-all duration-200 ${
                activeTab === tab.key
                  ? "brand-gradient text-white shadow-button"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-2 bg-card border border-border rounded-lg px-3 py-2 text-sm shadow-card">
            <Calendar size={14} className="text-muted-foreground" />
            <input type="date" className="bg-transparent text-sm text-muted-foreground outline-none cursor-pointer dark:text-muted-foreground dark:[color-scheme:dark]" placeholder="开始日期" value={startDate} onChange={(e) => { setStartDate(e.target.value); setPage(1); }} />
            <span className="text-muted-foreground">&rarr;</span>
            <input type="date" className="bg-transparent text-sm text-muted-foreground outline-none cursor-pointer dark:text-muted-foreground dark:[color-scheme:dark]" placeholder="结束日期" value={endDate} onChange={(e) => { setEndDate(e.target.value); setPage(1); }} />
          </div>
          <button
            onClick={() => {
              const params = new URLSearchParams();
              if (startDate) params.set("start_date", startDate);
              if (endDate) params.set("end_date", endDate);
              window.open(`/api/billing/transactions/export?${params}`, "_blank");
            }}
            className="flex items-center gap-1.5 bg-card border border-border rounded-lg px-3 py-2 text-sm shadow-card hover:bg-muted"
            title="按当前日期范围导出 Excel"
          >
            <Download size={14} className="text-muted-foreground" />
            导出
          </button>
        </div>
      </div>

      {/* Table */}
      <div className="data-table-card">
        <Table className="w-full">
          <TableHeader>
            <TableRow className="table-header">
              <TableHead className="text-left px-5 py-3">时间</TableHead>
              <TableHead className="text-left px-5 py-3">类型</TableHead>
              <TableHead className="text-left px-5 py-3">模型</TableHead>
              <TableHead className="text-right px-5 py-3">输入</TableHead>
              <TableHead className="text-right px-5 py-3">输出</TableHead>
              <TableHead className="text-right px-5 py-3">缓存命中</TableHead>
              <TableHead className="text-right px-5 py-3">缓存写入</TableHead>
              <TableHead className="text-right px-5 py-3">金额</TableHead>
              <TableHead className="text-right px-5 py-3">余额</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {transactions.length === 0 ? (
              <TableRow>
                <TableCell colSpan={9} className="px-5 py-16">
                  <div className="empty-state">
                    <div className="w-12 h-12 bg-muted rounded-full flex items-center justify-center mb-3">
                      <BarChart2 size={20} className="text-muted-foreground" />
                    </div>
                    <div className="text-sm font-medium text-muted-foreground">暂无记录</div>
                    <div className="text-xs text-muted-foreground/70 mt-1">当前筛选条件下没有交易记录</div>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              transactions.map((t, i) => {
                const isRecharge = t.type === "recharge";
                const isReward = t.type === "checkin" || t.type === "task_reward";
                const isSubscription = t.type === "subscription_usage";
                const isSubPurchase = t.type === "sub_purchase";
                const amountStr = (isRecharge || isReward)
                  ? `+\u00A5${t.amount.toFixed(4)}`
                  : `-\u00A5${Math.abs(t.amount).toFixed(4)}`;
                const amountColor = (isRecharge || isReward) ? "text-success" : isSubscription ? "text-violet-600 dark:text-violet-400" : isSubPurchase ? "text-indigo-600 dark:text-indigo-400" : "text-destructive";
                return (
                  <TableRow key={t.id} className={`border-t border-border hover:bg-muted/30 transition-colors ${i % 2 === 0 ? "" : "bg-muted/10"}`}>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{formatTime(t.created_at)}</TableCell>
                    <TableCell className="px-5 py-3.5"><TypeBadge type={t.type} /></TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground font-mono">{t.model ?? "—"}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground text-right font-mono">
                      {t.prompt_tokens != null ? t.prompt_tokens.toLocaleString() : "—"}
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground text-right font-mono">
                      {t.completion_tokens != null ? t.completion_tokens.toLocaleString() : "—"}
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground text-right font-mono">
                      {t.cache_read_tokens != null ? t.cache_read_tokens.toLocaleString() : "—"}
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground text-right font-mono">
                      {t.cache_creation_5m_tokens || t.cache_creation_1h_tokens
                        ? `5m: ${(t.cache_creation_5m_tokens ?? 0).toLocaleString()} / 1h: ${(t.cache_creation_1h_tokens ?? 0).toLocaleString()}`
                        : t.cache_creation_tokens != null
                          ? t.cache_creation_tokens.toLocaleString()
                          : "—"}
                    </TableCell>
                    <TableCell className={`px-5 py-3.5 text-sm font-medium text-right ${amountColor}`}>
                      {amountStr}
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-foreground text-right font-medium">{`\u00A5${t.balance_after.toFixed(4)}`}</TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>

        {/* Pagination */}
        <div className="px-5 py-3.5 border-t border-border flex items-center justify-between">
          <div className="text-xs text-muted-foreground">共 {total} 条记录</div>
          <div className="flex items-center gap-2">
            <button
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
              className={`flex h-9 w-9 items-center justify-center rounded-lg text-sm transition-colors ${
                page <= 1 ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted/60 cursor-pointer"
              }`}
              aria-label="上一页"
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
              className={`flex h-9 w-9 items-center justify-center rounded-lg text-sm transition-colors ${
                page >= totalPages ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted/60 cursor-pointer"
              }`}
              aria-label="下一页"
            >
              <ChevronRight size={14} />
            </button>
            <select
              className="ml-2 bg-muted border border-border rounded-lg px-2 py-1 text-xs text-foreground outline-none cursor-pointer dark:bg-muted dark:border-border"
              value={size}
              onChange={(e) => { setSize(Number(e.target.value)); setPage(1); }}
            >
              <option value={10}>10条/页</option>
              <option value={20}>20条/页</option>
              <option value={50}>50条/页</option>
            </select>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TransactionsPage;
