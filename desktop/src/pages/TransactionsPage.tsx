import { useState } from "react";
import { useTransactions } from "@/hooks/use-api";
import { Loader2 } from "../components/icons";

const tabs = [
  { key: "all", label: "全部" },
  { key: "consumption", label: "余额消费" },
  { key: "subscription_usage", label: "订阅消费" },
  { key: "sub_purchase", label: "套餐购买" },
  { key: "recharge", label: "充值" },
];

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("zh-CN");
}

function TypeBadge({ type }: { type: string }) {
  const styles: Record<string, string> = {
    consumption: "bg-red-500/10 text-red-400",
    subscription_usage: "bg-violet-500/10 text-violet-400",
    sub_purchase: "bg-indigo-500/10 text-indigo-400",
    recharge: "bg-emerald-500/10 text-emerald-400",
  };
  const labels: Record<string, string> = {
    consumption: "余额消费",
    subscription_usage: "订阅消费",
    sub_purchase: "套餐购买",
    recharge: "充值",
  };
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${styles[type] ?? "bg-obsidian-800 text-obsidian-400"}`}>
      {labels[type] ?? type}
    </span>
  );
}

export default function TransactionsPage() {
  const [activeTab, setActiveTab] = useState("all");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(20);

  const typeFilter = activeTab === "all" ? undefined : activeTab;
  const { data, isLoading } = useTransactions(page, size, typeFilter, startDate || undefined, endDate || undefined);
  const transactions = data?.transactions ?? [];
  const total = data?.total ?? 0;
  const totalConsumption = data?.total_consumption ?? 0;
  const totalRecharge = data?.total_recharge ?? 0;
  const totalSubscriptionUsage = data?.total_subscription_usage ?? 0;
  const totalSubscriptionPurchase = data?.total_sub_purchase ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));
  const net = totalRecharge - totalConsumption - totalSubscriptionPurchase;

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}>
        <Loader2 size={24} className="animate-spin text-amber-400" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-5">
        <h1 className="text-lg font-semibold text-obsidian-50">交易记录</h1>
        <p className="text-xs text-obsidian-400 mt-0.5">查看余额消费、订阅消费与充值记录</p>
      </div>

      {/* Summary stats */}
      <div className="grid grid-cols-3 gap-3 mb-5">
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-3">
          <div className="text-[10px] text-obsidian-500">总记录</div>
          <div className="text-lg font-bold text-obsidian-50">{total}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-3">
          <div className="text-[10px] text-obsidian-500">总消费</div>
          <div className="text-lg font-bold text-red-400">-¥{totalConsumption.toFixed(4)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-3">
          <div className="text-[10px] text-obsidian-500">总充值</div>
          <div className="text-lg font-bold text-emerald-400">+¥{totalRecharge.toFixed(4)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-3">
          <div className="text-[10px] text-obsidian-500">订阅消费</div>
          <div className="text-lg font-bold text-violet-400">¥{totalSubscriptionUsage.toFixed(4)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-3">
          <div className="text-[10px] text-obsidian-500">套餐购买</div>
          <div className="text-lg font-bold text-indigo-400">-¥{totalSubscriptionPurchase.toFixed(4)}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-3">
          <div className="text-[10px] text-obsidian-500">净额</div>
          <div className={`text-lg font-bold ${net >= 0 ? "text-amber-400" : "text-red-400"}`}>¥{net.toFixed(4)}</div>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex gap-1 bg-obsidian-800 rounded-lg p-0.5">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => { setActiveTab(tab.key); setPage(1); }}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all duration-200 ${
                activeTab === tab.key
                  ? "bg-amber-500 text-obsidian-950"
                  : "text-obsidian-400 hover:text-obsidian-200"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="flex gap-2">
          <input
            type="date"
            value={startDate}
            onChange={(e) => { setStartDate(e.target.value); setPage(1); }}
            className="px-2 py-1.5 bg-obsidian-800 border border-obsidian-700 rounded-lg text-xs text-obsidian-200 focus:outline-none focus:border-amber-500/50"
          />
          <input
            type="date"
            value={endDate}
            onChange={(e) => { setEndDate(e.target.value); setPage(1); }}
            className="px-2 py-1.5 bg-obsidian-800 border border-obsidian-700 rounded-lg text-xs text-obsidian-200 focus:outline-none focus:border-amber-500/50"
          />
        </div>
      </div>

      {/* Table */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
              <th className="text-left px-4 py-2 font-medium">时间</th>
              <th className="text-left px-4 py-2 font-medium">类型</th>
              <th className="text-left px-4 py-2 font-medium">模型</th>
              <th className="text-right px-4 py-2 font-medium">输入</th>
              <th className="text-right px-4 py-2 font-medium">输出</th>
              <th className="text-right px-4 py-2 font-medium">缓存命中</th>
              <th className="text-right px-4 py-2 font-medium">缓存写入</th>
              <th className="text-right px-4 py-2 font-medium">金额</th>
              <th className="text-right px-4 py-2 font-medium">余额</th>
            </tr>
          </thead>
          <tbody>
            {transactions.length === 0 ? (
              <tr>
                <td colSpan={9} className="px-4 py-12 text-center text-sm text-obsidian-500">暂无交易记录</td>
              </tr>
            ) : (
              transactions.map((t) => {
                const isRecharge = t.type === "recharge";
                const isSubscription = t.type === "subscription_usage";
                const isSubPurchase = t.type === "sub_purchase";
                const amountColor = isRecharge ? "text-emerald-400" : isSubscription ? "text-violet-400" : isSubPurchase ? "text-indigo-400" : "text-red-400";
                const cacheCreation = (t.cache_creation_5m_tokens ?? 0) + (t.cache_creation_1h_tokens ?? 0) || (t.cache_creation_tokens ?? null);
                return (
                  <tr key={t.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                    <td className="px-4 py-2.5 text-xs text-obsidian-400">{formatTime(t.created_at)}</td>
                    <td className="px-4 py-2.5"><TypeBadge type={t.type} /></td>
                    <td className="px-4 py-2.5 text-xs text-obsidian-400 font-mono truncate max-w-[120px]">{t.model ?? t.description ?? "—"}</td>
                    <td className="px-4 py-2.5 text-xs text-right text-obsidian-400 font-mono">
                      {t.prompt_tokens != null ? t.prompt_tokens.toLocaleString() : "—"}
                    </td>
                    <td className="px-4 py-2.5 text-xs text-right text-obsidian-400 font-mono">
                      {t.completion_tokens != null ? t.completion_tokens.toLocaleString() : "—"}
                    </td>
                    <td className="px-4 py-2.5 text-xs text-right text-obsidian-400 font-mono">
                      {t.cache_read_tokens != null ? t.cache_read_tokens.toLocaleString() : "—"}
                    </td>
                    <td className="px-4 py-2.5 text-xs text-right text-obsidian-400 font-mono">
                      {cacheCreation != null ? Number(cacheCreation).toLocaleString() : "—"}
                    </td>
                    <td className={`px-4 py-2.5 text-xs font-medium text-right ${amountColor}`}>
                      {isRecharge ? "+" : "-"}¥{Math.abs(t.amount).toFixed(4)}
                    </td>
                    <td className="px-4 py-2.5 text-xs text-right text-obsidian-200 font-medium">¥{t.balance_after.toFixed(4)}</td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>

        {/* Pagination */}
        <div className="px-4 py-3 border-t border-obsidian-800 flex items-center justify-between">
          <div className="text-xs text-obsidian-400">共 {total} 条记录</div>
          <div className="flex items-center gap-2">
            <button
              className="px-3 py-1.5 rounded-lg text-xs bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700 disabled:opacity-40 transition-colors"
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
            >上一页</button>
            <span className="text-xs text-obsidian-400">{page} / {totalPages}</span>
            <button
              className="px-3 py-1.5 rounded-lg text-xs bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700 disabled:opacity-40 transition-colors"
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
            >下一页</button>
            <select
              className="ml-1 bg-obsidian-800 border border-obsidian-700 rounded-lg px-2 py-1 text-xs text-obsidian-200 outline-none"
              value={size}
              onChange={(e) => { setSize(Number(e.target.value)); setPage(1); }}
            >
              <option value={10}>10条/页</option>
              <option value={20}>20条/页</option>
              <option value={50}>50条/页</option>
            </select>
          </div>
        </div>
      </div>
    </div>
  );
}
