import { useState } from "react";
import { Navigate, Outlet, useNavigate, useLocation } from "react-router-dom";
import { Building2, Key, ArrowLeftRight, BookOpen, LogOut, Cpu, Menu } from "lucide-react";
import { useSubUserAuth } from "@/contexts/SubUserAuthContext";
import { ThemeToggle, NavIconButton } from "@/components/ui/nav-atoms";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";

const SubUserLayout = () => {
  const auth = useSubUserAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileOpen, setMobileOpen] = useState(false);

  if (auth.isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  if (!auth.isAuthenticated) {
    return <Navigate to="/org/login" replace />;
  }

  const navItems = [
    { key: "/org", label: "概览", icon: Building2, exact: true },
    { key: "/org/models", label: "可用模型", icon: Cpu },
    { key: "/org/keys", label: "API 密钥", icon: Key },
    { key: "/org/transactions", label: "使用记录", icon: ArrowLeftRight },
    { key: "/org/docs", label: "API 文档", icon: BookOpen },
  ];

  const isActive = (key: string, exact?: boolean) => {
    if (exact) return location.pathname === key;
    return location.pathname.startsWith(key);
  };

  const handleLogout = async () => {
    await auth.logout();
    navigate("/org/login");
  };

  const renderNavigation = (horizontal = false) => (
    <div className={horizontal ? "flex items-center justify-center gap-1" : "space-y-1"}>
      {navItems.map((item) => {
        const Icon = item.icon;
        const active = isActive(item.key, item.exact);
        return (
          <button
            key={item.key}
            onClick={() => { navigate(item.key); setMobileOpen(false); }}
            className={`flex min-h-10 items-center gap-3 rounded-lg px-3 text-sm font-medium transition-colors duration-150 cursor-pointer ${horizontal ? "w-auto" : "w-full"} ${
              active ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted/70 hover:text-foreground"
            }`}
          >
            <Icon size={16} />
            {item.label}
          </button>
        );
      })}
    </div>
  );

  const quotaText = auth.user?.quota_limit != null
    ? `${auth.user.quota_remaining?.toFixed(2) ?? "0.00"} / ${auth.user.quota_limit.toFixed(2)} 元`
    : "无限额度";

  return (
    <div className="min-h-screen bg-background">
      <nav className="sticky top-0 z-50 flex h-16 items-center justify-between border-b border-border/70 bg-card/80 px-4 glass sm:px-6">
        <div className="flex min-w-0 items-center gap-3">
          <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
            <SheetTrigger asChild>
              <button className="admin-menu-button lg:hidden" aria-label="打开组织面板导航">
                <Menu size={18} />
              </button>
            </SheetTrigger>
            <SheetContent side="left" className="w-[280px] p-0">
              <SheetTitle className="sr-only">组织面板导航</SheetTitle>
              <div className="flex items-center gap-3 border-b border-border/70 px-5 py-5">
                <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-sky-500 shadow-button">
                  <Building2 size={16} className="text-white" />
                </div>
                <div>
                  <div className="text-sm font-bold tracking-tight text-foreground">组织面板</div>
                  <div className="text-[11px] text-muted-foreground">LLM Gateway Organization</div>
                </div>
              </div>
              <div className="px-3 py-4">{renderNavigation()}</div>
            </SheetContent>
          </Sheet>
          <div className="flex min-w-0 items-center gap-2.5 cursor-pointer select-none" onClick={() => navigate("/org")}>
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-sky-500 shadow-button">
              <Building2 size={15} className="text-white" />
            </div>
            <div className="min-w-0">
              <div className="truncate text-sm font-bold tracking-tight text-foreground">组织面板</div>
              <div className="hidden text-[11px] text-muted-foreground sm:block">{location.pathname === "/org" ? "组织概览" : "成员工作区"}</div>
            </div>
          </div>
        </div>

        <div className="hidden flex-1 items-center justify-center gap-1 lg:flex">{renderNavigation(true)}</div>

        <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
          <div className="hidden text-xs text-muted-foreground xl:block">
            额度: <span className="font-medium text-foreground">{quotaText}</span>
          </div>
          <ThemeToggle />
          <div className="hidden h-4 w-px bg-border sm:block" />
          <span className="hidden max-w-24 truncate text-sm font-medium text-foreground sm:inline">{auth.user?.nickname || auth.user?.username}</span>
          <NavIconButton icon={LogOut} onClick={handleLogout} title="退出登录" hover="destructive" />
        </div>
      </nav>

      <main className="app-shell-gradient min-h-[calc(100vh-4rem)]">
        <div className="mx-auto w-full max-w-6xl px-4 py-5 sm:px-6 sm:py-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
};

export default SubUserLayout;
