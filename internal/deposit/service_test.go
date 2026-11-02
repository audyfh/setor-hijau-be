package deposit

import (
	"testing"
)

func TestCalculateSubtotal(t *testing.T) {
	tests := []struct {
		name          string
		weightGrams   int64
		pricePerKgIDR int64
		expectedSub   int64
	}{
		{
			name:          "exact 1 kg",
			weightGrams:   1000,
			pricePerKgIDR: 5000,
			expectedSub:   5000,
		},
		{
			name:          "250 grams exact",
			weightGrams:   250,
			pricePerKgIDR: 4000,
			expectedSub:   1000,
		},
		{
			name:          "half-up round down (232.4 -> 232)",
			// 150 * 1549 = 232,350 -> +500 = 232,850 / 1000 = 232
			weightGrams:   150,
			pricePerKgIDR: 1549,
			expectedSub:   232,
		},
		{
			name:          "half-up round up (232.5 -> 233)",
			// 150 * 1550 = 232,500 -> +500 = 233,000 / 1000 = 233
			weightGrams:   150,
			pricePerKgIDR: 1550,
			expectedSub:   233,
		},
		{
			name:          "333 grams at 1000 idr/kg",
			// 333 * 1000 = 333,000 -> +500 = 333,500 / 1000 = 333
			weightGrams:   333,
			pricePerKgIDR: 1000,
			expectedSub:   333,
		},
		{
			name:          "zero weight",
			weightGrams:   0,
			pricePerKgIDR: 5000,
			expectedSub:   0,
		},
		{
			name:          "negative weight",
			weightGrams:   -100,
			pricePerKgIDR: 5000,
			expectedSub:   0,
		},
		{
			name:          "zero price",
			weightGrams:   1000,
			pricePerKgIDR: 0,
			expectedSub:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateSubtotal(tt.weightGrams, tt.pricePerKgIDR)
			if got != tt.expectedSub {
				t.Errorf("CalculateSubtotal(%d, %d) = %d; want %d", tt.weightGrams, tt.pricePerKgIDR, got, tt.expectedSub)
			}
		})
	}
}
