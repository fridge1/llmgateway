import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { Search, Copy, AlertCircle, Package, Clock, QrCode } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

interface CodexOrder {
  order_no: string;
  product: { name: string; sku: string };
  amount: number;
  status: string;
  pay_time?: string;
  redemption_code?: string;
  service_wechat: string;
  created_at: string;
}

type BadgeVariant = "default" | "secondary" | "destructive" | "outline";

const statusMap: Record<string, { label: string; variant: BadgeVariant }> = {
  pending: { label: "待支付", variant: "outline" },
  paid: { label: "已支付", variant: "default" },
  shipped: { label: "已发货", variant: "secondary" },
  completed: { label: "已完成", variant: "secondary" },
  cancelled: { label: "已取消", variant: "destructive" },
  refunded: { label: "已退款", variant: "destructive" },
  expired: { label: "已过期", variant: "outline" },
};

function DetailField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground mb-1">{label}</div>
      <div className="text-sm font-medium break-words">{children}</div>
    </div>
  );
}

export default function CodexOrderQueryPage() {
  const [orderNo, setOrderNo] = useState("");
  const [searchOrderNo, setSearchOrderNo] = useState("");

  const { data: order, isLoading, error } = useQuery({
    queryKey: ["codex", "order", searchOrderNo],
    queryFn: async () => {
      if (!searchOrderNo) return null;
      const res = await fetch(`/api/codex/orders/${searchOrderNo}`);
      if (!res.ok) {
        if (res.status === 404) throw new Error("订单不存在，请核对订单号是否正确");
        throw new Error("查询失败，请稍后重试");
      }
      return res.json() as Promise<CodexOrder>;
    },
    enabled: !!searchOrderNo,
    retry: false,
  });

  const handleSearch = () => {
    const trimmed = orderNo.trim();
    if (trimmed) setSearchOrderNo(trimmed);
  };

  return (
    <div className="container mx-auto px-4 py-8 max-w-2xl">
      <h1 className="text-2xl font-bold mb-6 text-center">Codex 订单查询</h1>

      {/* Search */}
      <div className="flex gap-2 mb-8">
        <Input
          type="text"
          placeholder="请输入订单号（CDX开头）"
          value={orderNo}
          onChange={(e) => setOrderNo(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && handleSearch()}
          aria-label="订单号"
          className="flex-1"
        />
        <Button onClick={handleSearch} disabled={!orderNo.trim()} className="gap-2">
          <Search className="h-4 w-4" />
          查询
        </Button>
      </div>

      {/* Empty state before any search */}
      {!searchOrderNo && (
        <Card className="border-dashed">
          <CardContent className="py-12 text-center">
            <Package className="h-10 w-10 text-muted-foreground/50 mx-auto mb-3" />
            <p className="text-sm text-muted-foreground">输入订单号查询支付与发货状态</p>
            <p className="text-xs text-muted-foreground/70 mt-1">订单号在创建订单时生成，以 CDX 开头</p>
          </CardContent>
        </Card>
      )}

      {/* Loading skeleton */}
      {isLoading && (
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-4 w-48 mt-2" />
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              {[0, 1, 2, 3].map(i => <Skeleton key={i} className="h-12" />)}
            </div>
            <Skeleton className="h-20 w-full" />
          </CardContent>
        </Card>
      )}

      {/* Error state */}
      {error && (
        <Card className="border-destructive/50">
          <CardContent className="pt-6 flex items-start gap-3">
            <AlertCircle className="h-5 w-5 text-destructive shrink-0 mt-0.5" />
            <div>
              <p className="text-destructive font-medium">查询失败</p>
              <p className="text-sm text-muted-foreground mt-1">{(error as Error).message}</p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Result */}
      {order && !isLoading && !error && (
        <Card>
          <CardHeader>
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <CardTitle>订单详情</CardTitle>
                <CardDescription className="font-mono break-all mt-1">{order.order_no}</CardDescription>
              </div>
              <Badge variant={statusMap[order.status]?.variant || "outline"} className="shrink-0">
                {statusMap[order.status]?.label || order.status}
              </Badge>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <DetailField label="商品">{order.product.name}</DetailField>
              <DetailField label="金额"><span className="font-bold text-primary text-lg">¥{order.amount}</span></DetailField>
              <DetailField label="创建时间">
                <span className="flex items-center gap-1.5 text-sm">
                  <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                  {new Date(order.created_at).toLocaleString("zh-CN")}
                </span>
              </DetailField>
              {order.pay_time && (
                <DetailField label="支付时间">
                  <span className="flex items-center gap-1.5 text-sm">
                    <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                    {new Date(order.pay_time).toLocaleString("zh-CN")}
                  </span>
                </DetailField>
              )}
            </div>

            {/* Redemption code */}
            {order.redemption_code && (
              <div className="bg-primary/5 border border-primary/20 rounded-lg p-4">
                <div className="flex items-center justify-between gap-2 mb-2">
                  <div className="text-sm text-muted-foreground">兑换码/账号信息</div>
                  <button
                    onClick={() => { navigator.clipboard.writeText(order.redemption_code!); toast.success("兑换码已复制"); }}
                    className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
                  >
                    <Copy className="h-3.5 w-3.5" />复制
                  </button>
                </div>
                <div className="font-mono text-lg font-bold text-primary break-all">
                  {order.redemption_code}
                </div>
              </div>
            )}

            {/* Pending shipment: contact wechat */}
            {order.status === "paid" && !order.redemption_code && (
              <div className="bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-900 rounded-lg p-4">
                <div className="text-amber-800 dark:text-amber-400">
                  <p className="font-medium mb-2">订单已支付，等待发货</p>
                  <p className="text-sm mb-3">请扫描下方二维码添加客服微信</p>
                  <div className="bg-white rounded-lg p-4 inline-block">
                    <img src="/wechat_QR.png" alt="客服微信二维码" className="w-48 h-48 object-contain" />
                  </div>
                  <p className="text-xs mt-3">提示：添加客服后，请提供订单号 <span className="font-mono font-bold">{order.order_no}</span></p>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
