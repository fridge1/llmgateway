import { useState, useRef } from "react";
import { FileText, Loader2, ChevronLeft, ChevronRight, X, Upload } from "lucide-react";
import {
  useAdminInvoiceRequests,
  useAdminInvoiceRequestDetail,
  useAdminProcessInvoice,
  useAdminCompleteInvoice,
  useAdminRejectInvoice,
} from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 20;

const STATUS_TABS = [
  { key: "", label: "全部" },
  { key: "pending", label: "待处理" },
  { key: "processing", label: "处理中" },
  { key: "completed", label: "已完成" },
  { key: "rejected", label: "已驳回" },
] as const;

const statusLabel = (s: string) => {
  switch (s) {
    case "pending": return "待处理";
    case "processing": return "处理中";
    case "completed": return "已完成";
    case "rejected": return "已驳回";
    case "cancelled": return "已取消";
    default: return s;
  }
};

const statusBadge = (s: string) => {
  switch (s) {
    case "pending":
      return "bg-amber-500/10 text-amber-600 border-amber-500/20 dark:text-amber-400";
    case "processing":
      return "bg-blue-500/10 text-blue-600 border-blue-500/20 dark:text-blue-400";
    case "completed":
      return "bg-emerald-500/10 text-emerald-600 border-emerald-500/20 dark:text-emerald-400";
    case "rejected":
      return "bg-destructive/10 text-destructive border-destructive/20";
    case "cancelled":
      return "bg-muted text-muted-foreground border-border";
    default:
      return "bg-muted text-muted-foreground border-border";
  }
};

const typeBadge = (t: string) => {
  if (t === "special") return "bg-amber-500/10 text-amber-600 border-amber-500/20 dark:text-amber-400";
  return "bg-blue-500/10 text-blue-600 border-blue-500/20 dark:text-blue-400";
};

const typeLabel = (t: string) => (t === "special" ? "专票" : "普票");

const AdminInvoices = () => {
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("");
  const [selectedId, setSelectedId] = useState(0);
  const [showRejectForm, setShowRejectForm] = useState(false);
  const [showCompleteForm, setShowCompleteForm] = useState(false);
  const [rejectReason, setRejectReason] = useState("");
  const [invoiceNumber, setInvoiceNumber] = useState("");
  const [invoiceFile, setInvoiceFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { data, isLoading } = useAdminInvoiceRequests(page, PAGE_SIZE, statusFilter);
  const { data: detailData } = useAdminInvoiceRequestDetail(selectedId);
  const processInvoice = useAdminProcessInvoice();
  const rejectInvoice = useAdminRejectInvoice();
  const completeInvoice = useAdminCompleteInvoice();

  const requests = data?.requests ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const pendingCount = requests.filter((r) => r.status === "pending").length;

  const closeModal = () => {
    setSelectedId(0);
    setShowRejectForm(false);
    setShowCompleteForm(false);
    setRejectReason("");
    setInvoiceNumber("");
    setInvoiceFile(null);
  };

  const handleProcess = async (id: number) => {
    await processInvoice.mutateAsync(id);
    closeModal();
  };

  const handleReject = async (id: number) => {
    if (!rejectReason.trim()) return;
    await rejectInvoice.mutateAsync({ id, reason: rejectReason.trim() });
    closeModal();
  };

  const handleComplete = async (id: number) => {
    if (!invoiceNumber.trim() || !invoiceFile) return;
    await completeInvoice.mutateAsync({ id, file: invoiceFile, invoiceNumber: invoiceNumber.trim() });
    closeModal();
  };

  const detail = detailData?.request;
  const detailOrders = detailData?.orders ?? [];
  const detailTitle = detailData?.title;
  const detailUser = detailData?.user;

  return (
    <div className="page-container">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-lg font-bold text-foreground">开票申请管理</h1>
        <p className="text-sm text-muted-foreground mt-0.5">管理所有用户的发票申请</p>
      </div>

      {/* Status filter bar */}
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
              {tab.key === "pending" && pendingCount > 0 && (
                <span className={`ml-1.5 px-1.5 py-0.5 rounded-full text-xs ${
                  isActive ? "bg-primary-foreground/20 text-primary-foreground" : "bg-amber-500/15 text-amber-600"
                }`}>
                  {pendingCount}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-foreground">申请列表</span>
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
                <TableHead className="text-left px-5 py-3.5">申请编号</TableHead>
                <TableHead className="text-left px-5 py-3.5">用户</TableHead>
                <TableHead className="text-left px-5 py-3.5">抬头</TableHead>
                <TableHead className="text-left px-5 py-3.5">类型</TableHead>
                <TableHead className="text-left px-5 py-3.5">金额</TableHead>
                <TableHead className="text-left px-5 py-3.5">申请时间</TableHead>
                <TableHead className="text-left px-5 py-3.5">状态</TableHead>
                <TableHead className="text-right px-5 py-3.5">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {requests.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="px-5 py-16">
                    <div className="empty-state">
                      <div className="w-12 h-12 bg-muted rounded-xl flex items-center justify-center mb-3">
                        <FileText size={18} className="text-muted-foreground/50" />
                      </div>
                      <div className="text-sm text-muted-foreground">暂无申请记录</div>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                requests.map((r, i) => (
                  <TableRow
                    key={r.id}
                    className={`border-t border-border hover:bg-accent/30 transition-colors duration-150 ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                  >
                    <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground">#{r.id}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{r.user_identifier}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-foreground">{r.title?.title_name ?? "-"}</TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium border ${typeBadge(r.invoice_type)}`}>
                        {typeLabel(r.invoice_type)}
                      </span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground">¥{r.total_amount.toFixed(2)}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                      {new Date(r.created_at).toLocaleString("zh-CN")}
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium border ${statusBadge(r.status)}`}>
                        {statusLabel(r.status)}
                      </span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <div className="flex justify-end">
                        <button
                          onClick={() => setSelectedId(r.id)}
                          className="flex items-center gap-1 text-xs text-primary hover:text-primary/80 px-2.5 py-1.5 rounded-lg hover:bg-primary/8 transition-colors font-medium border border-primary/20"
                        >
                          <FileText size={11} />
                          详情
                        </button>
                      </div>
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
              className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${page <= 1 ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"}`}
              aria-label="上一页"
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
              className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${page >= totalPages ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"}`}
              aria-label="下一页"
            >
              <ChevronRight size={13} />
            </button>
          </div>
        </div>
      </div>

      {/* Detail Modal */}
      {selectedId > 0 && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="bg-card rounded-2xl shadow-modal w-full max-w-3xl max-h-[85vh] overflow-y-auto p-6 slide-up">
            {/* Modal header */}
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-base font-bold text-foreground">申请详情</h3>
              <button onClick={closeModal} className="w-7 h-7 rounded-lg flex items-center justify-center hover:bg-muted transition-colors text-muted-foreground hover:text-foreground">
                <X size={15} />
              </button>
            </div>

            {!detail ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 size={20} className="animate-spin text-muted-foreground" />
              </div>
            ) : (
              <>
                {/* 1. Basic Info */}
                <div className="mb-5">
                  <div className="text-sm font-semibold text-foreground mb-2">基本信息</div>
                  <div className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
                    <div>
                      <span className="text-muted-foreground">编号：</span>
                      <span className="text-foreground">#{detail.id}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">状态：</span>
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium border ${statusBadge(detail.status)}`}>
                        {statusLabel(detail.status)}
                      </span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">用户：</span>
                      <span className="text-foreground">{detailUser?.phone ?? "-"}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">申请时间：</span>
                      <span className="text-foreground">{new Date(detail.created_at).toLocaleString("zh-CN")}</span>
                    </div>
                  </div>
                </div>

                {/* 2. Title Info */}
                {detailTitle && (
                  <div className="mb-5">
                    <div className="text-sm font-semibold text-foreground mb-2">抬头信息</div>
                    <div className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
                      <div>
                        <span className="text-muted-foreground">名称：</span>
                        <span className="text-foreground">{detailTitle.title_name}</span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">税号：</span>
                        <span className="text-foreground">{detailTitle.tax_number || "-"}</span>
                      </div>
                      {detailTitle.type === "enterprise" && (
                        <>
                          <div>
                            <span className="text-muted-foreground">开户行：</span>
                            <span className="text-foreground">{detailTitle.bank_name || "-"}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground">账号：</span>
                            <span className="text-foreground">{detailTitle.bank_account || "-"}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground">地址：</span>
                            <span className="text-foreground">{detailTitle.address || "-"}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground">电话：</span>
                            <span className="text-foreground">{detailTitle.phone || "-"}</span>
                          </div>
                        </>
                      )}
                    </div>
                  </div>
                )}

                {/* 3. Linked Orders */}
                <div className="mb-5">
                  <div className="text-sm font-semibold text-foreground mb-2">关联订单</div>
                  <div className="bg-muted/30 rounded-lg overflow-hidden border border-border">
                    <Table className="w-full text-sm">
                      <TableHeader>
                        <TableRow className="border-b border-border">
                          <TableHead className="text-left px-4 py-2 text-muted-foreground font-medium">订单号</TableHead>
                          <TableHead className="text-right px-4 py-2 text-muted-foreground font-medium">金额</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {detailOrders.map((o) => (
                          <TableRow key={o.id} className="border-t border-border">
                            <TableCell className="px-4 py-2 text-foreground">{o.order_id}</TableCell>
                            <TableCell className="px-4 py-2 text-right text-foreground">¥{o.amount.toFixed(2)}</TableCell>
                          </TableRow>
                        ))}
                        <TableRow className="border-t border-border bg-muted/40">
                          <TableCell className="px-4 py-2 font-medium text-foreground">合计</TableCell>
                          <TableCell className="px-4 py-2 text-right font-medium text-foreground">
                            ¥{detailOrders.reduce((s, o) => s + o.amount, 0).toFixed(2)}
                          </TableCell>
                        </TableRow>
                      </TableBody>
                    </Table>
                  </div>
                </div>

                {/* 4. Reject Form (expandable) */}
                {showRejectForm && (
                  <div className="mb-5 bg-destructive/5 border border-destructive/20 rounded-lg p-4">
                    <div className="text-sm font-semibold text-foreground mb-2">驳回原因</div>
                    <textarea
                      className="input-field w-full mb-3"
                      rows={3}
                      placeholder="请输入驳回原因..."
                      value={rejectReason}
                      onChange={(e) => setRejectReason(e.target.value)}
                    />
                    <div className="flex justify-end gap-2">
                      <button
                        onClick={() => { setShowRejectForm(false); setRejectReason(""); }}
                        className="btn-secondary text-xs"
                      >
                        取消
                      </button>
                      <button
                        onClick={() => handleReject(detail.id)}
                        disabled={!rejectReason.trim() || rejectInvoice.isPending}
                        className={`btn-primary bg-destructive hover:bg-destructive/90 text-xs flex items-center gap-1.5 ${!rejectReason.trim() ? "opacity-50 cursor-not-allowed" : ""}`}
                      >
                        {rejectInvoice.isPending ? "提交中..." : "确认驳回"}
                      </button>
                    </div>
                  </div>
                )}

                {/* 5. Complete Form (expandable) */}
                {showCompleteForm && (
                  <div className="mb-5 bg-primary/5 border border-primary/20 rounded-lg p-4">
                    <div className="text-sm font-semibold text-foreground mb-2">完成开票</div>
                    <div className="mb-3">
                      <label className="block text-sm font-medium text-foreground mb-1.5">发票号码</label>
                      <input
                        className="input-field w-full"
                        placeholder="请输入发票号码"
                        value={invoiceNumber}
                        onChange={(e) => setInvoiceNumber(e.target.value)}
                      />
                    </div>
                    <div className="mb-3">
                      <label className="block text-sm font-medium text-foreground mb-1.5">发票文件 (PDF)</label>
                      <input
                        ref={fileInputRef}
                        type="file"
                        accept=".pdf"
                        className="hidden"
                        onChange={(e) => setInvoiceFile(e.target.files?.[0] ?? null)}
                      />
                      <button
                        onClick={() => fileInputRef.current?.click()}
                        className="flex items-center gap-2 px-3 py-2 rounded-lg border border-border text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                      >
                        <Upload size={14} />
                        {invoiceFile ? invoiceFile.name : "选择 PDF 文件"}
                      </button>
                    </div>
                    <div className="flex justify-end gap-2">
                      <button
                        onClick={() => { setShowCompleteForm(false); setInvoiceNumber(""); setInvoiceFile(null); }}
                        className="btn-secondary text-xs"
                      >
                        取消
                      </button>
                      <button
                        onClick={() => handleComplete(detail.id)}
                        disabled={!invoiceNumber.trim() || !invoiceFile || completeInvoice.isPending}
                        className={`btn-primary text-xs flex items-center gap-1.5 ${!invoiceNumber.trim() || !invoiceFile ? "opacity-50 cursor-not-allowed" : ""}`}
                      >
                        {completeInvoice.isPending ? "提交中..." : "确认完成"}
                      </button>
                    </div>
                  </div>
                )}

                {/* 6. Action Buttons */}
                {(detail.status === "pending" || detail.status === "processing") && !showRejectForm && !showCompleteForm && (
                  <div className="flex gap-3 justify-end">
                    <button
                      onClick={() => { setShowRejectForm(true); setShowCompleteForm(false); }}
                      className="btn-primary bg-destructive hover:bg-destructive/90 text-xs"
                    >
                      驳回
                    </button>
                    {detail.status === "pending" && (
                      <button
                        onClick={() => handleProcess(detail.id)}
                        disabled={processInvoice.isPending}
                        className="btn-primary bg-blue-600 hover:bg-blue-700 text-xs"
                      >
                        {processInvoice.isPending ? "处理中..." : "标记处理中"}
                      </button>
                    )}
                    <button
                      onClick={() => { setShowCompleteForm(true); setShowRejectForm(false); }}
                      className="btn-primary text-xs"
                    >
                      完成开票
                    </button>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminInvoices;
