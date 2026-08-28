import { useParams, useNavigate } from "react-router-dom";
import { useTenantStats } from "@/hooks/use-tenant";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from "recharts";
import { Loader2 } from "../../components/icons";
import { useState } from "react";

export default function TenantAnalyticsPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [days, setDays] = useState(7);
  const { data: stats, isLoading } = useTenantStats(id, days);

  if (isLoading) {
    return <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}><Loader2 size={24} className="animate-spin text-amber-400" /></div>;
  }

  const dailyTrend = stats?.daily_trend ?? [];
  const modelBreakdown = stats?.model_breakdown ?? [];

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-5">
        <div className="flex items-center gap-2">
          <button onClick={() => navigate(`/tenants/${id}`)} className="text-xs text-obsidian-400 hover:text-obsidian-200">← 返回</button>
          <h1 className="text-lg font-semibold text-obsidian-50">使用分析</h1>
        </div>
        <div className="flex gap-1 bg-obsidian-800 rounded-lg p-0.5">
          {[7, 14, 30].map((d) => (
            <button key={d} onClick={() => setDays(d)} className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${days === d ? "bg-amber-500 text-obsidian-950" : "text-obsidian-400"}`}>{d} 天</button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-3 gap-3 mb-5">
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-xs text-obsidian-400 mb-1">今日消费</div>
          <div className="text-xl font-bold text-obsidian-50">¥{(stats?.today_cost ?? 0).toFixed(4)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-xs text-obsidian-400 mb-1">本月消费</div>
          <div className="text-xl font-bold text-obsidian-50">¥{(stats?.month_cost ?? 0).toFixed(4)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-xs text-obsidian-400 mb-1">日均消费</div>
          <div className="text-xl font-bold text-obsidian-50">¥{(stats?.daily_average ?? 0).toFixed(4)}</div>
        </div>
      </div>

      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4 mb-5">
        <div className="text-sm font-semibold text-obsidian-50 mb-3">消费趋势</div>
        {dailyTrend.length === 0 ? (
          <div className="py-12 text-center text-sm text-obsidian-500">暂无数据</div>
        ) : (
          <div style={{ height: 200 }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={dailyTrend} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                <defs>
                  <linearGradient id="tAnalyticsCostGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.2} />
                    <stop offset="95%" stopColor="#f59e0b" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a38" vertical={false} />
                <XAxis dataKey="date" tick={{ fontSize: 11, fill: "#8888a0" }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: "#8888a0" }} axisLine={false} tickLine={false} />
                <Tooltip contentStyle={{ background: "#111118", border: "1px solid #2a2a38", borderRadius: "8px", fontSize: 12 }} formatter={(v) => [`¥${Number(v).toFixed(4)}`, "消费"]} />
                <Area type="monotone" dataKey="cost" stroke="#f59e0b" strokeWidth={2} fill="url(#tAnalyticsCostGrad)" dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
        <div className="text-sm font-semibold text-obsidian-50 mb-3">模型消费分布</div>
        {modelBreakdown.length === 0 ? (
          <div className="py-12 text-center text-sm text-obsidian-500">暂无数据</div>
        ) : (
          <div style={{ height: 200 }}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={modelBreakdown} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a38" vertical={false} />
                <XAxis dataKey="model" tick={{ fontSize: 9, fill: "#8888a0" }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: "#8888a0" }} axisLine={false} tickLine={false} />
                <Tooltip contentStyle={{ background: "#111118", border: "1px solid #2a2a38", borderRadius: "8px", fontSize: 12 }} formatter={(v) => [`¥${Number(v).toFixed(4)}`, "消费"]} />
                <Bar dataKey="cost" fill="#f59e0b" radius={[4, 4, 0, 0]} fillOpacity={0.8} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>
    </div>
  );
}
