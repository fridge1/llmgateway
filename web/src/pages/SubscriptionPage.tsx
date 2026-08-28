import { useState } from "react";
import { toast } from "sonner";
import { Crown, Loader2, AlertCircle, Wallet } from "lucide-react";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  useSubscriptionPlans,
  useSubscriptionCurrent,
  useSubscribe,
  useCreateSubscriptionPayment,
} from "@/hooks/use-api";
import type { SubscriptionPlan, SubscriptionWithUsage, SubscribeResponse } from "@/types/api";
import CurrentTab from "./subscription/CurrentTab";
import HistoryTab from "./subscription/HistoryTab";
import ShopTab from "./subscription/ShopTab";

import { PageHeader } from "@/components/ui/page-header";

const SubscriptionPage = () => {
  const { data: plansData, isLoading: plansLoading } = useSubscriptionPlans();
  const { data: current, isLoading: currentLoading } = useSubscriptionCurrent();

  if (plansLoading || currentLoading) {
    return (
      <div className="page-container fade-in flex items-center justify-center py-32">
        <Loader2 size={24} className="animate-spin text-primary" />
      </div>
    );
  }

  const activeSubs = current?.subscriptions ?? [];
  return (
    <SubscriptionPageInner
      plans={plansData?.plans}
      activeSubs={activeSubs}
      purchaseDisabled={plansData?.purchase_disabled}
      disabledReason={plansData?.disabled_reason}
    />
  );
};

interface InnerProps {
  plans: SubscriptionPlan[] | undefined;
  activeSubs: SubscriptionWithUsage[];
  purchaseDisabled?: boolean;
  disabledReason?: string;
}

const SubscriptionPageInner = ({ plans, activeSubs, purchaseDisabled, disabledReason }: InnerProps) => {
  const subscribe = useSubscribe();
  const createPayment = useCreateSubscriptionPayment();
  const availableSubs = activeSubs.filter(
    (item) => (item.usage?.usage_percent ?? 0) < 100,
  );
  const [activeTab, setActiveTab] = useState<string>(
    availableSubs.length > 0 ? "current" : "shop",
  );
  const [confirmPlanId, setConfirmPlanId] = useState<number | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [paymentInfo, setPaymentInfo] = useState<SubscribeResponse | null>(null);

  const closeConfirm = () => {
    setConfirmPlanId(null);
    setActionError(null);
    setPaymentInfo(null);
  };

  const handleSubscribe = async () => {
    if (!confirmPlanId) return;
    setActionError(null);
    setPaymentInfo(null);
    try {
      const isMobile = /Android|iPhone|iPad/i.test(navigator.userAgent);
      const result = await subscribe.mutateAsync({
        planId: confirmPlanId,
        clientType: isMobile ? "mobile" : undefined,
      });
      if (result.need_payment) {
        setPaymentInfo(result);
      } else {
        toast.success("套餐订阅成功！");
        setConfirmPlanId(null);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : "操作失败，请重试";
      setActionError(msg);
    }
  };

  const handleGoToPay = async () => {
    if (!confirmPlanId) return;
    setActionError(null);
    try {
      const isMobile = /Android|iPhone|iPad/i.test(navigator.userAgent);
      const result = await createPayment.mutateAsync({
        planId: confirmPlanId,
        clientType: isMobile ? "mobile" : undefined,
      });
      if (result.pay_url) {
        window.open(result.pay_url, "_blank");
        toast.success("支付窗口已打开，请在新窗口中完成支付");
      }
      closeConfirm();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "创建支付订单失败，请重试";
      setActionError(msg);
    }
  };

  const confirmedPlan = plans?.find((p) => p.id === confirmPlanId);

  return (
    <div className="page-container fade-in">
      <PageHeader
        eyebrow="订阅"
        title={<span className="flex items-center gap-2.5"><Crown size={20} className="text-primary" />订阅套餐</span>}
        description="选择适合你的套餐，按月或按年灵活订阅。"
      />

      <Tabs value={activeTab} onValueChange={setActiveTab} className="gap-5">
        <TabsList className="h-10">
          <TabsTrigger value="current" className="px-4">
            我的套餐
            {availableSubs.length > 0 && (
              <span className="ml-1.5 px-1.5 py-0.5 rounded-full text-[10px] font-semibold bg-primary/15 text-primary leading-none">
                {availableSubs.length}
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="history" className="px-4">购买记录</TabsTrigger>
          <TabsTrigger value="shop" className="px-4">购买套餐</TabsTrigger>
        </TabsList>

        <TabsContent value="current">
          <CurrentTab
            availableSubs={availableSubs}
            plans={plans}
            onGoToShop={() => setActiveTab("shop")}
          />
        </TabsContent>

        <TabsContent value="history">
          <HistoryTab />
        </TabsContent>

        <TabsContent value="shop">
          <ShopTab
            plans={plans}
            activeSubs={activeSubs}
            onBuyClick={(planId) => setConfirmPlanId(planId)}
            purchaseDisabled={purchaseDisabled}
            disabledReason={disabledReason}
          />
        </TabsContent>
      </Tabs>

      {confirmPlanId && !paymentInfo && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true" onClick={closeConfirm}>
          <div className="bg-card border border-border rounded-xl shadow-modal p-6 w-full max-w-sm mx-4 slide-up" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-base font-semibold text-foreground mb-2">确认购买</h3>
            <p className="text-sm text-muted-foreground mb-5">
              {`确认购买 ${confirmedPlan?.display_name} 套餐？将从余额扣除 ¥${confirmedPlan?.monthly_price_cny}，有效期 ${confirmedPlan?.duration_days ?? 30} 天，从购买日起计算。`}
            </p>
            {actionError && (
              <div className="flex items-center gap-2 text-sm text-destructive bg-destructive/8 dark:bg-destructive/15 border border-destructive/20 rounded-lg px-3 py-2.5 mb-4">
                <AlertCircle size={14} className="shrink-0" />
                <span>{actionError}</span>
              </div>
            )}
            <div className="flex gap-3">
              <button onClick={closeConfirm} className="flex-1 btn-secondary py-2">取消</button>
              <button
                onClick={handleSubscribe}
                disabled={subscribe.isPending}
                className="flex-1 py-2 rounded-lg text-sm font-semibold text-white transition-colors cursor-pointer flex items-center justify-center gap-1.5 btn-primary disabled:opacity-50"
              >
                {subscribe.isPending && <Loader2 size={14} className="animate-spin" />}
                确认购买
              </button>
            </div>
          </div>
        </div>
      )}

      {paymentInfo && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true" onClick={closeConfirm}>
          <div className="bg-card border border-border rounded-xl shadow-modal p-6 w-full max-w-sm mx-4 slide-up" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-2 mb-4">
              <Wallet size={18} className="text-primary" />
              <h3 className="text-base font-semibold text-foreground">余额不足</h3>
            </div>
            <div className="space-y-2.5 mb-5">
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">套餐价格</span>
                <span className="font-medium text-foreground">¥{paymentInfo.plan_price?.toFixed(2)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">当前余额</span>
                <span className="font-medium text-foreground">¥{paymentInfo.balance?.toFixed(2)}</span>
              </div>
              <div className="border-t border-border my-2" />
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">需要支付</span>
                <span className="font-semibold text-primary">¥{paymentInfo.shortfall?.toFixed(2)}</span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground mb-4">
              支付完成后将自动完成订阅，无需再次操作。
            </p>
            {actionError && (
              <div className="flex items-center gap-2 text-sm text-destructive bg-destructive/8 dark:bg-destructive/15 border border-destructive/20 rounded-lg px-3 py-2.5 mb-4">
                <AlertCircle size={14} className="shrink-0" />
                <span>{actionError}</span>
              </div>
            )}
            <div className="flex gap-3">
              <button onClick={closeConfirm} className="flex-1 btn-secondary py-2">取消</button>
              <button
                onClick={handleGoToPay}
                disabled={createPayment.isPending}
                className="flex-1 py-2 rounded-lg text-sm font-semibold text-white transition-colors cursor-pointer flex items-center justify-center gap-1.5 btn-primary disabled:opacity-50"
              >
                {createPayment.isPending && <Loader2 size={14} className="animate-spin" />}
                确认支付
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default SubscriptionPage;
