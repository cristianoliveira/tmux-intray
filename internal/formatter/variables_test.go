package formatter

import (
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

func TestVariableResolver_ResolveCountVariables(t *testing.T) {
	resolver := NewVariableResolver()

	ctx := VariableContext{
		UnreadCount:    5,
		ReadCount:      3,
		TotalCount:     8,
		ActiveCount:    4,
		DismissedCount: 1,
	}

	tests := []struct {
		name    string
		varName string
		want    string
		wantErr bool
	}{
		{
			name:    "unread-count",
			varName: "unread-count",
			want:    "5",
		},
		{
			name:    "read-count",
			varName: "read-count",
			want:    "3",
		},
		{
			name:    "total-count (alias for unread-count)",
			varName: "total-count",
			want:    "5",
		},
		{
			name:    "active-count",
			varName: "active-count",
			want:    "4",
		},
		{
			name:    "dismissed-count",
			varName: "dismissed-count",
			want:    "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Resolve(tt.varName, ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve() got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVariableResolver_ResolveContentVariables(t *testing.T) {
	resolver := NewVariableResolver()

	ctx := VariableContext{
		LatestMessage: "Test message from notification",
		SessionList:   "work,personal",
		WindowList:    "editor,browser,terminal",
		PaneList:      "pane1,pane2,pane3",
	}

	tests := []struct {
		name    string
		varName string
		want    string
		wantErr bool
	}{
		{
			name:    "latest-message",
			varName: "latest-message",
			want:    "Test message from notification",
		},
		{
			name:    "session-list",
			varName: "session-list",
			want:    "work,personal",
		},
		{
			name:    "window-list",
			varName: "window-list",
			want:    "editor,browser,terminal",
		},
		{
			name:    "pane-list",
			varName: "pane-list",
			want:    "pane1,pane2,pane3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Resolve(tt.varName, ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve() got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVariableResolver_ResolveBooleanVariables(t *testing.T) {
	resolver := NewVariableResolver()

	tests := []struct {
		name    string
		ctx     VariableContext
		varName string
		want    string
	}{
		{
			name:    "has-unread true",
			varName: "has-unread",
			ctx: VariableContext{
				HasUnread: true,
			},
			want: "true",
		},
		{
			name:    "has-unread false",
			varName: "has-unread",
			ctx: VariableContext{
				HasUnread: false,
			},
			want: "false",
		},
		{
			name:    "has-active true",
			varName: "has-active",
			ctx: VariableContext{
				HasActive: true,
			},
			want: "true",
		},
		{
			name:    "has-active false",
			varName: "has-active",
			ctx: VariableContext{
				HasActive: false,
			},
			want: "false",
		},
		{
			name:    "has-dismissed true",
			varName: "has-dismissed",
			ctx: VariableContext{
				HasDismissed: true,
			},
			want: "true",
		},
		{
			name:    "has-dismissed false",
			varName: "has-dismissed",
			ctx: VariableContext{
				HasDismissed: false,
			},
			want: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Resolve(tt.varName, tt.ctx)
			if err != nil {
				t.Errorf("Resolve() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve() got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVariableResolver_ResolveSeverityVariable(t *testing.T) {
	resolver := NewVariableResolver()

	tests := []struct {
		name     string
		severity domain.NotificationLevel
		want     string
	}{
		{
			name:     "critical = 1",
			severity: domain.LevelCritical,
			want:     "1",
		},
		{
			name:     "error = 2",
			severity: domain.LevelError,
			want:     "2",
		},
		{
			name:     "warning = 3",
			severity: domain.LevelWarning,
			want:     "3",
		},
		{
			name:     "info = 4",
			severity: domain.LevelInfo,
			want:     "4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := VariableContext{
				HighestSeverity: tt.severity,
			}
			got, err := resolver.Resolve("highest-severity", ctx)
			if err != nil {
				t.Errorf("Resolve() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve() got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVariableResolver_ResolveUnknownVariable(t *testing.T) {
	resolver := NewVariableResolver()
	ctx := VariableContext{}

	_, err := resolver.Resolve("unknown-variable", ctx)
	if err == nil {
		t.Errorf("Resolve() expected error for unknown variable, got nil")
	}

	// Check that error message contains available variables
	errStr := err.Error()
	if !strings.Contains(errStr, "unknown-variable") {
		t.Errorf("Error message should contain unknown variable name, got: %s", errStr)
	}
	if !strings.Contains(errStr, "Available variables:") {
		t.Errorf("Error message should contain available variables list, got: %s", errStr)
	}
	if !strings.Contains(errStr, "unread-count") {
		t.Errorf("Error message should list available variables, got: %s", errStr)
	}
}

func TestVariableResolver_AllVariables(t *testing.T) {
	resolver := NewVariableResolver()

	ctx := VariableContext{
		UnreadCount:     5,
		ReadCount:       3,
		TotalCount:      8,
		ActiveCount:     4,
		DismissedCount:  1,
		InfoCount:       2,
		WarningCount:    1,
		ErrorCount:      1,
		CriticalCount:   0,
		LatestMessage:   "Test message",
		HasUnread:       true,
		HasActive:       true,
		HasDismissed:    false,
		HighestSeverity: domain.LevelError,
		SessionList:     "work",
		WindowList:      "editor",
		PaneList:        "pane1",
	}

	// All template variables
	variables := []string{
		"unread-count",
		"total-count",
		"read-count",
		"active-count",
		"dismissed-count",
		"info-count",
		"warning-count",
		"error-count",
		"critical-count",
		"latest-message",
		"has-unread",
		"has-active",
		"has-dismissed",
		"highest-severity",
		"session-list",
		"window-list",
		"pane-list",
	}

	for _, varName := range variables {
		t.Run(varName, func(t *testing.T) {
			value, err := resolver.Resolve(varName, ctx)
			if err != nil {
				t.Errorf("Resolve(%q) error = %v", varName, err)
				return
			}
			if value == "" {
				t.Logf("Warning: Resolve(%q) returned empty string", varName)
			}
		})
	}
}

func TestBoolToString(t *testing.T) {
	tests := []struct {
		input bool
		want  string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := boolToString(tt.input)
			if got != tt.want {
				t.Errorf("boolToString(%v) got %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSeverityToOrdinal(t *testing.T) {
	tests := []struct {
		level domain.NotificationLevel
		want  string
	}{
		{domain.LevelCritical, "1"},
		{domain.LevelError, "2"},
		{domain.LevelWarning, "3"},
		{domain.LevelInfo, "4"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			got := severityToOrdinal(tt.level)
			if got != tt.want {
				t.Errorf("severityToOrdinal(%s) got %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}
