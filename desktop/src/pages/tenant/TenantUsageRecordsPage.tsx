import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTenantAllSubUserTransactions, useTenantSubUsers } from "@/hooks/use-tenant";
import { Loader2 } from "../../components/icons";

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("zh-CN");
}

export default function TenantUsageRecordsPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: subUsers = [] } = useTenantSubUsers(id);
  const [page, setPage] = useState(1);
  const [filterUser, setFilterUser] = useState("");
  const size = 20;
  const { data: txData, isLoading } = useTenantAllSubUserTransactions(id, page, size, filterUser || undefined);
  const transactions = txData?.transactions ?? [];
  const total = txData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  if (isLoading) {
    return <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}><Loader2 size={24} className="animate-spin text-amber-400" /></div>;
  }

  return (
    <div className="p-6">
      <div className="flex items-center gap-2 mb-5">
        <button onClick={() => navigate(`/tenants/${id}`)} className="text-xs text-obsidian-400 hover:text-obsidian-200">← 返回</button>
        <h1 className="text-lg font-semibold text-obsidian-50">使用记录</h1>
      </div>

      <div className="flex items-center gap-3 mb-4">
        <select value={filterUser} onChange={(e) => { setFilterUser(e.target.value); setPage(1); }} className="px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 focus:outline-none focus:border-amber-500/50">
          <option value="">全部子用户</option>
          {subUsers.map(u => (
            <option key={u.id} value={u.id}>{u.nickname || u.username}</option>
          ))}
        </select>
        <div className="text-xs text-obsidian-500">共 {total} 条记录</div>
      </div>

      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
              <th className="text-left px-4 py-2 font-medium">时间</th>
              <th className="text-left px-4 py-2 font-medium">子用户</th>
              <th className="text-left px-4 py-2 font-medium">类型</th>
              <th className="text-left px-4 py-2 font-medium">金额</th>
              <th className="text-left px-4 py-2 font-medium">模型</th>
              <th className="text-left px-4 py-2 font-medium">Tokens</th>
            </tr>
          </thead>
          <tbody>
            {transactions.length === 0 ? (
              <tr><td colSpan={6} className="px-4 py-12 text-center text-sm text-obsidian-500">暂无记录</td></tr>
            ) : transactions.map((t) => (
              <tr key={t.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                <td className="px-4 py-2.5 text-xs text-obsidian-400">{formatTime(t.created_at)}</td>
                <td className="px-4 py-2.5 text-xs text-obsidian-300">{t.sub_user_username || "—"}</td>
                <td className="px-4 py-2.5 text-xs text-obsidian-300">{t.type}</td>
                <td className={`px-4 py-2.5 text-xs font-medium ${t.amount >= 0 ? "text-emerald-400" : "text-red-400"}`}>
                  {t.amount >= 0 ? "+" : ""}¥{Math.abs(t.amount).toFixed(4)}
                </td>
                <td className="px-4 py-2.5 text-xs text-obsidian-400 truncate max-w-[130px]">{t.model || "—"}</td>
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
