package image

import (
	"testing"

	"github.com/zhulang/llm-gateway/internal/proxy"
	"github.com/zhulang/llm-gateway/internal/store"
)

func TestApplyBillingIdentity(t *testing.T) {
	tests := []struct {
		name string
		auth *proxy.AuthResult
		want store.ImageTask
	}{
		{
			name: "user key",
			auth: &proxy.AuthResult{
				User:     &store.User{ID: "user-1"},
				APIKeyID: "key-1",
			},
			want: store.ImageTask{UserID: "user-1", APIKeyID: "key-1"},
		},
		{
			name: "user key with tenant membership carries tenant id but no tenant key",
			auth: &proxy.AuthResult{
				User:           &store.User{ID: "user-1"},
				APIKeyID:       "key-1",
				MemberTenantID: "tenant-1",
			},
			want: store.ImageTask{UserID: "user-1", APIKeyID: "key-1", TenantID: "tenant-1"},
		},
		{
			name: "tenant key",
			auth: &proxy.AuthResult{
				TenantKey: &store.TenantAPIKey{ID: "tkey-1", TenantID: "tenant-1"},
			},
			want: store.ImageTask{UserID: "tenant-1", TenantID: "tenant-1", TenantKeyID: "tkey-1"},
		},
		{
			name: "sub-user key",
			auth: &proxy.AuthResult{
				SubUser:    &store.TenantSubUser{ID: "su-1"},
				SubUserKey: &store.TenantSubUserKey{ID: "sukey-1", TenantID: "tenant-9"},
			},
			want: store.ImageTask{UserID: "su-1", SubUserID: "su-1", SubUserKeyID: "sukey-1", TenantID: "tenant-9"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var task store.ImageTask
			applyBillingIdentity(&task, tt.auth)
			if task.UserID != tt.want.UserID ||
				task.TenantID != tt.want.TenantID ||
				task.TenantKeyID != tt.want.TenantKeyID ||
				task.SubUserID != tt.want.SubUserID ||
				task.SubUserKeyID != tt.want.SubUserKeyID ||
				task.APIKeyID != tt.want.APIKeyID {
				t.Errorf("applyBillingIdentity() = %+v, want %+v", task, tt.want)
			}
		})
	}
}

func TestPublicTaskOwnedBy(t *testing.T) {
	shareKeyID := "share-1"
	userAuth := &proxy.AuthResult{User: &store.User{ID: "user-1"}, APIKeyID: "key-1"}
	memberAuth := &proxy.AuthResult{User: &store.User{ID: "user-1"}, APIKeyID: "key-1", MemberTenantID: "tenant-1"}
	tenantAuth := &proxy.AuthResult{TenantKey: &store.TenantAPIKey{ID: "tkey-1", TenantID: "tenant-1"}}
	subUserAuth := &proxy.AuthResult{
		SubUser:    &store.TenantSubUser{ID: "su-1"},
		SubUserKey: &store.TenantSubUserKey{ID: "sukey-1", TenantID: "tenant-1"},
	}

	memberTask := store.ImageTask{UserID: "user-1", APIKeyID: "key-1", TenantID: "tenant-1"}
	tenantKeyTask := store.ImageTask{UserID: "tenant-1", TenantID: "tenant-1", TenantKeyID: "tkey-1"}
	subUserTask := store.ImageTask{UserID: "su-1", SubUserID: "su-1", SubUserKeyID: "sukey-1", TenantID: "tenant-1"}
	plainUserTask := store.ImageTask{UserID: "user-1", APIKeyID: "key-1"}
	shareTask := store.ImageTask{UserID: "user-1", ImageShareKeyID: &shareKeyID}

	tests := []struct {
		name string
		task store.ImageTask
		auth *proxy.AuthResult
		want bool
	}{
		{"user owns own task", plainUserTask, userAuth, true},
		{"member owns own tenant-billed task", memberTask, memberAuth, true},
		{"tenant key does not own member personal task", memberTask, tenantAuth, false},
		{"tenant key owns its own task", tenantKeyTask, tenantAuth, true},
		{"member does not own tenant-key task", tenantKeyTask, memberAuth, false},
		{"user does not own another user's task", store.ImageTask{UserID: "user-2"}, userAuth, false},
		{"user does not own image-share task", shareTask, userAuth, false},
		{"sub-user owns own task", subUserTask, subUserAuth, true},
		{"tenant key does not own sub-user task", subUserTask, tenantAuth, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.task
			if got := publicTaskOwnedBy(&task, tt.auth); got != tt.want {
				t.Errorf("publicTaskOwnedBy() = %v, want %v", got, tt.want)
			}
		})
	}
}
