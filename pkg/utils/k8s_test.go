package utils

import "testing"

func TestDetermineDistribution(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected Distribution
	}{
		"GKE": {
			input:    "v1.20.10-gke.301",
			expected: DistributionGKE,
		},
		"Unknown": {
			input:    "v1.22.8",
			expected: DistributionUnknown,
		},
	}

	for name, test := range tests {
		tt := test

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := DetermineDistribution(tt.input)

			if tt.expected != actual {
				t.Errorf("distribution doesn't match: %d != %d", tt.expected, actual)
			}
		})
	}
}
