// TypeScript types matching backend Go structs.
// All JSON field names use snake_case to match the Go `json:` tags.

export interface User {
  id: string;
  phone?: string;
  email?: string;
  nickname?: string;
  role: string; // "admin" | "user"
  status: string; // "active" | "disabled"
  email_verified?: boolean;
  created_at: string;
  updated_at: string;
}

/** Subset returned by GET /api/me */
export interface MeResponse {
  user_id: string;
  phone?: string;
  email?: string;
  role: string;
  email_verified?: boolean;
  first_recharge_bonus_cny?: number;
  first_recharge_bonus_granted?: boolean;
  image_share_enabled?: boolean;
}

/** Returned by GET /api/image-share/me when the session is an image-share key holder. */
export interface ImageShareMe {
  role: "image_share";
  key_id: string;
  name: string;
  key_prefix: string;
  quota_total: number;
  quota_used: number;
  quota_remaining: number;
  owner_user_id: string;
  status: string;
}

export interface ImageShareKey {
  id: string;
  owner_user_id: string;
  key_prefix: string;
  name: string;
  quota_total: number;
  quota_used: number;
  status: string;
  last_used_at?: string | null;
  created_at: string;
  updated_at: string;
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
  type: string; // "consumption" | "recharge" | "subscription_usage"
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
  daily_trend: DailyCost[];
  model_breakdown: ModelCost[];
}

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
  status: string; // "pending" | "paid" | "expired"
  pay_time?: string;
  created_at: string;
  expired_at: string;
}

export interface AdminOrder extends Order {
  user_identifier: string;
}

export interface AdminOrdersResponse {
  orders: AdminOrder[];
  total: number;
  page: number;
  size: number;
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
  protocols: string[];
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

export interface TimeBasedPricingRule {
  name: string;
  days: number[];        // 0=Sunday, 1=Monday, ..., 6=Saturday
  start_time: string;    // "HH:MM"
  end_time: string;      // "HH:MM"
  multiplier: number;    // e.g. 1.5 or 0.8
}

export interface ModelPricing {
  id: number;
  model_name: string;
  /** CNY per 1M tokens */
  input_price: number;
  output_price: number;
  cached_input_price: number;
  cache_creation_price: number;
  cache_creation_1h_price: number;
  /** "token" (default) or "image" */
  billing_type: string;
  is_active: boolean;
  updated_at: string;
  pricing_tiers?: PricingTier[];
  time_based_rules?: TimeBasedPricingRule[];
  /** Tenant discount rate (0,1]; present only when caller's tenant has a discount. */
  discount_rate?: number;
  /** Original global pricing for struck-through comparison; present only under discount. */
  original_pricing?: ModelPricing;
}

export interface PricingListResponse {
  pricing: ModelPricing[];
}

export interface PricingChangeLog {
  id: number;
  model_name: string;
  change_type: string;
  admin_user_id: string;
  old_values: Record<string, unknown>;
  new_values: Record<string, unknown>;
  created_at: string;
}

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

// --- Paginated response wrappers ---

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

export interface UserWithBalance extends User {
  balance: number;
  image_share_enabled?: boolean;
}

export interface AdminUsersResponse {
  users: UserWithBalance[];
  total: number;
  global_total: number;
  active_count: number;
  total_balance: number;
  page: number;
  size: number;
}

// --- Request / response helpers ---

export interface LoginRequest {
  phone: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  phone: string;
}

export interface RegisterRequest {
  phone: string;
  code: string;
  password: string;
}

export interface RegisterResponse {
  token: string;
  user: { id: string; phone: string; role: string };
}

export interface SendSmsRequest {
  phone: string;
  purpose: string;
}

export interface CreateKeyRequest {
  name: string;
}

export interface CreateKeyResponse {
  key: string; // full key "sk-..."
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

export interface RechargeRequest {
  amount: number;
  description?: string;
}

export interface RechargeResponse {
  status: string;
  balance: number;
}

export interface UpdatePricingRequest {
  input_price?: number;
  output_price?: number;
  cached_input_price?: number;
  cache_creation_price?: number;
  cache_creation_1h_price?: number;
  billing_type?: string;
  is_active?: boolean;
  pricing_tiers?: PricingTier[];
  time_based_rules?: TimeBasedPricingRule[];
  discount_rate?: number;
}

export interface TenantPricing {
  id: number;
  tenant_id: string;
  model_name: string;
  input_price: number;
  output_price: number;
  cached_input_price: number;
  cache_creation_price: number;
  cache_creation_1h_price: number;
  billing_type: string;
  is_active: boolean;
  pricing_tiers?: PricingTier[];
  discount_rate?: number;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface TenantPricingListResponse {
  pricing: TenantPricing[];
}

export interface UserPricing {
  id: number;
  user_id: string;
  model_name: string;
  input_price: number;
  output_price: number;
  cached_input_price: number;
  cache_creation_price: number;
  cache_creation_1h_price: number;
  billing_type: string;
  is_active: boolean;
  pricing_tiers?: PricingTier[];
  discount_rate?: number;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface UserPricingListResponse {
  pricing: UserPricing[];
}

export interface TenantModelUpstream {
  id: number;
  tenant_id: string;
  model_name: string;
  provider: string;
  protocol: string;
  protocols: string[];
  upstream_provider: string;
  upstream_name: string;
  base_url: string;
  api_key: string;
  model_override: string;
  weight: number;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface TenantModelUpstreamListResponse {
  upstreams: TenantModelUpstream[];
}

export interface TenantUpstreamInput {
  provider: string;
  protocol?: string;
  protocols?: string[];
  upstream_provider?: string;
  upstream_name?: string;
  base_url: string;
  api_key: string;
  model_override?: string;
  weight?: number;
}

export interface UpstreamState {
  provider: string;
  base_url: string;
  state: string;
  failure_count: number;
}

export interface GatewayStatus {
  models: Record<string, { upstreams: UpstreamState[] }>;
}

// --- Playground / Streaming ---

export interface ChatCompletionMessage {
  role: "system" | "user" | "assistant";
  content: string;
}

export interface ChatCompletionRequest {
  model: string;
  messages: ChatCompletionMessage[];
  stream: boolean;
  temperature?: number;
  max_tokens?: number;
  top_p?: number;
}

export interface SSEDelta {
  content?: string;
}

export interface SSEChoice {
  index: number;
  delta: SSEDelta;
  finish_reason: string | null;
}

export interface SSEUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

export interface SSEChunk {
  id: string;
  object: string;
  model: string;
  choices: SSEChoice[];
  usage?: SSEUsage;
}

// --- Invoice ---

export interface InvoiceTitle {
  id: number;
  user_id: string;
  type: string; // "personal" | "enterprise"
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
  invoice_type: string; // "normal" | "special"
  total_amount: number;
  status: string; // "pending" | "processing" | "completed" | "rejected" | "cancelled"
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

export interface AdminInvoiceRequest extends InvoiceRequest {
  title: InvoiceTitle;
  user_identifier: string;
}

export interface AdminInvoiceRequestsResponse {
  requests: AdminInvoiceRequest[];
  total: number;
  page: number;
  size: number;
}

export interface CompanyInfo {
  name: string;
  tax_number: string;
}

export interface ModelTokenStats {
  model: string;
  prompt_tokens: number;
  completion_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  total_cost: number;
  prompt_cost: number;
  completion_cost: number;
  cache_read_cost: number;
  cache_creation_cost: number;
  request_count: number;
  breakdown_estimated: boolean;
  failure_count?: number;
  success_rate?: number;
}

export interface AdminConsumptionStats {
  total_cost: number;
  total_requests: number;
  models: ModelTokenStats[];
  daily_trend: DailyCost[];
}

export interface FunnelStage {
  key: string;
  label: string;
  count: number;
}

export interface AdminFunnelStats {
  days: number;
  stages: FunnelStage[];
  first_recharge_rate: number;
  repeat_recharge_rate: number;
  post_recharge_use_rate: number;
}

export interface ImageDurationStats {
  model: string;
  request_count: number;
  min_seconds: number;
  avg_seconds: number;
  max_seconds: number;
}

export interface ImageDurationStatsResponse {
  models: ImageDurationStats[];
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
  status: string; // "draft" | "published" | "archived"
  priority: string; // "normal" | "important" | "urgent"
  display_mode: string; // "banner" | "dialog"
  created_by?: string;
  published_at?: string;
  created_at: string;
  updated_at: string;
}

export interface AnnouncementsResponse {
  announcements: Announcement[];
  total: number;
  page: number;
  size: number;
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
  auto_renew: boolean;
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

export interface AdminSubscriptionPlansResponse {
  plans: SubscriptionPlan[];
}

export interface SubscriptionHistoryResponse {
  items: UserSubscription[];
}

export interface SubscriptionsResponse {
  subscriptions: UserSubscription[];
  total: number;
  page: number;
  size: number;
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

export interface SubscriptionOrderDaily {
  date: string;
  count: number;
  amount: number;
}

export interface SubscriptionOrderByPlan {
  plan_name: string;
  count: number;
  amount: number;
}

export interface SubscriptionOrderByType {
  type: string;
  count: number;
  amount: number;
}

export interface SubscriptionOrderStats {
  total_revenue: number;
  total_orders: number;
  paid_orders: number;
  pending_orders: number;
  avg_order_value: number;
  daily_trend: SubscriptionOrderDaily[];
  plan_breakdown: SubscriptionOrderByPlan[];
  type_breakdown: SubscriptionOrderByType[];
}

export interface AdminSubscriptionOrder {
  id: string;
  user_id: string;
  user_identifier: string;
  plan_id: number;
  plan_name: string;
  amount_cny: number;
  type: string;
  status: string;
  payment_method?: string;
  created_at: string;
  paid_at?: string;
}

export interface AdminSubscriptionOrdersResponse {
  orders: AdminSubscriptionOrder[];
  total: number;
  page: number;
  size: number;
}

export interface AdminSubscriptionUserUsage {
  subscription_id: string;
  user_id: string;
  user_identifier: string;
  user_nickname: string;
  plan_id: number;
  plan_name: string;
  plan_category: string; // "image" | "openai" | "claude"
  plan_price_cny: number;
  quota_amount_cny: number;
  amount_used: number;
  amount_remaining: number;
  usage_percent: number;
  request_count: number;
  status: string;
  started_at: string;
  expires_at: string;
  auto_renew: boolean;
}

export interface AdminSubscriptionUsersUsageResponse {
  users: AdminSubscriptionUserUsage[];
  total: number;
  active_count: number;
  total_usage: number;
  page: number;
  size: number;
}

// --- Recharge Promotions ---

export interface RechargePromotion {
  id: number;
  name: string;
  starts_at: string;
  ends_at: string;
  bonus_ratio: number;
  min_recharge_amount: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface RechargePromotionInput {
  name: string;
  starts_at: string;
  ends_at: string;
  bonus_ratio: number;
  min_recharge_amount: number;
  is_active: boolean;
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

// --- Recharge Lottery ---
export interface RechargeLottery {
  id: number;
  name: string;
  status: "active" | "paused";
  trigger_every: number;
  total_rounds: number;
  created_at: string;
  updated_at: string;
}

export interface RechargeLotteryRound {
  id: number;
  lottery_id: number;
  round_no: number;
  winner_user_id: string;
  winner_nickname?: string;
  winner_phone?: string;
  winner_amount: number;
  winner_order_no: string;
  participant_count: number;
  created_at: string;
}

// --- Referral ---
export interface ReferralInfo {
  referral_code: string;
  invited_count: number;
  rewarded_count: number;
  total_reward: number;
}

// --- Lottery ---
export interface LotteryEvent {
  id: number;
  name: string;
  description: string;
  status: "active" | "paused" | "ended";
  min_recharge_cny: number;
  min_order_count_to_draw: number;
  start_time: string | null;
  end_time: string | null;
  participant_count: number;
  drawn_at: string | null;
  drawn_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface LotteryPrize {
  id: number;
  event_id: number;
  name: string;
  description: string;
  weight: number;
  total_stock: number;
  remaining_stock: number;
  prize_type: "none" | "balance" | "match_recharge" | "physical";
  prize_value: number;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface LotteryRecord {
  id: number;
  event_id: number;
  user_id: string;
  phone: string;
  nickname: string;
  prize_id: number | null;
  prize_name: string;
  prize_type: string;
  prize_value: number;
  order_no: string;
  recharge_amount: number;
  created_at: string;
}

export interface PublicLotteryRecord {
  id: number;
  masked_phone: string;
  prize_name: string;
  prize_type: string;
  prize_value: number;
  created_at: string;
}
