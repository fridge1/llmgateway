import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTenants, useCreateTenant, usePendingInvitations, useAcceptInvitation } from "@/hooks/use-tenant";
import { Loader2 } from "../../components/icons";

export default function TenantListPage() {
  const navigate = useNavigate();
  const { data: tenants = [], isLoading } = useTenants();
  const { data: invitations = [] } = usePendingInvitations();
  const createTenant = useCreateTenant();
  const acceptInvitation = useAcceptInvitation();
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");

  const handleCreate = async () => {
    if (!name.trim()) return;
    try {
      await createTenant.mutateAsync(name.trim());
      setShowCreate(false);
      setName("");
    } catch {}
  };

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}>
        <Loader2 size={24} className="animate-spin text-amber-400" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-5">
        <div>
          <h1 className="text-lg font-semibold text-obsidian-50">我的组织</h1>
          <p className="text-xs text-obsidian-400 mt-0.5">管理您的组织和团队</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 bg-amber-500 hover:bg-amber-400 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200"
        >
          创建组织
        </button>
      </div>

      {/* Pending invitations */}
      {invitations.length > 0 && (
        <div className="bg-amber-500/5 border border-amber-500/20 rounded-xl p-4 mb-5">
          <div className="text-sm font-semibold text-amber-400 mb-2">待接受邀请</div>
          <div className="space-y-2">
            {invitations.map((inv) => (
              <div key={inv.id} className="flex items-center justify-between bg-obsidian-900/60 rounded-lg p-3">
                <div>
                  <div className="text-sm text-obsidian-200">{inv.tenant_name || "组织邀请"}</div>
                  <div className="text-xs text-obsidian-400">角色: {inv.role}</div>
                </div>
                <button
                  onClick={() => acceptInvitation.mutate(inv.id)}
                  disabled={acceptInvitation.isPending}
                  className="px-3 py-1.5 bg-amber-500 text-obsidian-950 text-xs font-semibold rounded-lg hover:bg-amber-400 transition-colors"
                >
                  接受
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tenant list */}
      {tenants.length === 0 ? (
        <div className="py-16 text-center text-sm text-obsidian-500">暂无组织</div>
      ) : (
        <div className="grid grid-cols-2 gap-3">
          {tenants.map((t) => (
            <button
              key={t.id}
              onClick={() => navigate(`/tenants/${t.id}`)}
              className="bg-obsidian-900 border border-obsidian-700 hover:border-obsidian-600 rounded-xl p-4 text-left transition-all duration-200"
            >
              <div className="text-sm font-semibold text-obsidian-50 mb-1">{t.name}</div>
              <div className="text-xs text-obsidian-400">
                角色: <span className={t.role === "owner" ? "text-amber-400" : "text-obsidian-300"}>{t.role}</span>
              </div>
            </button>
          ))}
        </div>
      )}

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[400px] p-6">
            <h3 className="text-base font-semibold text-obsidian-50 mb-4">创建组织</h3>
            <input
              className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50 mb-4"
              placeholder="组织名称"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleCreate()}
            />
            <div className="flex gap-3 justify-end">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 text-sm text-obsidian-300 hover:text-obsidian-100">取消</button>
              <button
                onClick={handleCreate}
                disabled={createTenant.isPending || !name.trim()}
                className="px-4 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200"
              >
                创建
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
