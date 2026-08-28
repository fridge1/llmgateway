import { Users, DollarSign, Zap, Activity, ChevronRight, RefreshCw, AlertCircle, Loader, Coins, TrendingDown, type LucideIcon } from "lucide-react";
import { useAdminDashboard, useAdminOrders, useGatewayStatus, useAdminConsumptionStats, useAdminFunnelStats } from "@/hooks/use-api";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/query-keys";
import { useMemo } from "react";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const AdminOverview = ({ onNavigate }: { onNavigate?: (page: string) => void }) => {
  const qc = useQueryClient();
  const { data: dashboard, isLoading: dashLoading } = useAdminDashboard();
  const { data: status, isLoading: statusLoading, isFetching } = useGatewayStatus();
  const { data: ordersData, isLoading: ordersLoading } = useAdminOrders(1, 10, "");
  const { data: consumptionData, isLoading: consumptionLoading } = useAdminConsumptionStats(30);
  const { data: funnelData, isLoading: funnelLoading } = useAdminFunnelStats(30);

  const pct = (v: number) => `${(v * 100).toFixed(1)}%`;

  const upstreamRows = useMemo(() => {
    if (!status?.models) return [];
    const rows: { model: string; provider: string; baseUrl: string; state: string; failures: number }[] = [];
    for (const [modelName, info] of Object.entries(status.models)) {
      for (const u of info.upstreams) {
        rows.push({
          model: modelName,
          provider: u.provider,
          baseUrl: u.base_url,
          state: u.state,
          failures: u.failure_count,
        });
      }
    }
    return rows;
  }, [status]);

  const modelCount = status?.models ? Object.keys(status.models).length : 0;
  const upstreamCount = upstreamRows.length;
  const normalCount = upstreamRows.filter((m) => m.state.toLowerCase() === "closed" || m.state.toLowerCase() === "normal").length;
  const errorCount = upstreamRows.filter((m) => m.state.toLowerCase() !== "closed" && m.state.toLowerCase() !== "normal").length;

  const handleRefresh = () => {
    qc.invalidateQueries({ queryKey: queryKeys.status() });
  };

  const isLoading = dashLoading || statusLoading || consumptionLoading;

  type StatCard = {
    label: string;
    value: string;
    sub: string;
    subColor: string;
    icon: LucideIcon;
    iconBg: string;
    iconColor: string;
    clickable?: boolean;
    valueColor?: string;
  };

  const statCards: StatCard[] = [
    {
      label: "注册用户",
      value: dashLoading ? "..." : String(dashboard?.total_users ?? 0),
      sub: "",
      subColor: "text-muted-foreground",
      icon: Users,
      iconBg: "bg-primary/10",
      iconColor: "text-primary",
    },
    {
      label: "今日收入",
      value: dashLoading ? "..." : `¥${(dashboard?.today_revenue ?? 0).toFixed(2)}`,
      sub: "",
      subColor: "text-muted-foreground",
      icon: DollarSign,
      iconBg: "bg-emerald-500/10 dark:bg-emerald-500/15",
      iconColor: "text-emerald-500",
    },
    {
      label: "今日请求",
      value: dashLoading ? "..." : (dashboard?.today_requests ? String(dashboard.today_requests) : "\u2014"),
      sub: "",
      subColor: "text-muted-foreground",
      icon: Zap,
      iconBg: "bg-violet-500/10 dark:bg-violet-500/15",
      iconColor: "text-violet-500",
    },
    {
      label: "30天总消耗",
      value: consumptionLoading ? "..." : `¥${(consumptionData?.total_cost ?? 0).toFixed(2)}`,
      sub: consumptionLoading ? "" : `${consumptionData?.total_requests ?? 0} 次请求`,
      subColor: "text-muted-foreground",
      icon: Coins,
      iconBg: "bg-orange-500/10 dark:bg-orange-500/15",
      iconColor: "text-orange-500",
      clickable: true,
    },
    {
      label: "网关状态",
      value: statusLoading ? "..." : (errorCount > 0 ? "有异常" : "运行中"),
      valueColor: statusLoading ? undefined : (errorCount > 0 ? "text-destructive" : "text-success"),
      sub: statusLoading ? "" : `${modelCount} 模型 / ${upstreamCount} 上游`,
      subColor: "text-muted-foreground",
      icon: Activity,
      iconBg: "bg-success/10",
      iconColor: "text-success",
    },
  ];

  const funnelStages = funnelData?.stages ?? [];
  const funnelMax = funnelStages.length > 0 ? Math.max(funnelStages[0].count, 1) : 1;
  const funnelRates: { label: string; value: number }[] = funnelData
    ? [
        { label: "注册→首充", value: funnelData.first_recharge_rate },
        { label: "首充→复充", value: funnelData.repeat_recharge_rate },
        { label: "首充→有消费", value: funnelData.post_recharge_use_rate },
      ]
    : [];

  return (
    <div className="page-container">
      {/* Page header */}
      <div className="mb-6">
        <h1 className="text-lg font-bold text-foreground">运营概览</h1>
        <p className="text-sm text-muted-foreground mt-0.5">实时监控平台运营指标和上游状态</p>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-1 gap-4 mb-6 sm:grid-cols-2 xl:grid-cols-4">
        {statCards.map((card, i) => {
          const Icon = card.icon;
          return (
            <div
              key={card.label}
              onClick={() => card.clickable && onNavigate?.("consumption")}
              className={`stagger-item flex-1 bg-card border border-border rounded-xl p-5 shadow-card hover:shadow-elevated hover:-translate-y-0.5 transition-all duration-200 ${card.clickable ? "cursor-pointer" : ""}`}
              style={{ animationDelay: `${i * 80}ms` }}
            >
              <div className="flex items-start justify-between mb-3">
                <div className={`w-9 h-9 ${card.iconBg} rounded-lg flex items-center justify-center`}>
                  <Icon size={17} className={card.iconColor} />
                </div>
                <span className="text-xs text-muted-foreground">{card.label}</span>
              </div>
              <div className={`text-2xl font-bold mb-1.5 ${card.valueColor || "text-foreground"}`}>{card.value}</div>
              {card.sub && <div className={`text-xs ${card.subColor}`}>{card.sub}</div>}
            </div>
          );
        })}
      </div>

      {/* Conversion funnel: 注册 → 首充 → 复充 → 首充后有消费（近 30 天注册队列） */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden mb-6">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div>
            <div className="text-sm font-semibold text-foreground">转化漏斗</div>
            <div className="text-xs text-muted-foreground mt-0.5">近 30 天注册用户队列 · 判断卡在拉新还是转化</div>
          </div>
        </div>
        {funnelLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="p-5">
            <div className="flex items-stretch gap-2">
              {(funnelData?.stages ?? []).map((stage, i) => {
                const top = funnelData?.stages?.[0]?.count ?? 0;
                const width = top > 0 ? Math.max(8, (stage.count / top) * 100) : 8;
                // 相对上一层的转化率
                const prev = i > 0 ? funnelData?.stages?.[i - 1]?.count ?? 0 : 0;
                const stepRate = i > 0 && prev > 0 ? stage.count / prev : null;
                return (
                  <div key={stage.key} className="flex-1 flex items-center gap-2">
                    <div className="flex-1">
                      <div className="flex items-baseline justify-between mb-1.5">
                        <span className="text-xs text-muted-foreground">{stage.label}</span>
                        <span className="text-lg font-bold text-foreground">{stage.count}</span>
                      </div>
                      <div className="h-2 bg-muted rounded-full overflow-hidden">
                        <div
                          className="h-full bg-gradient-to-r from-indigo-500 to-purple-500 rounded-full transition-all"
                          style={{ width: `${width}%` }}
                        />
                      </div>
                    </div>
                    {i < (funnelData?.stages?.length ?? 0) - 1 && (
                      <div className="flex flex-col items-center justify-center px-1 self-center">
                        <ChevronRight size={14} className="text-muted-foreground/50" />
                        {stepRate !== null && (
                          <span className="text-[10px] text-muted-foreground whitespace-nowrap">{pct(stepRate)}</span>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
            <div className="flex gap-6 mt-4 pt-4 border-t border-border text-xs">
              <div>
                <span className="text-muted-foreground">注册→首充　</span>
                <span className="font-semibold text-foreground">{pct(funnelData?.first_recharge_rate ?? 0)}</span>
              </div>
              <div>
                <span className="text-muted-foreground">首充→复充　</span>
                <span className="font-semibold text-foreground">{pct(funnelData?.repeat_recharge_rate ?? 0)}</span>
              </div>
              <div>
                <span className="text-muted-foreground">首充→有消费　</span>
                <span className="font-semibold text-foreground">{pct(funnelData?.post_recharge_use_rate ?? 0)}</span>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Conversion funnel */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden mb-6">
        <div className="px-5 py-4 border-b border-border flex items-center gap-2">
          <div className="w-9 h-9 bg-sky-500/10 dark:bg-sky-500/15 rounded-lg flex items-center justify-center">
            <TrendingDown size={17} className="text-sky-500" />
          </div>
          <div>
            <div className="text-sm font-semibold text-foreground">转化漏斗（近 30 天注册用户）</div>
            <div className="text-xs text-muted-foreground mt-0.5">
              定位问题：漏在获客（注册少）还是转化（注册多但首充少）
            </div>
          </div>
        </div>
        {funnelLoading ? (
          <div className="flex items-center justify-center py-16">
            <Loader size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : funnelStages.length === 0 ? (
          <div className="py-12 text-center text-sm text-muted-foreground">暂无数据</div>
        ) : (
          <div className="p-5">
            <div className="space-y-3">
              {funnelStages.map((stage) => {
                const pct = Math.round((stage.count / funnelMax) * 100);
                return (
                  <div key={stage.key} className="flex items-center gap-3">
                    <div className="w-24 text-sm text-muted-foreground shrink-0">{stage.label}</div>
                    <div className="flex-1 bg-muted/40 rounded-lg overflow-hidden h-8 relative">
                      <div
                        className="h-full bg-gradient-to-r from-sky-500 to-indigo-500 rounded-lg transition-all duration-500 flex items-center justify-end pr-2"
                        style={{ width: `${Math.max(pct, 4)}%` }}
                      >
                        <span className="text-xs font-medium text-white">{stage.count}</span>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
            <div className="grid grid-cols-1 gap-4 mt-5 pt-4 border-t border-border sm:grid-cols-3">
              {funnelRates.map((r) => (
                <div key={r.label} className="flex-1 text-center">
                  <div className="text-xl font-bold text-foreground">{(r.value * 100).toFixed(1)}%</div>
                  <div className="text-xs text-muted-foreground mt-0.5">{r.label}</div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Body: upstream + orders */}
      <div className="grid grid-cols-1 gap-5 mb-6 lg:grid-cols-5">
        {/* Upstream health */}
        <div className="lg:col-span-3 bg-card border border-border rounded-xl shadow-card overflow-hidden">
          <div className="px-5 py-4 border-b border-border flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="text-sm font-semibold text-foreground">上游健康状态</div>
              {!statusLoading && (
                <div className="flex items-center gap-2 ml-2">
                  <span className="badge-success">
                    <span className="w-1.5 h-1.5 rounded-full bg-success inline-block mr-1" />
                    正常 {normalCount}
                  </span>
                  {errorCount > 0 && (
                    <span className="badge-danger">
                      <AlertCircle size={10} />
                      异常 {errorCount}
                    </span>
                  )}
                </div>
              )}
            </div>
            <button
              onClick={handleRefresh}
              className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground px-2.5 py-1.5 rounded-lg hover:bg-muted transition-colors"
            >
              <RefreshCw size={12} className={isFetching ? "animate-spin" : ""} />
              刷新
            </button>
          </div>

          <div className="overflow-auto" style={{ maxHeight: 420 }}>
            {statusLoading ? (
              <div className="flex items-center justify-center py-16">
                <Loader size={20} className="animate-spin text-muted-foreground" />
              </div>
            ) : (
              <Table className="w-full">
                <TableHeader className="sticky top-0 z-10">
                  <TableRow className="table-header">
                    <TableHead className="text-left px-5 py-3 font-semibold">模型</TableHead>
                    <TableHead className="text-left px-5 py-3 font-semibold">供应商</TableHead>
                    <TableHead className="text-left px-5 py-3 font-semibold">熔断状态</TableHead>
                    <TableHead className="text-left px-5 py-3 font-semibold">失败次数</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {upstreamRows.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="px-5 py-12 text-center text-sm text-muted-foreground">
                        暂无上游数据
                      </TableCell>
                    </TableRow>
                  ) : (
                    upstreamRows.map((m, i) => {
                      const isNormal = m.state.toLowerCase() === "closed" || m.state.toLowerCase() === "normal";
                      return (
                        <TableRow
                          key={`${m.model}-${m.provider}-${i}`}
                          className={`border-t border-border hover:bg-muted/40 transition-colors ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                        >
                          <TableCell className="px-5 py-3 text-sm font-mono text-foreground">{m.model}</TableCell>
                          <TableCell className="px-5 py-3 text-sm text-muted-foreground">{m.provider}</TableCell>
                          <TableCell className="px-5 py-3">
                            <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${isNormal ? "text-success" : "text-destructive"}`}>
                              <span className={`w-1.5 h-1.5 rounded-full ${isNormal ? "bg-success" : "bg-destructive"}`} />
                              {isNormal ? "正常" : "熔断"}
                            </span>
                          </TableCell>
                          <TableCell className="px-5 py-3 text-sm text-muted-foreground">{m.failures}</TableCell>
                        </TableRow>
                      );
                    })
                  )}
                </TableBody>
              </Table>
            )}
          </div>
        </div>

        {/* Recent recharge orders */}
        <div className="lg:col-span-2 bg-card border border-border rounded-xl shadow-card overflow-hidden flex flex-col">
          <div className="px-5 py-4 border-b border-border flex items-center justify-between">
            <div className="text-sm font-semibold text-foreground">最近充值订单</div>
            <button
              onClick={() => onNavigate?.("orders")}
              className="text-xs text-primary hover:text-primary/80 flex items-center gap-0.5 transition-colors"
            >
              查看全部 <ChevronRight size={11} />
            </button>
          </div>
          {ordersLoading ? (
            <div className="flex-1 flex items-center justify-center py-16">
              <Loader size={20} className="animate-spin text-muted-foreground" />
            </div>
          ) : !ordersData?.orders?.length ? (
            <div className="flex-1 flex items-center justify-center py-16">
              <div className="text-center">
                <div className="w-12 h-12 bg-muted rounded-xl flex items-center justify-center mx-auto mb-3">
                  <DollarSign size={18} className="text-muted-foreground/50" />
                </div>
                <div className="text-sm text-muted-foreground">暂无订单记录</div>
                <div className="text-xs text-muted-foreground/60 mt-1">用户充值订单将在这里显示</div>
              </div>
            </div>
          ) : (
            <div className="overflow-auto flex-1" style={{ maxHeight: 420 }}>
              {ordersData.orders.map((order) => (
                <div key={order.id} className="px-5 py-3 border-b border-border last:border-0 hover:bg-muted/40 transition-colors">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-sm font-medium text-foreground">¥{order.amount.toFixed(2)}</span>
                    <span className={`text-xs px-2 py-0.5 rounded-full ${
                      order.status === "paid" ? "bg-emerald-500/10 text-emerald-600 dark:bg-emerald-500/15 dark:text-emerald-400" :
                      order.status === "pending" ? "bg-amber-500/10 text-amber-600 dark:bg-amber-500/15 dark:text-amber-400" :
                      "bg-muted text-muted-foreground"
                    }`}>
                      {order.status === "paid" ? "已支付" : order.status === "pending" ? "待支付" : "已过期"}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">{order.user_identifier || "—"}</span>
                    <span className="text-xs text-muted-foreground">
                      {new Date(order.created_at).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default AdminOverview;
