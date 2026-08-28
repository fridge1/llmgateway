import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  BarChart2, ChevronRight, Loader2, Filter, Download,
} from "lucide-react";
import {
  useTenantDetail,
  useTenantSubUsers,
  useTenantAllSubUserTransactions,
  exportTenantTransactions,
} from "@/hooks/use-tenant";
import { TenantPageHeader } from "@/components/tenant/TenantPageHeader";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 20;

const TenantUsageRecordsPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(PAGE_SIZE);
  const [selectedSubUser, setSelectedSubUser] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [exporting, setExporting] = useState(false);

  const { data: tenant } = useTenantDetail(id!);
  const { data: subUsers } = useTenantSubUsers(id!);
  const { data: txData, isLoading } = useTenantAllSubUserTransactions(
    id!, page, size, selectedSubUser || undefined
  );

  const transactions = txData?.transactions ?? [];
  const total = txData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  const handleSubUserChange = (subUserId: string) => {
    setSelectedSubUser(subUserId);
    setPage(1);
  };

  const handleExport = async () => {
    setExporting(true);
    try {
      await exportTenantTransactions(id!, startDate || undefined, endDate || undefined, selectedSubUser || undefined);
    } catch {
      // silently fail — browser will show download error
    } finally {
      setExporting(false);
    }
  };

  return (
    <div className="page-container fade-in">
      <TenantPageHeader
        title="使用记录"
        description="查看组织下所有成员的 API 调用与消费情况"
        tenantName={tenant?.name}
        icon={BarChart2}
        onBack={() => navigate(`/dashboard/tenants/${id}`)}
      />

      {/* Filter bar */}
      <div className="flex flex-wrap items-center gap-3 mb-6">
        <div className="flex items-center gap-2 bg-card border border-border rounded-lg px-3 py-2 text-sm shadow-card">
          <Filter size={14} className="text-muted-foreground" />
          <select
            className="bg-transparent text-sm text-foreground outline-none cursor-pointer"
            value={selectedSubUser}
            onChange={(e) => handleSubUserChange(e.target.value)}
          >
            <option value="">全部子用户</option>
            {(subUsers ?? []).map((su) => (
              <option key={su.id} value={su.id}>
                {su.nickname || su.username}
              </option>
            ))}
          </select>
        </div>
        <input
          type="date"
          value={startDate}
          onChange={(e) => setStartDate(e.target.value)}
          className="bg-card border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none shadow-card"
          placeholder="开始日期"
        />
        <span className="text-xs text-muted-foreground">至</span>
        <input
          type="date"
          value={endDate}
          onChange={(e) => setEndDate(e.target.value)}
          className="bg-card border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none shadow-card"
          placeholder="结束日期"
        />
        <button
          onClick={handleExport}
          disabled={exporting}
          className="btn-primary flex items-center gap-1.5 disabled:opacity-50"
        >
          {exporting ? <Loader2 size={14} className="animate-spin" /> : <Download size={14} />}
          导出 Excel
        </button>
        <div className="text-xs text-muted-foreground">
          共 {total} 条记录
        </div>
      </div>

      {/* Table */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 size={16} className="animate-spin mr-2 text-primary" />
          <span className="text-sm text-muted-foreground">加载中...</span>
        </div>
      ) : transactions.length === 0 ? (
        <div className="empty-state border border-border rounded-xl shadow-card">
          <div className="w-14 h-14 bg-muted rounded-full flex items-center justify-center mb-4">
            <BarChart2 size={24} className="text-muted-foreground" />
          </div>
          <p className="text-sm font-medium text-muted-foreground">暂无使用记录</p>
          <p className="text-xs text-muted-foreground/70 mt-1">子用户使用 API 调用模型后会自动生成记录</p>
        </div>
      ) : (
        <div className="data-table-card">
          <Table className="w-full">
              <TableHeader>
                <TableRow className="table-header">
                  <TableHead className="px-4 py-3 text-left">时间</TableHead>
                  <TableHead className="px-4 py-3 text-left">子用户</TableHead>
                  <TableHead className="px-4 py-3 text-left">模型</TableHead>
                  <TableHead className="px-4 py-3 text-right">输入</TableHead>
                  <TableHead className="px-4 py-3 text-right">输出</TableHead>
                  <TableHead className="px-4 py-3 text-right">缓存命中</TableHead>
                  <TableHead className="px-4 py-3 text-right">缓存写入</TableHead>
                  <TableHead className="px-4 py-3 text-right">金额</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {transactions.map((tx) => (
                  <TableRow
                    key={tx.id}
                    className="border-t border-border hover:bg-muted/30 transition-colors"
                  >
                    <TableCell className="px-4 py-3.5 text-sm text-muted-foreground">
                      {new Date(tx.created_at).toLocaleString("zh-CN")}
                    </TableCell>
                    <TableCell className="px-4 py-3.5 text-sm text-foreground">
                      {tx.sub_user_username || "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3.5 text-sm font-mono text-foreground">
                      {tx.model || "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3.5 text-sm text-muted-foreground text-right font-mono">
                      {tx.prompt_tokens != null ? tx.prompt_tokens.toLocaleString() : "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3.5 text-sm text-muted-foreground text-right font-mono">
                      {tx.completion_tokens != null ? tx.completion_tokens.toLocaleString() : "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3.5 text-sm text-muted-foreground text-right font-mono">
                      {tx.cache_read_tokens != null ? tx.cache_read_tokens.toLocaleString() : "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3.5 text-sm text-muted-foreground text-right font-mono">
                      {tx.cache_creation_tokens != null ? tx.cache_creation_tokens.toLocaleString() : "—"}
                    </TableCell>
                    <TableCell className="px-4 py-3.5 text-sm font-medium text-right text-destructive">
                      -¥{Math.abs(tx.amount).toFixed(4)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
                    {/* Pagination */}
          <div className="px-5 py-3.5 border-t border-border flex items-center justify-between">
            <div className="text-xs text-muted-foreground">共 {total} 条记录</div>
            <div className="flex items-center gap-2">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
                className="flex h-9 w-9 items-center justify-center rounded-lg text-sm transition-colors hover:bg-muted/60 disabled:text-muted-foreground disabled:cursor-not-allowed"
              >
                <ChevronLeft size={14} />
              </button>
              <span className="text-sm text-foreground px-2">
                <span className="font-medium">{page}</span>
                <span className="text-muted-foreground"> / {totalPages}</span>
              </span>
              <button
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
                className="flex h-9 w-9 items-center justify-center rounded-lg text-sm transition-colors hover:bg-muted/60 disabled:text-muted-foreground disabled:cursor-not-allowed"
              >
                <ChevronRight size={14} />
              </button>
              <select
                className="ml-2 bg-muted border border-border rounded-lg px-2 py-1 text-xs text-foreground outline-none cursor-pointer"
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
      )}
    </div>
  );
};

export default TenantUsageRecordsPage;
