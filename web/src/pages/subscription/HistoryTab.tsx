import { Loader2, Receipt, Clock } from "lucide-react";
import { useSubscriptionHistory } from "@/hooks/use-api";
import type { UserSubscription } from "@/types/api";
import { tierColors, isImagePlan, formatDate } from "./shared";

const statusConfig: Record<string, { label: string; className: string }> = {
  active: {
    label: "生效中",
    className:
      "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20",
  },
  expired: {
    label: "已过期",
    className:
      "bg-muted text-muted-foreground border-border",
  },
  cancelled: {
    label: "已取消",
    className:
      "bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-500/10 dark:text-rose-400 dark:border-rose-500/20",
  },
};

const HistoryTab = () => {
  const { data, isLoading } = useSubscriptionHistory();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 size={20} className="animate-spin text-primary" />
      </div>
    );
  }

  const items = data ?? [];

  if (items.length === 0) {
    return (
      <div className="bg-card border border-border rounded-xl shadow-card p-10 flex flex-col items-center justify-center text-center">
        <Receipt size={32} className="text-muted-foreground mb-3" />
        <h3 className="text-base font-semibold text-foreground mb-1.5">暂无购买记录</h3>
        <p className="text-sm text-muted-foreground">您尚未购买过任何套餐</p>
      </div>
    );
  }

  return (
    <div className="space-y-2.5">
      {items.map((sub) => (
        <HistoryRow key={sub.id} sub={sub} />
      ))}
    </div>
  );
};

const HistoryRow = ({ sub }: { sub: UserSubscription }) => {
  const plan = sub.plan;
  const planName = plan?.name ?? "";
  const status = statusConfig[sub.status] ?? statusConfig.expired;
  const isImage = isImagePlan(planName);

  return (
    <div className="bg-card border border-border rounded-xl shadow-card p-4 sm:p-5">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <div className={`px-2.5 py-1 rounded-full text-xs font-semibold border ${tierColors[planName]?.badge ?? "bg-muted text-foreground border-border"}`}>
          {plan?.display_name ?? "套餐"}
        </div>
        <div className={`px-2 py-0.5 rounded-full text-[11px] font-medium border ${status.className}`}>
          {status.label}
        </div>
        <div className="ml-auto text-base font-bold text-foreground tabular-nums">
          ¥{(plan?.monthly_price_cny ?? 0).toFixed(2)}
        </div>
      </div>
      <div className="mt-2.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <div className="flex items-center gap-1.5">
          <Clock size={12} />
          {formatDate(sub.started_at)} ~ {formatDate(sub.expires_at)}
        </div>
        {plan && (
          <div>
            额度：
            {isImage
              ? `${Math.round(plan.quota_amount_cny)} 张`
              : `¥${plan.quota_amount_cny.toFixed(0)}`}
          </div>
        )}
        <div>购买于 {formatDate(sub.created_at)}</div>
      </div>
    </div>
  );
};

export default HistoryTab;
