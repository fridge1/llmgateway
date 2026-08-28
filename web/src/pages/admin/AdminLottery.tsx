import { useState } from "react";
import { Gift, Loader, Plus, Pencil, Trash2, X, Check, StopCircle, Trophy, Inbox } from "lucide-react";
import {
  useAdminLotteryEvents,
  useAdminCreateLotteryEvent,
  useAdminUpdateLotteryEvent,
  useAdminLotteryPrizes,
  useAdminCreateLotteryPrize,
  useAdminUpdateLotteryPrize,
  useAdminDeleteLotteryPrize,
  useAdminLotteryRecords,
  useAdminDrawEventLottery,
} from "@/hooks/use-api";
import type { LotteryEvent, LotteryPrize } from "@/types/api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const fmtDate = (s: string) => new Date(s).toLocaleString("zh-CN", { hour12: false });
const fmtCny = (n: number) => `¥${n.toFixed(2)}`;

// 将 UTC 时间字符串转换为本地 datetime-local 格式
const toLocalDatetimeString = (utcString: string | null) => {
  if (!utcString) return "";
  const date = new Date(utcString);
  // 转换为本地时间并格式化为 YYYY-MM-DDTHH:mm
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hours}:${minutes}`;
};

function EventCard({ event, onSelect }: { event: LotteryEvent; onSelect: () => void }) {
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState({
    name: event.name,
    description: event.description,
    status: event.status,
    min_recharge_cny: String(event.min_recharge_cny),
    min_order_count_to_draw: String(event.min_order_count_to_draw ?? 0),
    start_time: toLocalDatetimeString(event.start_time),
    end_time: toLocalDatetimeString(event.end_time),
  });
  const update = useAdminUpdateLotteryEvent();
  const drawLottery = useAdminDrawEventLottery();

  const save = () => {
    update.mutate(
      {
        id: event.id,
        name: form.name,
        description: form.description,
        status: form.status,
        min_recharge_cny: Number(form.min_recharge_cny),
        min_order_count_to_draw: Math.max(0, Number(form.min_order_count_to_draw) || 0),
        start_time: form.start_time ? new Date(form.start_time).toISOString() : null,
        end_time: form.end_time ? new Date(form.end_time).toISOString() : null,
      },
      { onSuccess: () => setEditing(false) },
    );
  };

  const endEvent = () => {
    if (!confirm("确认结束此活动？结束后用户将无法继续抽奖。")) return;
    update.mutate({
      id: event.id,
      name: event.name,
      description: event.description,
      status: "ended",
      min_recharge_cny: event.min_recharge_cny,
      min_order_count_to_draw: event.min_order_count_to_draw,
      start_time: event.start_time,
      end_time: event.end_time,
    });
  };

  const handleDraw = () => {
    if (!confirm("确认开奖？系统将从所有参与者中随机抽取中奖者。")) return;
    drawLottery.mutate(event.id, {
      onSuccess: (result) => {
        alert(`开奖成功！共 ${result.count} 位中奖者。`);
      },
      onError: (error: any) => {
        alert(`开奖失败：${error?.message || "未知错误"}`);
      },
    });
  };

  if (editing) {
    return (
      <div className="bg-card border border-border rounded-xl p-5 space-y-4">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label className="text-xs text-muted-foreground">活动名称</label>
            <input className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} />
          </div>
          <div>
            <label className="text-xs text-muted-foreground">最低充值金额（元）</label>
            <input type="number" min={0} step={0.01} className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.min_recharge_cny} onChange={e => setForm({ ...form, min_recharge_cny: e.target.value })} />
          </div>
          <div>
            <label className="text-xs text-muted-foreground">开奖所需最低笔数（0=不限）</label>
            <input type="number" min={0} step={1} className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.min_order_count_to_draw} onChange={e => setForm({ ...form, min_order_count_to_draw: e.target.value })} />
          </div>
          <div>
            <label className="text-xs text-muted-foreground">开始时间（留空=立即开始）</label>
            <input type="datetime-local" className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.start_time} onChange={e => setForm({ ...form, start_time: e.target.value })} />
          </div>
          <div>
            <label className="text-xs text-muted-foreground">结束时间（留空=不限）</label>
            <input type="datetime-local" className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.end_time} onChange={e => setForm({ ...form, end_time: e.target.value })} />
          </div>
          <div>
            <label className="text-xs text-muted-foreground">状态</label>
            <select className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.status} onChange={e => setForm({ ...form, status: e.target.value as "active" | "paused" | "ended" })}>
              <option value="active">启用</option>
              <option value="paused">暂停</option>
              <option value="ended">已结束</option>
            </select>
          </div>
          <div>
            <label className="text-xs text-muted-foreground">活动描述</label>
            <input className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} />
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
    <div onClick={onSelect} className="bg-card border border-border rounded-xl p-5 hover:border-primary/50 cursor-pointer transition">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold">{event.name}</h3>
            <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${event.status === "active" ? "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400" : event.status === "paused" ? "bg-yellow-50 text-yellow-600 dark:bg-yellow-500/10 dark:text-yellow-400" : "bg-muted text-muted-foreground"}`}>
              {event.status === "active" ? "进行中" : event.status === "paused" ? "已暂停" : "已结束"}
            </span>
          </div>
          {event.description && <p className="text-xs text-muted-foreground mt-1">{event.description}</p>}
          <div className="mt-3 flex gap-6 text-xs text-muted-foreground">
            <span>最低充值：{fmtCny(event.min_recharge_cny)}</span>
            <span>参与人数：{event.participant_count}</span>
            {event.min_order_count_to_draw > 0 && (
              <span className={event.participant_count >= event.min_order_count_to_draw ? "text-emerald-600 dark:text-emerald-400" : "text-amber-600 dark:text-amber-400"}>
                开奖门槛：{event.participant_count}/{event.min_order_count_to_draw} 笔
              </span>
            )}
            {event.start_time && <span>开始：{fmtDate(event.start_time)}</span>}
            {event.end_time && <span>结束：{fmtDate(event.end_time)}</span>}
            <span>创建于 {fmtDate(event.created_at)}</span>
          </div>
        </div>
        <div className="ml-4 flex items-center gap-1">
          {event.status !== "ended" && !event.drawn_at && (
            <>
              <button
                onClick={(e) => { e.stopPropagation(); handleDraw(); }}
                disabled={drawLottery.isPending}
                className="p-2 rounded-lg hover:bg-emerald-50 dark:hover:bg-emerald-500/10 text-muted-foreground hover:text-emerald-600 disabled:opacity-50"
                title="开奖"
              >
                <Trophy size={15} />
              </button>
              <button
                onClick={(e) => { e.stopPropagation(); endEvent(); }}
                disabled={update.isPending}
                className="p-2 rounded-lg hover:bg-red-50 dark:hover:bg-red-500/10 text-muted-foreground hover:text-destructive disabled:opacity-50"
                title="结束活动"
              >
                <StopCircle size={15} />
              </button>
            </>
          )}
          {event.drawn_at && (
            <span className="text-xs text-muted-foreground px-2 py-1 rounded-full bg-muted/50" title={`开奖于 ${fmtDate(event.drawn_at)}`}>
              已开奖
            </span>
          )}
          <button onClick={(e) => { e.stopPropagation(); setEditing(true); }} className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground">
            <Pencil size={15} />
          </button>
        </div>
      </div>
    </div>
  );
}

function PrizeManager({ eventId }: { eventId: number }) {
  const { data, isLoading } = useAdminLotteryPrizes(eventId);
  const createPrize = useAdminCreateLotteryPrize();
  const updatePrize = useAdminUpdateLotteryPrize();
  const deletePrize = useAdminDeleteLotteryPrize();
  const [editingId, setEditingId] = useState<number | null>(null);
  const [adding, setAdding] = useState(false);
  const [form, setForm] = useState({
    name: "",
    description: "",
    weight: "100",
    total_stock: "0",
    prize_type: "none",
    prize_value: "0",
    sort_order: "0",
  });

  const prizes = data?.prizes ?? [];

  const resetForm = () => {
    setForm({ name: "", description: "", weight: "100", total_stock: "0", prize_type: "none", prize_value: "0", sort_order: "0" });
    setAdding(false);
    setEditingId(null);
  };

  const handleCreate = () => {
    createPrize.mutate(
      { eventId, name: form.name, description: form.description, weight: Number(form.weight), total_stock: Number(form.total_stock), prize_type: form.prize_type, prize_value: Number(form.prize_value), sort_order: Number(form.sort_order) },
      { onSuccess: resetForm },
    );
  };

  const handleUpdate = (prizeId: number) => {
    updatePrize.mutate(
      { id: prizeId, eventId, name: form.name, description: form.description, weight: Number(form.weight), total_stock: Number(form.total_stock), prize_type: form.prize_type, prize_value: Number(form.prize_value), sort_order: Number(form.sort_order) },
      { onSuccess: resetForm },
    );
  };

  const handleDelete = (prizeId: number) => {
    if (confirm("确定删除此奖品？")) {
      deletePrize.mutate({ id: prizeId, eventId });
    }
  };

  const startEdit = (prize: LotteryPrize) => {
    setForm({
      name: prize.name,
      description: prize.description,
      weight: String(prize.weight),
      total_stock: String(prize.total_stock),
      prize_type: prize.prize_type,
      prize_value: String(prize.prize_value),
      sort_order: String(prize.sort_order),
    });
    setEditingId(prize.id);
  };

  if (isLoading) return <div className="flex justify-center py-8"><Loader size={18} className="animate-spin text-muted-foreground" /></div>;

  const prizeTypeLabel = (t: string) => {
    if (t === "none") return "谢谢参与";
    if (t === "balance") return "余额奖励";
    if (t === "match_recharge") return "等额充值";
    if (t === "physical") return "实物";
    return t;
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">奖品配置</h3>
        <button onClick={() => setAdding(true)} className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-primary text-primary-foreground rounded-lg hover:opacity-90">
          <Plus size={14} />添加奖品
        </button>
      </div>

      {adding || editingId !== null ? (
        <div className="bg-muted/30 border border-border rounded-lg p-4 space-y-3">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div>
              <label className="text-xs text-muted-foreground">奖品名称</label>
              <input className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">类型</label>
              <select className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.prize_type} onChange={e => setForm({ ...form, prize_type: e.target.value })}>
                <option value="none">谢谢参与</option>
                <option value="balance">余额奖励（固定金额）</option>
                <option value="match_recharge">等额充值（与充值额相同）</option>
                <option value="physical">实物奖品</option>
              </select>
            </div>
            <div>
              <label className="text-xs text-muted-foreground">奖励金额（元）</label>
              <input type="number" min={0} step={0.01} disabled={form.prize_type !== "balance"} className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background disabled:opacity-50" placeholder={form.prize_type === "match_recharge" ? "由充值额决定" : "0"} value={form.prize_value} onChange={e => setForm({ ...form, prize_value: e.target.value })} />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">权重</label>
              <input type="number" min={1} className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.weight} onChange={e => setForm({ ...form, weight: e.target.value })} />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">总库存（0=不限量）</label>
              <input type="number" min={0} className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.total_stock} onChange={e => setForm({ ...form, total_stock: e.target.value })} />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">排序</label>
              <input type="number" className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.sort_order} onChange={e => setForm({ ...form, sort_order: e.target.value })} />
            </div>
          </div>
          <div>
            <label className="text-xs text-muted-foreground">描述</label>
            <input className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} />
          </div>
          <div className="flex gap-2">
            <button onClick={editingId !== null ? () => handleUpdate(editingId) : handleCreate} disabled={createPrize.isPending || updatePrize.isPending} className="flex items-center gap-1.5 px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:opacity-90 disabled:opacity-50">
              <Check size={14} />{editingId !== null ? "保存" : "创建"}
            </button>
            <button onClick={resetForm} className="flex items-center gap-1.5 px-4 py-2 text-sm border border-border rounded-lg hover:bg-muted">
              <X size={14} />取消
            </button>
          </div>
        </div>
      ) : null}

      {prizes.length === 0 ? (
        <div className="border border-dashed border-border rounded-lg py-10 text-center">
          <Inbox className="h-8 w-8 text-muted-foreground/50 mx-auto mb-2" />
          <p className="text-sm text-muted-foreground">暂无奖品，点击「添加奖品」开始配置</p>
        </div>
      ) : (
        <div className="border border-border rounded-lg overflow-hidden">
          <Table className="w-full text-sm">
            <TableHeader className="bg-muted/40">
              <TableRow>
                <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">名称</TableHead>
                <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">类型</TableHead>
                <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">奖励金额</TableHead>
                <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">权重</TableHead>
                <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">库存</TableHead>
                <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {prizes.map(p => (
                <TableRow key={p.id} className="border-t border-border hover:bg-muted/20">
                  <TableCell className="px-4 py-3">{p.name}</TableCell>
                  <TableCell className="px-4 py-3">{prizeTypeLabel(p.prize_type)}</TableCell>
                  <TableCell className="px-4 py-3">{p.prize_type === "balance" ? fmtCny(p.prize_value) : p.prize_type === "match_recharge" ? "等额充值" : "—"}</TableCell>
                  <TableCell className="px-4 py-3">{p.weight}</TableCell>
                  <TableCell className="px-4 py-3">{p.total_stock === 0 ? "不限量" : `${p.remaining_stock} / ${p.total_stock}`}</TableCell>
                  <TableCell className="px-4 py-3">
                    <div className="flex gap-2">
                      <button onClick={() => startEdit(p)} className="text-xs text-primary hover:underline">编辑</button>
                      <button onClick={() => handleDelete(p.id)} className="text-xs text-destructive hover:underline">删除</button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

function RecordsTable({ eventId }: { eventId: number }) {
  const [page, setPage] = useState(1);
  const size = 20;
  const { data, isLoading } = useAdminLotteryRecords(eventId, page, size);
  const records = data?.records ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  if (isLoading) return <div className="flex justify-center py-8"><Loader size={18} className="animate-spin text-muted-foreground" /></div>;

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold">抽奖记录</h3>
      {records.length === 0 ? (
        <div className="border border-dashed border-border rounded-lg py-10 text-center">
          <Inbox className="h-8 w-8 text-muted-foreground/50 mx-auto mb-2" />
          <p className="text-sm text-muted-foreground">暂无抽奖记录，充值达标用户将自动参与</p>
        </div>
      ) : (
        <>
          <div className="border border-border rounded-lg overflow-hidden">
            <Table className="w-full text-sm">
              <TableHeader className="bg-muted/40">
                <TableRow>
                  <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">用户</TableHead>
                  <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">充值金额</TableHead>
                  <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">奖品</TableHead>
                  <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">奖励</TableHead>
                  <TableHead className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {records.map(r => (
                  <TableRow key={r.id} className="border-t border-border hover:bg-muted/20">
                    <TableCell className="px-4 py-3">
                      {r.nickname || r.phone || <span className="font-mono text-xs text-muted-foreground">{r.user_id.slice(0, 12)}...</span>}
                    </TableCell>
                    <TableCell className="px-4 py-3">{fmtCny(r.recharge_amount)}</TableCell>
                    <TableCell className="px-4 py-3">{r.prize_name || "—"}</TableCell>
                    <TableCell className="px-4 py-3">{(r.prize_type === "balance" || r.prize_type === "match_recharge") ? <span className="text-emerald-600">{fmtCny(r.prize_value)}</span> : "—"}</TableCell>
                    <TableCell className="px-4 py-3 text-xs text-muted-foreground">{fmtDate(r.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2">
              <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} className="px-3 py-1 text-xs border border-border rounded hover:bg-muted disabled:opacity-50">上一页</button>
              <span className="text-xs text-muted-foreground">{page} / {totalPages}</span>
              <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages} className="px-3 py-1 text-xs border border-border rounded hover:bg-muted disabled:opacity-50">下一页</button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

const AdminLottery = () => {
  const { data, isLoading } = useAdminLotteryEvents(1, 100);
  const createEvent = useAdminCreateLotteryEvent();
  const [selectedEventId, setSelectedEventId] = useState<number | null>(null);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({
    name: "充值抽奖活动",
    description: "",
    status: "active",
    min_recharge_cny: "1",
    min_order_count_to_draw: "0",
    start_time: "",
    end_time: "",
  });

  const events = data?.events ?? [];
  const selectedEvent = events.find(e => e.id === selectedEventId);

  const handleCreate = () => {
    createEvent.mutate(
      {
        name: form.name,
        description: form.description,
        status: form.status,
        min_recharge_cny: Number(form.min_recharge_cny),
        min_order_count_to_draw: Math.max(0, Number(form.min_order_count_to_draw) || 0),
        start_time: form.start_time ? new Date(form.start_time).toISOString() : null,
        end_time: form.end_time ? new Date(form.end_time).toISOString() : null,
      },
      {
        onSuccess: (evt) => {
          setCreating(false);
          setSelectedEventId(evt.id);
          setForm({ name: "充值抽奖活动", description: "", status: "active", min_recharge_cny: "1", min_order_count_to_draw: "0", start_time: "", end_time: "" });
        },
      },
    );
  };

  if (isLoading) return <div className="flex justify-center items-center h-96"><Loader size={24} className="animate-spin text-muted-foreground" /></div>;

  return (
    <div data-cmp="AdminLottery" className="p-6 max-w-7xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Gift size={20} className="text-muted-foreground" />
          <h1 className="text-xl font-semibold">抽奖活动管理</h1>
        </div>
        <button onClick={() => setCreating(true)} className="flex items-center gap-1.5 px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:opacity-90">
          <Plus size={16} />创建活动
        </button>
      </div>

      {creating && (
        <div className="bg-muted/30 border border-border rounded-xl p-5 space-y-4">
          <h3 className="text-sm font-semibold">创建抽奖活动</h3>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label className="text-xs text-muted-foreground">活动名称</label>
              <input className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">最低充值金额（元）</label>
              <input type="number" min={0} step={0.01} className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.min_recharge_cny} onChange={e => setForm({ ...form, min_recharge_cny: e.target.value })} />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">开奖所需最低笔数（0=不限）</label>
              <input type="number" min={0} step={1} className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.min_order_count_to_draw} onChange={e => setForm({ ...form, min_order_count_to_draw: e.target.value })} />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">开始时间（留空=立即开始）</label>
              <input type="datetime-local" className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.start_time} onChange={e => setForm({ ...form, start_time: e.target.value })} />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">结束时间（留空=不限）</label>
              <input type="datetime-local" className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.end_time} onChange={e => setForm({ ...form, end_time: e.target.value })} />
            </div>
            <div className="col-span-2">
              <label className="text-xs text-muted-foreground">活动描述</label>
              <input className="w-full mt-1 px-3 py-2 text-sm rounded-lg border border-border bg-background" value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} />
            </div>
          </div>
          <div className="flex gap-2">
            <button onClick={handleCreate} disabled={createEvent.isPending} className="flex items-center gap-1.5 px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:opacity-90 disabled:opacity-50">
              <Check size={14} />创建
            </button>
            <button onClick={() => setCreating(false)} className="flex items-center gap-1.5 px-4 py-2 text-sm border border-border rounded-lg hover:bg-muted">
              <X size={14} />取消
            </button>
          </div>
        </div>
      )}

      <div className="space-y-3">
        {events.map(e => (
          <EventCard key={e.id} event={e} onSelect={() => setSelectedEventId(e.id)} />
        ))}
        {events.length === 0 && !creating && (
          <p className="text-sm text-muted-foreground text-center py-12">暂无抽奖活动，点击"创建活动"开始</p>
        )}
      </div>

      {selectedEvent && (
        <div className="space-y-6 border-t border-border pt-6">
          <PrizeManager eventId={selectedEvent.id} />
          <RecordsTable eventId={selectedEvent.id} />
        </div>
      )}
    </div>
  );
};

export default AdminLottery;
