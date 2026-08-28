import { useState, useCallback, useMemo } from "react";
import { X, Megaphone, AlertTriangle, AlertCircle } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { usePublishedAnnouncements } from "@/hooks/use-api";
import type { Announcement } from "@/types/api";

const DISMISSED_KEY = "dismissed_announcements";

function getDismissedIds(): Set<number> {
  try {
    const raw = localStorage.getItem(DISMISSED_KEY);
    if (!raw) return new Set();
    return new Set(JSON.parse(raw) as number[]);
  } catch {
    return new Set();
  }
}

function dismissId(id: number) {
  const ids = getDismissedIds();
  ids.add(id);
  localStorage.setItem(DISMISSED_KEY, JSON.stringify([...ids]));
}

const priorityStyles: Record<string, { bg: string; border: string; icon: typeof Megaphone }> = {
  normal: { bg: "bg-indigo-50 dark:bg-indigo-500/10", border: "border-indigo-200 dark:border-indigo-500/20", icon: Megaphone },
  important: { bg: "bg-orange-50 dark:bg-orange-500/10", border: "border-orange-200 dark:border-orange-500/20", icon: AlertTriangle },
  urgent: { bg: "bg-red-50 dark:bg-red-500/10", border: "border-red-200 dark:border-red-500/20", icon: AlertCircle },
};

const priorityTextColor: Record<string, string> = {
  normal: "text-indigo-700 dark:text-indigo-400",
  important: "text-orange-700 dark:text-orange-400",
  urgent: "text-red-700 dark:text-red-400",
};

/* ───── placeholder ───── */

const AnnouncementBanner = () => {
  const { data } = usePublishedAnnouncements();
  const [dismissed, setDismissed] = useState<Set<number>>(getDismissedIds);
  // userClosed: ids the user manually closed in this session, not yet
  // committed to localStorage by the close handler (avoid re-opening dialog
  // before localStorage write completes).
  const [userClosed, setUserClosed] = useState<Set<number>>(() => new Set());

  const allAnnouncements = useMemo(
    () => data?.announcements ?? [],
    [data?.announcements],
  );

  // Filter out dismissed
  const banners = useMemo(
    () =>
      allAnnouncements.filter(
        (a) => a.display_mode === "banner" && !dismissed.has(a.id),
      ),
    [allAnnouncements, dismissed],
  );
  const dialogs = useMemo(
    () =>
      allAnnouncements.filter(
        (a) =>
          a.display_mode === "dialog" &&
          !dismissed.has(a.id) &&
          !userClosed.has(a.id),
      ),
    [allAnnouncements, dismissed, userClosed],
  );

  // Derived: show the first undismissed dialog (no setState in effect).
  const dialogItem: Announcement | null = dialogs[0] ?? null;

  const handleDismiss = useCallback((id: number) => {
    dismissId(id);
    setDismissed((prev) => new Set(prev).add(id));
  }, []);

  const handleDialogClose = useCallback(() => {
    if (dialogItem) {
      handleDismiss(dialogItem.id);
      setUserClosed((prev) => new Set(prev).add(dialogItem.id));
    }
  }, [dialogItem, handleDismiss]);

  return (
    <>
      {/* Banner announcements */}
      {banners.map((a) => {
        const style = priorityStyles[a.priority] ?? priorityStyles.normal;
        const textColor = priorityTextColor[a.priority] ?? "text-blue-700";
        const Icon = style.icon;
        return (
          <div
            key={a.id}
            className={`${style.bg} ${style.border} border-b px-6 py-3 flex items-start gap-3`}
          >
            <Icon size={16} className={`${textColor} mt-0.5 shrink-0`} />
            <div className="flex-1 min-w-0">
              <div className={`text-sm font-medium ${textColor}`}>{a.title}</div>
              {a.content && (
                <div className={`text-xs mt-1 ${textColor} opacity-80 prose prose-sm max-w-none [&>*]:my-0.5`}>
                  <ReactMarkdown remarkPlugins={[remarkGfm]}>{a.content}</ReactMarkdown>
                </div>
              )}
            </div>
            <button
              onClick={() => handleDismiss(a.id)}
              className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-md ${textColor} opacity-60 hover:bg-black/5 hover:opacity-100 cursor-pointer`}
              aria-label={`关闭公告：${a.title}`}
            >
              <X size={14} />
            </button>
          </div>
        );
      })}

      {/* Dialog announcements */}
      <Dialog open={!!dialogItem} onOpenChange={(open) => { if (!open) handleDialogClose(); }}>
        <DialogContent className="max-w-[500px]">
          <DialogHeader>
            <DialogTitle>{dialogItem?.title}</DialogTitle>
            <DialogDescription className="sr-only">系统公告</DialogDescription>
          </DialogHeader>
          {dialogItem?.content && (
            <div className="prose prose-sm max-w-none text-foreground mt-2">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{dialogItem.content}</ReactMarkdown>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
};

export default AnnouncementBanner;
