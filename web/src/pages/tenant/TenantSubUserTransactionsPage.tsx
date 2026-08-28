import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Receipt, ChevronRight, Loader2, Wallet, BarChart3,
} from "lucide-react";
import {
  useTenantDetail,
  useTenantSubUsers,
  useTenantSubUserTransactions,
  useTenantSubUserModelStats,
} from "@/hooks/use-tenant";
import { TenantPageHeader } from "@/components/tenant/TenantPageHeader";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 20;

const txTypeLabel: Record<string, string> = {
  consumption: "消费",
  recharge: "充值",
};

const txTypeBadge: Record<string, string> = {
  consumption: "badge-danger",
  recharge: "badge-success",
};

type TabType = "transactions" | "model-stats";

const TenantSubUserTransactionsPage = () => {
  const { id, subUserId } = useParams<{ id: string; subUserId: string }>();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [activeTab, setActiveTab] = useState<TabType>("transactions");
  const [dateRange, setDateRange] = useState<{ start: string; end: string }>({
    start: "",
    end: "",
  });

  const { data: tenant } = useTenantDetail(id!);
  const { data: subUsers } = useTenantSubUsers(id!);
  const { data: txData, isLoading } = useTenantSubUserTransactions(id!, subUserId!, page, PAGE_SIZE);
  const { data: modelStats, isLoading: isLoadingStats } = useTenantSubUserModelStats(
    id!,
    subUserId!,
    dateRange.start || undefined,
    dateRange.end || undefined
  );

  const subUser = subUsers?.find((su) => su.id === subUserId);
  const transactions = txData?.transactions ?? [];
  const total = txData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  if (isLoading && !txData) {
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
      <TenantPageHeader
        title="使用记录"
        description={`成员：${subUser?.nickname || subUser?.username || subUserId}`}
        tenantName={tenant?.name}
        icon={Receipt}
        onBack={() => navigate(`/dashboard/tenants/${id}/members`)}
      />

      {/* Quota Info */}
      {subUser && (
        <div className="bg-card border border-border rounded-xl p-5 shadow-card mb-6">
          <div className="flex items-center gap-2 mb-4">
            <div className="w-7 h-7 rounded-lg bg-primary/10 flex items-center justify-center">
              <Wallet size={14} className="text-primary" />
            </div>
            <h3 className="text-sm font-semibold text-foreground">额度概览</h3>
          </div>
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
            <div className="stagger-item" style={{ animationDelay: "0ms" }}>
              <p className="text-xs text-muted-foreground mb-1">已使用</p>
              <p className="text-2xl font-bold text-destructive">¥{subUser.quota_used.toFixed(2)}</p>
            </div>
            <div className="stagger-item" style={{ animationDelay: "60ms" }}>
              <p className="text-xs text-muted-foreground mb-1">额度上限</p>
              <p className="text-2xl font-bold text-foreground">
                {subUser.quota_limit !== null ? `¥${subUser.quota_limit.toFixed(2)}` : "不限"}
              </p>
            </div>
            {subUser.quota_remaining !== null && (
              <div className="stagger-item" style={{ animationDelay: "120ms" }}>
                <p className="text-xs text-muted-foreground mb-1">剩余额度</p>
                <p className="text-2xl font-bold text-green-600">¥{subUser.quota_remaining.toFixed(2)}</p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-2 mb-6 border-b border-border">
        <button
          onClick={() => setActiveTab("transactions")}
          className={`px-4 py-2 text-sm font-medium transition-colors relative ${
            activeTab === "transactions"
              ? "text-primary"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <div className="flex items-center gap-2">
            <Receipt size={16} />
            <span>消费记录</span>
          </div>
          {activeTab === "transactions" && (
            <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
          )}
        </button>
        <button
          onClick={() => setActiveTab("model-stats")}
          className={`px-4 py-2 text-sm font-medium transition-colors relative ${
            activeTab === "model-stats"
              ? "text-primary"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <div className="flex items-center gap-2">
            <BarChart3 size={16} />
            <span>按模型统计</span>
          </div>
          {activeTab === "model-stats" && (
            <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
          )}
        </button>
      </div>

      {/* Transactions Tab */}
      {activeTab === "transactions" && (
      <div className="data-table-card">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div>
            <div className="text-sm font-semibold text-foreground">消费记录</div>
            <div className="text-xs text-muted-foreground mt-0.5">共 {total} 条记录</div>
          </div>
        </div>
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 size={16} className="animate-spin mr-2 text-primary" />
            <span className="text-sm text-muted-foreground">加载中...</span>
          </div>
        ) : transactions.length === 0 ? (
          <div className="empty-state">
            <div className="w-12 h-12 bg-muted rounded-full flex items-center justify-center mb-3">
              <Receipt size={20} className="text-muted-foreground" />
            </div>
            <div className="text-sm font-medium text-muted-foreground">暂无使用记录</div>
          </div>
        ) : (
          <>
            <Table className="w-full text-sm">
                <TableHeader>
                  <TableRow className="table-header">
                    <TableHead className="text-left px-5 py-3">类型</TableHead>
                    <TableHead className="text-left px-5 py-3">金额</TableHead>
                    <TableHead className="text-left px-5 py-3">模型</TableHead>
                    <TableHead className="text-left px-5 py-3">描述</TableHead>
                    <TableHead className="text-left px-5 py-3">时间</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {transactions.map((tx) => (
                    <TableRow key={tx.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                      <TableCell className="px-5 py-3.5">
                        <span className={txTypeBadge[tx.type] ?? "badge-neutral"}>
                          {txTypeLabel[tx.type] ?? tx.type}
                        </span>
                      </TableCell>
                      <TableCell className={`px-5 py-3.5 font-mono text-sm font-medium ${tx.amount > 0 ? "text-success" : "text-destructive"}`}>
                        {tx.amount > 0 ? "-" : ""}{tx.amount.toFixed(4)}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-muted-foreground">{tx.model || "-"}</TableCell>
                      <TableCell className="px-5 py-3.5 text-muted-foreground">{tx.description || "-"}</TableCell>
                      <TableCell className="px-5 py-3.5 text-muted-foreground">
                        {new Date(tx.created_at).toLocaleString("zh-CN")}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            {totalPages > 1 && (
              <div className="px-5 py-3.5 border-t border-border flex items-center justify-between">
                <div className="text-xs text-muted-foreground">共 {total} 条记录</div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                    className="flex h-9 w-9 items-center justify-center rounded-lg text-sm transition-colors hover:bg-muted/60 disabled:text-muted-foreground disabled:cursor-not-allowed"
                  >
                    <ChevronLeft size={14} />
                  </button>
                  <span className="text-sm text-foreground px-2">
                    <span className="font-medium">{page}</span>
                    <span className="text-muted-foreground"> / {totalPages}</span>
                  </span>
                  <button
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page === totalPages}
                    className="flex h-9 w-9 items-center justify-center rounded-lg text-sm transition-colors hover:bg-muted/60 disabled:text-muted-foreground disabled:cursor-not-allowed"
                  >
                    <ChevronRight size={14} />
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
      )}

      {/* Model Stats Tab */}
      {activeTab === "model-stats" && (
        <div className="data-table-card">
          <div className="px-5 py-4 border-b border-border">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-semibold text-foreground">按模型统计</div>
                <div className="text-xs text-muted-foreground mt-0.5">
                  {modelStats?.period
                    ? `${modelStats.period.start_date} 至 ${modelStats.period.end_date}`
                    : "全部历史数据"}
                </div>
              </div>
              <div className="flex gap-2">
                <input
                  type="date"
                  value={dateRange.start}
                  onChange={(e) => setDateRange({ ...dateRange, start: e.target.value })}
                  className="px-3 py-1.5 text-xs border border-border rounded-lg bg-background"
                  placeholder="开始日期"
                />
                <input
                  type="date"
                  value={dateRange.end}
                  onChange={(e) => setDateRange({ ...dateRange, end: e.target.value })}
                  className="px-3 py-1.5 text-xs border border-border rounded-lg bg-background"
                  placeholder="结束日期"
                />
                {(dateRange.start || dateRange.end) && (
                  <button
                    onClick={() => setDateRange({ start: "", end: "" })}
                    className="px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
                  >
                    清除
                  </button>
                )}
              </div>
            </div>
          </div>

          {isLoadingStats ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 size={16} className="animate-spin mr-2 text-primary" />
              <span className="text-sm text-muted-foreground">加载中...</span>
            </div>
          ) : !modelStats || modelStats.model_breakdown.length === 0 ? (
            <div className="empty-state">
              <div className="w-12 h-12 bg-muted rounded-full flex items-center justify-center mb-3">
                <BarChart3 size={20} className="text-muted-foreground" />
              </div>
              <div className="text-sm font-medium text-muted-foreground">暂无统计数据</div>
            </div>
          ) : (
            <>
              {/* Summary */}
              <div className="px-5 py-4 bg-muted/30 border-b border-border">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <div>
                    <p className="text-xs text-muted-foreground mb-1">总消费</p>
                    <p className="text-lg font-bold text-destructive">¥{modelStats.total_cost.toFixed(4)}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground mb-1">总请求数</p>
                    <p className="text-lg font-bold text-foreground">{modelStats.total_requests.toLocaleString()}</p>
                  </div>
                </div>
              </div>

              {/* Model Breakdown Table */}
              <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="table-header">
                      <TableHead className="text-left px-5 py-3">模型</TableHead>
                      <TableHead className="text-right px-5 py-3">费用</TableHead>
                      <TableHead className="text-right px-5 py-3">请求数</TableHead>
                      <TableHead className="text-right px-5 py-3">输入 Token</TableHead>
                      <TableHead className="text-right px-5 py-3">输出 Token</TableHead>
                      <TableHead className="text-right px-5 py-3">缓存命中</TableHead>
                      <TableHead className="text-right px-5 py-3">缓存写入</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {modelStats.model_breakdown.map((model, idx) => (
                      <TableRow key={idx} className="border-t border-border hover:bg-muted/30 transition-colors">
                        <TableCell className="px-5 py-3.5 font-medium text-foreground">{model.model}</TableCell>
                        <TableCell className="px-5 py-3.5 text-right font-mono text-sm font-medium text-destructive">
                          ¥{model.cost.toFixed(4)}
                        </TableCell>
                        <TableCell className="px-5 py-3.5 text-right text-muted-foreground">
                          {model.request_count.toLocaleString()}
                        </TableCell>
                        <TableCell className="px-5 py-3.5 text-right text-muted-foreground">
                          {model.prompt_tokens.toLocaleString()}
                        </TableCell>
                        <TableCell className="px-5 py-3.5 text-right text-muted-foreground">
                          {model.completion_tokens.toLocaleString()}
                        </TableCell>
                        <TableCell className="px-5 py-3.5 text-right text-muted-foreground">
                          {model.cache_read_tokens.toLocaleString()}
                        </TableCell>
                        <TableCell className="px-5 py-3.5 text-right text-muted-foreground">
                          {model.cache_creation_tokens.toLocaleString()}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
            </>
          )}
        </div>
      )}
    </div>
  );
};

export default TenantSubUserTransactionsPage;
