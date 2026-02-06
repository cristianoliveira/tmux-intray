package version

import (
	"testing"
)

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		commit   string
		expected string
	}{
		{
			name:     "development version without commit",
			version:  "development",
			commit:   "unknown",
			expected: "development",
		},
		{
			name:     "release version with commit",
			version:  "1.0.0",
			commit:   "abc1234",
			expected: "1.0.0+abc1234",
		},
		{
			name:     "version with commit hash",
			version:  "0.5.0",
			commit:   "def5678",
			expected: "0.5.0+def5678",
		},
		{
			name:     "unknown commit shows only version",
			version:  "2.0.0",
			commit:   "unknown",
			expected: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origVersion := Version
			origCommit := Commit
			defer func() {
				Version = origVersion
				Commit = origCommit
			}()

			// Set test values
			Version = tt.version
			Commit = tt.commit

			// Test String()
			if got := String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}
