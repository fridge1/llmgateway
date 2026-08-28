import { useState } from "react";
import { Wallet, Flame, CalendarDays, Key, TrendingUp, ArrowDownRight, ChevronRight, Plus, Zap, Sparkles, ShoppingBag } from "lucide-react";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from "recharts";
import { useNavigate } from "react-router-dom";
import { useBalance, useBillingStats, useApiKeys, useTransactions, useTokenStats, usePublicRechargeLottery, usePublicRechargeLotteryRounds } from "@/hooks/use-api";
import { useAuth } from "@/contexts/AuthContext";
import { Skeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";

function formatTime(iso: string): string {
  const d = new Date(iso);
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  const hh = String(d.getHours()).padStart(2, "0");
  const mi = String(d.getMinutes()).padStart(2, "0");
  return `${mm}-${dd} ${hh}:${mi}`;
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

const DashboardPage = () => {
  const navigate = useNavigate();
  const { user } = useAuth();
  const balance = useBalance();
  const billingStats = useBillingStats(7);
  const keys = useApiKeys();
  const transactions = useTransactions(1, 5);
  const [tokenDays, setTokenDays] = useState<7 | 30 | 90>(7);
  const tokenStats = useTokenStats(tokenDays);
  const { data: lotteryData } = usePublicRechargeLottery();
  const { data: lotteryRoundsData } = usePublicRechargeLotteryRounds();
  const lottery = lotteryData?.lottery;
  const currentEntries = lotteryData?.current_entries ?? 0;
  const latestWinner = lotteryRoundsData?.rounds?.[0];

  const isLoading = balance.isLoading || billingStats.isLoading || keys.isLoading || transactions.isLoading || tokenStats.isLoading;

  const activeKeysCount = (keys.data ?? []).length;

  const totalBalance = balance.data?.balance ?? 0;

  const stats = [
    {
      label: "总余额",
      value: `\u00A5${totalBalance.toFixed(4)}`,
      sub: `冻结 \u00A5${(balance.data?.frozen ?? 0).toFixed(4)}`,
      icon: Wallet,
      iconBg: "bg-primary/10",
      iconColor: "text-primary",
      accent: true,
    },
    {
      label: "今日消费",
      value: `\u00A5${(billingStats.data?.today_cost ?? 0).toFixed(4)}`,
      sub: "今日累计",
      icon: Flame,
      iconBg: "bg-orange-50 dark:bg-orange-500/10",
      iconColor: "text-orange-500",
      trend: "neutral",
    },
    {
      label: "本月消费",
      value: `\u00A5${(billingStats.data?.month_cost ?? 0).toFixed(4)}`,
      sub: "本月累计",
      icon: CalendarDays,
      iconBg: "bg-emerald-50 dark:bg-emerald-500/10",
      iconColor: "text-emerald-500",
      trend: "neutral",
    },
    {
      label: "API 密钥",
      value: `${(keys.data ?? []).length}`,
      sub: `${activeKeysCount} 个活跃`,
      icon: Key,
      iconBg: "bg-violet-50 dark:bg-violet-500/10",
      iconColor: "text-violet-500",
    },
  ];

  const chartData = billingStats.data?.daily_trend ?? [];
  const modelData = billingStats.data?.model_breakdown ?? [];
  const txList = transactions.data?.transactions ?? [];

  if (isLoading) {
    return (
      <div className="page-container fade-in">
        <div className="mb-6">
          <Skeleton className="h-7 w-24" />
          <Skeleton className="h-4 w-48 mt-2" />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="rounded-xl border p-4">
              <Skeleton className="h-4 w-16 mb-2" />
              <Skeleton className="h-7 w-28" />
              <Skeleton className="h-3 w-20 mt-1" />
            </div>
          ))}
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <Skeleton className="h-64 rounded-xl" />
          <Skeleton className="h-64 rounded-xl" />
        </div>
      </div>
    );
  }

  return (
    <div className="page-container fade-in">
      <PageHeader
        eyebrow="工作台"
        title="仪表盘"
        description="欢迎回来，集中查看余额、调用量、费用趋势与最近交易。"
      />

      {/* Recharge lottery banner */}
      {lottery?.status === "active" && (
        <div className="mb-6 rounded-2xl px-5 py-4 border border-violet-200/60 dark:border-violet-500/30 banner-violet transition-shadow duration-300 hover:shadow-card">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-xl bg-violet-500/15 dark:bg-violet-400/20 flex items-center justify-center shrink-0">
                <Sparkles size={18} className="text-violet-600 dark:text-violet-400" />
              </div>
              <div className="min-w-0">
                <div className="text-sm font-semibold text-foreground">{lottery.name} · 第 {lottery.total_rounds + 1} 期</div>
                <div className="text-xs text-muted-foreground mt-0.5">
                  每 {lottery.trigger_every} 笔 ≥¥20 充值自动开奖 · 已累计 {currentEntries} 笔
                </div>
              </div>
            </div>
            {latestWinner && (
              <span className="text-xs text-violet-700/80 dark:text-violet-400/90 sm:ml-auto sm:text-right">
                上期：{latestWinner.winner_nickname || "用户***"} 获 ¥{latestWinner.winner_amount.toFixed(2)}
              </span>
            )}
          </div>
          <div className="mt-3">
            <div className="mb-1 flex justify-between text-xs text-muted-foreground">
              <span>开奖进度</span>
              <span>{Math.min(currentEntries, lottery.trigger_every)}/{lottery.trigger_every}</span>
            </div>
            <div className="w-full bg-violet-200/40 dark:bg-violet-500/20 rounded-full h-1.5">
              <div
                className="bg-violet-500 dark:bg-violet-400 h-1.5 rounded-full transition-all duration-500"
                style={{ width: `${Math.min(100, (currentEntries / lottery.trigger_every) * 100)}%` }}
              />
            </div>
          </div>
        </div>
      )}

      {/* Stats row */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4 mb-6">
        {stats.map((stat, i) => {
          const Icon = stat.icon;
          return (
            <div
              key={stat.label}
              className={`flex-1 rounded-xl p-5 border transition-all duration-300 cursor-default stagger-item ${
                stat.accent
                  ? "brand-gradient text-white border-transparent shadow-card hover:shadow-elevated hover:-translate-y-0.5"
                  : "bg-card border-border shadow-card hover:shadow-elevated hover:-translate-y-0.5"
              }`}
              style={{ animationDelay: `${i * 80}ms` }}
            >
              <div className="flex items-start justify-between mb-3">
                <div className={`w-9 h-9 rounded-lg flex items-center justify-center ${stat.accent ? "bg-white/15" : stat.iconBg}`}>
                  <Icon size={18} className={stat.accent ? "text-white" : stat.iconColor} />
                </div>
                {stat.accent && (
                  <button
                    onClick={() => navigate("balance")}
                    className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-white/20 hover:bg-white/30 text-white text-xs font-medium transition-colors"
                  >
                    <Plus size={14} />
                    充值
                  </button>
                )}
                {stat.trend !== undefined && (
                  <span className="flex items-center gap-0.5 text-xs text-muted-foreground">
                    <TrendingUp size={12} />
                  </span>
                )}
              </div>
              <div className={`text-2xl font-bold mb-1 ${stat.accent ? "text-white" : "text-foreground"}`}>
                {stat.value}
              </div>
              <div className={`text-xs ${stat.accent ? "text-white/70" : "text-muted-foreground"}`}>
                {stat.label} · {stat.sub}
              </div>
            </div>
          );
        })}
      </div>

      {/* Codex 商店入口 */}
      <div
        className="mb-6 rounded-2xl px-5 py-4 border border-emerald-200/60 dark:border-emerald-500/30 banner-emerald cursor-pointer hover:shadow-card hover:-translate-y-0.5 transition-all duration-300"
        onClick={() => navigate("/codex-shop")}
      >
        <div className="flex items-center gap-3">
          <div className="w-11 h-11 rounded-xl bg-emerald-500/15 dark:bg-emerald-400/20 flex items-center justify-center shrink-0">
            <ShoppingBag size={20} className="text-emerald-600 dark:text-emerald-400" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="text-sm font-semibold text-foreground">Codex 代充服务</div>
            <div className="text-xs text-muted-foreground mt-0.5">
              ChatGPT Pro / Plus 账号代充 · 在线支付 · 自动发货
            </div>
          </div>
          <ChevronRight size={18} className="text-muted-foreground shrink-0" />
        </div>
      </div>

      {/* Token usage card */}
      <div className="bg-card border border-border rounded-xl p-5 shadow-card mb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <div className="text-sm font-semibold text-foreground">Token 用量</div>
            <div className="text-xs text-muted-foreground mt-0.5">最近 {tokenDays} 天 · 含累计</div>
          </div>
          <div className="inline-flex rounded-lg border border-border bg-muted/30 p-0.5">
            {([7, 30, 90] as const).map((d) => (
              <button
                key={d}
                onClick={() => setTokenDays(d)}
                className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                  tokenDays === d
                    ? "bg-card text-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {d} 天
              </button>
            ))}
          </div>
        </div>
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {([
            { label: "输入", key: "prompt", color: "text-sky-600 dark:text-sky-400" },
            { label: "输出", key: "completion", color: "text-emerald-600 dark:text-emerald-400" },
            { label: "缓存命中", key: "cache_read", color: "text-violet-600 dark:text-violet-400" },
            { label: "缓存写入", key: "cache_creation", color: "text-amber-600 dark:text-amber-400" },
          ] as const).map((c) => {
            const periodVal = tokenStats.data?.period[c.key] ?? 0;
            const allTimeVal = tokenStats.data?.all_time[c.key] ?? 0;
            return (
              <div key={c.label} className="rounded-lg bg-muted/20 border border-border/50 p-4">
                <div className="text-xs text-muted-foreground mb-1.5">{c.label}</div>
                <div
                  className={`text-2xl font-bold ${c.color}`}
                  title={periodVal.toLocaleString()}
                >
                  {formatTokens(periodVal)}
                </div>
                <div
                  className="text-xs text-muted-foreground mt-1"
                  title={allTimeVal.toLocaleString()}
                >
                  累计 {formatTokens(allTimeVal)}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-1 gap-4 mb-6 lg:grid-cols-5">
        {/* Cost trend */}
        <div className="lg:col-span-3 bg-card border border-border rounded-xl p-5 shadow-card">
          <div className="flex items-center justify-between mb-4">
            <div>
              <div className="text-sm font-semibold text-foreground">费用趋势</div>
              <div className="text-xs text-muted-foreground mt-0.5">近 7 天消费变化</div>
            </div>
            <button onClick={() => navigate("transactions")} className="text-xs text-primary hover:underline flex items-center gap-1 font-medium">
              查看详情 <ChevronRight size={12} />
            </button>
          </div>
          <div style={{ height: 160 }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                <defs>
                  <linearGradient id="colorCost" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--primary)" stopOpacity={0.15} />
                    <stop offset="95%" stopColor="var(--primary)" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
                <XAxis dataKey="date" tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} />
                <Tooltip
                  contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "10px", fontSize: 12, boxShadow: "0 4px 6px -1px rgba(0,0,0,0.05)" }}
                  formatter={(v: number) => [`\u00A5${v.toFixed(4)}`, "消费"]}
                />
                <Area type="monotone" dataKey="cost" stroke="var(--primary)" strokeWidth={2} fill="url(#colorCost)" dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Model usage */}
        <div className="lg:col-span-2 bg-card border border-border rounded-xl p-5 shadow-card">
          <div className="flex items-center justify-between mb-4">
            <div>
              <div className="text-sm font-semibold text-foreground">模型用量</div>
              <div className="text-xs text-muted-foreground mt-0.5">模型消费分布</div>
            </div>
          </div>
          <div style={{ height: 160 }}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={modelData} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
                <XAxis dataKey="model" tick={{ fontSize: 10, fill: "var(--muted-foreground)" }} axisLine={false} tickLine={false} />
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
      </div>

      {/* Claude Overview Cards */}
      {(() => {
        const claudeModels = modelData.filter((m) => m.model.toLowerCase().includes('claude'));
        const claudeTotalCost = claudeModels.reduce((sum, m) => sum + m.cost, 0);
        const topClaudeModel = claudeModels.length > 0
          ? claudeModels.reduce((max, m) => m.cost > max.cost ? m : max, claudeModels[0])
          : null;

        if (claudeTotalCost > 0) {
          return (
            <div className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-3">
              <div className="bg-gradient-to-br from-orange-50 to-orange-100/50 dark:from-orange-500/10 dark:to-orange-600/5 border border-orange-200/60 dark:border-orange-500/20 rounded-xl p-5 shadow-card stagger-item transition-shadow duration-300 hover:shadow-card" style={{ animationDelay: "0ms" }}>
                <div className="flex items-center gap-2 mb-3">
                  <div className="w-8 h-8 bg-orange-500 rounded-lg flex items-center justify-center">
                    <span className="text-white font-bold text-sm">C</span>
                  </div>
                  <div className="text-sm font-semibold text-orange-900 dark:text-orange-300">Claude 总消费</div>
                </div>
                <div className="text-2xl font-bold text-orange-900 dark:text-orange-200 mb-1">
                  {"\u00A5"}{claudeTotalCost.toFixed(4)}
                </div>
                <div className="text-xs text-orange-700 dark:text-orange-400">
                  本月累计 · {claudeModels.length} 个模型
                </div>
              </div>

              <div className="bg-gradient-to-br from-indigo-50 to-indigo-100/50 dark:from-indigo-500/10 dark:to-indigo-600/5 border border-indigo-200/60 dark:border-indigo-500/20 rounded-xl p-5 shadow-card stagger-item transition-shadow duration-300 hover:shadow-card" style={{ animationDelay: "80ms" }}>
                <div className="flex items-center gap-2 mb-3">
                  <div className="w-8 h-8 bg-indigo-500 rounded-lg flex items-center justify-center">
                    <Zap size={16} className="text-white" />
                  </div>
                  <div className="text-sm font-semibold text-indigo-900 dark:text-indigo-300">最常用模型</div>
                </div>
                <div className="text-lg font-bold text-indigo-900 dark:text-indigo-200 mb-1">
                  {topClaudeModel?.model || '-'}
                </div>
                <div className="text-xs text-indigo-700 dark:text-indigo-400">
                  消费 {"\u00A5"}{(topClaudeModel?.cost ?? 0).toFixed(4)}
                </div>
              </div>

              <div className="bg-gradient-to-br from-violet-50 to-violet-100/50 dark:from-violet-500/10 dark:to-violet-600/5 border border-violet-200/60 dark:border-violet-500/20 rounded-xl p-5 shadow-card stagger-item transition-shadow duration-300 hover:shadow-card" style={{ animationDelay: "160ms" }}>
                <div className="flex items-center gap-2 mb-3">
                  <div className="w-8 h-8 bg-violet-500 rounded-lg flex items-center justify-center">
                    <TrendingUp size={16} className="text-white" />
                  </div>
                  <div className="text-sm font-semibold text-violet-900 dark:text-violet-300">省钱提示</div>
                </div>
                <div className="text-sm text-violet-900 dark:text-violet-200 mb-1">
                  使用 Prompt Cache
                </div>
                <div className="text-xs text-violet-700 dark:text-violet-400">
                  可节省 90% 输入成本
                </div>
              </div>
            </div>
          );
        }
        return null;
      })()}

      {/* Recent transactions */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div>
            <div className="text-sm font-semibold text-foreground">最近交易</div>
            <div className="text-xs text-muted-foreground mt-0.5">最新的消费记录</div>
          </div>
          <button onClick={() => navigate("transactions")} className="text-xs text-primary hover:underline flex items-center gap-1 font-medium">
            查看全部 <ChevronRight size={12} />
          </button>
        </div>
        <Table>
          <TableHeader>
            <TableRow className="table-header">
              <TableHead className="text-left px-5 py-3">时间</TableHead>
              <TableHead className="text-left px-5 py-3">类型</TableHead>
              <TableHead className="text-left px-5 py-3">金额</TableHead>
              <TableHead className="text-left px-5 py-3">余额</TableHead>
              <TableHead className="text-left px-5 py-3">模型 / 描述</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {txList.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="px-5 py-16">
                  <div className="empty-state">
                    <div className="w-12 h-12 bg-muted rounded-full flex items-center justify-center mb-3">
                      <ArrowDownRight size={20} className="text-muted-foreground" />
                    </div>
                    <div className="text-sm font-medium text-muted-foreground">暂无交易记录</div>
                    <div className="text-xs text-muted-foreground/70 mt-1">您的消费记录将在这里显示</div>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              txList.map((t) => {
                const isRecharge = t.type === "recharge";
                const isSubPurchase = t.type === "sub_purchase";
                const isSubUsage = t.type === "subscription_usage";
                const isCheckin = t.type === "checkin";
                const isTaskReward = t.type === "task_reward";
                const isReward = isCheckin || isTaskReward || t.type === "referral_bonus";
                const isCredit = isRecharge || isReward;
                const amountStr = isCredit
                  ? `+\u00A5${Math.abs(t.amount).toFixed(4)}`
                  : `-\u00A5${Math.abs(t.amount).toFixed(4)}`;
                return (
                  <TableRow key={t.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{formatTime(t.created_at)}</TableCell>
                    <TableCell className="px-5 py-3.5">
                      {isRecharge ? (
                        <span className="badge-success">充值</span>
                      ) : isSubPurchase ? (
                        <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-50 text-indigo-700 border border-indigo-200 dark:bg-indigo-500/10 dark:text-indigo-400 dark:border-indigo-500/30">套餐购买</span>
                      ) : isSubUsage ? (
                        <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-violet-50 text-violet-700 border border-violet-200 dark:bg-violet-500/10 dark:text-violet-400 dark:border-violet-500/30">订阅消费</span>
                      ) : isCheckin ? (
                        <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-orange-50 text-orange-700 border border-orange-200 dark:bg-orange-500/10 dark:text-orange-400 dark:border-orange-500/30">签到奖励</span>
                      ) : isTaskReward ? (
                        <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/30">任务奖励</span>
                      ) : (
                        <span className="badge-danger">消费</span>
                      )}
                    </TableCell>
                    <TableCell className={`px-5 py-3.5 text-sm font-medium ${isCredit ? "text-success" : "text-destructive"}`}>
                      {amountStr}
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-foreground font-medium">{"\u00A5"}{t.balance_after.toFixed(4)}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{t.model ?? t.description ?? "\u2014"}</TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
};

export default DashboardPage;
