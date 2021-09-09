package version

import "testing"

func TestCleanupVersion(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"already cleaned": {
			input:    "v21.42.43",
			expected: "v21.42.43",
		},
		"has postfix": {
			input:    "v21.42.43-alpha1",
			expected: "v21.42.43",
		},
	}

	for name, test := range tests {
		tt := test

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := cleanupVersion(tt.input)

			if tt.expected != actual {
				t.Errorf("cleaned version doesn't match: %s != %s", tt.expected, actual)
			}
		})
	}
}
