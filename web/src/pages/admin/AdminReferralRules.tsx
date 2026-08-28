import { useState } from "react";
import { Gift, Loader2, Plus } from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiGet, apiPost } from "@/lib/api-client";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
interface ReferralRule {
  id: number;
  inviter_bonus_cny: number;
  invitee_bonus_cny: number;
  min_first_recharge_cny: number;
  enabled: boolean;
  effective_from: string;
  created_at: string;
}

interface ReferralRulesResponse {
  rules: ReferralRule[];
  active: ReferralRule | null;
}

interface FormState {
  inviter_bonus_cny: string;
  invitee_bonus_cny: string;
  min_first_recharge_cny: string;
  enabled: boolean;
  effective_from: string;
}

const emptyForm: FormState = {
  inviter_bonus_cny: "",
  invitee_bonus_cny: "",
  min_first_recharge_cny: "",
  enabled: true,
  effective_from: "",
};

const formatTime = (iso: string) => new Date(iso).toLocaleString("zh-CN");

const AdminReferralRules = () => {
  const qc = useQueryClient();
  const [form, setForm] = useState<FormState>(emptyForm);
  const [formError, setFormError] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["admin", "referral-rules"],
    queryFn: () => apiGet<ReferralRulesResponse>("/api/admin/referral/rules"),
  });

  const createMut = useMutation({
    mutationFn: (body: {
      inviter_bonus_cny: number;
      invitee_bonus_cny: number;
      min_first_recharge_cny: number;
      enabled: boolean;
      effective_from?: string;
    }) => apiPost("/api/admin/referral/rules", body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "referral-rules"] });
      setForm(emptyForm);
      toast.success("返佣规则已创建");
    },
    onError: (err: Error) => {
      toast.error(err.message || "创建失败");
    },
  });

  const rules = data?.rules ?? [];
  const active = data?.active ?? null;

  const handleSubmit = () => {
    setFormError(null);
    const inviter = Number(form.inviter_bonus_cny);
    const invitee = Number(form.invitee_bonus_cny);
    const minRecharge = Number(form.min_first_recharge_cny);
    if (form.inviter_bonus_cny === "" || !Number.isFinite(inviter) || inviter < 0) {
      setFormError("邀请人奖励金额无效");
      return;
    }
    if (form.invitee_bonus_cny === "" || !Number.isFinite(invitee) || invitee < 0) {
      setFormError("被邀请人奖励金额无效");
      return;
    }
    if (form.min_first_recharge_cny === "" || !Number.isFinite(minRecharge) || minRecharge < 0) {
      setFormError("首充门槛金额无效");
      return;
    }
    createMut.mutate({
      inviter_bonus_cny: inviter,
      invitee_bonus_cny: invitee,
      min_first_recharge_cny: minRecharge,
      enabled: form.enabled,
      ...(form.effective_from
        ? { effective_from: new Date(form.effective_from).toISOString() }
        : {}),
    });
  };

  return (
    <div className="page-container">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-lg font-bold text-foreground">返佣配置</h1>
        <p className="text-sm text-muted-foreground mt-0.5">
          配置邀请返佣规则。规则为追加式历史记录，最新生效的规则优先。
        </p>
      </div>

      {/* Active rule card */}
      <div className="bg-card border border-border rounded-xl shadow-card px-5 py-4 mb-5">
        <div className="flex items-center gap-2 mb-3">
          <div className="w-8 h-8 bg-primary/10 rounded-lg flex items-center justify-center">
            <Gift size={15} className="text-primary" />
          </div>
          <span className="text-sm font-semibold text-foreground">当前生效规则</span>
        </div>
        {isLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground py-2">
            <Loader2 size={14} className="animate-spin" /> 加载中...
          </div>
        ) : active ? (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <div className="text-xs text-muted-foreground mb-0.5">邀请人奖励</div>
              <div className="text-base font-bold text-foreground">
                ¥{active.inviter_bonus_cny.toFixed(2)}
              </div>
            </div>
            <div>
              <div className="text-xs text-muted-foreground mb-0.5">被邀请人奖励</div>
              <div className="text-base font-bold text-foreground">
                ¥{active.invitee_bonus_cny.toFixed(2)}
              </div>
            </div>
            <div>
              <div className="text-xs text-muted-foreground mb-0.5">首充门槛</div>
              <div className="text-base font-bold text-foreground">
                ¥{active.min_first_recharge_cny.toFixed(2)}
              </div>
            </div>
            <div>
              <div className="text-xs text-muted-foreground mb-0.5">生效时间</div>
              <div className="text-sm font-medium text-foreground">
                {formatTime(active.effective_from)}
              </div>
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            暂无生效的数据库规则，当前使用配置文件默认值。
          </p>
        )}
      </div>

      {/* New rule form */}
      <div className="bg-card border border-border rounded-xl shadow-card px-5 py-4 mb-5">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-sm font-semibold text-foreground">新增规则</span>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          规则为追加式历史，不可修改或删除；新增规则到达生效时间后，将覆盖旧规则（最新生效者优先）。
        </p>
        <div className="mb-3 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">
              邀请人奖励（元）
            </label>
            <input
              type="number"
              step="0.01"
              min="0"
              className="input-field w-full"
              placeholder="例如 10"
              value={form.inviter_bonus_cny}
              onChange={(e) => setForm({ ...form, inviter_bonus_cny: e.target.value })}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">
              被邀请人奖励（元）
            </label>
            <input
              type="number"
              step="0.01"
              min="0"
              className="input-field w-full"
              placeholder="例如 5"
              value={form.invitee_bonus_cny}
              onChange={(e) => setForm({ ...form, invitee_bonus_cny: e.target.value })}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">
              首充门槛（元）
            </label>
            <input
              type="number"
              step="0.01"
              min="0"
              className="input-field w-full"
              placeholder="0 表示无门槛"
              value={form.min_first_recharge_cny}
              onChange={(e) =>
                setForm({ ...form, min_first_recharge_cny: e.target.value })
              }
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">
              生效时间（留空 = 立即生效）
            </label>
            <input
              type="datetime-local"
              className="input-field w-full"
              value={form.effective_from}
              onChange={(e) => setForm({ ...form, effective_from: e.target.value })}
            />
          </div>
        </div>
        <div className="flex items-center gap-2 mb-3">
          <input
            id="referral-rule-enabled"
            type="checkbox"
            checked={form.enabled}
            onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
          />
          <label
            htmlFor="referral-rule-enabled"
            className="text-sm text-foreground select-none"
          >
            启用返佣（关闭表示该时间起停用返佣）
          </label>
        </div>
        {formError && (
          <div className="text-xs text-destructive bg-destructive/5 border border-destructive/20 rounded-lg px-3 py-2 mb-3">
            {formError}
          </div>
        )}
        <button
          onClick={handleSubmit}
          disabled={createMut.isPending}
          className="btn-primary flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
        >
          {createMut.isPending ? (
            <Loader2 size={14} className="animate-spin" />
          ) : (
            <Plus size={14} />
          )}
          新增规则
        </button>
      </div>

      {/* Rule history */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center gap-2">
          <span className="text-sm font-semibold text-foreground">规则历史</span>
          <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
            {isLoading ? "..." : `${rules.length} 条`}
          </span>
        </div>
        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 size={18} className="animate-spin text-muted-foreground" />
          </div>
        ) : rules.length === 0 ? (
          <div className="text-center py-16 text-sm text-muted-foreground">暂无规则</div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">ID</TableHead>
                <TableHead className="text-right px-5 py-3.5 text-xs font-medium text-muted-foreground">邀请人奖励</TableHead>
                <TableHead className="text-right px-5 py-3.5 text-xs font-medium text-muted-foreground">被邀请人奖励</TableHead>
                <TableHead className="text-right px-5 py-3.5 text-xs font-medium text-muted-foreground">首充门槛</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">状态</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">生效时间</TableHead>
                <TableHead className="text-left px-5 py-3.5 text-xs font-medium text-muted-foreground">创建时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rules.map((r, i) => (
                <TableRow
                  key={r.id}
                  className={`border-t border-border hover:bg-accent/30 transition-colors duration-150 ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                >
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{r.id}</TableCell>
                  <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground text-right">
                    ¥{r.inviter_bonus_cny.toFixed(2)}
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground text-right">
                    ¥{r.invitee_bonus_cny.toFixed(2)}
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground text-right">
                    {r.min_first_recharge_cny > 0
                      ? `¥${r.min_first_recharge_cny.toFixed(2)}`
                      : "无"}
                  </TableCell>
                  <TableCell className="px-5 py-3.5">
                    <div className="flex items-center gap-1.5">
                      <span
                        className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                          r.enabled
                            ? "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400"
                            : "bg-muted text-muted-foreground"
                        }`}
                      >
                        {r.enabled ? "启用" : "停用"}
                      </span>
                      {active?.id === r.id && (
                        <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-primary/10 text-primary">
                          当前生效
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                    {formatTime(r.effective_from)}
                  </TableCell>
                  <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                    {formatTime(r.created_at)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  );
};

export default AdminReferralRules;
