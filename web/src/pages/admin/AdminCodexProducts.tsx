import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Package, Loader, Plus, Pencil, Trash2, X, Check, Inbox } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
interface CodexProduct {
  id: number;
  sku: string;
  name: string;
  description: string;
  price_cny: number;
  sort_order: number;
  status: string;
  created_at: string;
  updated_at: string;
}

const fmtCny = (n: number) => `¥${n.toFixed(2)}`;

const emptyForm = { sku: "", name: "", description: "", price_cny: "", sort_order: "0", status: "active" };

export default function AdminCodexProducts() {
  const queryClient = useQueryClient();
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState(emptyForm);

  const { data, isLoading, error } = useQuery({
    queryKey: ["admin", "codex", "products"],
    queryFn: async () => {
      const res = await fetch("/api/admin/codex/products", { credentials: "include" });
      if (!res.ok) {
        const e = await res.json().catch(() => ({}));
        throw new Error(e?.error?.message || "加载失败");
      }
      return res.json() as Promise<{ products: CodexProduct[] }>;
    },
  });

  const createMutation = useMutation({
    mutationFn: async (p: Omit<CodexProduct, "id" | "created_at" | "updated_at">) => {
      const res = await fetch("/api/admin/codex/products", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(p),
      });
      if (!res.ok) {
        const e = await res.json().catch(() => ({}));
        throw new Error(e?.error?.message || "创建失败");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("商品已创建");
      queryClient.invalidateQueries({ queryKey: ["admin", "codex", "products"] });
      resetForm();
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, ...p }: Omit<CodexProduct, "created_at" | "updated_at">) => {
      const res = await fetch(`/api/admin/codex/products/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(p),
      });
      if (!res.ok) {
        const e = await res.json().catch(() => ({}));
        throw new Error(e?.error?.message || "更新失败");
      }
      return res.json();
    },
    onSuccess: () => {
      toast.success("商品已更新");
      queryClient.invalidateQueries({ queryKey: ["admin", "codex", "products"] });
      resetForm();
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      const res = await fetch(`/api/admin/codex/products/${id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!res.ok && res.status !== 204) {
        const e = await res.json().catch(() => ({}));
        throw new Error(e?.error?.message || "删除失败");
      }
    },
    onSuccess: () => {
      toast.success("商品已删除");
      queryClient.invalidateQueries({ queryKey: ["admin", "codex", "products"] });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const resetForm = () => {
    setForm(emptyForm);
    setAdding(false);
    setEditingId(null);
  };

  const startEdit = (p: CodexProduct) => {
    setForm({
      sku: p.sku,
      name: p.name,
      description: p.description,
      price_cny: String(p.price_cny),
      sort_order: String(p.sort_order),
      status: p.status,
    });
    setEditingId(p.id);
    setAdding(false);
  };

  const submit = () => {
    if (!form.sku.trim() || !form.name.trim()) {
      toast.error("SKU 和名称必填");
      return;
    }
    const price = Number(form.price_cny);
    if (!(price > 0)) {
      toast.error("价格必须大于 0");
      return;
    }
    const payload = {
      sku: form.sku.trim(),
      name: form.name.trim(),
      description: form.description,
      price_cny: price,
      sort_order: Number(form.sort_order) || 0,
      status: form.status,
    };
    if (editingId !== null) {
      updateMutation.mutate({ ...payload, id: editingId });
    } else {
      createMutation.mutate(payload);
    }
  };

  const products = data?.products ?? [];

  return (
    <div className="container mx-auto px-4 py-8">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Package size={18} className="text-muted-foreground" />
              <CardTitle>Codex 商品管理</CardTitle>
            </div>
            <Button size="sm" onClick={() => { setAdding(true); setEditingId(null); }} disabled={adding || editingId !== null}>
              <Plus size={14} />添加商品
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoading ? (
            <div className="flex justify-center py-8"><Loader size={18} className="animate-spin text-muted-foreground" /></div>
          ) : error ? (
            <div className="text-center py-8 text-destructive">{(error as Error).message}</div>
          ) : (
            <>
              {(adding || editingId !== null) && (
                <div className="border border-border rounded-lg p-4 space-y-3 bg-muted/30">
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                    <div>
                      <Label className="text-xs text-muted-foreground">SKU *</Label>
                      <Input value={form.sku} onChange={e => setForm({ ...form, sku: e.target.value })} placeholder="gpt-pro-20x" />
                    </div>
                    <div>
                      <Label className="text-xs text-muted-foreground">名称 *</Label>
                      <Input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} />
                    </div>
                    <div>
                      <Label className="text-xs text-muted-foreground">价格（元）*</Label>
                      <Input type="number" min={0} step={0.01} value={form.price_cny} onChange={e => setForm({ ...form, price_cny: e.target.value })} />
                    </div>
                    <div>
                      <Label className="text-xs text-muted-foreground">排序</Label>
                      <Input type="number" value={form.sort_order} onChange={e => setForm({ ...form, sort_order: e.target.value })} />
                    </div>
                    <div>
                      <Label className="text-xs text-muted-foreground">状态</Label>
                      <select className="w-full h-9 px-3 rounded-lg border border-border bg-background text-sm" value={form.status} onChange={e => setForm({ ...form, status: e.target.value })}>
                        <option value="active">启用</option>
                        <option value="inactive">停用</option>
                      </select>
                    </div>
                    <div className="col-span-2 md:col-span-3">
                      <Label className="text-xs text-muted-foreground">描述</Label>
                      <Input value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} />
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" onClick={submit} disabled={createMutation.isPending || updateMutation.isPending}>
                      <Check size={14} />{editingId !== null ? "保存" : "创建"}
                    </Button>
                    <Button size="sm" variant="outline" onClick={resetForm}><X size={14} />取消</Button>
                  </div>
                </div>
              )}

              {products.length === 0 && !adding ? (
                <div className="border border-dashed border-border rounded-lg py-10 text-center">
                  <Inbox className="h-8 w-8 text-muted-foreground/50 mx-auto mb-2" />
                  <p className="text-sm text-muted-foreground">暂无商品，点击「添加商品」开始配置</p>
                </div>
              ) : (
                <div className="border border-border rounded-lg overflow-hidden">
                  <Table className="w-full text-sm">
                    <TableHeader className="bg-muted/40">
                      <TableRow>
                        {["SKU", "名称", "价格", "排序", "状态", "操作"].map(h => (
                          <TableHead key={h} className="text-left px-4 py-2 text-xs font-medium text-muted-foreground">{h}</TableHead>
                        ))}
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {products.map(p => (
                        <TableRow key={p.id} className="border-t border-border hover:bg-muted/20">
                          <TableCell className="px-4 py-3 font-mono text-xs">{p.sku}</TableCell>
                          <TableCell className="px-4 py-3">{p.name}</TableCell>
                          <TableCell className="px-4 py-3 font-bold">{fmtCny(p.price_cny)}</TableCell>
                          <TableCell className="px-4 py-3">{p.sort_order}</TableCell>
                          <TableCell className="px-4 py-3">
                            <Badge variant={p.status === "active" ? "default" : "secondary"}>
                              {p.status === "active" ? "启用" : "停用"}
                            </Badge>
                          </TableCell>
                          <TableCell className="px-4 py-3">
                            <div className="flex gap-1">
                              <button onClick={() => startEdit(p)} className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground" title="编辑">
                                <Pencil size={14} />
                              </button>
                              <button
                                onClick={() => { if (confirm(`确认删除商品「${p.name}」？仍有待支付/已支付订单时不可删除。`)) deleteMutation.mutate(p.id); }}
                                className="p-1.5 rounded-lg hover:bg-red-50 dark:hover:bg-red-500/10 text-muted-foreground hover:text-destructive"
                                title="删除"
                              >
                                <Trash2 size={14} />
                              </button>
                            </div>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
