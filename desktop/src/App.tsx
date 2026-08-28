import { useState, useEffect, useCallback } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { motion } from "motion/react";
import { Toaster } from "sonner";
import { useAuth } from "./hooks/useAuth";
import { api } from "./lib/tauri";
import { Gem, AlertCircle } from "./components/icons";
import LoginPage from "./pages/LoginPage";
import Layout from "./components/Layout";
import DashboardPage from "./pages/DashboardPage";
import KeysPage from "./pages/KeysPage";
import UsagePage from "./pages/UsagePage";
import ModelsPage from "./pages/ModelsPage";
import BalancePage from "./pages/BalancePage";
import TransactionsPage from "./pages/TransactionsPage";
import OrdersPage from "./pages/OrdersPage";
import SubscriptionPage from "./pages/SubscriptionPage";
import GrowthCenterPage from "./pages/GrowthCenterPage";
import TicketsPage from "./pages/TicketsPage";
import InvoiceTitlesPage from "./pages/InvoiceTitlesPage";
import InvoiceRequestsPage from "./pages/InvoiceRequestsPage";
import TenantListPage from "./pages/tenant/TenantListPage";
import TenantDashboard from "./pages/tenant/TenantDashboard";
import TenantMembersPage from "./pages/tenant/TenantMembersPage";
import TenantKeysPage from "./pages/tenant/TenantKeysPage";
import TenantBillingPage from "./pages/tenant/TenantBillingPage";
import TenantUsageRecordsPage from "./pages/tenant/TenantUsageRecordsPage";
import TenantAnalyticsPage from "./pages/tenant/TenantAnalyticsPage";
import TenantSettingsPage from "./pages/tenant/TenantSettingsPage";
import TenantSubUserTransactionsPage from "./pages/tenant/TenantSubUserTransactionsPage";
import PlaygroundPage from "./pages/PlaygroundPage";
import TranslatePage from "./pages/TranslatePage";
import ImagePage from "./pages/ImagePage";
import PptPage from "./pages/PptPage";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: true,
    },
  },
});

function AppInner() {
  const { user, loading, login, register, sendSmsCode, resetPassword, logout } = useAuth();
  const [authExpiredMsg, setAuthExpiredMsg] = useState<string | null>(null);

  useEffect(() => {
    if (!user) return;
    const checkBalance = async () => {
      try {
        const data = await api.getBalance();
        if (data.balance < 10) {
          const { isPermissionGranted, requestPermission, sendNotification } = await import("@tauri-apps/plugin-notification");
          let permitted = await isPermissionGranted();
          if (!permitted) permitted = (await requestPermission()) === "granted";
          if (permitted) {
            sendNotification({ title: "LLM Gateway", body: `您的余额低于 ¥10（当前 ¥${data.balance.toFixed(2)}），请及时充值` });
          }
        }
      } catch (err) {
        const msg = String(err);
        if (msg.includes("登录已过期") || msg.includes("Unauthorized")) {
          setAuthExpiredMsg("登录已过期，请重新登录");
        }
      }
    };
    const interval = setInterval(checkBalance, 30 * 60 * 1000);
    return () => clearInterval(interval);
  }, [user]);

  useEffect(() => {
    let unlisten: (() => void) | undefined;
    (async () => {
      const { listen } = await import("@tauri-apps/api/event");
      const u1 = await listen("tray-logout", () => {
        handleLogout();
      });
      unlisten = u1;
    })();
    return () => { unlisten?.(); };
  }, []);

  const handleLogout = useCallback(async () => {
    const clearConfig = window.confirm("是否同时清除已配置的 AI 工具配置？");
    if (clearConfig) {
      try {
        const result = await api.scanTools();
        for (const tool of result.tools.filter(t => t.configured)) {
          await api.clearToolConfig(tool.tool);
        }
      } catch {}
    }
    await logout();
    setAuthExpiredMsg(null);
  }, [logout]);

  const handleExpiredDismiss = useCallback(async () => {
    await logout();
    setAuthExpiredMsg(null);
  }, [logout]);

  if (loading) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-obsidian-950 bg-noise">
        <div className="relative z-10 flex flex-col items-center">
          <Gem size={32} className="text-amber-400 animate-pulse" />
          <p className="text-sm text-obsidian-400 mt-3">LLM Gateway</p>
          <div className="flex gap-1 mt-4">
            {[0, 1, 2].map(i => (
              <div
                key={i}
                className="w-1.5 h-1.5 rounded-full bg-amber-400"
                style={{ animation: "pulse-subtle 1.5s ease-in-out infinite", animationDelay: `${i * 0.2}s` }}
              />
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (!user) {
    return <LoginPage onLogin={async (phone, password, remember) => { await login(phone, password, remember); }} onRegister={async (phone, code, password, adminToken) => { await register(phone, code, password, adminToken); }} onSendSmsCode={sendSmsCode} onResetPassword={resetPassword} />;
  }

  return (
    <>
      <MemoryRouter>
        <Routes>
          <Route element={<Layout phone={user.phone} onLogout={handleLogout} />}>
            <Route index element={<DashboardPage />} />
            <Route path="keys" element={<KeysPage />} />
            <Route path="usage" element={<UsagePage />} />
            <Route path="models" element={<ModelsPage />} />
            <Route path="balance" element={<BalancePage />} />
            <Route path="transactions" element={<TransactionsPage />} />
            <Route path="orders" element={<OrdersPage />} />
            <Route path="subscription" element={<SubscriptionPage />} />
            <Route path="growth" element={<GrowthCenterPage />} />
            <Route path="tickets" element={<TicketsPage />} />
            <Route path="invoice/titles" element={<InvoiceTitlesPage />} />
            <Route path="invoice/requests" element={<InvoiceRequestsPage />} />
            <Route path="tenants" element={<TenantListPage />} />
            <Route path="tenants/:id" element={<TenantDashboard />} />
            <Route path="tenants/:id/members" element={<TenantMembersPage />} />
            <Route path="tenants/:id/members/:subUserId/transactions" element={<TenantSubUserTransactionsPage />} />
            <Route path="tenants/:id/keys" element={<TenantKeysPage />} />
            <Route path="tenants/:id/billing" element={<TenantBillingPage />} />
            <Route path="tenants/:id/usage" element={<TenantUsageRecordsPage />} />
            <Route path="tenants/:id/analytics" element={<TenantAnalyticsPage />} />
            <Route path="tenants/:id/settings" element={<TenantSettingsPage />} />
            <Route path="tools/chat" element={<PlaygroundPage />} />
            <Route path="tools/translate" element={<TranslatePage />} />
            <Route path="tools/image" element={<ImagePage />} />
            <Route path="tools/ppt" element={<PptPage />} />
          </Route>
        </Routes>
      </MemoryRouter>
      {authExpiredMsg && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50">
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-6 max-w-sm mx-4 shadow-card-hover"
          >
            <div className="flex flex-col items-center text-center">
              <AlertCircle size={32} className="text-amber-400 mb-3" />
              <h3 className="font-semibold text-obsidian-50 mb-2">登录已过期</h3>
              <p className="text-obsidian-400 text-sm mb-4">{authExpiredMsg}</p>
              <button
                onClick={handleExpiredDismiss}
                className="w-full py-2.5 bg-amber-500 hover:bg-amber-400 hover:shadow-glow-amber text-obsidian-950 rounded-lg font-semibold transition-all duration-200"
              >
                重新登录
              </button>
            </div>
          </motion.div>
        </div>
      )}
    </>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppInner />
      <Toaster position="top-center" theme="dark" richColors />
    </QueryClientProvider>
  );
}

export default App;
