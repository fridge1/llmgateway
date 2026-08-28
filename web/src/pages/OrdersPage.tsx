import { useState } from "react";
import { FileText, CheckCircle, Clock, XCircle, ChevronLeft, ChevronRight, Loader2 } from "lucide-react";
import { useOrders, useRepayOrder } from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { PageHeader } from "@/components/ui/page-header";
function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("zh-CN");
}

const OrdersPage = () => {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data, isLoading } = useOrders(page, pageSize);
  const ordersList = data?.orders ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const statusCounts = data?.status_counts ?? { paid: 0, pending: 0, expired: 0 };

  const stats = [
    { label: "总订单", value: total, icon: FileText, iconBg: "bg-indigo-50 dark:bg-indigo-500/10", iconColor: "text-indigo-500" },
    { label: "已支付", value: statusCounts.paid, icon: CheckCircle, iconBg: "bg-emerald-50 dark:bg-emerald-500/10", iconColor: "text-emerald-500" },
    { label: "待支付", value: statusCounts.pending, icon: Clock, iconBg: "bg-amber-50 dark:bg-amber-500/10", iconColor: "text-amber-500" },
    { label: "已过期", value: statusCounts.expired, icon: XCircle, iconBg: "bg-red-50 dark:bg-red-500/10", iconColor: "text-red-500" },
  ];

  const repayOrder = useRepayOrder();
  const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
    navigator.userAgent
  );

  const handleRepay = async (orderNo: string) => {
    try {
      const res = await repayOrder.mutateAsync({
        order_no: orderNo,
        client_type: isMobile ? "mobile" : undefined,
      });
      if (isMobile) {
        // eslint-disable-next-line react-hooks/immutability
        window.location.href = res.pay_url;
      } else {
        window.open(res.pay_url, "_blank");
      }
    } catch {
      // error handled by React Query
    }
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
      <PageHeader eyebrow="订单" title="充值订单" description="跟踪充值订单状态并继续未完成的支付。" />

      {/* Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4 mb-6">
        {stats.map((s, i) => {
          const Icon = s.icon;
          return (
            <div key={s.label} className="flex-1 stat-card stagger-item" style={{ animationDelay: `${i * 80}ms` }}>
              <div className="flex items-start justify-between mb-3">
                <div className={`w-9 h-9 ${s.iconBg} rounded-lg flex items-center justify-center`}>
                  <Icon size={18} className={s.iconColor} />
                </div>
              </div>
              <div className="text-2xl font-bold text-foreground mb-1">{s.value}</div>
              <div className="text-xs text-muted-foreground">{s.label}</div>
            </div>
          );
        })}
      </div>

      {/* Table */}
      <div className="data-table-card">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div className="text-sm font-semibold text-foreground">订单列表</div>
          <div className="text-xs text-muted-foreground">共 {total} 条记录</div>
        </div>
        <Table className="w-full">
          <TableHeader>
            <TableRow className="table-header">
              <TableHead className="text-left px-5 py-3">订单号</TableHead>
              <TableHead className="text-left px-5 py-3">金额</TableHead>
              <TableHead className="text-left px-5 py-3">支付方式</TableHead>
              <TableHead className="text-left px-5 py-3">状态</TableHead>
              <TableHead className="text-left px-5 py-3">创建时间</TableHead>
              <TableHead className="text-left px-5 py-3">支付时间</TableHead>
              <TableHead className="text-left px-5 py-3">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {ordersList.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="px-5 py-16">
                  <div className="empty-state">
                    <div className="w-14 h-14 bg-muted rounded-2xl flex items-center justify-center mb-4">
                      <FileText size={22} className="text-muted-foreground/50" />
                    </div>
                    <div className="text-sm font-semibold text-foreground mb-1">暂无订单</div>
                    <div className="text-xs text-muted-foreground">您的充值订单将在这里显示</div>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              ordersList.map((order) => (
                <TableRow key={order.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                  <TableCell className="px-5 py-3.5 text-sm font-mono text-muted-foreground">{order.order_no}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground">¥{order.amount.toFixed(2)}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{order.pay_method || "—"}</TableCell>
                  <TableCell className="px-5 py-3.5">
                    {order.status === "paid" && <span className="badge-success"><CheckCircle size={10} />已支付</span>}
                    {order.status === "pending" && <span className="badge-warning"><Clock size={10} />待支付</span>}
                    {order.status === "expired" && <span className="badge-danger"><XCircle size={10} />已过期</span>}
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{formatTime(order.created_at)}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{order.pay_time ? formatTime(order.pay_time) : "—"}</TableCell>
                  <TableCell className="px-5 py-3.5">
                    {order.status === "pending" && new Date(order.expired_at) > new Date() && (
                      <button
                        disabled={repayOrder.isPending}
                        onClick={() => handleRepay(order.order_no)}
                        className="text-xs font-medium text-primary hover:text-primary/80 transition-colors disabled:opacity-50"
                      >
                        {repayOrder.isPending ? "处理中..." : "继续支付"}
                      </button>
                    )}
                  </TableCell>
                </TableRow>
              ))
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
                page <= 1 ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"
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
                page >= totalPages ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"
              }`}
              aria-label="下一页"
            >
              <ChevronRight size={14} />
            </button>
            <select
              className="ml-2 bg-muted border border-border rounded-lg px-2 py-1 text-xs text-foreground outline-none cursor-pointer"
              value={pageSize}
              onChange={(e) => { setPageSize(Number(e.target.value)); setPage(1); }}
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

export default OrdersPage;
