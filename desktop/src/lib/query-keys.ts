export const queryKeys = {
  me: () => ["me"] as const,

  keys: {
    all: () => ["keys"] as const,
    usage: (days: number) => ["keys", "usage", { days }] as const,
    transactions: (keyId: string, page: number, size: number) =>
      ["keys", "transactions", keyId, { page, size }] as const,
  },

  billing: {
    balance: () => ["billing", "balance"] as const,
    transactions: (page: number, size: number, type?: string, startDate?: string, endDate?: string) =>
      ["billing", "transactions", { page, size, type, startDate, endDate }] as const,
    stats: (days: number) => ["billing", "stats", { days }] as const,
    tokenStats: (days: number) => ["billing", "token-stats", { days }] as const,
  },

  pricing: {
    public: () => ["pricing"] as const,
  },

  orders: {
    all: (page: number, size: number) =>
      ["orders", { page, size }] as const,
  },

  models: {
    all: () => ["models"] as const,
  },

  imageShare: {
    models: () => ["image-share", "models"] as const,
  },

  image: {
    tasks: (limit: number, offset: number) =>
      ["image-tasks", limit, offset] as const,
  },

  chat: {
    sessions: () => ["chat", "sessions"] as const,
    messages: (sessionId: string) => ["chat", "messages", sessionId] as const,
  },

  config: () => ["config"] as const,

  invoice: {
    titles: () => ["invoice", "titles"] as const,
    availableOrders: () => ["invoice", "available-orders"] as const,
    requests: (page: number, size: number) =>
      ["invoice", "requests", { page, size }] as const,
    requestDetail: (id: number) =>
      ["invoice", "requests", id] as const,
    companySearch: (keyword: string) =>
      ["invoice", "company", keyword] as const,
  },

  announcements: {
    published: () => ["announcements", "published"] as const,
  },

  subscription: {
    plans: () => ["subscription", "plans"] as const,
    current: () => ["subscription", "current"] as const,
    history: () => ["subscription", "history"] as const,
  },

  notifications: {
    all: () => ["notifications"] as const,
    list: (page: number, size: number) =>
      ["notifications", "list", { page, size }] as const,
    unreadCount: () => ["notifications", "unread-count"] as const,
  },

  promotion: {
    rules: () => ["promotion", "rules"] as const,
  },

  checkin: {
    status: () => ["checkin", "status"] as const,
  },

  tasks: {
    all: () => ["tasks"] as const,
  },

  referral: {
    info: () => ["referral"] as const,
  },

  tickets: {
    all: () => ["tickets"] as const,
    list: (page: number, size: number) =>
      ["tickets", "list", { page, size }] as const,
    detail: (id: string) => ["tickets", "detail", id] as const,
  },

  tenants: {
    all: () => ["tenants"] as const,
    detail: (id: string) => ["tenants", id] as const,
    members: (id: string) => ["tenants", id, "members"] as const,
    keys: (id: string) => ["tenants", id, "keys"] as const,
    balance: (id: string) => ["tenants", id, "balance"] as const,
    transactions: (id: string, page: number, size: number) =>
      ["tenants", id, "transactions", { page, size }] as const,
    analytics: (id: string, days: number) =>
      ["tenants", id, "analytics", { days }] as const,
    subUserTransactions: (id: string, subUserId: string, page: number, size: number) =>
      ["tenants", id, "sub-users", subUserId, "transactions", { page, size }] as const,
    usageRecords: (id: string, page: number, size: number) =>
      ["tenants", id, "usage-records", { page, size }] as const,
    invitations: () => ["tenants", "invitations"] as const,
  },
} as const;
