import { useState, useMemo } from "react";
import { FileText, Download, ChevronLeft, ChevronRight, Loader2, X, Check, Circle } from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/ui/page-header";
import {
  useAvailableOrders,
  useInvoiceTitles,
  useCreateInvoiceRequest,
  useInvoiceRequests,
  useCancelInvoiceRequest,
} from "@/hooks/use-api";
import type { Order, InvoiceTitle } from "@/types/api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("zh-CN");
}

const statusConfig: Record<string, { bg: string; text: string; label: string }> = {
  pending: { bg: "bg-amber-50 dark:bg-amber-500/10", text: "text-amber-700 dark:text-amber-400", label: "待处理" },
  processing: { bg: "bg-blue-50 dark:bg-blue-500/10", text: "text-blue-700 dark:text-blue-400", label: "处理中" },
  completed: { bg: "bg-emerald-50 dark:bg-emerald-500/10", text: "text-emerald-700 dark:text-emerald-400", label: "已完成" },
  rejected: { bg: "bg-red-50 dark:bg-red-500/10", text: "text-red-700 dark:text-red-400", label: "已驳回" },
  cancelled: { bg: "bg-muted", text: "text-muted-foreground", label: "已取消" },
};

const steps = ["已提交", "处理中", "已完成"];

function InvoiceStepper({ status }: { status: string }) {
  if (status === "cancelled" || status === "rejected") {
    const sc = statusConfig[status];
    return (
      <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${sc.bg} ${sc.text}`}>
        {sc.label}
      </span>
    );
  }
  const activeIdx = status === "pending" ? 0 : status === "processing" ? 1 : 2;
  return (
    <div className="flex items-center gap-1">
      {steps.map((label, i) => {
        const done = i <= activeIdx;
        const isCurrent = i === activeIdx;
        return (
          <div key={label} className="flex items-center gap-1">
            {i > 0 && <div className={`w-3 h-px ${done ? "bg-primary" : "bg-border"}`} />}
            <div className="flex items-center gap-0.5">
              {done ? (
                <div className={`w-3.5 h-3.5 rounded-full flex items-center justify-center ${isCurrent && status !== "completed" ? "bg-primary/20" : "bg-primary"}`}>
                  {status === "completed" || i < activeIdx ? (
                    <Check size={8} className="text-primary-foreground" />
                  ) : (
                    <Circle size={6} className="text-primary" />
                  )}
                </div>
              ) : (
                <div className="w-3.5 h-3.5 rounded-full border border-border" />
              )}
              <span className={`text-[10px] ${done ? "text-foreground" : "text-muted-foreground"}`}>{label}</span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

const InvoiceRequestsPage = () => {
  const [activeTab, setActiveTab] = useState<"apply" | "history">("apply");

  // Apply tab state
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [invoiceType, setInvoiceType] = useState<string>("normal");
  const [selectedTitleId, setSelectedTitleId] = useState<number>(0);

  // History tab state
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(20);

  // Queries
  const { data: availableOrders, isLoading: ordersLoading } = useAvailableOrders();
  const { data: titles } = useInvoiceTitles();
  const createRequest = useCreateInvoiceRequest();
  const { data: requestsData, isLoading: requestsLoading } = useInvoiceRequests(page, size);
  const cancelRequest = useCancelInvoiceRequest();

  const orders = useMemo(() => availableOrders ?? [], [availableOrders]);
  const titlesList = useMemo(() => titles ?? [], [titles]);
  const requests = requestsData?.requests ?? [];
  const totalRequests = requestsData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalRequests / size));

  // Derived: selected title object
  const selectedTitle = useMemo(
    () => titlesList.find((t) => t.id === selectedTitleId),
    [titlesList, selectedTitleId],
  );

  // Lock invoice type to normal when personal title selected
  const effectiveInvoiceType =
    selectedTitle?.type === "personal" ? "normal" : invoiceType;

  // Computed totals
  const selectedTotal = useMemo(() => {
    return orders
      .filter((o) => selected.has(o.id))
      .reduce((sum, o) => sum + o.amount, 0);
  }, [orders, selected]);

  // Select all toggle
  const allSelected = orders.length > 0 && selected.size === orders.length;
  const toggleSelectAll = () => {
    if (allSelected) {
      setSelected(new Set());
    } else {
      setSelected(new Set(orders.map((o) => o.id)));
    }
  };

  const toggleOrder = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleSubmit = async () => {
    if (selected.size === 0 || selectedTitleId === 0) return;
    if (effectiveInvoiceType === "special" && selectedTitle) {
      const missing: string[] = [];
      if (!selectedTitle.bank_name) missing.push("开户银行");
      if (!selectedTitle.bank_account) missing.push("银行账号");
      if (!selectedTitle.address) missing.push("企业地址");
      if (!selectedTitle.phone) missing.push("企业电话");
      if (missing.length > 0) {
        toast.error(`开具增值税专用发票需先在「发票抬头」中补齐：${missing.join("、")}`);
        return;
      }
    }
    try {
      await createRequest.mutateAsync({
        title_id: selectedTitleId,
        invoice_type: effectiveInvoiceType,
        order_ids: Array.from(selected),
      });
      setSelected(new Set());
      setActiveTab("history");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "提交失败";
      toast.error(msg);
    }
  };

  const handleCancel = async (id: number) => {
    if (!window.confirm("确定要取消该开票申请吗？")) return;
    try {
      await cancelRequest.mutateAsync(id);
    } catch {
      // error handled by mutation
    }
  };

  const isLoading = activeTab === "apply" ? ordersLoading : requestsLoading;

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
      <PageHeader eyebrow="发票" title="我的发票" description="申请并跟踪发票开具进度" />
      <div className="mb-6">
        <div className="flex items-center gap-2">
          <button
            onClick={() => setActiveTab("apply")}
            className={`px-4 py-1.5 rounded-lg text-sm font-medium transition-all duration-200 ${
              activeTab === "apply"
                ? "bg-primary/10 text-primary font-semibold"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            申请开票
          </button>
          <button
            onClick={() => setActiveTab("history")}
            className={`px-4 py-1.5 rounded-lg text-sm font-medium transition-all duration-200 ${
              activeTab === "history"
                ? "bg-primary/10 text-primary font-semibold"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            开票记录
          </button>
        </div>
      </div>

      {/* Apply Tab */}
      {activeTab === "apply" && (
        <>
          <p className="text-sm text-muted-foreground mb-4">
            勾选需要开票的订单，合并提交开票申请
          </p>

          {/* Orders table */}
          <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
            <Table className="w-full">
              <TableHeader>
                <TableRow className="table-header">
                  <TableHead className="text-left px-5 py-3 w-10">
                    <input
                      type="checkbox"
                      className="rounded border-border"
                      checked={allSelected}
                      onChange={toggleSelectAll}
                    />
                  </TableHead>
                  <TableHead className="text-left px-5 py-3">订单号</TableHead>
                  <TableHead className="text-left px-5 py-3">支付时间</TableHead>
                  <TableHead className="text-left px-5 py-3">支付方式</TableHead>
                  <TableHead className="text-right px-5 py-3">金额</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {orders.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="px-5 py-16">
                      <div className="empty-state">
                        <div className="w-14 h-14 bg-muted rounded-2xl flex items-center justify-center mb-4">
                          <FileText size={22} className="text-muted-foreground/50" />
                        </div>
                        <div className="text-sm font-semibold text-foreground mb-1">暂无可开票订单</div>
                        <div className="text-xs text-muted-foreground">已支付的订单将在这里显示</div>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  orders.map((order) => (
                    <TableRow
                      key={order.id}
                      className="border-t border-border hover:bg-muted/40 transition-colors"
                    >
                      <TableCell className="px-5 py-3.5">
                        <input
                          type="checkbox"
                          className="rounded border-border"
                          checked={selected.has(order.id)}
                          onChange={() => toggleOrder(order.id)}
                        />
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm font-mono text-muted-foreground">
                        {order.order_no}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                        {order.pay_time ? formatTime(order.pay_time) : "—"}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                        {order.pay_method || "—"}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground text-right">
                        ¥{order.amount.toFixed(2)}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* Bottom action bar */}
          <div className="bg-muted/50 rounded-xl p-4 mt-4 flex items-center justify-between">
            <div className="flex items-center gap-4 text-sm">
              <span className="text-muted-foreground">
                已选 <span className="font-medium text-foreground">{selected.size}</span> 笔订单
              </span>
              <span className="text-muted-foreground">
                开票金额: <span className="font-medium text-foreground">¥{selectedTotal.toFixed(2)}</span>
              </span>
            </div>
            <div className="flex items-center gap-3">
              <select
                className="bg-card border border-border rounded-lg px-3 py-1.5 text-sm text-foreground outline-none cursor-pointer"
                value={effectiveInvoiceType}
                onChange={(e) => setInvoiceType(e.target.value)}
                disabled={selectedTitle?.type === "personal"}
              >
                <option value="normal">普通发票</option>
                <option value="special">专用发票</option>
              </select>
              <select
                className="bg-card border border-border rounded-lg px-3 py-1.5 text-sm text-foreground outline-none cursor-pointer"
                value={selectedTitleId}
                onChange={(e) => setSelectedTitleId(Number(e.target.value))}
              >
                <option value={0}>选择抬头</option>
                {titlesList.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.title_name}
                  </option>
                ))}
              </select>
              <button
                onClick={handleSubmit}
                disabled={selected.size === 0 || selectedTitleId === 0 || createRequest.isPending}
                className="px-4 py-1.5 rounded-lg text-sm font-medium bg-primary text-primary-foreground shadow-button hover:brightness-105 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {createRequest.isPending ? (
                  <Loader2 size={14} className="animate-spin inline mr-1" />
                ) : null}
                提交申请
              </button>
            </div>
          </div>
        </>
      )}

      {/* History Tab */}
      {activeTab === "history" && (
        <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
          <div className="px-5 py-4 border-b border-border flex items-center justify-between">
            <div className="text-sm font-semibold text-foreground">开票记录</div>
            <div className="text-xs text-muted-foreground">共 {totalRequests} 条记录</div>
          </div>
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3">编号</TableHead>
                <TableHead className="text-left px-5 py-3">抬头</TableHead>
                <TableHead className="text-left px-5 py-3">类型</TableHead>
                <TableHead className="text-right px-5 py-3">金额</TableHead>
                <TableHead className="text-left px-5 py-3">状态</TableHead>
                <TableHead className="text-left px-5 py-3">申请时间</TableHead>
                <TableHead className="text-left px-5 py-3">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {requests.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="px-5 py-16">
                    <div className="empty-state">
                      <div className="w-14 h-14 bg-muted rounded-2xl flex items-center justify-center mb-4">
                        <FileText size={22} className="text-muted-foreground/50" />
                      </div>
                      <div className="text-sm font-semibold text-foreground mb-1">暂无开票记录</div>
                      <div className="text-xs text-muted-foreground">您的开票申请将在这里显示</div>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                requests.map((req) => {
                  // Resolve title name from titlesList
                  const titleObj = titlesList.find((t) => t.id === req.title_id);
                  return (
                    <TableRow
                      key={req.id}
                      className="border-t border-border hover:bg-muted/40 transition-colors"
                    >
                      <TableCell className="px-5 py-3.5 text-sm font-mono text-muted-foreground">
                        {req.id}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm text-foreground">
                        {titleObj?.title_name ?? "—"}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                        {req.invoice_type === "special" ? "专用发票" : "普通发票"}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground text-right">
                        ¥{req.total_amount.toFixed(2)}
                      </TableCell>
                      <TableCell className="px-5 py-3.5">
                        <InvoiceStepper status={req.status} />
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                        {formatTime(req.created_at)}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm">
                        {req.status === "completed" && (
                          <a
                            href={`/api/invoice/requests/${req.id}/download`}
                            target="_blank"
                            className="text-primary hover:underline"
                            rel="noreferrer"
                          >
                            下载发票
                          </a>
                        )}
                        {req.status === "pending" && (
                          <button
                            onClick={() => handleCancel(req.id)}
                            disabled={cancelRequest.isPending}
                            className="text-destructive hover:underline text-sm cursor-pointer disabled:opacity-50"
                          >
                            取消
                          </button>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>

          {/* Pagination */}
          <div className="px-5 py-3.5 border-t border-border flex items-center justify-between">
            <div className="text-xs text-muted-foreground">共 {totalRequests} 条记录</div>
            <div className="flex items-center gap-2">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
                className={`flex h-9 w-9 items-center justify-center rounded-lg text-sm transition-colors ${
                  page <= 1
                    ? "text-muted-foreground cursor-not-allowed"
                    : "text-foreground hover:bg-muted cursor-pointer"
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
                  page >= totalPages
                    ? "text-muted-foreground cursor-not-allowed"
                    : "text-foreground hover:bg-muted cursor-pointer"
                }`}
                aria-label="下一页"
              >
                <ChevronRight size={14} />
              </button>
              <select
                className="ml-2 bg-muted border border-border rounded-lg px-2 py-1 text-xs text-foreground outline-none cursor-pointer"
                value={size}
                onChange={(e) => {
                  setSize(Number(e.target.value));
                  setPage(1);
                }}
              >
                <option value={10}>10条/页</option>
                <option value={20}>20条/页</option>
                <option value={50}>50条/页</option>
              </select>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default InvoiceRequestsPage;
