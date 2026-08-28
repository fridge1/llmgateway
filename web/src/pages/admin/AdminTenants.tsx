import { useState, useRef, useMemo } from "react";
import {
  Building2, DollarSign, Trash2, ChevronLeft, ChevronRight,
  Loader2, X, Eye, Plus, Search, Pencil, Wallet, TrendingUp,
  ArrowUp, ArrowDown, Shield, ShieldAlert, AlertTriangle,
} from "lucide-react";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import {
  useAdminTenants,
  useAdminTenantBalance,
  useAdminTenantTransactions,
  useAdminRechargeTenant,
  useAdminDeleteTenant,
  useAdminCreateTenant,
  useAdminUsers,
  useAdminTenantPricing,
  useAdminUpsertTenantPricing,
  useAdminDeleteTenantPricing,
  useAdminPricing,
  useAdminTenantModelUpstreams,
  useAdminReplaceTenantModelUpstreams,
  useAdminDeleteTenantModelUpstreams,
  useGatewayModels,
} from "@/hooks/use-api";
import { cn } from "@/lib/utils";
import { formatPricingFactor } from "@/lib/utils";
import type { AdminTenant } from "@/hooks/use-api";
import type { UserWithBalance, TenantPricing, PricingTier, TenantModelUpstream, TenantUpstreamInput } from "@/types/api";

const PAGE_SIZE = 20;
const TX_PAGE_SIZE = 10;

const txTypeLabel: Record<string, string> = {
  consumption: "消费",
  recharge: "充值",
  freeze: "冻结",
  unfreeze: "解冻",
  settlement: "结算",
};

const txTypeBadge: Record<string, string> = {
  consumption: "bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400",
  recharge: "bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-400",
  freeze: "bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400",
  unfreeze: "bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400",
  settlement: "bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400",
};

// ---------------------------------------------------------------------------
// Tenant Detail Modal
// ---------------------------------------------------------------------------

// 只在“鼠标按下时落点就是遮罩层本身”时才关闭，避免标签切换/内容高度变化导致
// mouseup 落点漂移到遮罩层而误触 onClose。
function useBackdropClose(onClose: () => void) {
  const downOnBackdrop = useRef(false);
  return {
    onMouseDown: (e: React.MouseEvent) => {
      downOnBackdrop.current = e.target === e.currentTarget;
    },
    onClick: (e: React.MouseEvent) => {
      if (downOnBackdrop.current && e.target === e.currentTarget) onClose();
    },
  };
}

const emptyPricingForm = () => ({
  modelName: "",
  discountPercent: "",
  billingType: "token",
  enabled: true,
});

// ---------------------------------------------------------------------------
// Tenant Model Upstreams Tab
// ---------------------------------------------------------------------------

const emptyTenantUpstream = (): TenantUpstreamInput => ({
  provider: "",
  protocol: "",
  protocols: [],
  upstream_provider: "",
  upstream_name: "",
  base_url: "",
  api_key: "",
  model_override: "",
  weight: 1,
});

const TENANT_PROTOCOL_OPTIONS: { value: string; label: string }[] = [
  { value: "openai", label: "OpenAI (Chat Completions)" },
  { value: "openai-compatible", label: "OpenAI 兼容" },
  { value: "anthropic", label: "Anthropic" },
  { value: "gemini", label: "Gemini" },
  { value: "responses", label: "Responses API" },
];

function TenantUpstreamsTab({ tenant }: { tenant: AdminTenant }) {
  const { data: upstreamsData, isLoading } = useAdminTenantModelUpstreams(tenant.id);
  const { data: models } = useGatewayModels();
  const replaceUpstreams = useAdminReplaceTenantModelUpstreams();
  const deleteUpstreams = useAdminDeleteTenantModelUpstreams();

  const [showForm, setShowForm] = useState(false);
  const [editingModel, setEditingModel] = useState<string | null>(null);
  const [formModelName, setFormModelName] = useState("");
  const [formUpstreams, setFormUpstreams] = useState<TenantUpstreamInput[]>([emptyTenantUpstream()]);
  const formBackdrop = useBackdropClose(() => setShowForm(false));

  const globalModelNames = useMemo(() => new Set((models ?? []).map((m) => m.name)), [models]);

  // Group flat rows by model_name (backend returns them ordered by model, sort_order).
  const grouped = useMemo(() => {
    const map = new Map<string, TenantModelUpstream[]>();
    for (const u of upstreamsData?.upstreams ?? []) {
      const arr = map.get(u.model_name) ?? [];
      arr.push(u);
      map.set(u.model_name, arr);
    }
    return map;
  }, [upstreamsData]);

  const configuredModels = [...grouped.keys()];
  const availableModels = (models ?? []).filter((m) => !grouped.has(m.name));

  const openAdd = () => {
    setEditingModel(null);
    setFormModelName("");
    setFormUpstreams([emptyTenantUpstream()]);
    setShowForm(true);
  };

  const openEdit = (modelName: string) => {
    setEditingModel(modelName);
    setFormModelName(modelName);
    setFormUpstreams(
      (grouped.get(modelName) ?? []).map((u) => ({
        provider: u.provider,
        protocol: u.protocol,
        protocols: u.protocols && u.protocols.length > 0
          ? u.protocols
          : u.protocol
          ? [u.protocol]
          : [],
        upstream_provider: u.upstream_provider,
        upstream_name: u.upstream_name,
        base_url: u.base_url,
        api_key: u.api_key,
        model_override: u.model_override,
        weight: u.weight,
      })),
    );
    setShowForm(true);
  };

  const updateUpstream = (idx: number, field: keyof TenantUpstreamInput, value: string | number | string[]) => {
    setFormUpstreams((prev) => {
      const arr = [...prev];
      arr[idx] = { ...arr[idx], [field]: value };
      return arr;
    });
  };

  const moveUpstream = (idx: number, dir: "up" | "down") => {
    setFormUpstreams((prev) => {
      const arr = [...prev];
      const j = dir === "up" ? idx - 1 : idx + 1;
      if (j < 0 || j >= arr.length) return prev;
      [arr[idx], arr[j]] = [arr[j], arr[idx]];
      return arr;
    });
  };

  const removeUpstream = (idx: number) => {
    setFormUpstreams((prev) => (prev.length > 1 ? prev.filter((_, i) => i !== idx) : prev));
  };

  const formValid =
    formModelName.trim() !== "" &&
    formUpstreams.length > 0 &&
    formUpstreams.every((u) => u.provider.trim() && u.base_url.trim() && u.api_key.trim());

  const handleSave = () => {
    if (!formValid) return;
    replaceUpstreams.mutate(
      { tenantId: tenant.id, modelName: formModelName.trim(), upstreams: formUpstreams },
      { onSuccess: () => { setShowForm(false); setEditingModel(null); } },
    );
  };

  const handleDelete = (modelName: string) => {
    if (!confirm(`确定要删除模型「${modelName}」的专属上游吗？删除后该租户将走全局默认上游。`)) return;
    deleteUpstreams.mutate({ tenantId: tenant.id, modelName });
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h4 className="text-sm font-medium text-foreground">
          专属模型上游
          <span className="ml-1.5 text-xs text-muted-foreground">({configuredModels.length})</span>
        </h4>
        <button
          onClick={openAdd}
          className="flex items-center gap-1 px-3 py-1.5 text-xs font-medium bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
        >
          <Plus size={13} />
          添加配置
        </button>
      </div>

      <div className="mb-3 text-xs text-muted-foreground bg-muted/50 rounded-lg px-3 py-2">
        配置后，该租户（含子用户）对此模型的请求将<span className="font-medium text-foreground">只走专属上游</span>，全部失败时直接报错，不会回退到全局上游。未配置的模型仍走全局默认。
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 size={16} className="animate-spin text-primary" />
        </div>
      ) : configuredModels.length === 0 ? (
        <div className="text-center py-8 text-sm text-muted-foreground border border-dashed border-border rounded-xl">
          暂无专属上游，所有模型走全局默认上游
        </div>
      ) : (
        <div className="border border-border rounded-lg overflow-hidden">
          <Table className="w-full text-sm">
            <TableHeader className="bg-muted/40">
              <TableRow>
                <TableHead className="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground">模型</TableHead>
                <TableHead className="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground">上游</TableHead>
                <TableHead className="text-right px-3 py-2.5 text-xs font-medium text-muted-foreground">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {configuredModels.map((modelName) => {
                const ups = grouped.get(modelName) ?? [];
                const orphan = !globalModelNames.has(modelName);
                return (
                  <TableRow key={modelName} className="border-t border-border/50 hover:bg-muted/20">
                    <TableCell className="px-3 py-2.5">
                      <div className="flex items-center gap-1.5">
                        <span className="font-medium text-xs">{modelName}</span>
                        {orphan && (
                          <span
                            className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400"
                            title="该模型已不在全局模型列表中，此配置不会生效"
                          >
                            <AlertTriangle size={10} />
                            模型不存在
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="px-3 py-2.5">
                      <div className="flex flex-col gap-0.5">
                        {ups.map((u, i) => (
                          <span key={u.id} className="text-xs text-muted-foreground font-mono truncate max-w-[280px]">
                            {i === 0 ? "主 " : `备${i} `}
                            {u.provider} · {u.base_url}
                          </span>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="px-3 py-2.5 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button onClick={() => openEdit(modelName)} className="p-1 rounded hover:bg-muted transition-colors" title="编辑">
                          <Pencil size={13} className="text-muted-foreground" />
                        </button>
                        <button onClick={() => handleDelete(modelName)} className="p-1 rounded hover:bg-muted transition-colors" title="删除">
                          <Trash2 size={13} className="text-destructive" />
                        </button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {/* Upstreams Form Modal */}
      {showForm && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center modal-backdrop" role="dialog" aria-modal="true" {...formBackdrop}>
          <div
            className="bg-card border border-border rounded-xl shadow-modal w-full max-w-xl max-h-[80vh] overflow-y-auto p-5 mx-4 slide-up"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-4">
              <h4 className="text-sm font-semibold">{editingModel ? "编辑专属上游" : "添加专属上游"}</h4>
              <button onClick={() => setShowForm(false)} className="flex h-9 w-9 items-center justify-center rounded-lg hover:bg-muted" aria-label="关闭">
                <X size={16} />
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium mb-1">模型</label>
                {editingModel ? (
                  <input type="text" value={formModelName} disabled className="input-field opacity-60" />
                ) : (
                  <select
                    value={formModelName}
                    onChange={(e) => setFormModelName(e.target.value)}
                    className="input-field"
                  >
                    <option value="">选择模型...</option>
                    {availableModels.map((m) => (
                      <option key={m.name} value={m.name}>{m.name}</option>
                    ))}
                  </select>
                )}
              </div>

              {formUpstreams.map((upstream, idx) => (
                <div key={idx} className="border border-border rounded-lg p-3 space-y-3">
                  <div className="flex items-center justify-between">
                    {idx === 0 ? (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-primary/15 text-primary text-xs font-semibold">
                        <Shield size={11} />
                        主渠道
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-muted text-muted-foreground text-xs font-semibold">
                        <ShieldAlert size={11} />
                        备选渠道 {idx}
                      </span>
                    )}
                    <div className="flex items-center gap-1">
                      {idx > 0 && (
                        <button onClick={() => moveUpstream(idx, "up")} className="flex h-9 w-9 items-center justify-center rounded hover:bg-muted transition-colors text-muted-foreground hover:text-foreground" title="上移" aria-label="上移">
                          <ArrowUp size={12} />
                        </button>
                      )}
                      {idx < formUpstreams.length - 1 && (
                        <button onClick={() => moveUpstream(idx, "down")} className="flex h-9 w-9 items-center justify-center rounded hover:bg-muted transition-colors text-muted-foreground hover:text-foreground" title="下移" aria-label="下移">
                          <ArrowDown size={12} />
                        </button>
                      )}
                      {formUpstreams.length > 1 && (
                        <button onClick={() => removeUpstream(idx)} className="flex h-9 w-9 items-center justify-center rounded hover:bg-destructive/10 transition-colors text-muted-foreground hover:text-destructive" title="删除" aria-label="删除">
                          <Trash2 size={12} />
                        </button>
                      )}
                    </div>
                  </div>
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                    <div>
                      <label className="block text-xs font-medium text-muted-foreground mb-1">提供商</label>
                      <input
                        className="input-field text-sm"
                        placeholder="例如：openai"
                        value={upstream.provider}
                        onChange={(e) => updateUpstream(idx, "provider", e.target.value)}
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-muted-foreground mb-1">协议（可多选）</label>
                      <div className="mt-1 grid grid-cols-1 gap-1.5 sm:grid-cols-2">
                        {TENANT_PROTOCOL_OPTIONS.map((opt) => {
                          const checked = upstream.protocols?.includes(opt.value) ?? false;
                          return (
                            <label key={opt.value} className="flex items-center gap-1.5 text-xs">
                              <input
                                type="checkbox"
                                checked={checked}
                                onChange={(e) => {
                                  const cur = new Set(upstream.protocols ?? []);
                                  if (e.target.checked) cur.add(opt.value);
                                  else cur.delete(opt.value);
                                  updateUpstream(idx, "protocols", Array.from(cur));
                                }}
                              />
                              {opt.label}
                            </label>
                          );
                        })}
                      </div>
                      <p className="text-[11px] text-muted-foreground mt-1 leading-snug">
                        上游可同时支持多种协议。<b>入口优先路由到与客户端协议一致的上游做透传</b>；该协议全部失败时，回退到声明了 OpenAI 兼容协议的上游做协议转换兜底。Gemini 入口不做转换。
                      </p>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-muted-foreground mb-1">上游提供商</label>
                      <input
                        className="input-field text-sm"
                        placeholder="可选"
                        value={upstream.upstream_provider}
                        onChange={(e) => updateUpstream(idx, "upstream_provider", e.target.value)}
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                    <div>
                      <label className="block text-xs font-medium text-muted-foreground mb-1">上游名称</label>
                      <input
                        className="input-field text-sm"
                        placeholder="可选，例如：tenant-us"
                        value={upstream.upstream_name}
                        onChange={(e) => updateUpstream(idx, "upstream_name", e.target.value)}
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-muted-foreground mb-1">Model Override</label>
                      <input
                        className="input-field text-sm"
                        placeholder="上游模型名（可选）"
                        value={upstream.model_override}
                        onChange={(e) => updateUpstream(idx, "model_override", e.target.value)}
                      />
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-muted-foreground mb-1">Base URL</label>
                    <input
                      className="input-field text-sm"
                      placeholder="例如：https://api.openai.com/v1"
                      value={upstream.base_url}
                      onChange={(e) => updateUpstream(idx, "base_url", e.target.value)}
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-muted-foreground mb-1">API Key</label>
                    <input
                      className="input-field text-sm"
                      placeholder="sk-..."
                      value={upstream.api_key}
                      onChange={(e) => updateUpstream(idx, "api_key", e.target.value)}
                    />
                  </div>
                </div>
              ))}

              <button
                onClick={() => setFormUpstreams((prev) => [...prev, emptyTenantUpstream()])}
                className="flex items-center justify-center gap-1.5 w-full py-2.5 rounded-lg border border-dashed border-border text-sm text-muted-foreground hover:text-foreground hover:border-primary/40 hover:bg-primary/3 transition-colors"
              >
                <Plus size={14} />
                添加备选渠道
              </button>

              <div className="flex justify-end gap-2 pt-1">
                <button onClick={() => setShowForm(false)} className="btn-secondary px-3 py-1.5 text-xs">取消</button>
                <button
                  onClick={handleSave}
                  disabled={replaceUpstreams.isPending || !formValid}
                  className="btn-primary px-3 py-1.5 text-xs disabled:opacity-50"
                >
                  {replaceUpstreams.isPending ? "保存中..." : "保存"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function TenantDetailModal({
  tenant,
  onClose,
}: {
  tenant: AdminTenant;
  onClose: () => void;
}) {
  const [txPage, setTxPage] = useState(1);
  const { data: balanceData, isLoading: balanceLoading } = useAdminTenantBalance(tenant.id);
  const { data: txData, isLoading: txLoading } = useAdminTenantTransactions(tenant.id, txPage, TX_PAGE_SIZE);

  const { data: pricingData, isLoading: pricingLoading } = useAdminTenantPricing(tenant.id);
  const { data: globalPricingData } = useAdminPricing();
  const upsertPricing = useAdminUpsertTenantPricing();
  const deletePricing = useAdminDeleteTenantPricing();
  const [showPricingForm, setShowPricingForm] = useState(false);
  const [editingPricing, setEditingPricing] = useState<TenantPricing | null>(null);
  const [pricingForm, setPricingForm] = useState(emptyPricingForm());
  const backdrop = useBackdropClose(onClose);
  const pricingFormBackdrop = useBackdropClose(() => setShowPricingForm(false));

  const transactions = txData?.transactions ?? [];
  const txTotal = txData?.total ?? 0;
  const txTotalPages = Math.max(1, Math.ceil(txTotal / TX_PAGE_SIZE));
  const tenantPricingList = pricingData?.pricing ?? [];
  const globalPricingList = globalPricingData?.pricing ?? [];

  const openAddPricing = () => {
    setEditingPricing(null);
    setPricingForm(emptyPricingForm());
    setShowPricingForm(true);
  };

  const openEditPricing = (p: TenantPricing) => {
    setEditingPricing(p);
    setPricingForm({
      modelName: p.model_name,
      discountPercent: p.discount_rate != null ? String(Math.round(p.discount_rate * 1000) / 10) : "",
      billingType: p.billing_type || "token",
      enabled: p.is_active,
    });
    setShowPricingForm(true);
  };

  const copyFromGlobal = (modelName: string) => {
    const gp = globalPricingList.find((p) => p.model_name === modelName);
    if (gp) {
      setPricingForm((prev) => ({
        ...prev,
        modelName: gp.model_name,
        billingType: gp.billing_type || "token",
      }));
    }
  };

  const handleSavePricing = () => {
    const modelName = pricingForm.modelName.trim();
    if (!modelName) return;
    const pct = parseFloat(pricingForm.discountPercent);
    if (!(pct > 0 && pct <= 1000)) {
      alert("请输入有效的定价因子（0-1000 之间，例如 80=8折，120=提价20%）");
      return;
    }
    const rate = Math.round((pct / 100) * 10000) / 10000;
    upsertPricing.mutate(
      {
        tenantId: tenant.id,
        modelName,
        pricing: {
          input_price: 0,
          output_price: 0,
          billing_type: pricingForm.billingType,
          is_active: pricingForm.enabled,
          discount_rate: rate,
        },
      },
      { onSuccess: () => { setShowPricingForm(false); setEditingPricing(null); } },
    );
  };

  const handleDeletePricing = (modelName: string) => {
    if (!confirm(`确定要删除模型「${modelName}」的自定义定价吗？删除后将使用全局定价。`)) return;
    deletePricing.mutate({ tenantId: tenant.id, modelName });
  };

  const overriddenModels = new Set(tenantPricingList.map((p) => p.model_name));
  const availableModels = globalPricingList.filter((p) => !overriddenModels.has(p.model_name));

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true" {...backdrop}>
      <div
        className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-3xl overflow-y-auto rounded-xl border border-border bg-card p-4 shadow-modal slide-up sm:p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-2.5">
            <div className="w-9 h-9 rounded-lg bg-primary/10 flex items-center justify-center">
              <Building2 size={16} className="text-primary" />
            </div>
            <div>
              <h3 className="text-base font-semibold text-foreground">{tenant.name}</h3>
              <p className="text-xs text-muted-foreground font-mono">{tenant.id}</p>
            </div>
          </div>
          <button onClick={onClose} className="p-1.5 hover:bg-muted rounded-lg transition-colors">
            <X size={18} />
          </button>
        </div>

        <Tabs defaultValue="balance" className="gap-4">
          <TabsList className="h-9">
            <TabsTrigger value="balance" className="px-4 text-xs">余额 & 交易</TabsTrigger>
            <TabsTrigger value="pricing" className="px-4 text-xs">定价管理</TabsTrigger>
            <TabsTrigger value="upstreams" className="px-4 text-xs">专属上游</TabsTrigger>
          </TabsList>

          <TabsContent value="balance">
            {/* Balance Cards */}
            <div className="mb-5 grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="p-4 bg-emerald-50/50 dark:bg-emerald-900/10 border border-emerald-100 dark:border-emerald-900/30 rounded-xl">
                <div className="flex items-center gap-2 mb-1">
                  <Wallet size={14} className="text-emerald-600" />
                  <span className="text-xs text-muted-foreground">可用余额</span>
                </div>
                {balanceLoading ? (
                  <Loader2 size={14} className="animate-spin text-muted-foreground" />
                ) : (
                  <p className="text-xl font-bold text-foreground">¥{(balanceData?.balance ?? 0).toFixed(2)}</p>
                )}
              </div>
              <div className="p-4 bg-amber-50/50 dark:bg-amber-900/10 border border-amber-100 dark:border-amber-900/30 rounded-xl">
                <div className="flex items-center gap-2 mb-1">
                  <TrendingUp size={14} className="text-amber-600" />
                  <span className="text-xs text-muted-foreground">冻结金额</span>
                </div>
                {balanceLoading ? (
                  <Loader2 size={14} className="animate-spin text-muted-foreground" />
                ) : (
                  <p className="text-xl font-bold text-foreground">¥{(balanceData?.frozen ?? 0).toFixed(2)}</p>
                )}
              </div>
            </div>

            {/* Transactions */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-sm font-medium text-foreground">交易记录</h4>
                <span className="text-xs text-muted-foreground">共 {txTotal} 条</span>
              </div>
              {txLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 size={16} className="animate-spin text-primary mr-2" />
                  <span className="text-sm text-muted-foreground">加载中...</span>
                </div>
              ) : transactions.length === 0 ? (
                <div className="text-center py-8 text-sm text-muted-foreground">暂无交易记录</div>
              ) : (
                <>
                  <div className="border border-border rounded-lg overflow-hidden">
                    <Table className="w-full text-sm">
                      <TableHeader className="bg-muted/40">
                        <TableRow>
                          <TableHead className="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground">类型</TableHead>
                          <TableHead className="text-right px-3 py-2.5 text-xs font-medium text-muted-foreground">金额</TableHead>
                          <TableHead className="text-right px-3 py-2.5 text-xs font-medium text-muted-foreground">余额</TableHead>
                          <TableHead className="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground">模型</TableHead>
                          <TableHead className="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground">描述</TableHead>
                          <TableHead className="text-right px-3 py-2.5 text-xs font-medium text-muted-foreground">时间</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {transactions.map((tx) => (
                          <TableRow key={tx.id} className="border-t border-border/50 hover:bg-muted/20">
                            <TableCell className="px-3 py-2.5">
                              <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${txTypeBadge[tx.type] ?? "bg-muted text-muted-foreground"}`}>
                                {txTypeLabel[tx.type] ?? tx.type}
                              </span>
                            </TableCell>
                            <TableCell className="px-3 py-2.5 text-right font-mono text-xs">
                              <span className={tx.amount > 0 ? "text-emerald-600" : "text-red-500"}>
                                {tx.amount > 0 ? "+" : ""}{tx.amount.toFixed(4)}
                              </span>
                            </TableCell>
                            <TableCell className="px-3 py-2.5 text-right font-mono text-xs text-muted-foreground">
                              {tx.balance_after.toFixed(2)}
                            </TableCell>
                            <TableCell className="px-3 py-2.5 text-xs text-muted-foreground max-w-[120px] truncate">
                              {tx.model || "-"}
                            </TableCell>
                            <TableCell className="px-3 py-2.5 text-xs text-muted-foreground max-w-[160px] truncate">
                              {tx.description || "-"}
                            </TableCell>
                            <TableCell className="px-3 py-2.5 text-right text-xs text-muted-foreground whitespace-nowrap">
                              {new Date(tx.created_at).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                  {txTotalPages > 1 && (
                    <div className="flex items-center justify-center gap-2 mt-3">
                      <button
                        onClick={() => setTxPage((p) => Math.max(1, p - 1))}
                        disabled={txPage === 1}
                        className="p-1.5 rounded-lg hover:bg-muted disabled:opacity-40 transition-colors"
                      >
                        <ChevronLeft size={14} />
                      </button>
                      <span className="text-xs text-muted-foreground">
                        {txPage} / {txTotalPages}
                      </span>
                      <button
                        onClick={() => setTxPage((p) => Math.min(txTotalPages, p + 1))}
                        disabled={txPage === txTotalPages}
                        className="p-1.5 rounded-lg hover:bg-muted disabled:opacity-40 transition-colors"
                      >
                        <ChevronRight size={14} />
                      </button>
                    </div>
                  )}
                </>
              )}
            </div>
          </TabsContent>

          <TabsContent value="pricing">
            <div>
              <div className="flex items-center justify-between mb-4">
                <h4 className="text-sm font-medium text-foreground">
                  自定义模型定价
                  <span className="ml-1.5 text-xs text-muted-foreground">({tenantPricingList.length})</span>
                </h4>
                <button
                  onClick={openAddPricing}
                  className="flex items-center gap-1 px-3 py-1.5 text-xs font-medium bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
                >
                  <Plus size={13} />
                  添加定价
                </button>
              </div>

              {pricingLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 size={16} className="animate-spin text-primary" />
                </div>
              ) : tenantPricingList.length === 0 ? (
                <div className="text-center py-8 text-sm text-muted-foreground border border-dashed border-border rounded-xl">
                  暂无自定义定价，将使用全局定价
                </div>
              ) : (
                <div className="border border-border rounded-lg overflow-hidden">
                  <Table className="w-full text-sm">
                    <TableHeader className="bg-muted/40">
                      <TableRow>
                        <TableHead className="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground">模型</TableHead>
                        <TableHead className="text-right px-3 py-2.5 text-xs font-medium text-muted-foreground">折扣</TableHead>
                        <TableHead className="text-right px-3 py-2.5 text-xs font-medium text-muted-foreground">输入(实际)</TableHead>
                        <TableHead className="text-right px-3 py-2.5 text-xs font-medium text-muted-foreground">输出(实际)</TableHead>
                        <TableHead className="text-center px-3 py-2.5 text-xs font-medium text-muted-foreground">状态</TableHead>
                        <TableHead className="text-right px-3 py-2.5 text-xs font-medium text-muted-foreground">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {tenantPricingList.map((p) => {
                        const gp = globalPricingList.find((g) => g.model_name === p.model_name);
                        const hasDiscount = p.discount_rate != null;
                        const rate = p.discount_rate ?? 1;
                        const effIn = hasDiscount && gp ? gp.input_price * rate : p.input_price;
                        const effOut = hasDiscount && gp ? gp.output_price * rate : p.output_price;
                        return (
                        <TableRow key={p.model_name} className="border-t border-border/50 hover:bg-muted/20">
                          <TableCell className="px-3 py-2.5 font-medium text-xs">{p.model_name}</TableCell>
                          <TableCell className="px-3 py-2.5 text-right font-mono text-xs">
                            {hasDiscount ? (
                              <span className={cn(
                                rate < 1 ? "text-rose-600" :
                                rate === 1 ? "text-muted-foreground" :
                                "text-amber-600"
                              )}>
                                {formatPricingFactor(rate)}
                              </span>
                            ) : (
                              <span className="text-muted-foreground">—</span>
                            )}
                          </TableCell>
                          <TableCell className="px-3 py-2.5 text-right font-mono text-xs text-muted-foreground">{Math.round(effIn * 10000) / 10000}</TableCell>
                          <TableCell className="px-3 py-2.5 text-right font-mono text-xs text-muted-foreground">{Math.round(effOut * 10000) / 10000}</TableCell>
                          <TableCell className="px-3 py-2.5 text-center">
                            <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${
                              p.is_active
                                ? "bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-400"
                                : "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400"
                            }`}>
                              {p.is_active ? "启用" : "禁用"}
                            </span>
                          </TableCell>
                          <TableCell className="px-3 py-2.5 text-right">
                            <div className="flex items-center justify-end gap-1">
                              <button onClick={() => openEditPricing(p)} className="p-1 rounded hover:bg-muted transition-colors" title="编辑">
                                <Pencil size={13} className="text-muted-foreground" />
                              </button>
                              <button onClick={() => handleDeletePricing(p.model_name)} className="p-1 rounded hover:bg-muted transition-colors" title="删除">
                                <Trash2 size={13} className="text-destructive" />
                              </button>
                            </div>
                          </TableCell>
                        </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </div>
              )}

              {/* Pricing Form Modal */}
              {showPricingForm && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center modal-backdrop" role="dialog" aria-modal="true" {...pricingFormBackdrop}>
                  <div className="bg-card border border-border rounded-xl shadow-modal w-full max-w-md p-5 mx-4 slide-up" onClick={(e) => e.stopPropagation()}>
                    <div className="flex items-center justify-between mb-4">
                      <h4 className="text-sm font-semibold">{editingPricing ? "编辑定价" : "添加定价"}</h4>
                      <button onClick={() => setShowPricingForm(false)} className="flex h-9 w-9 items-center justify-center rounded-lg hover:bg-muted" aria-label="关闭">
                        <X size={16} />
                      </button>
                    </div>
                    <div className="space-y-3">
                      <div>
                        <label className="block text-xs font-medium mb-1">模型名称</label>
                        {editingPricing ? (
                          <input type="text" value={pricingForm.modelName} disabled className="input-field opacity-60" />
                        ) : (
                          <div className="space-y-1.5">
                            <input
                              type="text"
                              value={pricingForm.modelName}
                              onChange={(e) => setPricingForm({ ...pricingForm, modelName: e.target.value })}
                              className="input-field"
                              placeholder="输入模型名称"
                            />
                            {availableModels.length > 0 && (
                              <select
                                value=""
                                onChange={(e) => { if (e.target.value) copyFromGlobal(e.target.value); }}
                                className="input-field text-xs"
                              >
                                <option value="">从全局定价复制...</option>
                                {availableModels.map((m) => (
                                  <option key={m.model_name} value={m.model_name}>{m.model_name}</option>
                                ))}
                              </select>
                            )}
                          </div>
                        )}
                      </div>
                      <div>
                        <label className="block text-xs font-medium mb-1">定价因子（%）</label>
                        <input
                          type="number"
                          step="0.1"
                          min="0"
                          max="1000"
                          value={pricingForm.discountPercent}
                          onChange={(e) => setPricingForm({ ...pricingForm, discountPercent: e.target.value })}
                          className="input-field"
                          placeholder="80=8折，100=原价，120=提价20%"
                        />
                        <p className="mt-1 text-[11px] text-muted-foreground">
                          实际价 = 全局价 × 定价因子。全局调价时本租户价格自动联动。
                        </p>
                        {(() => {
                          const pct = parseFloat(pricingForm.discountPercent);
                          const gp = globalPricingList.find((g) => g.model_name === pricingForm.modelName.trim());
                          if (!(pct > 0 && pct <= 1000) || !gp) return null;
                          const rate = pct / 100;
                          return (
                            <div className="mt-2 rounded-lg bg-muted/40 px-3 py-2 text-[11px] font-mono text-muted-foreground">
                              预览：输入 {Math.round(gp.input_price * rate * 10000) / 10000} / 输出 {Math.round(gp.output_price * rate * 10000) / 10000}
                              <span className="ml-1 text-[10px]">（全局 {gp.input_price}/{gp.output_price}）</span>
                            </div>
                          );
                        })()}
                      </div>
                      <div className="flex items-center gap-3">
                        <label className="flex items-center gap-2 text-xs">
                          <input type="checkbox" checked={pricingForm.enabled} onChange={(e) => setPricingForm({ ...pricingForm, enabled: e.target.checked })} className="rounded" />
                          启用
                        </label>
                      </div>
                      <div className="flex justify-end gap-2 pt-2">
                        <button onClick={() => setShowPricingForm(false)} className="btn-secondary px-3 py-1.5 text-xs">取消</button>
                        <button onClick={handleSavePricing} disabled={upsertPricing.isPending || !pricingForm.modelName.trim()} className="btn-primary px-3 py-1.5 text-xs disabled:opacity-50">
                          {upsertPricing.isPending ? "保存中..." : "保存"}
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>

          </TabsContent>

          <TabsContent value="upstreams">
            <TenantUpstreamsTab tenant={tenant} />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Recharge Modal
// ---------------------------------------------------------------------------

function RechargeModal({
  tenant,
  onClose,
}: {
  tenant: AdminTenant;
  onClose: () => void;
}) {
  const [amount, setAmount] = useState("");
  const [description, setDescription] = useState("");
  const recharge = useAdminRechargeTenant();
  const backdrop = useBackdropClose(onClose);

  const handleSubmit = () => {
    const val = parseFloat(amount);
    if (!val || val <= 0) return;
    recharge.mutate(
      { tenantId: tenant.id, amount: val, description: description.trim() || undefined },
      { onSuccess: () => onClose() },
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true" {...backdrop}>
      <div className="bg-card border border-border rounded-xl shadow-modal p-6 w-full max-w-sm mx-4 slide-up" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center gap-2 mb-4">
          <DollarSign size={16} className="text-emerald-600" />
          <h3 className="text-base font-semibold text-foreground">充值 - {tenant.name}</h3>
        </div>
        <div className="space-y-3">
          <div>
            <label className="block text-xs font-medium mb-1">充值金额 (CNY)</label>
            <input
              type="number"
              step="0.01"
              min="0"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              className="input-field"
              placeholder="输入金额"
              autoFocus
            />
          </div>
          <div>
            <label className="block text-xs font-medium mb-1">备注 (可选)</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="input-field"
              placeholder="充值备注"
            />
          </div>
        </div>
        <div className="flex gap-3 mt-5">
          <button onClick={onClose} className="flex-1 btn-secondary py-2">取消</button>
          <button
            onClick={handleSubmit}
            disabled={recharge.isPending || !amount || parseFloat(amount) <= 0}
            className="flex-1 btn-primary py-2 disabled:opacity-50 flex items-center justify-center gap-1.5"
          >
            {recharge.isPending && <Loader2 size={14} className="animate-spin" />}
            确认充值
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Create Tenant Modal
// ---------------------------------------------------------------------------

function CreateTenantModal({ onClose }: { onClose: () => void }) {
  const [name, setName] = useState("");
  const [ownerSearch, setOwnerSearch] = useState("");
  const [selectedOwner, setSelectedOwner] = useState<UserWithBalance | null>(null);
  const [contactPhone, setContactPhone] = useState("");
  const [contactEmail, setContactEmail] = useState("");
  const createTenant = useAdminCreateTenant();
  const { data: usersData } = useAdminUsers(1, 20, ownerSearch);
  const users = usersData?.users ?? [];
  const backdrop = useBackdropClose(onClose);

  const handleSubmit = () => {
    if (!name.trim() || !selectedOwner) return;
    createTenant.mutate(
      {
        name: name.trim(),
        owner_id: selectedOwner.id,
        contact_phone: contactPhone.trim() || undefined,
        contact_email: contactEmail.trim() || undefined,
      },
      { onSuccess: () => onClose() },
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true" {...backdrop}>
      <div className="bg-card border border-border rounded-xl shadow-modal p-6 w-full max-w-md mx-4 slide-up" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center gap-2 mb-4">
          <Building2 size={16} className="text-primary" />
          <h3 className="text-base font-semibold text-foreground">创建租户</h3>
        </div>
        <div className="space-y-3">
          <div>
            <label className="block text-xs font-medium mb-1">租户名称</label>
            <input type="text" value={name} onChange={(e) => setName(e.target.value)} className="input-field" placeholder="输入租户名称" autoFocus />
          </div>
          <div>
            <label className="block text-xs font-medium mb-1">所有者</label>
            {selectedOwner ? (
              <div className="flex items-center justify-between px-3 py-2 bg-muted/50 rounded-lg border border-border">
                <span className="text-sm">{selectedOwner.phone} {selectedOwner.nickname && `(${selectedOwner.nickname})`}</span>
                <button onClick={() => setSelectedOwner(null)} className="text-xs text-muted-foreground hover:text-foreground">更换</button>
              </div>
            ) : (
              <div className="space-y-1.5">
                <input
                  type="text"
                  value={ownerSearch}
                  onChange={(e) => setOwnerSearch(e.target.value)}
                  className="input-field"
                  placeholder="搜索用户手机号..."
                />
                {ownerSearch && users.length > 0 && (
                  <div className="border border-border rounded-lg max-h-32 overflow-y-auto">
                    {users.map((u) => (
                      <button
                        key={u.id}
                        onClick={() => { setSelectedOwner(u); setOwnerSearch(""); }}
                        className="w-full text-left px-3 py-2 text-sm hover:bg-muted/50 transition-colors"
                      >
                        {u.phone} {u.nickname && <span className="text-muted-foreground">({u.nickname})</span>}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <label className="block text-xs font-medium mb-1">联系电话</label>
              <input type="text" value={contactPhone} onChange={(e) => setContactPhone(e.target.value)} className="input-field" placeholder="可选" />
            </div>
            <div>
              <label className="block text-xs font-medium mb-1">联系邮箱</label>
              <input type="text" value={contactEmail} onChange={(e) => setContactEmail(e.target.value)} className="input-field" placeholder="可选" />
            </div>
          </div>
        </div>
        <div className="flex gap-3 mt-5">
          <button onClick={onClose} className="flex-1 btn-secondary py-2">取消</button>
          <button
            onClick={handleSubmit}
            disabled={createTenant.isPending || !name.trim() || !selectedOwner}
            className="flex-1 btn-primary py-2 disabled:opacity-50 flex items-center justify-center gap-1.5"
          >
            {createTenant.isPending && <Loader2 size={14} className="animate-spin" />}
            创建
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main Page
// ---------------------------------------------------------------------------

const AdminTenants = () => {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const { data, isLoading } = useAdminTenants(page, PAGE_SIZE);
  const deleteTenant = useAdminDeleteTenant();

  const [detailTenant, setDetailTenant] = useState<AdminTenant | null>(null);
  const [rechargeTenant, setRechargeTenant] = useState<AdminTenant | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  const tenants = (data?.tenants ?? []).filter(
    (t) => !search || t.name.toLowerCase().includes(search.toLowerCase()),
  );
  const total = search ? tenants.length : (data?.total ?? 0);
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const handleDelete = (t: AdminTenant) => {
    if (!confirm(`确定要删除租户「${t.name}」吗？此操作不可恢复。`)) return;
    deleteTenant.mutate(t.id);
  };

  return (
    <div className="page-container fade-in">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2.5">
          <Building2 size={20} className="text-primary" />
          <h1 className="text-lg font-bold text-foreground">租户管理</h1>
          <span className="text-xs text-muted-foreground">({total})</span>
        </div>
        <button onClick={() => setShowCreate(true)} className="btn-primary px-4 py-2 text-sm flex items-center gap-1.5">
          <Plus size={14} />
          创建租户
        </button>
      </div>

      {/* Search */}
      <div className="relative mb-5">
        <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="input-field pl-9"
          placeholder="搜索租户名称..."
        />
      </div>

      {/* Table */}
      {isLoading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 size={20} className="animate-spin text-primary" />
        </div>
      ) : tenants.length === 0 ? (
        <div className="text-center py-16 text-sm text-muted-foreground border border-dashed border-border rounded-xl">
          {search ? "未找到匹配的租户" : "暂无租户"}
        </div>
      ) : (
        <>
          <div className="border border-border rounded-xl overflow-hidden shadow-card">
            <Table className="w-full text-sm">
              <TableHeader className="bg-muted/40">
                <TableRow>
                  <TableHead className="text-left px-4 py-3 text-xs font-medium text-muted-foreground">名称</TableHead>
                  <TableHead className="text-left px-4 py-3 text-xs font-medium text-muted-foreground">ID</TableHead>
                  <TableHead className="text-left px-4 py-3 text-xs font-medium text-muted-foreground">状态</TableHead>
                  <TableHead className="text-right px-4 py-3 text-xs font-medium text-muted-foreground">创建时间</TableHead>
                  <TableHead className="text-right px-4 py-3 text-xs font-medium text-muted-foreground">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tenants.map((t) => (
                  <TableRow key={t.id} className="border-t border-border/50 hover:bg-muted/20 transition-colors">
                    <TableCell className="px-4 py-3">
                      <div className="font-medium text-foreground">{t.name}</div>
                    </TableCell>
                    <TableCell className="px-4 py-3 text-xs text-muted-foreground font-mono">{t.id.slice(0, 8)}...</TableCell>
                    <TableCell className="px-4 py-3">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ${
                        t.status === "active"
                          ? "bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-400"
                          : "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400"
                      }`}>
                        {t.status === "active" ? "活跃" : t.status}
                      </span>
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right text-xs text-muted-foreground whitespace-nowrap">
                      {new Date(t.created_at).toLocaleDateString("zh-CN")}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button onClick={() => setDetailTenant(t)} className="p-1.5 rounded-lg hover:bg-muted transition-colors" title="查看详情">
                          <Eye size={14} className="text-muted-foreground" />
                        </button>
                        <button onClick={() => setRechargeTenant(t)} className="p-1.5 rounded-lg hover:bg-muted transition-colors" title="充值">
                          <DollarSign size={14} className="text-emerald-600" />
                        </button>
                        <button onClick={() => handleDelete(t)} className="p-1.5 rounded-lg hover:bg-muted transition-colors" title="删除">
                          <Trash2 size={14} className="text-destructive" />
                        </button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-3 mt-4">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="p-2 rounded-lg hover:bg-muted disabled:opacity-40 transition-colors"
              >
                <ChevronLeft size={16} />
              </button>
              <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
                className="p-2 rounded-lg hover:bg-muted disabled:opacity-40 transition-colors"
              >
                <ChevronRight size={16} />
              </button>
            </div>
          )}
        </>
      )}

      {/* Modals */}
      {detailTenant && <TenantDetailModal tenant={detailTenant} onClose={() => setDetailTenant(null)} />}
      {rechargeTenant && <RechargeModal tenant={rechargeTenant} onClose={() => setRechargeTenant(null)} />}
      {showCreate && <CreateTenantModal onClose={() => setShowCreate(false)} />}
    </div>
  );
};

export default AdminTenants;
