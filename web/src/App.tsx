import { lazy, Suspense } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route, Navigate, useLocation } from "react-router-dom";
import { ThemeProvider } from "next-themes";
import { AuthProvider, useAuth } from "./contexts/AuthContext";
import { SubUserAuthProvider } from "./contexts/SubUserAuthContext";
import ProtectedRoute from "./components/ProtectedRoute";
import AdminRoute from "./components/AdminRoute";
import { isImageHost } from "@/lib/host";
import { PUBLIC_PAGE_META } from "@/seo-meta";
import { Toaster } from "@/components/ui/sonner";
import { Skeleton } from "@/components/ui/skeleton";

// Eagerly loaded (critical path)
import AuthPage from "./pages/AuthPage";
import HomePage from "./pages/HomePage";
import MainLayout from "./pages/Index";
import DashboardPage from "./pages/DashboardPage";

// Lazy loaded pages
const ApiKeysPage = lazy(() => import("./pages/ApiKeysPage"));
const ModelsPage = lazy(() => import("./pages/ModelsPage"));
const PlaygroundPage = lazy(() => import("./pages/PlaygroundPage"));
const TranslatePage = lazy(() => import("./pages/TranslatePage"));
const PptPage = lazy(() => import("./pages/PptPage"));
const ImagePage = lazy(() => import("./pages/ImagePage"));
const ToolsHubPage = lazy(() => import("./pages/ToolsHubPage"));
const BalancePage = lazy(() => import("./pages/BalancePage"));
const TransactionsPage = lazy(() => import("./pages/TransactionsPage"));
const TicketsPage = lazy(() => import("./pages/TicketsPage"));
const OrdersPage = lazy(() => import("./pages/OrdersPage"));
const InvoiceTitlesPage = lazy(() => import("./pages/InvoiceTitlesPage"));
const InvoiceRequestsPage = lazy(() => import("./pages/InvoiceRequestsPage"));
const DocsPage = lazy(() => import("./pages/DocsPage"));
const SubscriptionPage = lazy(() => import("./pages/SubscriptionPage"));
const ClaudeLandingPage = lazy(() => import("./pages/ClaudeLandingPage"));
const EnterprisePage = lazy(() => import("./pages/EnterprisePage"));
const DownloadPage = lazy(() => import("./pages/DownloadPage"));
const QualificationsPage = lazy(() => import("./pages/QualificationsPage"));
const OnboardingPage = lazy(() => import("./pages/OnboardingPage"));
const GrowthCenterPage = lazy(() => import("./pages/GrowthCenterPage"));
const NotificationsPage = lazy(() => import("./pages/NotificationsPage"));
const LotteryPage = lazy(() => import("./pages/LotteryPage"));
const AdminLayout = lazy(() => import("./pages/admin/AdminLayout"));
const SubUserLoginPage = lazy(() => import("./pages/sub-user/SubUserLoginPage"));
const SubUserLayout = lazy(() => import("./pages/sub-user/SubUserLayout"));
const SubUserDashboard = lazy(() => import("./pages/sub-user/SubUserDashboard"));
const SubUserKeysPage = lazy(() => import("./pages/sub-user/SubUserKeysPage"));
const SubUserModelsPage = lazy(() => import("./pages/sub-user/SubUserModelsPage"));
const SubUserTransactionsPage = lazy(() => import("./pages/sub-user/SubUserTransactionsPage"));
const SubUserDocsPage = lazy(() => import("./pages/sub-user/SubUserDocsPage"));
const TenantListPage = lazy(() => import("./pages/tenant/TenantListPage"));
const TenantDashboard = lazy(() => import("./pages/tenant/TenantDashboard"));
const TenantMembersPage = lazy(() => import("./pages/tenant/TenantMembersPage"));
const TenantKeysPage = lazy(() => import("./pages/tenant/TenantKeysPage"));
const TenantBillingPage = lazy(() => import("./pages/tenant/TenantBillingPage"));
const TenantSettingsPage = lazy(() => import("./pages/tenant/TenantSettingsPage"));
const TenantSubUserTransactionsPage = lazy(() => import("./pages/tenant/TenantSubUserTransactionsPage"));
const TenantUsageRecordsPage = lazy(() => import("./pages/tenant/TenantUsageRecordsPage"));
const TenantAnalyticsPage = lazy(() => import("./pages/tenant/TenantAnalyticsPage"));
const ImageShareLoginPage = lazy(() => import("./pages/ImageShareLoginPage"));
const ImageShareKeysPage = lazy(() => import("./pages/ImageShareKeysPage"));
const ImageShareLayout = lazy(() => import("./components/ImageShareLayout"));

// Codex pages
const CodexShopPage = lazy(() => import("./pages/CodexShopPage"));
const CodexOrderQueryPage = lazy(() => import("./pages/CodexOrderQueryPage"));
const AdminCodexOrders = lazy(() => import("./pages/admin/AdminCodexOrders"));

function PageLoader() {
  return (
    <div className="flex flex-col gap-4 p-6">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-3/4" />
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
        <Skeleton className="h-32" />
        <Skeleton className="h-32" />
        <Skeleton className="h-32" />
      </div>
    </div>
  );
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60_000, // 5 minutes for most data
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

/** Renders the image-share-only experience for keyholders. */
function ImageShareApp() {
  return (
    <ImageShareLayout>
      <Routes>
        <Route path="/image" element={<ImagePage />} />
        <Route path="*" element={<Navigate to="/image" replace />} />
      </Routes>
    </ImageShareLayout>
  );
}

/** Standalone image-share site served from image.your-domain.com. */
function ImageHostApp() {
  const auth = useAuth();
  if (auth.isImageShare) {
    return <ImageShareApp />;
  }
  return (
    <Routes>
      <Route path="/login" element={<ImageShareLoginPage />} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  );
}

// 公开营销/文档页：不依赖登录态，不等 auth 探测完成直接渲染。
// 既是真实用户的 FCP 优化，也是预渲染（scripts/prerender.mjs）的前提。
const PUBLIC_PATHS = new Set(Object.keys(PUBLIC_PAGE_META));

function AppRoutes() {
  const auth = useAuth();
  const location = useLocation();

  if (auth.isLoading && (isImageHost() || !PUBLIC_PATHS.has(location.pathname))) {
    return <PageLoader />;
  }

  if (isImageHost()) {
    return <ImageHostApp />;
  }

  // Image-share keyholders see a stripped-down app with only /image accessible.
  // The login page is the one allowed escape hatch (e.g. when re-entering after refresh).
  if (auth.isImageShare) {
    if (location.pathname === "/image-login") {
      return <Navigate to="/image" replace />;
    }
    return <ImageShareApp />;
  }

  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/login" element={<AuthPage />} />
      <Route path="/auth" element={<AuthPage />} />
      <Route path="/image-login" element={<ImageShareLoginPage />} />
      <Route path="/docs" element={<DocsPage />} />
      <Route path="/claude" element={<ClaudeLandingPage />} />
      <Route path="/enterprise" element={<EnterprisePage />} />
      <Route path="/download" element={<DownloadPage />} />
      <Route path="/qualifications" element={<QualificationsPage />} />
      <Route path="/onboarding" element={<ProtectedRoute><OnboardingPage /></ProtectedRoute>} />
      <Route path="/tools/chat" element={<ProtectedRoute><PlaygroundPage /></ProtectedRoute>} />
      <Route path="/tools/translate" element={<ProtectedRoute><TranslatePage /></ProtectedRoute>} />
      <Route path="/tools/ppt" element={<ProtectedRoute><PptPage /></ProtectedRoute>} />
      <Route path="/tools/image" element={<ProtectedRoute><ImagePage /></ProtectedRoute>} />
      <Route path="/playground" element={<Navigate to="/tools/chat" replace />} />
      <Route path="/dashboard" element={<ProtectedRoute><MainLayout /></ProtectedRoute>}>
        <Route index element={<DashboardPage />} />
        <Route path="keys" element={<ApiKeysPage />} />
        <Route path="models" element={<ModelsPage />} />
        <Route path="balance" element={<BalancePage />} />
        <Route path="transactions" element={<TransactionsPage />} />
        <Route path="tickets" element={<TicketsPage />} />
        <Route path="orders" element={<OrdersPage />} />
        <Route path="invoice/titles" element={<InvoiceTitlesPage />} />
        <Route path="invoice/requests" element={<InvoiceRequestsPage />} />
        <Route path="subscription" element={<SubscriptionPage />} />
        <Route path="tools" element={<ToolsHubPage />} />
        <Route path="growth" element={<GrowthCenterPage />} />
        <Route path="lottery" element={<LotteryPage />} />
        <Route path="notifications" element={<NotificationsPage />} />
        <Route path="image-share" element={<ImageShareKeysPage />} />
        {/* Tenant routes */}
        <Route path="tenants" element={<TenantListPage />} />
        <Route path="tenants/:id" element={<TenantDashboard />} />
        <Route path="tenants/:id/members" element={<TenantMembersPage />} />
        <Route path="tenants/:id/members/:subUserId/transactions" element={<TenantSubUserTransactionsPage />} />
        <Route path="tenants/:id/keys" element={<TenantKeysPage />} />
        <Route path="tenants/:id/billing" element={<TenantBillingPage />} />
        <Route path="tenants/:id/usage" element={<TenantUsageRecordsPage />} />
        <Route path="tenants/:id/analytics" element={<TenantAnalyticsPage />} />
        <Route path="tenants/:id/settings" element={<TenantSettingsPage />} />
      </Route>
      <Route path="/admin/*" element={<AdminRoute><AdminLayout /></AdminRoute>} />

      {/* Codex 公开路由（无需认证） */}
      <Route path="/codex-shop" element={<CodexShopPage />} />
      <Route path="/codex-order-query" element={<CodexOrderQueryPage />} />

      {/* Sub-user (organization) routes */}
      <Route path="/org/login" element={<SubUserAuthProvider><SubUserLoginPage /></SubUserAuthProvider>} />
      <Route path="/org" element={<SubUserAuthProvider><SubUserLayout /></SubUserAuthProvider>}>
        <Route index element={<SubUserDashboard />} />
        <Route path="keys" element={<SubUserKeysPage />} />
        <Route path="models" element={<SubUserModelsPage />} />
        <Route path="transactions" element={<SubUserTransactionsPage />} />
        <Route path="docs" element={<SubUserDocsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

/** 路由器以内的应用主体；预渲染（entry-server）用 StaticRouter 包同一棵树。 */
export function AppShell() {
  return (
    <>
      <Toaster richColors position="top-center" />
      <Suspense fallback={<PageLoader />}>
        <AppRoutes />
      </Suspense>
    </>
  );
}

/** 路由器以外的全部 Provider；客户端与预渲染共用。 */
export function AppProviders({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>{children}</AuthProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}

const App = () => (
  <AppProviders>
    <BrowserRouter>
      <AppShell />
    </BrowserRouter>
  </AppProviders>
);

export default App;
