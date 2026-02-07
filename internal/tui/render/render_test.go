package render

import (
	"strings"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/stretchr/testify/assert"
)

func TestLevelIcon(t *testing.T) {
	tests := []struct {
		level    string
		expected string
	}{
		{"error", "❌ err"},
		{"warning", "⚠️ wrn"},
		{"critical", "‼️ crt"},
		{"info", "ℹ️ inf"},
		{"", "ℹ️ inf"},
		{"notice", "ℹ️ not"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := levelIcon(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{"active", "●"},
		{"", "●"},
		{"dismissed", "○"},
		{"paused", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := statusIcon(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateAge(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 30, 0, time.UTC)

	assert.Equal(t, "30s", calculateAge("2024-01-01T12:00:00Z", now))
	assert.Equal(t, "", calculateAge("", now))
	assert.Equal(t, "", calculateAge("invalid", now))
}

func TestRowSessionAndPaneColumns(t *testing.T) {
	row := Row(RowState{
		Notification: notification.Notification{
			ID:        1,
			Session:   "$1",
			Window:    "@2",
			Pane:      "%3",
			Message:   "Test message",
			Timestamp: "2024-01-01T12:00:00Z",
			Level:     "info",
			State:     "active",
		},
		SessionName: "main-session",
		Width:       100,
		Selected:    false,
		Now:         time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})

	assert.True(t, strings.Contains(row, "main-session"))
	assert.True(t, strings.Contains(row, "%3"))
	assert.False(t, strings.Contains(row, "@2:%3"))
}
