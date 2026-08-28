import { useNavigate, useLocation, Outlet } from "react-router-dom";
import { motion } from "motion/react";
import { MessageSquare, Image, Gift, LifeBuoy } from "lucide-react";
import type { LucideIcon } from "./icons";
import { LayoutDashboard, Key, BarChart3, Cpu, LogOut, Gem, User, Wallet, FileText, CreditCard, Crown, Building2, ChevronDown } from "./icons";
import PageTransition from "./PageTransition";
import ErrorBoundary from "./ErrorBoundary";
import NotificationBell from "./NotificationBell";
import AnnouncementBanner from "./AnnouncementBanner";
import { useTenants } from "@/hooks/use-tenant";
import { useState } from "react";

interface NavItem {
  path: string;
  label: string;
  icon: LucideIcon;
}

interface NavSection {
  label: string;
  items: NavItem[];
  collapsible?: boolean;
  condition?: boolean;
}

interface Props {
  phone: string;
  onLogout: () => void;
}

const CORE_NAV: NavSection[] = [
  {
    label: "核心",
    items: [
      { path: "/", label: "概览", icon: LayoutDashboard },
      { path: "/keys", label: "API Key", icon: Key },
      { path: "/usage", label: "用量统计", icon: BarChart3 },
      { path: "/models", label: "模型列表", icon: Cpu },
    ],
  },
  {
    label: "工具",
    collapsible: true,
    items: [
      { path: "/tools/chat", label: "对话", icon: MessageSquare as LucideIcon },
      { path: "/tools/image", label: "图片生成", icon: Image as LucideIcon },
    ],
  },
  {
    label: "财务",
    collapsible: true,
    items: [
      { path: "/balance", label: "余额充值", icon: Wallet },
      { path: "/transactions", label: "交易记录", icon: CreditCard },
      { path: "/orders", label: "订单", icon: FileText },
      { path: "/subscription", label: "订阅套餐", icon: Crown },
      { path: "/growth", label: "成长中心", icon: Gift as LucideIcon },
    ],
  },
  {
    label: "发票",
    collapsible: true,
    items: [
      { path: "/invoice/titles", label: "发票抬头", icon: FileText },
      { path: "/invoice/requests", label: "我的发票", icon: FileText },
    ],
  },
  {
    label: "支持",
    collapsible: true,
    items: [
      { path: "/tickets", label: "工单支持", icon: LifeBuoy as LucideIcon },
    ],
  },
];

const TENANT_SECTION: NavSection = {
  label: "组织",
  collapsible: true,
  items: [
    { path: "/tenants", label: "我的组织", icon: Building2 },
  ],
};

export default function Layout({ phone, onLogout }: Props) {
  const navigate = useNavigate();
  const location = useLocation();
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const { data: tenants } = useTenants();
  const hasTenants = (tenants ?? []).length > 0;

  const sections = hasTenants ? [...CORE_NAV, TENANT_SECTION] : CORE_NAV;

  const toggleSection = (label: string) => {
    setCollapsed(prev => ({ ...prev, [label]: !prev[label] }));
  };

  const isActive = (path: string) => {
    if (path === "/") return location.pathname === "/";
    return location.pathname.startsWith(path);
  };

  return (
    <div className="h-screen bg-obsidian-950 text-white flex overflow-hidden">
      {/* Sidebar */}
      <div className="w-52 shrink-0 border-r border-obsidian-700 bg-obsidian-900 flex flex-col h-full">
        <div className="p-4 border-b border-obsidian-700">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Gem size={18} className="text-amber-400" />
              <span className="text-sm font-semibold text-obsidian-50">LLM Gateway</span>
            </div>
            <NotificationBell />
          </div>
          <div className="flex items-center gap-2 mt-2">
            <div className="w-7 h-7 rounded-full bg-obsidian-800 flex items-center justify-center">
              <User size={14} className="text-obsidian-400" />
            </div>
            <span className="text-xs text-obsidian-400 truncate">{phone}</span>
          </div>
        </div>
        <nav className="flex-1 p-2 overflow-y-auto">
          {sections.map(section => {
            const isCollapsed = collapsed[section.label];
            return (
              <div key={section.label} className="mb-2">
                {section.collapsible ? (
                  <button
                    onClick={() => toggleSection(section.label)}
                    className="w-full flex items-center justify-between px-3 py-1.5 text-[10px] uppercase tracking-wider text-obsidian-500 hover:text-obsidian-300 transition-colors"
                  >
                    {section.label}
                    <ChevronDown
                      size={12}
                      className={`transition-transform duration-200 ${isCollapsed ? "-rotate-90" : ""}`}
                    />
                  </button>
                ) : (
                  <div className="px-3 py-1.5 text-[10px] uppercase tracking-wider text-obsidian-500">
                    {section.label}
                  </div>
                )}
                {!isCollapsed && section.items.map(item => {
                  const Icon = item.icon;
                  const active = isActive(item.path);
                  return (
                    <button
                      key={item.path}
                      onClick={() => navigate(item.path)}
                      className={`w-full text-left px-3 py-2 rounded-lg text-sm mb-0.5 transition-colors relative flex items-center gap-2.5 ${
                        active
                          ? "bg-obsidian-800 text-obsidian-50"
                          : "text-obsidian-400 hover:bg-obsidian-800 hover:text-obsidian-200"
                      }`}
                    >
                      {active && (
                        <motion.div
                          layoutId="nav-indicator"
                          className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 bg-amber-400 rounded-full"
                        />
                      )}
                      <Icon size={16} className={active ? "text-amber-400" : ""} />
                      {item.label}
                    </button>
                  );
                })}
              </div>
            );
          })}
        </nav>
        <div className="p-2 border-t border-obsidian-700">
          <button
            onClick={onLogout}
            className="w-full text-left px-3 py-2 text-sm text-obsidian-500 hover:text-red-400 rounded-lg transition-colors flex items-center gap-2.5"
          >
            <LogOut size={16} />
            退出登录
          </button>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 overflow-y-auto bg-obsidian-950 bg-noise relative">
        <div className="relative z-10">
          <AnnouncementBanner />
          <PageTransition pageKey={location.pathname}>
            <ErrorBoundary>
              <Outlet />
            </ErrorBoundary>
          </PageTransition>
        </div>
      </div>
    </div>
  );
}
