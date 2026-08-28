import { useState } from "react";
import {
  Megaphone, Plus, ChevronLeft, ChevronRight, Loader, Trash2, X, Pencil, Eye,
} from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
  useAdminAnnouncements,
  useCreateAnnouncement,
  useUpdateAnnouncement,
  useDeleteAnnouncement,
} from "@/hooks/use-api";
import type { Announcement } from "@/types/api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 20;

const statusLabel: Record<string, string> = {
  draft: "草稿",
  published: "已发布",
  archived: "已归档",
};
const statusColor: Record<string, string> = {
  draft: "bg-muted text-muted-foreground",
  published: "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400",
  archived: "bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400",
};
const priorityLabel: Record<string, string> = {
  normal: "普通",
  important: "重要",
  urgent: "紧急",
};
const priorityColor: Record<string, string> = {
  normal: "bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-400",
  important: "bg-orange-50 text-orange-600 dark:bg-orange-500/10 dark:text-orange-400",
  urgent: "bg-red-50 text-red-600 dark:bg-red-500/10 dark:text-red-400",
};
const displayModeLabel: Record<string, string> = {
  banner: "横幅",
  dialog: "弹窗",
};

/* ───── placeholder for remaining code ───── */

interface FormData {
  title: string;
  content: string;
  status: string;
  priority: string;
  display_mode: string;
}

const emptyForm: FormData = {
  title: "",
  content: "",
  status: "draft",
  priority: "normal",
  display_mode: "banner",
};

const AdminAnnouncements = () => {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminAnnouncements(page, PAGE_SIZE);
  const createMut = useCreateAnnouncement();
  const updateMut = useUpdateAnnouncement();
  const deleteMut = useDeleteAnnouncement();

  const [showModal, setShowModal] = useState(false);
  const [editItem, setEditItem] = useState<Announcement | null>(null);
  const [form, setForm] = useState<FormData>(emptyForm);
  const [deleteConfirm, setDeleteConfirm] = useState<Announcement | null>(null);
  const [previewMode, setPreviewMode] = useState(false);

  const list = data?.announcements ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const openCreate = () => {
    setEditItem(null);
    setForm(emptyForm);
    setPreviewMode(false);
    setShowModal(true);
  };

  const openEdit = (a: Announcement) => {
    setEditItem(a);
    setForm({
      title: a.title,
      content: a.content,
      status: a.status,
      priority: a.priority,
      display_mode: a.display_mode,
    });
    setPreviewMode(false);
    setShowModal(true);
  };

  const handleSave = async () => {
    if (editItem) {
      await updateMut.mutateAsync({ id: editItem.id, ...form });
    } else {
      await createMut.mutateAsync(form);
    }
    setShowModal(false);
  };

  const handleDelete = async () => {
    if (!deleteConfirm) return;
    await deleteMut.mutateAsync(deleteConfirm.id);
    setDeleteConfirm(null);
  };

  /* ───── placeholder2 ───── */

  return (
    <div className="page-container">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-lg font-bold text-foreground">公告管理</h1>
          <p className="text-sm text-muted-foreground mt-0.5">创建和管理系统公告</p>
        </div>
        <button onClick={openCreate} className="btn-primary flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium">
          <Plus size={14} /> 新建公告
        </button>
      </div>

      {/* Stats */}
      <div className="flex gap-4 mb-5">
        <div className="flex-1 bg-card border border-border rounded-xl px-5 py-4 shadow-card flex items-center gap-3 stagger-item" style={{ animationDelay: "0ms" }}>
          <div className="w-9 h-9 bg-primary/10 rounded-lg flex items-center justify-center">
            <Megaphone size={16} className="text-primary" />
          </div>
          <div>
            <div className="text-lg font-bold text-foreground">{total}</div>
            <div className="text-xs text-muted-foreground">总公告数</div>
          </div>
        </div>
        <div className="flex-1 bg-card border border-border rounded-xl px-5 py-4 shadow-card flex items-center gap-3 stagger-item" style={{ animationDelay: "80ms" }}>
          <div className="w-9 h-9 bg-emerald-50 dark:bg-emerald-500/10 rounded-lg flex items-center justify-center">
            <Megaphone size={16} className="text-emerald-500" />
          </div>
          <div>
            <div className="text-lg font-bold text-foreground">{list.filter((a) => a.status === "published").length}</div>
            <div className="text-xs text-muted-foreground">已发布</div>
          </div>
        </div>
      </div>

      {/* Table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-20 text-muted-foreground">
            <Loader size={16} className="animate-spin mr-2" /> 加载中...
          </div>
        ) : list.length === 0 ? (
          <div className="text-center py-20 text-muted-foreground text-sm">暂无公告</div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">标题</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">状态</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">优先级</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">展示方式</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">发布时间</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">创建时间</TableHead>
                <TableHead className="text-right px-5 py-3.5 text-xs font-medium text-muted-foreground">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((a, i) => (
                <TableRow key={a.id} className={`border-t border-border hover:bg-accent/30 ${i % 2 === 0 ? "" : "bg-muted/10"}`}>
                  <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground max-w-[300px] truncate">{a.title}</TableCell>
                  <TableCell className="px-5 py-3.5">
                    <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${statusColor[a.status] ?? ""}`}>
                      {statusLabel[a.status] ?? a.status}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-3.5">
                    <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${priorityColor[a.priority] ?? ""}`}>
                      {priorityLabel[a.priority] ?? a.priority}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{displayModeLabel[a.display_mode] ?? a.display_mode}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                    {a.published_at ? new Date(a.published_at).toLocaleString("zh-CN") : "-"}
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{new Date(a.created_at).toLocaleString("zh-CN")}</TableCell>
                  <TableCell className="px-5 py-3.5 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <button onClick={() => openEdit(a)} className="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-primary/10 hover:text-primary transition-colors cursor-pointer" aria-label="编辑">
                        <Pencil size={13} />
                      </button>
                      <button onClick={() => setDeleteConfirm(a)} className="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors cursor-pointer" aria-label="删除">
                        <Trash2 size={13} />
                      </button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-5 py-3 border-t border-border">
            <span className="text-xs text-muted-foreground">共 {total} 条</span>
            <div className="flex items-center gap-2">
              <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)}
                className={`flex h-9 w-9 items-center justify-center rounded-md ${page <= 1 ? "text-muted-foreground/40 cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"}`} aria-label="上一页">
                <ChevronLeft size={14} />
              </button>
              <span className="text-xs text-muted-foreground">{page} / {totalPages}</span>
              <button disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}
                className={`flex h-9 w-9 items-center justify-center rounded-md ${page >= totalPages ? "text-muted-foreground/40 cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"}`} aria-label="下一页">
                <ChevronRight size={14} />
              </button>
            </div>
          </div>
        )}
      </div>

      {/* ───── placeholder3 ───── */}

      {/* Create / Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-[600px] overflow-y-auto rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-base font-bold text-foreground">{editItem ? "编辑公告" : "新建公告"}</h3>
              <button onClick={() => setShowModal(false)} className="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground cursor-pointer" aria-label="关闭">
                <X size={14} />
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">标题</label>
                <input
                  className="input-field w-full"
                  placeholder="公告标题"
                  value={form.title}
                  onChange={(e) => setForm({ ...form, title: e.target.value })}
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="text-xs font-medium text-muted-foreground">内容 (Markdown)</label>
                  <button
                    onClick={() => setPreviewMode(!previewMode)}
                    className="flex items-center gap-1 text-xs text-primary hover:text-primary/80 cursor-pointer"
                  >
                    <Eye size={12} /> {previewMode ? "编辑" : "预览"}
                  </button>
                </div>
                {previewMode ? (
                  <div className="border border-border rounded-lg px-4 py-3 min-h-[160px] prose prose-sm max-w-none text-foreground">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>{form.content || "*暂无内容*"}</ReactMarkdown>
                  </div>
                ) : (
                  <textarea
                    className="input-field w-full min-h-[160px] resize-y font-mono text-sm"
                    placeholder="支持 Markdown 格式"
                    value={form.content}
                    onChange={(e) => setForm({ ...form, content: e.target.value })}
                  />
                )}
              </div>

              <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">状态</label>
                  <select className="input-field w-full" value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}>
                    <option value="draft">草稿</option>
                    <option value="published">发布</option>
                    <option value="archived">归档</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">优先级</label>
                  <select className="input-field w-full" value={form.priority} onChange={(e) => setForm({ ...form, priority: e.target.value })}>
                    <option value="normal">普通</option>
                    <option value="important">重要</option>
                    <option value="urgent">紧急</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">展示方式</label>
                  <select className="input-field w-full" value={form.display_mode} onChange={(e) => setForm({ ...form, display_mode: e.target.value })}>
                    <option value="banner">横幅</option>
                    <option value="dialog">弹窗</option>
                  </select>
                </div>
              </div>
            </div>

            <div className="flex gap-3 justify-end mt-6">
              <button onClick={() => setShowModal(false)} className="btn-secondary px-4 py-2 rounded-lg text-sm">取消</button>
              <button
                onClick={handleSave}
                disabled={!form.title || createMut.isPending || updateMut.isPending}
                className="btn-primary px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
              >
                {(createMut.isPending || updateMut.isPending) ? "保存中..." : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="w-[calc(100vw-2rem)] max-w-[400px] rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <h3 className="text-base font-bold text-foreground mb-1">确认删除</h3>
            <p className="text-sm text-muted-foreground mb-4">
              确定要删除公告「{deleteConfirm.title}」吗？此操作不可撤销。
            </p>
            <div className="flex gap-3 justify-end">
              <button onClick={() => setDeleteConfirm(null)} className="btn-secondary px-4 py-2 rounded-lg text-sm">取消</button>
              <button
                onClick={handleDelete}
                disabled={deleteMut.isPending}
                className="bg-destructive text-destructive-foreground px-4 py-2 rounded-lg text-sm font-medium hover:bg-destructive/90 disabled:opacity-50 cursor-pointer"
              >
                {deleteMut.isPending ? "删除中..." : "删除"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminAnnouncements;
