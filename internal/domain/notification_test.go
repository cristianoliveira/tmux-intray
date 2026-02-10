package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationState_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		state NotificationState
		want  bool
	}{
		{"valid active", StateActive, true},
		{"valid dismissed", StateDismissed, true},
		{"invalid empty", NotificationState(""), false},
		{"invalid other", NotificationState("other"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.IsValid())
		})
	}
}

func TestNotificationState_String(t *testing.T) {
	tests := []struct {
		name  string
		state NotificationState
		want  string
	}{
		{"active", StateActive, "active"},
		{"dismissed", StateDismissed, "dismissed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestNotificationLevel_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		level NotificationLevel
		want  bool
	}{
		{"valid info", LevelInfo, true},
		{"valid warning", LevelWarning, true},
		{"valid error", LevelError, true},
		{"valid critical", LevelCritical, true},
		{"invalid empty", NotificationLevel(""), false},
		{"invalid other", NotificationLevel("other"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.level.IsValid())
		})
	}
}

func TestNotificationLevel_String(t *testing.T) {
	tests := []struct {
		name  string
		level NotificationLevel
		want  string
	}{
		{"info", LevelInfo, "info"},
		{"warning", LevelWarning, "warning"},
		{"error", LevelError, "error"},
		{"critical", LevelCritical, "critical"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.level.String())
		})
	}
}

func TestNotification_IsRead(t *testing.T) {
	tests := []struct {
		name          string
		readTimestamp string
		wantIsRead    bool
	}{
		{"read with timestamp", "2024-01-01T12:00:00Z", true},
		{"unread empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Notification{
				ReadTimestamp: tt.readTimestamp,
			}
			assert.Equal(t, tt.wantIsRead, n.IsRead())
		})
	}
}

func TestNotification_MarkRead(t *testing.T) {
	n := &Notification{}
	assert.False(t, n.IsRead())

	result := n.MarkRead()
	assert.True(t, result.IsRead())
	assert.NotNil(t, result.ReadTimestamp)

	// Verify RFC3339 format
	_, err := time.Parse(time.RFC3339, result.ReadTimestamp)
	assert.NoError(t, err)
}

func TestNotification_MarkUnread(t *testing.T) {
	n := &Notification{
		ReadTimestamp: "2024-01-01T12:00:00Z",
	}
	assert.True(t, n.IsRead())

	result := n.MarkUnread()
	assert.False(t, result.IsRead())
	assert.Equal(t, "", result.ReadTimestamp)
}

func TestNotification_Dismiss(t *testing.T) {
	n := &Notification{
		State: StateActive,
	}
	assert.Equal(t, StateActive, n.State)

	result := n.Dismiss()
	assert.Equal(t, StateDismissed, result.State)
}

func TestNotification_Validate(t *testing.T) {
	t.Run("valid notification", func(t *testing.T) {
		n := &Notification{
			ID:            1,
			Timestamp:     "2024-01-01T12:00:00Z",
			State:         StateActive,
			Session:       "$0",
			Window:        "@0",
			Pane:          "%0",
			Message:       "test message",
			PaneCreated:   "2024-01-01T11:00:00Z",
			Level:         LevelInfo,
			ReadTimestamp: "2024-01-01T13:00:00Z",
		}
		assert.NoError(t, n.Validate())
	})

	t.Run("invalid ID", func(t *testing.T) {
		n := &Notification{
			ID:        0,
			Timestamp: "2024-01-01T12:00:00Z",
			State:     StateActive,
			Level:     LevelInfo,
			Message:   "test",
		}
		assert.Error(t, n.Validate())
		assert.Contains(t, n.Validate().Error(), "invalid notification ID")
	})

	t.Run("empty timestamp", func(t *testing.T) {
		n := &Notification{
			ID:      1,
			State:   StateActive,
			Level:   LevelInfo,
			Message: "test",
		}
		assert.Error(t, n.Validate())
		assert.Contains(t, n.Validate().Error(), "timestamp cannot be empty")
	})

	t.Run("invalid timestamp format", func(t *testing.T) {
		n := &Notification{
			ID:        1,
			Timestamp: "invalid",
			State:     StateActive,
			Level:     LevelInfo,
			Message:   "test",
		}
		assert.Error(t, n.Validate())
		assert.Contains(t, n.Validate().Error(), "invalid timestamp format")
	})

	t.Run("invalid state", func(t *testing.T) {
		n := &Notification{
			ID:        1,
			Timestamp: "2024-01-01T12:00:00Z",
			State:     NotificationState("invalid"),
			Level:     LevelInfo,
			Message:   "test",
		}
		assert.Error(t, n.Validate())
		assert.Contains(t, n.Validate().Error(), "invalid notification state")
	})

	t.Run("invalid level", func(t *testing.T) {
		n := &Notification{
			ID:        1,
			Timestamp: "2024-01-01T12:00:00Z",
			State:     StateActive,
			Level:     NotificationLevel("invalid"),
			Message:   "test",
		}
		assert.Error(t, n.Validate())
		assert.Contains(t, n.Validate().Error(), "invalid notification level")
	})

	t.Run("empty message", func(t *testing.T) {
		n := &Notification{
			ID:        1,
			Timestamp: "2024-01-01T12:00:00Z",
			State:     StateActive,
			Level:     LevelInfo,
		}
		assert.Error(t, n.Validate())
		assert.Contains(t, n.Validate().Error(), "message cannot be empty")
	})

	t.Run("invalid read timestamp format", func(t *testing.T) {
		n := &Notification{
			ID:            1,
			Timestamp:     "2024-01-01T12:00:00Z",
			State:         StateActive,
			Level:         LevelInfo,
			Message:       "test",
			ReadTimestamp: "invalid",
		}
		assert.Error(t, n.Validate())
		assert.Contains(t, n.Validate().Error(), "invalid read timestamp format")
	})
}

func TestNewNotification(t *testing.T) {
	t.Run("valid notification", func(t *testing.T) {
		notif, err := NewNotification(
			1,
			"2024-01-01T12:00:00Z",
			StateActive,
			"$0", "@0", "%0",
			"test message",
			"2024-01-01T11:00:00Z",
			LevelInfo,
			"",
		)
		require.NoError(t, err)
		assert.NotNil(t, notif)
		assert.Equal(t, 1, notif.ID)
		assert.Equal(t, "test message", notif.Message)
	})

	t.Run("invalid notification", func(t *testing.T) {
		_, err := NewNotification(
			0,
			"",
			StateActive,
			"", "", "",
			"",
			"",
			LevelInfo,
			"",
		)
		assert.Error(t, err)
	})
}

func TestParseNotificationLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected NotificationLevel
		wantErr  bool
	}{
		{"info", LevelInfo, false},
		{"warning", LevelWarning, false},
		{"error", LevelError, false},
		{"critical", LevelCritical, false},
		{"invalid", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := ParseNotificationLevel(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, level)
			}
		})
	}
}

func TestParseNotificationState(t *testing.T) {
	tests := []struct {
		input    string
		expected NotificationState
		wantErr  bool
	}{
		{"active", StateActive, false},
		{"dismissed", StateDismissed, false},
		{"invalid", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			state, err := ParseNotificationState(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, state)
			}
		})
	}
}
