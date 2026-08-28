import { BadgePercent } from "lucide-react";
import { usePricing } from "@/hooks/use-api";
import { formatPricingFactor, cn } from "@/lib/utils";

// TenantDiscountBanner shows a notice when the caller's tenant has custom
// pricing (discount or markup) on one or more models. It reuses the /api/pricing
// payload (no extra request) and renders nothing when no custom pricing is present.
export default function TenantDiscountBanner() {
  const { data } = usePricing();
  const items = data?.pricing ?? [];

  const rates = Array.from(
    new Set(
      items
        .map((p) => p.discount_rate)
        .filter((r): r is number => typeof r === "number" && r !== 1),
    ),
  );

  if (rates.length === 0) return null;

  const hasDiscount = rates.some(r => r < 1);
  const hasUplift = rates.some(r => r > 1);

  const message = hasDiscount && hasUplift
    ? "您所在租户有专属定价调整（部分折扣，部分提价），以下消费均按调整后价格结算"
    : hasDiscount
    ? "您所在租户享有专属折扣，以下消费均按折后价结算"
    : "您所在租户有专属提价，以下消费均按调整后价格结算";

  const bgColor = hasDiscount && !hasUplift
    ? "bg-rose-50/60 dark:bg-rose-500/10 border-rose-200/70 dark:border-rose-500/30"
    : hasUplift && !hasDiscount
    ? "bg-amber-50/60 dark:bg-amber-500/10 border-amber-200/70 dark:border-amber-500/30"
    : "bg-blue-50/60 dark:bg-blue-500/10 border-blue-200/70 dark:border-blue-500/30";

  const textColor = hasDiscount && !hasUplift
    ? "text-rose-700 dark:text-rose-300"
    : hasUplift && !hasDiscount
    ? "text-amber-700 dark:text-amber-300"
    : "text-blue-700 dark:text-blue-300";

  const iconColor = hasDiscount && !hasUplift
    ? "text-rose-500"
    : hasUplift && !hasDiscount
    ? "text-amber-500"
    : "text-blue-500";

  return (
    <div className={cn("mb-5 border rounded-xl px-5 py-3.5 flex items-start gap-3", bgColor)}>
      <BadgePercent size={16} className={cn("mt-0.5 shrink-0", iconColor)} />
      <div className={cn("text-xs leading-relaxed", textColor)}>
        {message}
        {rates.length > 0 && (
          <span className="ml-2 font-mono">
            {rates.map(formatPricingFactor).join("、")}
          </span>
        )}
      </div>
    </div>
  );
}
