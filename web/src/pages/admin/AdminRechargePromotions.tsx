import { useState } from "react";
import { Gift, Plus, Loader, Trash2, X, Pencil } from "lucide-react";
import {
  useAdminRechargePromotions,
  useAdminCreateRechargePromotion,
  useAdminUpdateRechargePromotion,
  useAdminDeleteRechargePromotion,
} from "@/hooks/use-api";
import type { RechargePromotion, RechargePromotionInput } from "@/types/api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
interface FormState {
  name: string;
  starts_at: string;
  ends_at: string;
  bonus_percent: string;
  min_recharge_amount: string;
  is_active: boolean;
}

const emptyForm: FormState = {
  name: "",
  starts_at: "",
  ends_at: "",
  bonus_percent: "10",
  min_recharge_amount: "0",
  is_active: true,
};

// Convert ISO timestamp to a value usable by <input type="datetime-local"> in
// the user's local timezone. Returns "" for empty input.
const isoToLocalInput = (iso: string): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(
    d.getHours(),
  )}:${pad(d.getMinutes())}`;
};

const localInputToISO = (s: string): string => (s ? new Date(s).toISOString() : "");

const formatRange = (startsAt: string, endsAt: string): string => {
  const s = new Date(startsAt);
  const e = new Date(endsAt);
  const fmt = (d: Date) =>
    `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
      d.getDate(),
    ).padStart(2, "0")} ${String(d.getHours()).padStart(2, "0")}:${String(
      d.getMinutes(),
    ).padStart(2, "0")}`;
  return `${fmt(s)} ~ ${fmt(e)}`;
};

const promoStatus = (p: RechargePromotion): { label: string; className: string } => {
  if (!p.is_active) {
    return { label: "已停用", className: "bg-muted text-muted-foreground" };
  }
  const now = new Date();
  const starts = new Date(p.starts_at);
  const ends = new Date(p.ends_at);
  if (now < starts) {
    return {
      label: "未开始",
      className: "bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-400",
    };
  }
  if (now >= ends) {
    return {
      label: "已结束",
      className: "bg-muted text-muted-foreground",
    };
  }
  return {
    label: "进行中",
    className:
      "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400",
  };
};

const AdminRechargePromotions = () => {
  const { data, isLoading } = useAdminRechargePromotions();
  const createMut = useAdminCreateRechargePromotion();
  const updateMut = useAdminUpdateRechargePromotion();
  const deleteMut = useAdminDeleteRechargePromotion();

  const [showModal, setShowModal] = useState(false);
  const [editItem, setEditItem] = useState<RechargePromotion | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [formError, setFormError] = useState<string | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<RechargePromotion | null>(null);

  const list = data?.promotions ?? [];
  const activeCount = list.filter((p) => promoStatus(p).label === "进行中").length;

  const openCreate = () => {
    setEditItem(null);
    setForm(emptyForm);
    setFormError(null);
    setShowModal(true);
  };

  const openEdit = (p: RechargePromotion) => {
    setEditItem(p);
    setForm({
      name: p.name,
      starts_at: isoToLocalInput(p.starts_at),
      ends_at: isoToLocalInput(p.ends_at),
      bonus_percent: (p.bonus_ratio * 100).toString(),
      min_recharge_amount: p.min_recharge_amount.toString(),
      is_active: p.is_active,
    });
    setFormError(null);
    setShowModal(true);
  };

  const buildPayload = (): RechargePromotionInput | null => {
    if (!form.name.trim()) {
      setFormError("活动名称不能为空");
      return null;
    }
    if (!form.starts_at || !form.ends_at) {
      setFormError("请选择开始和结束时间");
      return null;
    }
    const startsAtISO = localInputToISO(form.starts_at);
    const endsAtISO = localInputToISO(form.ends_at);
    if (new Date(endsAtISO) <= new Date(startsAtISO)) {
      setFormError("结束时间必须晚于开始时间");
      return null;
    }
    const bonusPercent = Number(form.bonus_percent);
    if (!Number.isFinite(bonusPercent) || bonusPercent <= 0 || bonusPercent > 100) {
      setFormError("赠送比例必须在 (0, 100] 之间");
      return null;
    }
    const minAmount = Number(form.min_recharge_amount);
    if (!Number.isFinite(minAmount) || minAmount < 0) {
      setFormError("最低充值门槛不能为负");
      return null;
    }
    return {
      name: form.name.trim(),
      starts_at: startsAtISO,
      ends_at: endsAtISO,
      bonus_ratio: bonusPercent / 100,
      min_recharge_amount: minAmount,
      is_active: form.is_active,
    };
  };

  const handleSave = async () => {
    setFormError(null);
    const payload = buildPayload();
    if (!payload) return;
    try {
      if (editItem) {
        await updateMut.mutateAsync({ id: editItem.id, ...payload });
      } else {
        await createMut.mutateAsync(payload);
      }
      setShowModal(false);
    } catch (err) {
      const e = err as { message?: string };
      setFormError(e.message ?? "保存失败");
    }
  };

  const handleDelete = async () => {
    if (!deleteConfirm) return;
    await deleteMut.mutateAsync(deleteConfirm.id);
    setDeleteConfirm(null);
  };

  return (
    <div className="page-container">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-lg font-bold text-foreground">充值赠送</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            按时间段配置充值赠送活动，活动期间用户充值时按比例自动到账赠送额度
          </p>
        </div>
        <button
          onClick={openCreate}
          className="btn-primary flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium"
        >
          <Plus size={14} /> 新建活动
        </button>
      </div>

      <div className="grid grid-cols-1 gap-4 mb-5 sm:grid-cols-2">
        <div
          className="flex-1 bg-card border border-border rounded-xl px-5 py-4 shadow-card flex items-center gap-3 stagger-item"
          style={{ animationDelay: "0ms" }}
        >
          <div className="w-9 h-9 bg-primary/10 rounded-lg flex items-center justify-center">
            <Gift size={16} className="text-primary" />
          </div>
          <div>
            <div className="text-lg font-bold text-foreground">{list.length}</div>
            <div className="text-xs text-muted-foreground">活动总数</div>
          </div>
        </div>
        <div
          className="flex-1 bg-card border border-border rounded-xl px-5 py-4 shadow-card flex items-center gap-3 stagger-item"
          style={{ animationDelay: "80ms" }}
        >
          <div className="w-9 h-9 bg-emerald-50 dark:bg-emerald-500/10 rounded-lg flex items-center justify-center">
            <Gift size={16} className="text-emerald-500" />
          </div>
          <div>
            <div className="text-lg font-bold text-foreground">{activeCount}</div>
            <div className="text-xs text-muted-foreground">进行中</div>
          </div>
        </div>
      </div>

      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-20 text-muted-foreground">
            <Loader size={16} className="animate-spin mr-2" /> 加载中...
          </div>
        ) : list.length === 0 ? (
          <div className="text-center py-20 text-muted-foreground text-sm">暂无活动</div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">活动名称</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">活动时间</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">赠送比例</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">最低门槛</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">状态</TableHead>
                <TableHead className="text-right px-5 py-3.5 text-xs font-medium text-muted-foreground">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((p, i) => {
                const status = promoStatus(p);
                return (
                  <TableRow
                    key={p.id}
                    className={`border-t border-border hover:bg-accent/30 ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                  >
                    <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground">{p.name}</TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                      {formatRange(p.starts_at, p.ends_at)}
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm font-semibold text-amber-600 dark:text-amber-400">
                      +{(p.bonus_ratio * 100).toFixed(2).replace(/\.?0+$/, "")}%
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                      {p.min_recharge_amount > 0 ? `¥${p.min_recharge_amount.toFixed(2)}` : "无"}
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${status.className}`}>
                        {status.label}
                      </span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => openEdit(p)}
                          className="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-primary/10 hover:text-primary transition-colors cursor-pointer"
                          aria-label="编辑"
                        >
                          <Pencil size={13} />
                        </button>
                        <button
                          onClick={() => setDeleteConfirm(p)}
                          className="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors cursor-pointer"
                          aria-label="删除"
                        >
                          <Trash2 size={13} />
                        </button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </div>

      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-[560px] overflow-y-auto rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-base font-bold text-foreground">
                {editItem ? "编辑充值赠送活动" : "新建充值赠送活动"}
              </h3>
              <button
                onClick={() => setShowModal(false)}
                className="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground cursor-pointer"
                aria-label="关闭"
              >
                <X size={14} />
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">活动名称</label>
                <input
                  className="input-field w-full"
                  placeholder="例如：双11充值狂欢"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
              </div>

              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">开始时间</label>
                  <input
                    type="datetime-local"
                    className="input-field w-full"
                    value={form.starts_at}
                    onChange={(e) => setForm({ ...form, starts_at: e.target.value })}
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">结束时间</label>
                  <input
                    type="datetime-local"
                    className="input-field w-full"
                    value={form.ends_at}
                    onChange={(e) => setForm({ ...form, ends_at: e.target.value })}
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">
                    赠送比例（%）
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    min="0.01"
                    max="100"
                    className="input-field w-full"
                    placeholder="10"
                    value={form.bonus_percent}
                    onChange={(e) => setForm({ ...form, bonus_percent: e.target.value })}
                  />
                  <div className="text-[11px] text-muted-foreground mt-1">
                    例如填 10，则充 ¥100 赠送 ¥10
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">
                    最低充值门槛（元）
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    className="input-field w-full"
                    placeholder="0 表示无门槛"
                    value={form.min_recharge_amount}
                    onChange={(e) => setForm({ ...form, min_recharge_amount: e.target.value })}
                  />
                </div>
              </div>

              <div className="flex items-center gap-2">
                <input
                  id="recharge-promo-active"
                  type="checkbox"
                  checked={form.is_active}
                  onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                />
                <label htmlFor="recharge-promo-active" className="text-sm text-foreground select-none">
                  启用此活动
                </label>
              </div>

              {formError && (
                <div className="text-xs text-destructive bg-destructive/5 border border-destructive/20 rounded-lg px-3 py-2">
                  {formError}
                </div>
              )}
            </div>

            <div className="flex gap-3 justify-end mt-6">
              <button onClick={() => setShowModal(false)} className="btn-secondary px-4 py-2 rounded-lg text-sm">
                取消
              </button>
              <button
                onClick={handleSave}
                disabled={createMut.isPending || updateMut.isPending}
                className="btn-primary px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
              >
                {createMut.isPending || updateMut.isPending ? "保存中..." : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}

      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="w-[calc(100vw-2rem)] max-w-[400px] rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <h3 className="text-base font-bold text-foreground mb-1">确认删除</h3>
            <p className="text-sm text-muted-foreground mb-4">
              确定要删除活动「{deleteConfirm.name}」吗？此操作不可撤销。
            </p>
            <div className="flex gap-3 justify-end">
              <button onClick={() => setDeleteConfirm(null)} className="btn-secondary px-4 py-2 rounded-lg text-sm">
                取消
              </button>
              <button
                onClick={handleDelete}
                disabled={deleteMut.isPending}
                className="bg-destructive text-destructive-foreground px-4 py-2 rounded-lg text-sm font-medium hover:bg-destructive/90 disabled:opacity-50 cursor-pointer"
              >
                {deleteMut.isPending ? "删除中..." : "删除"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminRechargePromotions;
