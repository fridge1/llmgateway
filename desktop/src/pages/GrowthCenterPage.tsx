import { useState } from "react";
import { CheckSquare, Users, Flame, Trophy } from "lucide-react";
import { toast } from "sonner";
import { Copy, CheckCircle2, Loader2 } from "../components/icons";
import { useCheckinStatus, useCheckin, useTasks, useClaimTask, useReferralInfo } from "@/hooks/use-api";

// 桌面端没有 window.location.origin 可用（Tauri 环境），邀请链接固定指向官网
const SITE_ORIGIN = "https://your-domain.com";

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

  if (isLoading) return <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-5 animate-pulse h-24" />;

  return (
    <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-5 flex items-center justify-between">
      <div>
        <div className="flex items-center gap-2 mb-1">
          <Flame size={16} className="text-orange-400" />
          <span className="text-sm font-semibold text-obsidian-50">每日签到</span>
        </div>
        <p className="text-xs text-obsidian-400">
          {status?.checked_in_today
            ? `今日已签到 · 连续 ${status.current_streak} 天`
            : `连续 ${status?.current_streak ?? 0} 天 · 今日可得 ¥${(status?.next_reward_cny ?? 0).toFixed(2)}`}
        </p>
      </div>
      <button
        onClick={handleCheckin}
        disabled={status?.checked_in_today || checkin.isPending}
        className="px-5 py-2 bg-amber-500 hover:bg-amber-400 hover:shadow-glow-amber disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200 flex items-center gap-2"
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

  if (isLoading) {
    return (
      <div className="space-y-2.5">
        {[...Array(5)].map((_, i) => (
          <div key={i} className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4 animate-pulse h-14" />
        ))}
      </div>
    );
  }

  const tasks = data?.tasks ?? [];
  const done = tasks.filter((t) => t.status !== "pending").length;

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <CheckSquare size={16} className="text-amber-400" />
          <span className="text-sm font-semibold text-obsidian-50">成长任务</span>
        </div>
        <span className="text-xs text-obsidian-400">{done}/{tasks.length} 已完成</span>
      </div>
      <div className="space-y-2.5">
        {tasks.map((t) => (
          <div
            key={t.code}
            className={`bg-obsidian-900 border border-obsidian-700 rounded-xl p-4 flex items-center justify-between transition-colors ${t.status === "claimed" ? "opacity-60" : ""}`}
          >
            <div>
              <div className="text-sm font-medium text-obsidian-100">{t.title}</div>
              <div className="text-xs text-obsidian-500 mt-0.5">{t.description}</div>
            </div>
            <div className="flex items-center gap-3 shrink-0 ml-4">
              {t.reward_cny > 0 && (
                <span className="text-xs font-medium text-emerald-400">+¥{t.reward_cny.toFixed(2)}</span>
              )}
              {t.status === "completed" ? (
                <button
                  onClick={() => handleClaim(t.code)}
                  disabled={claim.isPending}
                  className="px-3 py-1 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 text-xs font-semibold rounded-lg transition-colors"
                >
                  领取
                </button>
              ) : (
                <span className="text-xs px-2 py-0.5 rounded-full bg-obsidian-800 text-obsidian-400">
                  {STATUS_LABEL[t.status]}
                </span>
              )}
            </div>
          </div>
        ))}
        {tasks.length === 0 && (
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl py-8 text-center text-xs text-obsidian-500">
            暂无任务
          </div>
        )}
      </div>
    </div>
  );
}

// ── Referral card ────────────────────────────────────────────────────────────

function ReferralCard() {
  const { data, isLoading } = useReferralInfo();
  const [copied, setCopied] = useState(false);

  const inviteLink = data ? `${SITE_ORIGIN}/register?ref=${data.referral_code}` : "";

  const copy = () => {
    if (!inviteLink) return;
    navigator.clipboard.writeText(inviteLink).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
      toast.success("邀请链接已复制");
    });
  };

  if (isLoading) return <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-5 animate-pulse h-40" />;

  return (
    <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-5">
      <div className="flex items-center gap-2 mb-4">
        <Users size={16} className="text-amber-400" />
        <span className="text-sm font-semibold text-obsidian-50">邀请好友</span>
      </div>
      <div className="grid grid-cols-3 gap-3 mb-4 text-center">
        <div className="bg-obsidian-800/60 rounded-lg p-3">
          <div className="text-xl font-bold text-obsidian-50">{data?.invited_count ?? 0}</div>
          <div className="text-[10px] text-obsidian-400 mt-1">已邀请</div>
        </div>
        <div className="bg-obsidian-800/60 rounded-lg p-3">
          <div className="text-xl font-bold text-obsidian-50">{data?.rewarded_count ?? 0}</div>
          <div className="text-[10px] text-obsidian-400 mt-1">已奖励</div>
        </div>
        <div className="bg-obsidian-800/60 rounded-lg p-3">
          <div className="text-xl font-bold text-emerald-400">¥{(data?.total_reward ?? 0).toFixed(2)}</div>
          <div className="text-[10px] text-obsidian-400 mt-1">累计奖励</div>
        </div>
      </div>
      <div className="flex items-center gap-2 bg-obsidian-800 border border-obsidian-700 rounded-lg px-3 py-2">
        <code className="flex-1 text-xs truncate text-obsidian-400">{inviteLink}</code>
        <button onClick={copy} className="text-amber-400 hover:text-amber-300 shrink-0 transition-colors">
          {copied ? <CheckCircle2 size={16} /> : <Copy size={16} />}
        </button>
      </div>
      <p className="text-[10px] text-obsidian-500 mt-2 text-center">
        好友首充后，双方均可获得余额奖励
      </p>
    </div>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function GrowthCenterPage() {
  return (
    <div className="p-6 max-w-2xl mx-auto">
      <div className="mb-5">
        <h1 className="text-lg font-semibold text-obsidian-50 flex items-center gap-2">
          <Trophy size={18} className="text-amber-400" />
          成长中心
        </h1>
        <p className="text-xs text-obsidian-400 mt-0.5">签到、完成任务、邀请好友，获取余额奖励</p>
      </div>
      <div className="space-y-4">
        <CheckinCard />
        <TaskList />
        <ReferralCard />
      </div>
    </div>
  );
}
