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
