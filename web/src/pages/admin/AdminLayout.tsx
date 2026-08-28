import { useState, useRef, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  Activity, BarChart3, ChevronDown, Image, LayoutDashboard,
  LogOut, Megaphone, Menu, Server, Settings, ShieldCheck, User, Users,
  WalletCards, Zap,
  type LucideIcon,
} from "lucide-react";
import { ThemeToggle } from "@/components/ui/nav-atoms";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";

type NavLink = { type: "link"; key: AdminPage; label: string; icon?: LucideIcon };
type NavGroup = { type: "group"; label: string; icon: LucideIcon; children: { key: AdminPage; label: string }[] };
import { useAuth } from "@/contexts/AuthContext";
import AdminOverview from "./AdminOverview";
import AdminUpstream from "./AdminUpstream";
import AdminModels from "./AdminModels";
import AdminPricing from "./AdminPricing";
import AdminUsers from "./AdminUsers";
import AdminConfig from "./AdminConfig";
import AdminInvoices from "./AdminInvoices";
import AdminOrders from "./AdminOrders";
import AdminConsumption from "./AdminConsumption";
import AdminAnnouncements from "./AdminAnnouncements";
import AdminSubscriptionOrders from "./AdminSubscriptionOrders";
import AdminSubscriptionUsage from "./AdminSubscriptionUsage";
import AdminSubscriptionPlans from "./AdminSubscriptionPlans";
import AdminRechargePromotions from "./AdminRechargePromotions";
import AdminTenants from "./AdminTenants";
import AdminRechargeLottery from "./AdminRechargeLottery";
import AdminLottery from "./AdminLottery";
import AdminAlerts from "./AdminAlerts";
import AdminModeration from "./AdminModeration";
import AdminBlockedIPs from "./AdminBlockedIPs";
import AdminTickets from "./AdminTickets";
import AdminRefunds from "./AdminRefunds";
import AdminReferralRules from "./AdminReferralRules";
import AdminImageStats from "./AdminImageStats";
import AdminCodexOrders from "./AdminCodexOrders";
import AdminCodexProducts from "./AdminCodexProducts";

export type AdminPage = "overview" | "consumption" | "image-stats" | "upstream" | "models" | "pricing" | "users" | "tenants" | "tickets-support" | "orders" | "refunds" | "subscription-orders" | "subscription-usage" | "subscription-plans" | "invoices" | "announcements" | "recharge-promotions" | "recharge-lottery" | "lottery" | "referral-rules" | "alerts" | "moderation" | "blocked-ips" | "config" | "codex-orders" | "codex-products";

const ADMIN_NAV_ITEMS: (NavLink | NavGroup)[] = [
  { type: "link",  key: "overview",    label: "概览", icon: LayoutDashboard },
  { type: "link",  key: "consumption", label: "消耗统计", icon: BarChart3 },
  { type: "link",  key: "image-stats", label: "图片时长", icon: Image },
  { type: "group", label: "资源管理", icon: Server, children: [
    { key: "upstream", label: "上游管理" },
    { key: "models",   label: "模型管理" },
    { key: "pricing",  label: "定价管理" },
  ]},
  { type: "group", label: "用户", icon: Users, children: [
    { key: "users",   label: "用户管理" },
    { key: "tenants", label: "租户管理" },
    { key: "tickets-support", label: "工单管理" },
  ]},
  { type: "group", label: "财务", icon: WalletCards, children: [
    { key: "orders",              label: "订单管理" },
    { key: "refunds",             label: "退款记录" },
    { key: "codex-orders",        label: "Codex订单" },
    { key: "codex-products",      label: "Codex商品" },
    { key: "subscription-plans",  label: "套餐管理" },
    { key: "subscription-orders", label: "订阅订单" },
    { key: "subscription-usage",  label: "订阅用量" },
    { key: "invoices",            label: "发票管理" },
  ]},
  { type: "group", label: "营销", icon: Megaphone, children: [
    { key: "announcements",       label: "公告管理" },
    { key: "recharge-promotions", label: "充值赠送" },
    { key: "recharge-lottery",    label: "充值抽奖" },
    { key: "lottery",             label: "抽奖活动" },
    { key: "referral-rules",      label: "返佣配置" },
  ]},
  { type: "group", label: "安全", icon: ShieldCheck, children: [
    { key: "alerts",      label: "运维告警" },
    { key: "moderation",  label: "内容安全" },
    { key: "blocked-ips", label: "IP 封禁" },
  ]},
  { type: "link", key: "config", label: "系统配置", icon: Settings },
];

const AdminLayout = () => {
  const navigate = useNavigate();
  const { user, isLoading } = useAuth();
  const [activePage, setActivePage] = useState<AdminPage>("overview");
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [openGroup, setOpenGroup] = useState<string | null>(() => {
    // 进入时自动展开当前激活页所在分组
    const grp = navItems.find(
      (it) => it.type === "group" && it.children.some((c) => c.key === activePage),
    );
    return grp && grp.type === "group" ? grp.label : null;
  });
  const [mobileOpen, setMobileOpen] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);
  const navRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setUserMenuOpen(false);
      }
      // 移动端菜单打开时，点击由 Sheet 自行处理，不在此折叠分组
      if (!mobileOpen && navRef.current && !navRef.current.contains(e.target as Node)) {
        setOpenGroup(null);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [mobileOpen]);

  const navItems = ADMIN_NAV_ITEMS;

  const activeLabel = navItems.flatMap((item) =>
    item.type === "link" ? [item] : item.children,
  ).find((item) => item.key === activePage)?.label ?? "概览";

  const selectPage = (page: AdminPage) => {
    setActivePage(page);
    // 切换页面时自动展开目标页所在分组
    const grp = navItems.find(
      (it) => it.type === "group" && it.children.some((c) => c.key === page),
    );
    setOpenGroup(grp && grp.type === "group" ? grp.label : null);
    setMobileOpen(false);
  };

  const renderNavigation = () => (
    <div className="space-y-1">
      {navItems.map((item) => {
        if (item.type === "link") {
          const Icon = item.icon;
          const isActive = activePage === item.key;
          return (
            <button
              key={item.key}
              onClick={() => selectPage(item.key)}
              className={`admin-nav-item ${isActive ? "admin-nav-item-active" : ""}`}
            >
              {Icon && <Icon size={16} />}
              <span>{item.label}</span>
            </button>
          );
        }

        const GroupIcon = item.icon;
        const isGroupActive = item.children.some((child) => child.key === activePage);
        const isOpen = openGroup === item.label;
        return (
          <div key={item.label}>
            <button
              onClick={() => setOpenGroup(isOpen ? null : item.label)}
              className={`admin-nav-item ${isGroupActive ? "admin-nav-item-active" : ""}`}
              aria-expanded={isOpen}
            >
              <GroupIcon size={16} />
              <span className="flex-1 text-left">{item.label}</span>
              <ChevronDown size={14} className={`transition-transform duration-200 ${isOpen ? "rotate-180" : ""}`} />
            </button>
            {isOpen && (
              <div className="ml-4 border-l border-border/70 pl-3 py-1 space-y-0.5">
                {item.children.map((child) => {
                  const childActive = activePage === child.key;
                  return (
                    <button
                      key={child.key}
                      onClick={() => selectPage(child.key)}
                      className={`admin-nav-subitem ${childActive ? "admin-nav-subitem-active" : ""}`}
                    >
                      <span className="h-1.5 w-1.5 rounded-full bg-current opacity-50" />
                      {child.label}
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );

  const renderPage = () => {
    switch (activePage) {
      case "overview": return <AdminOverview onNavigate={(p) => selectPage(p as AdminPage)} />;
      case "consumption": return <AdminConsumption />;
      case "image-stats": return <AdminImageStats />;
      case "upstream": return <AdminUpstream />;
      case "models": return <AdminModels />;
      case "pricing": return <AdminPricing />;
      case "users": return <AdminUsers />;
      case "tenants": return <AdminTenants />;
      case "tickets-support": return <AdminTickets />;
      case "orders": return <AdminOrders />;
      case "refunds": return <AdminRefunds />;
      case "codex-orders": return <AdminCodexOrders />;
      case "codex-products": return <AdminCodexProducts />;
      case "subscription-plans": return <AdminSubscriptionPlans />;
      case "subscription-orders": return <AdminSubscriptionOrders />;
      case "subscription-usage": return <AdminSubscriptionUsage />;
      case "invoices": return <AdminInvoices />;
      case "announcements": return <AdminAnnouncements />;
      case "recharge-promotions": return <AdminRechargePromotions />;
      case "recharge-lottery": return <AdminRechargeLottery />;
      case "lottery": return <AdminLottery />;
      case "referral-rules": return <AdminReferralRules />;
      case "alerts": return <AdminAlerts />;
      case "moderation": return <AdminModeration />;
      case "blocked-ips": return <AdminBlockedIPs />;
      case "config": return <AdminConfig />;
      default: return <AdminOverview />;
    }
  };

  return (
    <div data-cmp="AdminLayout" className="min-h-screen bg-background">
      <div className="flex min-h-screen">
        <aside className="admin-sidebar hidden lg:flex">
          <div className="flex items-center gap-3 px-5 py-5">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl brand-gradient shadow-button">
              <Zap size={16} className="text-white" />
            </div>
            <div>
              <div className="text-sm font-bold tracking-tight text-foreground">管理后台</div>
              <div className="text-[11px] text-muted-foreground">LLM Gateway Console</div>
            </div>
          </div>
          <div className="mx-4 mb-5 rounded-xl border border-primary/15 bg-primary/5 px-3.5 py-3">
            <div className="flex items-center gap-2 text-xs font-semibold text-primary">
              <Activity size={14} />
              <span>运营控制台</span>
            </div>
            <div className="mt-1 text-[11px] leading-relaxed text-muted-foreground">管理模型、用户、资金与平台运行状态</div>
          </div>
          <div className="flex-1 overflow-y-auto px-3 pb-4" ref={navRef}>
            {renderNavigation()}
          </div>
          <div className="border-t border-border/70 p-4">
            <button onClick={() => navigate("/dashboard")} className="admin-back-link">
              <LogOut size={15} />
              返回用户端
            </button>
          </div>
        </aside>

        <div className="min-w-0 flex-1">
          <header className="sticky top-0 z-40 flex h-16 items-center justify-between border-b border-border/70 bg-card/80 px-4 glass sm:px-6 lg:px-8">
            <div className="flex min-w-0 items-center gap-3">
              <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
                <SheetTrigger asChild>
                  <button className="admin-menu-button lg:hidden" aria-label="打开管理后台导航">
                    <Menu size={18} />
                  </button>
                </SheetTrigger>
                <SheetContent side="left" className="w-[290px] p-0">
                  <SheetTitle className="sr-only">管理后台导航</SheetTitle>
                  <div className="flex items-center gap-3 border-b border-border/70 px-5 py-5">
                    <div className="flex h-9 w-9 items-center justify-center rounded-xl brand-gradient shadow-button">
                      <Zap size={16} className="text-white" />
                    </div>
                    <div>
                      <div className="text-sm font-bold tracking-tight text-foreground">管理后台</div>
                      <div className="text-[11px] text-muted-foreground">LLM Gateway Console</div>
                    </div>
                  </div>
                  <div className="overflow-y-auto px-3 py-4">{renderNavigation()}</div>
                </SheetContent>
              </Sheet>
              <div className="min-w-0">
                <div className="text-xs font-medium text-muted-foreground">运营控制台</div>
                <div className="truncate text-base font-semibold text-foreground">{activeLabel}</div>
              </div>
            </div>

            <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
              <span className="hidden rounded-full bg-destructive/10 px-2.5 py-1 text-xs font-medium text-destructive sm:inline-flex">管理员</span>
              <ThemeToggle />
              <div className="relative" ref={userMenuRef}>
                <button
                  onClick={() => setUserMenuOpen((v) => !v)}
                  className="flex h-9 items-center gap-2 rounded-lg px-2 text-sm text-muted-foreground transition-colors duration-150 hover:bg-muted hover:text-foreground cursor-pointer"
                  aria-expanded={userMenuOpen}
                >
                  <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10">
                    <User size={12} className="text-primary" />
                  </div>
                  <span className="hidden max-w-28 truncate text-xs font-medium sm:inline">{isLoading ? "…" : (user?.phone ?? "-")}</span>
                  <ChevronDown size={12} className={`hidden transition-transform duration-200 sm:block ${userMenuOpen ? "rotate-180" : ""}`} />
                </button>
                {userMenuOpen && (
                  <div className="absolute right-0 top-full z-50 mt-2 w-44 rounded-xl border border-border/70 bg-card/95 py-1.5 shadow-elevated glass fade-in">
                    <button
                      onClick={() => { setUserMenuOpen(false); navigate("/dashboard"); }}
                      className="flex w-full items-center gap-2.5 px-3.5 py-2.5 text-sm text-destructive transition-colors duration-150 hover:bg-destructive/8 cursor-pointer"
                    >
                      <LogOut size={14} />
                      返回用户端
                    </button>
                  </div>
                )}
              </div>
            </div>
          </header>
          <main className="fade-in">{renderPage()}</main>
        </div>
      </div>
    </div>
  );
};

export default AdminLayout;
