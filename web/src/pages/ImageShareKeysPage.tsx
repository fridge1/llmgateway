import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Copy, Check, RotateCcw, Power, Trash2, AlertTriangle, Image as ImageIcon } from "lucide-react";
import { apiGet, apiPost, apiPatch, apiDelete } from "@/lib/api-client";
import type { ImageShareKey } from "@/types/api";
import { useAuth } from "@/contexts/AuthContext";
import { PageHeader } from "@/components/ui/page-header";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const QK = ["image-share-keys"] as const;

function useImageShareKeys() {
  return useQuery<ImageShareKey[]>({
    queryKey: QK,
    queryFn: () => apiGet<{ keys: ImageShareKey[] }>("/api/image-share/keys").then((r) => r.keys ?? []),
  });
}

function useCreateImageShareKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; quota_total: number }) =>
      apiPost<{ key: string; id: string; name: string; quota_total: number }>("/api/image-share/keys", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: QK }),
  });
}

function usePatchImageShareKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: Record<string, unknown> }) =>
      apiPatch<ImageShareKey>(`/api/image-share/keys/${id}`, patch),
    onSuccess: () => qc.invalidateQueries({ queryKey: QK }),
  });
}

function useDeleteImageShareKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete<void>(`/api/image-share/keys/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: QK }),
  });
}

export default function ImageShareKeysPage() {
  const auth = useAuth();
  const enabled = auth.user?.image_share_enabled === true;
  const { data: keys = [], isLoading } = useImageShareKeys();
  const createMut = useCreateImageShareKey();
  const patchMut = usePatchImageShareKey();
  const deleteMut = useDeleteImageShareKey();

  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newQuota, setNewQuota] = useState("100");
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");

  if (!enabled) {
    return (
      <div className="page-container fade-in">
        <PageHeader
          eyebrow="图片服务"
          title={<span className="flex items-center gap-2.5"><ImageIcon size={20} className="text-primary" />图片分发密钥</span>}
          description="创建独立子密钥并管理图片生成配额。"
        />
        <div className="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-amber-800 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-200">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" />
          <div>
            <p className="font-medium">未开通图片分发权限</p>
            <p className="text-sm">请联系管理员为你的账号开启此功能。</p>
          </div>
        </div>
      </div>
    );
  }

  const handleCreate = async () => {
    setError("");
    const quota = parseInt(newQuota, 10);
    if (!quota || quota <= 0) {
      setError("张数必须大于 0");
      return;
    }
    try {
      const res = await createMut.mutateAsync({ name: newName.trim() || "未命名", quota_total: quota });
      setCreatedKey(res.key);
      setNewName("");
      setNewQuota("100");
      setShowCreate(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : "创建失败");
    }
  };

  const handleCopy = async (text: string) => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleToggleStatus = (k: ImageShareKey) =>
    patchMut.mutate({ id: k.id, patch: { status: k.status === "active" ? "disabled" : "active" } });

  const handleResetQuota = (k: ImageShareKey) => {
    if (!confirm(`确定将 "${k.name}" 的已用配额重置为 0 吗？`)) return;
    patchMut.mutate({ id: k.id, patch: { reset_used: true } });
  };

  const handleDelete = (k: ImageShareKey) => {
    if (!confirm(`确定删除密钥 "${k.name}" 吗？此操作不可恢复。`)) return;
    deleteMut.mutate(k.id);
  };

  return (
    <div className="page-container space-y-6 fade-in">
      <PageHeader
        eyebrow="图片服务"
        title={<span className="flex items-center gap-2.5"><ImageIcon size={20} className="text-primary" />图片分发密钥</span>}
        description="创建子密钥分发给他人使用图片生成功能，费用从你的账户扣除。"
        actions={
          <button onClick={() => setShowCreate(true)} className="btn-primary">
            <Plus className="h-4 w-4" />
            新建密钥
          </button>
        }
      />

      {createdKey && (
        <div className="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-900 dark:bg-green-950">
          <div className="flex flex-col items-start justify-between gap-4 sm:flex-row">
            <div className="flex-1">
              <p className="font-medium text-green-800 dark:text-green-200">密钥已创建，请立即保存</p>
              <p className="mt-1 text-xs text-green-700 dark:text-green-300">关闭后将无法再次查看完整密钥。</p>
              <code className="mt-2 block break-all rounded bg-background px-2 py-1.5 font-mono text-sm">{createdKey}</code>
            </div>
            <div className="flex gap-2 sm:flex-col">
              <button
                onClick={() => handleCopy(createdKey)}
                className="inline-flex items-center gap-1 rounded-md border bg-background px-3 py-1.5 text-xs hover:bg-muted"
              >
                {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                {copied ? "已复制" : "复制"}
              </button>
              <button
                onClick={() => setCreatedKey(null)}
                className="inline-flex items-center gap-1 rounded-md border bg-background px-3 py-1.5 text-xs hover:bg-muted"
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}

      {showCreate && (
        <div className="rounded-lg border bg-card p-4">
          <h3 className="mb-3 font-medium">新建密钥</h3>
          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <label className="text-xs text-muted-foreground">备注名称</label>
              <input
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="例如：客户A"
                className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm outline-none focus:border-primary"
              />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">总张数配额</label>
              <input
                type="number"
                min={1}
                value={newQuota}
                onChange={(e) => setNewQuota(e.target.value)}
                className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm outline-none focus:border-primary"
              />
            </div>
          </div>
          {error && <p className="mt-2 text-sm text-destructive">{error}</p>}
          <div className="mt-4 flex justify-end gap-2">
            <button
              onClick={() => setShowCreate(false)}
              className="rounded-md border px-3 py-1.5 text-sm hover:bg-muted"
            >
              取消
            </button>
            <button
              onClick={handleCreate}
              disabled={createMut.isPending}
              className="rounded-md bg-primary px-3 py-1.5 text-sm text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {createMut.isPending ? "创建中…" : "创建"}
            </button>
          </div>
        </div>
      )}

      <div className="data-table-card">
        <Table className="w-full text-sm">
            <TableHeader className="border-b bg-muted/50 text-xs uppercase text-muted-foreground">
              <TableRow>
                <TableHead className="px-4 py-2 text-left font-medium">备注</TableHead>
                <TableHead className="px-4 py-2 text-left font-medium">前缀</TableHead>
                <TableHead className="px-4 py-2 text-left font-medium">已用 / 总数</TableHead>
                <TableHead className="px-4 py-2 text-left font-medium">状态</TableHead>
                <TableHead className="px-4 py-2 text-left font-medium">最后使用</TableHead>
                <TableHead className="px-4 py-2 text-right font-medium">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading && (
                <TableRow>
                  <TableCell colSpan={6} className="px-4 py-6 text-center text-muted-foreground">
                    加载中…
                  </TableCell>
                </TableRow>
              )}
              {!isLoading && keys.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                    暂无密钥，点击右上方按钮创建。
                  </TableCell>
                </TableRow>
              )}
              {keys.map((k) => {
                const remaining = k.quota_total - k.quota_used;
                const exhausted = remaining <= 0;
                return (
                  <TableRow key={k.id} className="border-b last:border-b-0">
                    <TableCell className="px-4 py-2.5">{k.name}</TableCell>
                    <TableCell className="px-4 py-2.5 font-mono text-xs">{k.key_prefix}***</TableCell>
                    <TableCell className="px-4 py-2.5">
                      <span className={exhausted ? "text-destructive" : ""}>{k.quota_used}</span>
                      <span className="text-muted-foreground"> / {k.quota_total}</span>
                    </TableCell>
                    <TableCell className="px-4 py-2.5">
                      <span
                        className={
                          k.status === "active"
                            ? "rounded-full bg-green-100 px-2 py-0.5 text-xs text-green-800 dark:bg-green-900 dark:text-green-200"
                            : "rounded-full bg-zinc-100 px-2 py-0.5 text-xs text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300"
                        }
                      >
                        {k.status === "active" ? "可用" : "已禁用"}
                      </span>
                    </TableCell>
                    <TableCell className="px-4 py-2.5 text-xs text-muted-foreground">
                      {k.last_used_at ? new Date(k.last_used_at).toLocaleString() : "未使用"}
                    </TableCell>
                    <TableCell className="px-4 py-2.5 text-right">
                      <div className="inline-flex gap-1">
                        <button
                          onClick={() => handleToggleStatus(k)}
                          title={k.status === "active" ? "禁用" : "启用"}
                          className="flex h-9 w-9 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
                          aria-label={k.status === "active" ? "禁用密钥" : "启用密钥"}
                        >
                          <Power className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleResetQuota(k)}
                          title="重置已用配额"
                          className="flex h-9 w-9 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
                          aria-label="重置已用配额"
                        >
                          <RotateCcw className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(k)}
                          title="删除"
                          className="flex h-9 w-9 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-destructive"
                          aria-label="删除密钥"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
              </div>
    </div>
  );
}
