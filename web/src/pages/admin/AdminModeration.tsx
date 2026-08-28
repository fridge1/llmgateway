import { useState } from "react";
import { ShieldAlert, Loader, Trash2, Plus } from "lucide-react";
import { toast } from "sonner";
import {
  useAdminModerationSettings,
  useAdminUpdateModerationSettings,
  useAdminModerationKeywords,
  useAdminCreateModerationKeyword,
  useAdminDeleteModerationKeyword,
  useAdminModerationHits,
} from "@/hooks/use-api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const fmtDate = (s: string) => new Date(s).toLocaleString("zh-CN", { hour12: false });

function SettingsCard() {
  const { data, isLoading } = useAdminModerationSettings();
  const update = useAdminUpdateModerationSettings();

  if (isLoading || !data) {
    return (
      <div className="bg-card border border-border rounded-xl p-5 flex items-center justify-center py-8 text-muted-foreground">
        <Loader size={18} className="animate-spin" />
      </div>
    );
  }

  const toggle = (patch: Partial<{ enabled: boolean; enforce_all: boolean }>) => {
    update.mutate(
      { enabled: data.enabled, enforce_all: data.enforce_all, ...patch },
      {
        onSuccess: () => toast.success("设置已更新"),
        onError: () => toast.error("更新失败"),
      },
    );
  };

  return (
    <div className="bg-card border border-border rounded-xl p-5 flex items-center gap-8">
      <div className="flex items-center gap-3">
        <span className="text-sm font-medium">内容审核</span>
        <button
          onClick={() => toggle({ enabled: !data.enabled })}
          disabled={update.isPending}
          className={`text-xs px-3 py-1 rounded-full font-medium ${
            data.enabled
              ? "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400"
              : "bg-muted text-muted-foreground"
          }`}
        >
          {data.enabled ? "已开启" : "已关闭"}
        </button>
      </div>
      <div className="flex items-center gap-3">
        <span className="text-sm font-medium">作用范围</span>
        <button
          onClick={() => toggle({ enforce_all: !data.enforce_all })}
          disabled={update.isPending || !data.enabled}
          className="text-xs px-3 py-1 rounded-full font-medium bg-muted hover:bg-muted/70 disabled:opacity-50"
        >
          {data.enforce_all ? "全部请求" : "仅按模型/租户开关"}
        </button>
      </div>
      <p className="text-xs text-muted-foreground flex-1">
        命中关键词的请求将被拒绝（HTTP 403）并记录。规则改动约 30 秒内对全部节点生效。
      </p>
    </div>
  );
}

function KeywordsCard() {
  const { data, isLoading } = useAdminModerationKeywords();
  const create = useAdminCreateModerationKeyword();
  const del = useAdminDeleteModerationKeyword();
  const [keyword, setKeyword] = useState("");
  const [category, setCategory] = useState("general");

  const add = () => {
    const kw = keyword.trim();
    if (!kw) return;
    create.mutate(
      { keyword: kw, category },
      {
        onSuccess: () => {
          setKeyword("");
          toast.success("关键词已添加");
        },
        onError: () => toast.error("添加失败"),
      },
    );
  };

  const keywords = data?.keywords ?? [];

  return (
    <div className="bg-card border border-border rounded-xl overflow-hidden">
      <div className="px-4 py-3 border-b border-border/60 flex items-center justify-between">
        <h2 className="text-sm font-semibold">关键词库（{keywords.length}）</h2>
        <div className="flex items-center gap-2">
          <input
            className="w-48 px-3 py-1.5 text-sm rounded-lg border border-border bg-background"
            placeholder="新增关键词"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && add()}
          />
          <select
            className="px-2 py-1.5 text-sm rounded-lg border border-border bg-background"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          >
            <option value="general">通用</option>
            <option value="politics">时政</option>
            <option value="violence">暴恐</option>
            <option value="porn">色情</option>
            <option value="fraud">诈骗</option>
          </select>
          <button
            onClick={add}
            disabled={create.isPending || !keyword.trim()}
            className="flex items-center gap-1 px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded-lg hover:opacity-90 disabled:opacity-50"
          >
            <Plus size={14} />
            添加
          </button>
        </div>
      </div>
      {isLoading ? (
        <div className="flex items-center justify-center py-10 text-muted-foreground">
          <Loader size={18} className="animate-spin" />
        </div>
      ) : (
        <div className="p-4 flex flex-wrap gap-2">
          {keywords.map((k) => (
            <span
              key={k.id}
              className="inline-flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full bg-muted"
            >
              {k.keyword}
              <span className="text-muted-foreground">· {k.category}</span>
              <button
                onClick={() =>
                  del.mutate(k.id, {
                    onSuccess: () => toast.success("已删除"),
                    onError: () => toast.error("删除失败"),
                  })
                }
                className="text-muted-foreground hover:text-destructive"
                title="删除"
              >
                <Trash2 size={12} />
              </button>
            </span>
          ))}
          {keywords.length === 0 && (
            <p className="text-sm text-muted-foreground py-4 w-full text-center">
              暂无关键词，添加后审核才会生效
            </p>
          )}
        </div>
      )}
    </div>
  );
}

function HitsCard() {
  const [page, setPage] = useState(1);
  const [userId, setUserId] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const size = 20;
  const { data, isLoading } = useAdminModerationHits(page, size, userId, from, to);

  const hits = data?.hits ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  return (
    <div className="bg-card border border-border rounded-xl overflow-hidden">
      <div className="px-4 py-3 border-b border-border/60 flex items-center gap-3">
        <h2 className="text-sm font-semibold flex-1">命中记录</h2>
        <input
          className="w-64 px-3 py-1.5 text-xs rounded-lg border border-border bg-background font-mono"
          placeholder="按用户 ID 筛选"
          value={userId}
          onChange={(e) => {
            setUserId(e.target.value);
            setPage(1);
          }}
        />
        <input
          type="date"
          className="px-2 py-1.5 text-xs rounded-lg border border-border bg-background"
          value={from}
          onChange={(e) => {
            setFrom(e.target.value);
            setPage(1);
          }}
        />
        <span className="text-xs text-muted-foreground">至</span>
        <input
          type="date"
          className="px-2 py-1.5 text-xs rounded-lg border border-border bg-background"
          value={to}
          onChange={(e) => {
            setTo(e.target.value);
            setPage(1);
          }}
        />
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
                <TableHead className="px-4 py-2 font-medium">时间</TableHead>
                <TableHead className="px-4 py-2 font-medium">用户/租户</TableHead>
                <TableHead className="px-4 py-2 font-medium">模型</TableHead>
                <TableHead className="px-4 py-2 font-medium">命中规则</TableHead>
                <TableHead className="px-4 py-2 font-medium">片段</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {hits.map((h) => (
                <TableRow key={h.id} className="border-b border-border/60 last:border-0">
                  <TableCell className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                    {fmtDate(h.created_at)}
                  </TableCell>
                  <TableCell className="px-4 py-3 text-xs font-mono">
                    {h.user_id ?? h.tenant_id ?? "-"}
                  </TableCell>
                  <TableCell className="px-4 py-3 text-xs">{h.model}</TableCell>
                  <TableCell className="px-4 py-3 text-xs font-medium text-destructive">{h.matched_rule}</TableCell>
                  <TableCell className="px-4 py-3 text-xs text-muted-foreground max-w-md truncate" title={h.snippet}>
                    {h.snippet}
                  </TableCell>
                </TableRow>
              ))}
              {hits.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="px-4 py-8 text-center text-sm text-muted-foreground">
                    暂无命中记录
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
  );
}

const AdminModeration = () => (
  <div data-cmp="AdminModeration" className="space-y-6">
    <div className="flex items-center gap-2">
      <ShieldAlert size={18} className="text-primary" />
      <h1 className="text-lg font-bold">内容安全</h1>
    </div>
    <SettingsCard />
    <KeywordsCard />
    <HitsCard />
  </div>
);

export default AdminModeration;
