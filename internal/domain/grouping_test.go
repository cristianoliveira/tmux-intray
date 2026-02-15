package domain

import (
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/dedup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupByMode_IsValid(t *testing.T) {
	tests := []struct {
		name string
		mode GroupByMode
		want bool
	}{
		{"valid none", GroupByNone, true},
		{"valid session", GroupBySession, true},
		{"valid window", GroupByWindow, true},
		{"valid pane", GroupByPane, true},
		{"valid level", GroupByLevel, true},
		{"valid message", GroupByMessage, true},
		{"invalid", GroupByMode("invalid"), false},
		{"invalid empty", GroupByMode(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.mode.IsValid())
		})
	}
}

func TestGroupByMode_String(t *testing.T) {
	tests := []struct {
		name string
		mode GroupByMode
		want string
	}{
		{"none", GroupByNone, "none"},
		{"session", GroupBySession, "session"},
		{"window", GroupByWindow, "window"},
		{"pane", GroupByPane, "pane"},
		{"level", GroupByLevel, "level"},
		{"message", GroupByMessage, "message"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.mode.String())
		})
	}
}

func TestGroupNotifications(t *testing.T) {
	notifications := []Notification{
		{
			ID:            1,
			Session:       "$1",
			Window:        "@1",
			Pane:          "%1",
			Level:         LevelInfo,
			Message:       "test message 1",
			ReadTimestamp: "",
		},
		{
			ID:            2,
			Session:       "$1",
			Window:        "@1",
			Pane:          "%1",
			Level:         LevelWarning,
			Message:       "test message 2",
			ReadTimestamp: "2024-01-01T12:00:00Z",
		},
		{
			ID:            3,
			Session:       "$2",
			Window:        "@1",
			Pane:          "%2",
			Level:         LevelInfo,
			Message:       "test message 3",
			ReadTimestamp: "",
		},
	}

	t.Run("group by session", func(t *testing.T) {
		result := GroupNotifications(notifications, GroupBySession)
		assert.Equal(t, GroupBySession, result.Mode)
		assert.Len(t, result.Groups, 2)
		assert.Equal(t, 3, result.TotalCount)
		assert.Equal(t, 2, result.TotalUnread)

		// Check groups are sorted alphabetically
		assert.Equal(t, "$1", result.Groups[0].DisplayName)
		assert.Equal(t, 2, result.Groups[0].Count)
		assert.Equal(t, 1, result.Groups[0].UnreadCount)
		assert.Equal(t, "$2", result.Groups[1].DisplayName)
		assert.Equal(t, 1, result.Groups[1].Count)
		assert.Equal(t, 1, result.Groups[1].UnreadCount)
	})

	t.Run("group by window", func(t *testing.T) {
		result := GroupNotifications(notifications, GroupByWindow)
		assert.Equal(t, GroupByWindow, result.Mode)
		assert.Len(t, result.Groups, 2)
	})

	t.Run("group by pane", func(t *testing.T) {
		result := GroupNotifications(notifications, GroupByPane)
		assert.Equal(t, GroupByPane, result.Mode)
		assert.Len(t, result.Groups, 2)
	})

	t.Run("group by level", func(t *testing.T) {
		result := GroupNotifications(notifications, GroupByLevel)
		assert.Equal(t, GroupByLevel, result.Mode)
		assert.Len(t, result.Groups, 2)
		assert.Equal(t, "info", result.Groups[0].DisplayName)
		assert.Equal(t, 2, result.Groups[0].Count)
		assert.Equal(t, "warning", result.Groups[1].DisplayName)
		assert.Equal(t, 1, result.Groups[1].Count)
	})

	t.Run("group by message", func(t *testing.T) {
		result := GroupNotifications(notifications, GroupByMessage)
		assert.Equal(t, GroupByMessage, result.Mode)
		assert.Len(t, result.Groups, 3)
		assert.Equal(t, "test message 1", result.Groups[0].DisplayName)
		assert.Equal(t, 1, result.Groups[0].Count)
		assert.Equal(t, "test message 2", result.Groups[1].DisplayName)
		assert.Equal(t, 1, result.Groups[1].Count)
		assert.Equal(t, "test message 3", result.Groups[2].DisplayName)
		assert.Equal(t, 1, result.Groups[2].Count)
	})

	t.Run("group by none", func(t *testing.T) {
		result := GroupNotifications(notifications, GroupByNone)
		assert.Equal(t, GroupByNone, result.Mode)
		assert.Len(t, result.Groups, 0)
		assert.Equal(t, 3, result.TotalCount)
	})

	t.Run("invalid mode falls back to none", func(t *testing.T) {
		result := GroupNotifications(notifications, GroupByMode("invalid"))
		assert.Equal(t, GroupByNone, result.Mode)
	})

	t.Run("empty notifications", func(t *testing.T) {
		result := GroupNotifications([]Notification{}, GroupBySession)
		assert.Equal(t, GroupBySession, result.Mode)
		assert.Len(t, result.Groups, 0)
		assert.Equal(t, 0, result.TotalCount)
	})
}

func TestGetNotificationsBySession(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Session: "$1"},
		{ID: 2, Session: "$2"},
		{ID: 3, Session: "$1"},
	}

	groups := GetNotificationsBySession(notifications)
	assert.Len(t, groups, 2)
	assert.Equal(t, 2, groups[0].Count)
	assert.Equal(t, "$1", groups[0].DisplayName)
	assert.Equal(t, 1, groups[1].Count)
	assert.Equal(t, "$2", groups[1].DisplayName)
}

func TestGetNotificationsByWindow(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Window: "@1"},
		{ID: 2, Window: "@2"},
		{ID: 3, Window: "@1"},
	}

	groups := GetNotificationsByWindow(notifications)
	assert.Len(t, groups, 2)
	assert.Equal(t, 2, groups[0].Count)
	assert.Equal(t, "@1", groups[0].DisplayName)
}

func TestGetNotificationsByPane(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Pane: "%1"},
		{ID: 2, Pane: "%2"},
		{ID: 3, Pane: "%1"},
	}

	groups := GetNotificationsByPane(notifications)
	assert.Len(t, groups, 2)
	assert.Equal(t, 2, groups[0].Count)
	assert.Equal(t, "%1", groups[0].DisplayName)
}

func TestGetNotificationsByLevel(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Level: LevelInfo},
		{ID: 2, Level: LevelWarning},
		{ID: 3, Level: LevelInfo},
	}

	groups := GetNotificationsByLevel(notifications)
	assert.Len(t, groups, 2)
	assert.Equal(t, 2, groups[0].Count)
	assert.Equal(t, "info", groups[0].DisplayName)
}

func TestGetNotificationsByMessage(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Message: "error: file not found"},
		{ID: 2, Message: "warning: low disk space"},
		{ID: 3, Message: "error: file not found"},
	}

	groups := GetNotificationsByMessage(notifications)
	assert.Len(t, groups, 2)
	assert.Equal(t, 2, groups[0].Count)
	assert.Equal(t, "error: file not found", groups[0].DisplayName)
	assert.Equal(t, 1, groups[1].Count)
	assert.Equal(t, "warning: low disk space", groups[1].DisplayName)
}

func TestGetGroupCounts(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Session: "$1"},
		{ID: 2, Session: "$2"},
		{ID: 3, Session: "$1"},
	}

	counts := GetGroupCounts(notifications, GroupBySession)
	assert.Len(t, counts, 2)
	assert.Equal(t, 2, counts["$1"])
	assert.Equal(t, 1, counts["$2"])
}

func TestGroupNotificationsWithDedupCriteria(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Message: "disk full", Level: LevelError, Timestamp: "2024-01-01T10:00:00Z"},
		{ID: 2, Message: "disk full", Level: LevelWarning, Timestamp: "2024-01-01T10:05:00Z"},
	}

	result := GroupNotificationsWithDedup(notifications, GroupByMessage, dedup.Options{Criteria: dedup.CriteriaMessageLevel})
	require.Len(t, result.Groups, 2)
	assert.Equal(t, "disk full", result.Groups[0].DisplayName)
	assert.Equal(t, "disk full", result.Groups[1].DisplayName)

	result = GroupNotificationsWithDedup(notifications, GroupByMessage, dedup.Options{Criteria: dedup.CriteriaMessage})
	require.Len(t, result.Groups, 1)
	assert.Equal(t, 2, result.Groups[0].Count)
}

func TestGroupNotificationsWithDedupWindow(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Message: "disk full", Level: LevelError, Timestamp: "2024-01-01T10:00:00Z"},
		{ID: 2, Message: "disk full", Level: LevelError, Timestamp: "2024-01-01T09:50:00Z"},
		{ID: 3, Message: "disk full", Level: LevelError, Timestamp: "2024-01-01T09:20:00Z"},
	}

	result := GroupNotificationsWithDedup(notifications, GroupByMessage, dedup.Options{Criteria: dedup.CriteriaMessage, Window: 15 * time.Minute})
	require.Len(t, result.Groups, 2)
	// Instead of assuming order, check for the presence of groups with expected counts.
	var counts []int
	for _, group := range result.Groups {
		counts = append(counts, group.Count)
	}
	assert.Contains(t, counts, 2, "Expected a group with count 2")
	assert.Contains(t, counts, 1, "Expected a group with count 1")
}

func TestExtractDisplayName(t *testing.T) {
	tests := []struct {
		name string
		key  string
		mode GroupByMode
		want string
	}{
		{"session key", "$1", GroupBySession, "$1"},
		{"window key", "$1\x00@1", GroupByWindow, "@1"},
		{"pane key", "$1\x00@1\x00%1", GroupByPane, "%1"},
		{"level key", "info", GroupByLevel, "info"},
		{"message key", "test message", GroupByMessage, "test message"},
		{"empty key", "", GroupBySession, "(empty)"},
		{"empty key message", "", GroupByMessage, "(empty)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractDisplayName(tt.key, tt.mode))
		})
	}
}

func TestSplitWithNull(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"single", []string{"single"}},
		{"two\000parts", []string{"two", "parts"}},
		{"three\000null\000parts", []string{"three", "null", "parts"}},
		{"trailing\000", []string{"trailing"}},
		{"\000leading", []string{"", "leading"}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, splitWithNull(tt.input))
		})
	}
}

func TestCountUnread(t *testing.T) {
	notifications := []Notification{
		{ID: 1, ReadTimestamp: ""},
		{ID: 2, ReadTimestamp: "2024-01-01T12:00:00Z"},
		{ID: 3, ReadTimestamp: ""},
	}

	count := countUnread(notifications)
	assert.Equal(t, 2, count)
}
