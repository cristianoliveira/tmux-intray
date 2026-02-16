package format

import (
	"bytes"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestDefaultTableConfig(t *testing.T) {
	config := DefaultTableConfig()

	assert.True(t, config.ShowHeaders)
	assert.Equal(t, "\x1b[0;34m", config.HeaderColor)
	assert.Equal(t, 4, config.ColumnWidths["ID"])
	assert.Equal(t, 23, config.ColumnWidths["Date"])
	assert.Equal(t, 32, config.ColumnWidths["Message"])
	assert.Equal(t, "right", config.ColumnAlignments["ID"])
	assert.Equal(t, "left", config.ColumnAlignments["Date"])
}

func TestExtendedTableFormatter(t *testing.T) {
	formatter := NewExtendedTableFormatter()
	var buf bytes.Buffer

	notifications := []*domain.Notification{
		{
			ID:        1,
			Timestamp: "2025-01-01T10:00:00Z",
			Level:     domain.LevelInfo,
			Message:   "short message",
		},
		{
			ID:        2,
			Timestamp: "2025-01-01T11:00:00Z",
			Level:     domain.LevelWarning,
			Message:   "this is a very long message that should be truncated",
		},
	}

	err := formatter.FormatNotifications(notifications, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "Date")
	assert.Contains(t, output, "Level")
	assert.Contains(t, output, "Message")
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "2")
	assert.Contains(t, output, "info")
	assert.Contains(t, output, "warning")
	assert.Contains(t, output, "short message")
	assert.Contains(t, output, "this is a very long message t...")
}

func TestExtendedTableFormatterWithCustomColumn(t *testing.T) {
	formatter := NewExtendedTableFormatter()

	// Add a custom column for Session
	formatter.WithColumns(TableColumn{
		Name:      "Session",
		Width:     10,
		Extractor: func(n *domain.Notification) string { return formatString(n.Session, 10, "left") },
	})

	var buf bytes.Buffer

	notifications := []*domain.Notification{
		{
			ID:        1,
			Timestamp: "2025-01-01T10:00:00Z",
			Level:     domain.LevelInfo,
			Session:   "session1",
			Message:   "test message",
		},
	}

	err := formatter.FormatNotifications(notifications, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "Date")
	assert.Contains(t, output, "Level")
	assert.Contains(t, output, "Message")
	assert.Contains(t, output, "Session")
	assert.Contains(t, output, "session1")
}

func TestFormatString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		width     int
		alignment string
		expected  string
	}{
		{"Left align short", "test", 10, "left", "test      "},
		{"Right align short", "test", 10, "right", "      test"},
		{"Center align short", "test", 10, "center", "   test   "},
		{"Left align long", "very long string", 10, "left", "very long "},
		{"Right align long", "very long string", 10, "right", "very long "},
		{"Center align long", "very long string", 10, "center", "very long "},
		{"Default alignment", "test", 10, "invalid", "test      "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatString(tt.input, tt.width, tt.alignment)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{"Short string", "test", 10, "test      "},
		{"Exactly width", "1234567890", 10, "1234567890"},
		{"Truncate", "very long string", 10, "very lo..."},
		{"Truncate with ellipsis", "very long string", 12, "very long..."},
		{"Truncate with ellipsis small width", "test", 3, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.width)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMakeSeparator(t *testing.T) {
	result := makeSeparator(10)
	assert.Equal(t, "----------", result)
}

func TestFormatIntToString(t *testing.T) {
	tests := []struct {
		name      string
		input     int
		width     int
		alignment string
		expected  string
	}{
		{"Left align short", 1, 10, "left", "1         "},
		{"Right align short", 1, 10, "right", "         1"},
		{"Center align short", 1, 10, "center", "    1     "},
		{"Left align long", 1234567890, 5, "left", "12345"},
		{"Right align long", 1234567890, 5, "right", "12345"},
		{"Center align long", 1234567890, 5, "center", "12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatIntToString(tt.input, tt.width, tt.alignment)
			assert.Equal(t, tt.expected, result)
		})
	}
}
