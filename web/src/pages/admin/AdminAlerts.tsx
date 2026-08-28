import { useState } from "react";
import { BellRing, Loader, Pencil, X, Check } from "lucide-react";
import { toast } from "sonner";
import {
  useAdminAlertRules,
  useAdminUpdateAlertRule,
  useAdminAlertEvents,
  type AlertRule,
} from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const fmtDate = (s: string) => new Date(s).toLocaleString("zh-CN", { hour12: false });

function RuleRow({ rule }: { rule: AlertRule }) {
  const [editing, setEditing] = useState(false);
  const [threshold, setThreshold] = useState(String(rule.threshold));
  const [cooldown, setCooldown] = useState(String(rule.cooldown_seconds));
  const update = useAdminUpdateAlertRule();

  const save = (enabled: boolean) => {
    const t = Number(threshold);
    const c = Number(cooldown);
    if (!Number.isFinite(t) || t < 1 || !Number.isFinite(c) || c < 0) {
      toast.error("阈值须 ≥1，冷却秒数须 ≥0");
      return;
    }
    update.mutate(
      { id: rule.id, threshold: t, cooldown_seconds: c, enabled },
      {
        onSuccess: () => {
          setEditing(false);
          toast.success("规则已更新");
        },
        onError: () => toast.error("更新失败"),
      },
    );
  };

  return (
    <TableRow className="border-b border-border/60 last:border-0">
      <TableCell className="px-4 py-3 text-sm font-medium">{rule.display_name}</TableCell>
      <TableCell className="px-4 py-3 text-xs text-muted-foreground font-mono">{rule.metric}</TableCell>
      <TableCell className="px-4 py-3 text-sm">
        {editing ? (
          <input
            type="number"
            min={1}
            className="w-24 px-2 py-1 text-sm rounded-lg border border-border bg-background"
            value={threshold}
            onChange={(e) => setThreshold(e.target.value)}
          />
        ) : (
          rule.threshold
        )}
      </TableCell>
      <TableCell className="px-4 py-3 text-sm">
        {editing ? (
          <input
            type="number"
            min={0}
            className="w-24 px-2 py-1 text-sm rounded-lg border border-border bg-background"
            value={cooldown}
            onChange={(e) => setCooldown(e.target.value)}
          />
        ) : (
          `${rule.cooldown_seconds}s`
        )}
      </TableCell>
      <TableCell className="px-4 py-3">
        <button
          onClick={() => save(!rule.enabled)}
          disabled={update.isPending}
          className={`text-xs px-2 py-0.5 rounded-full font-medium ${
            rule.enabled
              ? "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400"
              : "bg-muted text-muted-foreground"
          }`}
        >
          {rule.enabled ? "已启用" : "已停用"}
        </button>
      </TableCell>
      <TableCell className="px-4 py-3">
        {editing ? (
          <div className="flex gap-1">
            <button
              onClick={() => save(rule.enabled)}
              disabled={update.isPending}
              className="p-1.5 rounded-lg hover:bg-muted text-emerald-600"
              title="保存"
            >
              <Check size={14} />
            </button>
            <button
              onClick={() => {
                setEditing(false);
                setThreshold(String(rule.threshold));
                setCooldown(String(rule.cooldown_seconds));
              }}
              className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground"
              title="取消"
            >
              <X size={14} />
            </button>
          </div>
        ) : (
          <button
            onClick={() => setEditing(true)}
            className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground"
            title="编辑"
          >
            <Pencil size={14} />
          </button>
        )}
      </TableCell>
    </TableRow>
  );
}

const AdminAlerts = () => {
  const [page, setPage] = useState(1);
  const size = 20;
  const { data: rulesData, isLoading: rulesLoading } = useAdminAlertRules();
  const { data: eventsData, isLoading: eventsLoading } = useAdminAlertEvents(page, size);

  const rules = rulesData?.rules ?? [];
  const events = eventsData?.events ?? [];
  const total = eventsData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  return (
    <div data-cmp="AdminAlerts" className="space-y-6">
      <div className="flex items-center gap-2">
        <BellRing size={18} className="text-primary" />
        <h1 className="text-lg font-bold">运维告警</h1>
      </div>

      {/* Rules */}
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-4 py-3 border-b border-border/60">
          <h2 className="text-sm font-semibold">告警规则</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            达到阈值时向管理员发送站内通知与短信；冷却窗口内同一告警不重复发送。
          </p>
        </div>
        {rulesLoading ? (
          <div className="flex items-center justify-center py-10 text-muted-foreground">
            <Loader size={18} className="animate-spin" />
          </div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="text-left text-xs text-muted-foreground border-b border-border/60">
                <TableHead className="px-4 py-2 font-medium">名称</TableHead>
                <TableHead className="px-4 py-2 font-medium">指标</TableHead>
                <TableHead className="px-4 py-2 font-medium">阈值（每周期）</TableHead>
                <TableHead className="px-4 py-2 font-medium">冷却</TableHead>
                <TableHead className="px-4 py-2 font-medium">状态</TableHead>
                <TableHead className="px-4 py-2 font-medium">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rules.map((r) => (
                <RuleRow key={r.id} rule={r} />
              ))}
              {rules.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="px-4 py-8 text-center text-sm text-muted-foreground">
                    暂无告警规则
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </div>

      {/* Events */}
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-4 py-3 border-b border-border/60">
          <h2 className="text-sm font-semibold">最近告警事件</h2>
        </div>
        {eventsLoading ? (
          <div className="flex items-center justify-center py-10 text-muted-foreground">
            <Loader size={18} className="animate-spin" />
          </div>
        ) : (
          <>
            <Table className="w-full">
              <TableHeader>
                <TableRow className="text-left text-xs text-muted-foreground border-b border-border/60">
                  <TableHead className="px-4 py-2 font-medium">时间</TableHead>
                  <TableHead className="px-4 py-2 font-medium">指标</TableHead>
                  <TableHead className="px-4 py-2 font-medium">详情</TableHead>
                  <TableHead className="px-4 py-2 font-medium">数值/阈值</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {events.map((e) => (
                  <TableRow key={e.id} className="border-b border-border/60 last:border-0">
                    <TableCell className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                      {fmtDate(e.created_at)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-xs font-mono">{e.metric}</TableCell>
                    <TableCell className="px-4 py-3 text-sm">{e.message}</TableCell>
                    <TableCell className="px-4 py-3 text-sm">
                      {e.value} / {e.threshold}
                    </TableCell>
                  </TableRow>
                ))}
                {events.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} className="px-4 py-8 text-center text-sm text-muted-foreground">
                      暂无告警事件
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
            {totalPages > 1 && (
              <div className="flex items-center justify-end gap-2 px-4 py-3 border-t border-border/60">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="px-3 py-1 text-xs border border-border rounded-lg hover:bg-muted disabled:opacity-50"
                >
                  上一页
                </button>
                <span className="text-xs text-muted-foreground">
                  {page} / {totalPages}
                </span>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="px-3 py-1 text-xs border border-border rounded-lg hover:bg-muted disabled:opacity-50"
                >
                  下一页
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default AdminAlerts;
