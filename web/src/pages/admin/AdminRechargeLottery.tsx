import { useState } from "react";
import { Dices, Loader, Pencil, X, Check } from "lucide-react";
import {
  useAdminRechargeLottery,
  useAdminCreateRechargeLottery,
  useAdminUpdateRechargeLottery,
  useAdminRechargeLotteryRounds,
} from "@/hooks/use-api";
import type { RechargeLottery } from "@/types/api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const fmtDate = (s: string) => new Date(s).toLocaleString("zh-CN", { hour12: false });
const fmtCny = (n: number) => `¥${n.toFixed(2)}`;

function ConfigCard({ lottery }: { lottery: RechargeLottery }) {
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState<{ name: string; status: "active" | "paused"; trigger_every: string }>({ name: lottery.name, status: lottery.status, trigger_every: String(lottery.trigger_every) });
  const update = useAdminUpdateRechargeLottery();

  const save = () => {
    update.mutate(
      { id: lottery.id, name: form.name, status: form.status, trigger_every: Number(form.trigger_every) },
      { onSuccess: () => setEditing(false) },
    );
  };

  if (editing) {
    return (
      <div className="bg-card border border-border rounded-xl p-5 space-y-4">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <label className="text-xs text-muted-foreground">活动名称</label>
            <input className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} />
          </div>
          <div>
            <label className="text-xs text-muted-foreground">每满多少笔开奖</label>
            <input type="number" min={1} className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.trigger_every} onChange={e => setForm({ ...form, trigger_every: e.target.value })} />
          </div>
          <div>
            <label className="text-xs text-muted-foreground">状态</label>
            <select className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.status} onChange={e => setForm({ ...form, status: e.target.value as "active" | "paused" })}>
              <option value="active">启用</option>
              <option value="paused">暂停</option>
            </select>
          </div>
        </div>
        <div className="flex gap-2">
          <button onClick={save} disabled={update.isPending} className="flex items-center gap-1.5 px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:opacity-90 disabled:opacity-50">
            <Check size={14} />保存
          </button>
          <button onClick={() => setEditing(false)} className="flex items-center gap-1.5 px-4 py-2 text-sm border border-border rounded-lg hover:bg-muted">
            <X size={14} />取消
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-card border border-border rounded-xl p-5">
      <div className="flex items-start justify-between">
        <div className="grid flex-1 grid-cols-1 gap-6 sm:grid-cols-2 xl:grid-cols-4">
          <div>
            <p className="text-xs text-muted-foreground">活动名称</p>
            <p className="text-sm font-medium mt-0.5">{lottery.name}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">每满多少笔开奖</p>
            <p className="text-sm font-medium mt-0.5">{lottery.trigger_every} 笔</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">已完成轮数</p>
            <p className="text-sm font-medium mt-0.5">{lottery.total_rounds} 轮</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">状态</p>
            <span className={`inline-block mt-0.5 text-xs px-2 py-0.5 rounded-full font-medium ${lottery.status === "active" ? "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400" : "bg-muted text-muted-foreground"}`}>
              {lottery.status === "active" ? "启用" : "已暂停"}
            </span>
          </div>
        </div>
        <button onClick={() => setEditing(true)} className="ml-4 p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground">
          <Pencil size={15} />
        </button>
      </div>
    </div>
  );
}

function RoundsTable({ lotteryId }: { lotteryId: number }) {
  const [page, setPage] = useState(1);
  const size = 20;
  const { data, isLoading } = useAdminRechargeLotteryRounds(lotteryId, page, size);
  const rounds = data?.rounds ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  if (isLoading) return <div className="flex justify-center py-8"><Loader size={18} className="animate-spin text-muted-foreground" /></div>;

  return (
    <div>
      <h3 className="text-sm font-semibold mb-3">开奖记录</h3>
      <div className="border border-border rounded-xl overflow-hidden">
        <Table className="w-full text-sm">
          <TableHeader className="bg-muted/40">
            <TableRow>
              {["轮次", "中奖用户", "电话", "充值金额（等额赠送）", "本轮参与笔数", "开奖时间"].map(h => (
                <TableHead key={h} className="text-left px-4 py-2.5 text-xs font-medium text-muted-foreground">{h}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rounds.length === 0 ? (
              <TableRow><TableCell colSpan={6} className="text-center py-8 text-muted-foreground text-sm">暂无开奖记录</TableCell></TableRow>
            ) : rounds.map(r => (
              <TableRow key={r.id} className="border-t border-border hover:bg-muted/20">
                <TableCell className="px-4 py-3 font-medium">第 {r.round_no} 轮</TableCell>
                <TableCell className="px-4 py-3 text-muted-foreground">{r.winner_nickname || r.winner_user_id.slice(0, 8) + "…"}</TableCell>
                <TableCell className="px-4 py-3 text-muted-foreground tabular-nums">{r.winner_phone || "—"}</TableCell>
                <TableCell className="px-4 py-3 text-emerald-600 dark:text-emerald-400 font-medium">{fmtCny(r.winner_amount)}</TableCell>
                <TableCell className="px-4 py-3">{r.participant_count} 笔</TableCell>
                <TableCell className="px-4 py-3 text-muted-foreground">{fmtDate(r.created_at)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {totalPages > 1 && (
        <div className="flex justify-end gap-2 mt-3">
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} className="px-3 py-1.5 text-xs border border-border rounded-lg disabled:opacity-40 hover:bg-muted">上一页</button>
          <span className="px-3 py-1.5 text-xs text-muted-foreground">{page} / {totalPages}</span>
          <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages} className="px-3 py-1.5 text-xs border border-border rounded-lg disabled:opacity-40 hover:bg-muted">下一页</button>
        </div>
      )}
    </div>
  );
}

const AdminRechargeLottery = () => {
  const { data, isLoading } = useAdminRechargeLottery();
  const create = useAdminCreateRechargeLottery();
  const [form, setForm] = useState({ name: "充值幸运奖", trigger_every: "10" });

  const lottery = data?.lottery ?? null;

  return (
    <div data-cmp="AdminRechargeLottery" className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center gap-2">
        <Dices size={18} className="text-primary" />
        <h2 className="text-base font-semibold">充值抽奖</h2>
        <span className="text-xs text-muted-foreground">充值订单满 N 笔后随机抽取一笔，赠送等额金额</span>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-12"><Loader size={20} className="animate-spin text-muted-foreground" /></div>
      ) : lottery ? (
        <>
          <ConfigCard lottery={lottery} />
          <RoundsTable lotteryId={lottery.id} />
        </>
      ) : (
        <div className="bg-card border border-border rounded-xl p-6 max-w-sm space-y-4">
          <p className="text-sm font-medium">创建充值抽奖活动</p>
          <div>
            <label className="text-xs text-muted-foreground">活动名称</label>
            <input className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} />
          </div>
          <div>
            <label className="text-xs text-muted-foreground">每满多少笔开奖</label>
            <input type="number" min={1} className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.trigger_every} onChange={e => setForm({ ...form, trigger_every: e.target.value })} />
          </div>
          <button onClick={() => create.mutate({ name: form.name, trigger_every: Number(form.trigger_every) })} disabled={create.isPending} className="flex items-center gap-2 px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:opacity-90 disabled:opacity-50">
            {create.isPending ? <Loader size={14} className="animate-spin" /> : <Dices size={14} />}创建活动
          </button>
        </div>
      )}
    </div>
  );
};

export default AdminRechargeLottery;
