import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  CreditCard, ChevronRight, Loader2, Wallet,
} from "lucide-react";
import {
  useTenantDetail,
  useTenantBalance,
  useTenantTransactions,
} from "@/hooks/use-tenant";
import { TenantPageHeader } from "@/components/tenant/TenantPageHeader";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 20;

const txTypeLabel: Record<string, string> = {
  consumption: "消费",
  recharge: "充值",
  freeze: "冻结",
  unfreeze: "解冻",
  settlement: "结算",
};

const txTypeBadge: Record<string, string> = {
  consumption: "badge-danger",
  recharge: "badge-success",
  freeze: "badge-warning",
  unfreeze: "badge-success",
  settlement: "badge-danger",
};

const TenantBillingPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const { data: tenant } = useTenantDetail(id!);
  const { data: balance, isLoading: balanceLoading } = useTenantBalance(id!);
  const { data: txData, isLoading: txLoading } = useTenantTransactions(id!, page, PAGE_SIZE);

  const transactions = txData?.transactions ?? [];
  const total = txData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  if (balanceLoading && txLoading) {
    return (
      <div className="page-container fade-in flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={28} className="animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">加载中...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="page-container fade-in">
      <TenantPageHeader
        title="财务管理"
        description="查看余额与组织资金流水"
        tenantName={tenant?.name}
        icon={CreditCard}
        onBack={() => navigate(`/dashboard/tenants/${id}`)}
      />

      {/* Balance Card */}
      <div className="bg-card border border-border rounded-xl p-6 shadow-card mb-6">
        <div className="flex items-center gap-2 mb-4">
          <div className="w-7 h-7 rounded-lg bg-primary/10 flex items-center justify-center">
            <Wallet size={14} className="text-primary" />
          </div>
          <h3 className="text-sm font-semibold text-foreground">余额信息</h3>
        </div>
        {balance ? (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4 sm:gap-6">
            <div className="stagger-item" style={{ animationDelay: "0ms" }}>
              <p className="text-xs text-muted-foreground mb-1">可用余额</p>
              <p className="text-2xl font-bold text-foreground">¥{balance.balance.toFixed(2)}</p>
            </div>
            <div className="stagger-item" style={{ animationDelay: "60ms" }}>
              <p className="text-xs text-muted-foreground mb-1">冻结金额</p>
              <p className="text-2xl font-bold text-amber-500">¥{balance.frozen.toFixed(2)}</p>
            </div>
            <div className="stagger-item" style={{ animationDelay: "120ms" }}>
              <p className="text-xs text-muted-foreground mb-1">累计充值</p>
              <p className="text-2xl font-bold text-green-600">¥{(balance.total_recharged ?? 0).toFixed(2)}</p>
            </div>
            <div className="stagger-item" style={{ animationDelay: "180ms" }}>
              <p className="text-xs text-muted-foreground mb-1">累计消费</p>
              <p className="text-2xl font-bold text-destructive">¥{(balance.total_consumed ?? 0).toFixed(2)}</p>
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">暂无余额信息</p>
        )}
      </div>

      {/* Transactions */}
      <div className="data-table-card">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div>
            <div className="text-sm font-semibold text-foreground">交易记录</div>
            <div className="text-xs text-muted-foreground mt-0.5">共 {total} 条记录</div>
          </div>
        </div>
        {txLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 size={16} className="animate-spin mr-2 text-primary" />
            <span className="text-sm text-muted-foreground">加载中...</span>
          </div>
        ) : transactions.length === 0 ? (
          <div className="empty-state">
            <div className="w-12 h-12 bg-muted rounded-full flex items-center justify-center mb-3">
              <CreditCard size={20} className="text-muted-foreground" />
            </div>
            <div className="text-sm font-medium text-muted-foreground">暂无交易记录</div>
          </div>
        ) : (
          <>
            <Table className="w-full text-sm">
                <TableHeader>
                  <TableRow className="table-header">
                    <TableHead className="text-left px-5 py-3">类型</TableHead>
                    <TableHead className="text-left px-5 py-3">金额</TableHead>
                    <TableHead className="text-left px-5 py-3">余额</TableHead>
                    <TableHead className="text-left px-5 py-3">模型</TableHead>
                    <TableHead className="text-left px-5 py-3">描述</TableHead>
                    <TableHead className="text-left px-5 py-3">时间</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {transactions.map((tx) => (
                    <TableRow key={tx.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                      <TableCell className="px-5 py-3.5">
                        <span className={txTypeBadge[tx.type] ?? "badge-neutral"}>
                          {txTypeLabel[tx.type] ?? tx.type}
                        </span>
                      </TableCell>
                      <TableCell className={`px-5 py-3.5 font-mono text-sm font-medium ${tx.amount > 0 ? "text-success" : "text-destructive"}`}>
                        {tx.amount > 0 ? "+" : ""}{tx.amount.toFixed(4)}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 font-mono text-sm">{tx.balance_after.toFixed(4)}</TableCell>
                      <TableCell className="px-5 py-3.5 text-muted-foreground">{tx.model || "-"}</TableCell>
                      <TableCell className="px-5 py-3.5 text-muted-foreground">{tx.description || "-"}</TableCell>
                      <TableCell className="px-5 py-3.5 text-muted-foreground">
                        {new Date(tx.created_at).toLocaleString("zh-CN")}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
                        {totalPages > 1 && (
              <div className="px-5 py-3.5 border-t border-border flex items-center justify-between">
                <div className="text-xs text-muted-foreground">共 {total} 条记录</div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                    className="flex h-9 w-9 items-center justify-center rounded-lg text-sm transition-colors hover:bg-muted/60 disabled:text-muted-foreground disabled:cursor-not-allowed"
                  >
                    <ChevronLeft size={14} />
                  </button>
                  <span className="text-sm text-foreground px-2">
                    <span className="font-medium">{page}</span>
                    <span className="text-muted-foreground"> / {totalPages}</span>
                  </span>
                  <button
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page === totalPages}
                    className="flex h-9 w-9 items-center justify-center rounded-lg text-sm transition-colors hover:bg-muted/60 disabled:text-muted-foreground disabled:cursor-not-allowed"
                  >
                    <ChevronRight size={14} />
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default TenantBillingPage;
