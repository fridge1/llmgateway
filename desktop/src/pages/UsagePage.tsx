import { useState } from "react";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from "recharts";
import { useBillingStats } from "@/hooks/use-api";
import { Loader2 } from "../components/icons";

export default function UsagePage() {
  const [days, setDays] = useState(7);
  const { data: stats, isLoading } = useBillingStats(days);

  const chartData = stats?.daily_trend ?? [];
  const modelData = stats?.model_breakdown ?? [];
  const totalCost = modelData.reduce((sum, m) => sum + (m.cost ?? 0), 0);

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
      <div className="flex items-center justify-between mb-5">
        <div>
          <h1 className="text-lg font-semibold text-obsidian-50">用量统计</h1>
          <p className="text-xs text-obsidian-400 mt-0.5">查看 API 使用和消费趋势</p>
        </div>
        <div className="flex gap-1 bg-obsidian-800 rounded-lg p-0.5">
          {[7, 14, 30].map((d) => (
            <button
              key={d}
              onClick={() => setDays(d)}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all duration-200 ${
                days === d
                  ? "bg-amber-500 text-obsidian-950"
                  : "text-obsidian-400 hover:text-obsidian-200"
              }`}
            >
              {d} 天
            </button>
          ))}
        </div>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-3 gap-3 mb-5">
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-xs text-obsidian-400 mb-1">今日费用</div>
          <div className="text-xl font-bold text-obsidian-50">¥{(stats?.today_cost ?? 0).toFixed(4)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-xs text-obsidian-400 mb-1">本月费用</div>
          <div className="text-xl font-bold text-obsidian-50">¥{(stats?.month_cost ?? 0).toFixed(4)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="text-xs text-obsidian-400 mb-1">日均费用</div>
          <div className="text-xl font-bold text-obsidian-50">¥{(stats?.daily_average ?? 0).toFixed(4)}</div>
        </div>
      </div>

      {/* Cost trend chart */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4 mb-5">
        <div className="text-sm font-semibold text-obsidian-50 mb-1">每日消费趋势</div>
        <div className="text-xs text-obsidian-400 mb-3">近 {days} 天消费变化</div>
        {chartData.length === 0 ? (
          <div className="py-12 text-center text-sm text-obsidian-500">暂无数据</div>
        ) : (
          <div style={{ height: 200 }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                <defs>
                  <linearGradient id="usageCostGrad" x1="0" y1="0" x2="0" y2="1">
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
                <Area type="monotone" dataKey="cost" stroke="#f59e0b" strokeWidth={2} fill="url(#usageCostGrad)" dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      {/* Model breakdown */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
        <div className="text-sm font-semibold text-obsidian-50 mb-1">模型消费分布</div>
        <div className="text-xs text-obsidian-400 mb-3">按模型统计的消费占比</div>
        {modelData.length === 0 ? (
          <div className="py-12 text-center text-sm text-obsidian-500">暂无数据</div>
        ) : (
          <>
            <div style={{ height: 200 }}>
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={modelData} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#2a2a38" vertical={false} />
                  <XAxis dataKey="model" tick={{ fontSize: 9, fill: "#8888a0" }} axisLine={false} tickLine={false} />
                  <YAxis tick={{ fontSize: 11, fill: "#8888a0" }} axisLine={false} tickLine={false} />
                  <Tooltip
                    contentStyle={{ background: "#111118", border: "1px solid #2a2a38", borderRadius: "8px", fontSize: 12 }}
                    formatter={(v) => [`¥${Number(v).toFixed(4)}`, "消费"]}
                  />
                  <Bar dataKey="cost" fill="#f59e0b" radius={[4, 4, 0, 0]} fillOpacity={0.8} />
                </BarChart>
              </ResponsiveContainer>
            </div>

            <div className="mt-4 space-y-2">
              {modelData.map((m, i) => (
                <div key={i} className="flex items-center gap-3">
                  <span className="text-xs font-mono text-obsidian-200 flex-1 truncate">{m.model}</span>
                  <div className="w-32 h-1.5 rounded-full bg-obsidian-800">
                    <div
                      className="h-full rounded-full bg-amber-500"
                      style={{ width: `${totalCost > 0 ? (m.cost / totalCost * 100) : 0}%` }}
                    />
                  </div>
                  <span className="text-xs font-mono text-obsidian-300 w-20 text-right">¥{m.cost.toFixed(4)}</span>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
