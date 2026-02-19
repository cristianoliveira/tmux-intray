package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortByField_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		field SortByField
		want  bool
	}{
		{"valid id", SortByIDField, true},
		{"valid timestamp", SortByTimestampField, true},
		{"valid state", SortByStateField, true},
		{"valid level", SortByLevelField, true},
		{"valid session", SortBySessionField, true},
		{"valid message", SortByMessageField, true},
		{"valid read_status", SortByReadStatusField, true},
		{"invalid", SortByField("invalid"), false},
		{"invalid empty", SortByField(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.field.IsValid())
		})
	}
}

func TestSortByField_String(t *testing.T) {
	tests := []struct {
		name  string
		field SortByField
		want  string
	}{
		{"id", SortByIDField, "id"},
		{"timestamp", SortByTimestampField, "timestamp"},
		{"state", SortByStateField, "state"},
		{"level", SortByLevelField, "level"},
		{"session", SortBySessionField, "session"},
		{"message", SortByMessageField, "message"},
		{"read_status", SortByReadStatusField, "read_status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.field.String())
		})
	}
}

func TestSortOrder_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		order SortOrder
		want  bool
	}{
		{"valid asc", SortOrderAsc, true},
		{"valid desc", SortOrderDesc, true},
		{"invalid", SortOrder("invalid"), false},
		{"invalid empty", SortOrder(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.order.IsValid())
		})
	}
}

func TestSortOrder_String(t *testing.T) {
	tests := []struct {
		name  string
		order SortOrder
		want  string
	}{
		{"asc", SortOrderAsc, "asc"},
		{"desc", SortOrderDesc, "desc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.order.String())
		})
	}
}

func TestSortNotifications(t *testing.T) {
	notifications := []Notification{
		{ID: 3, Timestamp: "2024-01-03T12:00:00Z", State: StateDismissed, Level: LevelError, Session: "$3", Message: "gamma"},
		{ID: 1, Timestamp: "2024-01-01T12:00:00Z", State: StateActive, Level: LevelInfo, Session: "$1", Message: "Alpha"},
		{ID: 2, Timestamp: "2024-01-02T12:00:00Z", State: StateActive, Level: LevelWarning, Session: "$2", Message: "beta"},
	}

	t.Run("sort by ID ascending", func(t *testing.T) {
		opts := SortOptions{Field: SortByIDField, Order: SortOrderAsc}
		result := SortNotifications(notifications, opts)
		assert.Len(t, result, 3)
		assert.Equal(t, 1, result[0].ID)
		assert.Equal(t, 2, result[1].ID)
		assert.Equal(t, 3, result[2].ID)
	})

	t.Run("sort by ID descending", func(t *testing.T) {
		opts := SortOptions{Field: SortByIDField, Order: SortOrderDesc}
		result := SortNotifications(notifications, opts)
		assert.Len(t, result, 3)
		assert.Equal(t, 3, result[0].ID)
		assert.Equal(t, 2, result[1].ID)
		assert.Equal(t, 1, result[2].ID)
	})

	t.Run("sort by timestamp ascending", func(t *testing.T) {
		opts := SortOptions{Field: SortByTimestampField, Order: SortOrderAsc}
		result := SortNotifications(notifications, opts)
		assert.Equal(t, 1, result[0].ID)
		assert.Equal(t, 2, result[1].ID)
		assert.Equal(t, 3, result[2].ID)
	})

	t.Run("sort by timestamp descending", func(t *testing.T) {
		opts := SortOptions{Field: SortByTimestampField, Order: SortOrderDesc}
		result := SortNotifications(notifications, opts)
		assert.Equal(t, 3, result[0].ID)
		assert.Equal(t, 2, result[1].ID)
		assert.Equal(t, 1, result[2].ID)
	})

	t.Run("sort by state ascending", func(t *testing.T) {
		opts := SortOptions{Field: SortByStateField, Order: SortOrderAsc}
		result := SortNotifications(notifications, opts)
		assert.Equal(t, StateActive, result[0].State)
		assert.Equal(t, StateDismissed, result[2].State)
	})

	t.Run("sort by level ascending", func(t *testing.T) {
		opts := SortOptions{Field: SortByLevelField, Order: SortOrderAsc}
		result := SortNotifications(notifications, opts)
		assert.Equal(t, LevelError, result[0].Level)
		assert.Equal(t, LevelInfo, result[1].Level)
		assert.Equal(t, LevelWarning, result[2].Level)
	})

	t.Run("sort by session ascending", func(t *testing.T) {
		opts := SortOptions{Field: SortBySessionField, Order: SortOrderAsc}
		result := SortNotifications(notifications, opts)
		assert.Equal(t, "$1", result[0].Session)
		assert.Equal(t, "$2", result[1].Session)
		assert.Equal(t, "$3", result[2].Session)
	})

	t.Run("sort by message case sensitive", func(t *testing.T) {
		opts := SortOptions{Field: SortByMessageField, Order: SortOrderAsc, CaseInsensitive: false}
		result := SortNotifications(notifications, opts)
		assert.Equal(t, "Alpha", result[0].Message)
		assert.Equal(t, "beta", result[1].Message)
		assert.Equal(t, "gamma", result[2].Message)
	})

	t.Run("sort by message case insensitive", func(t *testing.T) {
		opts := SortOptions{Field: SortByMessageField, Order: SortOrderAsc, CaseInsensitive: true}
		result := SortNotifications(notifications, opts)
		assert.Equal(t, "Alpha", result[0].Message)
		assert.Equal(t, "beta", result[1].Message)
		assert.Equal(t, "gamma", result[2].Message)
	})

	t.Run("invalid field defaults to timestamp", func(t *testing.T) {
		opts := SortOptions{Field: SortByField("invalid"), Order: SortOrderDesc}
		result := SortNotifications(notifications, opts)
		assert.Equal(t, 3, result[0].ID) // Should sort by timestamp desc (ID 3 has latest timestamp)
	})

	t.Run("invalid order defaults to desc", func(t *testing.T) {
		opts := SortOptions{Field: SortByTimestampField, Order: SortOrder("invalid")}
		result := SortNotifications(notifications, opts)
		assert.Equal(t, 3, result[0].ID)
	})

	t.Run("empty notifications returns empty", func(t *testing.T) {
		opts := SortOptions{Field: SortByTimestampField, Order: SortOrderAsc}
		result := SortNotifications([]Notification{}, opts)
		assert.Len(t, result, 0)
	})

	t.Run("sort by read status ascending (unread first)", func(t *testing.T) {
		notifs := []Notification{
			{ID: 1, ReadTimestamp: "2024-01-01T12:00:00Z"}, // read
			{ID: 2, ReadTimestamp: ""},                     // unread
			{ID: 3, ReadTimestamp: "2024-01-02T12:00:00Z"}, // read
			{ID: 4, ReadTimestamp: ""},                     // unread
		}
		opts := SortOptions{Field: SortByReadStatusField, Order: SortOrderAsc}
		result := SortNotifications(notifs, opts)
		// unread first (ID 2, 4), then read (ID 1, 3)
		assert.Equal(t, 2, result[0].ID)
		assert.Equal(t, 4, result[1].ID)
		assert.Equal(t, 1, result[2].ID)
		assert.Equal(t, 3, result[3].ID)
	})

	t.Run("sort by read status descending (read first)", func(t *testing.T) {
		notifs := []Notification{
			{ID: 1, ReadTimestamp: "2024-01-01T12:00:00Z"},
			{ID: 2, ReadTimestamp: ""},
			{ID: 3, ReadTimestamp: "2024-01-02T12:00:00Z"},
			{ID: 4, ReadTimestamp: ""},
		}
		opts := SortOptions{Field: SortByReadStatusField, Order: SortOrderDesc}
		result := SortNotifications(notifs, opts)
		// read first (ID 1, 3), then unread (ID 2, 4)
		assert.Equal(t, 1, result[0].ID)
		assert.Equal(t, 3, result[1].ID)
		assert.Equal(t, 2, result[2].ID)
		assert.Equal(t, 4, result[3].ID)
	})
}

func TestSortByID(t *testing.T) {
	notifications := []Notification{
		{ID: 3},
		{ID: 1},
		{ID: 2},
	}

	result := SortByID(notifications, SortOrderAsc)
	assert.Equal(t, 1, result[0].ID)
	assert.Equal(t, 2, result[1].ID)
	assert.Equal(t, 3, result[2].ID)
}

func TestSortByTimestamp(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Timestamp: "2024-01-03T12:00:00Z"},
		{ID: 2, Timestamp: "2024-01-01T12:00:00Z"},
		{ID: 3, Timestamp: "2024-01-02T12:00:00Z"},
	}

	result := SortByTimestamp(notifications, SortOrderAsc)
	assert.Equal(t, 2, result[0].ID)
	assert.Equal(t, 3, result[1].ID)
	assert.Equal(t, 1, result[2].ID)
}

func TestSortByState(t *testing.T) {
	notifications := []Notification{
		{ID: 1, State: StateDismissed},
		{ID: 2, State: StateActive},
		{ID: 3, State: StateActive},
	}

	result := SortByState(notifications, SortOrderAsc)
	assert.Equal(t, StateActive, result[0].State)
	assert.Equal(t, StateDismissed, result[2].State)
}

func TestSortByLevel(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Level: LevelError},
		{ID: 2, Level: LevelInfo},
		{ID: 3, Level: LevelWarning},
	}

	result := SortByLevel(notifications, SortOrderAsc)
	assert.Equal(t, LevelError, result[0].Level)
	assert.Equal(t, LevelInfo, result[1].Level)
	assert.Equal(t, LevelWarning, result[2].Level)
}

func TestSortBySession(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Session: "$3"},
		{ID: 2, Session: "$1"},
		{ID: 3, Session: "$2"},
	}

	result := SortBySession(notifications, SortOrderAsc)
	assert.Equal(t, "$1", result[0].Session)
	assert.Equal(t, "$2", result[1].Session)
	assert.Equal(t, "$3", result[2].Session)
}

func TestSortByMessage(t *testing.T) {
	notifications := []Notification{
		{ID: 1, Message: "zebra"},
		{ID: 2, Message: "apple"},
		{ID: 3, Message: "banana"},
	}

	result := SortByMessage(notifications, SortOrderAsc, false)
	assert.Equal(t, "apple", result[0].Message)
	assert.Equal(t, "banana", result[1].Message)
	assert.Equal(t, "zebra", result[2].Message)
}

func TestSortByReadStatus(t *testing.T) {
	notifications := []Notification{
		{ID: 1, ReadTimestamp: "2024-01-01T12:00:00Z"}, // read
		{ID: 2, ReadTimestamp: ""},                     // unread
		{ID: 3, ReadTimestamp: "2024-01-02T12:00:00Z"}, // read
		{ID: 4, ReadTimestamp: ""},                     // unread
	}

	result := SortByReadStatus(notifications, SortOrderAsc)
	// unread first (ID 2, 4), then read (ID 1, 3)
	assert.Equal(t, 2, result[0].ID)
	assert.Equal(t, 4, result[1].ID)
	assert.Equal(t, 1, result[2].ID)
	assert.Equal(t, 3, result[3].ID)
}

func TestDefaultSortOptions(t *testing.T) {
	opts := DefaultSortOptions()
	assert.Equal(t, SortByTimestampField, opts.Field)
	assert.Equal(t, SortOrderDesc, opts.Order)
	assert.False(t, opts.CaseInsensitive)
}

func TestParseSortByField(t *testing.T) {
	tests := []struct {
		input   string
		want    SortByField
		wantErr bool
	}{
		{"id", SortByIDField, false},
		{"timestamp", SortByTimestampField, false},
		{"state", SortByStateField, false},
		{"level", SortByLevelField, false},
		{"session", SortBySessionField, false},
		{"message", SortByMessageField, false},
		{"read_status", SortByReadStatusField, false},
		{"invalid", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSortByField(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseSortOrder(t *testing.T) {
	tests := []struct {
		input   string
		want    SortOrder
		wantErr bool
	}{
		{"asc", SortOrderAsc, false},
		{"desc", SortOrderDesc, false},
		{"invalid", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSortOrder(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSortWithUnreadFirst(t *testing.T) {
	t.Run("unread first with timestamp desc", func(t *testing.T) {
		notifs := []Notification{
			{ID: 1, Timestamp: "2024-01-03T12:00:00Z", ReadTimestamp: "2024-01-04T12:00:00Z"}, // read
			{ID: 2, Timestamp: "2024-01-01T12:00:00Z", ReadTimestamp: ""},                     // unread
			{ID: 3, Timestamp: "2024-01-02T12:00:00Z", ReadTimestamp: "2024-01-05T12:00:00Z"}, // read
			{ID: 4, Timestamp: "2024-01-04T12:00:00Z", ReadTimestamp: ""},                     // unread
		}
		opts := SortOptions{Field: SortByTimestampField, Order: SortOrderDesc}
		result := SortWithUnreadFirst(notifs, opts)

		// Verify unread items come first
		assert.False(t, result[0].IsRead())
		assert.False(t, result[1].IsRead())
		assert.True(t, result[2].IsRead())
		assert.True(t, result[3].IsRead())

		// Verify timestamp order within each group (descending)
		// Unread: ID 4 (2024-01-04) comes before ID 2 (2024-01-01)
		assert.Equal(t, 4, result[0].ID)
		assert.Equal(t, 2, result[1].ID)
		// Read: ID 1 (2024-01-03) comes before ID 3 (2024-01-02)
		assert.Equal(t, 1, result[2].ID)
		assert.Equal(t, 3, result[3].ID)
	})

	t.Run("unread first with timestamp asc", func(t *testing.T) {
		notifs := []Notification{
			{ID: 1, Timestamp: "2024-01-03T12:00:00Z", ReadTimestamp: "2024-01-04T12:00:00Z"}, // read
			{ID: 2, Timestamp: "2024-01-01T12:00:00Z", ReadTimestamp: ""},                     // unread
			{ID: 3, Timestamp: "2024-01-02T12:00:00Z", ReadTimestamp: "2024-01-05T12:00:00Z"}, // read
			{ID: 4, Timestamp: "2024-01-04T12:00:00Z", ReadTimestamp: ""},                     // unread
		}
		opts := SortOptions{Field: SortByTimestampField, Order: SortOrderAsc}
		result := SortWithUnreadFirst(notifs, opts)

		// Verify unread items come first
		assert.False(t, result[0].IsRead())
		assert.False(t, result[1].IsRead())
		assert.True(t, result[2].IsRead())
		assert.True(t, result[3].IsRead())

		// Verify timestamp order within each group (ascending)
		// Unread: ID 2 (2024-01-01) comes before ID 4 (2024-01-04)
		assert.Equal(t, 2, result[0].ID)
		assert.Equal(t, 4, result[1].ID)
		// Read: ID 3 (2024-01-02) comes before ID 1 (2024-01-03)
		assert.Equal(t, 3, result[2].ID)
		assert.Equal(t, 1, result[3].ID)
	})

	t.Run("unread first with id desc", func(t *testing.T) {
		notifs := []Notification{
			{ID: 1, ReadTimestamp: "2024-01-04T12:00:00Z"}, // read
			{ID: 2, ReadTimestamp: ""},                     // unread
			{ID: 3, ReadTimestamp: "2024-01-05T12:00:00Z"}, // read
			{ID: 4, ReadTimestamp: ""},                     // unread
		}
		opts := SortOptions{Field: SortByIDField, Order: SortOrderDesc}
		result := SortWithUnreadFirst(notifs, opts)

		// Verify unread items come first, sorted by ID desc
		assert.Equal(t, 4, result[0].ID) // unread, larger ID
		assert.Equal(t, 2, result[1].ID) // unread, smaller ID
		assert.Equal(t, 3, result[2].ID) // read, larger ID
		assert.Equal(t, 1, result[3].ID) // read, smaller ID
	})

	t.Run("all unread notifications", func(t *testing.T) {
		notifs := []Notification{
			{ID: 3, Timestamp: "2024-01-03T12:00:00Z", ReadTimestamp: ""},
			{ID: 1, Timestamp: "2024-01-01T12:00:00Z", ReadTimestamp: ""},
			{ID: 2, Timestamp: "2024-01-02T12:00:00Z", ReadTimestamp: ""},
		}
		opts := SortOptions{Field: SortByTimestampField, Order: SortOrderDesc}
		result := SortWithUnreadFirst(notifs, opts)

		// All are unread, should be sorted by timestamp descending
		assert.Equal(t, 3, result[0].ID)
		assert.Equal(t, 2, result[1].ID)
		assert.Equal(t, 1, result[2].ID)
	})

	t.Run("all read notifications", func(t *testing.T) {
		notifs := []Notification{
			{ID: 3, Timestamp: "2024-01-03T12:00:00Z", ReadTimestamp: "2024-01-03T13:00:00Z"},
			{ID: 1, Timestamp: "2024-01-01T12:00:00Z", ReadTimestamp: "2024-01-01T13:00:00Z"},
			{ID: 2, Timestamp: "2024-01-02T12:00:00Z", ReadTimestamp: "2024-01-02T13:00:00Z"},
		}
		opts := SortOptions{Field: SortByTimestampField, Order: SortOrderDesc}
		result := SortWithUnreadFirst(notifs, opts)

		// All are read, should be sorted by timestamp descending
		assert.Equal(t, 3, result[0].ID)
		assert.Equal(t, 2, result[1].ID)
		assert.Equal(t, 1, result[2].ID)
	})

	t.Run("empty notifications", func(t *testing.T) {
		opts := SortOptions{Field: SortByTimestampField, Order: SortOrderDesc}
		result := SortWithUnreadFirst([]Notification{}, opts)
		assert.Len(t, result, 0)
	})
}
