package main

import (
	"bytes"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/version"
)

func TestPrintVersion(t *testing.T) {
	// Save original writer and version variables
	origWriter := versionOutputWriter
	origVersion := version.Version
	origCommit := version.Commit
	defer func() {
		versionOutputWriter = origWriter
		version.Version = origVersion
		version.Commit = origCommit
	}()

	tests := []struct {
		name     string
		ver      string
		commit   string
		expected string
	}{
		{
			name:     "development version without commit",
			ver:      "development",
			commit:   "unknown",
			expected: "tmux-intray version development\n",
		},
		{
			name:     "release version with commit",
			ver:      "1.0.0",
			commit:   "abc1234",
			expected: "tmux-intray version 1.0.0+abc1234\n",
		},
		{
			name:     "version with commit hash",
			ver:      "0.5.0",
			commit:   "def5678",
			expected: "tmux-intray version 0.5.0+def5678\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			versionOutputWriter = &buf
			version.Version = tt.ver
			version.Commit = tt.commit
			PrintVersion()
			if buf.String() != tt.expected {
				t.Errorf("PrintVersion() printed %q, want %q", buf.String(), tt.expected)
			}
		})
	}
}
