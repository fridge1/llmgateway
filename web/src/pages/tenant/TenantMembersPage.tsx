import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Users, Plus, Trash2, KeyRound, Edit3, X, Loader2, Copy, Check, Receipt,
} from "lucide-react";
import {
  useTenantDetail,
  useTenantSubUsers,
  useCreateSubUser,
  useDeleteSubUser,
  useResetSubUserPassword,
  useUpdateSubUserQuota,
} from "@/hooks/use-tenant";
import { TenantPageHeader } from "@/components/tenant/TenantPageHeader";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
function CreateSubUserModal({
  tenantId,
  onClose,
}: {
  tenantId: string;
  onClose: () => void;
}) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [nickname, setNickname] = useState("");
  const [quotaLimit, setQuotaLimit] = useState("");
  const create = useCreateSubUser(tenantId);

  const handleSubmit = () => {
    if (!username.trim() || !password.trim()) return;
    create.mutate(
      {
        username: username.trim(),
        password: password.trim(),
        nickname: nickname.trim() || undefined,
        quota_limit: quotaLimit ? parseFloat(quotaLimit) : null,
      },
      { onSuccess: () => onClose() }
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop" role="dialog" aria-modal="true" onClick={onClose}>
      <div
        className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-md overflow-y-auto rounded-xl border border-border bg-card p-4 shadow-modal slide-up sm:p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">创建子用户</h3>
          <button onClick={onClose} className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg hover:bg-muted transition-colors" aria-label="关闭">
            <X size={18} />
          </button>
        </div>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">用户名 *</label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="input-field"
              placeholder="输入用户名"
              autoFocus
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">密码 *</label>
            <input
              type="text"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="input-field"
              placeholder="输入密码"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">昵称</label>
            <input
              type="text"
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              className="input-field"
              placeholder="选填"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">额度限制（元）</label>
            <input
              type="number"
              value={quotaLimit}
              onChange={(e) => setQuotaLimit(e.target.value)}
              className="input-field"
              placeholder="留空为不限"
              min="0"
              step="0.01"
            />
            <p className="text-xs text-muted-foreground mt-1">留空表示不限制额度</p>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button onClick={onClose} className="btn-secondary">
              取消
            </button>
            <button
              onClick={handleSubmit}
              disabled={!username.trim() || !password.trim() || create.isPending}
              className="btn-primary disabled:opacity-50"
            >
              {create.isPending ? "创建中..." : "创建"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function ResetPasswordModal({
  tenantId,
  subUserId,
  username,
  onClose,
}: {
  tenantId: string;
  subUserId: string;
  username: string;
  onClose: () => void;
}) {
  const [password, setPassword] = useState("");
  const reset = useResetSubUserPassword(tenantId);

  const handleSubmit = () => {
    if (!password.trim()) return;
    reset.mutate(
      { subUserId, password: password.trim() },
      { onSuccess: () => onClose() }
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop" role="dialog" aria-modal="true" onClick={onClose}>
      <div
        className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-sm overflow-y-auto rounded-xl border border-border bg-card p-4 shadow-modal slide-up sm:p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">重置密码</h3>
          <button onClick={onClose} className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg hover:bg-muted transition-colors" aria-label="关闭">
            <X size={18} />
          </button>
        </div>
        <p className="text-sm text-muted-foreground mb-4">为用户 <span className="font-medium text-foreground">{username}</span> 设置新密码</p>
        <div className="space-y-4">
          <input
            type="text"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="input-field"
            placeholder="输入新密码"
            autoFocus
          />
          <div className="flex justify-end gap-2">
            <button onClick={onClose} className="btn-secondary">取消</button>
            <button
              onClick={handleSubmit}
              disabled={!password.trim() || reset.isPending}
              className="btn-primary disabled:opacity-50"
            >
              {reset.isPending ? "重置中..." : "确认重置"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function UpdateQuotaModal({
  tenantId,
  subUserId,
  username,
  currentQuota,
  onClose,
}: {
  tenantId: string;
  subUserId: string;
  username: string;
  currentQuota: number | null;
  onClose: () => void;
}) {
  const [quotaLimit, setQuotaLimit] = useState(currentQuota !== null ? String(currentQuota) : "");
  const update = useUpdateSubUserQuota(tenantId);

  const handleSubmit = () => {
    update.mutate(
      { subUserId, quota_limit: quotaLimit ? parseFloat(quotaLimit) : null },
      { onSuccess: () => onClose() }
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop" role="dialog" aria-modal="true" onClick={onClose}>
      <div
        className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-sm overflow-y-auto rounded-xl border border-border bg-card p-4 shadow-modal slide-up sm:p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">修改额度</h3>
          <button onClick={onClose} className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg hover:bg-muted transition-colors" aria-label="关闭">
            <X size={18} />
          </button>
        </div>
        <p className="text-sm text-muted-foreground mb-4">修改用户 <span className="font-medium text-foreground">{username}</span> 的额度限制</p>
        <div className="space-y-4">
          <div>
            <input
              type="number"
              value={quotaLimit}
              onChange={(e) => setQuotaLimit(e.target.value)}
              className="input-field"
              placeholder="留空为不限"
              min="0"
              step="0.01"
              autoFocus
            />
            <p className="text-xs text-muted-foreground mt-1">留空表示不限制额度，单位：元</p>
          </div>
          <div className="flex justify-end gap-2">
            <button onClick={onClose} className="btn-secondary">取消</button>
            <button
              onClick={handleSubmit}
              disabled={update.isPending}
              className="btn-primary disabled:opacity-50"
            >
              {update.isPending ? "保存中..." : "保存"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

const TenantMembersPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: tenant } = useTenantDetail(id!);
  const { data: subUsers, isLoading } = useTenantSubUsers(id!);
  const deleteSubUser = useDeleteSubUser(id!);

  const [showCreate, setShowCreate] = useState(false);
  const [resetTarget, setResetTarget] = useState<{ id: string; username: string } | null>(null);
  const [quotaTarget, setQuotaTarget] = useState<{ id: string; username: string; quota: number | null } | null>(null);
  const [copied, setCopied] = useState(false);

  const handleDelete = (subUser: { id: string; username: string }) => {
    if (!confirm(`确定要删除子用户「${subUser.username}」吗？删除后该用户将无法登录。`)) return;
    deleteSubUser.mutate(subUser.id);
  };

  const loginUrl = `${window.location.origin}/org/login`;

  const handleCopyId = () => {
    navigator.clipboard.writeText(id!);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
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
        title="子用户管理"
        description="管理组织成员、登录信息与用量额度"
        tenantName={tenant?.name}
        icon={Users}
        onBack={() => navigate(`/dashboard/tenants/${id}`)}
        actions={
          <button
            onClick={() => setShowCreate(true)}
            className="btn-primary flex items-center gap-1.5"
          >
            <Plus size={14} />
            创建子用户
          </button>
        }
      />

      {/* Login Info */}
      <div className="bg-card border border-border rounded-xl p-4 shadow-card mb-6">
        <p className="text-sm font-medium mb-2">子用户登录信息</p>
        <div className="flex flex-col gap-1.5 text-sm text-muted-foreground">
          <div className="flex items-center gap-2">
            <span>登录地址：</span>
            <code className="px-1.5 py-0.5 bg-muted rounded text-xs font-mono">{loginUrl}</code>
          </div>
          <div className="flex items-center gap-2">
            <span>组织 ID：</span>
            <code className="px-1.5 py-0.5 bg-muted rounded text-xs font-mono">{id}</code>
            <button
              onClick={handleCopyId}
              className="flex h-9 w-9 items-center justify-center rounded hover:bg-muted transition-colors"
              title="复制组织 ID"
            >
              {copied ? <Check size={12} className="text-green-500" /> : <Copy size={12} />}
            </button>
          </div>
        </div>
      </div>

      {/* Sub-Users List */}
      {!subUsers || subUsers.length === 0 ? (
        <div className="empty-state border border-border rounded-xl shadow-card">
          <div className="w-14 h-14 bg-muted rounded-full flex items-center justify-center mb-4">
            <Users size={24} className="text-muted-foreground" />
          </div>
          <p className="text-sm font-medium text-muted-foreground">暂无子用户</p>
          <p className="text-xs text-muted-foreground/70 mt-1">点击「创建子用户」添加第一个子用户</p>
        </div>
      ) : (
        <div className="data-table-card">
          <Table className="w-full text-sm">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3">用户名</TableHead>
                <TableHead className="text-left px-5 py-3">昵称</TableHead>
                <TableHead className="text-left px-5 py-3">额度</TableHead>
                <TableHead className="text-left px-5 py-3">状态</TableHead>
                <TableHead className="text-left px-5 py-3">创建时间</TableHead>
                <TableHead className="text-right px-5 py-3">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {subUsers.map((su) => (
                <TableRow key={su.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                  <TableCell className="px-5 py-3.5 font-medium">{su.username}</TableCell>
                  <TableCell className="px-5 py-3.5 text-muted-foreground">{su.nickname || "-"}</TableCell>
                  <TableCell className="px-5 py-3.5">
                    <span className="font-mono text-xs">
                      ¥{su.quota_used.toFixed(2)}
                      {su.quota_limit !== null ? ` / ¥${su.quota_limit.toFixed(2)}` : " / 不限"}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-3.5">
                    <span className={su.status === "active" ? "badge-success" : "badge-neutral"}>
                      {su.status === "active" ? "正常" : "禁用"}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-muted-foreground">
                    {new Date(su.created_at).toLocaleDateString("zh-CN")}
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <button
                        onClick={() => navigate(`/dashboard/tenants/${id}/members/${su.id}/transactions`)}
                        className="p-1.5 rounded-lg hover:bg-muted transition-colors"
                        title="查看记录"
                      >
                        <Receipt size={14} />
                      </button>
                      <button
                        onClick={() => setQuotaTarget({ id: su.id, username: su.username, quota: su.quota_limit })}
                        className="p-1.5 rounded-lg hover:bg-muted transition-colors"
                        title="修改额度"
                      >
                        <Edit3 size={14} />
                      </button>
                      <button
                        onClick={() => setResetTarget({ id: su.id, username: su.username })}
                        className="p-1.5 rounded-lg hover:bg-muted transition-colors"
                        title="重置密码"
                      >
                        <KeyRound size={14} />
                      </button>
                      <button
                        onClick={() => handleDelete(su)}
                        className="p-1.5 rounded-lg hover:bg-muted text-destructive transition-colors"
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

      {/* Modals */}
      {showCreate && <CreateSubUserModal tenantId={id!} onClose={() => setShowCreate(false)} />}
      {resetTarget && (
        <ResetPasswordModal
          tenantId={id!}
          subUserId={resetTarget.id}
          username={resetTarget.username}
          onClose={() => setResetTarget(null)}
        />
      )}
      {quotaTarget && (
        <UpdateQuotaModal
          tenantId={id!}
          subUserId={quotaTarget.id}
          username={quotaTarget.username}
          currentQuota={quotaTarget.quota}
          onClose={() => setQuotaTarget(null)}
        />
      )}
    </div>
  );
};

export default TenantMembersPage;
