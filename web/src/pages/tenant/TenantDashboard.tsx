import { useParams, useNavigate } from "react-router-dom";
import {
  Building2, Users, Key, CreditCard, Settings, Loader2, BarChart2, TrendingUp,
  Wallet,
} from "lucide-react";
import {
  useTenantDetail,
  useTenantBalance,
  useTenantSubUsers,
  useTenantKeys,
} from "@/hooks/use-tenant";
import { TenantPageHeader } from "@/components/tenant/TenantPageHeader";

const TenantDashboard = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: tenant, isLoading: tenantLoading } = useTenantDetail(id!);
  const { data: balance } = useTenantBalance(id!);
  const { data: subUsers } = useTenantSubUsers(id!);
  const { data: keys } = useTenantKeys(id!);

  if (tenantLoading) {
    return (
      <div className="page-container fade-in flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={28} className="animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">加载中...</span>
        </div>
      </div>
    );
  }

  if (!tenant) {
    return (
      <div className="page-container fade-in">
        <div className="empty-state">
          <div className="w-14 h-14 bg-muted rounded-full flex items-center justify-center mb-4">
            <Building2 size={24} className="text-muted-foreground" />
          </div>
          <p className="text-sm font-medium text-muted-foreground">组织不存在</p>
        </div>
      </div>
    );
  }

  const cards = [
    {
      title: "子用户管理",
      icon: Users,
      value: subUsers?.length || 0,
      unit: "个子用户",
      path: `/dashboard/tenants/${id}/members`,
      iconBg: "bg-blue-50 dark:bg-blue-500/10",
      iconColor: "text-blue-500",
    },
    {
      title: "API 密钥",
      icon: Key,
      value: keys?.length || 0,
      unit: "个密钥",
      path: `/dashboard/tenants/${id}/keys`,
      iconBg: "bg-emerald-50 dark:bg-emerald-500/10",
      iconColor: "text-emerald-500",
    },
    {
      title: "使用记录",
      icon: BarChart2,
      value: "",
      unit: "查看所有子用户用量",
      path: `/dashboard/tenants/${id}/usage`,
      iconBg: "bg-violet-50 dark:bg-violet-500/10",
      iconColor: "text-violet-500",
    },
    {
      title: "数据分析",
      icon: TrendingUp,
      value: "",
      unit: "费用趋势与模型分布",
      path: `/dashboard/tenants/${id}/analytics`,
      iconBg: "bg-indigo-50 dark:bg-indigo-500/10",
      iconColor: "text-indigo-500",
    },
    {
      title: "设置",
      icon: Settings,
      value: "",
      unit: "组织设置",
      path: `/dashboard/tenants/${id}/settings`,
      iconBg: "bg-gray-100 dark:bg-gray-500/10",
      iconColor: "text-gray-500",
    },
  ];

  return (
    <div className="page-container fade-in">
      <TenantPageHeader
        title={tenant.name}
        description="组织概览与快捷入口"
        icon={Building2}
        onBack={() => navigate("/dashboard/tenants")}
      />

      {/* Balance Accent Card */}
      <div
        className="rounded-xl p-5 border border-transparent brand-gradient text-white shadow-button mb-6 stagger-item cursor-pointer transition-all duration-300 hover:opacity-95"
        onClick={() => navigate(`/dashboard/tenants/${id}/billing`)}
        style={{ animationDelay: "0ms" }}
      >
        <div className="flex items-start justify-between mb-3">
          <div className="w-9 h-9 rounded-lg bg-white/15 flex items-center justify-center">
            <Wallet size={18} className="text-white" />
          </div>
        </div>
        <div className="text-2xl font-bold text-white mb-1">
          ¥{(balance?.balance || 0).toFixed(2)}
        </div>
        <div className="text-xs text-white/70">
          余额 · 冻结 ¥{(balance?.frozen || 0).toFixed(2)}
        </div>
      </div>

      {/* Feature Cards */}
      <div className="grid grid-cols-1 gap-4 mb-6 sm:grid-cols-2 xl:grid-cols-3">
        {cards.map((card, i) => {
          const Icon = card.icon;
          return (
            <div
              key={card.title}
              onClick={() => navigate(card.path)}
              className="bg-card border border-border rounded-xl p-5 shadow-card hover:shadow-elevated hover:-translate-y-0.5 cursor-pointer transition-all duration-300 stagger-item"
              style={{ animationDelay: `${(i + 1) * 80}ms` }}
            >
              <div className="flex items-start justify-between mb-3">
                <div className={`w-9 h-9 rounded-lg flex items-center justify-center ${card.iconBg}`}>
                  <Icon size={18} className={card.iconColor} />
                </div>
              </div>
              <div>
                {card.value !== "" && (
                  <p className="text-2xl font-bold text-foreground mb-1">{card.value}</p>
                )}
                <p className="text-sm font-medium text-foreground">{card.title}</p>
                <p className="text-xs text-muted-foreground mt-0.5">{card.unit}</p>
              </div>
            </div>
          );
        })}
      </div>

      {/* Financial Overview */}
      {balance && (
        <div className="bg-card border border-border rounded-xl p-5 shadow-card stagger-item" style={{ animationDelay: "480ms" }}>
          <div className="flex items-center gap-2 mb-4">
            <div className="w-7 h-7 rounded-lg bg-amber-50 dark:bg-amber-500/10 flex items-center justify-center">
              <CreditCard size={14} className="text-amber-500" />
            </div>
            <h3 className="text-sm font-semibold text-foreground">财务概览</h3>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <div>
              <p className="text-xs text-muted-foreground mb-1">可用余额</p>
              <p className="text-lg font-semibold text-foreground">¥{balance.balance.toFixed(2)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground mb-1">冻结金额</p>
              <p className="text-lg font-semibold text-amber-500">¥{balance.frozen.toFixed(2)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground mb-1">累计充值</p>
              <p className="text-lg font-semibold text-green-600">¥{(balance.total_recharged ?? 0).toFixed(2)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground mb-1">累计消费</p>
              <p className="text-lg font-semibold text-destructive">¥{(balance.total_consumed ?? 0).toFixed(2)}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default TenantDashboard;
