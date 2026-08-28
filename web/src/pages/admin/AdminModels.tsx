import { useState, useMemo } from "react";
import { Plus, Pencil, Trash2, Search, ChevronLeft, ChevronRight, Loader, ArrowUp, ArrowDown, Shield, ShieldAlert } from "lucide-react";
import { useGatewayModels, useCreateModel, useUpdateModel, useDeleteModel } from "@/hooks/use-api";
import type { GatewayModel } from "@/types/api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PAGE_SIZE = 10;

interface UpstreamForm {
  provider: string;
  protocol: string;
  protocols: string[];
  upstream_provider: string;
  upstream_name: string;
  base_url: string;
  api_key: string;
  model_override: string;
  weight: number;
}

const emptyUpstream = (): UpstreamForm => ({
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

const PROTOCOL_OPTIONS: { value: string; label: string }[] = [
  { value: "openai", label: "OpenAI (Chat Completions)" },
  { value: "openai-compatible", label: "OpenAI 兼容" },
  { value: "anthropic", label: "Anthropic" },
  { value: "gemini", label: "Gemini" },
  { value: "responses", label: "Responses API" },
];

const AdminModels = () => {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [showAddModal, setShowAddModal] = useState(false);
  const [editModel, setEditModel] = useState<GatewayModel | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null);

  // Form state
  const [formName, setFormName] = useState("");
  const [formDisplayName, setFormDisplayName] = useState("");
  const [formCategory, setFormCategory] = useState("chat");
  const [formUpstreams, setFormUpstreams] = useState<UpstreamForm[]>([emptyUpstream()]);

  const { data: models, isLoading } = useGatewayModels();
  const createModel = useCreateModel();
  const updateModel = useUpdateModel();
  const deleteModel = useDeleteModel();

  const modelList = useMemo(() => models ?? [], [models]);

  const filtered = useMemo(() => {
    const s = search.toLowerCase();
    if (!s) return modelList;
    return modelList.filter(
      (m) =>
        m.name.toLowerCase().includes(s) ||
        m.upstreams.some((u) => u.provider.toLowerCase().includes(s))
    );
  }, [modelList, search]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  const providerColor: Record<string, string> = {
    openai: "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20",
    anthropic: "bg-orange-50 text-orange-700 border-orange-200 dark:bg-orange-500/10 dark:text-orange-400 dark:border-orange-500/20",
    deepseek: "bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-500/10 dark:text-blue-400 dark:border-blue-500/20",
    volcengine: "bg-violet-50 text-violet-700 border-violet-200 dark:bg-violet-500/10 dark:text-violet-400 dark:border-violet-500/20",
    google: "bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20",
    alibaba: "bg-cyan-50 text-cyan-700 border-cyan-200 dark:bg-cyan-500/10 dark:text-cyan-400 dark:border-cyan-500/20",
    azure: "bg-sky-50 text-sky-700 border-sky-200 dark:bg-sky-500/10 dark:text-sky-400 dark:border-sky-500/20",
    moonshot: "bg-pink-50 text-pink-700 border-pink-200 dark:bg-pink-500/10 dark:text-pink-400 dark:border-pink-500/20",
  };

  const resetForm = () => {
    setFormName("");
    setFormDisplayName("");
    setFormCategory("chat");
    setFormUpstreams([emptyUpstream()]);
  };

  const openAdd = () => {
    resetForm();
    setEditModel(null);
    setShowAddModal(true);
  };

  const openEdit = (m: GatewayModel) => {
    setFormName(m.name);
    setFormDisplayName(m.display_name || "");
    setFormCategory(m.category || "chat");
    // Load ALL upstreams (sorted by sort_order from backend)
    if (m.upstreams.length > 0) {
      setFormUpstreams(
        m.upstreams.map((u) => ({
          provider: u.provider,
          protocol: u.protocol || "",
          protocols: u.protocols && u.protocols.length > 0
            ? u.protocols
            : u.protocol
            ? [u.protocol]
            : [],
          upstream_provider: u.upstream_provider || "",
          upstream_name: u.upstream_name || "",
          base_url: u.base_url,
          api_key: u.api_key,
          model_override: u.model_override,
          weight: u.weight || 1,
        }))
      );
    } else {
      setFormUpstreams([emptyUpstream()]);
    }
    setEditModel(m);
    setShowAddModal(true);
  };

  const updateUpstream = (index: number, field: keyof UpstreamForm, value: string | number | string[]) => {
    setFormUpstreams((prev) => prev.map((u, i) => (i === index ? { ...u, [field]: value } : u)));
  };

  const addUpstream = () => {
    setFormUpstreams((prev) => [...prev, emptyUpstream()]);
  };

  const removeUpstream = (index: number) => {
    setFormUpstreams((prev) => prev.filter((_, i) => i !== index));
  };

  const moveUpstream = (index: number, direction: "up" | "down") => {
    setFormUpstreams((prev) => {
      const arr = [...prev];
      const target = direction === "up" ? index - 1 : index + 1;
      if (target < 0 || target >= arr.length) return prev;
      [arr[index], arr[target]] = [arr[target], arr[index]];
      return arr;
    });
  };

  const handleSave = async () => {
    const upstreams = formUpstreams.map((u) => ({
      provider: u.provider,
      protocol: u.protocols.length === 1 ? u.protocols[0] : u.protocol,
      protocols: u.protocols,
      upstream_provider: u.upstream_provider,
      upstream_name: u.upstream_name,
      base_url: u.base_url,
      api_key: u.api_key,
      model_override: u.model_override,
      weight: u.weight || 1,
    }));
    if (editModel) {
      await updateModel.mutateAsync({ id: editModel.id, name: formName, display_name: formDisplayName, category: formCategory, upstreams });
    } else {
      await createModel.mutateAsync({ name: formName, display_name: formDisplayName, category: formCategory, upstreams });
    }
    setShowAddModal(false);
    resetForm();
    setEditModel(null);
  };

  const handleDelete = async () => {
    if (deleteConfirm == null) return;
    await deleteModel.mutateAsync(deleteConfirm);
    setDeleteConfirm(null);
  };

  const getUniqueProviders = (m: GatewayModel) => {
    return [...new Set(m.upstreams.map((u) => u.provider))];
  };

  const canSave = formName && formUpstreams.length > 0 && formUpstreams[0].provider && formUpstreams[0].base_url;

  return (
    <div className="page-container">
      {/* Page header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-lg font-bold text-foreground">模型管理</h1>
          <p className="text-sm text-muted-foreground mt-0.5">管理网关代理的 AI 模型和上游配置（主备模式）</p>
        </div>
      </div>

      {/* Table card */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-sm font-semibold text-foreground">模型列表</span>
            <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
              {isLoading ? "..." : `${filtered.length} 个模型`}
            </span>
          </div>
          <div className="flex items-center gap-2">
            {/* Search */}
            <div className="relative">
              <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
              <input
                className="input-field h-8 text-xs w-52" style={{ paddingLeft: "2rem" }}
                placeholder="搜索模型名称或提供商..."
                value={search}
                onChange={(e) => { setSearch(e.target.value); setPage(1); }}
              />
            </div>
            <button
              onClick={openAdd}
              className="btn-primary flex items-center gap-1.5 h-8 text-xs"
            >
              <Plus size={13} />
              添加模型
            </button>
          </div>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3.5">模型名称</TableHead>
                <TableHead className="text-left px-5 py-3.5">渠道配置</TableHead>
                <TableHead className="text-left px-5 py-3.5">提供商列表</TableHead>
                <TableHead className="text-left px-5 py-3.5">创建时间</TableHead>
                <TableHead className="text-right px-5 py-3.5">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paged.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="px-5 py-16">
                    <div className="empty-state">
                      {modelList.length === 0 ? (
                        <>
                          <div className="w-14 h-14 bg-muted rounded-2xl flex items-center justify-center mb-4">
                            <Plus size={22} className="text-muted-foreground/40" />
                          </div>
                          <div className="text-sm font-medium text-muted-foreground mb-1">尚未配置任何模型</div>
                          <div className="text-xs text-muted-foreground/60 mb-4">添加模型并配置上游渠道后，用户即可通过 API 调用</div>
                          <button onClick={openAdd} className="btn-primary flex items-center gap-2">
                            <Plus size={13} />
                            添加模型
                          </button>
                        </>
                      ) : (
                        <>
                          <div className="w-12 h-12 bg-muted rounded-xl flex items-center justify-center mb-3">
                            <Search size={18} className="text-muted-foreground/50" />
                          </div>
                          <div className="text-sm text-muted-foreground">未找到匹配的模型</div>
                        </>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                paged.map((m, i) => (
                  <TableRow
                    key={m.id}
                    className={`border-t border-border hover:bg-accent/30 transition-colors duration-150 ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                  >
                    <TableCell className="px-5 py-3 text-sm font-mono text-foreground font-medium">{m.name}</TableCell>
                    <TableCell className="px-5 py-3">
                      <div className="flex items-center gap-1.5">
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-primary/10 text-primary text-xs font-medium">
                          <Shield size={10} />
                          主 1
                        </span>
                        {m.upstreams.length > 1 && (
                          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-muted text-muted-foreground text-xs font-medium">
                            <ShieldAlert size={10} />
                            备 {m.upstreams.length - 1}
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="px-5 py-3">
                      <div className="flex items-center gap-1.5 flex-wrap">
                        {getUniqueProviders(m).map((p) => (
                          <span
                            key={p}
                            className={`inline-flex items-center px-2 py-0.5 rounded-md border text-xs font-medium ${providerColor[p] || "bg-muted text-muted-foreground border-border"}`}
                          >
                            {p}
                          </span>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="px-5 py-3 text-sm text-muted-foreground">
                      {new Date(m.created_at).toLocaleString("zh-CN")}
                    </TableCell>
                    <TableCell className="px-5 py-3">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => openEdit(m)}
                          className="flex items-center gap-1 text-xs text-primary hover:text-primary/80 px-2.5 py-1.5 rounded-lg hover:bg-primary/8 transition-colors"
                        >
                          <Pencil size={11} />
                          编辑
                        </button>
                        <button
                          onClick={() => setDeleteConfirm(m.id)}
                          className="flex items-center gap-1 text-xs text-destructive hover:text-destructive/80 px-2.5 py-1.5 rounded-lg hover:bg-destructive/8 transition-colors"
                        >
                          <Trash2 size={11} />
                          删除
                        </button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        )}

        {/* Pagination */}
        {filtered.length > PAGE_SIZE && (
          <div className="px-5 py-3.5 border-t border-border flex items-center justify-between">
            <div className="text-xs text-muted-foreground">共 {filtered.length} 条，第 {page}/{totalPages} 页</div>
            <div className="flex items-center gap-1">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
                className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${page <= 1 ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"}`}
                aria-label="上一页"
              >
                <ChevronLeft size={13} />
              </button>
              {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => i + 1).map((p) => (
                <button
                  key={p}
                  onClick={() => setPage(p)}
                  className={`flex h-9 w-9 items-center justify-center rounded-lg text-xs transition-colors ${page === p ? "bg-primary text-primary-foreground font-medium" : "text-foreground hover:bg-muted cursor-pointer"}`}
                >
                  {p}
                </button>
              ))}
              <button
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
                className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${page >= totalPages ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"}`}
                aria-label="下一页"
              >
                <ChevronRight size={13} />
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Add/Edit model modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-[580px] overflow-y-auto rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <h3 className="text-base font-bold text-foreground mb-1">
              {editModel ? "编辑模型" : "添加模型"}
            </h3>
            <p className="text-sm text-muted-foreground mb-5">配置模型上游渠道，第一个为主渠道，其余为备选渠道</p>

            <div className="flex flex-col gap-4 mb-5 overflow-y-auto flex-1 pr-1">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">模型名称</label>
                <input
                  className="input-field"
                  placeholder="例如：claude-sonnet-4-6"
                  value={formName}
                  onChange={(e) => setFormName(e.target.value)}
                />
              </div>

              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1.5">显示名称</label>
                  <input
                    className="input-field"
                    placeholder="例如：gw/claude-sonnet-4-6"
                    value={formDisplayName}
                    onChange={(e) => setFormDisplayName(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground mt-1">用于 Cursor 等客户端展示，避免与内置模型冲突</p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1.5">分类</label>
                  <select
                    className="input-field"
                    value={formCategory}
                    onChange={(e) => setFormCategory(e.target.value)}
                  >
                    <option value="chat">chat</option>
                    <option value="text">text</option>
                    <option value="multimodal">multimodal</option>
                    <option value="reasoning">reasoning</option>
                    <option value="multimodal,reasoning">multimodal,reasoning</option>
                    <option value="embedding">embedding</option>
                    <option value="text-to-image">text-to-image</option>
                    <option value="image-edit">image-edit</option>
                  </select>
                </div>
              </div>

              {/* Upstream list */}
              {formUpstreams.map((upstream, idx) => (
                <div key={idx} className={`border rounded-lg p-4 ${idx === 0 ? "border-primary/40 bg-primary/3" : "border-border"}`}>
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
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
                    </div>
                    <div className="flex items-center gap-1">
                      {idx > 0 && (
                        <button
                          onClick={() => moveUpstream(idx, "up")}
                          className="flex h-9 w-9 items-center justify-center rounded hover:bg-muted transition-colors text-muted-foreground hover:text-foreground"
                          title="上移"
                        >
                          <ArrowUp size={12} />
                        </button>
                      )}
                      {idx < formUpstreams.length - 1 && (
                        <button
                          onClick={() => moveUpstream(idx, "down")}
                          className="flex h-9 w-9 items-center justify-center rounded hover:bg-muted transition-colors text-muted-foreground hover:text-foreground"
                          title="下移"
                        >
                          <ArrowDown size={12} />
                        </button>
                      )}
                      {formUpstreams.length > 1 && (
                        <button
                          onClick={() => removeUpstream(idx)}
                          className="flex h-9 w-9 items-center justify-center rounded hover:bg-destructive/10 transition-colors text-muted-foreground hover:text-destructive"
                          title="删除"
                        >
                          <Trash2 size={12} />
                        </button>
                      )}
                    </div>
                  </div>
                  <div className="flex flex-col gap-3">
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
                          {PROTOCOL_OPTIONS.map((opt) => {
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
                          placeholder="例如：openai"
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
                          placeholder="例如：openai-us"
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
                </div>
              ))}

              {/* Add backup button */}
              <button
                onClick={addUpstream}
                className="flex items-center justify-center gap-1.5 w-full py-2.5 rounded-lg border border-dashed border-border text-sm text-muted-foreground hover:text-foreground hover:border-primary/40 hover:bg-primary/3 transition-colors"
              >
                <Plus size={14} />
                添加备选渠道
              </button>

              {/* Info hint */}
              {formUpstreams.length > 1 && (
                <div className="text-xs text-muted-foreground bg-muted/50 rounded-lg px-3 py-2">
                  所有请求优先走主渠道。当主渠道连续失败触发熔断后，自动切换到备选渠道。主渠道恢复后流量自动回流。
                </div>
              )}
            </div>

            <div className="flex gap-3 justify-end pt-2 border-t border-border mt-auto">
              <button onClick={() => { setShowAddModal(false); setEditModel(null); }} className="btn-secondary">取消</button>
              <button
                onClick={handleSave}
                disabled={!canSave || createModel.isPending || updateModel.isPending}
                className={`btn-primary ${!canSave ? "opacity-50 cursor-not-allowed" : ""}`}
              >
                {(createModel.isPending || updateModel.isPending) ? "保存中..." : editModel ? "保存更改" : "添加"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirm */}
      {deleteConfirm != null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="w-[calc(100vw-2rem)] max-w-[380px] rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <h3 className="text-base font-bold text-foreground mb-2">确认删除</h3>
            <p className="text-sm text-muted-foreground mb-5">此操作不可撤销，确认删除该模型配置？</p>
            <div className="flex gap-3 justify-end">
              <button onClick={() => setDeleteConfirm(null)} className="btn-secondary">取消</button>
              <button
                onClick={handleDelete}
                disabled={deleteModel.isPending}
                className="flex items-center gap-1.5 bg-destructive text-destructive-foreground rounded-lg px-4 py-2 text-sm font-medium hover:opacity-90 transition-opacity cursor-pointer"
              >
                <Trash2 size={13} />
                {deleteModel.isPending ? "删除中..." : "确认删除"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminModels;
