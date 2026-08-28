import { useState } from "react";
import { Clock, BarChart3, ChevronDown, Crown } from "lucide-react";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import type { SubscriptionPlan, SubscriptionWithUsage } from "@/types/api";
import {
  tierColors,
  brandLabels,
  brandColors,
  brandOrder,
  isImagePlan,
  formatDate,
  getBrandFromPlanId,
  type Brand,
} from "./shared";

interface CurrentTabProps {
  availableSubs: SubscriptionWithUsage[];
  plans: SubscriptionPlan[] | undefined;
  onGoToShop: () => void;
}

const CurrentTab = ({ availableSubs, plans, onGoToShop }: CurrentTabProps) => {
  if (availableSubs.length === 0) {
    return (
      <div className="bg-card border border-border rounded-xl shadow-card p-10 flex flex-col items-center justify-center text-center">
        <Crown size={32} className="text-muted-foreground mb-3" />
        <h3 className="text-base font-semibold text-foreground mb-1.5">暂无订阅</h3>
        <p className="text-sm text-muted-foreground mb-5">购买套餐后可在此查看生效中的订阅与用量</p>
        <button onClick={onGoToShop} className="btn-primary px-5 py-2 text-sm font-semibold">
          去购买套餐
        </button>
      </div>
    );
  }

  const grouped: Record<Brand, SubscriptionWithUsage[]> = {
    claude: [],
    openai: [],
    image: [],
  };
  for (const item of availableSubs) {
    const brand = getBrandFromPlanId(item.subscription.plan_id, plans);
    grouped[brand].push(item);
  }

  return (
    <div className="space-y-5">
      {brandOrder.map((brand) => {
        const items = grouped[brand];
        if (!items || items.length === 0) return null;
        return (
          <div key={brand}>
            <div className="flex items-center gap-2 mb-2.5">
              <span className={`inline-block w-1 h-4 rounded-full bg-gradient-to-b ${brandColors[brand]}`} />
              <span className="text-sm font-semibold text-foreground">{brandLabels[brand]}</span>
            </div>
            <div className="space-y-2.5">
              {items.map((item) => (
                <CompactSubscriptionRow
                  key={item.subscription.id}
                  item={item}
                  plan={plans?.find((p) => p.id === item.subscription.plan_id)}
                />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
};

interface CompactRowProps {
  item: SubscriptionWithUsage;
  plan: SubscriptionPlan | undefined;
}

const CompactSubscriptionRow = ({ item, plan }: CompactRowProps) => {
  const [open, setOpen] = useState(false);
  const sub = item.subscription;
  const usage = item.usage;
  const usagePct = usage?.usage_percent ?? 0;
  const planName = plan?.name ?? "";
  const isImage = isImagePlan(planName);
  const hasModelDetails = !!usage?.model_details && usage.model_details.length > 0;

  const progressBg =
    usagePct > 90
      ? "var(--destructive)"
      : usagePct > 70
        ? "#F59E0B"
        : "linear-gradient(90deg, var(--brand-gradient-start), var(--brand-gradient-end))";

  return (
    <Collapsible open={open} onOpenChange={setOpen} className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
      <div className="p-4 sm:p-5">
        <div className="flex flex-wrap items-center gap-3 mb-3">
          <div className={`px-2.5 py-1 rounded-full text-xs font-semibold border ${tierColors[planName]?.badge ?? "bg-muted text-foreground border-border"}`}>
            {plan?.display_name ?? "套餐"}
          </div>
          <div className="px-2 py-0.5 rounded-full text-[11px] font-medium bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20">
            生效中
          </div>
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground ml-auto">
            <Clock size={12} />
            到期 {formatDate(sub.expires_at)}
          </div>
        </div>

        <div className="flex items-center gap-3">
          <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden">
            <div
              className="h-full rounded-full transition-all duration-500"
              style={{ width: `${Math.min(usagePct, 100)}%`, background: progressBg }}
            />
          </div>
          <div className="text-xs font-medium text-foreground tabular-nums whitespace-nowrap">
            {usagePct.toFixed(1)}%
          </div>
        </div>
        <div className="flex items-center justify-between mt-1.5">
          <div className="text-xs text-muted-foreground">
            {isImage ? (
              <>已使用 {Math.round(usage?.total_amount_used ?? 0)} / {Math.round(usage?.quota_amount_cny ?? 0)} 张</>
            ) : (
              <>已使用 ¥{(usage?.total_amount_used ?? 0).toFixed(2)} / ¥{(usage?.quota_amount_cny ?? 0).toFixed(2)}</>
            )}
            {usagePct >= 100 && <span className="ml-1 text-destructive">（超出按余额扣费）</span>}
          </div>
          {hasModelDetails && (
            <CollapsibleTrigger asChild>
              <button className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
                <BarChart3 size={12} />
                {open ? "收起" : "展开用量明细"}
                <ChevronDown size={12} className={`transition-transform duration-200 ${open ? "rotate-180" : ""}`} />
              </button>
            </CollapsibleTrigger>
          )}
        </div>
      </div>
      {hasModelDetails && (
        <CollapsibleContent>
          <div className="border-t border-border bg-muted/20 dark:bg-muted/10 px-4 sm:px-5 py-3">
            <div className="text-xs font-medium text-muted-foreground mb-2">各模型用量明细</div>
            <div className="grid gap-1.5">
              {usage!.model_details.map((m) => (
                <div key={m.model_name} className="flex items-center justify-between text-xs py-1.5 px-3 bg-card rounded-lg border border-border">
                  <span className="font-medium text-foreground truncate mr-3">{m.model_name}</span>
                  <div className="flex items-center gap-4 text-muted-foreground whitespace-nowrap">
                    <span>{m.request_count} 次</span>
                    <span className="tabular-nums">
                      {isImage ? `${Math.round(m.amount_used)} 张` : `¥${m.amount_used.toFixed(4)}`}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </CollapsibleContent>
      )}
    </Collapsible>
  );
};

export default CurrentTab;
