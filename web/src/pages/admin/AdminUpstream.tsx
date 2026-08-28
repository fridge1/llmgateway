import { useState, useMemo } from "react";
import { RefreshCw, Link, AlertTriangle, ArrowUpDown, Loader } from "lucide-react";
import { useGatewayStatus, useTestUpstreamByName } from "@/hooks/use-api";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/query-keys";
import { toast } from "sonner";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
interface FlatUpstream {
  model: string;
  provider: string;
  baseUrl: string;
  state: string;
  failures: number;
}

const AdminUpstream = () => {
  const qc = useQueryClient();
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [sortField, setSortField] = useState<"model" | "failures" | null>(null);
  const [sortAsc, setSortAsc] = useState(true);
  const [testingKey, setTestingKey] = useState<string | null>(null);
  const testMutation = useTestUpstreamByName();

  const { data: status, isLoading, isFetching, dataUpdatedAt } = useGatewayStatus({
    refetchInterval: autoRefresh ? 30_000 : false,
  });

  // Flatten models Record into rows
  const upstreamRows = useMemo((): FlatUpstream[] => {
    if (!status?.models) return [];
    const rows: FlatUpstream[] = [];
    for (const [modelName, info] of Object.entries(status.models)) {
      for (const u of info.upstreams) {
        rows.push({
          model: modelName,
          provider: u.provider,
          baseUrl: u.base_url,
          state: u.state,
          failures: u.failure_count,
        });
      }
    }
    return rows;
  }, [status]);

  const handleRefresh = () => {
    qc.invalidateQueries({ queryKey: queryKeys.status() });
  };

  const handleSort = (field: "model" | "failures") => {
    if (sortField === field) {
      setSortAsc((v) => !v);
    } else {
      setSortField(field);
      setSortAsc(true);
    }
  };

  const isNormal = (state: string) => state.toLowerCase() === "closed" || state.toLowerCase() === "normal";
  const normalCount = upstreamRows.filter((m) => isNormal(m.state)).length;
  const errorCount = upstreamRows.filter((m) => !isNormal(m.state)).length;

  const displayData = [...upstreamRows];
  if (sortField === "model") {
    displayData.sort((a, b) => sortAsc ? a.model.localeCompare(b.model) : b.model.localeCompare(a.model));
  } else if (sortField === "failures") {
    displayData.sort((a, b) => sortAsc ? a.failures - b.failures : b.failures - a.failures);
  }

  return (
    <div className="page-container">
      {/* Page header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-lg font-bold text-foreground">上游状态</h1>
          <div className="flex items-center gap-3 mt-1">
            <span className="text-sm text-muted-foreground">监控所有上游模型的连接状态</span>
            {!isLoading && (
              <>
                <span className="badge-success">
                  <span className="w-1.5 h-1.5 rounded-full bg-success inline-block" />
                  正常 {normalCount}
                </span>
                {errorCount > 0 && (
                  <span className="badge-danger">
                    <AlertTriangle size={10} />
                    异常 {errorCount}
                  </span>
                )}
              </>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {/* Auto refresh toggle */}
          <div className="flex items-center gap-2 bg-card border border-border rounded-lg px-3 py-1.5 shadow-card">
            {dataUpdatedAt > 0 && (
              <span className="text-xs text-muted-foreground tabular-nums">
                {new Date(dataUpdatedAt).toLocaleTimeString()}
              </span>
            )}
            <span className="text-xs text-muted-foreground">自动刷新</span>
            <button
              onClick={() => {
                const next = !autoRefresh;
                setAutoRefresh(next);
                toast(next ? "自动刷新已开启 (30秒)" : "自动刷新已关闭", { duration: 2000 });
              }}
              className={`relative w-9 h-5 rounded-full transition-colors duration-200 ${autoRefresh ? "bg-primary" : "bg-border"}`}
            >
              <span
                className="absolute bg-white rounded-full shadow-card transition-transform duration-200"
                style={{ width: 16, height: 16, top: 2, left: 0, transform: autoRefresh ? "translateX(18px)" : "translateX(2px)" }}
              />
            </button>
          </div>
          <button
            onClick={handleRefresh}
            className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground border border-border bg-card px-3 py-1.5 rounded-lg hover:bg-muted transition-colors shadow-card"
          >
            <RefreshCw size={13} className={isFetching ? "animate-spin" : ""} />
            刷新
          </button>
        </div>
      </div>

      {/* Table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3.5">
                  <button
                    className="flex items-center gap-1 hover:text-foreground transition-colors"
                    onClick={() => handleSort("model")}
                  >
                    模型
                    <ArrowUpDown size={12} className={sortField === "model" ? "text-primary" : ""} />
                  </button>
                </TableHead>
                <TableHead className="text-left px-5 py-3.5">提供商</TableHead>
                <TableHead className="text-left px-5 py-3.5">Base URL</TableHead>
                <TableHead className="text-left px-5 py-3.5">熔断器状态</TableHead>
                <TableHead className="text-left px-5 py-3.5">
                  <button
                    className="flex items-center gap-1 hover:text-foreground transition-colors"
                    onClick={() => handleSort("failures")}
                  >
                    失败次数
                    <ArrowUpDown size={12} className={sortField === "failures" ? "text-primary" : ""} />
                  </button>
                </TableHead>
                <TableHead className="text-right px-5 py-3.5">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {displayData.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="px-5 py-16 text-center text-sm text-muted-foreground">
                    暂无上游数据
                  </TableCell>
                </TableRow>
              ) : (
                displayData.map((m, i) => {
                  const normal = isNormal(m.state);
                  return (
                    <TableRow
                      key={`${m.model}-${m.provider}-${i}`}
                      className={`border-t border-border hover:bg-accent/30 transition-colors duration-150 ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                    >
                      <TableCell className="px-5 py-3 text-sm font-mono text-foreground">{m.model}</TableCell>
                      <TableCell className="px-5 py-3">
                        <span className="inline-flex items-center px-2 py-0.5 rounded-md bg-muted text-muted-foreground text-xs font-medium">
                          {m.provider}
                        </span>
                      </TableCell>
                      <TableCell className="px-5 py-3 text-sm text-muted-foreground font-mono">{m.baseUrl}</TableCell>
                      <TableCell className="px-5 py-3">
                        {normal ? (
                          <span className="inline-flex items-center gap-1.5 text-xs font-medium text-success">
                            <span className="w-1.5 h-1.5 rounded-full bg-success" />
                            正常
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1.5 text-xs font-medium text-destructive">
                            <span className="w-1.5 h-1.5 rounded-full bg-destructive" />
                            熔断
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="px-5 py-3 text-sm text-muted-foreground">{m.failures}</TableCell>
                      <TableCell className="px-5 py-3">
                        <div className="flex justify-end">
                          <button
                            className={`flex items-center gap-1.5 text-xs font-medium px-3 py-1.5 rounded-lg border transition-all duration-150 ${
                              testingKey === `${m.model}-${m.baseUrl}`
                                ? "text-primary border-primary/30 bg-primary/5"
                                : "text-muted-foreground border-border hover:text-foreground hover:border-foreground/20"
                            }`}
                            disabled={testingKey !== null}
                            onClick={() => {
                              const key = `${m.model}-${m.baseUrl}`;
                              setTestingKey(key);
                              testMutation.mutate(
                                { model: m.model, base_url: m.baseUrl },
                                {
                                  onSuccess: (data) => {
                                    if (data.success) {
                                      toast.success(`${data.message} (${data.latency})`);
                                    } else {
                                      toast.error(data.message);
                                    }
                                    setTestingKey(null);
                                  },
                                  onError: (err) => {
                                    toast.error(err instanceof Error ? err.message : "测试失败");
                                    setTestingKey(null);
                                  },
                                },
                              );
                            }}
                          >
                            {testingKey === `${m.model}-${m.baseUrl}` ? (
                              <Loader size={11} className="animate-spin" />
                            ) : (
                              <Link size={11} />
                            )}
                            测试
                          </button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  );
};

export default AdminUpstream;
