package proxy

import (
	"testing"

	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/store"
)

func TestAsyncChargeAuth_MemberTenantScenario(t *testing.T) {
	// Test that tenant member using personal API key correctly sets tenantKeyID
	// in the billing job so that api_key_id is recorded in the database.

	c := &Core{
		billingCh: make(chan billingJob, 1),
	}

	auth := &AuthResult{
		User:           &store.User{ID: "user-123"},
		APIKeyID:       "key-abc-456",
		MemberTenantID: "tenant-789",
	}

	usage := billing.UsageInfo{
		PromptTokens:     10,
		CompletionTokens: 20,
	}

	// Call AsyncChargeAuth (skip subscription check for this test)
	c.AsyncChargeAuth(auth, "gpt-4", "req-001", usage)

	// Retrieve the job from the channel
	var job billingJob
	select {
	case job = <-c.billingCh:
		// Success
	default:
		t.Fatal("expected billing job to be queued")
	}

	// Verify the job fields
	if job.tenantID != "tenant-789" {
		t.Errorf("expected tenantID = tenant-789, got %q", job.tenantID)
	}

	// Key assertion: tenantKeyID should contain the personal API key ID
	if job.tenantKeyID != "key-abc-456" {
		t.Errorf("expected tenantKeyID = key-abc-456 (personal key), got %q", job.tenantKeyID)
	}

	// userID and apiKeyID should be empty (member tenant route uses tenant billing)
	if job.userID != "" {
		t.Errorf("expected userID to be empty for member tenant, got %q", job.userID)
	}

	if job.apiKeyID != "" {
		t.Errorf("expected apiKeyID to be empty for member tenant, got %q", job.apiKeyID)
	}

	if job.model != "gpt-4" {
		t.Errorf("expected model = gpt-4, got %q", job.model)
	}

	if job.requestID != "req-001" {
		t.Errorf("expected requestID = req-001, got %q", job.requestID)
	}
}
