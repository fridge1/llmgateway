package proxy

import (
	"testing"

	"github.com/zhulang/llm-gateway/internal/store"
)

type planModelStore struct {
	store.Store
	models []string
	err    error
}

func (s *planModelStore) GetSubscriptionPlanModels(_ int) ([]string, error) {
	return s.models, s.err
}

func TestCheckModelAccess(t *testing.T) {
	planID := 1

	tests := []struct {
		name    string
		auth    *AuthResult
		model   string
		models  []string
		wantErr bool
	}{
		{
			name:    "no plan restriction allows any model",
			auth:    &AuthResult{User: &store.User{ID: "u1"}},
			model:   "any-model",
			wantErr: false,
		},
		{
			name:    "plan restriction allows model in list",
			auth:    &AuthResult{User: &store.User{ID: "u1"}, PlanID: &planID},
			model:   "claude-sonnet-4-6",
			models:  []string{"claude-sonnet-4-6", "claude-opus-4-7"},
			wantErr: false,
		},
		{
			name:    "plan restriction blocks model not in list",
			auth:    &AuthResult{User: &store.User{ID: "u1"}, PlanID: &planID},
			model:   "gpt-4o",
			models:  []string{"claude-sonnet-4-6", "claude-opus-4-7"},
			wantErr: true,
		},
		{
			name:    "tenant key ignores plan restriction",
			auth:    &AuthResult{TenantKey: &store.TenantAPIKey{ID: "tk1"}, PlanID: &planID},
			model:   "any-model",
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &Core{Store: &planModelStore{models: tc.models}}
			err := c.CheckModelAccess(tc.auth, tc.model)
			if (err != nil) != tc.wantErr {
				t.Errorf("CheckModelAccess() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestAuthResultTenantID(t *testing.T) {
	tests := []struct {
		name string
		auth *AuthResult
		want string
	}{
		{
			name: "user key has no tenant",
			auth: &AuthResult{User: &store.User{ID: "u1"}},
			want: "",
		},
		{
			name: "user key with tenant membership returns member tenant id",
			auth: &AuthResult{User: &store.User{ID: "u1"}, MemberTenantID: "t9"},
			want: "t9",
		},
		{
			name: "tenant key ignores member tenant id",
			auth: &AuthResult{TenantKey: &store.TenantAPIKey{TenantID: "t1"}, MemberTenantID: "t9"},
			want: "t1",
		},
		{
			name: "tenant key returns tenant id",
			auth: &AuthResult{TenantKey: &store.TenantAPIKey{TenantID: "t1"}},
			want: "t1",
		},
		{
			name: "sub-user key returns tenant id",
			auth: &AuthResult{
				SubUserKey: &store.TenantSubUserKey{TenantID: "t2"},
				SubUser:    &store.TenantSubUser{ID: "su1"},
			},
			want: "t2",
		},
		{
			name: "sub-user key without sub-user is not a sub-user auth",
			auth: &AuthResult{SubUserKey: &store.TenantSubUserKey{TenantID: "t2"}},
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.auth.TenantID(); got != tc.want {
				t.Errorf("TenantID() = %q, want %q", got, tc.want)
			}
		})
	}
}
