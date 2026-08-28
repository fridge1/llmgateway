import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Key, Plus, Trash2, Copy, X, Loader2,
} from "lucide-react";
import {
  useTenantDetail,
  useTenantKeys,
  useCreateTenantKey,
  useDeleteTenantKey,
} from "@/hooks/use-tenant";
import { TenantPageHeader } from "@/components/tenant/TenantPageHeader";
import { toast } from "sonner";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
function CreateKeyModal({
  tenantId,
  onClose,
}: {
  tenantId: string;
  onClose: () => void;
}) {
  const [name, setName] = useState("");
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const create = useCreateTenantKey(tenantId);

  const handleSubmit = () => {
    if (!name.trim()) return;
    create.mutate(name.trim(), {
      onSuccess: (data) => {
        setCreatedKey(data.key);
      },
    });
  };

  const handleCopy = () => {
    if (createdKey) {
      navigator.clipboard.writeText(createdKey);
      toast.success("已复制到剪贴板");
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop" role="dialog" aria-modal="true" onClick={onClose}>
      <div
        className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-md overflow-y-auto rounded-xl border border-border bg-card p-4 shadow-modal slide-up sm:p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">创建 API 密钥</h3>
          <button onClick={onClose} className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg hover:bg-muted transition-colors" aria-label="关闭">
            <X size={18} />
          </button>
        </div>

        {!createdKey ? (
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">密钥名称</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
                className="input-field"
                placeholder="输入密钥名称"
                autoFocus
              />
            </div>
            <div className="flex justify-end gap-2">
              <button onClick={onClose} className="btn-secondary">
                取消
              </button>
              <button
                onClick={handleSubmit}
                disabled={create.isPending || !name.trim()}
                className="btn-primary disabled:opacity-50"
              >
                {create.isPending ? "创建中..." : "创建"}
              </button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
              <p className="text-xs text-amber-800 dark:text-amber-200 mb-2">
                请妥善保存此密钥，关闭后将无法再次查看
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 px-2 py-1 bg-background rounded text-xs font-mono break-all">
                  {createdKey}
                </code>
                <button
                  onClick={handleCopy}
                  className="p-1.5 hover:bg-muted rounded-lg transition-colors"
                  title="复制"
                >
                  <Copy size={14} />
                </button>
              </div>
            </div>
            <div className="flex justify-end">
              <button onClick={onClose} className="btn-primary">
                完成
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

const TenantKeysPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: tenant } = useTenantDetail(id!);
  const { data: keys, isLoading } = useTenantKeys(id!);
  const deleteKey = useDeleteTenantKey(id!);
  const [showCreate, setShowCreate] = useState(false);

  const handleDelete = (keyId: string, keyPrefix: string) => {
    if (!confirm(`确定要删除密钥「${keyPrefix}」吗？此操作不可恢复。`)) return;
    deleteKey.mutate(keyId);
  };

  if (isLoading) {
    return (
      <div className="page-container fade-in flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={28} className="animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">加载中...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="page-container fade-in">
      <TenantPageHeader
        title="API 密钥"
        description="管理组织级 API 访问凭证"
        tenantName={tenant?.name}
        icon={Key}
        onBack={() => navigate(`/dashboard/tenants/${id}`)}
        actions={
          <button
            onClick={() => setShowCreate(true)}
            className="btn-primary flex items-center gap-1.5"
          >
            <Plus size={14} />
            创建密钥
          </button>
        }
      />

      {/* Keys List */}
      {!keys || keys.length === 0 ? (
        <div className="empty-state border border-border rounded-xl shadow-card">
          <div className="w-14 h-14 bg-muted rounded-full flex items-center justify-center mb-4">
            <Key size={24} className="text-muted-foreground" />
          </div>
          <p className="text-sm font-medium text-muted-foreground mb-4">还没有创建任何密钥</p>
          <button
            onClick={() => setShowCreate(true)}
            className="btn-primary"
          >
            创建第一个密钥
          </button>
        </div>
      ) : (
        <div className="data-table-card">
          <Table className="w-full text-sm">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3">名称</TableHead>
                <TableHead className="text-left px-5 py-3">密钥前缀</TableHead>
                <TableHead className="text-left px-5 py-3">状态</TableHead>
                <TableHead className="text-left px-5 py-3">最后使用</TableHead>
                <TableHead className="text-left px-5 py-3">创建时间</TableHead>
                <TableHead className="text-right px-5 py-3">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((k) => (
                <TableRow key={k.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                  <TableCell className="px-5 py-3.5 font-medium">{k.name || "未命名"}</TableCell>
                  <TableCell className="px-5 py-3.5 font-mono text-xs">{k.key_prefix}</TableCell>
                  <TableCell className="px-5 py-3.5">
                    <span className={k.status === "active" ? "badge-success" : "badge-neutral"}>
                      {k.status === "active" ? "活跃" : k.status}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-muted-foreground">
                    {k.last_used_at
                      ? new Date(k.last_used_at).toLocaleString("zh-CN")
                      : "从未使用"}
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-muted-foreground">
                    {new Date(k.created_at).toLocaleString("zh-CN")}
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-right">
                    <button
                      onClick={() => handleDelete(k.id, k.key_prefix)}
                      className="p-1.5 rounded-lg hover:bg-muted text-destructive transition-colors"
                      title="删除"
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

      {showCreate && <CreateKeyModal tenantId={id!} onClose={() => setShowCreate(false)} />}
    </div>
  );
};

export default TenantKeysPage;
