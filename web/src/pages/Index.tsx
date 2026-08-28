import { useEffect, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  ArrowLeftRight,
  Bell,
  BookOpen,
  Building2,
  ChevronDown,
  CircleDollarSign,
  Crown,
  Download,
  FileText,
  Gift,
  Headset,
  Image as ImageIcon,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Menu,
  Receipt,
  Settings,
  Sparkles,
  TicketCheck,
  Trophy,
  User,
  WalletCards,
  Wrench,
  Zap,
  type LucideIcon,
} from "lucide-react";
import AnnouncementBanner from "../components/AnnouncementBanner";
import NotificationBell from "@/components/NotificationBell";
import { ThemeToggle } from "@/components/ui/nav-atoms";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { useAuth } from "@/contexts/AuthContext";
import { useTenants } from "@/hooks/use-tenant";

type UserNavLink = {
  type: "link";
  path: string;
  label: string;
  icon: LucideIcon;
  matchPrefix?: string;
};

type UserNavGroup = {
  type: "group";
  label: string;
  icon: LucideIcon;
  children: UserNavLink[];
};

const MainLayout = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const auth = useAuth();
  const { data: tenants } = useTenants();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [openGroup, setOpenGroup] = useState<string | null>(null);

  const navItems: (UserNavLink | UserNavGroup)[] = [
    { type: "link", path: "/dashboard", label: "工作台", icon: LayoutDashboard },
    { type: "link", path: "/dashboard/keys", label: "API 密钥", icon: KeyRound },
    { type: "link", path: "/dashboard/models", label: "可用模型", icon: Sparkles },
    {
      type: "group",
      label: "资金与账单",
      icon: WalletCards,
      children: [
        { type: "link", path: "/dashboard/balance", label: "账户余额", icon: CircleDollarSign },
        { type: "link", path: "/dashboard/transactions", label: "交易记录", icon: ArrowLeftRight },
        { type: "link", path: "/dashboard/orders", label: "充值订单", icon: Receipt },
        { type: "link", path: "/dashboard/invoice/titles", label: "发票抬头", icon: FileText },
        { type: "link", path: "/dashboard/invoice/requests", label: "开票记录", icon: TicketCheck },
      ],
    },
    { type: "link", path: "/dashboard/subscription", label: "订阅套餐", icon: Crown },
    {
      type: "group",
      label: "应用与服务",
      icon: Wrench,
      children: [
        { type: "link", path: "/dashboard/tools", label: "AI 工具集", icon: Wrench },
        ...(tenants && tenants.length > 0
          ? [{ type: "link" as const, path: "/dashboard/tenants", label: "我的组织", icon: Building2, matchPrefix: "/dashboard/tenants" }]
          : []),
        ...(auth.user?.image_share_enabled
          ? [{ type: "link" as const, path: "/dashboard/image-share", label: "图片分发", icon: ImageIcon }]
          : []),
      ],
    },
    {
      type: "group",
      label: "成长与支持",
      icon: Gift,
      children: [
        { type: "link", path: "/dashboard/growth", label: "成长中心", icon: Gift },
        { type: "link", path: "/dashboard/lottery", label: "抽奖活动", icon: Trophy },
        { type: "link", path: "/dashboard/notifications", label: "通知中心", icon: Bell },
        { type: "link", path: "/dashboard/tickets", label: "工单支持", icon: Headset },
      ],
    },
  ];

  const isActive = (item: UserNavLink) => {
    const prefix = item.matchPrefix ?? item.path;
    return item.path === "/dashboard"
      ? location.pathname === item.path
      : location.pathname === item.path || location.pathname.startsWith(`${prefix}/`);
  };

  const activeItem = navItems
    .flatMap((item) => (item.type === "group" ? item.children : [item]))
    .find(isActive);
  const activeLabel = activeItem?.label ?? "用户中心";

  // 路由变化时自动展开当前激活项所在分组，便于用户感知位置；
  // 同时允许用户手动折叠/展开其他分组（不再被激活态强制锁定）。
  useEffect(() => {
    const activeGroup = navItems.find(
      (item) => item.type === "group" && item.children.some(isActive),
    );
    setOpenGroup(activeGroup && activeGroup.type === "group" ? activeGroup.label : null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname]);
  const userIdentifier = auth.user?.phone ?? auth.user?.email ?? "已登录用户";

  const goTo = (path: string) => {
    navigate(path);
    setMobileOpen(false);
    setOpenGroup(null);
  };

  const handleLogout = async () => {
    await auth.logout();
    navigate("/login");
  };

  const renderNavigation = () => (
    <div className="space-y-1">
      {navItems.map((item) => {
        if (item.type === "link") {
          const Icon = item.icon;
          const active = isActive(item);
          return (
            <button
              key={item.path}
              onClick={() => goTo(item.path)}
              className={`admin-nav-item ${active ? "admin-nav-item-active" : ""}`}
              aria-current={active ? "page" : undefined}
            >
              <Icon size={16} aria-hidden="true" />
              <span>{item.label}</span>
            </button>
          );
        }

        const GroupIcon = item.icon;
        const groupActive = item.children.some(isActive);
        const expanded = openGroup === item.label;
        return (
          <div key={item.label}>
            <button
              onClick={() => setOpenGroup(openGroup === item.label ? null : item.label)}
              className={`admin-nav-item ${groupActive ? "admin-nav-item-active" : ""}`}
              aria-expanded={expanded}
            >
              <GroupIcon size={16} aria-hidden="true" />
              <span className="flex-1 text-left">{item.label}</span>
              <ChevronDown size={14} aria-hidden="true" className={`transition-transform duration-200 ${expanded ? "rotate-180" : ""}`} />
            </button>
            {expanded && (
              <div className="ml-4 space-y-0.5 border-l border-border/70 py-1 pl-3">
                {item.children.map((child) => {
                  const ChildIcon = child.icon;
                  const active = isActive(child);
                  return (
                    <button
                      key={child.path}
                      onClick={() => goTo(child.path)}
                      className={`admin-nav-subitem ${active ? "admin-nav-subitem-active" : ""}`}
                      aria-current={active ? "page" : undefined}
                    >
                      <ChildIcon size={14} aria-hidden="true" />
                      <span>{child.label}</span>
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

  const sidebarContent = (mobile = false) => (
    <>
      <div className="flex items-center gap-3 px-5 py-5">
        <button
          onClick={() => goTo("/dashboard")}
          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl brand-gradient shadow-button"
          aria-label="返回用户工作台"
        >
          <Zap size={16} className="text-white" aria-hidden="true" />
        </button>
        <div className="min-w-0">
          <div className="truncate text-sm font-bold tracking-tight text-foreground">LLM Gateway</div>
          <div className="text-[11px] text-muted-foreground">User Console</div>
        </div>
      </div>
      <div className="mx-4 mb-5 rounded-xl border border-primary/15 bg-primary/5 px-3.5 py-3">
        <div className="flex items-center gap-2 text-xs font-semibold text-primary">
          <User size={14} aria-hidden="true" />
          <span>个人工作台</span>
        </div>
        <div className="mt-1 truncate text-[11px] leading-relaxed text-muted-foreground" title={userIdentifier}>
          {userIdentifier}
        </div>
      </div>
      <div className={`flex-1 overflow-y-auto px-3 pb-4 ${mobile ? "min-h-0" : ""}`}>
        {renderNavigation()}
      </div>
      <div className="space-y-1 border-t border-border/70 p-4">
        <button onClick={() => goTo("/docs")} className="admin-back-link">
          <BookOpen size={15} aria-hidden="true" />
          API 文档
        </button>
        <button onClick={() => goTo("/download")} className="admin-back-link">
          <Download size={15} aria-hidden="true" />
          下载客户端
        </button>
        {auth.isAdmin && (
          <button onClick={() => goTo("/admin")} className="admin-back-link text-primary">
            <Settings size={15} aria-hidden="true" />
            管理后台
          </button>
        )}
      </div>
    </>
  );

  return (
    <div data-cmp="MainLayout" className="min-h-screen bg-background">
      <div className="flex min-h-screen">
        <aside className="user-sidebar hidden lg:flex">{sidebarContent()}</aside>

        <div className="min-w-0 flex-1">
          <header className="sticky top-0 z-40 flex h-16 items-center justify-between border-b border-border/70 bg-card/80 px-4 glass sm:px-6 lg:px-8">
            <div className="flex min-w-0 items-center gap-3">
              <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
                <SheetTrigger asChild>
                  <button className="admin-menu-button lg:!hidden" aria-label="打开用户中心导航">
                    <Menu size={18} aria-hidden="true" />
                  </button>
                </SheetTrigger>
                <SheetContent side="left" className="w-[290px] gap-0 p-0 [&_.admin-back-link]:min-h-11 [&_.admin-nav-item]:min-h-11 [&_.admin-nav-subitem]:min-h-11">
                  <SheetTitle className="sr-only">用户中心导航</SheetTitle>
                  {sidebarContent(true)}
                </SheetContent>
              </Sheet>
              <div className="min-w-0">
                <div className="text-xs font-medium text-muted-foreground">用户控制台</div>
                <div className="truncate text-base font-semibold text-foreground">{activeLabel}</div>
              </div>
            </div>

            <div className="flex shrink-0 items-center gap-1 sm:gap-2">
              <Popover>
                <PopoverTrigger asChild>
                  <button className="admin-menu-button h-11 w-11 sm:h-9 sm:w-9" aria-label="联系客服">
                    <Headset size={16} aria-hidden="true" />
                  </button>
                </PopoverTrigger>
                <PopoverContent align="end" className="w-56 p-4">
                  <div className="flex flex-col items-center gap-2">
                    <p className="text-sm font-medium text-foreground">联系客服</p>
                    <img src="/wechat_QR.png" alt="客服微信二维码" className="h-40 w-40 rounded-lg" />
                    <p className="text-xs text-muted-foreground">微信扫码添加客服</p>
                  </div>
                </PopoverContent>
              </Popover>
              <NotificationBell />
              <ThemeToggle className="h-11 w-11 sm:h-9 sm:w-9" />
              <Popover>
                <PopoverTrigger asChild>
                  <button className="flex h-11 items-center gap-2 rounded-lg px-2 text-sm text-muted-foreground transition-colors duration-150 hover:bg-muted hover:text-foreground cursor-pointer sm:h-9" aria-label="打开账户菜单">
                    <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10">
                      <User size={12} className="text-primary" aria-hidden="true" />
                    </div>
                    <span className="hidden max-w-28 truncate text-xs font-medium sm:inline">{userIdentifier}</span>
                    <ChevronDown size={12} className="hidden sm:block" aria-hidden="true" />
                  </button>
                </PopoverTrigger>
                <PopoverContent align="end" className="w-52 p-1.5">
                  <div className="border-b border-border px-3 py-2.5">
                    <div className="text-xs text-muted-foreground">当前账户</div>
                    <div className="mt-0.5 truncate text-sm font-medium text-foreground" title={userIdentifier}>{userIdentifier}</div>
                  </div>
                  {auth.isAdmin && (
                    <button onClick={() => goTo("/admin")} className="flex min-h-10 w-full items-center gap-2.5 rounded-lg px-3 text-sm text-foreground hover:bg-muted cursor-pointer">
                      <Settings size={14} aria-hidden="true" />
                      管理后台
                    </button>
                  )}
                  <button onClick={handleLogout} className="flex min-h-10 w-full items-center gap-2.5 rounded-lg px-3 text-sm text-destructive hover:bg-destructive/10 cursor-pointer">
                    <LogOut size={14} aria-hidden="true" />
                    退出登录
                  </button>
                </PopoverContent>
              </Popover>
            </div>
          </header>

          <div className="border-b border-border/50"><AnnouncementBanner /></div>
          <main className="app-shell-gradient min-h-[calc(100vh-4rem)]">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
};

export default MainLayout;
