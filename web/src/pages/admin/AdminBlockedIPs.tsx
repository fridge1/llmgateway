import { useState } from "react";
import { ShieldBan, Loader, Trash2, Plus } from "lucide-react";
import { toast } from "sonner";
import {
  useAdminBlockedIPs,
  useAdminBlockIP,
  useAdminUnblockIP,
} from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const fmtDate = (s: string | null) =>
  s ? new Date(s).toLocaleString("zh-CN", { hour12: false }) : "永久";

function BlockForm() {
  const block = useAdminBlockIP();
  const [ip, setIp] = useState("");
  const [reason, setReason] = useState("");
  const [days, setDays] = useState("");

  const submit = () => {
    const ipVal = ip.trim();
    const reasonVal = reason.trim();
    if (!ipVal || !reasonVal) {
      toast.error("IP 与封禁原因必填");
      return;
    }
    const d = days.trim() === "" ? undefined : Number(days);
    if (d !== undefined && (!Number.isFinite(d) || d <= 0)) {
      toast.error("封禁天数须为正数或留空（永久）");
      return;
    }
    block.mutate(
      { ip_address: ipVal, reason: reasonVal, expires_in_days: d },
      {
        onSuccess: () => {
          setIp("");
          setReason("");
          setDays("");
          toast.success("IP 已封禁");
        },
        onError: () => toast.error("封禁失败"),
      },
    );
  };

  return (
    <div className="bg-card border border-border rounded-xl p-5">
      <div className="flex items-center gap-3 mb-3">
        <Plus size={16} className="text-primary" />
        <h2 className="text-sm font-semibold">新增封禁</h2>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <input
          className="w-48 px-3 py-1.5 text-sm rounded-lg border border-border bg-background font-mono"
          placeholder="IP 地址（如 1.2.3.4）"
          value={ip}
          onChange={(e) => setIp(e.target.value)}
        />
        <input
          className="w-64 px-3 py-1.5 text-sm rounded-lg border border-border bg-background"
          placeholder="封禁原因（必填）"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        <input
          type="number"
          min={1}
          className="w-28 px-3 py-1.5 text-sm rounded-lg border border-border bg-background"
          placeholder="天数（留空=永久）"
          value={days}
          onChange={(e) => setDays(e.target.value)}
        />
        <button
          onClick={submit}
          disabled={block.isPending || !ip.trim() || !reason.trim()}
          className="flex items-center gap-1 px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded-lg hover:opacity-90 disabled:opacity-50"
        >
          <Plus size={14} />
          添加
        </button>
      </div>
      <p className="text-xs text-muted-foreground mt-3">
        新增后即时生效，IPBlocker 中间件对该 IP 的后续请求返回 403。永久封禁留空天数即可。
      </p>
    </div>
  );
}

function BlockedList() {
  const [page, setPage] = useState(1);
  const size = 20;
  const { data, isLoading } = useAdminBlockedIPs(page, size);
  const unblock = useAdminUnblockIP();

  const items = data?.items ?? [];
  const offset = data?.offset ?? 0;
  const total = items.length + offset + (items.length === size ? 1 : 0);
  const hasMore = items.length === size;

  return (
    <div className="bg-card border border-border rounded-xl overflow-hidden">
      <div className="px-4 py-3 border-b border-border/60 flex items-center justify-between">
        <h2 className="text-sm font-semibold">已封禁 IP</h2>
        <span className="text-xs text-muted-foreground">第 {page} 页</span>
      </div>
      {isLoading ? (
        <div className="flex items-center justify-center py-10 text-muted-foreground">
          <Loader size={18} className="animate-spin" />
        </div>
      ) : (
        <>
          <Table className="w-full">
            <TableHeader>
              <TableRow className="text-left text-xs text-muted-foreground border-b border-border/60">
                <TableHead className="px-4 py-2 font-medium">IP 地址</TableHead>
                <TableHead className="px-4 py-2 font-medium">原因</TableHead>
                <TableHead className="px-4 py-2 font-medium">封禁时间</TableHead>
                <TableHead className="px-4 py-2 font-medium">过期时间</TableHead>
                <TableHead className="px-4 py-2 font-medium">操作人</TableHead>
                <TableHead className="px-4 py-2 font-medium w-16">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((it) => (
                <TableRow key={it.ip_address} className="border-b border-border/60 last:border-0">
                  <TableCell className="px-4 py-3 text-xs font-mono">{it.ip_address}</TableCell>
                  <TableCell className="px-4 py-3 text-xs">{it.reason}</TableCell>
                  <TableCell className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                    {new Date(it.blocked_at).toLocaleString("zh-CN", { hour12: false })}
                  </TableCell>
                  <TableCell className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                    {fmtDate(it.expires_at)}
                  </TableCell>
                  <TableCell className="px-4 py-3 text-xs text-muted-foreground font-mono">
                    {it.blocked_by ?? "-"}
                  </TableCell>
                  <TableCell className="px-4 py-3">
                    <button
                      onClick={() =>
                        unblock.mutate(it.ip_address, {
                          onSuccess: () => toast.success("已解禁"),
                          onError: () => toast.error("解禁失败"),
                        })
                      }
                      disabled={unblock.isPending}
                      className="text-muted-foreground hover:text-destructive disabled:opacity-50"
                      title="解禁"
                    >
                      <Trash2 size={14} />
                    </button>
                  </TableCell>
                </TableRow>
              ))}
              {items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="px-4 py-8 text-center text-sm text-muted-foreground">
                    暂无封禁记录
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
          {(page > 1 || hasMore) && (
            <div className="flex items-center justify-end gap-2 px-4 py-3 border-t border-border/60">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="px-3 py-1 text-xs border border-border rounded-lg hover:bg-muted disabled:opacity-50"
              >
                上一页
              </button>
              <span className="text-xs text-muted-foreground">{page}</span>
              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={!hasMore}
                className="px-3 py-1 text-xs border border-border rounded-lg hover:bg-muted disabled:opacity-50"
              >
                下一页
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

const AdminBlockedIPs = () => (
  <div data-cmp="AdminBlockedIPs" className="space-y-6">
    <div className="flex items-center gap-2">
      <ShieldBan size={18} className="text-primary" />
      <h1 className="text-lg font-bold">IP 封禁</h1>
    </div>
    <BlockForm />
    <BlockedList />
  </div>
);

export default AdminBlockedIPs;
