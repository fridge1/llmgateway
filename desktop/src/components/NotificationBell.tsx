import { useState, useRef, useEffect } from "react";
import { useNotifications, useUnreadNotificationCount, useMarkNotificationRead, useMarkAllNotificationsRead } from "@/hooks/use-api";
import { Bell } from "./icons";

export default function NotificationBell() {
  const [open, setOpen] = useState(false);
  const [page] = useState(1);
  const ref = useRef<HTMLDivElement>(null);
  const { data: unreadData } = useUnreadNotificationCount();
  const { data: notifData } = useNotifications(page, 20);
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllNotificationsRead();

  const count = unreadData?.count ?? 0;
  const notifications = notifData?.notifications ?? [];

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  return (
    <div className="relative" ref={ref}>
      <button onClick={() => setOpen(!open)} className="relative p-1.5 rounded-lg hover:bg-obsidian-800 transition-colors">
        <Bell size={16} className="text-obsidian-400" />
        {count > 0 && (
          <span className="absolute -top-0.5 -right-0.5 w-4 h-4 bg-red-500 rounded-full text-[9px] font-bold text-white flex items-center justify-center">
            {count > 99 ? "99+" : count}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-2 w-80 bg-obsidian-900 border border-obsidian-700 rounded-xl shadow-xl z-50 overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-obsidian-800">
            <span className="text-sm font-semibold text-obsidian-50">通知</span>
            {count > 0 && (
              <button onClick={() => markAllRead.mutate()} className="text-xs text-amber-400 hover:text-amber-300">全部已读</button>
            )}
          </div>
          <div className="max-h-80 overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="py-8 text-center text-sm text-obsidian-500">暂无通知</div>
            ) : notifications.map(n => (
              <div
                key={n.id}
                onClick={() => { if (!n.is_read) markRead.mutate(n.id); }}
                className={`px-4 py-3 border-b border-obsidian-800 hover:bg-obsidian-800/50 cursor-pointer transition-colors ${!n.is_read ? "bg-obsidian-800/30" : ""}`}
              >
                <div className="flex items-start gap-2">
                  {!n.is_read && <div className="w-1.5 h-1.5 rounded-full bg-amber-400 mt-1.5 flex-shrink-0" />}
                  <div className="flex-1 min-w-0">
                    <div className="text-xs font-medium text-obsidian-100 truncate">{n.title}</div>
                    <div className="text-xs text-obsidian-400 mt-0.5 line-clamp-2">{n.content}</div>
                    <div className="text-[10px] text-obsidian-500 mt-1">{new Date(n.created_at).toLocaleString("zh-CN")}</div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
