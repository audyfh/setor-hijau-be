package cashbook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateSubtotal(t *testing.T) {
	tests := []struct {
		name          string
		weightGrams   int64
		pricePerKgIDR int64
		expectedIDR   int64
	}{
		{
			name:          "Exact 1 kg at Rp 3,000/kg",
			weightGrams:   1000,
			pricePerKgIDR: 3000,
			expectedIDR:   3000,
		},
		{
			name:          "2.5 kg at Rp 4,500/kg",
			weightGrams:   2500,
			pricePerKgIDR: 4500,
			expectedIDR:   11250,
		},
		{
			name:          "Rounding half up: 1.5g at Rp 1000/kg (1500 + 500) / 1000 = 2",
			weightGrams:   1,
			pricePerKgIDR: 1500,
			expectedIDR:   2,
		},
		{
			name:          "Rounding down: 1.4g at Rp 1000/kg (1400 + 500) / 1000 = 1",
			weightGrams:   1,
			pricePerKgIDR: 1400,
			expectedIDR:   1,
		},
		{
			name:          "Zero weight returns 0",
			weightGrams:   0,
			pricePerKgIDR: 5000,
			expectedIDR:   0,
		},
		{
			name:          "Zero price returns 0",
			weightGrams:   5000,
			pricePerKgIDR: 0,
			expectedIDR:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := CalculateSubtotal(tt.weightGrams, tt.pricePerKgIDR)
			assert.Equal(t, tt.expectedIDR, actual)
		})
	}
}
