package impact

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateDigitalEmissions(t *testing.T) {
	tests := []struct {
		name               string
		bytesTransferred   int64
		expectedMicrograms int64
		expectedGrams      int64
	}{
		{
			name:               "Zero bytes",
			bytesTransferred:   0,
			expectedMicrograms: 0,
			expectedGrams:      0,
		},
		{
			name:               "Negative bytes",
			bytesTransferred:   -500,
			expectedMicrograms: 0,
			expectedGrams:      0,
		},
		{
			name:               "Small payload: 1 KB (1,024 bytes)",
			bytesTransferred:   1024,
			expectedMicrograms: 36, // (1024 * 35568 + 500000) / 1000000 = 36
			expectedGrams:      0,
		},
		{
			name:               "Web page visit: 1 MB (1,000,000 bytes)",
			bytesTransferred:   1000000,
			expectedMicrograms: 35568, // Exactly 35,568 micrograms = 0.035568 g
			expectedGrams:      0,
		},
		{
			name:               "Heavy session: 100 MB (100,000,000 bytes)",
			bytesTransferred:   100000000,
			expectedMicrograms: 3556800,
			expectedGrams:      4, // (3556800 + 500000) / 1000000 = 4
		},
		{
			name:               "Standard 1 GB (1,000,000,000 bytes)",
			bytesTransferred:   1000000000,
			expectedMicrograms: 35568000,
			expectedGrams:      36, // 35.568 g rounds half-up to 36 g
		},
		{
			name:               "Large 10 GB (10,000,000,000 bytes)",
			bytesTransferred:   10000000000,
			expectedMicrograms: 355680000,
			expectedGrams:      356, // 355.68 g rounds half-up to 356 g
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			micro, grams := CalculateDigitalEmissions(tt.bytesTransferred)
			assert.Equal(t, tt.expectedMicrograms, micro)
			assert.Equal(t, tt.expectedGrams, grams)
		})
	}
}
