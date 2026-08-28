import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTenantSubUserTransactions, useTenantSubUsers } from "@/hooks/use-tenant";
import { Loader2 } from "../../components/icons";

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("zh-CN");
}

export default function TenantSubUserTransactionsPage() {
  const { id = "", subUserId = "" } = useParams();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const size = 20;
  const { data: txData, isLoading } = useTenantSubUserTransactions(id, subUserId, page, size);
  const { data: subUsers = [] } = useTenantSubUsers(id);
  const transactions = txData?.transactions ?? [];
  const total = txData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  const subUser = subUsers.find(u => u.id === subUserId);

  if (isLoading) {
    return <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}><Loader2 size={24} className="animate-spin text-amber-400" /></div>;
  }

  return (
    <div className="p-6">
      <div className="flex items-center gap-2 mb-5">
        <button onClick={() => navigate(`/tenants/${id}/members`)} className="text-xs text-obsidian-400 hover:text-obsidian-200">← 返回成员管理</button>
        <h1 className="text-lg font-semibold text-obsidian-50">
          {subUser ? `${subUser.nickname || subUser.username} 的交易记录` : "子用户交易记录"}
        </h1>
      </div>

      {subUser && (
        <div className="grid grid-cols-3 gap-3 mb-5">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
            <div className="text-xs text-obsidian-400 mb-1">用户名</div>
            <div className="text-sm font-medium text-obsidian-50">{subUser.username}</div>
          </div>
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
            <div className="text-xs text-obsidian-400 mb-1">额度</div>
            <div className="text-sm font-medium text-obsidian-50">{subUser.quota_limit !== null ? `¥${subUser.quota_limit}` : "无限制"}</div>
          </div>
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
            <div className="text-xs text-obsidian-400 mb-1">已用</div>
            <div className="text-sm font-medium text-obsidian-50">¥{subUser.quota_used.toFixed(2)}</div>
          </div>
        </div>
      )}

      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
              <th className="text-left px-4 py-2 font-medium">时间</th>
              <th className="text-left px-4 py-2 font-medium">类型</th>
              <th className="text-left px-4 py-2 font-medium">金额</th>
              <th className="text-left px-4 py-2 font-medium">模型</th>
              <th className="text-left px-4 py-2 font-medium">Tokens</th>
            </tr>
          </thead>
          <tbody>
            {transactions.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-12 text-center text-sm text-obsidian-500">暂无交易</td></tr>
            ) : transactions.map((t) => (
              <tr key={t.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                <td className="px-4 py-2.5 text-xs text-obsidian-400">{formatTime(t.created_at)}</td>
                <td className="px-4 py-2.5 text-xs text-obsidian-300">{t.type}</td>
                <td className={`px-4 py-2.5 text-xs font-medium ${t.amount >= 0 ? "text-emerald-400" : "text-red-400"}`}>
                  {t.amount >= 0 ? "+" : ""}¥{Math.abs(t.amount).toFixed(4)}
                </td>
                <td className="px-4 py-2.5 text-xs text-obsidian-400 truncate max-w-[150px]">{t.model || "—"}</td>
                <td className="px-4 py-2.5 text-xs text-obsidian-500">
                  {t.prompt_tokens || t.completion_tokens ? (
                    <span>{t.prompt_tokens ?? 0} / {t.completion_tokens ?? 0}</span>
                  ) : "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 mt-4">
          <button className="px-3 py-1.5 rounded-lg text-xs bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700 disabled:opacity-40" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</button>
          <span className="text-xs text-obsidian-400">{page} / {totalPages}</span>
          <button className="px-3 py-1.5 rounded-lg text-xs bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700 disabled:opacity-40" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</button>
        </div>
      )}
    </div>
  );
}
