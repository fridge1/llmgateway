import { useState } from "react";
import { CheckSquare, Gift, Users, Copy, Check, Flame, Trophy, Loader2 } from "lucide-react";
import { useCheckinStatus, useCheckin, useTasks, useClaimTask, useReferralInfo } from "@/hooks/use-api";
import { toast } from "sonner";
import { PageHeader } from "@/components/ui/page-header";

// ── Check-in card ────────────────────────────────────────────────────────────

function CheckinCard() {
  const { data: status, isLoading } = useCheckinStatus();
  const checkin = useCheckin();

  const handleCheckin = () => {
    checkin.mutate(undefined, {
      onSuccess: (res) => toast.success(`签到成功！连续 ${res.streak} 天，获得 ¥${res.reward_cny.toFixed(2)}`),
      onError: (e: unknown) => {
        const msg = e instanceof Error ? e.message : "签到失败";
        toast.error(msg.includes("already") ? "今日已签到" : msg);
      },
    });
  };

  if (isLoading) return <div className="rounded-xl border p-6 animate-pulse h-28" />;

  return (
    <div className="flex flex-col items-start justify-between gap-4 rounded-xl border bg-card p-5 shadow-card sm:flex-row sm:items-center sm:p-6">
      <div>
        <div className="flex items-center gap-2 mb-1">
          <Flame size={18} className="text-orange-500" />
          <span className="font-semibold">每日签到</span>
        </div>
        <p className="text-sm text-muted-foreground">
          {status?.checked_in_today
            ? `今日已签到 · 连续 ${status.current_streak} 天`
            : `连续 ${status?.current_streak ?? 0} 天 · 今日可得 ¥${(status?.next_reward_cny ?? 0).toFixed(2)}`}
        </p>
      </div>
      <button
        onClick={handleCheckin}
        disabled={status?.checked_in_today || checkin.isPending}
        className="btn-primary px-5 py-2 text-sm rounded-lg disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
      >
        {checkin.isPending && <Loader2 size={14} className="animate-spin" />}
        {status?.checked_in_today ? "已签到" : "立即签到"}
      </button>
    </div>
  );
}

// ── Task list ────────────────────────────────────────────────────────────────

const STATUS_LABEL = { pending: "待完成", completed: "可领取", claimed: "已领取" } as const;

function TaskList() {
  const { data, isLoading } = useTasks();
  const claim = useClaimTask();

  const handleClaim = (code: string) => {
    claim.mutate(code, {
      onSuccess: (res) => toast.success(`奖励已到账 ¥${res.reward_cny.toFixed(2)}`),
      onError: () => toast.error("领取失败，请稍后重试"),
    });
  };

  if (isLoading) return <div className="space-y-3">{[...Array(5)].map((_, i) => <div key={i} className="rounded-xl border p-4 animate-pulse h-16" />)}</div>;

  const tasks = data?.tasks ?? [];
  const done = tasks.filter((t) => t.status !== "pending").length;

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <CheckSquare size={18} className="text-primary" />
          <span className="font-semibold">成长任务</span>
        </div>
        <span className="text-sm text-muted-foreground">{done}/{tasks.length} 已完成</span>
      </div>
      <div className="space-y-3">
        {tasks.map((t) => (
          <div key={t.code} className={`flex flex-col items-start justify-between gap-3 rounded-xl border bg-card p-4 transition-colors sm:flex-row sm:items-center ${t.status === "claimed" ? "opacity-60" : ""}`}>
            <div>
              <div className="font-medium text-sm">{t.title}</div>
              <div className="text-xs text-muted-foreground mt-0.5">{t.description}</div>
            </div>
            <div className="flex shrink-0 items-center gap-3 sm:ml-4">
              {t.reward_cny > 0 && (
                <span className="text-xs font-medium text-emerald-600 dark:text-emerald-400">+¥{t.reward_cny.toFixed(2)}</span>
              )}
              {t.status === "completed" ? (
                <button
                  onClick={() => handleClaim(t.code)}
                  disabled={claim.isPending}
                  className="btn-primary px-3 py-1 text-xs rounded-lg"
                >
                  领取
                </button>
              ) : (
                <span className={`text-xs px-2 py-0.5 rounded-full ${t.status === "claimed" ? "bg-muted text-muted-foreground" : "bg-muted text-muted-foreground"}`}>
                  {STATUS_LABEL[t.status]}
                </span>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ── Referral card ────────────────────────────────────────────────────────────

function ReferralCard() {
  const { data, isLoading } = useReferralInfo();
  const [copied, setCopied] = useState(false);

  const inviteLink = data ? `${window.location.origin}/register?ref=${data.referral_code}` : "";

  const copy = () => {
    if (!inviteLink) return;
    navigator.clipboard.writeText(inviteLink).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
      toast.success("邀请链接已复制");
    });
  };

  if (isLoading) return <div className="rounded-xl border p-6 animate-pulse h-40" />;

  return (
    <div className="rounded-xl border bg-card p-5 shadow-card sm:p-6">
      <div className="flex items-center gap-2 mb-4">
        <Users size={18} className="text-primary" />
        <span className="font-semibold">邀请好友</span>
      </div>
      <div className="grid grid-cols-3 gap-4 mb-5 text-center">
        <div>
          <div className="text-2xl font-bold">{data?.invited_count ?? 0}</div>
          <div className="text-xs text-muted-foreground mt-1">已邀请</div>
        </div>
        <div>
          <div className="text-2xl font-bold">{data?.rewarded_count ?? 0}</div>
          <div className="text-xs text-muted-foreground mt-1">已奖励</div>
        </div>
        <div>
          <div className="text-2xl font-bold text-emerald-500">¥{(data?.total_reward ?? 0).toFixed(2)}</div>
          <div className="text-xs text-muted-foreground mt-1">累计奖励</div>
        </div>
      </div>
      <div className="flex items-center gap-2 bg-muted rounded-lg px-3 py-2">
        <code className="flex-1 text-xs truncate text-muted-foreground">{inviteLink}</code>
        <button onClick={copy} className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-primary hover:bg-primary/10" aria-label="复制邀请链接">
          {copied ? <Check size={16} /> : <Copy size={16} />}
        </button>
      </div>
      <p className="text-xs text-muted-foreground mt-2 text-center">
        好友首充后，双方均可获得余额奖励
      </p>
    </div>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

const GrowthCenterPage = () => (
  <div className="page-container fade-in max-w-3xl mx-auto">
    <PageHeader
      eyebrow="账户成长"
      title={<span className="flex items-center gap-2.5"><Trophy size={20} className="text-amber-500" />成长中心</span>}
      description="签到、完成任务、邀请好友，持续获取账户余额奖励。"
    />
    <div className="space-y-5">
      <CheckinCard />
      <TaskList />
      <ReferralCard />
    </div>
  </div>
);

export default GrowthCenterPage;
