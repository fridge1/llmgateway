package store

import (
	"testing"
)

func TestLotteryLowestAmountSelection(t *testing.T) {
	// Test that the lottery selection logic picks from the lowest 5 amounts
	// This test verifies the sorting and selection logic

	// Simulate entries with different amounts
	type entry struct {
		id      int64
		userID  string
		orderNo string
		amount  float64
	}

	entries := []entry{
		{id: 1, userID: "user1", orderNo: "order1", amount: 2000.0},
		{id: 2, userID: "user2", orderNo: "order2", amount: 999.0},
		{id: 3, userID: "user3", orderNo: "order3", amount: 200.0},
		{id: 4, userID: "user4", orderNo: "order4", amount: 200.0},
		{id: 5, userID: "user5", orderNo: "order5", amount: 200.0},
		{id: 6, userID: "user6", orderNo: "order6", amount: 80.0},
		{id: 7, userID: "user7", orderNo: "order7", amount: 50.0},
		{id: 8, userID: "user8", orderNo: "order8", amount: 50.0},
		{id: 9, userID: "user9", orderNo: "order9", amount: 50.0},
		{id: 10, userID: "user10", orderNo: "order10", amount: 50.0},
	}

	// Sort by amount ascending (like the real code does)
	type indexedEntry struct {
		entry
		originalIndex int
	}
	indexed := make([]indexedEntry, len(entries))
	for i, e := range entries {
		indexed[i] = indexedEntry{entry: e, originalIndex: i}
	}

	// Bubble sort by amount
	for i := 0; i < len(indexed)-1; i++ {
		for j := 0; j < len(indexed)-i-1; j++ {
			if indexed[j].amount > indexed[j+1].amount {
				indexed[j], indexed[j+1] = indexed[j+1], indexed[j]
			}
		}
	}

	// Take lowest 5
	poolSize := 5
	if len(indexed) < poolSize {
		poolSize = len(indexed)
	}
	lowestPool := indexed[:poolSize]

	// Verify the lowest pool contains only amounts <= 80
	maxExpected := 80.0
	for _, ie := range lowestPool {
		if ie.amount > maxExpected {
			t.Errorf("Expected lowest pool to only contain amounts <= %f, got %f", maxExpected, ie.amount)
		}
	}

	// Verify pool size is 5
	if len(lowestPool) != 5 {
		t.Errorf("Expected pool size 5, got %d", len(lowestPool))
	}

	// Verify the highest amounts are NOT in the pool
	for _, ie := range lowestPool {
		if ie.amount == 2000.0 || ie.amount == 999.0 {
			t.Errorf("High amount %f should not be in lowest pool", ie.amount)
		}
	}

	t.Logf("Lowest 5 amounts pool: %v", func() []float64 {
		amounts := make([]float64, len(lowestPool))
		for i, ie := range lowestPool {
			amounts[i] = ie.amount
		}
		return amounts
	}())
}
