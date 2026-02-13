package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewVersionCmdPanicsWhenClientIsNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got nil")
		}

		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected panic message as string, got %T", r)
		}
		if !strings.Contains(msg, "client dependency cannot be nil") {
			t.Fatalf("expected panic message to mention nil dependency, got %q", msg)
		}
	}()

	NewVersionCmd(nil)
}

func TestVersionOutput(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "development version",
			version:  "development",
			expected: "tmux-intray version development\n",
		},
		{
			name:     "release version",
			version:  "1.0.0",
			expected: "tmux-intray version 1.0.0\n",
		},
		{
			name:     "version with commit",
			version:  "0.5.0+abc1234",
			expected: "tmux-intray version 0.5.0+abc1234\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeVersionClient{version: tt.version}
			cmd := NewVersionCmd(client)
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			err := cmd.RunE(cmd, []string{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buf.String() != tt.expected {
				t.Errorf("version output %q, want %q", buf.String(), tt.expected)
			}
		})
	}
}

func TestNewVersionCmdOutput(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		expectedVersion string
	}{
		{
			name:            "development version",
			version:         "development",
			expectedVersion: "tmux-intray version development\n",
		},
		{
			name:            "release version",
			version:         "1.0.0",
			expectedVersion: "tmux-intray version 1.0.0\n",
		},
		{
			name:            "version with commit",
			version:         "0.5.0+abc1234",
			expectedVersion: "tmux-intray version 0.5.0+abc1234\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeVersionClient{version: tt.version}
			cmd := NewVersionCmd(client)

			var buf bytes.Buffer
			cmd.SetOut(&buf)

			err := cmd.RunE(cmd, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if buf.String() != tt.expectedVersion {
				t.Errorf("expected %q, got %q", tt.expectedVersion, buf.String())
			}
		})
	}
}

type fakeVersionClient struct {
	version string
}

func (f *fakeVersionClient) Version() string {
	return f.version
}
