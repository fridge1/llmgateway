import { useState } from "react";
import { Bell, CheckCheck, Loader2, ChevronLeft, ChevronRight } from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiGet, apiPut } from "@/lib/api-client";
import { Switch } from "@/components/ui/switch";
import {
  useNotifications,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
} from "@/hooks/use-api";

const PAGE_SIZE = 20;

interface NotificationPreference {
  event_type: string;
  sms: boolean;
}

interface PreferencesResponse {
  preferences: NotificationPreference[];
}

const typeLabels: Record<string, string> = {
  balance_low: "余额不足",
  subscription_expiry: "订阅到期",
  ticket: "工单动态",
  ops_alert: "运维告警",
};

const typeLabel = (t: string) => typeLabels[t] ?? t;

import { PageHeader } from "@/components/ui/page-header";

const NotificationsPage = () => {
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [typeFilter, setTypeFilter] = useState<string>("all");

  const { data, isLoading } = useNotifications(page, PAGE_SIZE);
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllNotificationsRead();

  const { data: prefData, isLoading: prefLoading } = useQuery({
    queryKey: ["notification-preferences"],
    queryFn: () => apiGet<PreferencesResponse>("/api/notification/preferences"),
  });

  const updatePref = useMutation({
    mutationFn: (body: { event_type: string; sms: boolean }) =>
      apiPut("/api/notification/preferences", body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["notification-preferences"] });
      toast.success("通知偏好已更新");
    },
    onError: (err: Error) => {
      qc.invalidateQueries({ queryKey: ["notification-preferences"] });
      toast.error(err.message || "更新失败");
    },
  });

  const notifications = data?.notifications ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  // 类型筛选（前端过滤当前页数据）
  const availableTypes = Array.from(new Set(notifications.map((n) => n.type)));
  const filtered =
    typeFilter === "all"
      ? notifications
      : notifications.filter((n) => n.type === typeFilter);

  const preferences = prefData?.preferences ?? [];

  return (
    <div className="page-container">
      <PageHeader
        eyebrow="消息"
        title="通知中心"
        description="集中查看系统与业务通知，并调整接收偏好。"
        actions={
          <button onClick={() => markAllRead.mutate()} disabled={markAllRead.isPending} className="btn-secondary flex items-center gap-1.5">
            <CheckCheck size={14} />
            全部已读
          </button>
        }
      />

      {/* Notification list */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden mb-6">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-foreground">通知列表</span>
            <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
              {isLoading ? "..." : `${total} 条`}
            </span>
          </div>
          {/* Type filter */}
          <div className="flex items-center gap-1">
            <button
              onClick={() => setTypeFilter("all")}
              className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors cursor-pointer ${
                typeFilter === "all"
                  ? "bg-primary/10 text-primary"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted/60"
              }`}
            >
              全部
            </button>
            {availableTypes.map((t) => (
              <button
                key={t}
                onClick={() => setTypeFilter(t)}
                className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors cursor-pointer ${
                  typeFilter === t
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted/60"
                }`}
              >
                {typeLabel(t)}
              </button>
            ))}
          </div>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 size={18} className="animate-spin text-muted-foreground" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="px-5 py-16">
            <div className="flex flex-col items-center">
              <div className="w-12 h-12 bg-muted rounded-xl flex items-center justify-center mb-3">
                <Bell size={18} className="text-muted-foreground/50" />
              </div>
              <div className="text-sm text-muted-foreground">暂无通知</div>
            </div>
          </div>
        ) : (
          <div>
            {filtered.map((n) => (
              <div
                key={n.id}
                className={`px-5 py-4 border-b border-border last:border-b-0 flex items-start gap-3 transition-colors ${
                  !n.is_read ? "bg-primary/5" : ""
                }`}
              >
                <span
                  className={`mt-1.5 w-2 h-2 rounded-full shrink-0 ${
                    !n.is_read ? "bg-primary" : "bg-transparent"
                  }`}
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium text-foreground leading-snug">
                      {n.title}
                    </p>
                    <span className="text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-muted text-muted-foreground shrink-0">
                      {typeLabel(n.type)}
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground mt-1 whitespace-pre-wrap">
                    {n.content}
                  </p>
                  <p className="text-[10px] text-muted-foreground/60 mt-1.5">
                    {new Date(n.created_at).toLocaleString("zh-CN")}
                  </p>
                </div>
                {!n.is_read && (
                  <button
                    onClick={() => markRead.mutate(n.id)}
                    disabled={markRead.isPending}
                    className="text-xs text-primary hover:underline shrink-0 cursor-pointer disabled:opacity-50"
                  >
                    标记已读
                  </button>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Pagination */}
        <div className="px-5 py-3.5 border-t border-border flex items-center justify-between">
          <div className="text-xs text-muted-foreground">共 {total} 条通知</div>
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

      {/* Notification preferences */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <span className="text-sm font-semibold text-foreground">通知偏好</span>
          <p className="text-xs text-muted-foreground mt-0.5">
            配置各类事件是否通过短信通知
          </p>
        </div>
        {prefLoading ? (
          <div className="flex items-center justify-center py-10">
            <Loader2 size={18} className="animate-spin text-muted-foreground" />
          </div>
        ) : preferences.length === 0 ? (
          <div className="text-center py-10 text-sm text-muted-foreground">
            暂无可配置项
          </div>
        ) : (
          <div>
            {preferences.map((p) => (
              <div
                key={p.event_type}
                className="px-5 py-3.5 border-b border-border last:border-b-0 flex items-center justify-between"
              >
                <div>
                  <div className="text-sm font-medium text-foreground">
                    {typeLabel(p.event_type)}
                  </div>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    {p.event_type}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground">短信</span>
                  <Switch
                    checked={p.sms}
                    disabled={updatePref.isPending}
                    onCheckedChange={(checked) =>
                      updatePref.mutate({ event_type: p.event_type, sms: checked })
                    }
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default NotificationsPage;
