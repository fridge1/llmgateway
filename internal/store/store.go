package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"time"
)

// ErrInvalidCredentials is returned when authentication fails (wrong phone or password).
// A single sentinel avoids leaking whether the phone exists.
var ErrInvalidCredentials = errors.New("invalid credentials")

// Store defines all data-access methods for the gateway.
// Implementations must be safe for concurrent use.
type Store interface {
	io.Closer
	DB() *sql.DB

	// User methods
	CreateUser(phone, password, role string) (*User, error)
	CreateUserWithEmail(email, password, role string) (*User, error)
	Authenticate(phone, password string) (*User, error)
	AuthenticateByEmail(email, password string) (*User, error)
	AuthenticateByIdentifier(identifier, password string) (*User, error)
	UserCount() (int64, error)
	GetUserByID(id string) (*User, error)
	GetUserByPhone(phone string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByIdentifier(identifier string) (*User, error)
	MarkFirstRechargeGranted(userID string) (bool, error)
	MarkEmailVerified(userID string) error
	ListUsers(limit, offset int) ([]User, int, error)
	ListUsersWithBalance(limit, offset int, search string) (users []UserWithBalance, filteredTotal, globalTotal, globalActiveCount int, globalTotalBalance float64, err error)
	UpdateUserStatus(id, status string) error
	UpdateUserRole(id, role string) error
	DeleteUser(id string) error
	UpdatePassword(phone, newPassword string) error
	UpdatePasswordByEmail(email, newPassword string) error
	GetAdminDashboardStats() (*AdminDashboardStats, error)
	GetAdminConsumptionStats(days int) (*AdminConsumptionStats, error)
	GetAdminFunnelStats(days int) (*AdminFunnelStats, error)
	GetImageDurationStats(days int) ([]ImageDurationStats, error)

	// Public landing-page aggregates
	CountActiveUsers() (int64, error)
	CountEnterpriseTenants() (int64, error)
	GetPublicUsageTotals() (*PublicUsageTotals, error)

	// Model / upstream methods
	ListModels() ([]Model, error)
	GetModel(id int64) (*Model, error)
	GetModelByName(name string) (*Model, error)
	CreateModel(name, displayName, category string, upstreams []Upstream) (*Model, error)
	UpdateModel(id int64, name, displayName, category string, upstreams []Upstream) (*Model, error)
	DeleteModel(id int64) error

	// API key methods
	CreateAPIKey(userID, keyHash, keyPrefix, name string, planID *int) (*APIKey, error)
	ListAPIKeysByUser(userID string) ([]APIKey, error)
	GetAPIKeyByHash(keyHash string) (*APIKey, error)
	GetActiveAPIKeyByIDAndUser(keyID, userID string) (*APIKey, error)
	DeleteAPIKey(id, userID string) error
	RevokeAllAPIKeys(userID string) (int, error)
	TouchAPIKeyLastUsed(id string) error
	BatchTouchAPIKeysLastUsed(ids []string) error
	BatchTouchUsersLastActive(ids []string) error


	// Balance methods
	GetBalance(userID string) (*Balance, error)
	FreezeBalance(userID string, amount float64) error
	SettleBilling(userID string, frozenAmount, actualCost float64, model, requestID string, tokens TokenUsage, apiKeyID string) error
	DirectCharge(userID string, amount float64, model, requestID string, tokens TokenUsage, apiKeyID string) error
	DeductForSubscription(userID string, amount float64, description string) error
	UnfreezeBalance(userID string, amount float64) error
	Recharge(userID string, amount float64, description string) error
	// Transaction methods
	ListTransactions(userID string, limit, offset int, typeFilter string, startDate, endDate *time.Time) ([]Transaction, int, *TransactionSums, error)
	RecordSubscriptionTransaction(userID, subscriptionID, model, requestID string, amount float64, tokens TokenUsage) error
	GetBillingStats(userID string, days int) (*BillingStats, error)
	GetUserTokenStats(userID string, days int) (*UserTokenStatsResponse, error)
	GetUserConsumptionStats(userID string, days int) (*UserConsumptionStats, error)
	ListUserTransactionsForExport(userID string, startDate, endDate *time.Time) ([]Transaction, error)
	GetSubUserBillingStats(subUserID string, days int) (*BillingStats, error)
	GetAPIKeyUsageSummary(userID string, days int) ([]APIKeyUsageSummary, error)
	ListTransactionsByAPIKey(keyID string, limit, offset int) ([]Transaction, int, error)
	GetAPIKeyByID(id string) (*APIKey, error)
	RecordRequestFailure(userID, model string, httpStatus int) error
	GetModelSuccessStats(userID string, days int) ([]ModelSuccessStats, error)
	GetAdminModelSuccessStats(days int) ([]ModelSuccessStats, error)

	// Pricing methods
	ListPricing() ([]ModelPricing, error)
	ListActivePricing() ([]ModelPricing, error)
	UpsertPricing(modelName string, inputPrice, outputPrice, cachedInputPrice, cacheCreationPrice, cacheCreation1hPrice float64, billingType string, isActive bool, pricingTiers []PricingTier, timeBasedRules []TimeBasedPricingRule) error
	GetPricing(modelName string) (*ModelPricing, error)
	InsertPricingChangeLog(modelName, changeType, adminUserID string, oldValues, newValues map[string]any) error
	ListPricingChangeLogs(limit, offset int) ([]PricingChangeLog, int, error)

	// Order methods
	CreateOrder(userID string, amount float64, tenantID *string) (*Order, error)
	CreateOrderWithPlan(userID string, amount float64, tenantID *string, planID *int) (*Order, error)
	GetOrderByNo(orderNo string) (*Order, error)
	MarkOrderPaid(orderNo string, callbackData []byte) error
	// FulfillAlipayPaidOrder marks the order paid and credits balance in one transaction.
	// If the order is already paid, it ensures a matching recharge transaction exists (idempotent / recovery).
	FulfillAlipayPaidOrder(orderNo string, callbackData []byte) error
	ListOrders(userID string, limit, offset int) ([]Order, int, *OrderStatusCounts, error)
	ListAllOrders(limit, offset int, status string) ([]AdminOrder, int, error)
	ExpireOrders() (int, error)

	// Chat session methods
	ListSessions(userID string) ([]ChatSession, error)
	CreateSession(userID, model, title string) (*ChatSession, error)
	GetSession(userID, sessionID string) (*ChatSession, error)
	UpdateSessionTitle(userID, sessionID, title string) error
	DeleteSession(userID, sessionID string) error
	ListMessages(sessionID string) ([]ChatMessage, error)
	AddMessage(sessionID, role, content string, tokensUsed int, cost float64) (*ChatMessage, error)

	// Invoice title methods
	CreateInvoiceTitle(userID, titleType, titleName, taxNumber, bankName, bankAccount, address, phone string) (*InvoiceTitle, error)
	UpdateInvoiceTitle(id int64, userID, titleType, titleName, taxNumber, bankName, bankAccount, address, phone string) (*InvoiceTitle, error)
	DeleteInvoiceTitle(id int64, userID string) error
	ListInvoiceTitlesByUser(userID string) ([]InvoiceTitle, error)
	GetInvoiceTitle(id int64, userID string) (*InvoiceTitle, error)
	GetInvoiceTitleByID(id int64) (*InvoiceTitle, error)
	SetDefaultInvoiceTitle(id int64, userID string) error

	// Invoice request methods
	ListAvailableOrders(userID string) ([]Order, error)
	CreateInvoiceRequest(userID string, titleID int64, invoiceType, remark string, orderIDs []string) (*InvoiceRequest, error)
	ListInvoiceRequests(userID string, limit, offset int) ([]InvoiceRequest, int, error)
	GetInvoiceRequest(id int64, userID string) (*InvoiceRequest, error)
	GetInvoiceRequestOrders(requestID int64) ([]InvoiceRequestOrder, error)
	CancelInvoiceRequest(id int64, userID string) error
	UpdateInvoiceRequestStatus(id int64, status string) error
	CompleteInvoiceRequest(id int64, filePath, invoiceNumber string) error
	RejectInvoiceRequest(id int64, reason string) error
	AdminListInvoiceRequests(status string, limit, offset int) ([]InvoiceRequestDetail, int, error)
	SetInvoiceRequestRisk(id int64, riskLevel, riskReasons string) error
	CountRecentInvoiceRequestsByTitle(titleID int64, excludeRequestID int64, days int) (int, error)
	BatchUpdateInvoiceRequestStatus(ids []int64, fromStatus, toStatus string) ([]int64, error)

	// Announcement methods
	ListAnnouncements(limit, offset int) ([]Announcement, int, error)
	CreateAnnouncement(title, content, status, priority, displayMode, createdBy string) (*Announcement, error)
	GetAnnouncementByID(id int64) (*Announcement, error)
	UpdateAnnouncement(id int64, title, content, status, priority, displayMode string) (*Announcement, error)
	DeleteAnnouncement(id int64) error
	ListPublishedAnnouncements() ([]Announcement, error)

	// AUP methods
	MarkAUPAccepted(userID string) error

	// Subscription methods
	ListSubscriptionPlans() ([]SubscriptionPlan, error)
	GetSubscriptionPlan(id int) (*SubscriptionPlan, error)
	GetSubscriptionPlanModels(planID int) ([]string, error)
	ListAllSubscriptionPlans() ([]SubscriptionPlan, error)
	CreateSubscriptionPlan(p SubscriptionPlan) (*SubscriptionPlan, error)
	UpdateSubscriptionPlan(p SubscriptionPlan) error
	DeleteSubscriptionPlan(id int) error
	SetSubscriptionPlanModels(planID int, models []string) error
	GetActiveSubscription(userID string) (*UserSubscription, error)
	GetActiveSubscriptions(userID string) ([]UserSubscription, error)
	GetActiveSubscriptionByBrand(userID, brand string) (*UserSubscription, error)
	CreateSubscription(userID string, planID int, expiresAt time.Time, brand string) (*UserSubscription, error)
	UpgradeSubscriptionTx(p UpgradeSubscriptionParams) (*UserSubscription, error)
	AddSubscriptionQuota(subscriptionID string, additionalQuota float64) error
	CancelSubscription(subscriptionID string) error
	ResumeSubscription(subscriptionID string) error
	ExpireSubscription(subscriptionID string) error
	ExpireExpiredSubscriptions() (int, error)
	ExpireUserSubscriptionsByBrand(userID, brand string) error
	GetSubscriptionTotalUsage(subscriptionID string, period time.Time) (float64, error)
	IncrementSubscriptionUsage(subscriptionID, userID, model string, period time.Time, tokens TokenUsage, amount float64) error
	RecordSubscriptionUsageAndTransaction(subscriptionID, userID, model, requestID string, period time.Time, tokens TokenUsage, amount float64, quotaConsumed float64, apiKeyID string) error
	GetSubscriptionUsageSummary(subscriptionID string, period time.Time, quotaAmount float64) (*SubscriptionUsageSummary, error)
	CreateSubscriptionOrder(userID string, planID int, amountCNY float64, orderType string) (*SubscriptionOrder, error)
	CompleteSubscriptionOrder(orderID, paymentMethod, paymentID string) error
	ListUserSubscriptions(limit, offset int) ([]UserSubscription, int, error)
	ListUserSubscriptionHistory(userID string) ([]UserSubscription, error)
	GetSubscriptionOrderStats(days int) (*SubscriptionOrderStats, error)
	ListAllSubscriptionOrders(limit, offset int, status, orderType string) ([]AdminSubscriptionOrder, int, error)
	ListSubscriptionUsersUsage(limit, offset int, search, status, planFilter string) ([]AdminSubscriptionUserUsage, int, int, float64, error)

	// Recharge promotion methods
	CreateRechargePromotion(p *RechargePromotion) (*RechargePromotion, error)
	UpdateRechargePromotion(id int, p *RechargePromotion) (*RechargePromotion, error)
	DeleteRechargePromotion(id int) error
	ListRechargePromotions() ([]RechargePromotion, error)
	GetBestActiveRechargePromotion(now time.Time, amount float64) (*RechargePromotion, error)
	GetCurrentActiveRechargePromotion(now time.Time) (*RechargePromotion, error)

	// Notification methods
	CreateNotification(userID, nType, title, content string, refType, refID *string) (*Notification, error)
	BatchCreateNotifications(notifications []Notification) error
	TryClaimNotificationDedup(userID, kind string) (bool, error)
	GetWeeklyUsageSummaries() ([]WeeklyUsageSummary, error)
	GetSilentUsers(minDaysAgo, maxDaysAgo int) ([]string, error)
	DoCheckin(userID string, base float64) (*CheckinResult, error)
	GetCheckinStatus(userID string, base float64) (*CheckinStatus, error)
	ListUserTasks(userID string) ([]TaskDefinition, error)
	MarkTaskCompleted(userID, code string) error
	ClaimTaskReward(userID, code string) (rewardCNY float64, lotteryDraws int, err error)
	GetReferralCode(userID string) (string, error)
	GetUserIDByReferralCode(code string) (string, error)
	SetReferredBy(userID, referrerID string) error
	GrantReferralReward(invitedUserID string, inviterBonus, inviteeBonus float64) (bool, error)
	GetReferralInfo(userID string) (*ReferralInfo, error)
	GetActiveReferralRule() (*ReferralRule, error)
	ListReferralRules(limit int) ([]ReferralRule, error)
	CreateReferralRule(inviterBonus, inviteeBonus, minFirstRecharge float64, enabled bool, effectiveFrom time.Time, createdBy string) (*ReferralRule, error)
	ListNotifications(userID string, limit, offset int) ([]Notification, int, error)
	CountUnreadNotifications(userID string) (int, error)
	MarkNotificationRead(userID string, id int64) error
	MarkAllNotificationsRead(userID string) error

	// Tenant methods
	CreateTenant(name, ownerID string) (*Tenant, error)
	GetTenantByID(id string) (*Tenant, error)
	UpdateTenant(id, name string) error
	ListTenantsByUser(userID string) ([]TenantWithRole, error)
	ListAllTenants(limit, offset int) ([]Tenant, int, error)
	DeleteTenant(id string) error

	// Tenant member methods
	AddTenantMember(tenantID, userID, role string) error
	RemoveTenantMember(tenantID, userID string) error
	UpdateTenantMemberRole(tenantID, userID, role string) error
	ListTenantMembers(tenantID string) ([]TenantMember, error)
	GetTenantMember(tenantID, userID string) (*TenantMember, error)
	TransferTenantOwnership(tenantID, newOwnerID string) error

	// Tenant balance methods
	GetTenantBalance(tenantID string) (*TenantBalance, error)
	RechargeTenant(tenantID string, amount float64, operatorID, description string) error
	FreezeTenantBalance(tenantID string, amount float64) error
	SettleTenantBilling(tenantID string, frozenAmount, actualCost float64, model, requestID string, tokens TokenUsage, apiKeyID string) error
	DirectChargeTenant(tenantID string, amount float64, model, requestID string, tokens TokenUsage, apiKeyID string) error
	UnfreezeTenantBalance(tenantID string, amount float64) error
	ListTenantTransactions(tenantID string, limit, offset int, typeFilter string, startDate, endDate *time.Time) ([]Transaction, int, error)

	// Tenant pricing methods
	ListTenantPricing(tenantID string) ([]TenantPricing, error)
	GetTenantPricing(tenantID, modelName string) (*TenantPricing, error)
	UpsertTenantPricing(tenantID, modelName string, inputPrice, outputPrice, cachedInputPrice, cacheCreationPrice, cacheCreation1hPrice float64, billingType string, isActive bool, pricingTiers []PricingTier, discountRate *float64, createdBy string) error
	DeleteTenantPricing(tenantID, modelName string) error
	HasTenantCustomPricing(tenantID string) (bool, error)
	GetUserPrimaryPricingTenant(userID string) (string, error)

	// User pricing methods
	ListUserPricing(userID string) ([]UserPricing, error)
	GetUserPricing(userID, modelName string) (*UserPricing, error)
	UpsertUserPricing(userID, modelName string, inputPrice, outputPrice, cachedInputPrice, cacheCreationPrice, cacheCreation1hPrice float64, billingType string, isActive bool, pricingTiers []PricingTier, discountRate *float64, createdBy string) error
	DeleteUserPricing(userID, modelName string) error
	HasUserCustomPricing(userID string) (bool, error)

	// Tenant model upstream override methods
	ListTenantModelUpstreams(tenantID string) ([]TenantModelUpstream, error)
	ListAllTenantModelUpstreams() ([]TenantModelUpstream, error)
	ReplaceTenantModelUpstreams(tenantID, modelName string, upstreams []TenantModelUpstream) error
	DeleteTenantModelUpstreams(tenantID, modelName string) error

	// Tenant API key methods
	CreateTenantAPIKey(tenantID, keyHash, keyPrefix, name, createdBy string) (*TenantAPIKey, error)
	ListTenantAPIKeys(tenantID string) ([]TenantAPIKey, error)
	GetTenantAPIKeyByHash(keyHash string) (*TenantAPIKey, error)
	DeleteTenantAPIKey(id, tenantID string) error
	TouchTenantAPIKeyLastUsed(id string) error

	// Tenant invitation methods
	CreateTenantInvitation(tenantID, phone, role, invitedBy string) (*TenantInvitation, error)
	ListTenantInvitations(tenantID string) ([]TenantInvitation, error)
	AcceptTenantInvitation(invitationID, userID string) error
	DeleteTenantInvitation(id string) error
	GetPendingInvitationsByPhone(phone string) ([]TenantInvitation, error)

	// Enterprise tenant methods (admin operations)
	CreateEnterpriseTenant(name, ownerID, adminID, contactPhone, contactEmail string) (*Tenant, error)
	UpdateTenantEnterpriseInfo(tenantID, contactPhone, contactEmail string) error

	// Tenant sub-user methods
	CreateTenantSubUser(tenantID, username, password, nickname string, quotaLimit *float64, createdBy string) (*TenantSubUser, error)
	GetTenantSubUser(subUserID string) (*TenantSubUser, error)
	GetTenantSubUserByUsername(tenantID, username string) (*TenantSubUser, error)
	AuthenticateSubUser(tenantID, username, password string) (*TenantSubUser, error)
	ListTenantSubUsers(tenantID string) ([]TenantSubUserWithQuota, error)
	UpdateTenantSubUserQuota(subUserID string, quotaLimit *float64) error
	UpdateTenantSubUserStatus(subUserID, status string) error
	DeleteTenantSubUser(subUserID string) error
	ResetTenantSubUserPassword(subUserID, newPassword string) error

	// Tenant sub-user quota methods
	IncrementSubUserQuotaUsed(subUserID string, amount float64) error
	CheckSubUserQuota(subUserID string, estimatedCost float64) error

	// Tenant sub-user API key methods
	CreateTenantSubUserKey(subUserID, tenantID, keyHash, keyPrefix, name string) (*TenantSubUserKey, error)
	ListTenantSubUserKeys(subUserID string) ([]TenantSubUserKey, error)
	GetTenantSubUserKeyByHash(keyHash string) (*TenantSubUserKey, error)
	DeleteTenantSubUserKey(keyID, tenantID string) error
	DeleteSubUserOwnKey(keyID, subUserID string) error
	TouchTenantSubUserKeyLastUsed(keyID string) error

	// Tenant sub-user transaction methods
	ListTenantSubUserTransactions(subUserID string, limit, offset int) ([]Transaction, int, error)
	ListTenantAllSubUserTransactions(tenantID string, limit, offset int, subUserID string) ([]Transaction, int, error)
	RecordSubUserTransaction(tenantID, subUserID string, amount float64, model, requestID string, tokens TokenUsage, apiKeyID string) error
	GetSubUserModelStats(subUserID string, startDate, endDate *time.Time) (*SubUserModelStats, error)

	// Tenant analytics
	GetTenantBillingStats(tenantID string, days int) (*TenantBillingStats, error)
	ListTenantTransactionsForExport(tenantID string, startDate, endDate *time.Time, subUserID string) ([]Transaction, error)

	// Alerting methods
	ListAlertRules() ([]AlertRule, error)
	UpdateAlertRule(id int, threshold int64, cooldownSeconds int, enabled bool) error
	TryClaimAlertEvent(metric, message string, value, threshold int64, cooldownSeconds int) (bool, error)
	ListAlertEvents(limit, offset int) ([]AlertEvent, int, error)
	ListAdminUsers() ([]User, error)

	// Moderation methods
	GetModerationSettings() (*ModerationSettings, error)
	UpdateModerationSettings(enabled, enforceAll bool) error
	ListModerationKeywords() ([]ModerationKeyword, error)
	CreateModerationKeyword(keyword, category string) (*ModerationKeyword, error)
	DeleteModerationKeyword(id int) error
	CreateModerationHit(userID, tenantID *string, model, matchedRule, snippet string) error
	ListModerationHits(userID string, from, to *time.Time, limit, offset int) ([]ModerationHit, int, error)
	ListModerationEnabledTargets() (models []string, tenants []string, err error)
	SetModelModeration(modelName string, enabled bool) error

	// Ticket methods
	CreateTicket(userID, category, subject, content string, relatedOrderNo *string, attachments json.RawMessage) (*Ticket, error)
	GetTicket(id, userID string) (*Ticket, error)
	AppendTicketMessage(ticketID, senderRole, senderID, content string, attachments json.RawMessage) (*TicketMessage, error)
	ListTicketMessages(ticketID string) ([]TicketMessage, error)
	ListUserTickets(userID string, limit, offset int) ([]Ticket, int, error)
	ListAdminTickets(status string, limit, offset int) ([]Ticket, int, error)
	UpdateTicketStatus(id, status string) error

	// Refund methods
	CreateRefund(orderNo, operatorID string, amount float64, reason string) (*Refund, error)
	CompleteRefund(refundID, alipayTradeNo string) error
	FailRefund(refundID, errorMessage string) error
	ListRefunds(limit, offset int) ([]Refund, int, error)

	// Notification preference methods
	ListNotificationPreferences(userID string) ([]NotificationPreference, error)
	UpsertNotificationPreference(userID, eventType, channel string, enabled bool) error
	SMSNotificationEnabled(userID, eventType string) (bool, error)
	ListExpiringSubscriptions(withinDays int) ([]UserSubscription, error)

	// Recharge lottery methods
	GetActiveLottery() (*RechargeLottery, error)
	GetCurrentRoundEntryCount(lotteryID int) (int, error)
	RecordEntryAndMaybeDraw(lotteryID int, userID, orderNo string, amount float64) (*RechargeLotteryWin, error)
	CreateRechargeLottery(name string, triggerEvery int) (*RechargeLottery, error)
	UpdateRechargeLottery(id int, name, status string, triggerEvery int) (*RechargeLottery, error)
	ListRechargeLotteryRounds(lotteryID, limit, offset int) ([]RechargeLotteryRound, int, error)
	ListRechargeLotteryRoundsAdmin(lotteryID, limit, offset int) ([]RechargeLotteryRound, int, error)

	// Lottery methods (automatic draw on recharge)
	CreateLotteryEvent(name, description string, status string, minRechargeCNY float64, minOrderCountToDraw int, startTime, endTime *time.Time) (*LotteryEvent, error)
	UpdateLotteryEvent(id int, name, description, status string, minRechargeCNY float64, minOrderCountToDraw int, startTime, endTime *time.Time) (*LotteryEvent, error)
	ListLotteryEvents(limit, offset int) ([]LotteryEvent, int, error)
	GetLotteryEvent(id int) (*LotteryEvent, error)
	CreateLotteryPrize(eventID int, name, description string, weight, totalStock int, prizeType string, prizeValue float64, sortOrder int) (*LotteryPrize, error)
	UpdateLotteryPrize(id int, name, description string, weight, totalStock int, prizeType string, prizeValue float64, sortOrder int) (*LotteryPrize, error)
	DeleteLotteryPrize(id int) error
	ListLotteryPrizes(eventID int) ([]LotteryPrize, error)
	ListLotteryRecords(eventID, limit, offset int) ([]LotteryRecord, int, error)
	ListUserLotteryRecords(userID string, limit, offset int) ([]LotteryRecord, int, error)
	ListPublicLotteryWinners(eventID, limit, offset int) ([]PublicLotteryRecord, int, error)
	ListAllPublicLotteryWinners(limit, offset int) ([]PublicLotteryRecord, int, error)
	RecordLotteryParticipation(userID, orderNo string, rechargeAmount float64) error
	DrawEventLottery(eventID int, drawnBy string) ([]LotteryRecord, error)
	DrawLottery(userID, orderNo string, rechargeAmount float64) (*LotteryRecord, error) // Deprecated
	GetActiveLotteryInfo() (*LotteryEvent, []LotteryPrize, error)

	// IP blocking methods
	BlockIP(ipAddress, reason, blockedBy string, expiresAt *time.Time) error
	UnblockIP(ipAddress string) error
	IsIPBlocked(ipAddress string) (bool, error)
	GetBlockedIP(ipAddress string) (*BlockedIP, error)
	ListBlockedIPs(limit, offset int) ([]BlockedIP, error)
	CountBlockedIPs() (int, error)
	CleanupExpiredBlockedIPs() (int, error)

	// Codex service methods
	ListCodexProducts() ([]CodexProduct, error)
	GetCodexProductByID(id int) (*CodexProduct, error)
	ListAllCodexProducts() ([]CodexProduct, error)
	CreateCodexOrder(productID int, userID *string, guestContact json.RawMessage, serviceWechat string) (*CodexOrder, error)
	GetCodexOrderByNo(orderNo string) (*CodexOrder, error)
	MarkCodexOrderPaid(orderNo string, callbackData []byte) error
	ShipCodexOrder(orderNo, redemptionCode, adminUserID string) error
	ListAllCodexOrders(limit, offset int, status string) ([]AdminCodexOrder, int, error)
	ExpireCodexOrders() (int, error)
	RecordCodexRefund(orderNo, outRequestNo, reason string, amount float64, operatorID string, succeeded bool, alipayTradeNo, errMsg string) error
	CreateCodexProduct(sku, name, description string, priceCNY float64, sortOrder int, status string) (*CodexProduct, error)
	UpdateCodexProduct(id int, sku, name, description string, priceCNY float64, sortOrder int, status string) (*CodexProduct, error)
	DeleteCodexProduct(id int) error
}
