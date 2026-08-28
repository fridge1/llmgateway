import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeftRight, Flame, CalendarDays, BarChart2 } from "lucide-react";
import { apiGet } from "@/lib/api-client";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
interface Transaction {
  id: string;
  user_id: string;
  type: string;
  amount: number;
  balance_after: number;
  model: string;
  request_id: string;
  description: string;
  prompt_tokens: number;
  completion_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  cache_creation_5m_tokens: number;
  cache_creation_1h_tokens: number;
  created_at: string;
}

interface TransactionsResponse {
  transactions: Transaction[];
  total: number;
  limit: number;
  offset: number;
}

interface BillingStats {
  today_cost: number;
  month_cost: number;
}

const SubUserTransactionsPage = () => {
  const [page, setPage] = useState(0);
  const pageSize = 20;

  const { data, isLoading } = useQuery({
    queryKey: ["sub-user-transactions", page],
    queryFn: () => apiGet<TransactionsResponse>(`/api/sub-user/transactions?limit=${pageSize}&offset=${page * pageSize}`),
  });

  const { data: stats } = useQuery({
    queryKey: ["sub-user-stats"],
    queryFn: () => apiGet<BillingStats>("/api/sub-user/stats?days=30"),
  });

  const transactions = data?.transactions ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / pageSize);

  const todayCost = stats?.today_cost ?? 0;
  const monthCost = stats?.month_cost ?? 0;

  const statCards = [
    { label: "今日消费", value: `¥${todayCost.toFixed(4)}`, icon: Flame, iconBg: "bg-orange-50 dark:bg-orange-500/10", iconColor: "text-orange-500" },
    { label: "本月消费", value: `¥${monthCost.toFixed(4)}`, icon: CalendarDays, iconBg: "bg-emerald-50 dark:bg-emerald-500/10", iconColor: "text-emerald-500" },
    { label: "总记录", value: String(total), icon: BarChart2, iconBg: "bg-blue-50 dark:bg-blue-500/10", iconColor: "text-blue-500" },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold text-foreground mb-6">使用记录</h1>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        {statCards.map((stat) => {
          const Icon = stat.icon;
          return (
            <div key={stat.label} className="bg-card border border-border/60 rounded-xl p-4 shadow-card">
              <div className="flex items-start justify-between mb-3">
                <div className={`w-9 h-9 ${stat.iconBg} rounded-lg flex items-center justify-center`}>
                  <Icon size={18} className={stat.iconColor} />
                </div>
              </div>
              <div className="text-xl font-bold mb-1 text-foreground">{stat.value}</div>
              <div className="text-xs text-muted-foreground">{stat.label}</div>
            </div>
          );
        })}
      </div>

      {isLoading ? (
        <div className="flex justify-center py-12">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      ) : transactions.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground">
          <ArrowLeftRight size={48} className="mx-auto mb-4 opacity-30" />
          <p className="text-lg">暂无使用记录</p>
          <p className="text-sm mt-1">使用 API 密钥调用模型后会自动生成记录</p>
        </div>
      ) : (
        <>
          <div className="bg-card border border-border/60 rounded-xl overflow-hidden">
            <Table className="w-full">
              <TableHeader>
                <TableRow className="border-b border-border/60">
                  <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">时间</TableHead>
                  <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">类型</TableHead>
                  <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">模型</TableHead>
                  <TableHead className="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">输入</TableHead>
                  <TableHead className="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">输出</TableHead>
                  <TableHead className="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">缓存命中</TableHead>
                  <TableHead className="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">缓存写入</TableHead>
                  <TableHead className="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">金额</TableHead>
                  <TableHead className="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">余额</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {transactions.map((tx) => (
                  <TableRow key={tx.id} className="border-b border-border/40 last:border-none hover:bg-muted/30 transition-colors">
                    <TableCell className="px-4 py-3 text-sm text-muted-foreground">
                      {new Date(tx.created_at).toLocaleString("zh-CN")}
                    </TableCell>
                    <TableCell className="px-4 py-3">
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-700 border border-red-200 dark:bg-red-500/10 dark:text-red-400 dark:border-red-500/30">
                        余额消费
                      </span>
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm font-mono text-foreground">{tx.model || "—"}</TableCell>
                    <TableCell className="px-4 py-3 text-sm text-muted-foreground text-right font-mono">
                      {tx.prompt_tokens != null ? tx.prompt_tokens.toLocaleString() : "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm text-muted-foreground text-right font-mono">
                      {tx.completion_tokens != null ? tx.completion_tokens.toLocaleString() : "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm text-muted-foreground text-right font-mono">
                      {tx.cache_read_tokens != null ? tx.cache_read_tokens.toLocaleString() : "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm text-muted-foreground text-right font-mono">
                      {tx.cache_creation_5m_tokens || tx.cache_creation_1h_tokens
                        ? `5m: ${(tx.cache_creation_5m_tokens ?? 0).toLocaleString()} / 1h: ${(tx.cache_creation_1h_tokens ?? 0).toLocaleString()}`
                        : tx.cache_creation_tokens != null
                          ? tx.cache_creation_tokens.toLocaleString()
                          : "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm font-medium text-right text-destructive">
                      -¥{Math.abs(tx.amount).toFixed(4)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm text-foreground text-right font-medium">
                      ¥{tx.balance_after.toFixed(4)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4">
              <p className="text-sm text-muted-foreground">共 {total} 条记录</p>
              <div className="flex gap-2">
                <button
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                  disabled={page === 0}
                  className="px-3 py-1.5 text-sm border border-border rounded-lg disabled:opacity-40 hover:bg-muted/60 transition-colors"
                >
                  上一页
                </button>
                <span className="px-3 py-1.5 text-sm text-muted-foreground">
                  {page + 1} / {totalPages}
                </span>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                  disabled={page >= totalPages - 1}
                  className="px-3 py-1.5 text-sm border border-border rounded-lg disabled:opacity-40 hover:bg-muted/60 transition-colors"
                >
                  下一页
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default SubUserTransactionsPage;
