import { useState } from "react";
import { Plus, Pencil, Info, Loader, Layers, History, Clock } from "lucide-react";
import { useAdminPricing, useUpdateAdminPricing, useAdminPricingChangeLogs } from "@/hooks/use-api";
import type { ModelPricing, PricingTier, TimeBasedPricingRule } from "@/types/api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const emptyTier = (): PricingTier => ({ min_tokens: 0, max_tokens: 0, input_price: 0, output_price: 0, cached_input_price: 0 });
const emptyRule = (): TimeBasedPricingRule => ({ name: "", days: [1, 2, 3, 4, 5], start_time: "09:00", end_time: "18:00", multiplier: 1.0 });

const AdminPricing = () => {
  const [showModal, setShowModal] = useState(false);
  const [showChangeLogs, setShowChangeLogs] = useState(false);
  const [changeLogPage, setChangeLogPage] = useState(1);
  const [editItem, setEditItem] = useState<ModelPricing | null>(null);
  const [form, setForm] = useState({
    modelName: "",
    inputPrice: "",
    outputPrice: "",
    cachedInputPrice: "",
    cacheCreationPrice: "",
    cacheCreation1hPrice: "",
    billingType: "token",
    enabled: true,
    pricingTiers: [] as PricingTier[],
    timeBasedRules: [] as TimeBasedPricingRule[],
  });

  const { data: pricingRes, isLoading } = useAdminPricing();
  const updatePricing = useUpdateAdminPricing();
  const changeLogs = useAdminPricingChangeLogs(changeLogPage, 10);

  const pricingList = pricingRes?.pricing ?? [];

  const openAdd = () => {
    setEditItem(null);
    setForm({ modelName: "", inputPrice: "", outputPrice: "", cachedInputPrice: "", cacheCreationPrice: "", cacheCreation1hPrice: "", billingType: "token", enabled: true, pricingTiers: [], timeBasedRules: [] });
    setShowModal(true);
  };

  const openEdit = (item: ModelPricing) => {
    setEditItem(item);
    setForm({
      modelName: item.model_name,
      inputPrice: String(item.input_price),
      outputPrice: String(item.output_price),
      cachedInputPrice: String(item.cached_input_price || 0),
      cacheCreationPrice: String(item.cache_creation_price || 0),
      cacheCreation1hPrice: String(item.cache_creation_1h_price || 0),
      billingType: item.billing_type || "token",
      enabled: item.is_active,
      pricingTiers: item.pricing_tiers ?? [],
      timeBasedRules: item.time_based_rules ?? [],
    });
    setShowModal(true);
  };

  const handleToggle = (item: ModelPricing) => {
    updatePricing.mutate({
      model: item.model_name,
      input_price: item.input_price,
      output_price: item.output_price,
      cached_input_price: item.cached_input_price,
      cache_creation_price: item.cache_creation_price,
      cache_creation_1h_price: item.cache_creation_1h_price,
      billing_type: item.billing_type || "token",
      is_active: !item.is_active,
      pricing_tiers: item.pricing_tiers,
      time_based_rules: item.time_based_rules,
    });
  };

  const handleSave = async () => {
    await updatePricing.mutateAsync({
      model: form.modelName,
      input_price: parseFloat(form.inputPrice),
      output_price: parseFloat(form.outputPrice),
      cached_input_price: parseFloat(form.cachedInputPrice) || 0,
      cache_creation_price: parseFloat(form.cacheCreationPrice) || 0,
      cache_creation_1h_price: parseFloat(form.cacheCreation1hPrice) || 0,
      billing_type: form.billingType,
      is_active: form.enabled,
      pricing_tiers: form.pricingTiers.length > 0 ? form.pricingTiers : undefined,
      time_based_rules: form.timeBasedRules.length > 0 ? form.timeBasedRules : undefined,
    });
    setShowModal(false);
    setEditItem(null);
  };

  return (
    <div className="page-container">
      {/* Page header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-lg font-bold text-foreground">定价管理</h1>
          <p className="text-sm text-muted-foreground mt-0.5">配置各模型的计费价格（人民币）</p>
        </div>
      </div>

      {/* Info banner */}
      <div className="flex items-start gap-3 bg-primary/5 border border-primary/20 rounded-xl px-4 py-3 mb-5">
        <Info size={15} className="text-primary mt-0.5 flex-shrink-0" />
        <p className="text-sm text-primary/80">
          价格单位为 <strong>¥/百万 tokens</strong>，直接用于计费扣费；前台展示的 ¥ 与扣费一致。
        </p>
      </div>

      {/* Table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-foreground">价格列表</span>
            <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
              {isLoading ? "..." : `${pricingList.length} 条规则`}
            </span>
          </div>
          <button onClick={openAdd} className="btn-primary flex items-center gap-1.5 h-8 text-xs">
            <Plus size={13} />
            添加定价
          </button>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3.5">模型名</TableHead>
                <TableHead className="text-left px-5 py-3.5">输入 (¥/M)</TableHead>
                <TableHead className="text-left px-5 py-3.5">输出 (¥/M)</TableHead>
                <TableHead className="text-left px-5 py-3.5">缓存读 (¥/M)</TableHead>
                <TableHead className="text-left px-5 py-3.5">缓存写5m (¥/M)</TableHead>
                <TableHead className="text-left px-5 py-3.5">缓存写1h (¥/M)</TableHead>
                <TableHead className="text-left px-5 py-3.5">启用状态</TableHead>
                <TableHead className="text-left px-5 py-3.5">更新时间</TableHead>
                <TableHead className="text-right px-5 py-3.5">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pricingList.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9} className="px-5 py-20">
                    <div className="empty-state">
                      <div className="w-14 h-14 bg-muted rounded-2xl flex items-center justify-center mb-4">
                        <Info size={22} className="text-muted-foreground/40" />
                      </div>
                      <div className="text-sm font-medium text-muted-foreground mb-1">暂无定价规则</div>
                      <div className="text-xs text-muted-foreground/60 mb-4">添加模型定价后，将对用户的 API 调用进行计费</div>
                      <button onClick={openAdd} className="btn-primary flex items-center gap-2">
                        <Plus size={13} />
                        添加定价
                      </button>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                pricingList.map((item, i) => (
                  <TableRow
                    key={item.id}
                    className={`border-t border-border hover:bg-accent/30 transition-colors duration-150 ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                  >
                    <TableCell className="px-5 py-3.5 text-sm font-mono text-foreground">
                      {item.model_name}
                      {(item.billing_type === "image") && (
                        <span className="ml-2 px-1.5 py-0.5 rounded text-[10px] font-medium bg-pink-100 text-pink-700 dark:bg-pink-500/10 dark:text-pink-400">按张</span>
                      )}
                      {(item.pricing_tiers && item.pricing_tiers.length > 0) && (
                        <span className="ml-2 px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-100 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400">分梯次</span>
                      )}
                      {(item.time_based_rules && item.time_based_rules.length > 0) && (
                        <span className="ml-2 px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400">时段倍率</span>
                      )}
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className="text-sm font-medium text-foreground">¥{item.input_price.toFixed(4)}</span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className="text-sm font-medium text-foreground">¥{item.output_price.toFixed(4)}</span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className="text-sm font-medium text-muted-foreground">¥{(item.cached_input_price || 0).toFixed(4)}</span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className="text-sm font-medium text-muted-foreground">¥{(item.cache_creation_price || 0).toFixed(4)}</span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <span className="text-sm font-medium text-muted-foreground">¥{(item.cache_creation_1h_price || 0).toFixed(4)}</span>
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <button
                        onClick={() => handleToggle(item)}
                        className={`relative rounded-full transition-colors duration-200 ${item.is_active ? "bg-primary" : "bg-border"}`}
                        style={{ height: 22, width: 40 }}
                      >
                        <span
                          className="absolute bg-white rounded-full shadow transition-transform duration-200"
                          style={{
                            width: 18,
                            height: 18,
                            top: 2,
                            left: 0,
                            transform: item.is_active ? "translateX(20px)" : "translateX(2px)",
                          }}
                        />
                      </button>
                    </TableCell>
                    <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                      {new Date(item.updated_at).toLocaleString("zh-CN")}
                    </TableCell>
                    <TableCell className="px-5 py-3.5">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => openEdit(item)}
                          className="flex items-center gap-1 text-xs text-primary hover:text-primary/80 px-2.5 py-1.5 rounded-lg hover:bg-primary/8 transition-colors"
                        >
                          <Pencil size={11} />
                          编辑
                        </button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        )}
      </div>

      {/* Change log section */}
      <div className="bg-card border border-border rounded-xl shadow-card mt-6">
        <button
          onClick={() => setShowChangeLogs(!showChangeLogs)}
          className="w-full flex items-center justify-between px-5 py-4 text-left hover:bg-accent/30 transition-colors rounded-xl"
        >
          <div className="flex items-center gap-2">
            <History size={16} className="text-muted-foreground" />
            <span className="text-sm font-medium text-foreground">变更记录</span>
            {changeLogs.data && (
              <span className="text-xs text-muted-foreground">({changeLogs.data.total} 条)</span>
            )}
          </div>
          <span className="text-xs text-muted-foreground">{showChangeLogs ? "收起" : "展开"}</span>
        </button>
        {showChangeLogs && (
          <div className="border-t border-border">
            {changeLogs.isLoading ? (
              <div className="flex items-center justify-center py-10">
                <Loader size={16} className="animate-spin text-muted-foreground" />
              </div>
            ) : !changeLogs.data?.logs.length ? (
              <div className="text-center py-10 text-sm text-muted-foreground">暂无变更记录</div>
            ) : (
              <>
                <Table className="w-full">
                  <TableHeader>
                    <TableRow className="table-header">
                      <TableHead className="text-left px-5 py-3">时间</TableHead>
                      <TableHead className="text-left px-5 py-3">类型</TableHead>
                      <TableHead className="text-left px-5 py-3">模型</TableHead>
                      <TableHead className="text-left px-5 py-3">变更内容</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {changeLogs.data.logs.map((log) => (
                      <TableRow key={log.id} className="border-t border-border">
                        <TableCell className="px-5 py-2.5 text-xs text-muted-foreground whitespace-nowrap">
                          {new Date(log.created_at).toLocaleString("zh-CN")}
                        </TableCell>
                        <TableCell className="px-5 py-2.5">
                          <span className={`text-xs px-2 py-0.5 rounded font-medium ${
                            log.change_type === "fx_rate_change"
                              ? "bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400"
                              : "bg-blue-100 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400"
                          }`}>
                            {log.change_type === "fx_rate_change" ? "汇率变更(旧)" : "定价更新"}
                          </span>
                        </TableCell>
                        <TableCell className="px-5 py-2.5 text-xs font-mono text-foreground">
                          {log.model_name === "_fx_rate" ? "—" : log.model_name}
                        </TableCell>
                        <TableCell className="px-5 py-2.5 text-xs text-muted-foreground">
                          <span>
                            输入 ¥{String(log.old_values?.input_price ?? "—")} → ¥{String(log.new_values?.input_price ?? "—")}
                            {" / "}
                            输出 ¥{String(log.old_values?.output_price ?? "—")} → ¥{String(log.new_values?.output_price ?? "—")}
                          </span>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                {changeLogs.data.total > 10 && (
                  <div className="flex items-center justify-center gap-2 py-3 border-t border-border">
                    <button
                      className="px-3 py-1 rounded text-xs bg-muted text-muted-foreground disabled:opacity-40"
                      disabled={changeLogPage <= 1}
                      onClick={() => setChangeLogPage(changeLogPage - 1)}
                    >
                      上一页
                    </button>
                    <span className="text-xs text-muted-foreground">
                      {changeLogPage} / {Math.ceil(changeLogs.data.total / 10)}
                    </span>
                    <button
                      className="px-3 py-1 rounded text-xs bg-muted text-muted-foreground disabled:opacity-40"
                      disabled={changeLogPage >= Math.ceil(changeLogs.data.total / 10)}
                      onClick={() => setChangeLogPage(changeLogPage + 1)}
                    >
                      下一页
                    </button>
                  </div>
                )}
              </>
            )}
          </div>
        )}
      </div>

      {/* Add/Edit modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-[480px] overflow-y-auto rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <h3 className="text-base font-bold text-foreground mb-1">{editItem ? "编辑定价" : "添加定价"}</h3>
            <p className="text-sm text-muted-foreground mb-5">
              {form.billingType === "image"
                ? "设置图像模型的按张计费价格（人民币）"
                : "设置模型的 Token 计费价格（人民币）"}
            </p>

            <div className="flex flex-col gap-4 mb-5">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">模型名称</label>
                <input
                  className="input-field"
                  placeholder="例如：pa/gpt-4o"
                  value={form.modelName}
                  disabled={!!editItem}
                  onChange={(e) => setForm({ ...form, modelName: e.target.value })}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">计费类型</label>
                <div className="flex gap-2">
                  <button
                    className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                      form.billingType === "token"
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted text-muted-foreground hover:bg-muted/80"
                    }`}
                    onClick={() => setForm({ ...form, billingType: "token" })}
                  >
                    Token 计费
                  </button>
                  <button
                    className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                      form.billingType === "image"
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted text-muted-foreground hover:bg-muted/80"
                    }`}
                    onClick={() => setForm({ ...form, billingType: "image" })}
                  >
                    图片计费（按张）
                  </button>
                </div>
              </div>
              {form.billingType === "image" ? (
                <div className="flex gap-3">
                  <div className="flex-1">
                    <label className="block text-sm font-medium text-foreground mb-1.5">1K/2K 价格 (¥/张)</label>
                    <input
                      className="input-field"
                      placeholder="例如：0.9715"
                      type="number"
                      step="any"
                      value={form.inputPrice}
                      onChange={(e) => setForm({ ...form, inputPrice: e.target.value })}
                    />
                  </div>
                  <div className="flex-1">
                    <label className="block text-sm font-medium text-foreground mb-1.5">4K 价格 (¥/张)</label>
                    <input
                      className="input-field"
                      placeholder="例如：1.74"
                      type="number"
                      step="any"
                      value={form.outputPrice}
                      onChange={(e) => setForm({ ...form, outputPrice: e.target.value })}
                    />
                  </div>
                </div>
              ) : (
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="block text-sm font-medium text-foreground mb-1.5">输入价格 (¥/百万tokens)</label>
                  <input
                    className="input-field"
                    placeholder="例如：14.50"
                    type="number"
                    step="any"
                    value={form.inputPrice}
                    onChange={(e) => setForm({ ...form, inputPrice: e.target.value })}
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-sm font-medium text-foreground mb-1.5">输出价格 (¥/百万tokens)</label>
                  <input
                    className="input-field"
                    placeholder="例如：87.00"
                    type="number"
                    step="any"
                    value={form.outputPrice}
                    onChange={(e) => setForm({ ...form, outputPrice: e.target.value })}
                  />
                </div>
              </div>
              )}
              {form.billingType !== "image" && (
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="block text-sm font-medium text-foreground mb-1.5">缓存读取 (¥/百万tokens)</label>
                  <input
                    className="input-field"
                    placeholder="例如：2.18"
                    type="number"
                    step="any"
                    value={form.cachedInputPrice}
                    onChange={(e) => setForm({ ...form, cachedInputPrice: e.target.value })}
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-sm font-medium text-foreground mb-1.5">缓存写入 5min (¥/百万tokens)</label>
                  <input
                    className="input-field"
                    placeholder="例如：27.19"
                    type="number"
                    step="any"
                    value={form.cacheCreationPrice}
                    onChange={(e) => setForm({ ...form, cacheCreationPrice: e.target.value })}
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-sm font-medium text-foreground mb-1.5">缓存写入 1h (¥/百万tokens)</label>
                  <input
                    className="input-field"
                    placeholder="例如：72.50"
                    type="number"
                    step="any"
                    value={form.cacheCreation1hPrice}
                    onChange={(e) => setForm({ ...form, cacheCreation1hPrice: e.target.value })}
                  />
                </div>
              </div>
              )}
              {/* Tiered pricing section */}
              {form.billingType !== "image" && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                    <Layers size={13} />
                    分梯次定价（可选）
                  </label>
                  <button
                    type="button"
                    onClick={() => setForm({ ...form, pricingTiers: [...form.pricingTiers, emptyTier()] })}
                    className="flex items-center gap-1 text-xs text-primary hover:text-primary/80"
                  >
                    <Plus size={11} />
                    添加梯度
                  </button>
                </div>
                {form.pricingTiers.length > 0 && (
                  <div className="flex flex-col gap-2 bg-muted/30 rounded-lg p-3">
                    <div className="grid grid-cols-6 gap-2 text-[10px] text-muted-foreground font-medium">
                      <span>最小tokens</span><span>最大tokens</span><span>输入(¥/M)</span><span>输出(¥/M)</span><span>缓存读(¥/M)</span><span></span>
                    </div>
                    {form.pricingTiers.map((tier, idx) => (
                      <div key={idx} className="grid grid-cols-6 gap-2">
                        <input className="input-field text-xs" type="number" placeholder="1" value={tier.min_tokens || ""} onChange={(e) => { const t = [...form.pricingTiers]; t[idx] = { ...t[idx], min_tokens: parseInt(e.target.value) || 0 }; setForm({ ...form, pricingTiers: t }); }} />
                        <input className="input-field text-xs" type="number" placeholder="32768" value={tier.max_tokens || ""} onChange={(e) => { const t = [...form.pricingTiers]; t[idx] = { ...t[idx], max_tokens: parseInt(e.target.value) || 0 }; setForm({ ...form, pricingTiers: t }); }} />
                        <input className="input-field text-xs" type="number" step="any" placeholder="6" value={tier.input_price || ""} onChange={(e) => { const t = [...form.pricingTiers]; t[idx] = { ...t[idx], input_price: parseFloat(e.target.value) || 0 }; setForm({ ...form, pricingTiers: t }); }} />
                        <input className="input-field text-xs" type="number" step="any" placeholder="24" value={tier.output_price || ""} onChange={(e) => { const t = [...form.pricingTiers]; t[idx] = { ...t[idx], output_price: parseFloat(e.target.value) || 0 }; setForm({ ...form, pricingTiers: t }); }} />
                        <input className="input-field text-xs" type="number" step="any" placeholder="1.3" value={tier.cached_input_price || ""} onChange={(e) => { const t = [...form.pricingTiers]; t[idx] = { ...t[idx], cached_input_price: parseFloat(e.target.value) || 0 }; setForm({ ...form, pricingTiers: t }); }} />
                        <button type="button" onClick={() => { const t = form.pricingTiers.filter((_, i) => i !== idx); setForm({ ...form, pricingTiers: t }); }} className="text-xs text-red-500 hover:text-red-700">删除</button>
                      </div>
                    ))}
                  </div>
                )}
                {form.pricingTiers.length === 0 && (
                  <p className="text-xs text-muted-foreground">未配置梯度，将使用上方的固定价格</p>
                )}
              </div>
              )}
              {/* Time-based pricing section */}
              {form.billingType !== "image" && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                    <Clock size={13} />
                    时段定价（可选）
                  </label>
                  <button
                    type="button"
                    onClick={() => setForm({ ...form, timeBasedRules: [...form.timeBasedRules, emptyRule()] })}
                    className="flex items-center gap-1 text-xs text-primary hover:text-primary/80"
                  >
                    <Plus size={11} />
                    添加时段
                  </button>
                </div>
                {form.timeBasedRules.length > 0 ? (
                  <div className="flex flex-col gap-3 bg-muted/30 rounded-lg p-3 max-h-60 overflow-y-auto">
                    {form.timeBasedRules.map((rule, idx) => (
                      <div key={idx} className="flex flex-col gap-2 border border-border rounded-lg p-3 bg-card">
                        <input
                          className="input-field text-xs"
                          placeholder="时段名称（如：高峰期）"
                          value={rule.name}
                          onChange={(e) => {
                            const r = [...form.timeBasedRules];
                            r[idx] = { ...r[idx], name: e.target.value };
                            setForm({ ...form, timeBasedRules: r });
                          }}
                        />
                        <div className="flex gap-1 flex-wrap">
                          {["日", "一", "二", "三", "四", "五", "六"].map((day, d) => (
                            <button
                              key={d}
                              type="button"
                              onClick={() => {
                                const r = [...form.timeBasedRules];
                                const days = r[idx].days.includes(d)
                                  ? r[idx].days.filter(v => v !== d)
                                  : [...r[idx].days, d].sort((a, b) => a - b);
                                r[idx] = { ...r[idx], days };
                                setForm({ ...form, timeBasedRules: r });
                              }}
                              className={`px-2 py-1 text-xs rounded ${
                                rule.days.includes(d)
                                  ? "bg-primary text-primary-foreground"
                                  : "bg-muted text-muted-foreground"
                              }`}
                            >
                              周{day}
                            </button>
                          ))}
                        </div>
                        <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                          <input
                            type="time"
                            className="input-field text-xs"
                            value={rule.start_time}
                            onChange={(e) => {
                              const r = [...form.timeBasedRules];
                              r[idx] = { ...r[idx], start_time: e.target.value };
                              setForm({ ...form, timeBasedRules: r });
                            }}
                          />
                          <input
                            type="time"
                            className="input-field text-xs"
                            value={rule.end_time}
                            onChange={(e) => {
                              const r = [...form.timeBasedRules];
                              r[idx] = { ...r[idx], end_time: e.target.value };
                              setForm({ ...form, timeBasedRules: r });
                            }}
                          />
                          <input
                            type="number"
                            step="0.01"
                            min="0.01"
                            className="input-field text-xs"
                            placeholder="倍率"
                            value={rule.multiplier || ""}
                            onChange={(e) => {
                              const r = [...form.timeBasedRules];
                              r[idx] = { ...r[idx], multiplier: parseFloat(e.target.value) || 1.0 };
                              setForm({ ...form, timeBasedRules: r });
                            }}
                          />
                        </div>
                        <button
                          type="button"
                          onClick={() => {
                            const r = form.timeBasedRules.filter((_, i) => i !== idx);
                            setForm({ ...form, timeBasedRules: r });
                          }}
                          className="text-xs text-red-500 hover:text-red-700 self-end"
                        >
                          删除
                        </button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground">未配置时段，全天使用基础价格</p>
                )}
              </div>
              )}
              <div className="flex items-center gap-3">
                <label className="text-sm font-medium text-foreground">启用状态</label>
                <button
                  onClick={() => setForm({ ...form, enabled: !form.enabled })}
                  className={`relative rounded-full transition-colors duration-200 ${form.enabled ? "bg-primary" : "bg-border"}`}
                  style={{ width: 40, height: 22 }}
                >
                  <span
                    className="absolute bg-white rounded-full shadow transition-transform duration-200"
                    style={{ width: 18, height: 18, top: 2, left: 0, transform: form.enabled ? "translateX(20px)" : "translateX(2px)" }}
                  />
                </button>
                <span className="text-sm text-muted-foreground">{form.enabled ? "已启用" : "已禁用"}</span>
              </div>
            </div>

            <div className="flex gap-3 justify-end">
              <button onClick={() => setShowModal(false)} className="btn-secondary">
                取消
              </button>
              <button
                onClick={handleSave}
                disabled={
                  !form.modelName ||
                  !form.inputPrice ||
                  !form.outputPrice ||
                  updatePricing.isPending
                }
                className={`btn-primary ${(!form.modelName || !form.inputPrice || !form.outputPrice) ? "opacity-50 cursor-not-allowed" : ""}`}
              >
                {updatePricing.isPending ? "保存中..." : editItem ? "保存更改" : "添加"}
              </button>
            </div>
          </div>
        </div>
      )}

    </div>
  );
};

export default AdminPricing;
