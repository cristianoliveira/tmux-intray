package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter Filter
		want   bool
	}{
		{"empty filter", Filter{}, true},
		{"filter with level", Filter{Level: LevelInfo}, false},
		{"filter with state", Filter{State: StateActive}, false},
		{"filter with session", Filter{Session: "$0"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.filter.IsEmpty())
		})
	}
}

func TestFilterOptions_ToFilter(t *testing.T) {
	t.Run("valid filter options", func(t *testing.T) {
		opts := FilterOptions{
			Level:      "info",
			State:      "active",
			Session:    "$0",
			Window:     "@0",
			Pane:       "%0",
			OlderThan:  1,
			NewerThan:  7,
			ReadFilter: ReadFilterRead,
		}

		filter, err := opts.ToFilter()
		assert.NoError(t, err)
		assert.Equal(t, LevelInfo, filter.Level)
		assert.Equal(t, StateActive, filter.State)
		assert.Equal(t, "$0", filter.Session)
		assert.Equal(t, "@0", filter.Window)
		assert.Equal(t, "%0", filter.Pane)
		assert.NotEmpty(t, filter.OlderThan)
		assert.NotEmpty(t, filter.NewerThan)
		assert.Equal(t, ReadFilterRead, filter.ReadFilter)
	})

	t.Run("invalid level", func(t *testing.T) {
		opts := FilterOptions{
			Level: "invalid",
		}
		_, err := opts.ToFilter()
		assert.Error(t, err)
	})

	t.Run("invalid state", func(t *testing.T) {
		opts := FilterOptions{
			State: "invalid",
		}
		_, err := opts.ToFilter()
		assert.Error(t, err)
	})

	t.Run("invalid read filter", func(t *testing.T) {
		opts := FilterOptions{
			ReadFilter: "invalid",
		}
		_, err := opts.ToFilter()
		assert.Error(t, err)
	})
}

func TestFilterNotifications(t *testing.T) {
	notifications := []Notification{
		{
			ID:            1,
			Timestamp:     "2024-01-01T12:00:00Z",
			State:         StateActive,
			Level:         LevelInfo,
			Session:       "$1",
			Window:        "@1",
			Pane:          "%1",
			Message:       "test message 1",
			ReadTimestamp: "",
		},
		{
			ID:            2,
			Timestamp:     "2024-01-02T12:00:00Z",
			State:         StateDismissed,
			Level:         LevelWarning,
			Session:       "$1",
			Window:        "@2",
			Pane:          "%2",
			Message:       "test message 2",
			ReadTimestamp: "2024-01-02T13:00:00Z",
		},
		{
			ID:            3,
			Timestamp:     "2024-01-03T12:00:00Z",
			State:         StateActive,
			Level:         LevelError,
			Session:       "$2",
			Window:        "@1",
			Pane:          "%3",
			Message:       "test message 3",
			ReadTimestamp: "",
		},
	}

	t.Run("filter by level", func(t *testing.T) {
		filter := Filter{Level: LevelInfo}
		result := FilterNotifications(notifications, filter)
		assert.Len(t, result, 1)
		assert.Equal(t, 1, result[0].ID)
	})

	t.Run("filter by state", func(t *testing.T) {
		filter := Filter{State: StateActive}
		result := FilterNotifications(notifications, filter)
		assert.Len(t, result, 2)
	})

	t.Run("filter by session", func(t *testing.T) {
		filter := Filter{Session: "$1"}
		result := FilterNotifications(notifications, filter)
		assert.Len(t, result, 2)
	})

	t.Run("filter by read status", func(t *testing.T) {
		filter := Filter{ReadFilter: ReadFilterUnread}
		result := FilterNotifications(notifications, filter)
		assert.Len(t, result, 2)
	})

	t.Run("empty filter returns all", func(t *testing.T) {
		filter := Filter{}
		result := FilterNotifications(notifications, filter)
		assert.Len(t, result, 3)
	})
}

func TestFilterByLevel(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Level: LevelInfo},
		{ID: 2, Level: LevelWarning},
		{ID: 3, Level: LevelInfo},
	}

	result := FilterByLevel(notifications, "info")
	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].ID)
	assert.Equal(t, 3, result[1].ID)
}

func TestFilterByState(t *testing.T) {
	notifications := []Notification{
		{ID: 1, State: StateActive},
		{ID: 2, State: StateDismissed},
		{ID: 3, State: StateActive},
	}

	result := FilterByState(notifications, "active")
	assert.Len(t, result, 2)
}

func TestFilterBySession(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Session: "$1"},
		{ID: 2, Session: "$2"},
		{ID: 3, Session: "$1"},
	}

	result := FilterBySession(notifications, "$1")
	assert.Len(t, result, 2)
}

func TestFilterByWindow(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Window: "@1"},
		{ID: 2, Window: "@2"},
		{ID: 3, Window: "@1"},
	}

	result := FilterByWindow(notifications, "@1")
	assert.Len(t, result, 2)
}

func TestFilterByPane(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Pane: "%1"},
		{ID: 2, Pane: "%2"},
		{ID: 3, Pane: "%1"},
	}

	result := FilterByPane(notifications, "%1")
	assert.Len(t, result, 2)
}

func TestFilterByReadStatus(t *testing.T) {
	notifications := []Notification{
		{ID: 1, ReadTimestamp: "2024-01-01T12:00:00Z"},
		{ID: 2, ReadTimestamp: ""},
		{ID: 3, ReadTimestamp: "2024-01-01T12:00:00Z"},
	}

	t.Run("read", func(t *testing.T) {
		result := FilterByReadStatus(notifications, ReadFilterRead)
		assert.Len(t, result, 2)
	})

	t.Run("unread", func(t *testing.T) {
		result := FilterByReadStatus(notifications, ReadFilterUnread)
		assert.Len(t, result, 1)
	})
}

func TestSearchNotifications(t *testing.T) {
	notifications := []Notification{
		{
			ID:      1,
			Message: "test error in session",
			Session: "$1",
			Window:  "@1",
			Pane:    "%1",
			Level:   LevelError,
		},
		{
			ID:      2,
			Message: "warning message",
			Session: "$2",
			Window:  "@2",
			Pane:    "%2",
			Level:   LevelWarning,
		},
	}

	t.Run("case sensitive match", func(t *testing.T) {
		result := SearchNotifications(notifications, "error", false)
		assert.Len(t, result, 1)
		assert.Equal(t, 1, result[0].ID)
	})

	t.Run("case insensitive match", func(t *testing.T) {
		result := SearchNotifications(notifications, "ERROR", true)
		assert.Len(t, result, 1)
		assert.Equal(t, 1, result[0].ID)
	})

	t.Run("search session", func(t *testing.T) {
		result := SearchNotifications(notifications, "$1", false)
		assert.Len(t, result, 1)
		assert.Equal(t, 1, result[0].ID)
	})

	t.Run("search level", func(t *testing.T) {
		result := SearchNotifications(notifications, "warning", false)
		assert.Len(t, result, 1)
		assert.Equal(t, 2, result[0].ID)
	})

	t.Run("empty query returns all", func(t *testing.T) {
		result := SearchNotifications(notifications, "", false)
		assert.Len(t, result, 2)
	})

	t.Run("no match", func(t *testing.T) {
		result := SearchNotifications(notifications, "critical", false)
		assert.Len(t, result, 0)
	})
}

func TestNotificationMatchesFilter(t *testing.T) {
	notif := Notification{
		ID:            1,
		Timestamp:     "2024-01-01T12:00:00Z",
		State:         StateActive,
		Level:         LevelInfo,
		Session:       "$1",
		Window:        "@1",
		Pane:          "%1",
		ReadTimestamp: "",
	}

	t.Run("matches all criteria", func(t *testing.T) {
		filter := Filter{
			Level:   LevelInfo,
			State:   StateActive,
			Session: "$1",
			Window:  "@1",
			Pane:    "%1",
		}
		assert.True(t, notif.MatchesFilter(filter))
	})

	t.Run("does not match level", func(t *testing.T) {
		filter := Filter{Level: LevelWarning}
		assert.False(t, notif.MatchesFilter(filter))
	})

	t.Run("does not match state", func(t *testing.T) {
		filter := Filter{State: StateDismissed}
		assert.False(t, notif.MatchesFilter(filter))
	})

	t.Run("does not match read filter", func(t *testing.T) {
		filter := Filter{ReadFilter: ReadFilterRead}
		assert.False(t, notif.MatchesFilter(filter))
	})

	t.Run("matches read filter", func(t *testing.T) {
		filter := Filter{ReadFilter: ReadFilterUnread}
		assert.True(t, notif.MatchesFilter(filter))
	})
}
