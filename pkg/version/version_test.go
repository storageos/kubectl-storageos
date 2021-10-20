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
		"already cleaned versionless": {
			input:    "21.42.43",
			expected: "21.42.43",
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

func TestIsSupported(t *testing.T) {
	tests := map[string]struct {
		haveVersion string
		wantVersion string
		expected    bool
	}{
		"less than": {
			haveVersion: "1.17.3",
			wantVersion: "1.18.0",
		},
		"greater than": {
			haveVersion: "1.18.3",
			wantVersion: "1.18.0",
			expected:    true,
		},
		"greater than prefixed": {
			haveVersion: "v1.18.3",
			wantVersion: "v1.18.0",
			expected:    true,
		},
		"greater than postfixed": {
			haveVersion: "1.18.3-gke.301",
			wantVersion: "1.18.0#tag",
			expected:    true,
		},
	}

	for name, test := range tests {
		tt := test

		t.Run(name, func(t *testing.T) {
			// t.Parallel()

			actual, err := IsSupported(tt.haveVersion, tt.wantVersion)

			if err != nil {
				t.Errorf("error not allowed: %s", err.Error())
			}

			if tt.expected != actual {
				t.Errorf("supported value doesn't match: %t != %t", tt.expected, actual)
			}
		})
	}
}
