import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTenantKeys, useCreateTenantKey, useDeleteTenantKey } from "@/hooks/use-tenant";
import { Loader2 } from "../../components/icons";

export default function TenantKeysPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: keys = [], isLoading } = useTenantKeys(id);
  const createKey = useCreateTenantKey(id);
  const deleteKey = useDeleteTenantKey(id);
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [createdKey, setCreatedKey] = useState<string | null>(null);

  const handleCreate = async () => {
    if (!name.trim()) return;
    try {
      const res = await createKey.mutateAsync(name.trim());
      setCreatedKey(res.key);
      setShowCreate(false);
      setName("");
    } catch {}
  };

  if (isLoading) {
    return <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}><Loader2 size={24} className="animate-spin text-amber-400" /></div>;
  }

  return (
    <div className="p-6">
      <div className="flex items-center gap-2 mb-5">
        <button onClick={() => navigate(`/tenants/${id}`)} className="text-xs text-obsidian-400 hover:text-obsidian-200">← 返回</button>
        <h1 className="text-lg font-semibold text-obsidian-50">组织 API 密钥</h1>
      </div>

      <div className="flex justify-end mb-3">
        <button onClick={() => setShowCreate(true)} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 text-obsidian-950 text-sm font-semibold rounded-lg transition-all">创建密钥</button>
      </div>

      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
              <th className="text-left px-4 py-2 font-medium">名称</th>
              <th className="text-left px-4 py-2 font-medium">密钥前缀</th>
              <th className="text-left px-4 py-2 font-medium">状态</th>
              <th className="text-left px-4 py-2 font-medium">创建时间</th>
              <th className="text-right px-4 py-2 font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            {keys.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-12 text-center text-sm text-obsidian-500">暂无密钥</td></tr>
            ) : keys.map((k) => (
              <tr key={k.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                <td className="px-4 py-3 text-sm text-obsidian-200">{k.name}</td>
                <td className="px-4 py-3 text-sm font-mono text-obsidian-400">{k.key_prefix}****</td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded-full text-xs font-medium ${k.status === "active" ? "bg-emerald-500/10 text-emerald-400" : "bg-obsidian-700 text-obsidian-400"}`}>{k.status === "active" ? "活跃" : k.status}</span></td>
                <td className="px-4 py-3 text-xs text-obsidian-400">{new Date(k.created_at).toLocaleString("zh-CN")}</td>
                <td className="px-4 py-3 text-right">
                  <button onClick={() => window.confirm("确定删除此密钥？") && deleteKey.mutate(k.id)} className="px-2 py-1 text-xs text-obsidian-400 hover:text-red-400">删除</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[400px] p-6">
            <h3 className="text-base font-semibold text-obsidian-50 mb-4">创建 API 密钥</h3>
            <input className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50 mb-4" placeholder="密钥名称" value={name} onChange={(e) => setName(e.target.value)} onKeyDown={(e) => e.key === "Enter" && handleCreate()} />
            <div className="flex gap-3 justify-end">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 text-sm text-obsidian-300">取消</button>
              <button onClick={handleCreate} disabled={createKey.isPending || !name.trim()} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 text-obsidian-950 text-sm font-semibold rounded-lg transition-all">创建</button>
            </div>
          </div>
        </div>
      )}

      {createdKey && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[480px] p-6">
            <h3 className="text-base font-semibold text-obsidian-50 mb-2">密钥已创建</h3>
            <p className="text-xs text-obsidian-400 mb-3">请立即复制，此密钥仅显示一次。</p>
            <div className="bg-obsidian-800 rounded-lg px-4 py-3 mb-4 border border-obsidian-700">
              <code className="text-sm font-mono text-obsidian-100 break-all select-all">{createdKey}</code>
            </div>
            <div className="flex justify-end">
              <button onClick={() => setCreatedKey(null)} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 text-obsidian-950 text-sm font-semibold rounded-lg">关闭</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
