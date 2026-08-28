package pricing

import (
	"testing"

	"github.com/zhulang/llm-gateway/internal/store"
)

func TestApplyDiscount(t *testing.T) {
	global := &store.ModelPricing{
		ModelName:            "m1",
		InputPrice:           10,
		OutputPrice:          20,
		CachedInputPrice:     2,
		CacheCreationPrice:   4,
		CacheCreation1hPrice: 8,
		IsActive:             true,
		PricingTiers: []store.PricingTier{
			{MinTokens: 0, MaxTokens: 1000, InputPrice: 10, OutputPrice: 20, CachedInputPrice: 2},
		},
	}

	got := applyDiscount(global, 0.5)

	if got.InputPrice != 5 || got.OutputPrice != 10 || got.CachedInputPrice != 1 ||
		got.CacheCreationPrice != 2 || got.CacheCreation1hPrice != 4 {
		t.Fatalf("scalar fields not discounted: %+v", got)
	}
	if len(got.PricingTiers) != 1 || got.PricingTiers[0].InputPrice != 5 || got.PricingTiers[0].OutputPrice != 10 {
		t.Fatalf("tier fields not discounted: %+v", got.PricingTiers)
	}
	// Original must be untouched.
	if global.InputPrice != 10 || global.PricingTiers[0].InputPrice != 10 {
		t.Fatalf("original pricing mutated: %+v", global)
	}
}

func TestApplyDiscountExported(t *testing.T) {
	global := &store.ModelPricing{InputPrice: 10, OutputPrice: 20, IsActive: true}
	got := ApplyDiscount(global, 0.8, false)
	if got.InputPrice != 8 || got.OutputPrice != 16 {
		t.Fatalf("unexpected discounted prices: %+v", got)
	}
	if got.IsActive {
		t.Fatalf("expected IsActive override to false")
	}
}
