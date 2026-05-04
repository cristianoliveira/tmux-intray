package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNotificationLine_FullLine(t *testing.T) {
	line := "42\t2024-01-01T12:00:00Z\tactive\t$0\t@1\t%2\tHello World\t123\tinfo\t2024-01-02T01:02:03Z"
	n, err := ParseNotificationLine(line)
	require.NoError(t, err)

	assert.Equal(t, 42, n.ID)
	assert.Equal(t, "2024-01-01T12:00:00Z", n.Timestamp)
	assert.Equal(t, StateActive, n.State)
	assert.Equal(t, "$0", n.Session)
	assert.Equal(t, "@1", n.Window)
	assert.Equal(t, "%2", n.Pane)
	assert.Equal(t, "Hello World", n.Message)
	assert.Equal(t, "123", n.PaneCreated)
	assert.Equal(t, LevelInfo, n.Level)
	assert.Equal(t, "2024-01-02T01:02:03Z", n.ReadTimestamp)
}

func TestParseNotificationLine_WithoutReadTimestamp(t *testing.T) {
	line := "42\t2024-01-01T12:00:00Z\tactive\t$0\t@1\t%2\tHello World\t123\tinfo"
	n, err := ParseNotificationLine(line)
	require.NoError(t, err)

	assert.Equal(t, 42, n.ID)
	assert.Equal(t, "", n.ReadTimestamp)
}

func TestParseNotificationLine_InvalidFieldCount(t *testing.T) {
	_, err := ParseNotificationLine("too\tfew\tfields")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid notification field count")
}

func TestParseNotificationLine_RoundTrip(t *testing.T) {
	original := Notification{
		ID:            42,
		Timestamp:     "2024-01-01T12:00:00Z",
		State:         StateActive,
		Session:       "$0",
		Window:        "@1",
		Pane:          "%2",
		Message:       "Hello\tWorld\nTest",
		PaneCreated:   "123",
		Level:         LevelWarning,
		ReadTimestamp: "2024-01-02T01:02:03Z",
	}

	line := original.FormatNotificationLine()
	parsed, err := ParseNotificationLine(line)
	require.NoError(t, err)

	assert.Equal(t, original.ID, parsed.ID)
	assert.Equal(t, original.Timestamp, parsed.Timestamp)
	assert.Equal(t, original.State, parsed.State)
	assert.Equal(t, original.Session, parsed.Session)
	assert.Equal(t, original.Window, parsed.Window)
	assert.Equal(t, original.Pane, parsed.Pane)
	assert.Equal(t, original.Message, parsed.Message)
	assert.Equal(t, original.PaneCreated, parsed.PaneCreated)
	assert.Equal(t, original.Level, parsed.Level)
	assert.Equal(t, original.ReadTimestamp, parsed.ReadTimestamp)
}

func TestFormatNotificationLine(t *testing.T) {
	n := Notification{
		ID:            1,
		Timestamp:     "2024-01-01T12:00:00Z",
		State:         StateActive,
		Session:       "$0",
		Window:        "@1",
		Pane:          "%2",
		Message:       "plain message",
		PaneCreated:   "",
		Level:         LevelInfo,
		ReadTimestamp: "",
	}

	line := n.FormatNotificationLine()
	assert.Equal(t, "1\t2024-01-01T12:00:00Z\tactive\t$0\t@1\t%2\tplain message\t\tinfo\t", line)
}

func TestParseNotificationLine_EmptyFields(t *testing.T) {
	line := "1\t\t\t\t\t\t\t\t\t"
	n, err := ParseNotificationLine(line)
	require.NoError(t, err)

	assert.Equal(t, 1, n.ID)
	assert.Equal(t, "", n.Timestamp)
	assert.Equal(t, NotificationState(""), n.State)
	assert.Equal(t, "", n.Message)
	assert.Equal(t, NotificationLevel(""), n.Level)
}
