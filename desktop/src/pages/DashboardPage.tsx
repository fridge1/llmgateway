import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from "recharts";
import { useTools } from "../hooks/useTools";
import { useModels } from "../hooks/useModels";
import { useGateway } from "../hooks/useGateway";
import { useBalance, useBillingStats, useApiKeys, useTransactions, useTokenStats } from "@/hooks/use-api";
import ToolCard from "../components/ToolCard";
import StaggerContainer, { StaggerItem } from "../components/StaggerContainer";
import EmptyState from "../components/EmptyState";
import {
  Wallet, Terminal, Wrench, RefreshCw, CheckCircle2, AlertCircle,
  BarChart3, Key, Loader2, Plus, Zap
} from "../components/icons";
import { isWindows } from "../lib/utils";

function formatTime(iso: string): string {
  const d = new Date(iso);
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  const hh = String(d.getHours()).padStart(2, "0");
  const mi = String(d.getMinutes()).padStart(2, "0");
  return `${mm}-${dd} ${hh}:${mi}`;
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

export default function DashboardPage() {
  const navigate = useNavigate();
  const { tools, scanning, configuring, scan, configureAll, configureTool, clearTool } = useTools();
  const { models, loading: modelsLoading } = useModels();
  useGateway();
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const balance = useBalance();
  const billingStats = useBillingStats(7);
  const keys = useApiKeys();
  const txQuery = useTransactions(1, 5);
  const [tokenDays, setTokenDays] = useState<7 | 30 | 90>(7);
  const tokenStats = useTokenStats(tokenDays);

  const statsLoading = balance.isLoading || billingStats.isLoading || keys.isLoading;
  const unconfiguredCount = tools.filter(t => !t.configured).length;

  const chartData = billingStats.data?.daily_trend ?? [];
  const modelData = billingStats.data?.model_breakdown ?? [];
  const txList = txQuery.data?.transactions ?? [];

  const handleConfigure = async () => {
    setMessage(null);
    try {
      const unconfigured = tools.filter(t => !t.configured);
      if (unconfigured.length === 0) {
        setMessage({ type: "success", text: "所有工具已配置，无需操作" });
        return;
      }
      const results = await configureAll();
      if (results.length === 0) {
        setMessage({ type: "error", text: "未执行任何配置操作" });
        return;
      }
      const failed = results.filter(r => !r.success);
      const successCount = results.length - failed.length;
      if (failed.length > 0) {
        setMessage({ type: "error", text: `${successCount} 个成功，${failed.length} 个失败：${failed.map(r => `${r.tool}: ${r.message}`).join("；")}` });
      } else {
        setMessage({ type: "success", text: `${successCount} 个工具配置成功。${isWindows ? "请新开终端窗口后生效" : "已打开的终端需要新开窗口或执行 source ~/.zshrc 后生效"}` });
      }
    } catch (err) {
      setMessage({ type: "error", text: "配置失败: " + String(err) });
    }
  };

  const handleReconfigure = async () => {
    setMessage(null);
    try {
      const results = await configureAll(true);
      if (results.length === 0) {
        setMessage({ type: "error", text: "未执行任何配置操作" });
        return;
      }
      const failed = results.filter(r => !r.success);
      const successCount = results.length - failed.length;
      if (failed.length > 0) {
        setMessage({ type: "error", text: `${successCount} 个成功，${failed.length} 个失败：${failed.map(r => `${r.tool}: ${r.message}`).join("；")}` });
      } else {
        setMessage({ type: "success", text: `${successCount} 个工具重新配置成功` });
      }
    } catch (err) {
      setMessage({ type: "error", text: "配置失败: " + String(err) });
    }
  };

  const statCards = [
    {
      label: "总余额",
      value: `¥${(balance.data?.balance ?? 0).toFixed(4)}`,
      sub: `冻结 ¥${(balance.data?.frozen ?? 0).toFixed(4)}`,
      icon: Wallet,
      accent: true,
    },
    {
      label: "今日消费",
      value: `¥${(billingStats.data?.today_cost ?? 0).toFixed(4)}`,
      sub: "今日累计",
      icon: BarChart3,
    },
    {
      label: "本月消费",
      value: `¥${(billingStats.data?.month_cost ?? 0).toFixed(4)}`,
      sub: "本月累计",
      icon: BarChart3,
    },
    {
      label: "API 密钥",
      value: `${(keys.data ?? []).length}`,
      sub: `${(keys.data ?? []).length} 个活跃`,
      icon: Key,
    },
  ];

  return (
    <div className="p-6">
      {/* Stats cards */}
      {statsLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 size={24} className="text-amber-400 animate-spin" />
        </div>
      ) : (
        <div className="grid grid-cols-4 gap-3 mb-6">
          {statCards.map((s) => {
            const Icon = s.icon;
            return (
              <div
                key={s.label}
                className={`rounded-xl p-4 border transition-all duration-200 ${
                  s.accent
                    ? "bg-gradient-to-br from-amber-500 to-amber-600 border-amber-400/30"
                    : "bg-obsidian-900 border-obsidian-700 hover:border-obsidian-600"
                }`}
              >
                <div className="flex items-center justify-between mb-2">
                  <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${
                    s.accent ? "bg-white/15" : "bg-obsidian-800"
                  }`}>
                    <Icon size={16} className={s.accent ? "text-white" : "text-obsidian-400"} />
                  </div>
                  {s.accent && (
                    <button
                      onClick={() => navigate("/balance")}
                      className="flex items-center gap-1 px-2 py-1 rounded-lg bg-white/20 hover:bg-white/30 text-white text-xs font-medium transition-colors"
                    >
                      <Plus size={12} />
                      充值
                    </button>
                  )}
                </div>
                <div className={`text-xl font-bold mb-0.5 ${s.accent ? "text-white" : "text-obsidian-50"}`}>
                  {s.value}
                </div>
                <div className={`text-xs ${s.accent ? "text-white/70" : "text-obsidian-400"}`}>
                  {s.label} · {s.sub}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Charts */}
      <div className="grid grid-cols-5 gap-3 mb-6">
        <div className="col-span-3 bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="flex items-center justify-between mb-3">
            <div>
              <div className="text-sm font-semibold text-obsidian-50">费用趋势</div>
              <div className="text-xs text-obsidian-400 mt-0.5">近 7 天消费变化</div>
            </div>
            <button onClick={() => navigate("/transactions")} className="text-xs text-amber-400 hover:text-amber-300 font-medium">
              查看详情 →
            </button>
          </div>
          <div style={{ height: 160 }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData} margin={{ top: 5, right: 5, bottom: 0, left: -20 }}>
                <defs>
                  <linearGradient id="colorCost" x1="0" y1="0" x2="0" y2="1">
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
                <Area type="monotone" dataKey="cost" stroke="#f59e0b" strokeWidth={2} fill="url(#colorCost)" dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="col-span-2 bg-obsidian-900 border border-obsidian-700 rounded-xl p-4">
          <div className="mb-3">
            <div className="text-sm font-semibold text-obsidian-50">模型用量</div>
            <div className="text-xs text-obsidian-400 mt-0.5">模型消费分布</div>
          </div>
          <div style={{ height: 160 }}>
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
        </div>
      </div>

      {/* Token usage card */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4 mb-6">
        <div className="flex items-center justify-between mb-3">
          <div>
            <div className="text-sm font-semibold text-obsidian-50">Token 用量</div>
            <div className="text-xs text-obsidian-400 mt-0.5">最近 {tokenDays} 天 · 含累计</div>
          </div>
          <div className="flex items-center gap-1 bg-obsidian-800 rounded-lg p-0.5">
            {([7, 30, 90] as const).map((d) => (
              <button
                key={d}
                onClick={() => setTokenDays(d)}
                className={`px-2.5 py-1 text-xs font-medium rounded-md transition-colors ${
                  tokenDays === d
                    ? "bg-amber-500 text-obsidian-950"
                    : "text-obsidian-400 hover:text-obsidian-200"
                }`}
              >
                {d} 天
              </button>
            ))}
          </div>
        </div>
        <div className="grid grid-cols-4 gap-3">
          {([
            { label: "输入", key: "prompt" as const, color: "text-sky-400" },
            { label: "输出", key: "completion" as const, color: "text-emerald-400" },
            { label: "缓存命中", key: "cache_read" as const, color: "text-violet-400" },
            { label: "缓存写入", key: "cache_creation" as const, color: "text-amber-400" },
          ]).map((c) => {
            const periodVal = tokenStats.data?.period[c.key] ?? 0;
            const allTimeVal = tokenStats.data?.all_time[c.key] ?? 0;
            return (
              <div key={c.label} className="bg-obsidian-800/60 border border-obsidian-700 rounded-lg p-3">
                <div className="text-xs text-obsidian-500 mb-1">{c.label}</div>
                <div className={`text-xl font-bold ${c.color}`} title={periodVal.toLocaleString()}>
                  {formatTokens(periodVal)}
                </div>
                <div className="text-xs text-obsidian-500 mt-1" title={allTimeVal.toLocaleString()}>
                  累计 {formatTokens(allTimeVal)}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Claude overview cards */}
      {(() => {
        const claudeModels = modelData.filter((m) => m.model.toLowerCase().includes("claude"));
        const claudeTotalCost = claudeModels.reduce((sum, m) => sum + m.cost, 0);
        const topClaudeModel = claudeModels.length > 0
          ? claudeModels.reduce((max, m) => m.cost > max.cost ? m : max, claudeModels[0])
          : null;
        if (claudeTotalCost <= 0) return null;
        return (
          <div className="grid grid-cols-3 gap-3 mb-6">
            <div className="bg-orange-500/10 border border-orange-500/20 rounded-xl p-4">
              <div className="flex items-center gap-2 mb-2">
                <div className="w-7 h-7 bg-orange-500 rounded-lg flex items-center justify-center">
                  <span className="text-white font-bold text-xs">C</span>
                </div>
                <div className="text-xs font-semibold text-orange-300">Claude 总消费</div>
              </div>
              <div className="text-xl font-bold text-orange-200">¥{claudeTotalCost.toFixed(4)}</div>
              <div className="text-xs text-orange-400 mt-0.5">本月累计 · {claudeModels.length} 个模型</div>
            </div>
            <div className="bg-indigo-500/10 border border-indigo-500/20 rounded-xl p-4">
              <div className="flex items-center gap-2 mb-2">
                <div className="w-7 h-7 bg-indigo-500 rounded-lg flex items-center justify-center">
                  <Zap size={14} className="text-white" />
                </div>
                <div className="text-xs font-semibold text-indigo-300">最常用模型</div>
              </div>
              <div className="text-base font-bold text-indigo-200 truncate">{topClaudeModel?.model || "—"}</div>
              <div className="text-xs text-indigo-400 mt-0.5">消费 ¥{(topClaudeModel?.cost ?? 0).toFixed(4)}</div>
            </div>
            <div className="bg-violet-500/10 border border-violet-500/20 rounded-xl p-4">
              <div className="flex items-center gap-2 mb-2">
                <div className="w-7 h-7 bg-violet-500 rounded-lg flex items-center justify-center">
                  <BarChart3 size={14} className="text-white" />
                </div>
                <div className="text-xs font-semibold text-violet-300">省钱提示</div>
              </div>
              <div className="text-sm font-bold text-violet-200">使用 Prompt Cache</div>
              <div className="text-xs text-violet-400 mt-0.5">可节省 90% 输入成本</div>
            </div>
          </div>
        );
      })()}

      {/* Recent transactions */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden mb-6">
        <div className="px-4 py-3 border-b border-obsidian-700 flex items-center justify-between">
          <div className="text-sm font-semibold text-obsidian-50">最近交易</div>
          <button onClick={() => navigate("/transactions")} className="text-xs text-amber-400 hover:text-amber-300 font-medium">
            查看全部 →
          </button>
        </div>
        {txList.length === 0 ? (
          <div className="py-10 text-center text-sm text-obsidian-500">暂无交易记录</div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
                <th className="text-left px-4 py-2 font-medium">时间</th>
                <th className="text-left px-4 py-2 font-medium">类型</th>
                <th className="text-left px-4 py-2 font-medium">金额</th>
                <th className="text-left px-4 py-2 font-medium">余额</th>
                <th className="text-left px-4 py-2 font-medium">模型/描述</th>
              </tr>
            </thead>
            <tbody>
              {txList.map((t) => {
                const isRecharge = t.type === "recharge";
                const isSubPurchase = t.type === "sub_purchase";
                const isSubUsage = t.type === "subscription_usage";
                return (
                  <tr key={t.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                    <td className="px-4 py-2.5 text-xs text-obsidian-400">{formatTime(t.created_at)}</td>
                    <td className="px-4 py-2.5">
                      {isRecharge ? (
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400">充值</span>
                      ) : isSubPurchase ? (
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-indigo-500/10 text-indigo-400">套餐购买</span>
                      ) : isSubUsage ? (
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-violet-500/10 text-violet-400">订阅消费</span>
                      ) : (
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-500/10 text-red-400">消费</span>
                      )}
                    </td>
                    <td className={`px-4 py-2.5 text-xs font-medium ${isRecharge ? "text-emerald-400" : "text-red-400"}`}>
                      {isRecharge ? "+" : "-"}¥{Math.abs(t.amount).toFixed(4)}
                    </td>
                    <td className="px-4 py-2.5 text-xs text-obsidian-200 font-medium">¥{t.balance_after.toFixed(4)}</td>
                    <td className="px-4 py-2.5 text-xs text-obsidian-400 truncate max-w-[200px]">{t.model ?? t.description ?? "—"}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* AI Tools (desktop-unique) */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold flex items-center gap-2 text-obsidian-200">
            <Wrench size={16} className="text-obsidian-400" />
            AI 工具
          </h2>
          <button
            onClick={scan}
            disabled={scanning}
            className="text-xs text-obsidian-400 hover:text-obsidian-200 disabled:text-obsidian-600 flex items-center gap-1.5 transition-colors"
          >
            <RefreshCw size={14} className={scanning ? "animate-spin" : ""} />
            重新扫描
          </button>
        </div>

        {tools.length === 0 && !scanning && (
          <EmptyState
            icon={Terminal}
            title="未检测到已安装的 AI 工具"
            description="请先安装支持的 AI 开发工具"
          />
        )}

        <StaggerContainer className="grid grid-cols-1 gap-3">
          {tools.map(tool => (
            <StaggerItem key={tool.tool}>
              <ToolCard
                tool={tool}
                models={models}
                modelsLoading={modelsLoading}
                onConfigure={configureTool}
                onClear={clearTool}
              />
            </StaggerItem>
          ))}
        </StaggerContainer>
      </div>

      {/* Config message */}
      {message && (
        <div className={`flex items-start gap-2 p-3 rounded-lg mb-4 text-sm ${
          message.type === "success"
            ? "bg-emerald-500/10 text-emerald-400"
            : "bg-red-500/10 text-red-400"
        }`}>
          {message.type === "success"
            ? <CheckCircle2 size={16} className="mt-0.5 shrink-0" />
            : <AlertCircle size={16} className="mt-0.5 shrink-0" />
          }
          <span>{message.text}</span>
        </div>
      )}

      {/* Configure button */}
      {tools.length > 0 && (
        <div className="flex gap-3">
          {unconfiguredCount > 0 && (
            <button
              onClick={handleConfigure}
              disabled={configuring}
              className="px-4 py-2 bg-amber-500 hover:bg-amber-400 hover:shadow-glow-amber disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200 inline-flex items-center gap-1.5"
            >
              {configuring ? <Loader2 size={16} className="animate-spin" /> : null}
              一键配置 ({unconfiguredCount})
            </button>
          )}
          <button
            onClick={handleReconfigure}
            disabled={configuring}
            className="px-4 py-2 bg-obsidian-800 hover:bg-obsidian-700 disabled:text-obsidian-600 text-obsidian-200 text-sm rounded-lg transition-colors inline-flex items-center gap-1.5"
          >
            <RefreshCw size={14} />
            全部重新配置
          </button>
        </div>
      )}
    </div>
  );
}
