/**
 * Centralised React Query key factory.
 *
 * Convention: each domain exposes an `all` tuple (for broad invalidation)
 * and narrower keys for individual queries.
 */
export const queryKeys = {
  // --- Auth ---
  me: () => ["me"] as const,

  // --- API Keys ---
  keys: {
    all: () => ["keys"] as const,
  },

  // --- Billing ---
  billing: {
    balance: () => ["billing", "balance"] as const,
    transactions: (page: number, size: number, type?: string, startDate?: string, endDate?: string) =>
      ["billing", "transactions", { page, size, type, startDate, endDate }] as const,
    stats: (days: number) => ["billing", "stats", { days }] as const,
    tokenStats: (days: number) => ["billing", "token-stats", { days }] as const,
  },

  // --- Pricing ---
  pricing: {
    public: () => ["pricing"] as const,
    admin: () => ["admin", "pricing"] as const,
    changeLogs: (page: number, size: number) =>
      ["admin", "pricing", "change-logs", { page, size }] as const,
  },

  // --- Payment / Orders ---
  orders: {
    all: (page: number, size: number) =>
      ["orders", { page, size }] as const,
  },

  // --- Gateway Models ---
  models: {
    all: () => ["models"] as const,
  },

  // --- Chat ---
  chat: {
    sessions: () => ["chat", "sessions"] as const,
    messages: (sessionId: string) =>
      ["chat", "messages", sessionId] as const,
  },

  // --- Gateway Status ---
  status: () => ["status"] as const,

  // --- Config ---
  config: () => ["config"] as const,

  // --- Invoice ---
  invoice: {
    titles: () => ["invoice", "titles"] as const,
    availableOrders: () => ["invoice", "available-orders"] as const,
    requests: (page: number, size: number) =>
      ["invoice", "requests", { page, size }] as const,
    requestDetail: (id: number) =>
      ["invoice", "requests", id] as const,
    companySearch: (keyword: string) =>
      ["invoice", "company", keyword] as const,
    adminRequests: (page: number, size: number, status: string) =>
      ["invoice", "admin", "requests", { page, size, status }] as const,
    adminRequestDetail: (id: number) =>
      ["invoice", "admin", "requests", id] as const,
  },

  // --- Admin ---
  admin: {
    users: (page: number, size: number, search: string) =>
      ["admin", "users", { page, size, search }] as const,
    userTransactions: (userId: string, page: number, size: number, startDate?: string, endDate?: string, type?: string) =>
      ["admin", "users", userId, "transactions", { page, size, startDate, endDate, type }] as const,
    userConsumptionStats: (userId: string, days: number) =>
      ["admin", "users", userId, "consumption-stats", { days }] as const,
    dashboard: () => ["admin", "dashboard"] as const,
    alertRules: () => ["admin", "alert", "rules"] as const,
    alertEvents: (page: number, size: number) =>
      ["admin", "alert", "events", { page, size }] as const,
    moderationSettings: () => ["admin", "moderation", "settings"] as const,
    moderationKeywords: () => ["admin", "moderation", "keywords"] as const,
    moderationHits: (page: number, size: number, userId: string, from: string, to: string) =>
      ["admin", "moderation", "hits", { page, size, userId, from, to }] as const,
    orders: (page: number, size: number, status: string) =>
      ["admin", "orders", { page, size, status }] as const,
    consumptionStats: (days: number) =>
      ["admin", "consumption-stats", { days }] as const,
    funnelStats: (days: number) =>
      ["admin", "funnel-stats", { days }] as const,
    imageDurationStats: (days: number) =>
      ["admin", "image-duration-stats", { days }] as const,
    announcements: (page: number, size: number) =>
      ["admin", "announcements", { page, size }] as const,
    subscriptionOrderStats: (days: number) =>
      ["admin", "subscription-order-stats", { days }] as const,
    subscriptionOrders: (page: number, size: number, status: string, type: string) =>
      ["admin", "subscription-orders", { page, size, status, type }] as const,
    subscriptionUsersUsage: (page: number, size: number, search: string, status: string, planId: string) =>
      ["admin", "subscription-users-usage", { page, size, search, status, planId }] as const,
    tenants: (page: number, size: number) =>
      ["admin", "tenants", { page, size }] as const,
    tenantBalance: (id: string) =>
      ["admin", "tenants", id, "balance"] as const,
    tenantTransactions: (id: string, page: number, size: number) =>
      ["admin", "tenants", id, "transactions", { page, size }] as const,
    tenantPricing: (id: string) =>
      ["admin", "tenants", id, "pricing"] as const,
    tenantModelUpstreams: (id: string) =>
      ["admin", "tenants", id, "model-upstreams"] as const,
    userPricing: (id: string) =>
      ["admin", "users", id, "pricing"] as const,
    tickets: (page: number, size: number, status: string) =>
      ["admin", "tickets", { page, size, status }] as const,
    ticketDetail: (id: string) =>
      ["admin", "tickets", id] as const,
    blockedIPs: (page: number, size: number) =>
      ["admin", "blocked-ips", { page, size }] as const,
  },

  // --- Tickets (user-facing) ---
  tickets: {
    all: () => ["tickets"] as const,
    list: (page: number, size: number) =>
      ["tickets", "list", { page, size }] as const,
    detail: (id: string) => ["tickets", id] as const,
  },

  // --- Announcements (user-facing) ---
  announcements: {
    published: () => ["announcements", "published"] as const,
  },

  // --- Subscription ---
  subscription: {
    plans: () => ["subscription", "plans"] as const,
    current: () => ["subscription", "current"] as const,
    history: () => ["subscription", "history"] as const,
    adminPlans: () => ["admin", "subscription-plans"] as const,
  },

  // --- Recharge Promotions (admin) ---
  rechargePromotions: {
    all: () => ["admin", "recharge-promotions"] as const,
  },

  // --- Notifications ---
  notifications: {
    all: () => ["notifications"] as const,
    list: (page: number, size: number) =>
      ["notifications", "list", { page, size }] as const,
    unreadCount: () => ["notifications", "unread-count"] as const,
  },

  // --- Check-in ---
  checkin: {
    status: () => ["checkin", "status"] as const,
  },

  // --- Growth Tasks ---
  tasks: {
    all: () => ["tasks"] as const,
  },

  // --- Referral ---
  referral: {
    info: () => ["referral"] as const,
  },

  // --- Recharge Lottery ---
  rechargeLotteryPublic: {
    config: () => ["recharge-lottery"] as const,
    rounds: () => ["recharge-lottery", "rounds"] as const,
  },
  rechargeLottery: {
    config: () => ["admin", "recharge-lottery"] as const,
    rounds: (id: number, page: number, size: number) =>
      ["admin", "recharge-lottery", id, "rounds", { page, size }] as const,
  },

  // --- Lottery ---
  lottery: {
    events: (page: number, size: number) =>
      ["admin", "lottery", "events", { page, size }] as const,
    prizes: (eventId: number) =>
      ["admin", "lottery", "events", eventId, "prizes"] as const,
    records: (eventId: number, page: number, size: number) =>
      ["admin", "lottery", "events", eventId, "records", { page, size }] as const,
    currentEvent: () => ["lottery", "current"] as const,
    winnerRecords: (page: number, size: number) =>
      ["lottery", "winner-records", { page, size }] as const,
    myRecords: (page: number, size: number) =>
      ["lottery", "my-records", { page, size }] as const,
  },
} as const;
