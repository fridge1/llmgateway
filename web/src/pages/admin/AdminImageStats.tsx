import { useState } from "react";
import { Image, Clock, Loader } from "lucide-react";
import { useAdminImageDurationStats } from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const dayOptions = [
  { label: "7 天", value: 7 },
  { label: "30 天", value: 30 },
  { label: "90 天", value: 90 },
  { label: "365 天", value: 365 },
];

function fmtSeconds(s: number): string {
  if (s >= 60) return (s / 60).toFixed(1) + " 分钟";
  return s.toFixed(1) + " 秒";
}

const AdminImageStats = () => {
  const [days, setDays] = useState(30);
  const { data, isLoading } = useAdminImageDurationStats(days);
  const models = data?.models ?? [];

  const overallAvg = models.length > 0
    ? models.reduce((sum, m) => sum + m.avg_seconds * m.request_count, 0) /
      models.reduce((sum, m) => sum + m.request_count, 0)
    : 0;
  const overallMax = models.reduce((max, m) => Math.max(max, m.max_seconds), 0);
  const totalRequests = models.reduce((sum, m) => sum + m.request_count, 0);

  if (isLoading) {
    return (
      <div className="page-container fade-in flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader size={28} className="animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">加载中...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="page-container fade-in">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-lg font-bold text-foreground">图片生成时长统计</h1>
          <p className="text-sm text-muted-foreground mt-0.5">各图片模型的最短、平均、最长生成耗时</p>
        </div>
        <div className="flex items-center gap-1 bg-muted rounded-lg p-1">
          {dayOptions.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setDays(opt.value)}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
                days === opt.value
                  ? "bg-card text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-1 gap-4 mb-6 sm:grid-cols-3">
        {[
          { label: "平均生成时长", value: fmtSeconds(overallAvg), icon: Clock, iconBg: "bg-primary/10", iconColor: "text-primary" },
          { label: "最长单次耗时", value: fmtSeconds(overallMax), icon: Clock, iconBg: "bg-red-500/10 dark:bg-red-500/15", iconColor: "text-red-500" },
          { label: "统计总请求数", value: totalRequests.toLocaleString(), icon: Image, iconBg: "bg-violet-500/10 dark:bg-violet-500/15", iconColor: "text-violet-500" },
        ].map((card, i) => {
          const Icon = card.icon;
          return (
            <div key={card.label} className="stagger-item flex-1 bg-card border border-border rounded-xl p-5 shadow-card hover:shadow-elevated hover:-translate-y-0.5 transition-all duration-200" style={{ animationDelay: `${i * 80}ms` }}>
              <div className="flex items-start justify-between mb-3">
                <div className={`w-9 h-9 ${card.iconBg} rounded-lg flex items-center justify-center`}>
                  <Icon size={17} className={card.iconColor} />
                </div>
                <span className="text-xs text-muted-foreground">{card.label}</span>
              </div>
              <div className="text-2xl font-bold text-foreground">{card.value}</div>
            </div>
          );
        })}
      </div>

      {/* Table */}
      <div className="data-table-card">
        <div className="px-5 py-4 border-b border-border">
          <div className="text-sm font-semibold text-foreground">各模型生成时长明细</div>
          <div className="text-xs text-muted-foreground mt-0.5">仅统计 status=completed 且有完整起止时间的任务</div>
        </div>
        <div className="overflow-auto">
          <Table className="w-full">
            <TableHeader className="sticky top-0 z-10">
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3 font-semibold">模型</TableHead>
                <TableHead className="text-right px-5 py-3 font-semibold">完成请求数</TableHead>
                <TableHead className="text-right px-5 py-3 font-semibold">最短时长</TableHead>
                <TableHead className="text-right px-5 py-3 font-semibold">平均时长</TableHead>
                <TableHead className="text-right px-5 py-3 font-semibold">最长时长</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {models.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="px-5 py-12 text-center text-sm text-muted-foreground">
                    暂无数据
                  </TableCell>
                </TableRow>
              ) : (
                models.map((m, i) => (
                  <TableRow key={m.model} className={`border-t border-border hover:bg-muted/40 transition-colors ${i % 2 === 0 ? "" : "bg-muted/10"}`}>
                    <TableCell className="px-5 py-3 text-sm font-mono text-foreground whitespace-nowrap">{m.model}</TableCell>
                    <TableCell className="px-5 py-3 text-sm text-right text-muted-foreground tabular-nums">{m.request_count.toLocaleString()}</TableCell>
                    <TableCell className="px-5 py-3 text-sm text-right text-emerald-600 dark:text-emerald-400 tabular-nums font-medium">{fmtSeconds(m.min_seconds)}</TableCell>
                    <TableCell className="px-5 py-3 text-sm text-right text-foreground tabular-nums font-semibold">{fmtSeconds(m.avg_seconds)}</TableCell>
                    <TableCell className="px-5 py-3 text-sm text-right text-red-600 dark:text-red-400 tabular-nums font-medium">{fmtSeconds(m.max_seconds)}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
};

export default AdminImageStats;
