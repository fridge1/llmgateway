/**
 * React Query hooks for every API endpoint.
 *
 * Queries use `useQuery`, mutations use `useMutation` with cache invalidation.
 */
import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut, apiPatch, apiDelete } from "@/lib/api-client";
import { queryKeys } from "@/lib/query-keys";
import type {
  APIKey,
  Balance,
  BillingStats,
  TransactionsResponse,
  OrdersResponse,
  CreateKeyResponse,
  CreatePaymentResponse,
  GatewayModel,
  ModelPricing,
  PricingListResponse,
  ChatSession,
  ChatMessage,
  GatewayStatus,
  AdminUsersResponse,
  RechargeResponse,
  UpdatePricingRequest,
  InvoiceTitle,
  InvoiceRequest,
  InvoiceRequestsResponse,
  AdminInvoiceRequestsResponse,
  CompanyInfo,
  CreateInvoiceTitleRequest,
  CreateInvoiceRequestBody,
  InvoiceRequestOrder,
  Order,
  AdminOrdersResponse,
  AdminConsumptionStats,
  AdminFunnelStats,
  AnnouncementsResponse,
  Announcement,
  PublishedAnnouncementsResponse,
  PricingChangeLog,
  SubscriptionPlansResponse,
  SubscriptionPlan,
  AdminSubscriptionPlansResponse,
  SubscriptionCurrentResponse,
  SubscriptionHistoryResponse,
  SubscribeResponse,
  SubscriptionOrderStats,
  AdminSubscriptionOrdersResponse,
  APIKeyUsageSummary,
  AdminSubscriptionUsersUsageResponse,
  RechargePromotion,
  RechargePromotionInput,
  NotificationsResponse,
  UnreadCountResponse,
  TenantPricingListResponse,
  UserPricingListResponse,
  TenantModelUpstreamListResponse,
  TenantUpstreamInput,
  UserTokenStatsResponse,
  CheckinStatus,
  CheckinResult,
  TaskDefinition,
  ReferralInfo,
  RechargeLottery,
  RechargeLotteryRound,
  LotteryEvent,
  LotteryPrize,
  LotteryRecord,
  PublicLotteryRecord,
  ImageDurationStatsResponse,
} from "@/types/api";

// ---------------------------------------------------------------------------
// API Keys
// ---------------------------------------------------------------------------

export function useApiKeys(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.keys.all(),
    queryFn: () =>
      apiGet<{ keys: APIKey[] }>("/api/keys").then((r) => r.keys),
    enabled: options?.enabled ?? true,
  });
}

export function useCreateApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (params: { name: string; plan_id?: number }) =>
      apiPost<CreateKeyResponse>("/api/keys", params),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.keys.all() });
    },
  });
}

export function useDeleteApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/keys/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.keys.all() });
    },
  });
}

// ---------------------------------------------------------------------------
// Billing
// ---------------------------------------------------------------------------

export function useBalance() {
  return useQuery({
    queryKey: queryKeys.billing.balance(),
    queryFn: () => apiGet<Balance>("/api/billing/balance"),
    staleTime: 30_000,
    refetchOnWindowFocus: true,
  });
}

export function usePromotionRules() {
  return useQuery({
    queryKey: ["promotion", "rules"] as const,
    queryFn: () =>
      apiGet<{
        rules: {
          type: string;
          title: string;
          description: string;
          amount?: number;
          bonus_ratio?: number;
          starts_at?: string;
          ends_at?: string;
          min_recharge?: number;
        }[];
      }>("/api/promotion/rules"),
  });
}

export function useTransactions(page: number, size: number, type?: string, startDate?: string, endDate?: string) {
  return useQuery({
    queryKey: queryKeys.billing.transactions(page, size, type, startDate, endDate),
    queryFn: () => {
      let url = `/api/billing/transactions?page=${page}&size=${size}`;
      if (type) url += `&type=${type}`;
      if (startDate) url += `&start_date=${startDate}`;
      if (endDate) url += `&end_date=${endDate}`;
      return apiGet<TransactionsResponse>(url);
    },
    staleTime: 30_000,
  });
}

export function useBillingStats(days: number = 30) {
  return useQuery({
    queryKey: queryKeys.billing.stats(days),
    queryFn: () =>
      apiGet<BillingStats>(`/api/billing/stats?days=${days}`),
  });
}

export function useTokenStats(days: number = 7) {
  return useQuery({
    queryKey: queryKeys.billing.tokenStats(days),
    queryFn: () =>
      apiGet<UserTokenStatsResponse>(`/api/billing/token-stats?days=${days}`),
    staleTime: 30_000,
  });
}

export function useApiKeyUsage(days: number = 30) {
  return useQuery({
    queryKey: ["billing", "key-usage", days] as const,
    queryFn: () =>
      apiGet<APIKeyUsageSummary[]>(`/api/billing/key-usage?days=${days}`),
  });
}

// 获取当天（今日）API 密钥用量
export function useApiKeyUsageToday() {
  return useQuery({
    queryKey: ["billing", "key-usage", "today"] as const,
    queryFn: () =>
      apiGet<APIKeyUsageSummary[]>(`/api/billing/key-usage?days=0`),
    staleTime: 30_000, // 30秒缓存，因为当天数据变化较快
  });
}

export function useApiKeyTransactions(keyId: string, page: number, size: number) {
  return useQuery({
    queryKey: ["billing", "key-usage", keyId, "transactions", page, size] as const,
    queryFn: () =>
      apiGet<TransactionsResponse>(`/api/billing/key-usage/${keyId}/transactions?page=${page}&size=${size}`),
    enabled: !!keyId,
  });
}

// ---------------------------------------------------------------------------
// Pricing (public)
// ---------------------------------------------------------------------------

export function usePricing() {
  return useQuery({
    queryKey: queryKeys.pricing.public(),
    queryFn: () => apiGet<PricingListResponse>("/api/pricing"),
  });
}

// ---------------------------------------------------------------------------
// Payment / Orders
// ---------------------------------------------------------------------------

export function useCreatePayment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ amount, client_type }: { amount: number; client_type?: string }) =>
      apiPost<CreatePaymentResponse>("/api/payment/create", { amount, client_type }),
    onSuccess: () => {
      // Invalidate orders lists broadly
      qc.invalidateQueries({ queryKey: ["orders"] });
    },
  });
}

export function useRepayOrder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ order_no, client_type }: { order_no: string; client_type?: string }) =>
      apiPost<CreatePaymentResponse>("/api/payment/repay", { order_no, client_type }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["orders"] });
    },
  });
}

export function useOrders(page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.orders.all(page, size),
    queryFn: () =>
      apiGet<OrdersResponse>(
        `/api/payment/orders?page=${page}&size=${size}`,
      ),
  });
}

// ---------------------------------------------------------------------------
// Gateway Models
// ---------------------------------------------------------------------------

export function useGatewayModels(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: queryKeys.models.all(),
    queryFn: () => apiGet<GatewayModel[]>("/api/models"),
    enabled: options.enabled ?? true,
  });
}

export function useImageShareModels(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: ["image-share", "models"] as const,
    queryFn: () => apiGet<GatewayModel[]>("/api/image/models"),
    enabled: options.enabled ?? true,
  });
}

export function useCreateModel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; display_name: string; category: string; upstreams: unknown[] }) =>
      apiPost<GatewayModel>("/api/models", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.models.all() });
    },
  });
}

export function useUpdateModel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...body
    }: {
      id: number;
      name: string;
      display_name: string;
      category: string;
      upstreams: unknown[];
    }) => apiPut<GatewayModel>(`/api/models/${id}`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.models.all() });
    },
  });
}

export function useDeleteModel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiDelete(`/api/models/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.models.all() });
    },
  });
}

// ---------------------------------------------------------------------------
// Chat
// ---------------------------------------------------------------------------

export function useChatSessions() {
  return useQuery({
    queryKey: queryKeys.chat.sessions(),
    queryFn: () =>
      apiGet<{ sessions: ChatSession[] }>("/api/chat/sessions").then(
        (r) => r.sessions,
      ),
  });
}

export function useCreateChatSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { model: string; title?: string }) =>
      apiPost<ChatSession>("/api/chat/sessions", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.chat.sessions() });
    },
  });
}

export function useChatMessages(sessionId: string) {
  return useQuery({
    queryKey: queryKeys.chat.messages(sessionId),
    queryFn: () =>
      apiGet<{ messages: ChatMessage[] }>(
        `/api/chat/sessions/${sessionId}/messages`,
      ).then((r) => r.messages),
    enabled: !!sessionId,
  });
}

export function useSendChatMessage(sessionId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      role: string;
      content: string;
      tokens_used?: number;
      cost?: number;
      session_id?: string;
    }) => {
      const targetId = data.session_id || sessionId;
      const { session_id: _, ...body } = data;
      return apiPost(`/api/chat/sessions/${targetId}/messages`, body);
    },
    onSuccess: (_data, variables) => {
      const targetId = variables.session_id || sessionId;
      qc.invalidateQueries({
        queryKey: queryKeys.chat.messages(targetId),
      });
    },
  });
}

export function useUpdateChatSession(sessionId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { title: string }) =>
      apiPut(`/api/chat/sessions/${sessionId}`, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.chat.sessions() });
    },
  });
}

export function useDeleteChatSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/chat/sessions/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.chat.sessions() });
    },
  });
}

// ---------------------------------------------------------------------------
// Gateway Status & Config
// ---------------------------------------------------------------------------

export function useGatewayStatus(options?: { refetchInterval?: number | false }) {
  return useQuery({
    queryKey: queryKeys.status(),
    queryFn: () => apiGet<GatewayStatus>("/api/status"),
    refetchInterval: options?.refetchInterval ?? 30_000,
  });
}

export function useTestUpstreamByName() {
  return useMutation({
    mutationFn: (data: { model: string; base_url: string }) =>
      apiPost<{ success: boolean; message: string; latency?: string }>(
        "/api/admin/upstreams/test",
        data,
      ),
  });
}

export function useGatewayConfig() {
  return useQuery({
    queryKey: queryKeys.config(),
    queryFn: () => apiGet<Record<string, unknown>>("/api/config"),
  });
}

// ---------------------------------------------------------------------------
// Admin — Users
// ---------------------------------------------------------------------------

export function useAdminUsers(page: number, size: number, search: string) {
  return useQuery({
    queryKey: queryKeys.admin.users(page, size, search),
    queryFn: () =>
      apiGet<AdminUsersResponse>(
        `/api/admin/users?page=${page}&size=${size}&search=${encodeURIComponent(search)}`,
      ),
    placeholderData: keepPreviousData,
  });
}

export function useUpdateUserStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      apiPut(`/api/admin/users/${id}/status`, { status }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}

export function useToggleImageShare() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      apiPatch(`/api/admin/users/${id}/image-share`, { enabled }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}

export function useRechargeUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      amount,
      description,
    }: {
      id: string;
      amount: number;
      description?: string;
    }) =>
      apiPost<RechargeResponse>(`/api/admin/users/${id}/recharge`, {
        amount,
        description,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
      qc.invalidateQueries({ queryKey: queryKeys.billing.balance() });
    },
  });
}

export function useDeleteUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/admin/users/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}

export function useAdminUserTransactions(
  userId: string,
  page: number,
  size: number,
  startDate?: string,
  endDate?: string,
  type?: string
) {
  return useQuery({
    queryKey: queryKeys.admin.userTransactions(userId, page, size, startDate, endDate, type),
    queryFn: () => {
      const params = new URLSearchParams({
        page: String(page),
        size: String(size),
      });
      if (startDate) params.set("start_date", startDate);
      if (endDate) params.set("end_date", endDate);
      if (type) params.set("type", type);
      return apiGet<TransactionsResponse>(`/api/admin/users/${userId}/transactions?${params}`);
    },
    placeholderData: keepPreviousData,
  });
}

export function useAdminUserConsumptionStats(userId: string, days: number) {
  return useQuery({
    queryKey: queryKeys.admin.userConsumptionStats(userId, days),
    queryFn: () =>
      apiGet<AdminConsumptionStats>(`/api/admin/users/${userId}/consumption-stats?days=${days}`),
  });
}

export async function exportAdminUserTransactions(
  userId: string,
  startDate?: string,
  endDate?: string
) {
  const params = new URLSearchParams();
  if (startDate) params.set("start_date", startDate);
  if (endDate) params.set("end_date", endDate);
  const url = `/api/admin/users/${userId}/transactions/export?${params}`;
  const res = await fetch(url, { credentials: "include" });
  if (!res.ok) throw new Error("Export failed");
  const blob = await res.blob();
  const link = document.createElement("a");
  link.href = window.URL.createObjectURL(blob);
  const filename = res.headers.get("Content-Disposition")?.match(/filename="(.+)"/)?.[1] ?? "transactions.xlsx";
  link.download = filename;
  link.click();
}

// ---------------------------------------------------------------------------
// Admin — Pricing
// ---------------------------------------------------------------------------

export function useAdminPricing() {
  return useQuery({
    queryKey: queryKeys.pricing.admin(),
    queryFn: () => apiGet<PricingListResponse>("/api/admin/pricing"),
  });
}

export function useAdminPricingChangeLogs(page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.pricing.changeLogs(page, size),
    queryFn: () =>
      apiGet<{ logs: PricingChangeLog[]; total: number; page: number; size: number }>(
        `/api/admin/pricing/change-logs?page=${page}&size=${size}`,
      ),
  });
}

// ---------------------------------------------------------------------------
// Admin — Dashboard
// ---------------------------------------------------------------------------

export function useAdminDashboard() {
  return useQuery({
    queryKey: queryKeys.admin.dashboard(),
    queryFn: () =>
      apiGet<{ total_users: number; today_revenue: number; today_requests: number }>(
        "/api/admin/dashboard",
      ),
  });
}

export function useAdminOrders(page: number, size: number, status: string) {
  return useQuery({
    queryKey: queryKeys.admin.orders(page, size, status),
    queryFn: () =>
      apiGet<AdminOrdersResponse>(
        `/api/admin/orders?page=${page}&size=${size}&status=${encodeURIComponent(status)}`,
      ),
  });
}

export function useAdminConsumptionStats(days: number = 30) {
  return useQuery({
    queryKey: queryKeys.admin.consumptionStats(days),
    queryFn: () =>
      apiGet<AdminConsumptionStats>(
        `/api/admin/consumption-stats?days=${days}`,
      ),
  });
}

export function useAdminFunnelStats(days: number = 30) {
  return useQuery({
    queryKey: queryKeys.admin.funnelStats(days),
    queryFn: () =>
      apiGet<AdminFunnelStats>(
        `/api/admin/funnel-stats?days=${days}`,
      ),
  });
}

export function useAdminImageDurationStats(days: number = 30) {
  return useQuery({
    queryKey: queryKeys.admin.imageDurationStats(days),
    queryFn: () =>
      apiGet<ImageDurationStatsResponse>(
        `/api/admin/image-duration-stats?days=${days}`,
      ),
  });
}

export function useUpdateAdminPricing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      model,
      ...body
    }: UpdatePricingRequest & { model: string }) =>
      apiPut(`/api/admin/pricing/${encodeURIComponent(model)}`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.pricing.admin() });
      qc.invalidateQueries({ queryKey: queryKeys.pricing.public() });
    },
  });
}

// ---------------------------------------------------------------------------
// Invoice Titles
// ---------------------------------------------------------------------------

export function useInvoiceTitles() {
  return useQuery({
    queryKey: queryKeys.invoice.titles(),
    queryFn: () =>
      apiGet<{ titles: InvoiceTitle[] }>("/api/invoice/titles").then((r) => r.titles),
  });
}

export function useCreateInvoiceTitle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateInvoiceTitleRequest) =>
      apiPost<InvoiceTitle>("/api/invoice/titles", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.invoice.titles() });
    },
  });
}

export function useUpdateInvoiceTitle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: CreateInvoiceTitleRequest & { id: number }) =>
      apiPut<InvoiceTitle>(`/api/invoice/titles/${id}`, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.invoice.titles() });
    },
  });
}

export function useDeleteInvoiceTitle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiDelete(`/api/invoice/titles/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.invoice.titles() });
    },
  });
}

export function useSetDefaultInvoiceTitle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) =>
      apiPut(`/api/invoice/titles/${id}/default`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.invoice.titles() });
    },
  });
}

// ---------------------------------------------------------------------------
// Company Search
// ---------------------------------------------------------------------------

export function useCompanySearch(keyword: string) {
  return useQuery({
    queryKey: queryKeys.invoice.companySearch(keyword),
    queryFn: () =>
      apiGet<{ companies: CompanyInfo[] }>(
        `/api/invoice/company/search?keyword=${encodeURIComponent(keyword)}`,
      ).then((r) => r.companies),
    enabled: keyword.length >= 2,
  });
}

// ---------------------------------------------------------------------------
// Invoice Requests (User)
// ---------------------------------------------------------------------------

export function useAvailableOrders() {
  return useQuery({
    queryKey: queryKeys.invoice.availableOrders(),
    queryFn: () =>
      apiGet<{ orders: Order[] }>("/api/invoice/available-orders").then((r) => r.orders),
  });
}

export function useCreateInvoiceRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateInvoiceRequestBody) =>
      apiPost<InvoiceRequest>("/api/invoice/requests", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["invoice"] });
    },
  });
}

export function useInvoiceRequests(page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.invoice.requests(page, size),
    queryFn: () =>
      apiGet<InvoiceRequestsResponse>(
        `/api/invoice/requests?page=${page}&size=${size}`,
      ),
  });
}

export function useInvoiceRequestDetail(id: number) {
  return useQuery({
    queryKey: queryKeys.invoice.requestDetail(id),
    queryFn: () =>
      apiGet<{ request: InvoiceRequest; orders: InvoiceRequestOrder[]; title: InvoiceTitle }>(
        `/api/invoice/requests/${id}`,
      ),
    enabled: id > 0,
  });
}

export function useCancelInvoiceRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) =>
      apiPut(`/api/invoice/requests/${id}/cancel`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["invoice"] });
    },
  });
}

// ---------------------------------------------------------------------------
// Invoice Admin
// ---------------------------------------------------------------------------

export function useAdminInvoiceRequests(page: number, size: number, status: string) {
  return useQuery({
    queryKey: queryKeys.invoice.adminRequests(page, size, status),
    queryFn: () => {
      let url = `/api/admin/invoice/requests?page=${page}&size=${size}`;
      if (status) url += `&status=${status}`;
      return apiGet<AdminInvoiceRequestsResponse>(url);
    },
  });
}

export function useAdminInvoiceRequestDetail(id: number) {
  return useQuery({
    queryKey: queryKeys.invoice.adminRequestDetail(id),
    queryFn: () =>
      apiGet<{ request: InvoiceRequest; orders: InvoiceRequestOrder[]; title: InvoiceTitle; user: { id: string; phone: string } }>(
        `/api/admin/invoice/requests/${id}`,
      ),
    enabled: id > 0,
  });
}

export function useAdminProcessInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) =>
      apiPut(`/api/admin/invoice/requests/${id}/process`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["invoice"] });
    },
  });
}

export function useAdminRejectInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      apiPut(`/api/admin/invoice/requests/${id}/reject`, { reason }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["invoice"] });
    },
  });
}

export function useAdminCompleteInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, file, invoiceNumber }: { id: number; file: File; invoiceNumber: string }) => {
      const formData = new FormData();
      formData.append("file", file);
      formData.append("invoice_number", invoiceNumber);
      const res = await fetch(`/api/admin/invoice/requests/${id}/complete`, {
        method: "PUT",
        credentials: "include",
        body: formData,
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err?.error?.message || "upload failed");
      }
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["invoice"] });
    },
  });
}

// ---------------------------------------------------------------------------
// Announcements (Admin)
// ---------------------------------------------------------------------------

export function useAdminAnnouncements(page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.admin.announcements(page, size),
    queryFn: () =>
      apiGet<AnnouncementsResponse>(
        `/api/admin/announcements?page=${page}&size=${size}`,
      ),
  });
}

export function useCreateAnnouncement() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      title: string;
      content: string;
      status: string;
      priority: string;
      display_mode: string;
    }) => apiPost<Announcement>("/api/admin/announcements", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "announcements"] });
    },
  });
}

export function useUpdateAnnouncement() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...body
    }: {
      id: number;
      title?: string;
      content?: string;
      status?: string;
      priority?: string;
      display_mode?: string;
    }) => apiPut<Announcement>(`/api/admin/announcements/${id}`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "announcements"] });
    },
  });
}

export function useDeleteAnnouncement() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) =>
      apiDelete(`/api/admin/announcements/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "announcements"] });
    },
  });
}

// ---------------------------------------------------------------------------
// Announcements (User-facing)
// ---------------------------------------------------------------------------

export function usePublishedAnnouncements() {
  return useQuery({
    queryKey: queryKeys.announcements.published(),
    queryFn: () =>
      apiGet<PublishedAnnouncementsResponse>("/api/announcements"),
  });
}

// ---------------------------------------------------------------------------
// Subscription
// ---------------------------------------------------------------------------

export function useSubscriptionPlans() {
  return useQuery({
    queryKey: queryKeys.subscription.plans(),
    queryFn: () =>
      apiGet<SubscriptionPlansResponse>("/api/subscription/plans"),
  });
}

export function useSubscriptionCurrent() {
  return useQuery({
    queryKey: queryKeys.subscription.current(),
    queryFn: () =>
      apiGet<SubscriptionCurrentResponse>("/api/subscription/current"),
  });
}

export function useSubscriptionHistory() {
  return useQuery({
    queryKey: queryKeys.subscription.history(),
    queryFn: () =>
      apiGet<SubscriptionHistoryResponse>("/api/subscription/history").then(
        (r) => r.items,
      ),
  });
}

export type SubscriptionPlanInput = Omit<SubscriptionPlan, "id">;

export function useAdminSubscriptionPlans() {
  return useQuery({
    queryKey: queryKeys.subscription.adminPlans(),
    queryFn: () =>
      apiGet<AdminSubscriptionPlansResponse>("/api/admin/subscription-plans").then(
        (r) => r.plans,
      ),
  });
}

export function useCreateSubscriptionPlan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: SubscriptionPlanInput) =>
      apiPost<{ plan: SubscriptionPlan }>("/api/admin/subscription-plans", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.subscription.adminPlans() });
    },
  });
}

export function useUpdateSubscriptionPlan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...body }: SubscriptionPlanInput & { id: number }) =>
      apiPut<{ status: string }>(`/api/admin/subscription-plans/${id}`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.subscription.adminPlans() });
    },
  });
}

export function useDeleteSubscriptionPlan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiDelete(`/api/admin/subscription-plans/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.subscription.adminPlans() });
    },
  });
}

export function useSubscribe() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (params: { planId: number; clientType?: string }) =>
      apiPost<SubscribeResponse>("/api/subscription/subscribe", {
        plan_id: params.planId,
        client_type: params.clientType,
      }),
    onSuccess: (data) => {
      if (!data.need_payment) {
        qc.invalidateQueries({ queryKey: queryKeys.subscription.current() });
        qc.invalidateQueries({ queryKey: queryKeys.subscription.history() });
      }
    },
  });
}

export function useCreateSubscriptionPayment() {
  return useMutation({
    mutationFn: (params: { planId: number; clientType?: string }) =>
      apiPost<{ order_no: string; pay_url: string; expired_at: string }>(
        "/api/subscription/create-payment",
        { plan_id: params.planId, client_type: params.clientType },
      ),
  });
}

export function useUpgradeSubscription() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (planId: number) =>
      apiPost<{ price_diff: number }>("/api/subscription/upgrade", { plan_id: planId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.subscription.current() });
    },
  });
}

export function useCancelSubscription() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (brand?: string) => apiPost<{ status: string }>("/api/subscription/cancel", { brand: brand ?? "" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.subscription.current() });
    },
  });
}

export function useResumeSubscription() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (brand?: string) => apiPost<{ status: string }>("/api/subscription/resume", { brand: brand ?? "" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.subscription.current() });
    },
  });
}

// ---------------------------------------------------------------------------
// Admin — Subscription Orders
// ---------------------------------------------------------------------------

export function useAdminSubscriptionOrderStats(days: number) {
  return useQuery({
    queryKey: queryKeys.admin.subscriptionOrderStats(days),
    queryFn: () =>
      apiGet<SubscriptionOrderStats>(`/api/admin/subscription-orders/stats?days=${days}`),
  });
}

export function useAdminSubscriptionOrders(page: number, size: number, status: string, type: string) {
  return useQuery({
    queryKey: queryKeys.admin.subscriptionOrders(page, size, status, type),
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), size: String(size) });
      if (status) params.set("status", status);
      if (type) params.set("type", type);
      return apiGet<AdminSubscriptionOrdersResponse>(`/api/admin/subscription-orders?${params}`);
    },
  });
}

export function useAdminSubscriptionUsersUsage(
  page: number,
  size: number,
  search: string,
  status: string,
  planId: string
) {
  return useQuery({
    queryKey: queryKeys.admin.subscriptionUsersUsage(page, size, search, status, planId),
    queryFn: () => {
      const params = new URLSearchParams({ page: String(page), size: String(size) });
      if (search) params.set("search", search);
      if (status) params.set("status", status);
      if (planId) params.set("plan_id", planId);
      return apiGet<AdminSubscriptionUsersUsageResponse>(
        `/api/admin/subscription-users-usage?${params}`
      );
    },
  });
}

// ---------------------------------------------------------------------------
// Recharge promotions (admin)
// ---------------------------------------------------------------------------

export function useAdminRechargePromotions() {
  return useQuery({
    queryKey: queryKeys.rechargePromotions.all(),
    queryFn: () => apiGet<{ promotions: RechargePromotion[] }>("/api/admin/recharge-promotions"),
  });
}

export function useAdminCreateRechargePromotion() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: RechargePromotionInput) =>
      apiPost<RechargePromotion>("/api/admin/recharge-promotions", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.rechargePromotions.all() });
    },
  });
}

export function useAdminUpdateRechargePromotion() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: RechargePromotionInput & { id: number }) =>
      apiPut<RechargePromotion>(`/api/admin/recharge-promotions/${id}`, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.rechargePromotions.all() });
    },
  });
}

export function useAdminDeleteRechargePromotion() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiDelete(`/api/admin/recharge-promotions/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.rechargePromotions.all() });
    },
  });
}

// ---------------------------------------------------------------------------
// Notifications
// ---------------------------------------------------------------------------

export function useNotifications(page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.notifications.list(page, size),
    queryFn: () =>
      apiGet<NotificationsResponse>(
        `/api/notifications?page=${page}&size=${size}`,
      ),
  });
}

export function useUnreadNotificationCount() {
  return useQuery({
    queryKey: queryKeys.notifications.unreadCount(),
    queryFn: () =>
      apiGet<UnreadCountResponse>("/api/notifications/unread-count"),
    refetchInterval: 30_000,
  });
}

export function useMarkNotificationRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) =>
      apiPut(`/api/notifications/${id}/read`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.notifications.all() });
    },
  });
}

export function useMarkAllNotificationsRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiPut("/api/notifications/read-all"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.notifications.all() });
    },
  });
}

// ---------------------------------------------------------------------------
// Admin Tenants
// ---------------------------------------------------------------------------

export interface AdminTenant {
  id: string;
  name: string;
  status: string;
  created_at: string;
}

export interface AdminTenantsResponse {
  tenants: AdminTenant[];
  total: number;
  limit: number;
  offset: number;
}

export interface TenantBalanceResponse {
  tenant_id: string;
  balance: number;
  frozen: number;
}

export interface TenantTransaction {
  id: string;
  tenant_id: string;
  type: string;
  amount: number;
  balance_after: number;
  model: string;
  description: string;
  created_at: string;
}

export interface TenantTransactionsResponse {
  transactions: TenantTransaction[];
  total: number;
}

export function useAdminTenants(page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.admin.tenants(page, size),
    queryFn: () =>
      apiGet<AdminTenantsResponse>(
        `/api/admin/tenants?limit=${size}&offset=${(page - 1) * size}`,
      ),
    placeholderData: keepPreviousData,
  });
}

export function useAdminTenantBalance(tenantId: string) {
  return useQuery({
    queryKey: queryKeys.admin.tenantBalance(tenantId),
    queryFn: () =>
      apiGet<TenantBalanceResponse>(`/api/admin/tenants/${tenantId}/balance`),
    enabled: !!tenantId,
  });
}

export function useAdminTenantTransactions(tenantId: string, page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.admin.tenantTransactions(tenantId, page, size),
    queryFn: () =>
      apiGet<TenantTransactionsResponse>(
        `/api/admin/tenants/${tenantId}/transactions?limit=${size}&offset=${(page - 1) * size}`,
      ),
    enabled: !!tenantId,
    placeholderData: keepPreviousData,
  });
}

export function useAdminRechargeTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { tenantId: string; amount: number; description?: string }) =>
      apiPost(`/api/admin/tenants/${data.tenantId}/recharge`, {
        amount: data.amount,
        description: data.description,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "tenants"] });
    },
  });
}

export function useAdminCreateTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; owner_id: string; contact_phone?: string; contact_email?: string }) =>
      apiPost("/api/admin/tenants/enterprise", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "tenants"] });
    },
  });
}

export function useAdminDeleteTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (tenantId: string) =>
      apiDelete(`/api/admin/tenants/${tenantId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "tenants"] });
    },
  });
}

// ---------------------------------------------------------------------------
// Tenant Pricing
// ---------------------------------------------------------------------------

export function useAdminTenantPricing(tenantId: string) {
  return useQuery({
    queryKey: queryKeys.admin.tenantPricing(tenantId),
    queryFn: () =>
      apiGet<TenantPricingListResponse>(`/api/admin/tenants/${tenantId}/pricing`),
    enabled: !!tenantId,
  });
}

export function useAdminUpsertTenantPricing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      tenantId: string;
      modelName: string;
      pricing: UpdatePricingRequest;
    }) =>
      apiPut(
        `/api/admin/tenants/${data.tenantId}/pricing/${encodeURIComponent(data.modelName)}`,
        data.pricing,
      ),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: queryKeys.admin.tenantPricing(variables.tenantId),
      });
    },
  });
}

export function useAdminDeleteTenantPricing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { tenantId: string; modelName: string }) =>
      apiDelete(
        `/api/admin/tenants/${data.tenantId}/pricing/${encodeURIComponent(data.modelName)}`,
      ),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: queryKeys.admin.tenantPricing(variables.tenantId),
      });
    },
  });
}

// ---------------------------------------------------------------------------
// User Pricing
// ---------------------------------------------------------------------------

export function useAdminUserPricing(userId: string) {
  return useQuery({
    queryKey: queryKeys.admin.userPricing(userId),
    queryFn: () =>
      apiGet<UserPricingListResponse>(`/api/admin/users/${userId}/pricing`),
    enabled: !!userId,
  });
}

export function useAdminUpsertUserPricing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      userId: string;
      modelName: string;
      pricing: UpdatePricingRequest;
    }) =>
      apiPut(
        `/api/admin/users/${data.userId}/pricing/${encodeURIComponent(data.modelName)}`,
        data.pricing,
      ),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: queryKeys.admin.userPricing(variables.userId),
      });
    },
  });
}

export function useAdminDeleteUserPricing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { userId: string; modelName: string }) =>
      apiDelete(
        `/api/admin/users/${data.userId}/pricing/${encodeURIComponent(data.modelName)}`,
      ),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: queryKeys.admin.userPricing(variables.userId),
      });
    },
  });
}

// ---------------------------------------------------------------------------
// Tenant Model Upstream Overrides
// ---------------------------------------------------------------------------

export function useAdminTenantModelUpstreams(tenantId: string) {
  return useQuery({
    queryKey: queryKeys.admin.tenantModelUpstreams(tenantId),
    queryFn: () =>
      apiGet<TenantModelUpstreamListResponse>(
        `/api/admin/tenants/${tenantId}/model-upstreams`,
      ),
    enabled: !!tenantId,
  });
}

export function useAdminReplaceTenantModelUpstreams() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      tenantId: string;
      modelName: string;
      upstreams: TenantUpstreamInput[];
    }) =>
      apiPut(
        `/api/admin/tenants/${data.tenantId}/model-upstreams/${encodeURIComponent(data.modelName)}`,
        { upstreams: data.upstreams },
      ),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: queryKeys.admin.tenantModelUpstreams(variables.tenantId),
      });
    },
  });
}

export function useAdminDeleteTenantModelUpstreams() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { tenantId: string; modelName: string }) =>
      apiDelete(
        `/api/admin/tenants/${data.tenantId}/model-upstreams/${encodeURIComponent(data.modelName)}`,
      ),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: queryKeys.admin.tenantModelUpstreams(variables.tenantId),
      });
    },
  });
}

// --- Check-in ---
export function useCheckinStatus() {
  return useQuery({
    queryKey: queryKeys.checkin.status(),
    queryFn: () => apiGet<CheckinStatus>("/api/checkin/status"),
    staleTime: 60_000,
  });
}

export function useCheckin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiPost<CheckinResult>("/api/checkin"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.checkin.status() });
      qc.invalidateQueries({ queryKey: queryKeys.billing.balance() });
    },
  });
}

// --- Growth Tasks ---
export function useTasks() {
  return useQuery({
    queryKey: queryKeys.tasks.all(),
    queryFn: () => apiGet<{ tasks: TaskDefinition[] }>("/api/tasks"),
  });
}

export function useClaimTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (code: string) => apiPost<{ reward_cny: number; reward_lottery_draws: number }>(`/api/tasks/${code}/claim`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks.all() });
      qc.invalidateQueries({ queryKey: queryKeys.billing.balance() });
    },
  });
}

// --- Referral ---
export function useReferralInfo() {
  return useQuery({
    queryKey: queryKeys.referral.info(),
    queryFn: () => apiGet<ReferralInfo>("/api/referral"),
  });
}

// --- Recharge Lottery ---
export function useAdminRechargeLottery() {
  return useQuery({
    queryKey: queryKeys.rechargeLottery.config(),
    queryFn: () => apiGet<{ lottery: RechargeLottery | null }>("/api/admin/recharge-lottery"),
  });
}

export function useAdminCreateRechargeLottery() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; trigger_every: number }) =>
      apiPost<RechargeLottery>("/api/admin/recharge-lottery", data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: queryKeys.rechargeLottery.config() }); },
  });
}

export function useAdminUpdateRechargeLottery() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: { id: number; name: string; status: string; trigger_every: number }) =>
      apiPut<RechargeLottery>(`/api/admin/recharge-lottery/${id}`, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: queryKeys.rechargeLottery.config() }); },
  });
}

export function useAdminRechargeLotteryRounds(id: number, page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.rechargeLottery.rounds(id, page, size),
    queryFn: () => apiGet<{ rounds: RechargeLotteryRound[]; total: number }>(`/api/admin/recharge-lottery/${id}/rounds?page=${page}&size=${size}`),
    enabled: id > 0,
  });
}

export function usePublicRechargeLottery() {
  return useQuery({
    queryKey: queryKeys.rechargeLotteryPublic.config(),
    queryFn: () => apiGet<{ lottery: RechargeLottery | null; current_entries: number }>("/api/recharge-lottery"),
    staleTime: 60_000,
  });
}

export function usePublicRechargeLotteryRounds() {
  return useQuery({
    queryKey: queryKeys.rechargeLotteryPublic.rounds(),
    queryFn: () => apiGet<{ rounds: RechargeLotteryRound[] }>("/api/recharge-lottery/rounds"),
    staleTime: 30_000,
  });
}

// --- Lottery (automatic draw on recharge) ---

export function useAdminLotteryEvents(page = 1, size = 20) {
  return useQuery({
    queryKey: queryKeys.lottery.events(page, size),
    queryFn: () => apiGet<{ events: LotteryEvent[]; total: number }>(`/api/admin/lottery/events?page=${page}&size=${size}`),
  });
}

export function useAdminCreateLotteryEvent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; description: string; status: string; min_recharge_cny: number; min_order_count_to_draw: number; start_time?: string | null; end_time?: string | null }) =>
      apiPost<LotteryEvent>("/api/admin/lottery/events", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "lottery", "events"] });
    },
  });
}

export function useAdminUpdateLotteryEvent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: { id: number; name: string; description: string; status: string; min_recharge_cny: number; min_order_count_to_draw: number; start_time?: string | null; end_time?: string | null }) =>
      apiPut<LotteryEvent>(`/api/admin/lottery/events/${id}`, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "lottery", "events"] });
    },
  });
}

export function useAdminLotteryPrizes(eventId: number) {
  return useQuery({
    queryKey: queryKeys.lottery.prizes(eventId),
    queryFn: () => apiGet<{ prizes: LotteryPrize[] }>(`/api/admin/lottery/events/${eventId}/prizes`),
    enabled: eventId > 0,
  });
}

export function useAdminCreateLotteryPrize() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ eventId, ...data }: { eventId: number; name: string; description: string; weight: number; total_stock: number; prize_type: string; prize_value: number; sort_order: number }) =>
      apiPost<LotteryPrize>(`/api/admin/lottery/events/${eventId}/prizes`, data),
    onSuccess: (_, variables) => {
      qc.invalidateQueries({ queryKey: queryKeys.lottery.prizes(variables.eventId) });
    },
  });
}

export function useAdminUpdateLotteryPrize() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, eventId, ...data }: { id: number; eventId: number; name: string; description: string; weight: number; total_stock: number; prize_type: string; prize_value: number; sort_order: number }) =>
      apiPut<LotteryPrize>(`/api/admin/lottery/prizes/${id}`, data),
    onSuccess: (_, variables) => {
      qc.invalidateQueries({ queryKey: queryKeys.lottery.prizes(variables.eventId) });
    },
  });
}

export function useAdminDeleteLotteryPrize() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, eventId }: { id: number; eventId: number }) =>
      apiDelete(`/api/admin/lottery/prizes/${id}`),
    onSuccess: (_, variables) => {
      qc.invalidateQueries({ queryKey: queryKeys.lottery.prizes(variables.eventId) });
    },
  });
}

export function useAdminLotteryRecords(eventId: number, page = 1, size = 20) {
  return useQuery({
    queryKey: queryKeys.lottery.records(eventId, page, size),
    queryFn: () => apiGet<{ records: LotteryRecord[]; total: number }>(`/api/admin/lottery/events/${eventId}/records?page=${page}&size=${size}`),
    enabled: eventId > 0,
  });
}

export function useLotteryCurrentEvent() {
  return useQuery({
    queryKey: queryKeys.lottery.currentEvent(),
    queryFn: () => apiGet<{ event: LotteryEvent | null; prizes: LotteryPrize[] }>("/api/lottery/current"),
    staleTime: 60_000,
  });
}

export function useLotteryWinnerRecords(page = 1, size = 20) {
  return useQuery({
    queryKey: queryKeys.lottery.winnerRecords(page, size),
    queryFn: () => apiGet<{ records: PublicLotteryRecord[]; total: number }>(`/api/lottery/records?page=${page}&size=${size}`),
  });
}

// --- Ops alerting (admin) ---

export interface AlertRule {
  id: number;
  metric: string;
  display_name: string;
  threshold: number;
  cooldown_seconds: number;
  enabled: boolean;
  updated_at: string;
}

export interface AlertEvent {
  id: number;
  metric: string;
  message: string;
  value: number;
  threshold: number;
  created_at: string;
}

export function useAdminAlertRules() {
  return useQuery({
    queryKey: queryKeys.admin.alertRules(),
    queryFn: () => apiGet<{ rules: AlertRule[] }>("/api/admin/alert/rules"),
  });
}

export function useAdminUpdateAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { id: number; threshold: number; cooldown_seconds: number; enabled: boolean }) =>
      apiPut("/api/admin/alert/rules", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "alert"] }),
  });
}

export function useAdminAlertEvents(page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.admin.alertEvents(page, size),
    queryFn: () => apiGet<{ events: AlertEvent[]; total: number }>(`/api/admin/alert/events?page=${page}&size=${size}`),
    refetchInterval: 30_000,
  });
}

// --- Content moderation (admin) ---

export interface ModerationSettings {
  enabled: boolean;
  enforce_all: boolean;
  updated_at: string;
}

export interface ModerationKeyword {
  id: number;
  keyword: string;
  category: string;
  enabled: boolean;
  created_at: string;
}

export interface ModerationHit {
  id: number;
  user_id: string | null;
  tenant_id: string | null;
  model: string;
  matched_rule: string;
  snippet: string;
  created_at: string;
}

export function useAdminModerationSettings() {
  return useQuery({
    queryKey: queryKeys.admin.moderationSettings(),
    queryFn: () => apiGet<ModerationSettings>("/api/admin/moderation/settings"),
  });
}

export function useAdminUpdateModerationSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { enabled: boolean; enforce_all: boolean }) =>
      apiPut("/api/admin/moderation/settings", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "moderation"] }),
  });
}

export function useAdminModerationKeywords() {
  return useQuery({
    queryKey: queryKeys.admin.moderationKeywords(),
    queryFn: () => apiGet<{ keywords: ModerationKeyword[] }>("/api/admin/moderation/keywords"),
  });
}

export function useAdminCreateModerationKeyword() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { keyword: string; category: string }) =>
      apiPost("/api/admin/moderation/keywords", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "moderation", "keywords"] }),
  });
}

export function useAdminDeleteModerationKeyword() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiDelete(`/api/admin/moderation/keywords/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "moderation", "keywords"] }),
  });
}

export function useAdminModerationHits(page: number, size: number, userId: string, from: string, to: string) {
  const params = new URLSearchParams({ page: String(page), size: String(size) });
  if (userId) params.set("user_id", userId);
  if (from) params.set("from", from);
  if (to) params.set("to", to);
  return useQuery({
    queryKey: queryKeys.admin.moderationHits(page, size, userId, from, to),
    queryFn: () => apiGet<{ hits: ModerationHit[]; total: number }>(`/api/admin/moderation/hits?${params}`),
  });
}

// --- Support tickets ---

export interface Ticket {
  id: string;
  user_id: string;
  user_identifier?: string;
  category: string;
  subject: string;
  status: "open" | "pending" | "resolved" | "closed";
  related_order_no: string | null;
  created_at: string;
  updated_at: string;
}

export interface TicketMessage {
  id: number;
  ticket_id: string;
  sender_role: "user" | "admin";
  sender_id: string | null;
  content: string;
  attachments: string[];
  created_at: string;
}

export interface TicketsResponse {
  tickets: Ticket[];
  total: number;
  page: number;
  size: number;
}

export interface TicketDetailResponse {
  ticket: Ticket;
  messages: TicketMessage[];
}

export function useTickets(page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.tickets.list(page, size),
    queryFn: () => apiGet<TicketsResponse>(`/api/tickets?page=${page}&size=${size}`),
    placeholderData: keepPreviousData,
  });
}

export function useTicketDetail(id: string) {
  return useQuery({
    queryKey: queryKeys.tickets.detail(id),
    queryFn: () => apiGet<TicketDetailResponse>(`/api/tickets/${id}`),
    enabled: id !== "",
  });
}

export function useCreateTicket() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      category: string;
      subject: string;
      content: string;
      related_order_no?: string;
      attachments?: string[];
    }) => apiPost<Ticket>("/api/tickets", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.tickets.all() });
    },
  });
}

export function useCreateTicketMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, content, attachments }: { id: string; content: string; attachments?: string[] }) =>
      apiPost<TicketMessage>(`/api/tickets/${id}/messages`, { content, attachments }),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: queryKeys.tickets.detail(id) });
      qc.invalidateQueries({ queryKey: queryKeys.tickets.all() });
    },
  });
}

// --- Support tickets (admin) ---

export function useAdminTickets(page: number, size: number, status: string) {
  const params = new URLSearchParams({ page: String(page), size: String(size) });
  if (status) params.set("status", status);
  return useQuery({
    queryKey: queryKeys.admin.tickets(page, size, status),
    queryFn: () => apiGet<TicketsResponse>(`/api/admin/tickets?${params}`),
    placeholderData: keepPreviousData,
  });
}

export function useAdminTicketDetail(id: string) {
  return useQuery({
    queryKey: queryKeys.admin.ticketDetail(id),
    queryFn: () => apiGet<TicketDetailResponse>(`/api/admin/tickets/${id}`),
    enabled: id !== "",
  });
}

export function useAdminReplyTicket() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, content }: { id: string; content: string }) =>
      apiPost<TicketMessage>(`/api/admin/tickets/${id}/reply`, { content }),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: queryKeys.admin.ticketDetail(id) });
      qc.invalidateQueries({ queryKey: ["admin", "tickets"] });
    },
  });
}

export function useAdminUpdateTicketStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      apiPut<{ status: string }>(`/api/admin/tickets/${id}/status`, { status }),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: queryKeys.admin.ticketDetail(id) });
      qc.invalidateQueries({ queryKey: ["admin", "tickets"] });
    },
  });
}

// --- Lottery draw (admin) ---

export interface DrawEventResult {
  winners: LotteryRecord[];
  count: number;
}

export function useAdminDrawEventLottery() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (eventId: number) =>
      apiPost<DrawEventResult>(`/api/admin/lottery/events/${eventId}/draw`, {}),
    onSuccess: (_, eventId) => {
      qc.invalidateQueries({ queryKey: queryKeys.lottery.records(eventId, 1, 20) });
      qc.invalidateQueries({ queryKey: queryKeys.lottery.events(1, 20) });
    },
  });
}

export function useLotteryMyRecords(page = 1, size = 20) {
  return useQuery({
    queryKey: queryKeys.lottery.myRecords(page, size),
    queryFn: () => apiGet<{ records: LotteryRecord[]; total: number }>(`/api/lottery/my-records?page=${page}&size=${size}`),
  });
}

// --- IP blocking (admin) ---

export interface BlockedIP {
  ip_address: string;
  reason: string;
  blocked_at: string;
  expires_at: string | null;
  blocked_by: string | null;
  notes: string | null;
}

export function useAdminBlockedIPs(page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.admin.blockedIPs(page, size),
    queryFn: () =>
      apiGet<{ items: BlockedIP[]; limit: number; offset: number }>(
        `/api/admin/blocked-ips?limit=${size}&offset=${(page - 1) * size}`,
      ),
  });
}

export function useAdminBlockIP() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { ip_address: string; reason: string; expires_in_days?: number; notes?: string }) =>
      apiPost("/api/admin/blocked-ips", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "blocked-ips"] }),
  });
}

export function useAdminUnblockIP() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ip: string) => apiDelete(`/api/admin/blocked-ips/${encodeURIComponent(ip)}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "blocked-ips"] }),
  });
}
