import { useState } from "react";
import { MessageSquare, Plus, X, Send } from "lucide-react";
import { toast } from "sonner";
import { Loader2 } from "../components/icons";
import {
  useTickets,
  useTicketDetail,
  useCreateTicket,
  useCreateTicketMessage,
} from "@/hooks/use-api";

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

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { bg: string; text: string }> = {
    open: { bg: "bg-blue-500/10", text: "text-blue-400" },
    pending: { bg: "bg-amber-500/10", text: "text-amber-400" },
    resolved: { bg: "bg-emerald-500/10", text: "text-emerald-400" },
    closed: { bg: "bg-obsidian-700", text: "text-obsidian-400" },
  };
  const s = map[status] ?? { bg: "bg-obsidian-700", text: "text-obsidian-400" };
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${s.bg} ${s.text}`}>
      {statusLabel(status)}
    </span>
  );
}

const formatTime = (iso: string) => new Date(iso).toLocaleString("zh-CN");

export default function TicketsPage() {
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
    <div className="p-6">
      {/* Header */}
      <div className="mb-5 flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold text-obsidian-50">工单支持</h1>
          <p className="text-xs text-obsidian-400 mt-0.5">提交问题工单，客服将尽快回复</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 bg-amber-500 hover:bg-amber-400 hover:shadow-glow-amber text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200 flex items-center gap-1.5"
        >
          <Plus size={14} />
          新建工单
        </button>
      </div>

      {/* Table */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
        <div className="px-4 py-3 border-b border-obsidian-800 flex items-center justify-between">
          <div className="text-sm font-semibold text-obsidian-50">我的工单</div>
          <div className="text-xs text-obsidian-400">共 {total} 条记录</div>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 size={20} className="animate-spin text-amber-400" />
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
                <th className="text-left px-4 py-2 font-medium">主题</th>
                <th className="text-left px-4 py-2 font-medium">分类</th>
                <th className="text-left px-4 py-2 font-medium">关联订单</th>
                <th className="text-left px-4 py-2 font-medium">状态</th>
                <th className="text-left px-4 py-2 font-medium">创建时间</th>
                <th className="text-left px-4 py-2 font-medium">更新时间</th>
              </tr>
            </thead>
            <tbody>
              {tickets.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-16">
                    <div className="flex flex-col items-center text-center">
                      <div className="w-14 h-14 bg-obsidian-800 rounded-2xl flex items-center justify-center mb-4">
                        <MessageSquare size={22} className="text-obsidian-500" />
                      </div>
                      <div className="text-sm font-semibold text-obsidian-100 mb-1">暂无工单</div>
                      <div className="text-xs text-obsidian-500">点击右上角「新建工单」提交您的问题</div>
                    </div>
                  </td>
                </tr>
              ) : (
                tickets.map((t) => (
                  <tr
                    key={t.id}
                    onClick={() => setSelectedId(t.id)}
                    className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors cursor-pointer"
                  >
                    <td className="px-4 py-3 text-sm font-medium text-obsidian-100">{t.subject}</td>
                    <td className="px-4 py-3 text-xs text-obsidian-400">{categoryLabel(t.category)}</td>
                    <td className="px-4 py-3 text-xs font-mono text-obsidian-400">
                      {t.related_order_no || "—"}
                    </td>
                    <td className="px-4 py-3"><StatusBadge status={t.status} /></td>
                    <td className="px-4 py-3 text-xs text-obsidian-400">{formatTime(t.created_at)}</td>
                    <td className="px-4 py-3 text-xs text-obsidian-400">{formatTime(t.updated_at)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        )}

        {/* Pagination */}
        <div className="px-4 py-3 border-t border-obsidian-800 flex items-center justify-between">
          <div className="text-xs text-obsidian-400">共 {total} 条记录</div>
          <div className="flex items-center gap-2">
            <button
              className="px-3 py-1.5 rounded-lg text-xs bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700 disabled:opacity-40 transition-colors"
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
            >上一页</button>
            <span className="text-xs text-obsidian-400">{page} / {totalPages}</span>
            <button
              className="px-3 py-1.5 rounded-lg text-xs bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700 disabled:opacity-40 transition-colors"
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
            >下一页</button>
          </div>
        </div>
      </div>

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl shadow-card-hover w-full max-w-lg max-h-[85vh] overflow-y-auto p-6 mx-4">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-base font-semibold text-obsidian-50">新建工单</h3>
              <button
                onClick={closeCreate}
                className="w-7 h-7 rounded-lg flex items-center justify-center hover:bg-obsidian-800 transition-colors text-obsidian-400 hover:text-obsidian-100"
              >
                <X size={15} />
              </button>
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-obsidian-200 mb-1.5">问题分类</label>
              <select
                className="w-full bg-obsidian-800 border border-obsidian-700 rounded-lg px-3 py-2 text-sm text-obsidian-100 outline-none cursor-pointer focus:border-amber-500/50"
                value={category}
                onChange={(e) => setCategory(e.target.value)}
              >
                {CATEGORIES.map((c) => (
                  <option key={c.key} value={c.key}>{c.label}</option>
                ))}
              </select>
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-obsidian-200 mb-1.5">主题</label>
              <input
                className="w-full bg-obsidian-800 border border-obsidian-700 rounded-lg px-3 py-2 text-sm text-obsidian-100 placeholder-obsidian-500 outline-none focus:border-amber-500/50"
                placeholder="简要描述您的问题"
                maxLength={200}
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
              />
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-obsidian-200 mb-1.5">问题描述</label>
              <textarea
                className="w-full bg-obsidian-800 border border-obsidian-700 rounded-lg px-3 py-2 text-sm text-obsidian-100 placeholder-obsidian-500 outline-none focus:border-amber-500/50 resize-none"
                rows={5}
                placeholder="请详细描述您遇到的问题..."
                value={content}
                onChange={(e) => setContent(e.target.value)}
              />
            </div>

            <div className="mb-5">
              <label className="block text-sm font-medium text-obsidian-200 mb-1.5">
                关联订单号 <span className="text-obsidian-500 font-normal">(可选)</span>
              </label>
              <input
                className="w-full bg-obsidian-800 border border-obsidian-700 rounded-lg px-3 py-2 text-sm text-obsidian-100 placeholder-obsidian-500 outline-none focus:border-amber-500/50"
                placeholder="如与某笔订单相关，请填写订单号"
                value={relatedOrderNo}
                onChange={(e) => setRelatedOrderNo(e.target.value)}
              />
            </div>

            <div className="flex justify-end gap-2">
              <button
                onClick={closeCreate}
                className="px-4 py-2 bg-obsidian-800 hover:bg-obsidian-700 text-obsidian-200 text-sm font-medium rounded-lg transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleCreate}
                disabled={!subject.trim() || !content.trim() || createTicket.isPending}
                className="px-4 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 text-sm font-semibold rounded-lg transition-colors flex items-center gap-1.5"
              >
                {createTicket.isPending && <Loader2 size={14} className="animate-spin" />}
                {createTicket.isPending ? "提交中..." : "提交工单"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Detail Modal */}
      {selectedId !== "" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl shadow-card-hover w-full max-w-2xl max-h-[85vh] flex flex-col p-6 mx-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-base font-semibold text-obsidian-50">工单详情</h3>
              <button
                onClick={() => { setSelectedId(""); setReplyContent(""); }}
                className="w-7 h-7 rounded-lg flex items-center justify-center hover:bg-obsidian-800 transition-colors text-obsidian-400 hover:text-obsidian-100"
              >
                <X size={15} />
              </button>
            </div>

            {!detail ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 size={20} className="animate-spin text-amber-400" />
              </div>
            ) : (
              <>
                {/* Ticket info */}
                <div className="mb-4 pb-4 border-b border-obsidian-800">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="text-sm font-semibold text-obsidian-50">{detail.subject}</span>
                    <StatusBadge status={detail.status} />
                  </div>
                  <div className="flex items-center gap-4 text-xs text-obsidian-400">
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
                          ? "bg-amber-500/15 text-obsidian-50"
                          : "bg-obsidian-800 text-obsidian-100"
                      }`}>
                        <div className={`text-xs mb-1 ${m.sender_role === "user" ? "text-amber-400/80" : "text-obsidian-500"}`}>
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
                      className="w-full bg-obsidian-800 border border-obsidian-700 rounded-lg px-3 py-2 text-sm text-obsidian-100 placeholder-obsidian-500 outline-none focus:border-amber-500/50 resize-none"
                      rows={2}
                      placeholder="输入回复内容..."
                      value={replyContent}
                      onChange={(e) => setReplyContent(e.target.value)}
                    />
                    <button
                      onClick={handleReply}
                      disabled={!replyContent.trim() || createMessage.isPending}
                      className="px-4 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 text-sm font-semibold rounded-lg transition-colors flex items-center gap-1.5 shrink-0"
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
                  <div className="text-xs text-obsidian-500 text-center py-2">
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
}
