import { useState } from "react";
import { MessageSquare, Loader2, ChevronLeft, ChevronRight, X, Send } from "lucide-react";
import { toast } from "sonner";
import {
  useAdminTickets,
  useAdminTicketDetail,
  useAdminReplyTicket,
  useAdminUpdateTicketStatus,
} from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 20;

const STATUS_TABS = [
  { key: "", label: "全部" },
  { key: "open", label: "处理中" },
  { key: "pending", label: "待回复" },
  { key: "resolved", label: "已解决" },
  { key: "closed", label: "已关闭" },
] as const;

const categoryLabel = (c: string) => {
  switch (c) {
    case "billing": return "账单";
    case "api": return "API 问题";
    case "account": return "账号";
    case "invoice": return "发票";
    case "other": return "其他";
    default: return c;
  }
};

const statusLabel = (s: string) => {
  switch (s) {
    case "open": return "处理中";
    case "pending": return "待回复";
    case "resolved": return "已解决";
    case "closed": return "已关闭";
    default: return s;
  }
};

const statusBadge = (s: string) => {
  switch (s) {
    case "open":
      return "bg-blue-500/10 text-blue-600 border-blue-500/20 dark:text-blue-400";
    case "pending":
      return "bg-amber-500/10 text-amber-600 border-amber-500/20 dark:text-amber-400";
    case "resolved":
      return "bg-emerald-500/10 text-emerald-600 border-emerald-500/20 dark:text-emerald-400";
    case "closed":
      return "bg-muted text-muted-foreground border-border";
    default:
      return "bg-muted text-muted-foreground border-border";
  }
};

const formatTime = (iso: string) => new Date(iso).toLocaleString("zh-CN");

const AdminTickets = () => {
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("");
  const [selectedId, setSelectedId] = useState("");
  const [replyContent, setReplyContent] = useState("");

  const { data, isLoading } = useAdminTickets(page, PAGE_SIZE, statusFilter);
  const { data: detailData } = useAdminTicketDetail(selectedId);
  const replyTicket = useAdminReplyTicket();
  const updateStatus = useAdminUpdateTicketStatus();

  const tickets = data?.tickets ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const detail = detailData?.ticket;
  const messages = detailData?.messages ?? [];

  const closeModal = () => {
    setSelectedId("");
    setReplyContent("");
  };

  const handleReply = async () => {
    if (!replyContent.trim() || !selectedId) return;
    try {
      await replyTicket.mutateAsync({ id: selectedId, content: replyContent.trim() });
      setReplyContent("");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "回复失败";
      toast.error(msg);
    }
  };

  const handleStatusChange = async (status: string) => {
    if (!selectedId) return;
    try {
      await updateStatus.mutateAsync({ id: selectedId, status });
      toast.success(`已更新为「${statusLabel(status)}」`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "状态更新失败";
      toast.error(msg);
    }
  };

  // Available status transitions (excluding current status)
  const statusActions = detail
    ? (["open", "pending", "resolved", "closed"] as const).filter((s) => s !== detail.status)
    : [];

  return (
    <div className="page-container">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-lg font-bold text-foreground">工单管理</h1>
        <p className="text-sm text-muted-foreground mt-0.5">处理用户提交的支持工单</p>
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
            </button>
          );
        })}
      </div>

      {/* Table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-foreground">工单列表</span>
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
                <TableHead className="text-left px-5 py-3.5">主题</TableHead>
                <TableHead className="text-left px-5 py-3.5">用户</TableHead>
                <TableHead className="text-left px-5 py-3.5">分类</TableHead>
                <TableHead className="text-left px-5 py-3.5">关联订单</TableHead>
                <TableHead className="text-left px-5 py-3.5">状态</TableHead>
                <TableHead className="text-left px-5 py-3.5">更新时间</TableHead>
                <TableHead className="text-right px-5 py-3.5">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tickets.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="px-5 py-16">
                    <div className="empty-state">
                      <div className="w-12 h-12 bg-muted rounded-xl flex items-center justify-center mb-3">
                        <MessageSquare size={18} className="text-muted-foreground/50" />
                      </div>
                      <div className="text-sm text-muted-foreground">暂无工单</div>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                tickets.map((t, i) => (
                  <TableRow
                    key={t.id}
                    className={`border-t border-border hover:bg-accent/30 transition-colors duration-150 ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                  >
                    <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground">{t.subject}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{t.user_identifier ?? "-"}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{categoryLabel(t.category)}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm font-mono text-muted-foreground">
                      {t.related_order_no || "—"}
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium border ${statusBadge(t.status)}`}>
                        {statusLabel(t.status)}
                      </span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{formatTime(t.updated_at)}</TableCell>
                    <TableCell className="px-5 py-3.5">
                      <div className="flex justify-end">
                        <button
                          onClick={() => setSelectedId(t.id)}
                          className="flex items-center gap-1 text-xs text-primary hover:text-primary/80 px-2.5 py-1.5 rounded-lg hover:bg-primary/8 transition-colors font-medium border border-primary/20"
                        >
                          <MessageSquare size={11} />
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

      {/* Detail Modal */}
      {selectedId !== "" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-2xl overflow-y-auto rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-base font-bold text-foreground">工单详情</h3>
              <button onClick={closeModal} className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg hover:bg-muted transition-colors text-muted-foreground hover:text-foreground" aria-label="关闭">
                <X size={15} />
              </button>
            </div>

            {!detail ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 size={20} className="animate-spin text-muted-foreground" />
              </div>
            ) : (
              <>
                {/* Ticket info */}
                <div className="mb-4 pb-4 border-b border-border">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="text-sm font-semibold text-foreground">{detail.subject}</span>
                    <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium border ${statusBadge(detail.status)}`}>
                      {statusLabel(detail.status)}
                    </span>
                  </div>
                  <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                    <span>用户：{detail.user_identifier ?? detail.user_id}</span>
                    <span>分类：{categoryLabel(detail.category)}</span>
                    {detail.related_order_no && <span>关联订单：{detail.related_order_no}</span>}
                    <span>创建于 {formatTime(detail.created_at)}</span>
                  </div>
                </div>

                {/* Conversation (admin messages on the right) */}
                <div className="flex-1 overflow-y-auto space-y-3 mb-4 min-h-[200px]">
                  {messages.map((m) => (
                    <div
                      key={m.id}
                      className={`flex ${m.sender_role === "admin" ? "justify-end" : "justify-start"}`}
                    >
                      <div className={`max-w-[75%] rounded-xl px-4 py-2.5 ${
                        m.sender_role === "admin"
                          ? "bg-primary text-primary-foreground"
                          : "bg-muted text-foreground"
                      }`}>
                        <div className={`text-xs mb-1 ${m.sender_role === "admin" ? "text-primary-foreground/70" : "text-muted-foreground"}`}>
                          {m.sender_role === "admin" ? "客服" : "用户"} · {formatTime(m.created_at)}
                        </div>
                        <div className="text-sm whitespace-pre-wrap break-words">{m.content}</div>
                      </div>
                    </div>
                  ))}
                </div>

                {/* Reply box */}
                <div className="flex gap-2 items-end mb-4">
                  <textarea
                    className="input-field w-full"
                    rows={2}
                    placeholder="输入回复内容..."
                    value={replyContent}
                    onChange={(e) => setReplyContent(e.target.value)}
                  />
                  <button
                    onClick={handleReply}
                    disabled={!replyContent.trim() || replyTicket.isPending}
                    className={`btn-primary text-xs flex items-center gap-1.5 shrink-0 ${!replyContent.trim() ? "opacity-50 cursor-not-allowed" : ""}`}
                  >
                    {replyTicket.isPending ? (
                      <Loader2 size={13} className="animate-spin" />
                    ) : (
                      <Send size={13} />
                    )}
                    回复
                  </button>
                </div>

                {/* Status transition buttons */}
                <div className="flex items-center justify-end gap-2 pt-3 border-t border-border">
                  <span className="text-xs text-muted-foreground mr-auto">流转状态：</span>
                  {statusActions.map((s) => (
                    <button
                      key={s}
                      onClick={() => handleStatusChange(s)}
                      disabled={updateStatus.isPending}
                      className="btn-secondary text-xs disabled:opacity-50"
                    >
                      {statusLabel(s)}
                    </button>
                  ))}
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminTickets;
