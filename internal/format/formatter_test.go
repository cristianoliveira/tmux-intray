package format

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestFormatterFactory(t *testing.T) {
	tests := []struct {
		name     string
		ftype    FormatterType
		expected interface{}
	}{
		{"Simple", FormatterTypeSimple, &SimpleFormatter{}},
		{"Legacy", FormatterTypeLegacy, &LegacyFormatter{}},
		{"Table", FormatterTypeTable, &TableFormatter{}},
		{"Compact", FormatterTypeCompact, &CompactFormatter{}},
		{"JSON", FormatterTypeJSON, &JSONFormatter{}},
		{"Unknown", FormatterType("unknown"), &SimpleFormatter{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.ftype)
			assert.IsType(t, tt.expected, formatter)
		})
	}
}

func TestSimpleFormatter(t *testing.T) {
	formatter := NewSimpleFormatter()
	var buf bytes.Buffer

	notifications := []*domain.Notification{
		{
			ID:        1,
			Timestamp: "2025-01-01T10:00:00Z",
			Message:   "short message",
		},
		{
			ID:        2,
			Timestamp: "2025-01-01T11:00:00Z",
			Message:   "this is a very long message that should be truncated because it exceeds the maximum allowed length for display",
		},
	}

	err := formatter.FormatNotifications(notifications, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "2025-01-01T10:00:00Z")
	assert.Contains(t, output, "short message")
	assert.Contains(t, output, "this is a very long message that should be trun...")
	assert.NotContains(t, output, "because it exceeds")
}

func TestLegacyFormatter(t *testing.T) {
	formatter := NewLegacyFormatter()
	var buf bytes.Buffer

	notifications := []*domain.Notification{
		{
			ID:      1,
			Message: "message one",
		},
		{
			ID:      2,
			Message: "message two",
		},
	}

	err := formatter.FormatNotifications(notifications, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Equal(t, "message one\nmessage two\n", output)
}

func TestTableFormatter(t *testing.T) {
	formatter := NewTableFormatter()
	var buf bytes.Buffer

	notifications := []*domain.Notification{
		{
			ID:        1,
			Timestamp: "2025-01-01T10:00:00Z",
			Message:   "short message",
		},
		{
			ID:        2,
			Timestamp: "2025-01-01T11:00:00Z",
			Message:   "this is a very long message that should be truncated",
		},
	}

	err := formatter.FormatNotifications(notifications, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "DATE")
	assert.Contains(t, output, "Message")
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "2")
	assert.Contains(t, output, "2025-01-01T10:00:00Z")
	assert.Contains(t, output, "short message")
	assert.Contains(t, output, "this is a very long message t...")
}

func TestCompactFormatter(t *testing.T) {
	formatter := NewCompactFormatter()
	var buf bytes.Buffer

	notifications := []*domain.Notification{
		{
			Message: "short message",
		},
		{
			Message: "this is a very long message that should be truncated because it exceeds the maximum allowed length for display",
		},
	}

	err := formatter.FormatNotifications(notifications, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "short message")
	assert.Contains(t, output, "this is a very long message that should be truncated beca...")
	assert.NotContains(t, output, "it exceeds")
}

func TestJSONFormatter(t *testing.T) {
	formatter := NewJSONFormatter()
	var buf bytes.Buffer

	notifications := []*domain.Notification{
		{
			ID:        1,
			Timestamp: "2025-01-01T10:00:00Z",
			State:     domain.StateActive,
			Session:   "sess1",
			Window:    "win1",
			Pane:      "pane1",
			Message:   "test message",
			Level:     domain.LevelInfo,
		},
	}

	err := formatter.FormatNotifications(notifications, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"ID": 1`)
	assert.Contains(t, output, `"Timestamp": "2025-01-01T10:00:00Z"`)
	assert.Contains(t, output, `"State": "active"`)
	assert.Contains(t, output, `"Message": "test message"`)
}

func TestGroupCountFormatter(t *testing.T) {
	baseFormatter := NewSimpleFormatter()
	formatter := NewGroupCountFormatter(baseFormatter)
	var buf bytes.Buffer

	groups := domain.GroupResult{
		Mode: domain.GroupByLevel,
		Groups: []domain.Group{
			{
				DisplayName: "info",
				Count:       2,
			},
			{
				DisplayName: "warning",
				Count:       1,
			},
		},
	}

	err := formatter.FormatGroups(groups, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Group: info (2)")
	assert.Contains(t, output, "Group: warning (1)")
}

func TestGetFormatter(t *testing.T) {
	// Test valid formatters
	f := GetFormatter("simple", false)
	assert.IsType(t, &SimpleFormatter{}, f)

	f = GetFormatter("legacy", false)
	assert.IsType(t, &LegacyFormatter{}, f)

	f = GetFormatter("table", false)
	assert.IsType(t, &TableFormatter{}, f)

	f = GetFormatter("compact", false)
	assert.IsType(t, &CompactFormatter{}, f)

	f = GetFormatter("json", false)
	assert.IsType(t, &JSONFormatter{}, f)

	// Test unknown formatter (should fall back to simple)
	f = GetFormatter("unknown", false)
	assert.IsType(t, &SimpleFormatter{}, f)

	// Test group count formatter
	f = GetFormatter("simple", true)
	assert.IsType(t, &GroupCountFormatter{}, f)
}

func TestFormatterGroups(t *testing.T) {
	formatter := NewSimpleFormatter()
	var buf bytes.Buffer

	groups := domain.GroupResult{
		Mode: domain.GroupByLevel,
		Groups: []domain.Group{
			{
				DisplayName: "info",
				Count:       2,
				Notifications: []domain.Notification{
					{ID: 1, Message: "info message 1"},
					{ID: 2, Message: "info message 2"},
				},
			},
		},
	}

	err := formatter.FormatGroups(groups, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "=== info (2) ===")
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "info message 1")
}

func TestNotificationsToPointers(t *testing.T) {
	notifs := []domain.Notification{
		{ID: 1, Message: "test"},
		{ID: 2, Message: "test2"},
	}
	ptrs := notificationsToPointers(notifs)
	assert.Len(t, ptrs, 2)
	assert.Equal(t, 1, ptrs[0].ID)
	assert.Equal(t, 2, ptrs[1].ID)
}

func TestLegacyFormatterGroups(t *testing.T) {
	formatter := NewLegacyFormatter()
	var buf bytes.Buffer

	groups := domain.GroupResult{
		Mode: domain.GroupByLevel,
		Groups: []domain.Group{
			{
				DisplayName: "info",
				Count:       2,
				Notifications: []domain.Notification{
					{ID: 1, Message: "info message 1"},
					{ID: 2, Message: "info message 2"},
				},
			},
		},
	}

	err := formatter.FormatGroups(groups, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "=== info (2) ===")
	assert.Contains(t, output, "info message 1")
	assert.Contains(t, output, "info message 2")
}

func TestTableFormatterGroups(t *testing.T) {
	formatter := NewTableFormatter()
	var buf bytes.Buffer

	groups := domain.GroupResult{
		Mode: domain.GroupByLevel,
		Groups: []domain.Group{
			{
				DisplayName: "info",
				Count:       2,
				Notifications: []domain.Notification{
					{ID: 1, Timestamp: "2025-01-01T10:00:00Z", Message: "info message 1"},
					{ID: 2, Timestamp: "2025-01-01T11:00:00Z", Message: "info message 2"},
				},
			},
		},
	}

	err := formatter.FormatGroups(groups, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "=== info (2) ===")
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "DATE")
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "2")
	assert.Contains(t, output, "info message 1")
	assert.Contains(t, output, "info message 2")
}

func TestCompactFormatterGroups(t *testing.T) {
	formatter := NewCompactFormatter()
	var buf bytes.Buffer

	groups := domain.GroupResult{
		Mode: domain.GroupByLevel,
		Groups: []domain.Group{
			{
				DisplayName: "info",
				Count:       2,
				Notifications: []domain.Notification{
					{ID: 1, Message: "info message 1"},
					{ID: 2, Message: "info message 2"},
				},
			},
		},
	}

	err := formatter.FormatGroups(groups, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "=== info (2) ===")
	assert.Contains(t, output, "info message 1")
	assert.Contains(t, output, "info message 2")
}

func TestJSONFormatterGroups(t *testing.T) {
	formatter := NewJSONFormatter()
	var buf bytes.Buffer

	groups := domain.GroupResult{
		Mode: domain.GroupByLevel,
		Groups: []domain.Group{
			{
				DisplayName: "info",
				Count:       2,
				Notifications: []domain.Notification{
					{ID: 1, Message: "info message 1"},
					{ID: 2, Message: "info message 2"},
				},
			},
		},
	}

	err := formatter.FormatGroups(groups, &buf)
	assert.NoError(t, err)

	// Parse JSON back
	var decoded domain.GroupResult
	err = json.Unmarshal(buf.Bytes(), &decoded)
	assert.NoError(t, err)
	assert.Equal(t, groups.Mode, decoded.Mode)
	assert.Len(t, decoded.Groups, 1)
	assert.Equal(t, groups.Groups[0].DisplayName, decoded.Groups[0].DisplayName)
	assert.Equal(t, groups.Groups[0].Count, decoded.Groups[0].Count)
	assert.Len(t, decoded.Groups[0].Notifications, 2)
	assert.Equal(t, groups.Groups[0].Notifications[0].ID, decoded.Groups[0].Notifications[0].ID)
	assert.Equal(t, groups.Groups[0].Notifications[0].Message, decoded.Groups[0].Notifications[0].Message)
}

func TestGroupCountFormatterFormatNotifications(t *testing.T) {
	baseFormatter := NewSimpleFormatter()
	formatter := NewGroupCountFormatter(baseFormatter)
	var buf bytes.Buffer

	notifications := []*domain.Notification{
		{ID: 1, Message: "test"},
	}
	err := formatter.FormatNotifications(notifications, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
