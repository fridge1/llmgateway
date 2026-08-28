import { useState } from 'react';
import { Loader2, CheckCircle, XCircle, Clock, Download, Maximize2, ImageIcon, Pencil, Trash2, CheckSquare, Square } from 'lucide-react';
import { useImageTasks, useDeleteImageTask, useDeleteImageFromTask, useDeleteImageTasksBatch } from '@/hooks/use-image-tasks';
import type { ImageTask } from '@/lib/types-api';

const STATUS_CONFIG: Record<string, { label: string; icon: React.ReactNode; dotClass: string }> = {
  pending: { label: '排队中', icon: <Clock className="w-3 h-3" />, dotClass: 'bg-amber-400' },
  processing: { label: '生成中', icon: <Loader2 className="w-3 h-3 animate-spin" />, dotClass: 'bg-blue-400' },
  completed: { label: '已完成', icon: <CheckCircle className="w-3 h-3" />, dotClass: 'bg-emerald-400' },
  failed: { label: '失败', icon: <XCircle className="w-3 h-3" />, dotClass: 'bg-red-400' },
};

// Confirm dialog — plain native modal (no shadcn)
function ConfirmDialog({
  open, title, description, onConfirm, onCancel, confirmLabel, loading,
}: {
  open: boolean; title: string; description: string;
  onConfirm: () => void; onCancel: () => void;
  confirmLabel?: string; loading?: boolean;
}) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-5 w-80 shadow-xl">
        <h3 className="text-sm font-semibold text-obsidian-50 mb-2">{title}</h3>
        <p className="text-xs text-obsidian-400 mb-5 leading-relaxed">{description}</p>
        <div className="flex justify-end gap-2">
          <button
            onClick={onCancel}
            disabled={loading}
            className="px-3 py-1.5 text-xs rounded-md bg-obsidian-800 border border-obsidian-700 text-obsidian-200 hover:bg-obsidian-700 disabled:opacity-40 cursor-pointer"
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            disabled={loading}
            className="px-3 py-1.5 text-xs rounded-md bg-red-600 text-white hover:bg-red-700 disabled:opacity-40 cursor-pointer"
          >
            {loading ? '删除中…' : (confirmLabel ?? '删除')}
          </button>
        </div>
      </div>
    </div>
  );
}

// Preview lightbox
function PreviewDialog({ url, onClose }: { url: string | null; onClose: () => void }) {
  if (!url) return null;
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 cursor-pointer"
      onClick={onClose}
    >
      <img
        src={url}
        alt="Preview"
        className="max-w-[95vw] max-h-[90vh] object-contain rounded-md"
        onClick={(e) => e.stopPropagation()}
      />
    </div>
  );
}

function TaskCard({
  task, onPreview, onEdit, onDeleteTask, onDeleteImage, selectMode, selected, onToggleSelect,
}: {
  task: ImageTask; onPreview: (url: string) => void; onEdit?: (url: string) => void;
  onDeleteTask: (id: number) => void; onDeleteImage: (taskId: number, url: string) => void;
  selectMode: boolean; selected: boolean; onToggleSelect: (id: number) => void;
}) {
  const config = STATUS_CONFIG[task.status] || STATUS_CONFIG.pending;
  const timeStr = new Date(task.created_at).toLocaleString('zh-CN', {
    timeZone: 'Asia/Shanghai', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  });

  // Download: fetch directly — desktop webview supports https:// TOS URLs without a proxy
  const handleDownload = async (url: string, index: number) => {
    try {
      const response = await fetch(url);
      const blob = await response.blob();
      const blobUrl = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = blobUrl;
      link.download = `image-${Date.now()}-${index + 1}.png`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(blobUrl);
    } catch (error) {
      console.error('Download failed:', error);
    }
  };

  const hasImages = task.result_urls && task.result_urls.length > 0;
  const isActive = task.status === 'pending' || task.status === 'processing';
  const total = task.image_count;
  const done = task.result_urls?.length ?? 0;
  const placeholders = isActive && total > done ? total - done : 0;
  const renderGrid = hasImages || placeholders > 0;
  const totalSlots = done + placeholders;
  const gridCols = totalSlots <= 1 ? '' : 'flex flex-col gap-1';

  return (
    <div
      className={`relative rounded-xl border overflow-hidden mb-3 break-inside-avoid transition-shadow duration-300 ${
        selected ? 'border-amber-500 ring-2 ring-amber-500' : 'border-obsidian-700 bg-obsidian-900'
      }`}
    >
      {selectMode && (
        <button
          onClick={() => onToggleSelect(task.id)}
          className="absolute top-2 left-2 z-10 w-6 h-6 rounded-md bg-obsidian-900/90 backdrop-blur-sm flex items-center justify-center shadow-sm cursor-pointer"
        >
          {selected
            ? <CheckSquare className="w-4 h-4 text-amber-400" />
            : <Square className="w-4 h-4 text-obsidian-400" />}
        </button>
      )}

      {renderGrid && (
        <div className={gridCols} onClick={selectMode ? () => onToggleSelect(task.id) : undefined}>
          {task.result_urls?.map((url, i) => (
            <div key={i} className="relative group overflow-hidden">
              <img src={url} alt={`${task.prompt} - ${i + 1}`} className="w-full h-auto block" loading="lazy" />
              {!selectMode && (
                <div className="absolute inset-0 bg-black/35 opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex items-center justify-center gap-1.5">
                  <button onClick={() => onPreview(url)} className="w-7 h-7 rounded-full bg-white/85 backdrop-blur-sm flex items-center justify-center hover:bg-white transition-colors cursor-pointer">
                    <Maximize2 className="w-3 h-3 text-gray-700" />
                  </button>
                  <button onClick={() => handleDownload(url, i)} className="w-7 h-7 rounded-full bg-white/85 backdrop-blur-sm flex items-center justify-center hover:bg-white transition-colors cursor-pointer">
                    <Download className="w-3 h-3 text-gray-700" />
                  </button>
                  {onEdit && (
                    <button onClick={() => onEdit(url)} className="w-7 h-7 rounded-full bg-white/85 backdrop-blur-sm flex items-center justify-center hover:bg-white transition-colors cursor-pointer" title="修改">
                      <Pencil className="w-3 h-3 text-gray-700" />
                    </button>
                  )}
                  <button onClick={() => onDeleteImage(task.id, url)} className="w-7 h-7 rounded-full bg-white/85 backdrop-blur-sm flex items-center justify-center hover:bg-red-50 transition-colors cursor-pointer">
                    <Trash2 className="w-3 h-3 text-red-600" />
                  </button>
                </div>
              )}
            </div>
          ))}
          {Array.from({ length: placeholders }).map((_, i) => (
            <div key={`skel-${i}`} className="relative aspect-square bg-obsidian-800 animate-pulse flex items-center justify-center">
              <Loader2 className="w-5 h-5 text-obsidian-500 animate-spin" />
            </div>
          ))}
        </div>
      )}

      <div className="px-3 py-2.5 space-y-1.5 group/card">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5 text-[11px]">
            <span className={`w-1.5 h-1.5 rounded-full ${config.dotClass}`} />
            <span className="text-obsidian-400">{config.label}</span>
            <span className="text-obsidian-600">·</span>
            <span className="text-obsidian-400">{task.type === 'edit' ? '编辑' : '生成'}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-[10px] text-obsidian-500">{timeStr}</span>
            {!selectMode && (
              <button
                onClick={() => onDeleteTask(task.id)}
                className="opacity-0 group-hover/card:opacity-100 transition-opacity p-0.5 rounded hover:bg-red-500/10 cursor-pointer"
              >
                <Trash2 className="w-3 h-3 text-obsidian-500 hover:text-red-400" />
              </button>
            )}
          </div>
        </div>
        <p className="text-xs text-obsidian-200 leading-relaxed line-clamp-2">{task.prompt}</p>
        <div className="flex items-center gap-1.5 text-[10px] text-obsidian-500">
          <span>{task.model}</span><span>·</span><span>{task.size}</span><span>·</span><span>{task.image_count}张</span>
          {task.status === 'completed' && task.cost > 0 && (
            <><span>·</span><span>¥{task.cost.toFixed(4)}</span></>
          )}
        </div>
        {task.status === 'failed' && task.error_message && (
          <p className="text-[11px] text-red-400 bg-red-500/10 rounded-md px-2 py-1.5 mt-1">{task.error_message}</p>
        )}
        {isActive && (
          <div className="flex items-center gap-1.5 text-[11px] text-amber-400 mt-0.5">
            <Loader2 className="w-3 h-3 animate-spin" />
            <span>已生成 {done}/{total} 张…</span>
          </div>
        )}
      </div>
    </div>
  );
}

export function TaskList({ onEditImage }: { onEditImage?: (url: string) => void }) {
  const [offset, setOffset] = useState(0);
  const [previewImage, setPreviewImage] = useState<string | null>(null);
  const [confirmDeleteTask, setConfirmDeleteTask] = useState<number | null>(null);
  const [confirmDeleteImage, setConfirmDeleteImage] = useState<{ taskId: number; url: string } | null>(null);
  const [selectMode, setSelectMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [confirmBatchDelete, setConfirmBatchDelete] = useState(false);
  const limit = 20;

  const { data: tasks = [], isLoading } = useImageTasks(limit, offset);
  const deleteTask = useDeleteImageTask();
  const deleteImage = useDeleteImageFromTask();
  const deleteBatch = useDeleteImageTasksBatch();
  const isDeleting = deleteTask.isPending || deleteImage.isPending || deleteBatch.isPending;

  const exitSelectMode = () => { setSelectMode(false); setSelectedIds(new Set()); };
  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => { const next = new Set(prev); next.has(id) ? next.delete(id) : next.add(id); return next; });
  };
  const allSelected = tasks.length > 0 && tasks.every((t) => selectedIds.has(t.id));
  const toggleSelectAll = () => setSelectedIds(allSelected ? new Set() : new Set(tasks.map((t) => t.id)));

  const handleConfirmBatchDelete = async () => {
    if (selectedIds.size === 0) return;
    try { await deleteBatch.mutateAsync(Array.from(selectedIds)); } catch (e) { console.error(e); }
    finally { setConfirmBatchDelete(false); exitSelectMode(); }
  };

  const handleConfirmDeleteTask = async () => {
    if (confirmDeleteTask === null) return;
    try { await deleteTask.mutateAsync(confirmDeleteTask); } catch (e) { console.error(e); }
    finally { setConfirmDeleteTask(null); }
  };

  const handleConfirmDeleteImage = async () => {
    if (!confirmDeleteImage) return;
    try { await deleteImage.mutateAsync(confirmDeleteImage); } catch (e) { console.error(e); }
    finally { setConfirmDeleteImage(null); }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 className="w-5 h-5 animate-spin text-obsidian-400" />
      </div>
    );
  }

  if (tasks.length === 0 && offset === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <ImageIcon size={40} className="text-obsidian-600 mb-3" />
        <p className="text-sm text-obsidian-400">还没有生成记录</p>
        <p className="text-xs text-obsidian-500 mt-1">在左侧输入提示词，开始创作</p>
      </div>
    );
  }

  return (
    <>
      <div>
        <div className="flex items-center justify-between mb-4 gap-2">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold text-obsidian-50">任务历史</h2>
            {tasks.some((t) => t.status === 'pending' || t.status === 'processing') && (
              <span className="flex items-center gap-1 text-[11px] text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded-full">
                <Loader2 className="w-3 h-3 animate-spin" />处理中
              </span>
            )}
          </div>
          {selectMode ? (
            <div className="flex items-center gap-2">
              <button onClick={toggleSelectAll} className="text-xs px-2.5 py-1.5 rounded-md bg-obsidian-800 border border-obsidian-700 text-obsidian-200 hover:bg-obsidian-700 cursor-pointer">
                {allSelected ? '取消全选' : '全选'}
              </button>
              <button
                onClick={() => setConfirmBatchDelete(true)}
                disabled={selectedIds.size === 0 || isDeleting}
                className="text-xs px-2.5 py-1.5 rounded-md bg-red-600 text-white hover:bg-red-700 disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
              >
                删除选中 ({selectedIds.size})
              </button>
              <button onClick={exitSelectMode} disabled={isDeleting} className="text-xs px-2.5 py-1.5 rounded-md bg-obsidian-800 border border-obsidian-700 text-obsidian-200 hover:bg-obsidian-700 cursor-pointer">
                取消
              </button>
            </div>
          ) : (
            tasks.length > 0 && (
              <button onClick={() => setSelectMode(true)} className="text-xs px-2.5 py-1.5 rounded-md bg-obsidian-800 border border-obsidian-700 text-obsidian-200 hover:bg-obsidian-700 cursor-pointer">
                管理
              </button>
            )
          )}
        </div>

        <div className="columns-2 lg:columns-3 gap-3 [column-fill:_balance]">
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              onPreview={setPreviewImage}
              onEdit={onEditImage}
              onDeleteTask={(id) => setConfirmDeleteTask(id)}
              onDeleteImage={(taskId, url) => setConfirmDeleteImage({ taskId, url })}
              selectMode={selectMode}
              selected={selectedIds.has(task.id)}
              onToggleSelect={toggleSelect}
            />
          ))}
        </div>

        {tasks.length >= limit && (
          <div className="flex justify-center gap-2 pt-4 mt-4 border-t border-obsidian-700">
            <button
              disabled={offset === 0}
              onClick={() => setOffset(Math.max(0, offset - limit))}
              className="text-xs px-3 py-1.5 rounded-md bg-obsidian-800 border border-obsidian-700 text-obsidian-200 hover:bg-obsidian-700 disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
            >
              上一页
            </button>
            <button
              onClick={() => setOffset(offset + limit)}
              className="text-xs px-3 py-1.5 rounded-md bg-obsidian-800 border border-obsidian-700 text-obsidian-200 hover:bg-obsidian-700 cursor-pointer"
            >
              下一页
            </button>
          </div>
        )}
      </div>

      <PreviewDialog url={previewImage} onClose={() => setPreviewImage(null)} />

      <ConfirmDialog
        open={confirmBatchDelete}
        title={`删除选中的 ${selectedIds.size} 条任务？`}
        description="将删除选中任务及其所有产物图片，且会从对象存储（TOS）中一并清除。此操作无法撤销。"
        onConfirm={handleConfirmBatchDelete}
        onCancel={() => setConfirmBatchDelete(false)}
        loading={deleteBatch.isPending}
      />

      <ConfirmDialog
        open={confirmDeleteTask !== null}
        title="删除整条任务？"
        description="将删除该任务记录及其所有产物图片，且会从对象存储（TOS）中一并清除。此操作无法撤销。"
        onConfirm={handleConfirmDeleteTask}
        onCancel={() => setConfirmDeleteTask(null)}
        loading={deleteTask.isPending}
      />

      <ConfirmDialog
        open={confirmDeleteImage !== null}
        title="删除这张图片？"
        description="图片将从对象存储（TOS）中删除，无法恢复。若任务下没有剩余图片，整条任务记录也会一并被清除。"
        onConfirm={handleConfirmDeleteImage}
        onCancel={() => setConfirmDeleteImage(null)}
        loading={deleteImage.isPending}
      />
    </>
  );
}
