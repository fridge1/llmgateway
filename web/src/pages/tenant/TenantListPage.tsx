import { useNavigate } from "react-router-dom";
import {
  Building2, ChevronRight, Loader2, Mail, Check,
} from "lucide-react";
import { TenantPageHeader } from "@/components/tenant/TenantPageHeader";
import {
  useTenants,
  usePendingInvitations,
  useAcceptInvitation,
} from "@/hooks/use-tenant";

const TenantListPage = () => {
  const navigate = useNavigate();
  const { data: tenants, isLoading } = useTenants();
  const { data: invitations } = usePendingInvitations();
  const acceptInvitation = useAcceptInvitation();

  const roleLabel: Record<string, string> = {
    owner: "所有者",
    admin: "管理员",
    member: "成员",
  };

  const roleColor: Record<string, string> = {
    owner: "bg-primary/10 text-primary",
    admin: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    member: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400",
  };

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
      <TenantPageHeader
        title="我的组织"
        description="管理你加入的组织和待处理邀请。如需创建组织，请联系管理员。"
        icon={Building2}
      />

      {/* Pending Invitations */}
      {invitations && invitations.length > 0 && (
        <div className="border border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-900/20 rounded-xl p-4 shadow-card mb-6">
          <div className="flex items-center gap-2 mb-3">
            <div className="w-7 h-7 rounded-lg bg-amber-100 dark:bg-amber-900/40 flex items-center justify-center">
              <Mail size={14} className="text-amber-600" />
            </div>
            <h3 className="text-sm font-medium">待接受的邀请</h3>
          </div>
          <div className="space-y-2">
            {invitations.map((inv) => (
              <div key={inv.id} className="flex items-center justify-between bg-card rounded-lg px-4 py-2.5 border border-border shadow-card">
                <div>
                  <span className="text-sm font-medium">{inv.tenant_name || "未知组织"}</span>
                  <span className="text-xs text-muted-foreground ml-2">
                    角色: {roleLabel[inv.role] || inv.role}
                  </span>
                </div>
                <button
                  onClick={() => acceptInvitation.mutate(inv.id)}
                  disabled={acceptInvitation.isPending}
                  className="btn-primary flex items-center gap-1 !px-3 !py-1.5 !text-xs"
                >
                  <Check size={12} />
                  接受
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tenant List */}
      {!tenants || tenants.length === 0 ? (
        <div className="empty-state border border-border rounded-xl shadow-card">
          <div className="w-14 h-14 bg-muted rounded-full flex items-center justify-center mb-4">
            <Building2 size={24} className="text-muted-foreground" />
          </div>
          <p className="text-sm font-medium text-muted-foreground">你还没有加入任何组织</p>
          <p className="text-xs text-muted-foreground/70 mt-1">如需创建组织，请联系管理员开通</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          {tenants.map((t, i) => (
            <div
              key={t.id}
              onClick={() => navigate(`/dashboard/tenants/${t.id}`)}
              className="flex items-center justify-between p-5 bg-card border border-border rounded-xl shadow-card hover:shadow-elevated hover:-translate-y-0.5 cursor-pointer transition-all duration-300 stagger-item"
              style={{ animationDelay: `${i * 60}ms` }}
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-primary/10 rounded-xl flex items-center justify-center">
                  <Building2 size={18} className="text-primary" />
                </div>
                <div>
                  <p className="font-semibold text-foreground">{t.name}</p>
                  <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium mt-0.5 ${roleColor[t.role] || ""}`}>
                    {roleLabel[t.role] || t.role}
                  </span>
                </div>
              </div>
              <ChevronRight size={16} className="text-muted-foreground" />
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default TenantListPage;
