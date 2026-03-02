package format

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTmuxIntrayNotificationPrinter_PrintNotification(t *testing.T) {
	tests := []struct {
		name             string
		notification     TmuxIntrayNotification
		showIndex        bool
		index            int
		expectedContains []string
	}{
		{
			name: "with index",
			notification: TmuxIntrayNotification{
				Level:   "error",
				Message: "Test message",
				Session: "session1",
				Window:  "window1",
				Pane:    "pane1",
			},
			showIndex:        true,
			index:            1,
			expectedContains: []string{"1:", "[error]", "Test message", "session1:window1.pane1"},
		},
		{
			name: "without index",
			notification: TmuxIntrayNotification{
				Level:   "info",
				Message: "Another message",
				Session: "session2",
				Window:  "window2",
				Pane:    "pane2",
			},
			showIndex:        false,
			index:            0,
			expectedContains: []string{"[info]", "Another message", "session2:window2.pane2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output by redirecting colors output
			// Note: colors.Info prints to stdout directly, so we can't easily test the exact output
			// This test mainly verifies the function doesn't panic
			printer := NewTmuxIntrayNotificationPrinter(nil)
			printer.PrintNotification(tt.notification, tt.showIndex, tt.index)
		})
	}
}

func TestTmuxIntrayNotificationPrinter_PrintNotifications(t *testing.T) {
	tests := []struct {
		name          string
		notifications []TmuxIntrayNotification
		showIndex     bool
	}{
		{
			name: "multiple notifications with index",
			notifications: []TmuxIntrayNotification{
				{
					Level:   "error",
					Message: "First message",
					Session: "s1",
					Window:  "w1",
					Pane:    "p1",
				},
				{
					Level:   "info",
					Message: "Second message",
					Session: "s2",
					Window:  "w2",
					Pane:    "p2",
				},
			},
			showIndex: true,
		},
		{
			name: "multiple notifications without index",
			notifications: []TmuxIntrayNotification{
				{
					Level:   "warning",
					Message: "Warning message",
					Session: "s3",
					Window:  "w3",
					Pane:    "p3",
				},
			},
			showIndex: false,
		},
		{
			name:          "empty notifications",
			notifications: []TmuxIntrayNotification{},
			showIndex:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test mainly verifies the function doesn't panic
			printer := NewTmuxIntrayNotificationPrinter(nil)
			printer.PrintNotifications(tt.notifications, tt.showIndex)
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	tests := []struct {
		name         string
		field        string
		value        string
		validOptions string
		expected     string
	}{
		{
			name:         "index validation",
			field:        "index",
			value:        "abc",
			validOptions: "must be a number",
			expected:     "invalid index: abc (valid: must be a number)",
		},
		{
			name:         "level validation",
			field:        "level",
			value:        "debug",
			validOptions: "info, warning, error, critical",
			expected:     "invalid level: debug (valid: info, warning, error, critical)",
		},
		{
			name:         "state validation",
			field:        "state",
			value:        "deleted",
			validOptions: "active, dismissed",
			expected:     "invalid state: deleted (valid: active, dismissed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValidationError(tt.field, tt.value, tt.validOptions)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPrintError(t *testing.T) {
	// This test verifies the function doesn't panic
	PrintError("test error")
}

func TestPrintInfo(t *testing.T) {
	// This test verifies the function doesn't panic
	PrintInfo("test info")
}

func TestPrintDebug(t *testing.T) {
	// This test verifies the function doesn't panic
	PrintDebug("test debug")
}

func TestNewTmuxIntrayNotificationPrinter(t *testing.T) {
	var buf bytes.Buffer
	printer := NewTmuxIntrayNotificationPrinter(&buf)
	require.NotNil(t, printer)
}
