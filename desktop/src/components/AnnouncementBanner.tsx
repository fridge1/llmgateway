import { useState } from "react";
import { usePublishedAnnouncements } from "@/hooks/use-api";
import { Megaphone, X } from "./icons";

export default function AnnouncementBanner() {
  const { data } = usePublishedAnnouncements();
  const [dismissed, setDismissed] = useState<Set<number>>(new Set());

  const announcements = (data?.announcements ?? []).filter(a => !dismissed.has(a.id));

  if (announcements.length === 0) return null;

  const current = announcements[0];

  return (
    <div className="mx-4 mt-3 bg-amber-500/10 border border-amber-500/20 rounded-lg px-4 py-2.5 flex items-start gap-3">
      <Megaphone size={14} className="text-amber-400 mt-0.5 flex-shrink-0" />
      <div className="flex-1 min-w-0">
        <div className="text-xs font-medium text-amber-300">{current.title}</div>
        <div className="text-xs text-obsidian-300 mt-0.5 line-clamp-2">{current.content}</div>
      </div>
      <button onClick={() => setDismissed(prev => new Set(prev).add(current.id))} className="text-obsidian-500 hover:text-obsidian-300 flex-shrink-0">
        <X size={14} />
      </button>
    </div>
  );
}
