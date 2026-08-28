import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut, apiDelete } from "@/lib/api-client";
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
  PricingListResponse,
  InvoiceTitle,
  InvoiceRequest,
  InvoiceRequestsResponse,
  CompanyInfo,
  CreateInvoiceTitleRequest,
  CreateInvoiceRequestBody,
  InvoiceRequestOrder,
  Order,
  PublishedAnnouncementsResponse,
  SubscriptionPlansResponse,
  SubscriptionCurrentResponse,
  SubscriptionHistoryResponse,
  SubscribeResponse,
  APIKeyUsageSummary,
  NotificationsResponse,
  UnreadCountResponse,
  PromotionRulesResponse,
  UserTokenStatsResponse,
  ChatSession,
  ChatMessage,
  CheckinStatus,
  CheckinResult,
  TaskDefinition,
  ReferralInfo,
  Ticket,
  TicketMessage,
  TicketsResponse,
  TicketDetailResponse,
} from "@/lib/types-api";

// ---------------------------------------------------------------------------
// API Keys
// ---------------------------------------------------------------------------

export function useApiKeys() {
  return useQuery({
    queryKey: queryKeys.keys.all(),
    queryFn: () =>
      apiGet<{ keys: APIKey[] }>("/api/keys").then((r) => r.keys),
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
  });
}

export function usePromotionRules() {
  return useQuery({
    queryKey: queryKeys.promotion.rules(),
    queryFn: () => apiGet<PromotionRulesResponse>("/api/promotion/rules"),
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
  });
}

export function useBillingStats(days: number = 30) {
  return useQuery({
    queryKey: queryKeys.billing.stats(days),
    queryFn: () => apiGet<BillingStats>(`/api/billing/stats?days=${days}`),
  });
}

export function useTokenStats(days: number = 7) {
  return useQuery({
    queryKey: queryKeys.billing.tokenStats(days),
    queryFn: () => apiGet<UserTokenStatsResponse>(`/api/billing/token-stats?days=${days}`),
    staleTime: 30_000,
  });
}

export function useApiKeyUsage(days: number = 30) {
  return useQuery({
    queryKey: queryKeys.keys.usage(days),
    queryFn: () => apiGet<APIKeyUsageSummary[]>(`/api/billing/key-usage?days=${days}`),
  });
}

export function useApiKeyTransactions(keyId: string, page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.keys.transactions(keyId, page, size),
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
      apiGet<OrdersResponse>(`/api/payment/orders?page=${page}&size=${size}`),
  });
}

// ---------------------------------------------------------------------------
// Gateway Models
// ---------------------------------------------------------------------------

export function useGatewayModels() {
  return useQuery({
    queryKey: queryKeys.models.all(),
    queryFn: () => apiGet<GatewayModel[]>("/api/models"),
  });
}

export function useImageShareModels(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: queryKeys.imageShare.models(),
    queryFn: () => apiGet<GatewayModel[]>("/api/image/models"),
    enabled: options.enabled ?? true,
  });
}

// ---------------------------------------------------------------------------
// Chat / Playground
// ---------------------------------------------------------------------------

export function useChatSessions() {
  return useQuery({
    queryKey: queryKeys.chat.sessions(),
    queryFn: () =>
      apiGet<{ sessions: ChatSession[] }>("/api/chat/sessions").then((r) => r.sessions),
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
      apiGet<{ messages: ChatMessage[] }>(`/api/chat/sessions/${sessionId}/messages`).then(
        (r) => r.messages,
      ),
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
      const { session_id: _ignored, ...body } = data;
      return apiPost(`/api/chat/sessions/${targetId}/messages`, body);
    },
    onSuccess: (_data, variables) => {
      const targetId = variables.session_id || sessionId;
      qc.invalidateQueries({ queryKey: queryKeys.chat.messages(targetId) });
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
    mutationFn: (id: number) => apiPut(`/api/invoice/titles/${id}/default`),
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
// Invoice Requests
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
      apiGet<InvoiceRequestsResponse>(`/api/invoice/requests?page=${page}&size=${size}`),
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
    mutationFn: (id: number) => apiPut(`/api/invoice/requests/${id}/cancel`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["invoice"] });
    },
  });
}

// ---------------------------------------------------------------------------
// Announcements (User-facing)
// ---------------------------------------------------------------------------

export function usePublishedAnnouncements() {
  return useQuery({
    queryKey: queryKeys.announcements.published(),
    queryFn: () => apiGet<PublishedAnnouncementsResponse>("/api/announcements"),
  });
}

// ---------------------------------------------------------------------------
// Subscription
// ---------------------------------------------------------------------------

export function useSubscriptionPlans() {
  return useQuery({
    queryKey: queryKeys.subscription.plans(),
    queryFn: () => apiGet<SubscriptionPlansResponse>("/api/subscription/plans"),
  });
}

export function useSubscriptionCurrent() {
  return useQuery({
    queryKey: queryKeys.subscription.current(),
    queryFn: () => apiGet<SubscriptionCurrentResponse>("/api/subscription/current"),
  });
}

export function useSubscriptionHistory() {
  return useQuery({
    queryKey: queryKeys.subscription.history(),
    queryFn: () =>
      apiGet<SubscriptionHistoryResponse>("/api/subscription/history").then((r) => r.items),
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


// ---------------------------------------------------------------------------
// Notifications
// ---------------------------------------------------------------------------

export function useNotifications(page: number, size: number) {
  return useQuery({
    queryKey: queryKeys.notifications.list(page, size),
    queryFn: () => apiGet<NotificationsResponse>(`/api/notifications?page=${page}&size=${size}`),
  });
}

export function useUnreadNotificationCount() {
  return useQuery({
    queryKey: queryKeys.notifications.unreadCount(),
    queryFn: () => apiGet<UnreadCountResponse>("/api/notifications/unread-count"),
    refetchInterval: 30_000,
  });
}

export function useMarkNotificationRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiPut(`/api/notifications/${id}/read`),
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
// Check-in
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Growth Tasks
// ---------------------------------------------------------------------------

export function useTasks() {
  return useQuery({
    queryKey: queryKeys.tasks.all(),
    queryFn: () => apiGet<{ tasks: TaskDefinition[] }>("/api/tasks"),
  });
}

export function useClaimTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (code: string) =>
      apiPost<{ reward_cny: number; reward_lottery_draws: number }>(`/api/tasks/${code}/claim`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks.all() });
      qc.invalidateQueries({ queryKey: queryKeys.billing.balance() });
    },
  });
}

// ---------------------------------------------------------------------------
// Referral
// ---------------------------------------------------------------------------

export function useReferralInfo() {
  return useQuery({
    queryKey: queryKeys.referral.info(),
    queryFn: () => apiGet<ReferralInfo>("/api/referral"),
  });
}

// ---------------------------------------------------------------------------
// Support Tickets
// ---------------------------------------------------------------------------

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
