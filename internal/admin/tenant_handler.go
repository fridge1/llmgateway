package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/store"
)

// TenantPricingInvalidator is used to invalidate tenant pricing cache entries.
type TenantPricingInvalidator interface {
	InvalidateTenant(tenantID, modelName string)
}

// TenantAuditSink receives tenant-scoped audit events. Declared here (rather
// than importing internal/audit) to avoid an import cycle — audit imports
// admin for the context keys.
type TenantAuditSink interface {
	LogTenant(r *http.Request, tenantID, action, resourceType, resourceID string, details map[string]any)
}

// KeyCacheInvalidator drops cached API-key auth entries for a user. Called when
// tenant membership changes so the proxy re-resolves MemberTenantID promptly
// instead of waiting out the cache TTL.
type KeyCacheInvalidator interface {
	InvalidateUser(userID string)
}

// TenantHandler provides tenant management API endpoints.
type TenantHandler struct {
	store              store.Store
	pricingInvalidator TenantPricingInvalidator
	audit              TenantAuditSink
	onUpstreamsUpdate  func() // rebuilds the router after upstream override changes
	keyCache           KeyCacheInvalidator
}

// NewTenantHandler creates a TenantHandler.
func NewTenantHandler(s store.Store, pi TenantPricingInvalidator) *TenantHandler {
	return &TenantHandler{store: s, pricingInvalidator: pi}
}

// SetKeyCache wires the API-key cache invalidator (optional).
func (h *TenantHandler) SetKeyCache(kc KeyCacheInvalidator) { h.keyCache = kc }

// invalidateUserKeys drops a user's cached key auth after membership changes.
func (h *TenantHandler) invalidateUserKeys(userID string) {
	if h.keyCache != nil && userID != "" {
		h.keyCache.InvalidateUser(userID)
	}
}

// SetAudit wires the tenant audit sink (optional).
func (h *TenantHandler) SetAudit(a TenantAuditSink) { h.audit = a }

// auditTenant records a tenant-scoped audit event when a sink is wired.
func (h *TenantHandler) auditTenant(r *http.Request, tenantID, action, resourceType, resourceID string, details map[string]any) {
	if h.audit != nil {
		h.audit.LogTenant(r, tenantID, action, resourceType, resourceID, details)
	}
}

// ---------------------------------------------------------------------------
// Tenant CRUD
// ---------------------------------------------------------------------------

type createTenantRequest struct {
	Name string `json:"name"`
}

func (h *TenantHandler) HandleCreateTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	userID, _ := r.Context().Value(CtxUserIDKey).(string)
	var req createTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "name is required", "invalid_request_error", "missing_name")
		return
	}

	tenant, err := h.store.CreateTenant(strings.TrimSpace(req.Name), userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create tenant", "api_error", "create_failed")
		return
	}
	h.invalidateUserKeys(userID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tenant)
}

func (h *TenantHandler) HandleListTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	userID, _ := r.Context().Value(CtxUserIDKey).(string)
	tenants, err := h.store.ListTenantsByUser(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list tenants", "api_error", "list_failed")
		return
	}
	if tenants == nil {
		tenants = []store.TenantWithRole{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

func (h *TenantHandler) HandleGetTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	tenant, err := h.store.GetTenantByID(tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "tenant not found", "invalid_request_error", "not_found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenant)
}

type updateTenantRequest struct {
	Name string `json:"name"`
}

func (h *TenantHandler) HandleUpdateTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	var req updateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "name is required", "invalid_request_error", "missing_name")
		return
	}

	if err := h.store.UpdateTenant(tenantID, strings.TrimSpace(req.Name)); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update tenant", "api_error", "update_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *TenantHandler) HandleDeleteTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	if err := h.store.DeleteTenant(tenantID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete tenant", "api_error", "delete_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Members
// ---------------------------------------------------------------------------

type inviteMemberRequest struct {
	Phone string `json:"phone"`
	Role  string `json:"role"`
}

func (h *TenantHandler) HandleInviteMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	userID, _ := r.Context().Value(CtxUserIDKey).(string)

	var req inviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.Phone == "" {
		httputil.WriteError(w, http.StatusBadRequest, "phone is required", "invalid_request_error", "missing_phone")
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}
	if req.Role != "admin" && req.Role != "member" {
		httputil.WriteError(w, http.StatusBadRequest, "role must be admin or member", "invalid_request_error", "invalid_role")
		return
	}

	inv, err := h.store.CreateTenantInvitation(tenantID, req.Phone, req.Role, userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create invitation", "api_error", "invite_failed")
		return
	}
	h.auditTenant(r, tenantID, "tenant_member_invite", "tenant_invitation", inv.ID, map[string]any{"phone": req.Phone, "role": req.Role})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inv)
}

func (h *TenantHandler) HandleListMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	members, err := h.store.ListTenantMembers(tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list members", "api_error", "list_failed")
		return
	}
	if members == nil {
		members = []store.TenantMember{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func (h *TenantHandler) HandleRemoveMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	memberUID := extractLastSegment(r.URL.Path)

	// Cannot remove the owner
	tenant, err := h.store.GetTenantByID(tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "tenant not found", "invalid_request_error", "not_found")
		return
	}
	if tenant.OwnerID == memberUID {
		httputil.WriteError(w, http.StatusBadRequest, "cannot remove the owner", "invalid_request_error", "cannot_remove_owner")
		return
	}

	if err := h.store.RemoveTenantMember(tenantID, memberUID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to remove member", "api_error", "remove_failed")
		return
	}
	h.invalidateUserKeys(memberUID)
	h.auditTenant(r, tenantID, "tenant_member_remove", "tenant_member", memberUID, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type updateRoleRequest struct {
	Role string `json:"role"`
}

func (h *TenantHandler) HandleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	// Path: /api/tenants/{id}/members/{uid}/role
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	var memberUID string
	for i, p := range parts {
		if p == "members" && i+1 < len(parts) {
			memberUID = parts[i+1]
			break
		}
	}

	var req updateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.Role != "admin" && req.Role != "member" {
		httputil.WriteError(w, http.StatusBadRequest, "role must be admin or member", "invalid_request_error", "invalid_role")
		return
	}

	if err := h.store.UpdateTenantMemberRole(tenantID, memberUID, req.Role); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update role", "api_error", "update_failed")
		return
	}
	h.auditTenant(r, tenantID, "tenant_member_role", "tenant_member", memberUID, map[string]any{"role": req.Role})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type transferRequest struct {
	NewOwnerID string `json:"new_owner_id"`
}

func (h *TenantHandler) HandleTransferOwnership(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	var req transferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.NewOwnerID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "new_owner_id is required", "invalid_request_error", "missing_owner")
		return
	}

	if err := h.store.TransferTenantOwnership(tenantID, req.NewOwnerID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to transfer ownership", "api_error", "transfer_failed")
		return
	}
	h.auditTenant(r, tenantID, "tenant_ownership_transfer", "tenant", tenantID, map[string]any{"new_owner_id": req.NewOwnerID})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Tenant API Keys
// ---------------------------------------------------------------------------

type createTenantKeyRequest struct {
	Name string `json:"name"`
}

func (h *TenantHandler) HandleListTenantKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	keys, err := h.store.ListTenantAPIKeys(tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list keys", "api_error", "list_failed")
		return
	}
	if keys == nil {
		keys = []store.TenantAPIKey{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

func (h *TenantHandler) HandleCreateTenantKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	userID, _ := r.Context().Value(CtxUserIDKey).(string)

	var req createTenantKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	rawKey, keyHash, keyPrefix := generateAPIKey()

	key, err := h.store.CreateTenantAPIKey(tenantID, keyHash, keyPrefix, req.Name, userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create key", "api_error", "create_failed")
		return
	}
	h.auditTenant(r, tenantID, "tenant_key_create", "tenant_api_key", key.ID, map[string]any{"name": req.Name})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"key":    rawKey,
		"id":     key.ID,
		"name":   key.Name,
		"prefix": key.KeyPrefix,
	})
}

func (h *TenantHandler) HandleDeleteTenantKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	keyID := extractLastSegment(r.URL.Path)

	if err := h.store.DeleteTenantAPIKey(keyID, tenantID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete key", "api_error", "delete_failed")
		return
	}
	h.auditTenant(r, tenantID, "tenant_key_delete", "tenant_api_key", keyID, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Tenant Balance
// ---------------------------------------------------------------------------

func (h *TenantHandler) HandleGetTenantBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")

	tenant, err := h.store.GetTenantByID(tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get tenant", "api_error", "get_failed")
		return
	}

	ownerBal, err := h.store.GetBalance(tenant.OwnerID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get balance", "api_error", "get_failed")
		return
	}

	var totalRecharged, totalConsumed float64
	if tenantBal, err := h.store.GetTenantBalance(tenantID); err == nil {
		totalRecharged = tenantBal.TotalRecharged
		totalConsumed = tenantBal.TotalConsumed
	}

	resp := store.TenantBalance{
		TenantID:       tenantID,
		Balance:        ownerBal.Balance - ownerBal.Frozen,
		Frozen:         ownerBal.Frozen,
		TotalRecharged: totalRecharged,
		TotalConsumed:  totalConsumed,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *TenantHandler) HandleListTenantTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	typeFilter := r.URL.Query().Get("type")

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	txns, total, err := h.store.ListTenantTransactions(tenantID, limit, offset, typeFilter, nil, nil)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list transactions", "api_error", "list_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"transactions": txns,
		"total":        total,
	})
}

// ---------------------------------------------------------------------------
// Invitations
// ---------------------------------------------------------------------------

func (h *TenantHandler) HandleListInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	invitations, err := h.store.ListTenantInvitations(tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list invitations", "api_error", "list_failed")
		return
	}
	if invitations == nil {
		invitations = []store.TenantInvitation{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invitations)
}

func (h *TenantHandler) HandlePendingInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	phone, _ := r.Context().Value(CtxPhoneKey).(string)
	invitations, err := h.store.GetPendingInvitationsByPhone(phone)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get invitations", "api_error", "get_failed")
		return
	}
	if invitations == nil {
		invitations = []store.TenantInvitation{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invitations)
}

func (h *TenantHandler) HandleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	userID, _ := r.Context().Value(CtxUserIDKey).(string)
	invitationID := extractLastSegment(r.URL.Path)

	if err := h.store.AcceptTenantInvitation(invitationID, userID); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "accept_failed")
		return
	}
	h.invalidateUserKeys(userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *TenantHandler) HandleDeleteInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	invitationID := extractLastSegment(r.URL.Path)
	if err := h.store.DeleteTenantInvitation(invitationID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete invitation", "api_error", "delete_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Admin: Recharge tenant (platform admin only)
// ---------------------------------------------------------------------------

type rechargeTenantRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

func (h *TenantHandler) HandleRechargeTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	operatorID, _ := r.Context().Value(CtxUserIDKey).(string)

	var req rechargeTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if req.Amount <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "amount must be positive", "invalid_request_error", "invalid_amount")
		return
	}
	if req.Description == "" {
		req.Description = "管理员充值"
	}

	if err := h.store.RechargeTenant(tenantID, req.Amount, operatorID, req.Description); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to recharge", "api_error", "recharge_failed")
		return
	}
	h.auditTenant(r, tenantID, "tenant_recharge", "tenant", tenantID, map[string]any{"amount": req.Amount, "description": req.Description})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func generateAPIKey() (raw, hash, prefix string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	raw = "sk-" + hex.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(h[:])
	prefix = raw[:12]
	return
}

// extractPathSegment returns the segment immediately after the given key in the URL path.
func extractPathSegment(path, key string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, p := range parts {
		if p == key && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractLastSegment returns the last non-empty segment of the URL path.
func extractLastSegment(path string) string {
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// GetTenantMemberFromContext retrieves the tenant member from context (set by TenantRoleRequired middleware).
func GetTenantMemberFromContext(r *http.Request) *store.TenantMember {
	m, _ := r.Context().Value(CtxTenantMemberKey).(*store.TenantMember)
	return m
}

// ---------------------------------------------------------------------------
// Platform admin endpoints
// ---------------------------------------------------------------------------

// HandleAdminListTenants lists all tenants (platform admin only).
func (h *TenantHandler) HandleAdminListTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	tenants, total, err := h.store.ListAllTenants(limit, offset)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list tenants", "api_error", "list_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"tenants": tenants,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// ---------------------------------------------------------------------------
// Enterprise Tenant Management
// ---------------------------------------------------------------------------

type createEnterpriseTenantRequest struct {
	Name         string `json:"name"`
	OwnerID      string `json:"owner_id"`
	ContactPhone string `json:"contact_phone"`
	ContactEmail string `json:"contact_email"`
}

func (h *TenantHandler) HandleCreateEnterpriseTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	adminID, _ := r.Context().Value(CtxUserIDKey).(string)
	var req createEnterpriseTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.OwnerID) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "name and owner_id are required", "invalid_request_error", "missing_fields")
		return
	}

	tenant, err := h.store.CreateEnterpriseTenant(
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.OwnerID),
		adminID,
		strings.TrimSpace(req.ContactPhone),
		strings.TrimSpace(req.ContactEmail),
	)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create enterprise tenant", "api_error", "create_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tenant)
}

type updateEnterpriseInfoRequest struct {
	ContactPhone string `json:"contact_phone"`
	ContactEmail string `json:"contact_email"`
}

func (h *TenantHandler) HandleUpdateEnterpriseInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	tenantID := extractPathSegment(r.URL.Path, "tenants")
	var req updateEnterpriseInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	if err := h.store.UpdateTenantEnterpriseInfo(tenantID, req.ContactPhone, req.ContactEmail); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update enterprise info", "api_error", "update_failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Tenant Pricing Overrides
// ---------------------------------------------------------------------------

type upsertTenantPricingRequest struct {
	InputPrice           float64             `json:"input_price"`
	OutputPrice          float64             `json:"output_price"`
	CachedInputPrice     float64             `json:"cached_input_price"`
	CacheCreationPrice   float64             `json:"cache_creation_price"`
	CacheCreation1hPrice float64             `json:"cache_creation_1h_price"`
	BillingType          string              `json:"billing_type"`
	IsActive             *bool               `json:"is_active"`
	PricingTiers         []store.PricingTier `json:"pricing_tiers,omitempty"`
	DiscountRate         *float64            `json:"discount_rate,omitempty"`
}

// HandleListTenantPricing returns all pricing overrides for a tenant.
// GET /api/admin/tenants/{id}/pricing
func (h *TenantHandler) HandleListTenantPricing(w http.ResponseWriter, r *http.Request) {
	tenantID := extractPathSegment(r.URL.Path, "tenants")
	prices, err := h.store.ListTenantPricing(tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list tenant pricing", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"pricing": prices,
	})
}

// HandleUpsertTenantPricing creates or updates a tenant pricing override.
// PUT /api/admin/tenants/{id}/pricing/{model...}
func (h *TenantHandler) HandleUpsertTenantPricing(w http.ResponseWriter, r *http.Request) {
	tenantID := extractPathSegment(r.URL.Path, "tenants")
	modelName := extractTenantPricingModel(r.URL.Path)
	if modelName == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model name required", "invalid_request_error", "missing_model")
		return
	}

	var req upsertTenantPricingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}
	if req.InputPrice < 0 || req.OutputPrice < 0 || req.CacheCreationPrice < 0 {
		httputil.WriteError(w, http.StatusBadRequest, "prices must be non-negative", "invalid_request_error", "invalid_price")
		return
	}
	if req.DiscountRate != nil && (*req.DiscountRate <= 0 || *req.DiscountRate > 10) {
		httputil.WriteError(w, http.StatusBadRequest, "pricing_factor must be in (0, 10]", "invalid_request_error", "invalid_pricing_factor")
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	adminUserID, _ := r.Context().Value(CtxUserIDKey).(string)

	if err := h.store.UpsertTenantPricing(
		tenantID, modelName,
		req.InputPrice, req.OutputPrice, req.CachedInputPrice,
		req.CacheCreationPrice, req.CacheCreation1hPrice,
		req.BillingType, isActive, req.PricingTiers,
		req.DiscountRate, adminUserID,
	); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to upsert tenant pricing", "server_error", "db_error")
		return
	}

	if h.pricingInvalidator != nil {
		h.pricingInvalidator.InvalidateTenant(tenantID, modelName)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleDeleteTenantPricing removes a tenant pricing override.
// DELETE /api/admin/tenants/{id}/pricing/{model...}
func (h *TenantHandler) HandleDeleteTenantPricing(w http.ResponseWriter, r *http.Request) {
	tenantID := extractPathSegment(r.URL.Path, "tenants")
	modelName := extractTenantPricingModel(r.URL.Path)
	if modelName == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model name required", "invalid_request_error", "missing_model")
		return
	}

	if err := h.store.DeleteTenantPricing(tenantID, modelName); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete tenant pricing", "server_error", "db_error")
		return
	}

	if h.pricingInvalidator != nil {
		h.pricingInvalidator.InvalidateTenant(tenantID, modelName)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// extractTenantPathModel extracts the model name following marker from a path
// like /api/admin/tenants/{id}/pricing/{model...}. Model names may contain
// slashes (e.g. "pa/claude-sonnet-4-6").
func extractTenantPathModel(path, marker string) string {
	idx := strings.Index(path, marker)
	if idx < 0 {
		return ""
	}
	raw := strings.TrimSuffix(path[idx+len(marker):], "/")
	if decoded, err := url.PathUnescape(raw); err == nil {
		return decoded
	}
	return raw
}

// extractTenantPricingModel extracts the model name from a path like
// /api/admin/tenants/{id}/pricing/{model...}
func extractTenantPricingModel(path string) string {
	return extractTenantPathModel(path, "/pricing/")
}

// ---------------------------------------------------------------------------
// Tenant Model Upstream Overrides
// ---------------------------------------------------------------------------

type tenantUpstreamRequest struct {
	Provider         string   `json:"provider"`
	Protocol         string   `json:"protocol"`
	Protocols        []string `json:"protocols"`
	UpstreamProvider string   `json:"upstream_provider"`
	UpstreamName     string   `json:"upstream_name"`
	BaseURL          string   `json:"base_url"`
	APIKey           string   `json:"api_key"`
	ModelOverride    string   `json:"model_override"`
	Weight           int      `json:"weight"`
}

type replaceTenantUpstreamsRequest struct {
	Upstreams []tenantUpstreamRequest `json:"upstreams"`
}

// SetOnUpstreamsUpdate wires the callback invoked after tenant upstream
// overrides change (used to rebuild the router).
func (h *TenantHandler) SetOnUpstreamsUpdate(fn func()) { h.onUpstreamsUpdate = fn }

func (h *TenantHandler) notifyUpstreamsUpdate() {
	if h.onUpstreamsUpdate != nil {
		h.onUpstreamsUpdate()
	}
}

// HandleListTenantModelUpstreams returns all upstream overrides for a tenant.
// GET /api/admin/tenants/{id}/model-upstreams
func (h *TenantHandler) HandleListTenantModelUpstreams(w http.ResponseWriter, r *http.Request) {
	tenantID := extractPathSegment(r.URL.Path, "tenants")
	ups, err := h.store.ListTenantModelUpstreams(tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list tenant model upstreams", "server_error", "db_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"upstreams": ups,
	})
}

// HandleReplaceTenantModelUpstreams replaces the upstream override list for
// one tenant+model.
// PUT /api/admin/tenants/{id}/model-upstreams/{model...}
func (h *TenantHandler) HandleReplaceTenantModelUpstreams(w http.ResponseWriter, r *http.Request) {
	tenantID := extractPathSegment(r.URL.Path, "tenants")
	modelName := extractTenantPathModel(r.URL.Path, "/model-upstreams/")
	if modelName == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model name required", "invalid_request_error", "missing_model")
		return
	}

	var req replaceTenantUpstreamsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", "invalid_request_error", "bad_json")
		return
	}
	if len(req.Upstreams) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "upstreams must not be empty (use DELETE to remove the override)", "invalid_request_error", "empty_upstreams")
		return
	}
	for _, u := range req.Upstreams {
		if strings.TrimSpace(u.Provider) == "" || strings.TrimSpace(u.BaseURL) == "" || strings.TrimSpace(u.APIKey) == "" {
			httputil.WriteError(w, http.StatusBadRequest, "provider, base_url and api_key are required for each upstream", "invalid_request_error", "invalid_upstream")
			return
		}
	}
	// Guard against typos: the model must exist in the global table.
	if m, err := h.store.GetModelByName(modelName); err != nil || m == nil {
		httputil.WriteError(w, http.StatusBadRequest, "model not found: "+modelName, "invalid_request_error", "model_not_found")
		return
	}

	ups := make([]store.TenantModelUpstream, len(req.Upstreams))
	for i, u := range req.Upstreams {
		weight := u.Weight
		if weight <= 0 {
			weight = 1
		}
		ups[i] = store.TenantModelUpstream{
			Provider:         u.Provider,
			Protocol:         u.Protocol,
			Protocols:        u.Protocols,
			UpstreamProvider: u.UpstreamProvider,
			UpstreamName:     u.UpstreamName,
			BaseURL:          u.BaseURL,
			APIKey:           u.APIKey,
			ModelOverride:    u.ModelOverride,
			Weight:           weight,
		}
	}

	if err := h.store.ReplaceTenantModelUpstreams(tenantID, modelName, ups); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to replace tenant model upstreams", "server_error", "db_error")
		return
	}

	h.notifyUpstreamsUpdate()
	h.auditTenant(r, tenantID, "tenant.model_upstreams.replace", "model", modelName, map[string]any{
		"upstream_count": len(ups),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleDeleteTenantModelUpstreams removes all upstream overrides for one
// tenant+model, restoring the global default pool.
// DELETE /api/admin/tenants/{id}/model-upstreams/{model...}
func (h *TenantHandler) HandleDeleteTenantModelUpstreams(w http.ResponseWriter, r *http.Request) {
	tenantID := extractPathSegment(r.URL.Path, "tenants")
	modelName := extractTenantPathModel(r.URL.Path, "/model-upstreams/")
	if modelName == "" {
		httputil.WriteError(w, http.StatusBadRequest, "model name required", "invalid_request_error", "missing_model")
		return
	}

	if err := h.store.DeleteTenantModelUpstreams(tenantID, modelName); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete tenant model upstreams", "server_error", "db_error")
		return
	}

	h.notifyUpstreamsUpdate()
	h.auditTenant(r, tenantID, "tenant.model_upstreams.delete", "model", modelName, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

