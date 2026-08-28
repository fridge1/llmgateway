import { useParams, useNavigate } from "react-router-dom";
import { useTenantDetail, useTenantBalance, useTenantMembers, useTenantStats } from "@/hooks/use-tenant";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { Loader2 } from "../../components/icons";

export default function TenantDashboard() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: tenant } = useTenantDetail(id);
  const { data: balance } = useTenantBalance(id);
  const { data: members = [] } = useTenantMembers(id);
  const { data: stats, isLoading } = useTenantStats(id, 7);

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}>
        <Loader2 size={24} className="animate-spin text-amber-400" />
      </div>
    );
  }

  const navItems = [
    { path: `/tenants/${id}/members`, label: "成员管理" },
    { path: `/tenants/${id}/keys`, label: "API 密钥" },
    { path: `/tenants/${id}/billing`, label: "余额交易" },
    { path: `/tenants/${id}/usage`, label: "使用记录" },
    { path: `/tenants/${id}/analytics`, label: "使用分析" },
    { path: `/tenants/${id}/settings`, label: "设置" },
  ];

  return (
    <div className="p-6">
      <div className="flex items-center gap-2 mb-5">
        <button onClick={() => navigate("/tenants")} className="text-xs text-obsidian-400 hover:text-obsidian-200">← 返回</button>
        <h1 className="text-lg font-semibold text-obsidian-50">{tenant?.name ?? "组织"}</h1>
      </div>

      {/* Quick nav */}
      <div className="flex gap-2 mb-5 flex-wrap">
        {navItems.map((item) => (
          <button
            key={item.path}
            onClick={() => navigate(item.path)}
            className="px-3 py-1.5 bg-obsidian-800 hover:bg-obsidian-700 text-obsidian-300 text-xs rounded-lg transition-colors"
          >
            {item.label}
          </button>
        ))}
      </div>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-3 mb-5">
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-xs text-obsidian-400 mb-1">余额</div>
          <div className="text-xl font-bold text-obsidian-50">¥{(balance?.balance ?? 0).toFixed(2)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-xs text-obsidian-400 mb-1">今日消费</div>
          <div className="text-xl font-bold text-obsidian-50">¥{(stats?.today_cost ?? 0).toFixed(4)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-xs text-obsidian-400 mb-1">本月消费</div>
          <div className="text-xl font-bold text-obsidian-50">¥{(stats?.month_cost ?? 0).toFixed(4)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-xs text-obsidian-400 mb-1">成员</div>
          <div className="text-xl font-bold text-obsidian-50">{members.length}</div>
        </div>
      </div>

      {/* Cost trend chart */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4 mb-5">
        <div className="text-sm font-semibold text-obsidian-50 mb-3">费用趋势（近 7 天）</div>
        {(stats?.daily_trend ?? []).length === 0 ? (
          <div className="py-12 text-center text-sm text-obsidian-500">暂无数据</div>
        ) : (
          <div style={{ height: 180 }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={stats?.daily_trend ?? []} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                <defs>
                  <linearGradient id="tenantCostGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.2} />
                    <stop offset="95%" stopColor="#f59e0b" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a38" vertical={false} />
                <XAxis dataKey="date" tick={{ fontSize: 11, fill: "#8888a0" }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: "#8888a0" }} axisLine={false} tickLine={false} />
                <Tooltip
                  contentStyle={{ background: "#111118", border: "1px solid #2a2a38", borderRadius: "8px", fontSize: 12 }}
                  formatter={(v) => [`¥${Number(v).toFixed(4)}`, "消费"]}
                />
                <Area type="monotone" dataKey="cost" stroke="#f59e0b" strokeWidth={2} fill="url(#tenantCostGrad)" dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      {/* Sub-user ranking */}
      {(stats?.sub_user_ranking ?? []).length > 0 && (
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-sm font-semibold text-obsidian-50 mb-3">子用户消费排行</div>
          <div className="space-y-2">
            {stats!.sub_user_ranking.map((u, i) => (
              <div key={u.sub_user_id} className="flex items-center gap-3">
                <span className="w-5 text-xs text-obsidian-500 text-right">{i + 1}</span>
                <span className="text-xs text-obsidian-200 flex-1 truncate">{u.sub_user_username}</span>
                <span className="text-xs text-obsidian-400">{u.request_count} 次</span>
                <span className="text-xs text-obsidian-300 font-medium">¥{u.total_cost.toFixed(4)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
