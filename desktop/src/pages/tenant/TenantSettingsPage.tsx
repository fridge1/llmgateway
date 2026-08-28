import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTenantDetail, useUpdateTenant, useDeleteTenant, useTenantMembers, useTransferOwnership } from "@/hooks/use-tenant";
import { Loader2 } from "../../components/icons";

export default function TenantSettingsPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: tenant, isLoading } = useTenantDetail(id);
  const { data: members = [] } = useTenantMembers(id);
  const updateTenant = useUpdateTenant(id);
  const deleteTenant = useDeleteTenant(id);
  const transferOwnership = useTransferOwnership(id);

  const [newName, setNewName] = useState("");
  const [showRename, setShowRename] = useState(false);
  const [showTransfer, setShowTransfer] = useState(false);
  const [transferTarget, setTransferTarget] = useState("");
  const [showDelete, setShowDelete] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState("");

  const handleRename = async () => {
    if (!newName.trim()) return;
    try {
      await updateTenant.mutateAsync(newName.trim());
      setShowRename(false);
      setNewName("");
    } catch {}
  };

  const handleTransfer = async () => {
    if (!transferTarget) return;
    try {
      await transferOwnership.mutateAsync(transferTarget);
      setShowTransfer(false);
      setTransferTarget("");
    } catch {}
  };

  const handleDelete = async () => {
    if (deleteConfirm !== tenant?.name) return;
    try {
      await deleteTenant.mutateAsync();
      navigate("/tenants");
    } catch {}
  };

  if (isLoading) {
    return <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}><Loader2 size={24} className="animate-spin text-amber-400" /></div>;
  }

  const nonOwnerMembers = members.filter(m => m.role !== "owner");

  return (
    <div className="p-6">
      <div className="flex items-center gap-2 mb-5">
        <button onClick={() => navigate(`/tenants/${id}`)} className="text-xs text-obsidian-400 hover:text-obsidian-200">← 返回</button>
        <h1 className="text-lg font-semibold text-obsidian-50">组织设置</h1>
      </div>

      <div className="space-y-4">
        {/* Rename */}
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-5">
          <div className="text-sm font-semibold text-obsidian-50 mb-1">组织名称</div>
          <div className="text-xs text-obsidian-400 mb-3">当前名称：{tenant?.name}</div>
          <button onClick={() => { setShowRename(true); setNewName(tenant?.name ?? ""); }} className="px-4 py-2 bg-obsidian-800 hover:bg-obsidian-700 text-obsidian-200 text-sm rounded-lg transition-colors">重命名</button>
        </div>

        {/* Transfer */}
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-5">
          <div className="text-sm font-semibold text-obsidian-50 mb-1">转让组织</div>
          <div className="text-xs text-obsidian-400 mb-3">将组织所有权转让给其他成员。转让后您将变为管理员。</div>
          <button onClick={() => setShowTransfer(true)} disabled={nonOwnerMembers.length === 0} className="px-4 py-2 bg-obsidian-800 hover:bg-obsidian-700 text-obsidian-200 text-sm rounded-lg transition-colors disabled:opacity-40">转让所有权</button>
        </div>

        {/* Delete */}
        <div className="bg-obsidian-900 border border-red-900/50 rounded-xl p-5">
          <div className="text-sm font-semibold text-red-400 mb-1">删除组织</div>
          <div className="text-xs text-obsidian-400 mb-3">删除后所有数据将被永久清除，此操作不可撤销。</div>
          <button onClick={() => setShowDelete(true)} className="px-4 py-2 bg-red-500/10 hover:bg-red-500/20 text-red-400 text-sm rounded-lg transition-colors">删除组织</button>
        </div>
      </div>

      {/* Rename modal */}
      {showRename && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[400px] p-6">
            <h3 className="text-base font-semibold text-obsidian-50 mb-4">重命名组织</h3>
            <input className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50 mb-4" placeholder="新名称" value={newName} onChange={(e) => setNewName(e.target.value)} onKeyDown={(e) => e.key === "Enter" && handleRename()} />
            <div className="flex gap-3 justify-end">
              <button onClick={() => setShowRename(false)} className="px-4 py-2 text-sm text-obsidian-300">取消</button>
              <button onClick={handleRename} disabled={updateTenant.isPending || !newName.trim()} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 text-obsidian-950 text-sm font-semibold rounded-lg transition-all">确认</button>
            </div>
          </div>
        </div>
      )}

      {/* Transfer modal */}
      {showTransfer && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[400px] p-6">
            <h3 className="text-base font-semibold text-obsidian-50 mb-4">转让所有权</h3>
            <p className="text-xs text-obsidian-400 mb-3">选择要转让的成员：</p>
            <select value={transferTarget} onChange={(e) => setTransferTarget(e.target.value)} className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 focus:outline-none focus:border-amber-500/50 mb-4">
              <option value="">请选择</option>
              {nonOwnerMembers.map(m => (
                <option key={m.user_id} value={m.user_id}>{m.phone || m.user_id} ({m.role})</option>
              ))}
            </select>
            <div className="flex gap-3 justify-end">
              <button onClick={() => setShowTransfer(false)} className="px-4 py-2 text-sm text-obsidian-300">取消</button>
              <button onClick={handleTransfer} disabled={transferOwnership.isPending || !transferTarget} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 text-obsidian-950 text-sm font-semibold rounded-lg transition-all">确认转让</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete modal */}
      {showDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[400px] p-6">
            <h3 className="text-base font-semibold text-red-400 mb-2">删除组织</h3>
            <p className="text-xs text-obsidian-400 mb-3">请输入组织名称 <span className="text-obsidian-200 font-medium">{tenant?.name}</span> 以确认删除：</p>
            <input className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-red-500/50 mb-4" placeholder="输入组织名称" value={deleteConfirm} onChange={(e) => setDeleteConfirm(e.target.value)} />
            <div className="flex gap-3 justify-end">
              <button onClick={() => { setShowDelete(false); setDeleteConfirm(""); }} className="px-4 py-2 text-sm text-obsidian-300">取消</button>
              <button onClick={handleDelete} disabled={deleteTenant.isPending || deleteConfirm !== tenant?.name} className="px-4 py-2 bg-red-500 hover:bg-red-400 disabled:bg-obsidian-800 text-white text-sm font-semibold rounded-lg transition-all">确认删除</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
