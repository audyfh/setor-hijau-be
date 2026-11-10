package carbon_test

import (
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/carbon"
	"github.com/stretchr/testify/assert"
)

// TestCalculateAvoidedCO2e verifies the hand-calculated cases documented in docs/CARBON-METHOD.md.
func TestCalculateAvoidedCO2e(t *testing.T) {
	// Factor PPM values from US EPA WARM v16 seeded in database
	const (
		ppmCorrugated = 3428188  // Corrugated Containers (Kardus)
		ppmAluminium  = 10042057 // Aluminum Cans
		ppmPET        = 1267658  // PET Bottles
		ppmHDPE       = 936965   // HDPE Containers
		ppmGlass      = 308647   // Glass Containers
		ppmSteel      = 1995183  // Steel Cans
		ppmPaper      = 3891159  // Mixed Paper
	)

	testCases := []struct {
		name          string
		weightGrams   int64
		factorPPM     int64
		expectedCO2eG int64
	}{
		{
			name:          "TC-01: 1500g Corrugated Containers",
			weightGrams:   1500,
			factorPPM:     ppmCorrugated,
			expectedCO2eG: 5142,
		},
		{
			name:          "TC-02: 2000g Corrugated Containers",
			weightGrams:   2000,
			factorPPM:     ppmCorrugated,
			expectedCO2eG: 6856,
		},
		{
			name:          "TC-03: 500g Aluminum Cans",
			weightGrams:   500,
			factorPPM:     ppmAluminium,
			expectedCO2eG: 5021,
		},
		{
			name:          "TC-04: 1000g PET Plastic Bottles",
			weightGrams:   1000,
			factorPPM:     ppmPET,
			expectedCO2eG: 1268,
		},
		{
			name:          "TC-05: 2500g HDPE Plastic",
			weightGrams:   2500,
			factorPPM:     ppmHDPE,
			expectedCO2eG: 2342,
		},
		{
			name:          "TC-06: 5000g Glass Bottles",
			weightGrams:   5000,
			factorPPM:     ppmGlass,
			expectedCO2eG: 1543,
		},
		{
			name:          "TC-07: 1000g Steel Cans",
			weightGrams:   1000,
			factorPPM:     ppmSteel,
			expectedCO2eG: 1995,
		},
		{
			name:          "TC-08: 3000g Mixed Paper",
			weightGrams:   3000,
			factorPPM:     ppmPaper,
			expectedCO2eG: 11673,
		},
		{
			name:          "TC-09: Lower Bound 1g PET",
			weightGrams:   1,
			factorPPM:     ppmPET,
			expectedCO2eG: 1, // (1 * 1267658 + 500000) / 1000000 = 1
		},
		{
			name:          "TC-10: Half-Up Rounding Midpoint 500g Glass",
			weightGrams:   500,
			factorPPM:     ppmGlass,
			expectedCO2eG: 154, // (500 * 308647 + 500000) / 1000000 = 154.823.500 / 1000000 = 154
		},
		{
			name:          "Zero Weight produces Zero CO2e",
			weightGrams:   0,
			factorPPM:     ppmCorrugated,
			expectedCO2eG: 0,
		},
		{
			name:          "Zero Factor produces Zero CO2e",
			weightGrams:   1000,
			factorPPM:     0,
			expectedCO2eG: 0,
		},
		{
			name:          "Negative Weight produces Zero CO2e",
			weightGrams:   -500,
			factorPPM:     ppmCorrugated,
			expectedCO2eG: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := carbon.CalculateAvoidedCO2e(tc.weightGrams, tc.factorPPM)
			assert.Equal(t, tc.expectedCO2eG, actual, "perhitungan avoided CO2e harus cocok dengan hand-calculated spec")
		})
	}
}
