import { useState } from "react";
import { MessageSquare, Plus, Loader2, ChevronLeft, ChevronRight, X, Send } from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/ui/page-header";
import {
  useTickets,
  useTicketDetail,
  useCreateTicket,
  useCreateTicketMessage,
} from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 20;

const CATEGORIES = [
  { key: "billing", label: "账单" },
  { key: "api", label: "API 问题" },
  { key: "account", label: "账号" },
  { key: "invoice", label: "发票" },
  { key: "other", label: "其他" },
] as const;

const categoryLabel = (c: string) =>
  CATEGORIES.find((x) => x.key === c)?.label ?? c;

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

const TicketsPage = () => {
  const [page, setPage] = useState(1);
  const [selectedId, setSelectedId] = useState("");
  const [showCreate, setShowCreate] = useState(false);

  // Create form state
  const [category, setCategory] = useState("billing");
  const [subject, setSubject] = useState("");
  const [content, setContent] = useState("");
  const [relatedOrderNo, setRelatedOrderNo] = useState("");

  // Reply state
  const [replyContent, setReplyContent] = useState("");

  const { data, isLoading } = useTickets(page, PAGE_SIZE);
  const { data: detailData } = useTicketDetail(selectedId);
  const createTicket = useCreateTicket();
  const createMessage = useCreateTicketMessage();

  const tickets = data?.tickets ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const detail = detailData?.ticket;
  const messages = detailData?.messages ?? [];
  const canReply = detail && detail.status !== "closed";

  const closeCreate = () => {
    setShowCreate(false);
    setCategory("billing");
    setSubject("");
    setContent("");
    setRelatedOrderNo("");
  };

  const handleCreate = async () => {
    if (!subject.trim() || !content.trim()) return;
    try {
      await createTicket.mutateAsync({
        category,
        subject: subject.trim(),
        content: content.trim(),
        related_order_no: relatedOrderNo.trim() || undefined,
        attachments: [],
      });
      toast.success("工单已提交");
      closeCreate();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "提交失败";
      toast.error(msg);
    }
  };

  const handleReply = async () => {
    if (!replyContent.trim() || !selectedId) return;
    try {
      await createMessage.mutateAsync({
        id: selectedId,
        content: replyContent.trim(),
        attachments: [],
      });
      setReplyContent("");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "发送失败";
      toast.error(msg);
    }
  };

  return (
    <div className="page-container fade-in">
      <PageHeader
        eyebrow="支持"
        title="工单支持"
        description="提交问题工单，客服将尽快回复"
        actions={
          <button
            onClick={() => setShowCreate(true)}
            className="btn-primary text-sm flex items-center gap-1.5"
          >
            <Plus size={14} />
            新建工单
          </button>
        }
      />

      {/* Table */}
      <div className="data-table-card">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div className="text-sm font-semibold text-foreground">我的工单</div>
          <div className="text-xs text-muted-foreground">共 {total} 条记录</div>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3">主题</TableHead>
                <TableHead className="text-left px-5 py-3">分类</TableHead>
                <TableHead className="text-left px-5 py-3">关联订单</TableHead>
                <TableHead className="text-left px-5 py-3">状态</TableHead>
                <TableHead className="text-left px-5 py-3">创建时间</TableHead>
                <TableHead className="text-left px-5 py-3">更新时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tickets.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="px-5 py-16">
                    <div className="empty-state">
                      <div className="w-14 h-14 bg-muted rounded-2xl flex items-center justify-center mb-4">
                        <MessageSquare size={22} className="text-muted-foreground/50" />
                      </div>
                      <div className="text-sm font-semibold text-foreground mb-1">暂无工单</div>
                      <div className="text-xs text-muted-foreground">点击右上角「新建工单」提交您的问题</div>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                tickets.map((t) => (
                  <TableRow
                    key={t.id}
                    onClick={() => setSelectedId(t.id)}
                    className="border-t border-border hover:bg-muted/40 transition-colors cursor-pointer"
                  >
                    <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground">{t.subject}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{categoryLabel(t.category)}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm font-mono text-muted-foreground">
                      {t.related_order_no || "—"}
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium border ${statusBadge(t.status)}`}>
                        {statusLabel(t.status)}
                      </span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{formatTime(t.created_at)}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{formatTime(t.updated_at)}</TableCell>
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

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-lg overflow-y-auto rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-base font-bold text-foreground">新建工单</h3>
              <button onClick={closeCreate} className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg hover:bg-muted transition-colors text-muted-foreground hover:text-foreground" aria-label="关闭">
                <X size={15} />
              </button>
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-foreground mb-1.5">问题分类</label>
              <select
                className="bg-card border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none cursor-pointer w-full"
                value={category}
                onChange={(e) => setCategory(e.target.value)}
              >
                {CATEGORIES.map((c) => (
                  <option key={c.key} value={c.key}>{c.label}</option>
                ))}
              </select>
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-foreground mb-1.5">主题</label>
              <input
                className="input-field w-full"
                placeholder="简要描述您的问题"
                maxLength={200}
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
              />
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-foreground mb-1.5">问题描述</label>
              <textarea
                className="input-field w-full"
                rows={5}
                placeholder="请详细描述您遇到的问题..."
                value={content}
                onChange={(e) => setContent(e.target.value)}
              />
            </div>

            <div className="mb-5">
              <label className="block text-sm font-medium text-foreground mb-1.5">
                关联订单号 <span className="text-muted-foreground font-normal">(可选)</span>
              </label>
              <input
                className="input-field w-full"
                placeholder="如与某笔订单相关，请填写订单号"
                value={relatedOrderNo}
                onChange={(e) => setRelatedOrderNo(e.target.value)}
              />
            </div>

            <div className="flex justify-end gap-2">
              <button onClick={closeCreate} className="btn-secondary text-sm">
                取消
              </button>
              <button
                onClick={handleCreate}
                disabled={!subject.trim() || !content.trim() || createTicket.isPending}
                className={`btn-primary text-sm flex items-center gap-1.5 ${!subject.trim() || !content.trim() ? "opacity-50 cursor-not-allowed" : ""}`}
              >
                {createTicket.isPending ? "提交中..." : "提交工单"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Detail Modal */}
      {selectedId !== "" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-2xl overflow-y-auto rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-base font-bold text-foreground">工单详情</h3>
              <button
                onClick={() => { setSelectedId(""); setReplyContent(""); }}
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg hover:bg-muted transition-colors text-muted-foreground hover:text-foreground"
                aria-label="关闭"
              >
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
                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    <span>分类：{categoryLabel(detail.category)}</span>
                    {detail.related_order_no && <span>关联订单：{detail.related_order_no}</span>}
                    <span>创建于 {formatTime(detail.created_at)}</span>
                  </div>
                </div>

                {/* Conversation */}
                <div className="flex-1 overflow-y-auto space-y-3 mb-4 min-h-[200px]">
                  {messages.map((m) => (
                    <div
                      key={m.id}
                      className={`flex ${m.sender_role === "user" ? "justify-end" : "justify-start"}`}
                    >
                      <div className={`max-w-[75%] rounded-xl px-4 py-2.5 ${
                        m.sender_role === "user"
                          ? "bg-primary text-primary-foreground"
                          : "bg-muted text-foreground"
                      }`}>
                        <div className={`text-xs mb-1 ${m.sender_role === "user" ? "text-primary-foreground/70" : "text-muted-foreground"}`}>
                          {m.sender_role === "user" ? "我" : "客服"} · {formatTime(m.created_at)}
                        </div>
                        <div className="text-sm whitespace-pre-wrap break-words">{m.content}</div>
                      </div>
                    </div>
                  ))}
                </div>

                {/* Reply box */}
                {canReply ? (
                  <div className="flex gap-2 items-end">
                    <textarea
                      className="input-field w-full"
                      rows={2}
                      placeholder="输入回复内容..."
                      value={replyContent}
                      onChange={(e) => setReplyContent(e.target.value)}
                    />
                    <button
                      onClick={handleReply}
                      disabled={!replyContent.trim() || createMessage.isPending}
                      className={`btn-primary text-sm flex items-center gap-1.5 shrink-0 ${!replyContent.trim() ? "opacity-50 cursor-not-allowed" : ""}`}
                    >
                      {createMessage.isPending ? (
                        <Loader2 size={14} className="animate-spin" />
                      ) : (
                        <Send size={14} />
                      )}
                      发送
                    </button>
                  </div>
                ) : (
                  <div className="text-xs text-muted-foreground text-center py-2">
                    工单已关闭，无法继续回复
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

export default TicketsPage;
