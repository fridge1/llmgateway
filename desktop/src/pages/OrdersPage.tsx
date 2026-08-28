import { useState } from "react";
import { useOrders, useRepayOrder } from "@/hooks/use-api";
import { Loader2 } from "../components/icons";

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("zh-CN");
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { bg: string; text: string; label: string }> = {
    paid: { bg: "bg-emerald-500/10", text: "text-emerald-400", label: "已支付" },
    pending: { bg: "bg-amber-500/10", text: "text-amber-400", label: "待支付" },
    expired: { bg: "bg-obsidian-700", text: "text-obsidian-400", label: "已过期" },
  };
  const s = map[status] ?? { bg: "bg-obsidian-700", text: "text-obsidian-400", label: status };
  return <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${s.bg} ${s.text}`}>{s.label}</span>;
}

export default function OrdersPage() {
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(20);
  const { data, isLoading } = useOrders(page, size);
  const repayOrder = useRepayOrder();

  const orders = data?.orders ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));
  const statusCounts = data?.status_counts ?? { paid: 0, pending: 0, expired: 0 };

  const handleRepay = async (orderNo: string) => {
    try {
      const res = await repayOrder.mutateAsync({ order_no: orderNo, client_type: "desktop" });
      const { openUrl } = await import("@tauri-apps/plugin-opener");
      await openUrl(res.pay_url);
    } catch {}
  };

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
        <h1 className="text-lg font-semibold text-obsidian-50">订单</h1>
        <p className="text-xs text-obsidian-400 mt-0.5">查看充值订单记录</p>
      </div>

      {/* Status summary — 4 cards including total */}
      <div className="grid grid-cols-4 gap-3 mb-5">
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-3">
          <div className="text-[10px] text-obsidian-500">总订单</div>
          <div className="text-lg font-bold text-obsidian-50">{total}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-3">
          <div className="text-[10px] text-obsidian-500">已支付</div>
          <div className="text-lg font-bold text-emerald-400">{statusCounts.paid}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-3">
          <div className="text-[10px] text-obsidian-500">待支付</div>
          <div className="text-lg font-bold text-amber-400">{statusCounts.pending}</div>
        </div>
        <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-3">
          <div className="text-[10px] text-obsidian-500">已过期</div>
          <div className="text-lg font-bold text-obsidian-400">{statusCounts.expired}</div>
        </div>
      </div>

      {/* Orders table */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
              <th className="text-left px-4 py-2 font-medium">订单号</th>
              <th className="text-left px-4 py-2 font-medium">金额</th>
              <th className="text-left px-4 py-2 font-medium">支付方式</th>
              <th className="text-left px-4 py-2 font-medium">状态</th>
              <th className="text-left px-4 py-2 font-medium">创建时间</th>
              <th className="text-left px-4 py-2 font-medium">支付时间</th>
              <th className="text-right px-4 py-2 font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            {orders.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-4 py-12 text-center text-sm text-obsidian-500">暂无订单</td>
              </tr>
            ) : (
              orders.map((o) => (
                <tr key={o.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                  <td className="px-4 py-3 text-xs font-mono text-obsidian-300">{o.order_no}</td>
                  <td className="px-4 py-3 text-sm font-medium text-obsidian-100">¥{o.amount.toFixed(2)}</td>
                  <td className="px-4 py-3 text-xs text-obsidian-400">{o.pay_method || "—"}</td>
                  <td className="px-4 py-3"><StatusBadge status={o.status} /></td>
                  <td className="px-4 py-3 text-xs text-obsidian-400">{formatTime(o.created_at)}</td>
                  <td className="px-4 py-3 text-xs text-obsidian-400">{o.pay_time ? formatTime(o.pay_time) : "—"}</td>
                  <td className="px-4 py-3 text-right">
                    {o.status === "pending" && new Date(o.expired_at) > new Date() && (
                      <button
                        onClick={() => handleRepay(o.order_no)}
                        disabled={repayOrder.isPending}
                        className="px-3 py-1 bg-amber-500/10 text-amber-400 text-xs font-medium rounded-lg hover:bg-amber-500/20 transition-colors disabled:opacity-50"
                      >
                        继续支付
                      </button>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>

        {/* Pagination with page-size selector */}
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
