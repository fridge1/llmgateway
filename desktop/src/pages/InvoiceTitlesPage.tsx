import { useState, useEffect } from "react";
import {
  useInvoiceTitles, useCreateInvoiceTitle, useUpdateInvoiceTitle,
  useDeleteInvoiceTitle, useSetDefaultInvoiceTitle, useCompanySearch,
} from "@/hooks/use-api";
import type { CreateInvoiceTitleRequest } from "@/lib/types-api";
import { Loader2 } from "../components/icons";

const emptyForm: CreateInvoiceTitleRequest = {
  type: "personal",
  title_name: "",
  tax_number: "",
  bank_name: "",
  bank_account: "",
  address: "",
  phone: "",
};

export default function InvoiceTitlesPage() {
  const { data: titles = [], isLoading } = useInvoiceTitles();
  const createTitle = useCreateInvoiceTitle();
  const updateTitle = useUpdateInvoiceTitle();
  const deleteTitle = useDeleteInvoiceTitle();
  const setDefault = useSetDefaultInvoiceTitle();
  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<number | null>(null);
  const [form, setForm] = useState<CreateInvoiceTitleRequest>(emptyForm);
  const [formError, setFormError] = useState("");

  // 300ms debounce for company search
  const [searchInput, setSearchInput] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(searchInput), 300);
    return () => clearTimeout(t);
  }, [searchInput]);
  const { data: companies = [] } = useCompanySearch(debouncedQuery);

  const openCreate = () => {
    setEditId(null);
    setForm(emptyForm);
    setFormError("");
    setSearchInput("");
    setShowForm(true);
  };

  const handleEdit = (title: typeof titles[0]) => {
    setForm({
      type: title.type,
      title_name: title.title_name,
      tax_number: title.tax_number,
      bank_name: title.bank_name,
      bank_account: title.bank_account,
      address: title.address,
      phone: title.phone,
    });
    setEditId(title.id);
    setFormError("");
    setSearchInput(title.type === "enterprise" ? title.title_name : "");
    setShowForm(true);
  };

  const handleTabSwitch = (type: "personal" | "enterprise") => {
    setFormError("");
    if (type === "personal") {
      setForm({ ...emptyForm, type: "personal", title_name: form.title_name });
      setSearchInput("");
    } else {
      setForm({ ...emptyForm, type: "enterprise" });
      setSearchInput("");
    }
  };

  const handleSubmit = async () => {
    if (!form.title_name.trim()) return;
    if (form.type === "enterprise") {
      const missing: string[] = [];
      if (!form.bank_name?.trim()) missing.push("开户银行");
      if (!form.bank_account?.trim()) missing.push("银行账号");
      if (!form.address?.trim()) missing.push("企业地址");
      if (!form.phone?.trim()) missing.push("企业电话");
      if (missing.length > 0) {
        setFormError(`请填写：${missing.join("、")}（开具增值税专用发票时必填）`);
        return;
      }
    }
    setFormError("");
    try {
      if (editId) {
        await updateTitle.mutateAsync({ ...form, id: editId });
      } else {
        await createTitle.mutateAsync(form);
      }
      setShowForm(false);
      setEditId(null);
      setForm(emptyForm);
      setSearchInput("");
    } catch {}
  };

  const handleDelete = (id: number, name: string) => {
    if (window.confirm(`确定删除发票抬头「${name}」吗？`)) {
      deleteTitle.mutate(id);
    }
  };

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}>
        <Loader2 size={24} className="animate-spin text-amber-400" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-5">
        <div>
          <h1 className="text-lg font-semibold text-obsidian-50">发票抬头</h1>
          <p className="text-xs text-obsidian-400 mt-0.5">管理您的发票抬头信息</p>
        </div>
        <button onClick={openCreate} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200">
          添加抬头
        </button>
      </div>

      {titles.length === 0 ? (
        <div className="py-16 text-center text-sm text-obsidian-500">暂无发票抬头，请添加</div>
      ) : (
        <div className="space-y-3">
          {titles.map((t) => (
            <div key={t.id} className={`bg-obsidian-900 border rounded-xl p-4 ${t.is_default ? "border-amber-500/30" : "border-obsidian-700"}`}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-sm font-semibold text-obsidian-50">{t.title_name}</span>
                    <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${t.type === "personal" ? "bg-blue-900/40 text-blue-300" : "bg-amber-900/40 text-amber-300"}`}>
                      {t.type === "personal" ? "个人" : "企业"}
                    </span>
                    {t.is_default && <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-emerald-500/10 text-emerald-400">默认</span>}
                  </div>
                  {t.tax_number && <div className="text-xs text-obsidian-400">税号: {t.tax_number}</div>}
                  {t.bank_name && <div className="text-xs text-obsidian-400">开户行: {t.bank_name}</div>}
                </div>
                <div className="flex items-center gap-1">
                  {!t.is_default && (
                    <button onClick={() => setDefault.mutate(t.id)} className="px-2 py-1 text-xs text-obsidian-400 hover:text-amber-400 transition-colors">设为默认</button>
                  )}
                  <button onClick={() => handleEdit(t)} className="px-2 py-1 text-xs text-obsidian-400 hover:text-obsidian-200 transition-colors">编辑</button>
                  <button onClick={() => handleDelete(t.id, t.title_name)} className="px-2 py-1 text-xs text-obsidian-400 hover:text-red-400 transition-colors">删除</button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[480px] p-6 max-h-[85vh] overflow-y-auto">
            <h3 className="text-base font-semibold text-obsidian-50 mb-4">{editId ? "编辑" : "添加"}发票抬头</h3>

            <div className="space-y-3">
              {/* Type switcher — clears fields on switch */}
              <div>
                <label className="block text-xs font-medium text-obsidian-300 mb-1">类型</label>
                <div className="flex gap-2">
                  {(["personal", "enterprise"] as const).map((t) => (
                    <button
                      key={t}
                      onClick={() => handleTabSwitch(t)}
                      className={`flex-1 py-2 rounded-lg text-xs font-medium transition-all ${form.type === t ? "bg-amber-500 text-obsidian-950" : "bg-obsidian-800 text-obsidian-300"}`}
                    >
                      {t === "personal" ? "个人" : "企业"}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-obsidian-300 mb-1">
                  {form.type === "personal" ? "姓名" : "企业名称"}
                </label>
                {form.type === "enterprise" ? (
                  <div className="relative">
                    <input
                      className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50"
                      placeholder="输入企业名称搜索"
                      value={searchInput}
                      onChange={(e) => {
                        setSearchInput(e.target.value);
                        setForm({ ...form, title_name: e.target.value });
                      }}
                    />
                    {companies.length > 0 && debouncedQuery.length >= 2 && (
                      <div className="absolute left-0 right-0 top-full mt-1 bg-obsidian-800 border border-obsidian-700 rounded-lg max-h-40 overflow-y-auto z-10">
                        {companies.map((c, i) => (
                          <button
                            key={i}
                            className="w-full text-left px-3 py-2 text-xs text-obsidian-200 hover:bg-obsidian-700 transition-colors"
                            onClick={() => {
                              setForm({ ...form, title_name: c.name, tax_number: c.tax_number });
                              setSearchInput(c.name);
                              setDebouncedQuery("");
                            }}
                          >
                            <div>{c.name}</div>
                            <div className="text-obsidian-500">{c.tax_number}</div>
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                ) : (
                  <input
                    className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50"
                    placeholder="请输入姓名"
                    value={form.title_name}
                    onChange={(e) => setForm({ ...form, title_name: e.target.value })}
                  />
                )}
              </div>

              {form.type === "enterprise" && (
                <>
                  <FormField label="税号" value={form.tax_number ?? ""} onChange={(v) => setForm({ ...form, tax_number: v })} />
                  <FormField label="开户银行 *" value={form.bank_name ?? ""} onChange={(v) => setForm({ ...form, bank_name: v })} />
                  <FormField label="银行账号 *" value={form.bank_account ?? ""} onChange={(v) => setForm({ ...form, bank_account: v })} />
                  <FormField label="企业地址 *" value={form.address ?? ""} onChange={(v) => setForm({ ...form, address: v })} />
                  <FormField label="企业电话 *" value={form.phone ?? ""} onChange={(v) => setForm({ ...form, phone: v })} />
                </>
              )}

              {formError && (
                <div className="text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
                  {formError}
                </div>
              )}
            </div>

            <div className="flex gap-3 justify-end mt-5">
              <button onClick={() => { setShowForm(false); setEditId(null); setFormError(""); }} className="px-4 py-2 text-sm text-obsidian-300 hover:text-obsidian-100 transition-colors">取消</button>
              <button
                onClick={handleSubmit}
                disabled={createTitle.isPending || updateTitle.isPending || !form.title_name.trim()}
                className="px-4 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200"
              >
                {editId ? "保存" : "添加"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function FormField({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="block text-xs font-medium text-obsidian-300 mb-1">{label}</label>
      <input
        className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  );
}
