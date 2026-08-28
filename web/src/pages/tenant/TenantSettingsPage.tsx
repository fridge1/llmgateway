import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Settings, Trash2, X, Loader2, AlertTriangle, ArrowRightLeft,
} from "lucide-react";
import {
  useTenantDetail,
  useTenantMembers,
  useUpdateTenant,
  useDeleteTenant,
  useTransferOwnership,
  type TenantMember,
} from "@/hooks/use-tenant";
import { TenantPageHeader } from "@/components/tenant/TenantPageHeader";
import { useAuth } from "@/contexts/AuthContext";

function TransferModal({
  tenantId,
  members,
  onClose,
}: {
  tenantId: string;
  members: TenantMember[];
  onClose: () => void;
}) {
  const [selectedUserId, setSelectedUserId] = useState("");
  const transfer = useTransferOwnership(tenantId);
  const navigate = useNavigate();

  const handleSubmit = () => {
    if (!selectedUserId) return;
    if (!confirm("确定要转让所有权吗？转让后你将失去所有者权限。")) return;
    transfer.mutate(selectedUserId, {
      onSuccess: () => {
        onClose();
        navigate("/dashboard/tenants");
      },
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop" role="dialog" aria-modal="true" onClick={onClose}>
      <div
        className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-md overflow-y-auto rounded-xl border border-border bg-card p-4 shadow-modal slide-up sm:p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">转让所有权</h3>
          <button onClick={onClose} className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg hover:bg-muted transition-colors" aria-label="关闭">
            <X size={18} />
          </button>
        </div>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">选择新所有者</label>
            <select
              value={selectedUserId}
              onChange={(e) => setSelectedUserId(e.target.value)}
              className="input-field"
            >
              <option value="">请选择成员</option>
              {members
                .filter((m) => m.role !== "owner")
                .map((m) => (
                  <option key={m.user_id} value={m.user_id}>
                    {m.phone || m.user_id} ({m.role === "admin" ? "管理员" : "成员"})
                  </option>
                ))}
            </select>
          </div>
          <div className="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
            <p className="text-xs text-amber-800 dark:text-amber-200">
              转让后你将失去所有者权限，此操作不可撤销
            </p>
          </div>
          <div className="flex justify-end gap-2">
            <button onClick={onClose} className="btn-secondary">
              取消
            </button>
            <button
              onClick={handleSubmit}
              disabled={transfer.isPending || !selectedUserId}
              className="px-4 py-2 text-sm font-semibold bg-destructive text-destructive-foreground rounded-lg hover:bg-destructive/90 disabled:opacity-50 transition-colors"
            >
              {transfer.isPending ? "转让中..." : "确认转让"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

const TenantSettingsPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const auth = useAuth();
  const { data: tenant, isLoading } = useTenantDetail(id!);
  const { data: members } = useTenantMembers(id!);
  const updateTenant = useUpdateTenant(id!);
  const deleteTenant = useDeleteTenant(id!);
  const [name, setName] = useState("");
  const [showTransfer, setShowTransfer] = useState(false);

  const currentMember = members?.find((m) => m.user_id === auth.user?.user_id);
  const isOwner = currentMember?.role === "owner";
  const isAdmin = currentMember?.role === "admin" || isOwner;

  const handleUpdateName = () => {
    if (!name.trim() || name === tenant?.name) return;
    updateTenant.mutate(name.trim());
  };

  const handleDelete = () => {
    if (!confirm(`确定要删除组织「${tenant?.name}」吗？此操作不可恢复，所有数据将被永久删除。`)) return;
    deleteTenant.mutate(undefined, {
      onSuccess: () => {
        navigate("/dashboard/tenants");
      },
    });
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

  if (!tenant) {
    return (
      <div className="page-container fade-in">
        <div className="empty-state">
          <p className="text-sm font-medium text-muted-foreground">组织不存在</p>
        </div>
      </div>
    );
  }

  return (
    <div className="page-container fade-in">
      <TenantPageHeader
        title="组织设置"
        description="管理组织名称、密钥策略与危险操作"
        tenantName={tenant.name}
        icon={Settings}
        onBack={() => navigate(`/dashboard/tenants/${id}`)}
      />

      <div className="space-y-6">
        {/* General Settings */}
        {isAdmin && (
          <div className="bg-card border border-border rounded-xl p-5 shadow-card">
            <h3 className="text-sm font-semibold text-foreground mb-4">基本信息</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">组织名称</label>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={name || tenant.name}
                    onChange={(e) => setName(e.target.value)}
                    className="input-field flex-1"
                    placeholder="输入组织名称"
                  />
                  <button
                    onClick={handleUpdateName}
                    disabled={updateTenant.isPending || !name.trim() || name === tenant.name}
                    className="btn-primary disabled:opacity-50"
                  >
                    {updateTenant.isPending ? "保存中..." : "保存"}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Transfer Ownership */}
        {isOwner && (
          <div className="bg-card border border-amber-200 dark:border-amber-800 rounded-xl p-5 shadow-card">
            <div className="flex items-center gap-2 mb-2">
              <div className="w-7 h-7 rounded-lg bg-amber-50 dark:bg-amber-500/10 flex items-center justify-center">
                <ArrowRightLeft size={14} className="text-amber-500" />
              </div>
              <h3 className="text-sm font-semibold text-foreground">转让所有权</h3>
            </div>
            <p className="text-xs text-muted-foreground mb-4 ml-9">
              将组织所有权转让给其他成员。转让后你将失去所有者权限。
            </p>
            <button
              onClick={() => setShowTransfer(true)}
              className="ml-9 px-4 py-2 text-sm font-medium border border-amber-500 text-amber-600 dark:text-amber-400 rounded-lg hover:bg-amber-50 dark:hover:bg-amber-900/20 transition-colors"
            >
              转让所有权
            </button>
          </div>
        )}

        {/* Danger Zone */}
        {isOwner && (
          <div className="bg-card border border-destructive/50 rounded-xl p-5 shadow-card">
            <div className="flex items-center gap-2 mb-2">
              <div className="w-7 h-7 rounded-lg bg-red-50 dark:bg-red-500/10 flex items-center justify-center">
                <AlertTriangle size={14} className="text-destructive" />
              </div>
              <h3 className="text-sm font-semibold text-destructive">危险操作</h3>
            </div>
            <p className="text-xs text-muted-foreground mb-4 ml-9">
              删除组织将永久删除所有数据，包括成员、API 密钥和交易记录。此操作不可恢复。
            </p>
            <button
              onClick={handleDelete}
              disabled={deleteTenant.isPending}
              className="ml-9 flex items-center gap-1.5 px-4 py-2 text-sm font-semibold bg-destructive text-destructive-foreground rounded-lg hover:bg-destructive/90 disabled:opacity-50 transition-colors"
            >
              <Trash2 size={14} />
              {deleteTenant.isPending ? "删除中..." : "删除组织"}
            </button>
          </div>
        )}
      </div>

      {showTransfer && members && (
        <TransferModal
          tenantId={id!}
          members={members}
          onClose={() => setShowTransfer(false)}
        />
      )}
    </div>
  );
};

export default TenantSettingsPage;
