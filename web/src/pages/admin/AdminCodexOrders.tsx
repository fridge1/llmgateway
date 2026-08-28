import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface AdminCodexOrder {
  id: string;
  order_no: string;
  product: { name: string; sku: string };
  contact_info: string;
  amount: number;
  status: string;
  pay_time?: string;
  redemption_code?: string;
  created_at: string;
}

const statusMap: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  pending: { label: "待支付", variant: "outline" },
  paid: { label: "已支付", variant: "default" },
  shipped: { label: "已发货", variant: "default" },
  completed: { label: "已完成", variant: "secondary" },
  cancelled: { label: "已取消", variant: "destructive" },
  refunded: { label: "已退款", variant: "destructive" },
};

export default function AdminCodexOrders() {
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("all");
  const [shipDialogOpen, setShipDialogOpen] = useState(false);
  const [selectedOrder, setSelectedOrder] = useState<AdminCodexOrder | null>(null);
  const [redemptionCode, setRedemptionCode] = useState("");
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["admin", "codex", "orders", page, statusFilter],
    queryFn: async () => {
      const params = new URLSearchParams({
        page: page.toString(),
        size: "20",
        ...(statusFilter !== "all" && { status: statusFilter }),
      });
      const res = await fetch(`/api/admin/codex/orders?${params}`, {
        credentials: "include",
      });
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        throw new Error(errorData.error?.message || "Failed to fetch orders");
      }
      return res.json() as Promise<{
        orders: AdminCodexOrder[];
        total: number;
        page: number;
        size: number;
      }>;
    },
  });

  const shipMutation = useMutation({
    mutationFn: async ({ orderNo, code }: { orderNo: string; code: string }) => {
      const res = await fetch(`/api/admin/codex/orders/${orderNo}/ship`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ redemption_code: code }),
      });
      if (!res.ok) {
        const error = await res.json();
        throw new Error(error.error?.message || "发货失败");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("发货成功");
      setShipDialogOpen(false);
      setRedemptionCode("");
      queryClient.invalidateQueries({ queryKey: ["admin", "codex", "orders"] });
    },
    onError: (error: Error) => {
      toast.error(error.message || "发货失败");
    },
  });

  const handleShip = () => {
    if (!selectedOrder || !redemptionCode.trim()) {
      toast.error("请填写兑换码");
      return;
    }
    shipMutation.mutate({ orderNo: selectedOrder.order_no, code: redemptionCode.trim() });
  };

  const orders = data?.orders ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / 20);

  return (
    <div className="container mx-auto px-4 py-8">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Codex 订单管理</CardTitle>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="全部状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="pending">待支付</SelectItem>
                <SelectItem value="paid">已支付</SelectItem>
                <SelectItem value="shipped">已发货</SelectItem>
                <SelectItem value="completed">已完成</SelectItem>
                <SelectItem value="refunded">已退款</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="text-center py-8">加载中...</div>
          ) : error ? (
            <div className="text-center py-8">
              <div className="text-destructive mb-2">加载失败</div>
              <div className="text-sm text-muted-foreground">{(error as Error).message}</div>
            </div>
          ) : orders.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">暂无订单</div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>订单号</TableHead>
                    <TableHead>商品</TableHead>
                    <TableHead>联系方式</TableHead>
                    <TableHead>金额</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>创建时间</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {orders.map((order) => (
                    <TableRow key={order.id}>
                      <TableCell className="font-mono text-xs">{order.order_no}</TableCell>
                      <TableCell>{order.product.name}</TableCell>
                      <TableCell className="text-sm">{order.contact_info}</TableCell>
                      <TableCell className="font-bold">¥{order.amount}</TableCell>
                      <TableCell>
                        <Badge variant={statusMap[order.status]?.variant || "outline"}>
                          {statusMap[order.status]?.label || order.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm">
                        {new Date(order.created_at).toLocaleString("zh-CN")}
                      </TableCell>
                      <TableCell>
                        {order.status === "paid" && (
                          <Button
                            size="sm"
                            onClick={() => {
                              setSelectedOrder(order);
                              setShipDialogOpen(true);
                            }}
                          >
                            发货
                          </Button>
                        )}
                        {order.status === "shipped" && order.redemption_code && (
                          <div className="text-xs font-mono text-muted-foreground max-w-32 truncate">
                            {order.redemption_code}
                          </div>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              <div className="flex items-center justify-between mt-4">
                <div className="text-sm text-muted-foreground">
                  共 {total} 条，第 {page} / {totalPages} 页
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page <= 1}
                    onClick={() => setPage(page - 1)}
                  >
                    上一页
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page >= totalPages}
                    onClick={() => setPage(page + 1)}
                  >
                    下一页
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <Dialog open={shipDialogOpen} onOpenChange={setShipDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>发货</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label className="text-sm text-muted-foreground">订单号</Label>
              <div className="font-mono text-sm">{selectedOrder?.order_no}</div>
            </div>
            <div>
              <Label className="text-sm text-muted-foreground">商品</Label>
              <div>{selectedOrder?.product.name}</div>
            </div>
            <div>
              <Label htmlFor="redemption_code">兑换码/账号信息 *</Label>
              <Input
                id="redemption_code"
                value={redemptionCode}
                onChange={(e) => setRedemptionCode(e.target.value)}
                placeholder="请输入兑换码或账号密码信息"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShipDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleShip} disabled={shipMutation.isPending}>
              {shipMutation.isPending ? "发货中..." : "确认发货"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
