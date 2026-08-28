package billing

import (
	"fmt"
	"math"
	"testing"

	"github.com/zhulang/llm-gateway/internal/store"
)

// mockStore implements only the methods BillingService uses.
type mockStore struct {
	store.Store // embed to satisfy interface; unused methods will panic

	pricing  *store.ModelPricing
	balance  *store.Balance
	frozenOK bool

	// tenant discount path
	primaryTenant string               // value returned by GetUserPrimaryPricingTenant
	tenantPricing *store.TenantPricing // value returned by GetTenantPricing

	// capture args from SettleBilling / DirectCharge
	settledActualCost float64
	chargedAmount     float64
	chargedModel      string
}

func (m *mockStore) GetPricing(_ string) (*store.ModelPricing, error) {
	if m.pricing == nil {
		return nil, fmt.Errorf("no pricing")
	}
	return m.pricing, nil
}

func (m *mockStore) GetUserPrimaryPricingTenant(_ string) (string, error) {
	return m.primaryTenant, nil
}

func (m *mockStore) GetTenantPricing(_, _ string) (*store.TenantPricing, error) {
	return m.tenantPricing, nil
}

func (m *mockStore) GetUserPricing(_, _ string) (*store.UserPricing, error) {
	return nil, nil
}

func (m *mockStore) GetBalance(_ string) (*store.Balance, error) {
	if m.balance == nil {
		return nil, fmt.Errorf("no balance")
	}
	return m.balance, nil
}

func (m *mockStore) FreezeBalance(_ string, _ float64) error {
	if !m.frozenOK {
		return fmt.Errorf("insufficient balance")
	}
	return nil
}

func (m *mockStore) UnfreezeBalance(_ string, _ float64) error { return nil }

func (m *mockStore) SettleBilling(_ string, _, actualCost float64, _, _ string, _ store.TokenUsage, _ string) error {
	m.settledActualCost = actualCost
	return nil
}

func (m *mockStore) DirectCharge(_ string, amount float64, model, _ string, _ store.TokenUsage, _ string) error {
	m.chargedAmount = amount
	m.chargedModel = model
	return nil
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// --- Settle tests ---

func TestSettle_NoCachedTokens(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:       10.0, // CNY per 1M tokens
			OutputPrice:      20.0,
			CachedInputPrice: 1.0,
			IsActive:         true,
		},
	}
	svc := NewBillingService(ms, nil, nil)

	usage := UsageInfo{PromptTokens: 1000, CompletionTokens: 500, CachedTokens: 0}
	err := svc.Settle("user1", 1.0, "model-a", "req1", usage, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// expected: (1000*10 + 500*20) / 1M = (10000+10000)/1M = 0.02
	expected := 0.02
	if !almostEqual(ms.settledActualCost, expected) {
		t.Errorf("expected cost %f, got %f", expected, ms.settledActualCost)
	}
}

func TestSettle_WithCachedTokens(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:       10.0,
			OutputPrice:      20.0,
			CachedInputPrice: 1.0, // 10% of input price
			IsActive:         true,
		},
	}
	svc := NewBillingService(ms, nil, nil)

	usage := UsageInfo{PromptTokens: 1000, CompletionTokens: 500, CacheReadTokens: 800, CacheTokensIncludedInPrompt: true}
	err := svc.Settle("user1", 1.0, "model-a", "req1", usage, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nonCached = 1000 - 800 = 200
	// cost = (200*10 + 800*1 + 500*20) / 1M = (2000+800+10000)/1M = 0.0128
	expected := 0.0128
	if !almostEqual(ms.settledActualCost, expected) {
		t.Errorf("expected cost %f, got %f", expected, ms.settledActualCost)
	}
}

func TestSettle_CachedExceedsPrompt(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:       10.0,
			OutputPrice:      20.0,
			CachedInputPrice: 1.0,
			IsActive:         true,
		},
	}
	svc := NewBillingService(ms, nil, nil)

	// CachedTokens > PromptTokens (edge case)
	usage := UsageInfo{PromptTokens: 500, CompletionTokens: 100, CacheReadTokens: 800, CacheTokensIncludedInPrompt: true}
	err := svc.Settle("user1", 1.0, "model-a", "req1", usage, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nonCached = max(500-800, 0) = 0
	// cost = (0*10 + 800*1 + 100*20) / 1M = (0+800+2000)/1M = 0.0028
	expected := 0.0028
	if !almostEqual(ms.settledActualCost, expected) {
		t.Errorf("expected cost %f, got %f", expected, ms.settledActualCost)
	}
}

func TestSettle_ZeroCachedInputPrice(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:       10.0,
			OutputPrice:      20.0,
			CachedInputPrice: 0.0, // no cached pricing set
			IsActive:         true,
		},
	}
	svc := NewBillingService(ms, nil, nil)

	usage := UsageInfo{PromptTokens: 1000, CompletionTokens: 500, CacheReadTokens: 800, CacheTokensIncludedInPrompt: true}
	err := svc.Settle("user1", 1.0, "model-a", "req1", usage, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nonCached = 200
	// cost = (200*10 + 800*0 + 500*20) / 1M = (2000+0+10000)/1M = 0.012
	expected := 0.012
	if !almostEqual(ms.settledActualCost, expected) {
		t.Errorf("expected cost %f, got %f", expected, ms.settledActualCost)
	}
}

// --- Charge tests ---

func TestCharge_WithCachedTokens(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:       10.0,
			OutputPrice:      20.0,
			CachedInputPrice: 1.0,
			IsActive:         true,
		},
	}
	svc := NewBillingService(ms, nil, nil)

	usage := UsageInfo{PromptTokens: 1000, CompletionTokens: 500, CacheReadTokens: 600, CacheTokensIncludedInPrompt: true}
	err := svc.Charge("user1", "model-a", "req1", usage, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nonCached = 400
	// cost = (400*10 + 600*1 + 500*20) / 1M = (4000+600+10000)/1M = 0.0146
	expected := 0.0146
	if !almostEqual(ms.chargedAmount, expected) {
		t.Errorf("expected charge %f, got %f", expected, ms.chargedAmount)
	}
}

func TestCharge_NoCachedTokens(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:       10.0,
			OutputPrice:      20.0,
			CachedInputPrice: 1.0,
			IsActive:         true,
		},
	}
	svc := NewBillingService(ms, nil, nil)

	usage := UsageInfo{PromptTokens: 1000, CompletionTokens: 500, CachedTokens: 0}
	err := svc.Charge("user1", "model-a", "req1", usage, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 0.02
	if !almostEqual(ms.chargedAmount, expected) {
		t.Errorf("expected charge %f, got %f", expected, ms.chargedAmount)
	}
}

// --- CheckBalance tests ---

func TestCheckBalance_OnlyBalance(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:  10.0,
			OutputPrice: 20.0,
			IsActive:    true,
		},
		balance: &store.Balance{
			Balance: 5.0,
			Frozen:  0,
		},
	}
	svc := NewBillingService(ms, nil, nil)
	err := svc.CheckBalance("user1", "model-a")
	if err != nil {
		t.Errorf("expected no error for user with balance, got: %v", err)
	}
}

func TestCheckBalance_AllFrozen(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:  10.0,
			OutputPrice: 20.0,
			IsActive:    true,
		},
		balance: &store.Balance{
			Balance: 10.0,
			Frozen:  10.0,
		},
	}
	svc := NewBillingService(ms, nil, nil)
	err := svc.CheckBalance("user1", "model-a")
	if err == nil {
		t.Error("expected error for fully frozen balance, got nil")
	}
}

// --- PreCharge tests ---

func TestPreCharge_UsesFullInputPrice(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:       10.0,
			OutputPrice:      20.0,
			CachedInputPrice: 1.0,
			IsActive:         true,
		},
		frozenOK: true,
	}
	svc := NewBillingService(ms, nil, nil)

	frozen, err := svc.PreCharge("user1", "model-a", 2000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// PreCharge uses full InputPrice (not cached) as upper bound estimate
	// estimated = (2000*10 + 2000*20) / 1M = 60000/1M = 0.06
	expected := 0.06
	if !almostEqual(frozen, expected) {
		t.Errorf("expected frozen %f, got %f", expected, frozen)
	}
}

// --- Tiered pricing tests ---

func TestCalculateCost_TieredPricing_Tier1(t *testing.T) {
	// GLM-5.1 tier 1: tokens < 32768 → input ¥6/M, output ¥24/M, cached ¥1.3/M
	p := &store.ModelPricing{
		InputPrice:       6.0,
		OutputPrice:      24.0,
		CachedInputPrice: 1.3,
		IsActive:         true,
		PricingTiers: []store.PricingTier{
			{MinTokens: 1, MaxTokens: 32768, InputPrice: 6, OutputPrice: 24, CachedInputPrice: 1.3},
			{MinTokens: 32768, MaxTokens: 204800, InputPrice: 8, OutputPrice: 28, CachedInputPrice: 2},
		},
	}

	usage := UsageInfo{PromptTokens: 1000, CompletionTokens: 500}
	cost := calculateCost(usage, p)

	// tier 1: (1000*6 + 500*24) / 1M = (6000+12000)/1M = 0.018
	expected := 0.018
	if !almostEqual(cost, expected) {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}

func TestCalculateCost_TieredPricing_Tier2(t *testing.T) {
	// GLM-5.1 tier 2: tokens >= 32768 → input ¥8/M, output ¥28/M, cached ¥2/M
	p := &store.ModelPricing{
		InputPrice:       6.0,
		OutputPrice:      24.0,
		CachedInputPrice: 1.3,
		IsActive:         true,
		PricingTiers: []store.PricingTier{
			{MinTokens: 1, MaxTokens: 32768, InputPrice: 6, OutputPrice: 24, CachedInputPrice: 1.3},
			{MinTokens: 32768, MaxTokens: 204800, InputPrice: 8, OutputPrice: 28, CachedInputPrice: 2},
		},
	}

	usage := UsageInfo{PromptTokens: 40000, CompletionTokens: 500}
	cost := calculateCost(usage, p)

	// tier 2: (40000*8 + 500*28) / 1M = (320000+14000)/1M = 0.334
	expected := 0.334
	if !almostEqual(cost, expected) {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}

func TestCalculateCost_TieredPricing_WithCachedTokens(t *testing.T) {
	p := &store.ModelPricing{
		InputPrice:       6.0,
		OutputPrice:      24.0,
		CachedInputPrice: 1.3,
		IsActive:         true,
		PricingTiers: []store.PricingTier{
			{MinTokens: 1, MaxTokens: 32768, InputPrice: 6, OutputPrice: 24, CachedInputPrice: 1.3},
			{MinTokens: 32768, MaxTokens: 204800, InputPrice: 8, OutputPrice: 28, CachedInputPrice: 2},
		},
	}

	// Total input = 30000 + 5000 = 35000 → tier 2
	usage := UsageInfo{PromptTokens: 30000, CompletionTokens: 500, CacheReadTokens: 5000}
	cost := calculateCost(usage, p)

	// tier 2: (30000*8 + 5000*2 + 500*28) / 1M = (240000+10000+14000)/1M = 0.264
	expected := 0.264
	if !almostEqual(cost, expected) {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}

func TestCharge_StoresCanonicalModelName(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:  10.0,
			OutputPrice: 20.0,
			IsActive:    true,
		},
	}
	svc := NewBillingService(ms, nil, nil)

	canonicalName := "pa/gpt-5.4-pro"
	usage := UsageInfo{PromptTokens: 1000, CompletionTokens: 500}

	err := svc.Charge("user1", canonicalName, "req1", usage, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ms.chargedModel != canonicalName {
		t.Errorf("expected model %q stored in transaction, got %q", canonicalName, ms.chargedModel)
	}
}

// --- Image billing tests ---

func TestCalculateCost_ImageBilling_SingleImage(t *testing.T) {
	p := &store.ModelPricing{
		InputPrice:  0.4,
		OutputPrice: 0.8,
		BillingType: "image",
		IsActive:    true,
	}
	usage := UsageInfo{PromptTokens: 275, CompletionTokens: 4000, ImageCount: 1}
	cost := calculateCost(usage, p)
	expected := 0.4
	if !almostEqual(cost, expected) {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}

func TestCalculateCost_ImageBilling_MultipleImages(t *testing.T) {
	p := &store.ModelPricing{
		InputPrice:  0.4,
		OutputPrice: 0.8,
		BillingType: "image",
		IsActive:    true,
	}
	usage := UsageInfo{PromptTokens: 500, CompletionTokens: 8000, ImageCount: 2}
	cost := calculateCost(usage, p)
	expected := 0.8
	if !almostEqual(cost, expected) {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}

func TestCalculateCost_ImageBilling_ZeroCountDefaultsToOne(t *testing.T) {
	p := &store.ModelPricing{
		InputPrice:  0.4,
		OutputPrice: 0.8,
		BillingType: "image",
		IsActive:    true,
	}
	usage := UsageInfo{PromptTokens: 275, CompletionTokens: 4000, ImageCount: 0}
	cost := calculateCost(usage, p)
	expected := 0.4
	if !almostEqual(cost, expected) {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}

// TestCalculateCost_AnthropicCachedTokens verifies that Anthropic's cache tokens
// are correctly handled (input_tokens includes cache_read_input_tokens).
func TestCalculateCost_AnthropicCachedTokens(t *testing.T) {
	p := &store.ModelPricing{
		InputPrice:       3000.0,  // $3.00 per MTok
		OutputPrice:      15000.0, // $15.00 per MTok
		CachedInputPrice: 300.0,   // $0.30 per MTok (10% of input)
		IsActive:         true,
	}

	// Scenario: Anthropic returns input_tokens=1000 (which includes 800 cached tokens)
	// and cache_read_input_tokens=800
	// Expected: only 200 tokens charged at input price, 800 at cached price
	usage := UsageInfo{
		PromptTokens:                1000,
		CompletionTokens:            500,
		CacheReadTokens:             800,
		CacheTokensIncludedInPrompt: true, // Anthropic behavior
	}

	cost := calculateCost(usage, p)

	// nonCached = 1000 - 800 = 200
	// cost = (200*3000 + 800*300 + 500*15000) / 1M
	//      = (600000 + 240000 + 7500000) / 1M
	//      = 8340000 / 1M = 8.34
	expected := 8.34
	if !almostEqual(cost, expected) {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}

// --- Tenant discount path (personal key) ---

func TestCharge_PersonalKeyAppliesTenantDiscount(t *testing.T) {
	rate := 0.5
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:  10.0,
			OutputPrice: 20.0,
			IsActive:    true,
		},
		primaryTenant: "tenant-1",
		tenantPricing: &store.TenantPricing{
			ModelName:    "model-a",
			DiscountRate: &rate,
			IsActive:     true,
		},
	}
	// No pricing cache -> exercises the no-cache getTenantPricing discount branch.
	svc := NewBillingService(ms, nil, nil)

	usage := UsageInfo{PromptTokens: 1000, CompletionTokens: 500}
	if err := svc.Charge("user1", "model-a", "req1", usage, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Global cost = (1000*10 + 500*20)/1M = 0.02; with 0.5 discount = 0.01
	expected := 0.01
	if !almostEqual(ms.chargedAmount, expected) {
		t.Errorf("expected discounted cost %f, got %f", expected, ms.chargedAmount)
	}
}

func TestCharge_NoTenantUsesGlobalPricing(t *testing.T) {
	ms := &mockStore{
		pricing: &store.ModelPricing{
			InputPrice:  10.0,
			OutputPrice: 20.0,
			IsActive:    true,
		},
		primaryTenant: "", // not in any pricing tenant
	}
	svc := NewBillingService(ms, nil, nil)

	usage := UsageInfo{PromptTokens: 1000, CompletionTokens: 500}
	if err := svc.Charge("user1", "model-a", "req1", usage, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 0.02 // full global price
	if !almostEqual(ms.chargedAmount, expected) {
		t.Errorf("expected global cost %f, got %f", expected, ms.chargedAmount)
	}
}
