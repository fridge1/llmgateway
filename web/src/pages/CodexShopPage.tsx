import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { CheckCircle2, Copy, ShieldCheck, Zap, Headphones, ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

interface CodexProduct {
  id: number;
  sku: string;
  name: string;
  description: string;
  price_cny: number;
}

interface CreateOrderResponse {
  order_no: string;
  pay_url: string;
  expired_at: string;
  service_wechat: string;
}

export default function CodexShopPage() {
  const [selectedProduct, setSelectedProduct] = useState<number | null>(null);
  const [contactInfo, setContactInfo] = useState("");
  const [contactTouched, setContactTouched] = useState(false);
  const [orderNo, setOrderNo] = useState("");

  const { data: productsData, isLoading } = useQuery({
    queryKey: ["codex", "products"],
    queryFn: async () => {
      const res = await fetch("/api/codex/products");
      if (!res.ok) throw new Error("Failed to fetch products");
      return res.json() as Promise<{ products: CodexProduct[] }>;
    },
  });

  const products = productsData?.products ?? [];

  const createOrder = useMutation({
    mutationFn: async (data: { product_id: number; guest_contact: any; client_type?: string }) => {
      const res = await fetch("/api/codex/orders/create", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        const error = await res.json();
        throw new Error(error.error?.message || "创建订单失败");
      }
      return res.json() as Promise<CreateOrderResponse>;
    },
    onSuccess: (res) => {
      setOrderNo(res.order_no);
      toast.success(`订单创建成功：${res.order_no}`, {
        description: "支付成功后请在订单查询页面查看兑换码",
        duration: 5000,
      });
      const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
      if (isMobile) {
        window.location.href = res.pay_url;
      } else {
        window.open(res.pay_url, "_blank");
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || "创建订单失败，请稍后重试");
    },
  });

  const contactError = contactTouched && contactInfo.trim().length === 0;

  const handlePurchase = () => {
    setContactTouched(true);
    if (!selectedProduct) {
      toast.error("请选择商品");
      return;
    }
    if (!contactInfo.trim()) {
      toast.error("请填写联系方式（手机号/微信号/QQ号）");
      return;
    }
    const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
    createOrder.mutate({
      product_id: selectedProduct,
      guest_contact: contactInfo.trim(),
      client_type: isMobile ? "mobile" : undefined,
    });
  };

  return (
    <div className="container mx-auto px-4 py-8 max-w-5xl">
      {/* Hero */}
      <div className="mb-8 text-center">
        <h1 className="text-3xl font-bold mb-2">Codex 代充服务</h1>
        <p className="text-muted-foreground">选择套餐 → 填写联系方式 → 支付 → 客服发货，兑换码实时可查</p>
      </div>

      {/* Trust strip */}
      <div className="mb-8 grid grid-cols-3 gap-3 max-w-2xl mx-auto text-center text-xs sm:text-sm text-muted-foreground">
        <div className="flex flex-col items-center gap-1">
          <ShieldCheck className="h-5 w-5 text-primary" />
          <span>支付宝担保交易</span>
        </div>
        <div className="flex flex-col items-center gap-1">
          <Zap className="h-5 w-5 text-primary" />
          <span>支付后快速发货</span>
        </div>
        <div className="flex flex-col items-center gap-1">
          <Headphones className="h-5 w-5 text-primary" />
          <span>专属客服跟进</span>
        </div>
      </div>

      {/* Products */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          {[0, 1, 2].map(i => (
            <Card key={i} className="p-6 space-y-3">
              <Skeleton className="h-5 w-2/3" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-9 w-1/2 mt-2" />
            </Card>
          ))}
        </div>
      ) : products.length === 0 ? (
        <Card className="mb-8 border-dashed">
          <CardContent className="py-12 text-center">
            <p className="text-muted-foreground">暂无可购买的套餐，请稍后再来</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          {products.map((product) => {
            const selected = selectedProduct === product.id;
            return (
              <Card
                key={product.id}
                onClick={() => setSelectedProduct(product.id)}
                role="button"
                tabIndex={0}
                onKeyDown={e => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); setSelectedProduct(product.id); } }}
                aria-pressed={selected}
                className={`cursor-pointer transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.99] ${
                  selected
                    ? "border-primary border-2 shadow-lg ring-1 ring-primary/30"
                    : "hover:border-primary/50 hover:shadow-md"
                }`}
              >
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <CardTitle className="text-lg">{product.name}</CardTitle>
                    {selected && (
                      <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-foreground shrink-0">
                        <CheckCircle2 className="h-4 w-4" />
                      </span>
                    )}
                  </div>
                  <CardDescription>{product.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex items-baseline gap-1">
                    <span className="text-3xl font-bold text-primary">¥{product.price_cny}</span>
                    <span className="text-sm text-muted-foreground">/ 套</span>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      {/* Contact form */}
      <Card className="max-w-md mx-auto mb-6">
        <CardHeader>
          <CardTitle>填写联系方式</CardTitle>
          <CardDescription>请填写手机号、微信号或QQ号（任选其一）</CardDescription>
        </CardHeader>
        <CardContent>
          <div>
            <Label htmlFor="contact" className="mb-1.5 block">
              联系方式 <span className="text-destructive">*</span>
            </Label>
            <Input
              id="contact"
              type="text"
              placeholder="手机号 / 微信号 / QQ号"
              value={contactInfo}
              onChange={(e) => setContactInfo(e.target.value)}
              onBlur={() => setContactTouched(true)}
              aria-invalid={contactError}
              aria-describedby="contact-help"
              className={contactError ? "border-destructive focus-visible:ring-destructive" : ""}
            />
            {contactError ? (
              <p id="contact-error" className="text-xs text-destructive mt-2">联系方式不能为空</p>
            ) : (
              <p id="contact-help" className="text-xs text-muted-foreground mt-2">
                用于发货后联系您，请确保填写正确
              </p>
            )}
          </div>
        </CardContent>
      </Card>

      {/* CTA */}
      <div className="text-center">
        <Button
          onClick={handlePurchase}
          disabled={createOrder.isPending || !selectedProduct}
          size="lg"
          className="px-8 gap-2"
        >
          {createOrder.isPending ? "创建订单中..." : <>立即购买 <ArrowRight className="h-4 w-4" /></>}
        </Button>

        {orderNo && (
          <div className="mt-4 p-4 bg-primary/5 border border-primary/20 rounded-lg max-w-md mx-auto">
            <div className="flex items-center justify-between gap-3 mb-2">
              <p className="text-sm font-medium text-foreground">订单已创建</p>
              <button
                onClick={() => { navigator.clipboard.writeText(orderNo); toast.success("订单号已复制"); }}
                className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                <Copy className="h-3.5 w-3.5" />复制
              </button>
            </div>
            <p className="font-mono text-sm text-primary mb-3 break-all">{orderNo}</p>
            <p className="text-xs text-muted-foreground mb-3">
              完成支付后，请前往订单查询页面查看兑换码
            </p>
            <Button
              variant="outline"
              size="sm"
              onClick={() => window.location.href = "/codex-order-query"}
            >
              查询订单
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
