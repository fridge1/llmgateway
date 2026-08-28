import { useState } from "react";
import { DollarSign, ChevronLeft, ChevronRight, Loader, X, Undo2 } from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useAdminOrders } from "@/hooks/use-api";
import { apiPost, ApiError } from "@/lib/api-client";
import type { AdminOrder } from "@/types/api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
interface RefundResponse {
  status: string;
  refund_id: string;
  trade_no: string;
}

const PAGE_SIZE = 20;

const STATUS_TABS = [
  { key: "", label: "全部" },
  { key: "paid", label: "已支付" },
  { key: "pending", label: "待支付" },
  { key: "expired", label: "已过期" },
] as const;

const statusLabel = (s: string) => {
  switch (s) {
    case "paid": return "已支付";
    case "pending": return "待支付";
    case "expired": return "已过期";
    default: return s;
  }
};

const statusBadge = (s: string) => {
  switch (s) {
    case "paid": return "bg-emerald-500/10 text-emerald-600 border-emerald-500/20 dark:text-emerald-400";
    case "pending": return "bg-amber-500/10 text-amber-600 border-amber-500/20 dark:text-amber-400";
    case "expired": return "bg-muted text-muted-foreground border-border";
    default: return "bg-muted text-muted-foreground border-border";
  }
};

const AdminOrders = () => {
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("");
  const [refundOrder, setRefundOrder] = useState<AdminOrder | null>(null);
  const [refundAmount, setRefundAmount] = useState("");
  const [refundReason, setRefundReason] = useState("");
  const { data, isLoading } = useAdminOrders(page, PAGE_SIZE, statusFilter);
  const queryClient = useQueryClient();

  const refundMutation = useMutation({
    mutationFn: ({ orderNo, amount, reason }: { orderNo: string; amount: number; reason: string }) =>
      apiPost<RefundResponse>(`/api/admin/orders/${encodeURIComponent(orderNo)}/refund`, { amount, reason }),
    onSuccess: () => {
      toast.success("退款已提交");
      queryClient.invalidateQueries({ queryKey: ["admin", "orders"] });
      queryClient.invalidateQueries({ queryKey: ["admin", "refunds"] });
      closeRefundModal();
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "退款失败，请稍后重试");
    },
  });

  const orders = data?.orders ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const openRefundModal = (order: AdminOrder) => {
    setRefundOrder(order);
    setRefundAmount(order.amount.toFixed(2));
    setRefundReason("");
  };

  const closeRefundModal = () => {
    setRefundOrder(null);
    setRefundAmount("");
    setRefundReason("");
  };

  const parsedAmount = parseFloat(refundAmount);
  const refundValid =
    refundOrder !== null &&
    !Number.isNaN(parsedAmount) &&
    parsedAmount > 0 &&
    parsedAmount <= (refundOrder?.amount ?? 0) &&
    refundReason.trim().length > 0;

  const handleRefund = () => {
    if (!refundOrder || !refundValid || refundMutation.isPending) return;
    refundMutation.mutate({
      orderNo: refundOrder.order_no,
      amount: parsedAmount,
      reason: refundReason.trim(),
    });
  };

  return (
    <div className="page-container">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-lg font-bold text-foreground">订单管理</h1>
        <p className="text-sm text-muted-foreground mt-0.5">查看所有用户的充值订单</p>
      </div>

      {/* Status filter */}
      <div className="flex gap-2 mb-5">
        {STATUS_TABS.map((tab) => {
          const isActive = statusFilter === tab.key;
          return (
            <button
              key={tab.key}
              onClick={() => { setStatusFilter(tab.key); setPage(1); }}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                isActive
                  ? "brand-gradient text-white shadow-button"
                  : "bg-muted text-muted-foreground hover:text-foreground"
              }`}
            >
              {tab.label}
            </button>
          );
        })}
      </div>

      {/* Table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-foreground">订单列表</span>
            <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
              {isLoading ? "..." : `${total} 条`}
            </span>
          </div>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : orders.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20">
            <div className="w-12 h-12 bg-muted rounded-xl flex items-center justify-center mb-3">
              <DollarSign size={18} className="text-muted-foreground/50" />
            </div>
            <div className="text-sm text-muted-foreground">暂无订单</div>
          </div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3 font-semibold">订单号</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">用户</TableHead>
                <TableHead className="text-right px-5 py-3 font-semibold">金额</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">支付方式</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">状态</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">创建时间</TableHead>
                <TableHead className="text-left px-5 py-3 font-semibold">支付时间</TableHead>
                <TableHead className="text-right px-5 py-3 font-semibold">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.map((order, i) => (
                <TableRow
                  key={order.id}
                  className={`border-t border-border hover:bg-muted/40 transition-colors ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                >
                  <TableCell className="px-5 py-3 text-sm font-mono text-foreground">{order.order_no}</TableCell>
                  <TableCell className="px-5 py-3 text-sm text-muted-foreground">{order.user_identifier || "—"}</TableCell>
                  <TableCell className="px-5 py-3 text-sm text-foreground text-right">¥{order.amount.toFixed(2)}</TableCell>
                  <TableCell className="px-5 py-3 text-sm text-muted-foreground">{order.pay_method}</TableCell>
                  <TableCell className="px-5 py-3">
                    <span className={`inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full border ${statusBadge(order.status)}`}>
                      {statusLabel(order.status)}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-3 text-sm text-muted-foreground">
                    {new Date(order.created_at).toLocaleString("zh-CN")}
                  </TableCell>
                  <TableCell className="px-5 py-3 text-sm text-muted-foreground">
                    {order.pay_time ? new Date(order.pay_time).toLocaleString("zh-CN") : "—"}
                  </TableCell>
                  <TableCell className="px-5 py-3">
                    <div className="flex justify-end">
                      {order.status === "paid" && (
                        <button
                          onClick={() => openRefundModal(order)}
                          className="flex items-center gap-1 text-xs text-destructive hover:text-destructive/80 px-2.5 py-1.5 rounded-lg hover:bg-destructive/8 transition-colors font-medium border border-destructive/20"
                        >
                          <Undo2 size={11} />
                          退款
                        </button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {/* Pagination */}
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

      {/* Refund Modal */}
      {refundOrder && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="bg-card rounded-2xl shadow-modal w-full max-w-md p-6 slide-up">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-base font-bold text-foreground">订单退款</h3>
              <button
                onClick={closeRefundModal}
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg hover:bg-muted transition-colors text-muted-foreground hover:text-foreground"
                aria-label="关闭"
              >
                <X size={15} />
              </button>
            </div>

            <div className="mb-4 space-y-2 text-sm">
              <div>
                <span className="text-muted-foreground">订单号：</span>
                <span className="text-foreground font-mono">{refundOrder.order_no}</span>
              </div>
              <div>
                <span className="text-muted-foreground">订单金额：</span>
                <span className="text-foreground font-medium">¥{refundOrder.amount.toFixed(2)}</span>
              </div>
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-foreground mb-1.5">退款金额</label>
              <input
                type="number"
                min="0.01"
                step="0.01"
                max={refundOrder.amount}
                className="input-field w-full"
                placeholder="请输入退款金额"
                value={refundAmount}
                onChange={(e) => setRefundAmount(e.target.value)}
              />
              {!Number.isNaN(parsedAmount) && parsedAmount > refundOrder.amount && (
                <p className="text-xs text-destructive mt-1">退款金额不能超过订单金额</p>
              )}
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-foreground mb-1.5">退款理由</label>
              <textarea
                className="input-field w-full"
                rows={3}
                placeholder="请输入退款理由（必填）..."
                value={refundReason}
                onChange={(e) => setRefundReason(e.target.value)}
              />
            </div>

            <div className="mb-5 bg-amber-500/10 border border-amber-500/20 rounded-lg p-3 text-xs text-amber-600 dark:text-amber-400">
              确认后款项将原路退回用户支付宝，并从用户余额扣回，该操作不可撤销。
            </div>

            <div className="flex justify-end gap-2">
              <button onClick={closeRefundModal} className="btn-secondary text-xs">
                取消
              </button>
              <button
                onClick={handleRefund}
                disabled={!refundValid || refundMutation.isPending}
                className={`btn-primary bg-destructive hover:bg-destructive/90 text-xs ${!refundValid ? "opacity-50 cursor-not-allowed" : ""}`}
              >
                {refundMutation.isPending ? "退款中..." : "确认退款"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminOrders;
