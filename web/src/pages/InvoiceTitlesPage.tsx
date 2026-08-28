import { useState, useEffect } from "react";
import { FileText, Plus, Loader2, Pencil, Trash2, Star } from "lucide-react";
import {
  useInvoiceTitles,
  useCreateInvoiceTitle,
  useUpdateInvoiceTitle,
  useDeleteInvoiceTitle,
  useSetDefaultInvoiceTitle,
  useCompanySearch,
} from "@/hooks/use-api";
import type { InvoiceTitle } from "@/types/api";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { PageHeader } from "@/components/ui/page-header";

interface FormState {
  type: string;
  title_name: string;
  tax_number: string;
  bank_name: string;
  bank_account: string;
  address: string;
  phone: string;
}

const emptyForm: FormState = {
  type: "personal",
  title_name: "",
  tax_number: "",
  bank_name: "",
  bank_account: "",
  address: "",
  phone: "",
};

const InvoiceTitlesPage = () => {
  const { data: titles = [], isLoading } = useInvoiceTitles();
  const createTitle = useCreateInvoiceTitle();
  const updateTitle = useUpdateInvoiceTitle();
  const deleteTitle = useDeleteInvoiceTitle();
  const setDefault = useSetDefaultInvoiceTitle();

  const [showModal, setShowModal] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<FormState>({ ...emptyForm });
  const [formError, setFormError] = useState<string>("");

  // Company search with debounce
  const [searchKeyword, setSearchKeyword] = useState("");
  const [debouncedKeyword, setDebouncedKeyword] = useState("");
  const [showDropdown, setShowDropdown] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedKeyword(searchKeyword), 300);
    return () => clearTimeout(timer);
  }, [searchKeyword]);

  const { data: companies } = useCompanySearch(debouncedKeyword);

  const openCreate = () => {
    setEditingId(null);
    setForm({ ...emptyForm });
    setFormError("");
    setSearchKeyword("");
    setShowDropdown(false);
    setShowModal(true);
  };

  const openEdit = (t: InvoiceTitle) => {
    setEditingId(t.id);
    setForm({
      type: t.type,
      title_name: t.title_name,
      tax_number: t.tax_number,
      bank_name: t.bank_name,
      bank_account: t.bank_account,
      address: t.address,
      phone: t.phone,
    });
    setFormError("");
    setSearchKeyword(t.type === "enterprise" ? t.title_name : "");
    setShowDropdown(false);
    setShowModal(true);
  };

  const closeModal = () => {
    setShowModal(false);
    setEditingId(null);
    setForm({ ...emptyForm });
    setFormError("");
    setSearchKeyword("");
    setShowDropdown(false);
  };

  const handleTabSwitch = (type: string) => {
    setFormError("");
    if (type === "personal") {
      setForm({
        ...emptyForm,
        type: "personal",
        title_name: form.title_name,
      });
      setSearchKeyword("");
      setShowDropdown(false);
    } else {
      setForm({
        ...emptyForm,
        type: "enterprise",
      });
      setSearchKeyword("");
    }
  };

  const handleSave = async () => {
    if (!form.title_name.trim()) return;
    if (form.type === "enterprise") {
      const missing: string[] = [];
      if (!form.bank_name.trim()) missing.push("开户银行");
      if (!form.bank_account.trim()) missing.push("银行账号");
      if (!form.address.trim()) missing.push("企业地址");
      if (!form.phone.trim()) missing.push("企业电话");
      if (missing.length > 0) {
        setFormError(`请填写：${missing.join("、")}（开具增值税专用发票时必填）`);
        return;
      }
    }
    setFormError("");
    try {
      if (editingId !== null) {
        await updateTitle.mutateAsync({
          id: editingId,
          type: form.type,
          title_name: form.title_name.trim(),
          tax_number: form.type === "enterprise" ? form.tax_number.trim() : undefined,
          bank_name: form.type === "enterprise" ? form.bank_name.trim() : undefined,
          bank_account: form.type === "enterprise" ? form.bank_account.trim() : undefined,
          address: form.type === "enterprise" ? form.address.trim() : undefined,
          phone: form.type === "enterprise" ? form.phone.trim() : undefined,
        });
      } else {
        await createTitle.mutateAsync({
          type: form.type,
          title_name: form.title_name.trim(),
          tax_number: form.type === "enterprise" ? form.tax_number.trim() : undefined,
          bank_name: form.type === "enterprise" ? form.bank_name.trim() : undefined,
          bank_account: form.type === "enterprise" ? form.bank_account.trim() : undefined,
          address: form.type === "enterprise" ? form.address.trim() : undefined,
          phone: form.type === "enterprise" ? form.phone.trim() : undefined,
        });
      }
      closeModal();
    } catch {
      // error handled by React Query
    }
  };

  const handleDelete = (t: InvoiceTitle) => {
    if (window.confirm(`确定要删除发票抬头「${t.title_name}」吗？此操作不可恢复。`)) {
      deleteTitle.mutate(t.id);
    }
  };

  const handleSetDefault = (id: number) => {
    setDefault.mutate(id);
  };

  const isSaving = createTitle.isPending || updateTitle.isPending;

  if (isLoading) {
    return (
      <div className="page-container fade-in flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={28} className="animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">加载中...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="page-container fade-in">
      <PageHeader
        eyebrow="发票"
        title="发票抬头"
        description="维护个人与企业发票抬头，申请发票时可直接复用。"
        actions={
          <button onClick={openCreate} className="btn-primary flex items-center gap-2">
            <Plus size={16} />
            新增抬头
          </button>
        }
      />

      {/* Title cards list */}
      {titles.length === 0 ? (
        <div className="bg-card border border-border rounded-xl shadow-card">
          <div className="empty-state py-20">
            <div className="w-14 h-14 bg-primary/8 rounded-2xl flex items-center justify-center mb-4">
              <FileText size={24} className="text-primary/60" />
            </div>
            <div className="text-sm font-semibold text-foreground mb-1">暂无发票抬头</div>
            <div className="text-xs text-muted-foreground mb-5">点击右上角按钮添加您的第一个发票抬头</div>
            <button onClick={openCreate} className="btn-primary flex items-center gap-2">
              <Plus size={14} />
              新增抬头
            </button>
          </div>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {titles.map((t) => (
            <div
              key={t.id}
              className="bg-card border border-border rounded-xl shadow-card px-5 py-4 flex items-center justify-between"
            >
              <div className="flex flex-col gap-1">
                <div className="flex items-center gap-2">
                  {t.type === "personal" ? (
                    <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-600">
                      个人
                    </span>
                  ) : (
                    <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-amber-50 text-amber-600">
                      企业
                    </span>
                  )}
                  {t.is_default && (
                    <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-600">
                      默认
                    </span>
                  )}
                  <span className="text-sm font-medium text-foreground">{t.title_name}</span>
                </div>
                {t.type === "enterprise" && t.tax_number && (
                  <div className="text-xs text-muted-foreground ml-0.5">税号：{t.tax_number}</div>
                )}
              </div>
              <div className="flex items-center gap-1">
                {!t.is_default && (
                  <button
                    onClick={() => handleSetDefault(t.id)}
                    className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs text-muted-foreground hover:text-amber-500 hover:bg-amber-50 transition-colors"
                  >
                    <Star size={13} />
                    设为默认
                  </button>
                )}
                <button
                  onClick={() => openEdit(t)}
                  className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:text-primary hover:bg-primary/8 transition-colors"
                  aria-label="编辑发票抬头"
                >
                  <Pencil size={14} />
                </button>
                <button
                  onClick={() => handleDelete(t)}
                  className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/8 transition-colors"
                  aria-label="删除发票抬头"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Add/Edit Modal */}
      <Dialog open={showModal} onOpenChange={(o) => { if (!o) closeModal(); }}>
        <DialogContent className="sm:max-w-[520px] max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editingId !== null ? "编辑发票抬头" : "新增发票抬头"}</DialogTitle>
            <DialogDescription>{editingId !== null ? "修改发票抬头信息" : "填写发票抬头信息"}</DialogDescription>
          </DialogHeader>

            {/* Type tabs */}
            <div className="flex gap-2 mb-5">
              <button
                onClick={() => handleTabSwitch("personal")}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  form.type === "personal"
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-foreground hover:bg-muted/80"
                }`}
              >
                个人
              </button>
              <button
                onClick={() => handleTabSwitch("enterprise")}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  form.type === "enterprise"
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-foreground hover:bg-muted/80"
                }`}
              >
                企业
              </button>
            </div>

            {form.type === "personal" ? (
              <div className="mb-5">
                <label className="block text-sm font-medium text-foreground mb-1.5">抬头名称</label>
                <input
                  className="w-full px-3 py-2 text-sm border border-border rounded-lg bg-background text-foreground outline-none focus:border-primary transition-colors"
                  placeholder="请输入个人姓名"
                  value={form.title_name}
                  onChange={(e) => setForm({ ...form, title_name: e.target.value })}
                />
              </div>
            ) : (
              <>
                {/* Enterprise name with search */}
                <div className="mb-4 relative">
                  <label className="block text-sm font-medium text-foreground mb-1.5">企业名称</label>
                  <input
                    className="w-full px-3 py-2 text-sm border border-border rounded-lg bg-background text-foreground outline-none focus:border-primary transition-colors"
                    placeholder="请输入企业名称搜索"
                    value={searchKeyword}
                    onChange={(e) => {
                      setSearchKeyword(e.target.value);
                      setForm({ ...form, title_name: e.target.value });
                      setShowDropdown(true);
                    }}
                    onFocus={() => setShowDropdown(true)}
                  />
                  {showDropdown && companies && companies.length > 0 && (
                    <div className="absolute left-0 right-0 top-full mt-1 bg-card border border-border rounded-lg shadow-elevated z-10 max-h-48 overflow-y-auto">
                      {companies.map((c, i) => (
                        <button
                          key={i}
                          className="w-full text-left px-3 py-2 text-sm text-foreground hover:bg-muted/60 transition-colors"
                          onClick={() => {
                            setForm({ ...form, title_name: c.name, tax_number: c.tax_number });
                            setSearchKeyword(c.name);
                            setShowDropdown(false);
                          }}
                        >
                          <div className="font-medium">{c.name}</div>
                          <div className="text-xs text-muted-foreground">{c.tax_number}</div>
                        </button>
                      ))}
                    </div>
                  )}
                </div>

                <div className="mb-4">
                  <label className="block text-sm font-medium text-foreground mb-1.5">税号</label>
                  <input
                    className="w-full px-3 py-2 text-sm border border-border rounded-lg bg-background text-foreground outline-none focus:border-primary transition-colors"
                    placeholder="请输入纳税人识别号"
                    value={form.tax_number}
                    onChange={(e) => setForm({ ...form, tax_number: e.target.value })}
                  />
                </div>

                <div className="mb-4">
                  <label className="block text-sm font-medium text-foreground mb-1.5">
                    开户银行 <span className="text-destructive">*</span>
                  </label>
                  <input
                    className="w-full px-3 py-2 text-sm border border-border rounded-lg bg-background text-foreground outline-none focus:border-primary transition-colors"
                    placeholder="请输入开户银行"
                    value={form.bank_name}
                    onChange={(e) => setForm({ ...form, bank_name: e.target.value })}
                  />
                </div>

                <div className="mb-4">
                  <label className="block text-sm font-medium text-foreground mb-1.5">
                    银行账号 <span className="text-destructive">*</span>
                  </label>
                  <input
                    className="w-full px-3 py-2 text-sm border border-border rounded-lg bg-background text-foreground outline-none focus:border-primary transition-colors"
                    placeholder="请输入银行账号"
                    value={form.bank_account}
                    onChange={(e) => setForm({ ...form, bank_account: e.target.value })}
                  />
                </div>

                <div className="mb-4">
                  <label className="block text-sm font-medium text-foreground mb-1.5">
                    企业地址 <span className="text-destructive">*</span>
                  </label>
                  <input
                    className="w-full px-3 py-2 text-sm border border-border rounded-lg bg-background text-foreground outline-none focus:border-primary transition-colors"
                    placeholder="请输入企业地址"
                    value={form.address}
                    onChange={(e) => setForm({ ...form, address: e.target.value })}
                  />
                </div>

                <div className="mb-5">
                  <label className="block text-sm font-medium text-foreground mb-1.5">
                    企业电话 <span className="text-destructive">*</span>
                  </label>
                  <input
                    className="w-full px-3 py-2 text-sm border border-border rounded-lg bg-background text-foreground outline-none focus:border-primary transition-colors"
                    placeholder="请输入企业电话"
                    value={form.phone}
                    onChange={(e) => setForm({ ...form, phone: e.target.value })}
                  />
                </div>
              </>
            )}

            {formError && (
              <div className="mb-4 px-3 py-2 rounded-lg bg-destructive/10 text-destructive text-xs">
                {formError}
              </div>
            )}

            <div className="flex gap-3 justify-end">
              <button onClick={closeModal} className="px-4 py-2 bg-muted text-foreground rounded-lg text-sm font-medium hover:bg-muted/80 transition-colors">
                取消
              </button>
              <button
                onClick={handleSave}
                disabled={isSaving || !form.title_name.trim()}
                className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-50"
              >
                {isSaving ? "保存中..." : "保存"}
              </button>
            </div>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default InvoiceTitlesPage;
