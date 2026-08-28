import { useState, useMemo } from "react";
import {
  useInvoiceTitles, useAvailableOrders, useCreateInvoiceRequest,
  useInvoiceRequests, useCancelInvoiceRequest,
} from "@/hooks/use-api";
import { Loader2 } from "../components/icons";

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString("zh-CN");
}

const statusSteps = ["已提交", "处理中", "已完成"];

function InvoiceStepper({ status }: { status: string }) {
  if (status === "cancelled" || status === "rejected") {
    const label = status === "cancelled" ? "已取消" : "已拒绝";
    const color = status === "rejected" ? "bg-red-500/10 text-red-400" : "bg-obsidian-700 text-obsidian-400";
    return <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${color}`}>{label}</span>;
  }
  const activeIdx = status === "pending" ? 0 : status === "processing" ? 1 : 2;
  return (
    <div className="flex items-center gap-1">
      {statusSteps.map((label, i) => {
        const done = i <= activeIdx;
        return (
          <div key={label} className="flex items-center gap-1">
            {i > 0 && <div className={`w-3 h-px ${done ? "bg-amber-500" : "bg-obsidian-700"}`} />}
            <div className="flex items-center gap-0.5">
              <div className={`w-3.5 h-3.5 rounded-full border ${done ? "border-amber-500 bg-amber-500/20" : "border-obsidian-600"}`} />
              <span className={`text-[10px] ${done ? "text-obsidian-200" : "text-obsidian-500"}`}>{label}</span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export default function InvoiceRequestsPage() {
  const [activeTab, setActiveTab] = useState<"apply" | "history">("apply");
  const [page, setPage] = useState(1);
  const size = 10;
  const { data: reqData, isLoading: reqLoading } = useInvoiceRequests(page, size);
  const { data: titles = [] } = useInvoiceTitles();
  const { data: availableOrders = [], isLoading: ordersLoading } = useAvailableOrders();
  const createReq = useCreateInvoiceRequest();
  const cancelReq = useCancelInvoiceRequest();
  const [selectedTitleId, setSelectedTitleId] = useState<number>(0);
  const [selectedOrders, setSelectedOrders] = useState<Set<string>>(new Set());
  const [invoiceType, setInvoiceType] = useState("normal");
  const [remark, setRemark] = useState("");

  const requests = reqData?.requests ?? [];
  const total = reqData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  const totalSelectedAmount = useMemo(
    () => availableOrders.filter((o) => selectedOrders.has(o.id)).reduce((sum, o) => sum + o.amount, 0),
    [availableOrders, selectedOrders]
  );

  const allSelected = availableOrders.length > 0 && selectedOrders.size === availableOrders.length;

  const toggleSelectAll = () => {
    setSelectedOrders(allSelected ? new Set() : new Set(availableOrders.map(o => o.id)));
  };

  const toggleOrder = (id: string) => {
    setSelectedOrders((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleCreate = async () => {
    if (selectedTitleId <= 0 || selectedOrders.size === 0) return;
    try {
      await createReq.mutateAsync({
        title_id: selectedTitleId,
        invoice_type: invoiceType,
        order_ids: Array.from(selectedOrders),
        remark: remark || undefined,
      });
      setSelectedOrders(new Set());
      setRemark("");
      setActiveTab("history");
    } catch {}
  };

  const handleCancel = (id: number) => {
    if (window.confirm("确定取消此发票申请吗？")) cancelReq.mutate(id);
  };

  const isLoading = activeTab === "apply" ? ordersLoading : reqLoading;

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
        <h1 className="text-lg font-semibold text-obsidian-50">我的发票</h1>
        <p className="text-xs text-obsidian-400 mt-0.5">申请和查看发票</p>
      </div>

      {/* Tab switcher */}
      <div className="flex gap-1 bg-obsidian-800 rounded-lg p-0.5 mb-5 w-fit">
        {([
          { key: "apply", label: "申请开票" },
          { key: "history", label: "开票记录" },
        ] as const).map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={`px-4 py-1.5 rounded-md text-xs font-medium transition-all duration-200 ${
              activeTab === tab.key ? "bg-amber-500 text-obsidian-950" : "text-obsidian-400 hover:text-obsidian-200"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Apply tab */}
      {activeTab === "apply" && (
        <>
          <p className="text-xs text-obsidian-400 mb-4">勾选需要开票的订单，合并提交开票申请</p>
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden mb-4">
            <table className="w-full">
              <thead>
                <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
                  <th className="text-left px-4 py-2 w-8">
                    <input type="checkbox" checked={allSelected} onChange={toggleSelectAll} className="accent-amber-500" />
                  </th>
                  <th className="text-left px-4 py-2 font-medium">订单号</th>
                  <th className="text-left px-4 py-2 font-medium">支付时间</th>
                  <th className="text-right px-4 py-2 font-medium">金额</th>
                </tr>
              </thead>
              <tbody>
                {availableOrders.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="px-4 py-10 text-center text-xs text-obsidian-500">暂无可开票订单</td>
                  </tr>
                ) : (
                  availableOrders.map((o) => (
                    <tr key={o.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                      <td className="px-4 py-2.5">
                        <input type="checkbox" checked={selectedOrders.has(o.id)} onChange={() => toggleOrder(o.id)} className="accent-amber-500" />
                      </td>
                      <td className="px-4 py-2.5 text-xs font-mono text-obsidian-300">{o.order_no}</td>
                      <td className="px-4 py-2.5 text-xs text-obsidian-400">{o.pay_time ? formatTime(o.pay_time) : "—"}</td>
                      <td className="px-4 py-2.5 text-sm font-medium text-obsidian-100 text-right">¥{o.amount.toFixed(2)}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Action bar */}
          <div className="bg-obsidian-800/50 rounded-xl p-4 flex items-center justify-between gap-3">
            <span className="text-xs text-obsidian-400">
              已选 <span className="font-medium text-obsidian-100">{selectedOrders.size}</span> 笔，合计 <span className="font-medium text-obsidian-100">¥{totalSelectedAmount.toFixed(2)}</span>
            </span>
            <div className="flex items-center gap-2">
              <select
                value={invoiceType}
                onChange={(e) => setInvoiceType(e.target.value)}
                className="bg-obsidian-800 border border-obsidian-700 rounded-lg px-2 py-1.5 text-xs text-obsidian-200 outline-none"
              >
                <option value="normal">普通发票</option>
                <option value="special">专用发票</option>
              </select>
              <select
                value={selectedTitleId}
                onChange={(e) => setSelectedTitleId(Number(e.target.value))}
                className="bg-obsidian-800 border border-obsidian-700 rounded-lg px-2 py-1.5 text-xs text-obsidian-200 outline-none"
              >
                <option value={0}>选择抬头</option>
                {titles.map((t) => <option key={t.id} value={t.id}>{t.title_name}</option>)}
              </select>
              <input
                className="px-2 py-1.5 bg-obsidian-800 border border-obsidian-700 rounded-lg text-xs text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50"
                placeholder="备注（可选）"
                value={remark}
                onChange={(e) => setRemark(e.target.value)}
              />
              <button
                onClick={handleCreate}
                disabled={createReq.isPending || selectedTitleId <= 0 || selectedOrders.size === 0}
                className="px-4 py-1.5 rounded-lg text-xs font-semibold bg-amber-500 hover:bg-amber-400 text-obsidian-950 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {createReq.isPending ? "提交中..." : "提交申请"}
              </button>
            </div>
          </div>
        </>
      )}

      {/* History tab */}
      {activeTab === "history" && (
        <>
          {requests.length === 0 ? (
            <div className="py-16 text-center text-sm text-obsidian-500">暂无开票记录</div>
          ) : (
            <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
              <table className="w-full">
                <thead>
                  <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
                    <th className="text-left px-4 py-2 font-medium">申请时间</th>
                    <th className="text-left px-4 py-2 font-medium">类型</th>
                    <th className="text-left px-4 py-2 font-medium">金额</th>
                    <th className="text-left px-4 py-2 font-medium">状态</th>
                    <th className="text-right px-4 py-2 font-medium">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {requests.map((r) => (
                    <tr key={r.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                      <td className="px-4 py-3 text-xs text-obsidian-400">{formatTime(r.created_at)}</td>
                      <td className="px-4 py-3 text-xs text-obsidian-300">{r.invoice_type === "special" ? "增值税专票" : "增值税普票"}</td>
                      <td className="px-4 py-3 text-sm font-medium text-obsidian-100">¥{r.total_amount.toFixed(2)}</td>
                      <td className="px-4 py-3"><InvoiceStepper status={r.status} /></td>
                      <td className="px-4 py-3 text-right">
                        {r.status === "completed" && (
                          <a
                            href={`/api/invoice/requests/${r.id}/download`}
                            target="_blank"
                            rel="noreferrer"
                            className="px-3 py-1 text-xs text-blue-400 hover:text-blue-300 transition-colors"
                          >
                            下载发票
                          </a>
                        )}
                        {r.status === "pending" && (
                          <button
                            onClick={() => handleCancel(r.id)}
                            className="px-3 py-1 text-xs text-obsidian-400 hover:text-red-400 transition-colors"
                          >
                            取消
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 mt-4">
              <button className="px-3 py-1.5 rounded-lg text-xs bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700 disabled:opacity-40 transition-colors" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</button>
              <span className="text-xs text-obsidian-400">{page} / {totalPages}</span>
              <button className="px-3 py-1.5 rounded-lg text-xs bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700 disabled:opacity-40 transition-colors" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
