import { useState } from "react";
import { useApiKeys, useCreateApiKey, useDeleteApiKey, useApiKeyUsage, useSubscriptionCurrent } from "@/hooks/use-api";
import { Key, Plus, Loader2, Copy, CheckCircle2, Trash2, Terminal, KeyRound, AlertCircle } from "../components/icons";
import EmptyState from "../components/EmptyState";

export default function KeysPage() {
  const { data: keys = [], isLoading } = useApiKeys();
  const { data: keyUsage = [] } = useApiKeyUsage(30);
  const { data: subData } = useSubscriptionCurrent();
  const activeSubs = subData?.subscriptions ?? [];
  const createKey = useCreateApiKey();
  const deleteKey = useDeleteApiKey();
  const [copied, setCopied] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyPlanId, setNewKeyPlanId] = useState<number | undefined>(undefined);
  const [createdFullKey, setCreatedFullKey] = useState<string | null>(null);

  const handleCopy = async (text: string, id: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(id);
      setTimeout(() => setCopied(null), 2000);
    } catch {
      const textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      setCopied(id);
      setTimeout(() => setCopied(null), 2000);
    }
  };

  const handleCreate = async () => {
    if (!newKeyName.trim()) return;
    try {
      const res = await createKey.mutateAsync({ name: newKeyName.trim(), plan_id: newKeyPlanId });
      setCreatedFullKey(res.key);
      setShowCreate(false);
      setNewKeyName("");
      setNewKeyPlanId(undefined);
    } catch {}
  };

  const handleDelete = (id: string, name: string) => {
    if (window.confirm(`确定要停用密钥「${name}」吗？`)) {
      deleteKey.mutate(id);
    }
  };

  const getKeyUsage = (keyId: string) => keyUsage.find(u => u.key_id === keyId);

  const formatTime = (iso: string) => new Date(iso).toLocaleString("zh-CN");

  const curlCode = `curl https://your-gateway-host/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'`;

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={24} className="animate-spin text-amber-400" />
          <span className="text-sm text-obsidian-400">加载中...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-lg font-semibold text-obsidian-50 flex items-center gap-2">
            <Key size={20} className="text-obsidian-400" />
            API 密钥
          </h1>
          <p className="text-xs text-obsidian-400 mt-1">管理你的 API 访问凭证</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 bg-amber-500 hover:bg-amber-400 hover:shadow-glow-amber text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200 inline-flex items-center gap-1.5"
        >
          <Plus size={16} />
          创建密钥
        </button>
      </div>

      {/* Security notice */}
      <div className="flex items-start gap-3 bg-amber-500/5 border border-amber-500/20 rounded-xl px-4 py-3 mb-6">
        <AlertCircle size={16} className="text-amber-400 mt-0.5 shrink-0" />
        <p className="text-xs text-obsidian-300 leading-relaxed">
          API 密钥用于调用 LLM Gateway 的 OpenAI 兼容接口。请妥善保管，密钥仅在创建时显示一次。
        </p>
      </div>

      {/* Keys table */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden mb-6">
        <div className="px-4 py-3 border-b border-obsidian-700 flex items-center justify-between">
          <div className="text-sm font-semibold text-obsidian-50">密钥列表</div>
          <div className="text-xs text-obsidian-400">{keys.length} 个密钥</div>
        </div>

        {keys.length === 0 ? (
          <div className="py-16">
            <EmptyState
              icon={KeyRound}
              title="暂无 API 密钥"
              description="点击右上角按钮创建您的第一个 API 密钥"
              action={{ label: "创建密钥", onClick: () => setShowCreate(true) }}
            />
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
                <th className="text-left px-4 py-2 font-medium">名称</th>
                <th className="text-left px-4 py-2 font-medium">密钥</th>
                <th className="text-left px-4 py-2 font-medium">适用套餐</th>
                <th className="text-left px-4 py-2 font-medium">创建时间</th>
                <th className="text-left px-4 py-2 font-medium">最后使用</th>
                <th className="text-right px-4 py-2 font-medium">30天消费</th>
                <th className="text-right px-4 py-2 font-medium">请求次数</th>
                <th className="text-right px-4 py-2 font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((k) => {
                const usage = getKeyUsage(k.id);
                const planName = k.plan_id
                  ? (activeSubs.find(s => s.subscription.plan_id === k.plan_id)?.subscription.plan?.display_name ?? `套餐 ${k.plan_id}`)
                  : "全部模型";
                return (
                  <tr key={k.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                    <td className="px-4 py-3 text-sm font-medium text-obsidian-100">{k.name}</td>
                    <td className="px-4 py-3 text-sm font-mono text-obsidian-400">{k.key_prefix}****</td>
                    <td className="px-4 py-3 text-xs text-obsidian-400">{planName}</td>
                    <td className="px-4 py-3 text-xs text-obsidian-400">{formatTime(k.created_at)}</td>
                    <td className="px-4 py-3 text-xs text-obsidian-400">{k.last_used_at ? formatTime(k.last_used_at) : "从未使用"}</td>
                    <td className="px-4 py-3 text-xs text-right text-obsidian-300">
                      {usage ? `¥${usage.total_cost.toFixed(2)}` : "-"}
                    </td>
                    <td className="px-4 py-3 text-xs text-right text-obsidian-300">
                      {usage ? usage.request_count.toLocaleString() : "-"}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end">
                        <button
                          onClick={() => handleDelete(k.id, k.name)}
                          className="w-7 h-7 rounded-lg flex items-center justify-center text-obsidian-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Quick start */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
        <div className="px-4 py-3 border-b border-obsidian-700 flex items-center gap-2">
          <Terminal size={16} className="text-obsidian-400" />
          <div className="text-sm font-semibold text-obsidian-50">快速开始</div>
        </div>
        <div className="px-4 py-4">
          <p className="text-xs text-obsidian-400 mb-3">
            使用以下 curl 命令测试你的 API 密钥（替换 <code className="bg-obsidian-800 px-1.5 py-0.5 rounded text-xs font-mono text-obsidian-200">YOUR_API_KEY</code> 为你的实际密钥）：
          </p>
          <div className="bg-obsidian-950 border border-obsidian-700 rounded-lg overflow-hidden">
            <div className="flex items-center justify-between px-4 py-2 border-b border-obsidian-800">
              <span className="text-xs font-medium text-obsidian-500">BASH</span>
              <button
                onClick={() => handleCopy(curlCode, "curl")}
                className="flex items-center gap-1.5 text-xs transition-colors px-2 py-1 rounded text-obsidian-400 hover:text-obsidian-200"
              >
                {copied === "curl" ? <CheckCircle2 size={12} className="text-emerald-400" /> : <Copy size={12} />}
                {copied === "curl" ? "已复制" : "复制"}
              </button>
            </div>
            <pre className="px-4 py-3 text-xs overflow-x-auto text-obsidian-200 font-mono leading-relaxed">
              {curlCode}
            </pre>
          </div>
        </div>
      </div>

      {/* Create key modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[400px] p-6">
            <h3 className="text-base font-semibold text-obsidian-50 mb-1">创建 API 密钥</h3>
            <p className="text-xs text-obsidian-400 mb-4">为您的密钥设置一个易于识别的名称</p>
            <label className="block text-sm font-medium text-obsidian-200 mb-1.5">密钥名称</label>
            <input
              className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50 mb-4"
              placeholder="例如：生产环境密钥"
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleCreate()}
            />
            <label className="block text-sm font-medium text-obsidian-200 mb-1.5">适用套餐</label>
            <select
              className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 focus:outline-none focus:border-amber-500/50 mb-4"
              value={newKeyPlanId ?? ""}
              onChange={(e) => setNewKeyPlanId(e.target.value ? Number(e.target.value) : undefined)}
            >
              <option value="">不限制（全部模型）</option>
              {activeSubs.map(({ subscription: s }) => (
                <option key={s.plan_id} value={s.plan_id}>
                  {s.plan?.display_name ?? `套餐 ${s.plan_id}`}
                </option>
              ))}
            </select>
            <div className="flex gap-3 justify-end">
              <button onClick={() => { setShowCreate(false); setNewKeyName(""); setNewKeyPlanId(undefined); }} className="px-4 py-2 text-sm text-obsidian-300 hover:text-obsidian-100 transition-colors">取消</button>
              <button
                onClick={handleCreate}
                disabled={createKey.isPending || !newKeyName.trim()}
                className="px-4 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200"
              >
                {createKey.isPending ? "创建中..." : "创建"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Key created modal */}
      {createdFullKey && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[480px] p-6">
            <div className="flex items-center gap-2 mb-1">
              <CheckCircle2 size={18} className="text-emerald-400" />
              <h3 className="text-base font-semibold text-obsidian-50">密钥已创建</h3>
            </div>
            <p className="text-xs text-obsidian-400 mb-4">请立即复制并妥善保存，此密钥仅显示一次。</p>
            <div className="flex items-center gap-2 bg-obsidian-800 rounded-lg px-4 py-3 mb-4 border border-obsidian-700">
              <code className="flex-1 text-sm font-mono text-obsidian-100 break-all select-all">{createdFullKey}</code>
              <button
                onClick={() => handleCopy(createdFullKey, "created-key")}
                className="shrink-0 px-3 py-1.5 text-xs bg-obsidian-700 hover:bg-obsidian-600 text-obsidian-200 rounded-lg transition-colors flex items-center gap-1.5"
              >
                {copied === "created-key" ? <CheckCircle2 size={12} className="text-emerald-400" /> : <Copy size={12} />}
                {copied === "created-key" ? "已复制" : "复制"}
              </button>
            </div>
            <div className="flex justify-end">
              <button onClick={() => setCreatedFullKey(null)} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200">
                我已保存，关闭
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
