import { useState, useMemo } from "react";
import { Plus, Loader, Trash2, X, Pencil, Search } from "lucide-react";
import {
  useAdminSubscriptionPlans,
  useCreateSubscriptionPlan,
  useUpdateSubscriptionPlan,
  useDeleteSubscriptionPlan,
  useGatewayModels,
  type SubscriptionPlanInput,
} from "@/hooks/use-api";
import { Checkbox } from "@/components/ui/checkbox";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import type { SubscriptionPlan } from "@/types/api";

interface FormState {
  name: string;
  display_name: string;
  description: string;
  monthly_price_cny: string;
  quota_amount_cny: string;
  duration_days: string;
  sort_order: string;
  status: string;
  models: string[];
}

const emptyForm: FormState = {
  name: "",
  display_name: "",
  description: "",
  monthly_price_cny: "0",
  quota_amount_cny: "0",
  duration_days: "30",
  sort_order: "0",
  status: "active",
  models: [],
};

const AdminSubscriptionPlans = () => {
  const { data: plans, isLoading } = useAdminSubscriptionPlans();
  const { data: models } = useGatewayModels();
  const createMut = useCreateSubscriptionPlan();
  const updateMut = useUpdateSubscriptionPlan();
  const deleteMut = useDeleteSubscriptionPlan();

  const [showModal, setShowModal] = useState(false);
  const [editItem, setEditItem] = useState<SubscriptionPlan | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [formError, setFormError] = useState<string | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<SubscriptionPlan | null>(null);
  const [modelSearch, setModelSearch] = useState("");

  const list = plans ?? [];

  const modelNames = useMemo(
    () => (models ?? []).map((m) => m.name).sort((a, b) => a.localeCompare(b)),
    [models],
  );

  const filteredModels = useMemo(() => {
    const s = modelSearch.trim().toLowerCase();
    if (!s) return modelNames;
    return modelNames.filter((n) => n.toLowerCase().includes(s));
  }, [modelNames, modelSearch]);

  const openCreate = () => {
    setEditItem(null);
    setForm(emptyForm);
    setFormError(null);
    setModelSearch("");
    setShowModal(true);
  };

  const openEdit = (p: SubscriptionPlan) => {
    setEditItem(p);
    setForm({
      name: p.name,
      display_name: p.display_name,
      description: p.description ?? "",
      monthly_price_cny: p.monthly_price_cny.toString(),
      quota_amount_cny: p.quota_amount_cny.toString(),
      duration_days: p.duration_days.toString(),
      sort_order: p.sort_order.toString(),
      status: p.status,
      models: p.models ?? [],
    });
    setFormError(null);
    setModelSearch("");
    setShowModal(true);
  };

  const toggleModel = (name: string) => {
    setForm((prev) => ({
      ...prev,
      models: prev.models.includes(name)
        ? prev.models.filter((m) => m !== name)
        : [...prev.models, name],
    }));
  };

  const buildPayload = (): SubscriptionPlanInput | null => {
    if (!form.name.trim()) {
      setFormError("套餐标识（name）不能为空");
      return null;
    }
    const price = Number(form.monthly_price_cny);
    if (!Number.isFinite(price) || price < 0) {
      setFormError("月价不能为负");
      return null;
    }
    const quota = Number(form.quota_amount_cny);
    if (!Number.isFinite(quota) || quota < 0) {
      setFormError("配额不能为负");
      return null;
    }
    const duration = Number(form.duration_days);
    if (!Number.isInteger(duration) || duration <= 0) {
      setFormError("有效天数必须为正整数");
      return null;
    }
    const sort = Number(form.sort_order);
    if (!Number.isInteger(sort)) {
      setFormError("排序值必须为整数");
      return null;
    }
    return {
      name: form.name.trim(),
      display_name: form.display_name.trim(),
      description: form.description.trim(),
      monthly_price_cny: price,
      quota_amount_cny: quota,
      duration_days: duration,
      sort_order: sort,
      status: form.status,
      models: form.models,
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
          <h1 className="text-lg font-bold text-foreground">套餐管理</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            管理订阅套餐及其可用模型，勾选的模型即该套餐订阅用户可调用的范围，修改即时生效
          </p>
        </div>
        <button
          onClick={openCreate}
          className="btn-primary flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium"
        >
          <Plus size={14} /> 新增套餐
        </button>
      </div>

      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-20 text-muted-foreground">
            <Loader size={16} className="animate-spin mr-2" /> 加载中...
          </div>
        ) : list.length === 0 ? (
          <div className="text-center py-20 text-muted-foreground text-sm">暂无套餐</div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">标识</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">显示名</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">月价</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">配额</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">天数</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">模型数</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">排序</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">状态</TableHead>
                <TableHead className="text-right px-5 py-3.5 text-xs font-medium text-muted-foreground">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((p, i) => (
                <TableRow
                  key={p.id}
                  className={`border-t border-border hover:bg-accent/30 ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                >
                  <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground">{p.name}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{p.display_name}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-foreground">¥{p.monthly_price_cny.toFixed(2)}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-foreground">¥{p.quota_amount_cny.toFixed(2)}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{p.duration_days}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{p.models?.length ?? 0}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{p.sort_order}</TableCell>
                  <TableCell className="px-5 py-3.5">
                    <span
                      className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                        p.status === "active"
                          ? "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400"
                          : "bg-muted text-muted-foreground"
                      }`}
                    >
                      {p.status === "active" ? "启用" : "停用"}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <button
                        onClick={() => openEdit(p)}
                        className="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-primary/10 hover:text-primary transition-colors cursor-pointer"
                      >
                        <Pencil size={13} />
                      </button>
                      <button
                        onClick={() => setDeleteConfirm(p)}
                        className="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors cursor-pointer"
                      >
                        <Trash2 size={13} />
                      </button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-[600px] overflow-y-auto rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-base font-bold text-foreground">
                {editItem ? "编辑套餐" : "新增套餐"}
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
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">
                    标识（name，唯一）
                  </label>
                  <input
                    className="input-field w-full"
                    placeholder="例如：pro"
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">显示名</label>
                  <input
                    className="input-field w-full"
                    placeholder="例如：专业版"
                    value={form.display_name}
                    onChange={(e) => setForm({ ...form, display_name: e.target.value })}
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">描述</label>
                <input
                  className="input-field w-full"
                  placeholder="套餐说明"
                  value={form.description}
                  onChange={(e) => setForm({ ...form, description: e.target.value })}
                />
              </div>

              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">月价（元）</label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    className="input-field w-full"
                    value={form.monthly_price_cny}
                    onChange={(e) => setForm({ ...form, monthly_price_cny: e.target.value })}
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">配额（元）</label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    className="input-field w-full"
                    value={form.quota_amount_cny}
                    onChange={(e) => setForm({ ...form, quota_amount_cny: e.target.value })}
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">有效天数</label>
                  <input
                    type="number"
                    step="1"
                    min="1"
                    className="input-field w-full"
                    value={form.duration_days}
                    onChange={(e) => setForm({ ...form, duration_days: e.target.value })}
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">排序</label>
                  <input
                    type="number"
                    step="1"
                    className="input-field w-full"
                    value={form.sort_order}
                    onChange={(e) => setForm({ ...form, sort_order: e.target.value })}
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">状态</label>
                  <select
                    className="input-field w-full"
                    value={form.status}
                    onChange={(e) => setForm({ ...form, status: e.target.value })}
                  >
                    <option value="active">启用</option>
                    <option value="inactive">停用</option>
                  </select>
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="block text-xs font-medium text-muted-foreground">
                    可用模型（已选 {form.models.length}）
                  </label>
                </div>
                <div className="relative mb-2">
                  <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
                  <input
                    className="input-field w-full pl-8"
                    placeholder="搜索模型"
                    value={modelSearch}
                    onChange={(e) => setModelSearch(e.target.value)}
                  />
                </div>
                <div className="border border-border rounded-lg max-h-56 overflow-y-auto divide-y divide-border">
                  {filteredModels.length === 0 ? (
                    <div className="text-center py-6 text-xs text-muted-foreground">无匹配模型</div>
                  ) : (
                    filteredModels.map((name) => (
                      <label
                        key={name}
                        className="flex items-center gap-2.5 px-3 py-2 hover:bg-accent/30 cursor-pointer select-none"
                      >
                        <Checkbox
                          checked={form.models.includes(name)}
                          onCheckedChange={() => toggleModel(name)}
                        />
                        <span className="text-sm text-foreground">{name}</span>
                      </label>
                    ))
                  )}
                </div>
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
              确定要删除套餐「{deleteConfirm.display_name || deleteConfirm.name}」吗？此操作不可撤销，且会移除其模型关联。
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

export default AdminSubscriptionPlans;
