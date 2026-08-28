export interface User {
  id: string;
  phone: string;
  nickname?: string;
  role: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface MeResponse {
  user_id: string;
  phone: string;
  role: string;
  first_recharge_bonus_cny?: number;
  first_recharge_bonus_granted?: boolean;
  image_share_enabled?: boolean;
}

export interface Balance {
  user_id: string;
  balance: number;
  frozen: number;
  updated_at: string;
}

export interface Transaction {
  id: string;
  user_id: string;
  type: string;
  amount: number;
  balance_after: number;
  model?: string;
  description?: string;
  request_id?: string;
  prompt_tokens?: number;
  completion_tokens?: number;
  cache_read_tokens?: number;
  cache_creation_tokens?: number;
  cache_creation_5m_tokens?: number;
  cache_creation_1h_tokens?: number;
  subscription_id?: string;
  api_key_id?: string;
  created_at: string;
}

export interface APIKeyUsageSummary {
  key_id: string;
  key_name: string;
  key_prefix: string;
  total_cost: number;
  request_count: number;
  is_deleted: boolean;
}

export interface DailyCost {
  date: string;
  cost: number;
}

export interface ModelCost {
  model: string;
  cost: number;
}

export interface BillingStats {
  today_cost: number;
  month_cost: number;
  daily_average: number;
  daily_trend: DailyCost[];
  model_breakdown: ModelCost[];
}

export interface APIKey {
  id: string;
  user_id: string;
  key_prefix: string;
  name: string;
  status: string;
  plan_id?: number;
  last_used_at?: string;
  created_at: string;
}

export interface Order {
  id: string;
  user_id: string;
  order_no: string;
  amount: number;
  pay_method: string;
  status: string;
  pay_time?: string;
  created_at: string;
  expired_at: string;
}

export interface GatewayModel {
  id: number;
  name: string;
  display_name: string;
  category: string;
  upstreams: Upstream[];
  created_at: string;
  updated_at: string;
}

export interface Upstream {
  id: number;
  model_id: number;
  provider: string;
  protocol: string;
  upstream_provider: string;
  upstream_name: string;
  base_url: string;
  api_key: string;
  model_override: string;
  weight: number;
  sort_order: number;
}

export interface PricingTier {
  min_tokens: number;
  max_tokens: number;
  input_price: number;
  output_price: number;
  cached_input_price: number;
}

export interface ModelPricing {
  id: number;
  model_name: string;
  input_price: number;
  output_price: number;
  cached_input_price: number;
  cache_creation_price: number;
  cache_creation_1h_price: number;
  billing_type: string;
  is_active: boolean;
  updated_at: string;
  pricing_tiers?: PricingTier[];
  /** Tenant discount rate (0,1]; present only when caller's tenant has a discount. */
  discount_rate?: number;
  /** Original global pricing for struck-through comparison; present only under discount. */
  original_pricing?: ModelPricing;
}

export interface PricingListResponse {
  pricing: ModelPricing[];
}

export interface PaginatedResponse<T> {
  total: number;
  page: number;
  size: number;
  items: T[];
}

export interface TransactionsResponse {
  transactions: Transaction[];
  total: number;
  total_consumption: number;
  total_recharge: number;
  total_subscription_usage: number;
  total_sub_purchase: number;
  page: number;
  size: number;
}

export interface OrdersResponse {
  orders: Order[];
  total: number;
  status_counts: { paid: number; pending: number; expired: number };
  page: number;
  size: number;
}

export interface CreateKeyRequest {
  name: string;
}

export interface CreateKeyResponse {
  key: string;
  api_key: APIKey;
}

export interface CreatePaymentRequest {
  amount: number;
}

export interface CreatePaymentResponse {
  order_no: string;
  pay_url: string;
  expired_at: string;
}

// --- Invoice ---

export interface InvoiceTitle {
  id: number;
  user_id: string;
  type: string;
  title_name: string;
  tax_number: string;
  bank_name: string;
  bank_account: string;
  address: string;
  phone: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface InvoiceRequest {
  id: number;
  user_id: string;
  title_id: number;
  invoice_type: string;
  total_amount: number;
  status: string;
  remark: string;
  reject_reason: string;
  invoice_file_path: string;
  invoice_number: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface InvoiceRequestOrder {
  id: number;
  request_id: number;
  order_id: string;
  amount: number;
}

export interface InvoiceRequestsResponse {
  requests: InvoiceRequest[];
  total: number;
  page: number;
  size: number;
}

export interface CompanyInfo {
  name: string;
  tax_number: string;
}

export interface CreateInvoiceTitleRequest {
  type: string;
  title_name: string;
  tax_number?: string;
  bank_name?: string;
  bank_account?: string;
  address?: string;
  phone?: string;
}

export interface CreateInvoiceRequestBody {
  title_id: number;
  invoice_type: string;
  order_ids: string[];
  remark?: string;
}

// --- Announcements ---

export interface Announcement {
  id: number;
  title: string;
  content: string;
  status: string;
  priority: string;
  display_mode: string;
  published_at?: string;
  created_at: string;
  updated_at: string;
}

export interface PublishedAnnouncementsResponse {
  announcements: Announcement[];
}

// --- Subscription ---

export interface SubscriptionPlan {
  id: number;
  name: string;
  display_name: string;
  description: string;
  monthly_price_cny: number;
  quota_amount_cny: number;
  duration_days: number;
  sort_order: number;
  status: string;
  models: string[];
}

export interface UserSubscription {
  id: string;
  user_id: string;
  plan_id: number;
  plan?: SubscriptionPlan;
  status: string;
  started_at: string;
  expires_at: string;
  auto_renew?: boolean;
  created_at: string;
  updated_at: string;
}

export interface SubscriptionModelUsage {
  model_name: string;
  input_tokens_used: number;
  output_tokens_used: number;
  cache_read_tokens_used: number;
  cache_creation_tokens_used: number;
  amount_used: number;
  request_count: number;
}

export interface SubscriptionUsageSummary {
  total_amount_used: number;
  quota_amount_cny: number;
  usage_percent: number;
  model_details: SubscriptionModelUsage[];
}

export interface SubscriptionWithUsage {
  subscription: UserSubscription;
  usage: SubscriptionUsageSummary | null;
}

export interface SubscriptionCurrentResponse {
  subscriptions: SubscriptionWithUsage[];
}

export interface SubscriptionPlansResponse {
  plans: SubscriptionPlan[];
  purchase_disabled?: boolean;
  disabled_reason?: string;
}

export interface SubscriptionHistoryResponse {
  items: UserSubscription[];
}

export interface SubscribeResponse {
  subscription?: UserSubscription;
  need_payment?: boolean;
  shortfall?: number;
  balance?: number;
  plan_price?: number;
  order_no?: string;
  pay_url?: string;
  expired_at?: string;
}

// --- Notifications ---

export interface AppNotification {
  id: number;
  user_id: string;
  type: string;
  title: string;
  content: string;
  is_read: boolean;
  ref_type?: string;
  ref_id?: string;
  created_at: string;
}

export interface NotificationsResponse {
  notifications: AppNotification[];
  total: number;
  page: number;
  size: number;
}

export interface UnreadCountResponse {
  count: number;
}

// --- Promotion ---

export interface PromotionRule {
  type: string;
  threshold?: number;
  bonus_amount?: number;
  bonus_percent?: number;
  description: string;
}

export interface PromotionRulesResponse {
  rules: PromotionRule[];
  first_recharge_bonus_cny: number;
}

// --- Tenant ---

export interface Tenant {
  id: string;
  name: string;
  owner_id: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface TenantMember {
  id: string;
  tenant_id: string;
  user_id: string;
  role: string;
  phone?: string;
  nickname?: string;
  joined_at: string;
}

export interface TenantWithRole extends Tenant {
  role: string;
  member_count?: number;
}

export interface TenantInvitation {
  id: string;
  tenant_id: string;
  tenant_name: string;
  inviter_phone: string;
  role: string;
  status: string;
  created_at: string;
}

// --- Token stats ---

export interface UserTokenStats {
  prompt: number;
  completion: number;
  cache_read: number;
  cache_creation: number;
}

export interface UserTokenStatsResponse {
  all_time: UserTokenStats;
  period: UserTokenStats & { days: number };
}

// --- Chat / Playground ---

export interface ChatSession {
  id: string;
  user_id: string;
  title?: string;
  model: string;
  created_at: string;
  updated_at: string;
}

export interface ChatMessage {
  id: string;
  session_id: string;
  role: string;
  content: string;
  tokens_used?: number;
  cost?: number;
  created_at: string;
}

export interface ChatCompletionMessage {
  role: "system" | "user" | "assistant";
  content: string;
}

export interface SSEUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

// --- Image generation ---

export interface ImageTask {
  id: number;
  user_id: string;
  type: "generate" | "edit";
  status: "pending" | "processing" | "completed" | "failed";
  model: string;
  prompt: string;
  size: string;
  image_count: number;
  result_urls: string[] | null;
  cost: number;
  error_message: string;
  created_at: string;
  started_at: string | null;
  completed_at: string | null;
}

export type ImageTaskParams = Record<string, string | number>;

export interface SubmitTaskRequest {
  model: string;
  prompt: string;
  size: string;
  n: number;
  params?: ImageTaskParams;
}

export interface SubmitTaskResponse {
  id: number;
  status: string;
}

export interface EditTaskRequest {
  model: string;
  prompt: string;
  size: string;
  n: number;
  images: File[];
  mask?: File;
  params?: ImageTaskParams;
}

// --- Check-in ---

export interface CheckinStatus {
  checked_in_today: boolean;
  current_streak: number;
  next_reward_cny: number;
}

export interface CheckinResult {
  checkin_date: string;
  streak: number;
  reward_cny: number;
  balance_after: number;
}

// --- Growth Tasks ---

export interface TaskDefinition {
  code: string;
  title: string;
  description: string;
  reward_cny: number;
  reward_lottery_draws: number;
  status: "pending" | "completed" | "claimed";
  completed_at?: string;
  claimed_at?: string;
}

// --- Referral ---

export interface ReferralInfo {
  referral_code: string;
  invited_count: number;
  rewarded_count: number;
  total_reward: number;
}

// --- Support Tickets ---

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
