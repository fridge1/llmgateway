import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Key, Plus, Trash2, Copy, Check } from "lucide-react";
import { apiGet, apiPost, apiDelete } from "@/lib/api-client";
import { toast } from "sonner";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
interface SubUserKey {
  id: string;
  name: string;
  key_prefix: string;
  status: string;
  last_used_at: string | null;
  created_at: string;
}

const SubUserKeysPage = () => {
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [newKeyName, setNewKeyName] = useState("");
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const { data: keys = [], isLoading } = useQuery({
    queryKey: ["sub-user-keys"],
    queryFn: () => apiGet<SubUserKey[]>("/api/sub-user/keys"),
  });

  const createMutation = useMutation({
    mutationFn: (name: string) => apiPost<{ id: string; key: string }>("/api/sub-user/keys", { name }),
    onSuccess: (data) => {
      setCreatedKey(data.key);
      setNewKeyName("");
      queryClient.invalidateQueries({ queryKey: ["sub-user-keys"] });
      toast.success("API 密钥创建成功");
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "创建失败");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => apiDelete(`/api/sub-user/keys/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sub-user-keys"] });
      toast.success("密钥已删除");
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "删除失败");
    },
  });

  const handleCopy = () => {
    if (createdKey) {
      navigator.clipboard.writeText(createdKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-foreground">API 密钥</h1>
        <button
          onClick={() => { setShowCreate(true); setCreatedKey(null); }}
          className="btn-primary flex items-center gap-1.5 px-4 py-2 text-sm"
        >
          <Plus size={14} />
          创建密钥
        </button>
      </div>

      {createdKey && (
        <div className="bg-emerald-50 dark:bg-emerald-950 border border-emerald-200 dark:border-emerald-800 rounded-xl p-4 mb-6">
          <p className="text-sm font-medium text-emerald-800 dark:text-emerald-200 mb-2">密钥已创建，请立即保存（仅显示一次）</p>
          <div className="flex items-center gap-2">
            <code className="flex-1 bg-white dark:bg-card rounded-lg px-3 py-2 text-sm font-mono text-foreground border border-border break-all">
              {createdKey}
            </code>
            <button onClick={handleCopy} className="p-2 rounded-lg hover:bg-muted/60 transition-colors">
              {copied ? <Check size={16} className="text-emerald-500" /> : <Copy size={16} className="text-muted-foreground" />}
            </button>
          </div>
        </div>
      )}

      {showCreate && !createdKey && (
        <div className="bg-card border border-border/60 rounded-xl p-4 mb-6">
          <h3 className="text-sm font-medium text-foreground mb-3">创建新密钥</h3>
          <div className="flex gap-3">
            <input
              className="input-field flex-1"
              placeholder="密钥名称（可选）"
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
            />
            <button
              onClick={() => createMutation.mutate(newKeyName)}
              disabled={createMutation.isPending}
              className="btn-primary px-4 py-2 text-sm disabled:opacity-60"
            >
              {createMutation.isPending ? "创建中..." : "确认创建"}
            </button>
            <button
              onClick={() => setShowCreate(false)}
              className="px-4 py-2 text-sm text-muted-foreground hover:text-foreground border border-border rounded-lg"
            >
              取消
            </button>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center py-12">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      ) : keys.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground">
          <Key size={48} className="mx-auto mb-4 opacity-30" />
          <p className="text-lg">暂无 API 密钥</p>
          <p className="text-sm mt-1">点击上方按钮创建你的第一个密钥</p>
        </div>
      ) : (
        <div className="bg-card border border-border/60 rounded-xl overflow-hidden">
          <Table className="w-full">
            <TableHeader>
              <TableRow className="border-b border-border/60">
                <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">名称</TableHead>
                <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">密钥前缀</TableHead>
                <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">创建时间</TableHead>
                <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase">最后使用</TableHead>
                <TableHead className="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((key) => (
                <TableRow key={key.id} className="border-b border-border/40 last:border-none hover:bg-muted/30 transition-colors">
                  <TableCell className="px-4 py-3 text-sm text-foreground">{key.name || "—"}</TableCell>
                  <TableCell className="px-4 py-3 text-sm font-mono text-muted-foreground">{key.key_prefix}...</TableCell>
                  <TableCell className="px-4 py-3 text-sm text-muted-foreground">
                    {new Date(key.created_at).toLocaleDateString("zh-CN")}
                  </TableCell>
                  <TableCell className="px-4 py-3 text-sm text-muted-foreground">
                    {key.last_used_at ? new Date(key.last_used_at).toLocaleDateString("zh-CN") : "从未使用"}
                  </TableCell>
                  <TableCell className="px-4 py-3 text-right">
                    <button
                      onClick={() => {
                        if (confirm("确定要删除此密钥？")) {
                          deleteMutation.mutate(key.id);
                        }
                      }}
                      className="p-1.5 rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                    >
                      <Trash2 size={14} />
                    </button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
};

export default SubUserKeysPage;
