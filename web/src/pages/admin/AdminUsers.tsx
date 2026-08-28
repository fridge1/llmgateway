import { useState, useEffect } from "react";
import { Search, ChevronLeft, ChevronRight, DollarSign, Users, X, Loader, Trash2, BarChart2, Tag } from "lucide-react";
import { useAdminUsers, useUpdateUserStatus, useRechargeUser, useDeleteUser, useToggleImageShare, useAdminUserPricing, useAdminUpsertUserPricing, useAdminDeleteUserPricing, useAdminPricing } from "@/hooks/use-api";
import AdminUserConsumption from "./AdminUserConsumption";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
interface RechargeTarget {
  id: string;
  phone: string;
  email: string;
  balance: number;
}

const PAGE_SIZE = 20;

const AdminUsers = () => {
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [requestedPage, setRequestedPage] = useState(1);
  const [rechargeModal, setRechargeModal] = useState<RechargeTarget | null>(null);
  const [rechargeAmount, setRechargeAmount] = useState("");
  const [rechargeDesc, setRechargeDesc] = useState("");
  const [rechargeSuccess, setRechargeSuccess] = useState(false);
  const [rechargeError, setRechargeError] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; phone: string; email: string } | null>(null);
  const [selectedUser, setSelectedUser] = useState<{ id: string; phone: string; email: string } | null>(null);
  const [pricingUser, setPricingUser] = useState<{ id: string; phone: string; email: string; nickname?: string } | null>(null);
  const [showPricingForm, setShowPricingForm] = useState(false);
  const [pricingForm, setPricingForm] = useState({ modelName: "", discountPercent: "100", enabled: true });

  // Debounce search input by 300ms — only reset page when value actually changes
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch((prev) => {
        if (prev !== search) {
          setRequestedPage(1);
          return search;
        }
        return prev;
      });
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const { data, isLoading } = useAdminUsers(requestedPage, PAGE_SIZE, debouncedSearch);
  const updateStatus = useUpdateUserStatus();
  const rechargeUser = useRechargeUser();
  const deleteUser = useDeleteUser();
  const toggleImageShare = useToggleImageShare();

  const { data: userPricingData } = useAdminUserPricing(pricingUser?.id || "");
  const upsertUserPricing = useAdminUpsertUserPricing();
  const deleteUserPricing = useAdminDeleteUserPricing();
  const { data: globalPricingData } = useAdminPricing();

  // If showing consumption detail, render that instead
  if (selectedUser) {
    return (
      <AdminUserConsumption
        userId={selectedUser.id}
        userPhone={selectedUser.phone || selectedUser.email}
        onBack={() => setSelectedUser(null)}
      />
    );
  }

  const users = data?.users ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const globalTotal = data?.global_total ?? 0;
  const activeCount = data?.active_count ?? 0;
  const disabledCount = globalTotal - activeCount;
  const totalBalance = data?.total_balance ?? 0;

  // Clamp displayed page to [1, totalPages] without setState in effect
  const page = Math.min(Math.max(1, requestedPage), totalPages);
  const setPage = setRequestedPage;

  const handleToggle = (userId: string, currentStatus: string) => {
    const newStatus = currentStatus === "active" ? "disabled" : "active";
    updateStatus.mutate({ id: userId, status: newStatus });
  };

  const handleRecharge = async () => {
    if (!rechargeModal || !rechargeAmount) return;
    const amount = parseFloat(rechargeAmount);
    if (!Number.isFinite(amount) || amount <= 0) return;
    setRechargeError("");
    try {
      await rechargeUser.mutateAsync({
        id: rechargeModal.id,
        amount,
        description: rechargeDesc || undefined,
      });
      setRechargeSuccess(true);
      setTimeout(() => {
        setRechargeSuccess(false);
        setRechargeModal(null);
        setRechargeAmount("");
        setRechargeDesc("");
      }, 1000);
    } catch (err) {
      setRechargeError(err instanceof Error ? err.message : "充值失败，请重试");
    }
  };

  return (
    <div className="page-container">
      {/* Page header */}
      <div className="mb-6">
        <h1 className="text-lg font-bold text-foreground">用户管理</h1>
        <p className="text-sm text-muted-foreground mt-0.5">管理平台用户账户、状态和余额</p>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-1 gap-4 mb-5 sm:grid-cols-2 lg:grid-cols-4">
        <div className="flex-1 bg-card border border-border rounded-xl px-5 py-4 shadow-card flex items-center gap-3 stagger-item" style={{ animationDelay: "0ms" }}>
          <div className="w-9 h-9 bg-primary/10 rounded-lg flex items-center justify-center">
            <Users size={16} className="text-primary" />
          </div>
          <div>
            <div className="text-lg font-bold text-foreground">{isLoading ? "..." : globalTotal}</div>
            <div className="text-xs text-muted-foreground">总用户数</div>
          </div>
        </div>
        <div className="flex-1 bg-card border border-border rounded-xl px-5 py-4 shadow-card flex items-center gap-3 stagger-item" style={{ animationDelay: "80ms" }}>
          <div className="w-9 h-9 bg-success/10 rounded-lg flex items-center justify-center">
            <Users size={16} className="text-success" />
          </div>
          <div>
            <div className="text-lg font-bold text-foreground">{isLoading ? "..." : activeCount}</div>
            <div className="text-xs text-muted-foreground">活跃用户</div>
          </div>
        </div>
        <div className="flex-1 bg-card border border-border rounded-xl px-5 py-4 shadow-card flex items-center gap-3 stagger-item" style={{ animationDelay: "160ms" }}>
          <div className="w-9 h-9 bg-destructive/10 rounded-lg flex items-center justify-center">
            <Users size={16} className="text-destructive" />
          </div>
          <div>
            <div className="text-lg font-bold text-foreground">{isLoading ? "..." : disabledCount}</div>
            <div className="text-xs text-muted-foreground">禁用用户</div>
          </div>
        </div>
        <div className="flex-1 bg-card border border-border rounded-xl px-5 py-4 shadow-card flex items-center gap-3 stagger-item" style={{ animationDelay: "240ms" }}>
          <div className="w-9 h-9 bg-amber-500/10 dark:bg-amber-500/15 rounded-lg flex items-center justify-center">
            <DollarSign size={16} className="text-amber-500" />
          </div>
          <div>
            <div className="text-lg font-bold text-foreground">
              {isLoading ? "..." : `¥${totalBalance.toFixed(2)}`}
            </div>
            <div className="text-xs text-muted-foreground">账户余额总计</div>
          </div>
        </div>
      </div>

      {/* Table */}
      <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between">
          <span className="text-sm font-semibold text-foreground">用户列表</span>
          <div className="relative w-56">
            <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              className="input-field h-8 text-xs w-full" style={{ paddingLeft: "2rem" }}
              placeholder="按用户名搜索..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); }}
            />
            {search && (
              <button onClick={() => setSearch("")} className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground">
                <X size={12} />
              </button>
            )}
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
                <TableHead className="text-left px-5 py-3.5">用户名</TableHead>
                <TableHead className="text-left px-5 py-3.5">昵称</TableHead>
                <TableHead className="text-left px-5 py-3.5">角色</TableHead>
                <TableHead className="text-left px-5 py-3.5">状态</TableHead>
                <TableHead className="text-left px-5 py-3.5">余额</TableHead>
                <TableHead className="text-left px-5 py-3.5">图片分发</TableHead>
                <TableHead className="text-left px-5 py-3.5">注册时间</TableHead>
                <TableHead className="text-right px-5 py-3.5">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="px-5 py-16">
                    <div className="empty-state">
                      <div className="w-12 h-12 bg-muted rounded-xl flex items-center justify-center mb-3">
                        <Search size={18} className="text-muted-foreground/50" />
                      </div>
                      <div className="text-sm text-muted-foreground">未找到用户</div>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                users.map((u, i) => {
                  const isActive = u.status === "active";
                  const balance = u.balance ?? 0;
                  const imageShareEnabled = u.image_share_enabled === true;
                  const username = u.phone || u.email;
                  return (
                    <TableRow
                      key={u.id}
                      className={`border-t border-border hover:bg-accent/30 transition-colors duration-150 ${i % 2 === 0 ? "" : "bg-muted/10"}`}
                    >
                      <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground">{username}</TableCell>
                      <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">{u.nickname || <span className="text-muted-foreground/40">{"\u2014"}</span>}</TableCell>
                      <TableCell className="px-5 py-3.5">
                        {u.role === "admin" ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-md bg-destructive/10 text-destructive text-xs font-medium border border-destructive/20">
                            admin
                          </span>
                        ) : (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-md bg-muted text-muted-foreground text-xs font-medium">
                            user
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="px-5 py-3.5">
                        {u.role !== "admin" ? (
                        <button
                          onClick={() => handleToggle(u.id, u.status)}
                          disabled={updateStatus.isPending}
                          className={`relative rounded-full transition-colors duration-200 ${isActive ? "bg-primary" : "bg-border"}`}
                          style={{ width: 40, height: 22 }}
                        >
                          <span
                            className="absolute bg-white rounded-full shadow transition-transform duration-200"
                            style={{ width: 18, height: 18, top: 2, left: 0, transform: isActive ? "translateX(20px)" : "translateX(2px)" }}
                          />
                        </button>
                        ) : (
                          <span className="text-xs text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm font-medium text-foreground">¥ {balance.toFixed(4)}</TableCell>
                      <TableCell className="px-5 py-3.5">
                        {u.role !== "admin" ? (
                          <button
                            onClick={() => toggleImageShare.mutate({ id: u.id, enabled: !imageShareEnabled })}
                            disabled={toggleImageShare.isPending}
                            title={imageShareEnabled ? "已开通图片分发权限" : "未开通图片分发权限"}
                            className={`relative rounded-full transition-colors duration-200 ${imageShareEnabled ? "bg-primary" : "bg-border"}`}
                            style={{ width: 40, height: 22 }}
                          >
                            <span
                              className="absolute bg-white rounded-full shadow transition-transform duration-200"
                              style={{ width: 18, height: 18, top: 2, left: 0, transform: imageShareEnabled ? "translateX(20px)" : "translateX(2px)" }}
                            />
                          </button>
                        ) : (
                          <span className="text-xs text-muted-foreground">{"—"}</span>
                        )}
                      </TableCell>
                      <TableCell className="px-5 py-3.5 text-sm text-muted-foreground">
                        {new Date(u.created_at).toLocaleString("zh-CN")}
                      </TableCell>
                      <TableCell className="px-5 py-3.5">
                        <div className="flex justify-end gap-1.5">
                          <button
                            onClick={() => setSelectedUser({ id: u.id, phone: u.phone, email: u.email })}
                            className="flex items-center gap-1 text-xs text-blue-500 hover:text-blue-600 px-2.5 py-1.5 rounded-lg hover:bg-blue-500/8 transition-colors font-medium border border-blue-500/20"
                          >
                            <BarChart2 size={11} />
                            查看消费
                          </button>
                          <button
                            onClick={() => { setRechargeModal({ id: u.id, phone: u.phone, email: u.email, balance }); setRechargeAmount(""); setRechargeDesc(""); }}
                            className="flex items-center gap-1 text-xs text-primary hover:text-primary/80 px-2.5 py-1.5 rounded-lg hover:bg-primary/8 transition-colors font-medium border border-primary/20"
                          >
                            <DollarSign size={11} />
                            充值
                          </button>
                          <button
                            onClick={() => setPricingUser({ id: u.id, phone: u.phone, email: u.email, nickname: u.nickname })}
                            className="flex items-center gap-1 text-xs text-purple-500 hover:text-purple-600 px-2.5 py-1.5 rounded-lg hover:bg-purple-500/8 transition-colors font-medium border border-purple-500/20"
                          >
                            <Tag size={11} />
                            定价
                          </button>
                          {u.role !== "admin" && (
                            <button
                              onClick={() => setDeleteTarget({ id: u.id, phone: u.phone, email: u.email })}
                              className="flex items-center gap-1 text-xs text-destructive hover:text-destructive/80 px-2.5 py-1.5 rounded-lg hover:bg-destructive/8 transition-colors font-medium border border-destructive/20"
                            >
                              <Trash2 size={11} />
                              删除
                            </button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        )}

        {/* Pagination */}
        <div className="px-5 py-3.5 border-t border-border flex items-center justify-between">
          <div className="text-xs text-muted-foreground">共 {total} 条记录</div>
          <div className="flex items-center gap-1">
            <button
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
              className={`w-7 h-7 rounded-lg flex items-center justify-center transition-colors ${page <= 1 ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"}`}
            >
              <ChevronLeft size={13} />
            </button>
            <span className="text-xs text-foreground px-2">
              <span className="font-medium">{page}</span>
              <span className="text-muted-foreground"> / {totalPages}</span>
            </span>
            <button
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
              className={`w-7 h-7 rounded-lg flex items-center justify-center transition-colors ${page >= totalPages ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted cursor-pointer"}`}
            >
              <ChevronRight size={13} />
            </button>
          </div>
        </div>
      </div>

      {/* Recharge modal */}
      {rechargeModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="w-[calc(100vw-2rem)] max-w-[420px] rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            {rechargeSuccess ? (
              <div className="flex flex-col items-center py-6">
                <div className="w-14 h-14 bg-success/10 rounded-full flex items-center justify-center mb-4">
                  <DollarSign size={24} className="text-success" />
                </div>
                <div className="text-base font-bold text-foreground mb-1">充值成功</div>
                <div className="text-sm text-muted-foreground">已为 {rechargeModal.phone || rechargeModal.email} 充值 ¥{rechargeAmount}</div>
              </div>
            ) : (
              <>
                <h3 className="text-base font-bold text-foreground mb-1">用户充值</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  为用户 <span className="text-foreground font-medium">{rechargeModal.phone || rechargeModal.email}</span> 增加余额
                </p>
                <div className="bg-muted/60 rounded-lg px-4 py-3 mb-4 flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">当前余额</span>
                  <span className="text-sm font-semibold text-foreground">¥ {rechargeModal.balance.toFixed(4)}</span>
                </div>
                <label className="block text-sm font-medium text-foreground mb-1.5">充值金额（元）</label>
                <input
                  className="input-field mb-3"
                  placeholder="请输入充值金额"
                  type="number"
                  min="0.01"
                  step="0.01"
                  value={rechargeAmount}
                  onChange={(e) => setRechargeAmount(e.target.value)}
                />
                <label className="block text-sm font-medium text-foreground mb-1.5">备注（可选）</label>
                <input
                  className="input-field mb-3"
                  placeholder="充值备注"
                  value={rechargeDesc}
                  onChange={(e) => setRechargeDesc(e.target.value)}
                />
                {rechargeError && (
                  <div className="text-sm text-destructive mb-3">{rechargeError}</div>
                )}
                <div className="flex gap-3 justify-end">
                  <button onClick={() => { setRechargeModal(null); setRechargeError(""); }} className="btn-secondary">取消</button>
                  <button
                    disabled={!rechargeAmount || !Number.isFinite(parseFloat(rechargeAmount)) || parseFloat(rechargeAmount) <= 0 || rechargeUser.isPending}
                    onClick={handleRecharge}
                    className={`btn-primary flex items-center gap-1.5 ${!rechargeAmount || !Number.isFinite(parseFloat(rechargeAmount)) || parseFloat(rechargeAmount) <= 0 ? "opacity-50 cursor-not-allowed" : ""}`}
                  >
                    <DollarSign size={13} />
                    {rechargeUser.isPending ? "处理中..." : "确认充值"}
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Delete confirmation modal */}
      {deleteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="w-[calc(100vw-2rem)] max-w-[400px] rounded-2xl bg-card p-4 shadow-modal slide-up sm:p-6">
            <div className="flex flex-col items-center mb-5">
              <div className="w-14 h-14 bg-destructive/10 rounded-full flex items-center justify-center mb-4">
                <Trash2 size={24} className="text-destructive" />
              </div>
              <h3 className="text-base font-bold text-foreground mb-1">确认删除用户</h3>
              <p className="text-sm text-muted-foreground text-center">
                将永久删除用户 <span className="text-foreground font-medium">{deleteTarget.phone || deleteTarget.email}</span> 及其所有数据（余额、订单、流水等），此操作不可撤销。
              </p>
            </div>
            <div className="flex gap-3 justify-end">
              <button onClick={() => setDeleteTarget(null)} className="btn-secondary">取消</button>
              <button
                disabled={deleteUser.isPending}
                onClick={async () => {
                  await deleteUser.mutateAsync(deleteTarget.id);
                  setDeleteTarget(null);
                }}
                className="btn-primary bg-destructive hover:bg-destructive/90 flex items-center gap-1.5"
              >
                <Trash2 size={13} />
                {deleteUser.isPending ? "删除中..." : "确认删除"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* User pricing modal */}
      {pricingUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop fade-in" role="dialog" aria-modal="true">
          <div className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-[700px] overflow-y-auto rounded-2xl bg-card shadow-modal slide-up">
            <div className="sticky top-0 bg-card border-b border-border px-6 py-4 flex items-center justify-between z-10">
              <h3 className="text-lg font-bold text-foreground">
                {pricingUser.nickname || pricingUser.phone || pricingUser.email} - 定价管理
              </h3>
              <button onClick={() => { setPricingUser(null); setShowPricingForm(false); }} className="text-muted-foreground hover:text-foreground transition-colors">
                <X size={20} />
              </button>
            </div>

            <div className="p-6">
              <div className="flex items-center justify-between mb-4">
                <div className="text-sm text-muted-foreground">已配置 {userPricingData?.pricing?.length || 0} 个模型</div>
                <button
                  onClick={() => { setShowPricingForm(true); setPricingForm({ modelName: "", discountPercent: "100", enabled: true }); }}
                  className="btn-primary text-xs px-3 py-1.5"
                >
                  添加定价
                </button>
              </div>

              {!showPricingForm && (
                <div className="space-y-2">
                  {userPricingData?.pricing && userPricingData.pricing.length > 0 ? (
                    userPricingData.pricing.map((p) => (
                      <div key={p.id} className="flex items-center justify-between p-3 bg-muted/30 rounded-lg border border-border">
                        <div className="flex-1">
                          <div className="font-medium text-sm text-foreground">{p.model_name}</div>
                          <div className="text-xs text-muted-foreground mt-0.5">
                            {p.discount_rate ? (
                              <>折扣率: {(p.discount_rate * 100).toFixed(1)}%</>
                            ) : (
                              <>输入: {p.input_price} / 输出: {p.output_price}</>
                            )}
                            {!p.is_active && <span className="ml-2 text-amber-500">(已禁用)</span>}
                          </div>
                        </div>
                        <button
                          onClick={async () => {
                            if (confirm(`确认删除模型 ${p.model_name} 的定价？`)) {
                              await deleteUserPricing.mutateAsync({ userId: pricingUser.id, modelName: p.model_name });
                            }
                          }}
                          className="text-destructive hover:text-destructive/80 text-xs px-2 py-1"
                        >
                          删除
                        </button>
                      </div>
                    ))
                  ) : (
                    <div className="text-center py-8 text-sm text-muted-foreground">暂无定价配置</div>
                  )}
                </div>
              )}

              {showPricingForm && (
                <div className="space-y-4 p-4 bg-muted/20 rounded-lg border border-border">
                  <div>
                    <label className="block text-xs font-medium mb-1">模型名称</label>
                    <select
                      value={pricingForm.modelName}
                      onChange={(e) => setPricingForm({ ...pricingForm, modelName: e.target.value })}
                      className="input-field text-sm"
                    >
                      <option value="">选择模型</option>
                      {globalPricingData?.pricing?.map((gp) => (
                        <option key={gp.model_name} value={gp.model_name}>{gp.model_name}</option>
                      ))}
                    </select>
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
                      className="input-field text-sm"
                      placeholder="80=8折，100=原价，120=提价20%"
                    />
                    <p className="mt-1 text-[11px] text-muted-foreground">
                      实际价 = 全局价 × 定价因子。全局调价时本用户价格自动联动。
                    </p>
                    {(() => {
                      const pct = parseFloat(pricingForm.discountPercent);
                      const gp = globalPricingData?.pricing?.find((g) => g.model_name === pricingForm.modelName.trim());
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
                    <button
                      onClick={async () => {
                        if (!pricingForm.modelName.trim()) return;
                        const pct = parseFloat(pricingForm.discountPercent);
                        if (!(pct > 0 && pct <= 1000)) {
                          alert("定价因子必须在 (0, 1000] 范围内");
                          return;
                        }
                        await upsertUserPricing.mutateAsync({
                          userId: pricingUser.id,
                          modelName: pricingForm.modelName.trim(),
                          pricing: {
                            discount_rate: pct / 100,
                            is_active: pricingForm.enabled,
                          },
                        });
                        setShowPricingForm(false);
                        setPricingForm({ modelName: "", discountPercent: "100", enabled: true });
                      }}
                      disabled={upsertUserPricing.isPending || !pricingForm.modelName.trim()}
                      className="btn-primary px-3 py-1.5 text-xs disabled:opacity-50"
                    >
                      {upsertUserPricing.isPending ? "保存中..." : "保存"}
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminUsers;
