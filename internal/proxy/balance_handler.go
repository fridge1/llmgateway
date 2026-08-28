package proxy

import (
	"encoding/json"
	"net/http"

	"github.com/zhulang/llm-gateway/internal/httputil"
)

// ServeBalance handles GET /v1/balance — API-key authenticated balance query.
func (c *Core) ServeBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	auth, err := c.AuthenticateBearer(r)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid or missing API key", "auth_error", "invalid_api_key")
		return
	}

	resp, err := c.balancePayload(auth)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get balance", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// balancePayload builds the balance/quota response for the authenticated key,
// dispatching on key type (sub-user quota / tenant balance / user balance).
func (c *Core) balancePayload(auth *AuthResult) (map[string]any, error) {
	switch {
	case auth.IsSubUser():
		su := auth.SubUser
		resp := map[string]any{
			"type":        "sub_user",
			"quota_limit": su.QuotaLimit,
			"quota_used":  su.QuotaUsed,
			"currency":    "CNY",
		}
		if su.QuotaLimit != nil {
			resp["quota_remaining"] = *su.QuotaLimit - su.QuotaUsed
		}
		return resp, nil

	case auth.IsTenant():
		b, err := c.Store.GetTenantBalance(auth.TenantKey.TenantID)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"type":            "tenant",
			"balance":         b.Balance,
			"frozen":          b.Frozen,
			"available":       b.Balance - b.Frozen,
			"total_recharged": b.TotalRecharged,
			"total_consumed":  b.TotalConsumed,
			"currency":        "CNY",
		}, nil

	default:
		// A personal key whose owner belongs to a tenant is billed against the
		// tenant, so report the tenant balance the key actually spends from.
		if auth.MemberTenantID != "" {
			b, err := c.Store.GetTenantBalance(auth.MemberTenantID)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"type":            "tenant",
				"balance":         b.Balance,
				"frozen":          b.Frozen,
				"available":       b.Balance - b.Frozen,
				"total_recharged": b.TotalRecharged,
				"total_consumed":  b.TotalConsumed,
				"currency":        "CNY",
			}, nil
		}
		b, err := c.Store.GetBalance(auth.User.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"type":      "user",
			"balance":   b.Balance,
			"frozen":    b.Frozen,
			"available": b.Balance - b.Frozen,
			"currency":  "CNY",
		}, nil
	}
}
