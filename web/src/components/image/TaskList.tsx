import { useState } from 'react';
import { Loader2, CheckCircle, XCircle, Clock, Download, Maximize2, ImageIcon, Pencil, Trash2, CheckSquare, Square } from 'lucide-react';
import { useImageTasks, useDeleteImageTask, useDeleteImageFromTask, useDeleteImageTasksBatch } from '@/hooks/use-image-tasks';
import { ImageTask } from '@/types/image';
import {
  Dialog,
  DialogContent,
} from '@/components/ui/dialog';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

const STATUS_CONFIG: Record<string, { label: string; icon: React.ReactNode; dotClass: string }> = {
  pending: {
    label: '排队中',
    icon: <Clock className="w-3 h-3" />,
    dotClass: 'bg-amber-400',
  },
  processing: {
    label: '生成中',
    icon: <Loader2 className="w-3 h-3 animate-spin" />,
    dotClass: 'bg-blue-400',
  },
  completed: {
    label: '已完成',
    icon: <CheckCircle className="w-3 h-3" />,
    dotClass: 'bg-emerald-400',
  },
  failed: {
    label: '失败',
    icon: <XCircle className="w-3 h-3" />,
    dotClass: 'bg-red-400',
  },
};

function TaskCard({
  task,
  onPreview,
  onEdit,
  onDeleteTask,
  onDeleteImage,
  selectMode,
  selected,
  onToggleSelect,
}: {
  task: ImageTask;
  onPreview: (url: string) => void;
  onEdit?: (url: string) => void;
  onDeleteTask: (taskId: number) => void;
  onDeleteImage: (taskId: number, url: string) => void;
  selectMode: boolean;
  selected: boolean;
  onToggleSelect: (taskId: number) => void;
}) {
  const config = STATUS_CONFIG[task.status] || STATUS_CONFIG.pending;
  const timeStr = new Date(task.created_at).toLocaleString('zh-CN', {
    timeZone: 'Asia/Shanghai',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });

  const handleDownload = async (url: string, index: number) => {
    try {
      let downloadUrl = url;
      if (url.includes('your-tos-bucket.tos-cn-beijing.volces.com')) {
        downloadUrl = url.replace(
          'https://your-tos-bucket.tos-cn-beijing.volces.com',
          '/tos-proxy'
        );
      }
      const response = await fetch(downloadUrl);
      const blob = await response.blob();
      const blobUrl = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = blobUrl;
      // eslint-disable-next-line react-hooks/purity
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
  // 进行中：渲染 N 个槽位（前 done 个真图 + 剩余骨架占位），让首张图生成后立即可见。
  const placeholders = isActive && total > done ? total - done : 0;
  const renderGrid = hasImages || placeholders > 0;
  const totalSlots = done + placeholders;
  const gridCols = totalSlots <= 1 ? '' : 'flex flex-col gap-1';

  return (
    <div
      className={`relative rounded-xl border bg-card shadow-card overflow-hidden mb-3 break-inside-avoid transition-shadow duration-300 hover:shadow-elevated ${
        selected ? 'border-primary ring-2 ring-primary' : 'border-border'
      }`}
    >
      {/* Select-mode checkbox overlay */}
      {selectMode && (
        <button
          onClick={() => onToggleSelect(task.id)}
          className="absolute top-2 left-2 z-10 w-6 h-6 rounded-md bg-white/90 backdrop-blur-sm flex items-center justify-center shadow-sm cursor-pointer"
          title={selected ? '取消选择' : '选择'}
        >
          {selected ? (
            <CheckSquare className="w-4 h-4 text-primary" />
          ) : (
            <Square className="w-4 h-4 text-muted-foreground" />
          )}
        </button>
      )}

      {/* Images */}
      {renderGrid && (
        <div
          className={gridCols}
          onClick={selectMode ? () => onToggleSelect(task.id) : undefined}
        >
          {task.result_urls?.map((url, i) => (
            <div key={i} className="relative group overflow-hidden">
              <img
                src={url}
                alt={`${task.prompt} - ${i + 1}`}
                className="w-full h-auto block"
                loading="lazy"
              />
              {!selectMode && (
                <div className="absolute inset-0 bg-black/35 opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex items-center justify-center gap-1.5">
                  <button
                    onClick={() => onPreview(url)}
                    className="w-7 h-7 rounded-full bg-white/85 backdrop-blur-sm flex items-center justify-center hover:bg-white transition-colors cursor-pointer"
                  >
                    <Maximize2 className="w-3 h-3 text-gray-700" />
                  </button>
                  <button
                    onClick={() => handleDownload(url, i)}
                    className="w-7 h-7 rounded-full bg-white/85 backdrop-blur-sm flex items-center justify-center hover:bg-white transition-colors cursor-pointer"
                  >
                    <Download className="w-3 h-3 text-gray-700" />
                  </button>
                  {onEdit && (
                    <button
                      onClick={() => onEdit(url)}
                      className="w-7 h-7 rounded-full bg-white/85 backdrop-blur-sm flex items-center justify-center hover:bg-white transition-colors cursor-pointer"
                      title="修改"
                    >
                      <Pencil className="w-3 h-3 text-gray-700" />
                    </button>
                  )}
                  <button
                    onClick={() => onDeleteImage(task.id, url)}
                    className="w-7 h-7 rounded-full bg-white/85 backdrop-blur-sm flex items-center justify-center hover:bg-red-50 transition-colors cursor-pointer"
                    title="删除"
                  >
                    <Trash2 className="w-3 h-3 text-red-600" />
                  </button>
                </div>
              )}
            </div>
          ))}
          {Array.from({ length: placeholders }).map((_, i) => (
            <div
              key={`skel-${i}`}
              className="relative aspect-square bg-muted animate-pulse flex items-center justify-center"
            >
              <Loader2 className="w-5 h-5 text-muted-foreground/50 animate-spin" />
            </div>
          ))}
        </div>
      )}

      {/* Task info */}
      <div className="px-3 py-2.5 space-y-1.5 group/card">
        {/* Status row */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5 text-[11px]">
            <span className={`w-1.5 h-1.5 rounded-full ${config.dotClass}`} />
            <span className="text-muted-foreground">{config.label}</span>
            <span className="text-muted-foreground/50">·</span>
            <span className="text-muted-foreground">{task.type === 'edit' ? '编辑' : '生成'}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-[10px] text-muted-foreground/60">{timeStr}</span>
            {!selectMode && (
              <button
                onClick={() => onDeleteTask(task.id)}
                className="opacity-0 group-hover/card:opacity-100 transition-opacity p-0.5 rounded hover:bg-destructive/10 cursor-pointer"
                title="删除任务"
              >
                <Trash2 className="w-3 h-3 text-muted-foreground hover:text-destructive" />
              </button>
            )}
          </div>
        </div>

        {/* Prompt */}
        <p className="text-xs text-foreground leading-relaxed line-clamp-2">{task.prompt}</p>

        {/* Meta */}
        <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground/60">
          <span>{task.model}</span>
          <span>·</span>
          <span>{task.size}</span>
          <span>·</span>
          <span>{task.image_count}张</span>
        </div>

        {/* Error */}
        {task.status === 'failed' && task.error_message && (
          <p className="text-[11px] text-destructive bg-destructive/8 rounded-md px-2 py-1.5 mt-1">
            {task.error_message}
          </p>
        )}

        {/* Processing indicator */}
        {isActive && (
          <div className="flex items-center gap-1.5 text-[11px] text-primary mt-0.5">
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

  const exitSelectMode = () => {
    setSelectMode(false);
    setSelectedIds(new Set());
  };

  const toggleSelect = (taskId: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(taskId)) next.delete(taskId);
      else next.add(taskId);
      return next;
    });
  };

  const allSelected = tasks.length > 0 && tasks.every((t) => selectedIds.has(t.id));

  const toggleSelectAll = () => {
    setSelectedIds(allSelected ? new Set() : new Set(tasks.map((t) => t.id)));
  };

  const handleConfirmBatchDelete = async () => {
    if (selectedIds.size === 0) return;
    try {
      await deleteBatch.mutateAsync(Array.from(selectedIds));
    } catch (e) {
      console.error('batch delete failed', e);
    } finally {
      setConfirmBatchDelete(false);
      exitSelectMode();
    }
  };

  const handleConfirmDeleteTask = async () => {
    if (confirmDeleteTask === null) return;
    try {
      await deleteTask.mutateAsync(confirmDeleteTask);
    } catch (e) {
      console.error('delete task failed', e);
    } finally {
      setConfirmDeleteTask(null);
    }
  };

  const handleConfirmDeleteImage = async () => {
    if (!confirmDeleteImage) return;
    try {
      await deleteImage.mutateAsync(confirmDeleteImage);
    } catch (e) {
      console.error('delete image failed', e);
    } finally {
      setConfirmDeleteImage(null);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (tasks.length === 0 && offset === 0) {
    return (
      <div className="empty-state">
        <ImageIcon size={40} className="text-muted-foreground/30 mb-3" />
        <p className="text-sm text-muted-foreground">还没有生成记录</p>
        <p className="text-xs text-muted-foreground/60 mt-1">在左侧输入提示词，开始创作</p>
      </div>
    );
  }

  return (
    <>
      <div className="fade-in">
        <div className="flex items-center justify-between mb-4 gap-2">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold text-foreground">任务历史</h2>
            {tasks.some(t => t.status === 'pending' || t.status === 'processing') && (
              <span className="badge-neutral text-[11px]">
                <Loader2 className="w-3 h-3 animate-spin" />
                处理中
              </span>
            )}
          </div>
          {selectMode ? (
            <div className="flex items-center gap-2">
              <button
                onClick={toggleSelectAll}
                className="btn-secondary text-xs px-2.5 py-1.5"
              >
                {allSelected ? '取消全选' : '全选'}
              </button>
              <button
                onClick={() => setConfirmBatchDelete(true)}
                disabled={selectedIds.size === 0 || isDeleting}
                className="text-xs px-2.5 py-1.5 rounded-md bg-destructive text-destructive-foreground hover:bg-destructive/90 disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
              >
                删除选中 ({selectedIds.size})
              </button>
              <button
                onClick={exitSelectMode}
                disabled={isDeleting}
                className="btn-secondary text-xs px-2.5 py-1.5"
              >
                取消
              </button>
            </div>
          ) : (
            tasks.length > 0 && (
              <button
                onClick={() => setSelectMode(true)}
                className="btn-secondary text-xs px-2.5 py-1.5"
              >
                管理
              </button>
            )
          )}
        </div>

        {/* Masonry layout - real aspect ratio, newest first */}
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
          <div className="flex justify-center gap-2 pt-4 mt-4 border-t border-border">
            <button
              disabled={offset === 0}
              onClick={() => setOffset(Math.max(0, offset - limit))}
              className="btn-secondary text-xs px-3 py-1.5 disabled:opacity-40 disabled:cursor-not-allowed"
            >
              上一页
            </button>
            <button
              onClick={() => setOffset(offset + limit)}
              className="btn-secondary text-xs px-3 py-1.5"
            >
              下一页
            </button>
          </div>
        )}
      </div>

      {/* Preview dialog */}
      <Dialog open={previewImage !== null} onOpenChange={() => setPreviewImage(null)}>
        <DialogContent className="!max-w-[95vw] w-auto p-2 bg-black/95 border-none">
          {previewImage && (
            <img
              src={previewImage}
              alt="Preview"
              className="max-w-full max-h-[90vh] w-auto h-auto object-contain rounded-md mx-auto"
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Confirm batch delete */}
      <AlertDialog
        open={confirmBatchDelete}
        onOpenChange={(open) => { if (!open) setConfirmBatchDelete(false); }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除选中的 {selectedIds.size} 条任务？</AlertDialogTitle>
            <AlertDialogDescription>
              将删除选中任务及其所有产物图片，且会从对象存储（TOS）中一并清除。此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmBatchDelete}
              disabled={isDeleting}
              className="bg-destructive hover:bg-destructive/90"
            >
              {deleteBatch.isPending ? '删除中…' : '删除'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Confirm delete entire task */}
      <AlertDialog
        open={confirmDeleteTask !== null}
        onOpenChange={(open) => { if (!open) setConfirmDeleteTask(null); }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除整条任务？</AlertDialogTitle>
            <AlertDialogDescription>
              将删除该任务记录及其所有产物图片，且会从对象存储（TOS）中一并清除。此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDeleteTask}
              disabled={isDeleting}
              className="bg-destructive hover:bg-destructive/90"
            >
              {deleteTask.isPending ? '删除中…' : '删除'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Confirm delete single image */}
      <AlertDialog
        open={confirmDeleteImage !== null}
        onOpenChange={(open) => { if (!open) setConfirmDeleteImage(null); }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除这张图片？</AlertDialogTitle>
            <AlertDialogDescription>
              图片将从对象存储（TOS）中删除，无法恢复。若任务下没有剩余图片，整条任务记录也会一并被清除。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDeleteImage}
              disabled={isDeleting}
              className="bg-destructive hover:bg-destructive/90"
            >
              {deleteImage.isPending ? '删除中…' : '删除'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
