import { useState } from "react";
import { Undo2, Loader2, ChevronLeft, ChevronRight } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api-client";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 20;

interface Refund {
  id: string;
  order_no: string;
  user_id: string;
  user_identifier: string;
  amount: number;
  reason: string;
  status: "pending" | "success" | "failed";
  out_request_no: string;
  alipay_trade_no: string | null;
  operator_id: string | null;
  error_message: string | null;
  created_at: string;
  updated_at: string;
}

interface AdminRefundsResponse {
  refunds: Refund[];
  total: number;
  page: number;
  size: number;
}

const statusLabel = (s: string) => {
  switch (s) {
    case "success": return "成功";
    case "pending": return "处理中";
    case "failed": return "失败";
    default: return s;
  }
};

const statusBadge = (s: string) => {
  switch (s) {
    case "success":
      return "bg-emerald-500/10 text-emerald-600 border-emerald-500/20 dark:text-emerald-400";
    case "pending":
      return "bg-amber-500/10 text-amber-600 border-amber-500/20 dark:text-amber-400";
    case "failed":
      return "bg-destructive/10 text-destructive border-destructive/20";
    default:
      return "bg-muted text-muted-foreground border-border";
  }
};

const AdminRefunds = () => {
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ["admin", "refunds", { page, size: PAGE_SIZE }],
    queryFn: () =>
      apiGet<AdminRefundsResponse>(`/api/admin/refunds?page=${page}&size=${PAGE_SIZE}`),
  });

  const refunds = data?.refunds ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="page-container">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-lg font-bold text-foreground">退款管理</h1>
        <p className="text-sm text-muted-foreground mt-0.5">查看所有订单的退款记录</p>
      </div>

      {/* Table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-foreground">退款列表</span>
            <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
              {isLoading ? "..." : `${total} 条`}
            </span>
          </div>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3.5">时间</TableHead>
                <TableHead className="text-left px-5 py-3.5">订单号</TableHead>
                <TableHead className="text-left px-5 py-3.5">用户手机</TableHead>
                <TableHead className="text-right px-5 py-3.5">金额</TableHead>
                <TableHead className="text-left px-5 py-3.5">理由</TableHead>
                <TableHead className="text-left px-5 py-3.5">状态</TableHead>
                <TableHead className="text-left px-5 py-3.5">支付宝流水号</TableHead>
                <TableHead className="text-left px-5 py-3.5">失败原因</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {refunds.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="px-5 py-16">
                    <div className="empty-state">
                      <div className="w-12 h-12 bg-muted rounded-xl flex items-center justify-center mb-3">
                        <Undo2 size={18} className="text-muted-foreground/50" />
                      </div>
                      <div className="text-sm text-muted-foreground">暂无退款记录</div>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                refunds.map((r, i) => (
                  <TableRow
                    key={r.id}
                    className={`border-t border-border hover:bg-accent/30 transition-colors duration-150 ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                  >
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                      {new Date(r.created_at).toLocaleString("zh-CN")}
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm font-mono text-foreground">{r.order_no}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{r.user_identifier || "—"}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground text-right">
                      ¥{r.amount.toFixed(2)}
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground max-w-[200px] truncate" title={r.reason}>
                      {r.reason || "—"}
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium border ${statusBadge(r.status)}`}>
                        {statusLabel(r.status)}
                      </span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm font-mono text-muted-foreground">
                      {r.alipay_trade_no || "—"}
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-destructive max-w-[200px] truncate" title={r.error_message ?? undefined}>
                      {r.error_message || "—"}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        )}

        {/* Pagination */}
        <div className="px-5 py-3.5 border-t border-border flex items-center justify-between">
          <div className="text-xs text-muted-foreground">共 {total} 条记录</div>
          <div className="flex items-center gap-1">
            <button
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
              className={`w-7 h-7 rounded-lg flex items-center justify-center transition-colors ${page <= 1 ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"}`}
            >
              <ChevronLeft size={13} />
            </button>
            <span className="text-xs text-foreground px-2">
              <span className="font-medium">{page}</span>
              <span className="text-muted-foreground"> / {totalPages}</span>
            </span>
            <button
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
              className={`w-7 h-7 rounded-lg flex items-center justify-center transition-colors ${page >= totalPages ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"}`}
            >
              <ChevronRight size={13} />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AdminRefunds;
