package proxy

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/zhulang/llm-gateway/internal/store"
)

type balanceStore struct {
	store.Store
	userBal   *store.Balance
	tenantBal *store.TenantBalance
	err       error
}

func (s *balanceStore) GetBalance(_ string) (*store.Balance, error) {
	return s.userBal, s.err
}
func (s *balanceStore) GetTenantBalance(_ string) (*store.TenantBalance, error) {
	return s.tenantBal, s.err
}

func TestBalancePayload(t *testing.T) {
	limit := 50.0
	tests := []struct {
		name      string
		auth      *AuthResult
		st        *balanceStore
		wantErr   bool
		wantType  string
		wantKeys  []string
		noKey     string
	}{
		{
			name: "user key",
			auth: &AuthResult{User: &store.User{ID: "u1"}},
			st:   &balanceStore{userBal: &store.Balance{Balance: 100, Frozen: 5}},
			wantType: "user",
			wantKeys: []string{"balance", "frozen", "available", "currency"},
		},
		{
			name: "tenant key",
			auth: &AuthResult{TenantKey: &store.TenantAPIKey{TenantID: "t1"}},
			st:   &balanceStore{tenantBal: &store.TenantBalance{Balance: 200, Frozen: 10, TotalRecharged: 500, TotalConsumed: 300}},
			wantType: "tenant",
			wantKeys: []string{"balance", "frozen", "available", "total_recharged", "total_consumed", "currency"},
		},
		{
			name: "sub_user key with quota limit",
			auth: &AuthResult{
				SubUserKey: &store.TenantSubUserKey{},
				SubUser:    &store.TenantSubUser{QuotaLimit: &limit, QuotaUsed: 12.34},
			},
			st:       &balanceStore{},
			wantType: "sub_user",
			wantKeys: []string{"quota_limit", "quota_used", "quota_remaining", "currency"},
		},
		{
			name: "sub_user key unlimited",
			auth: &AuthResult{
				SubUserKey: &store.TenantSubUserKey{},
				SubUser:    &store.TenantSubUser{QuotaLimit: nil, QuotaUsed: 5},
			},
			st:       &balanceStore{},
			wantType: "sub_user",
			wantKeys: []string{"quota_limit", "quota_used", "currency"},
			noKey:    "quota_remaining",
		},
		{
			name:    "store error user key",
			auth:    &AuthResult{User: &store.User{ID: "u1"}},
			st:      &balanceStore{err: fmt.Errorf("db error")},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &Core{Store: tc.st}
			resp, err := c.balancePayload(tc.auth)
			if (err != nil) != tc.wantErr {
				t.Fatalf("wantErr=%v, got err=%v", tc.wantErr, err)
			}
			if tc.wantErr {
				return
			}
			if resp["type"] != tc.wantType {
				t.Errorf("type = %v, want %v", resp["type"], tc.wantType)
			}
			for _, k := range tc.wantKeys {
				if _, ok := resp[k]; !ok {
					t.Errorf("missing key %q in response", k)
				}
			}
			if tc.noKey != "" {
				if _, ok := resp[tc.noKey]; ok {
					t.Errorf("key %q should be absent but is present", tc.noKey)
				}
			}
			// available = balance - frozen for user/tenant
			if tc.wantType == "user" {
				data, _ := json.Marshal(resp)
				var m map[string]float64
				json.Unmarshal(data, &m)
				if m["available"] != m["balance"]-m["frozen"] {
					t.Errorf("available %v != balance %v - frozen %v", m["available"], m["balance"], m["frozen"])
				}
			}
		})
	}
}
