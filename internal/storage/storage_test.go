package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStorageTest(t *testing.T) {
	t.Helper()
	Reset()
	t.Cleanup(Reset)

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.toml"))
}

func TestNormalizeFields(t *testing.T) {
	t.Run("returns error when too few fields", func(t *testing.T) {
		fields := []string{"one", "two", "three"}
		result, err := NormalizeFields(fields)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected at least")
		assert.Nil(t, result)
	})

	t.Run("pads with empty strings when between MinFields and NumFields", func(t *testing.T) {
		// MinFields is 9, NumFields is 10
		fields := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
		result, err := NormalizeFields(fields)
		require.NoError(t, err)
		assert.Len(t, result, NumFields)
		// Original fields preserved
		assert.Equal(t, "1", result[0])
		assert.Equal(t, "9", result[8])
		// Padded field is empty
		assert.Empty(t, result[9])
	})

	t.Run("returns same slice when already at NumFields", func(t *testing.T) {
		fields := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
		result, err := NormalizeFields(fields)
		require.NoError(t, err)
		assert.Equal(t, fields, result)
	})

	t.Run("handles exactly MinFields", func(t *testing.T) {
		fields := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
		result, err := NormalizeFields(fields)
		require.NoError(t, err)
		assert.Len(t, result, NumFields)
	})
}

func TestEscapeMessage(t *testing.T) {
	t.Run("escapes backslashes", func(t *testing.T) {
		msg := `path\to\file`
		result := EscapeMessage(msg)
		assert.Equal(t, `path\\to\\file`, result)
	})

	t.Run("escapes tabs", func(t *testing.T) {
		msg := "col1\tcol2\tcol3"
		result := EscapeMessage(msg)
		assert.Equal(t, `col1\tcol2\tcol3`, result)
	})

	t.Run("escapes newlines", func(t *testing.T) {
		msg := "line1\nline2\nline3"
		result := EscapeMessage(msg)
		assert.Equal(t, `line1\nline2\nline3`, result)
	})

	t.Run("escapes all special characters together", func(t *testing.T) {
		msg := "path\\to\\file\nwith\tnewline\tand\ttab"
		result := EscapeMessage(msg)
		assert.Equal(t, `path\\to\\file\nwith\tnewline\tand\ttab`, result)
	})

	t.Run("returns empty string unchanged", func(t *testing.T) {
		result := EscapeMessage("")
		assert.Equal(t, "", result)
	})

	t.Run("returns normal string unchanged", func(t *testing.T) {
		msg := "normal message without special chars"
		result := EscapeMessage(msg)
		assert.Equal(t, msg, result)
	})
}

func TestUnescapeMessage(t *testing.T) {
	t.Run("unescapes newlines", func(t *testing.T) {
		msg := `line1\nline2\nline3`
		result := UnescapeMessage(msg)
		assert.Equal(t, "line1\nline2\nline3", result)
	})

	t.Run("unescapes tabs", func(t *testing.T) {
		msg := `col1\tcol2\tcol3`
		result := UnescapeMessage(msg)
		assert.Equal(t, "col1\tcol2\tcol3", result)
	})

	t.Run("unescapes backslashes", func(t *testing.T) {
		msg := `path\\dir\\file`
		result := UnescapeMessage(msg)
		expected := "path\\dir\\file"
		assert.Equal(t, expected, result)
	})

	t.Run("unescapes mixed newline and tab", func(t *testing.T) {
		msg := `row1\tcol2\nrow2\tcol2`
		result := UnescapeMessage(msg)
		expected := "row1\tcol2\nrow2\tcol2"
		assert.Equal(t, expected, result)
	})

	t.Run("returns empty string unchanged", func(t *testing.T) {
		result := UnescapeMessage("")
		assert.Equal(t, "", result)
	})

	t.Run("returns normal string unchanged", func(t *testing.T) {
		msg := "normal message without special chars"
		result := UnescapeMessage(msg)
		assert.Equal(t, msg, result)
	})
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"simple", "hello world"},
		{"with tabs", "col1\tcol2\tcol3"},
		{"with newlines", "line1\nline2\nline3"},
		{"with mixed newline and tab", "row1\tcol2\nrow2\tcol2"},
		{"multiline text", "first line\nsecond line\nthird line"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			escaped := EscapeMessage(tt.message)
			unescaped := UnescapeMessage(escaped)
			assert.Equal(t, tt.message, unescaped, "round-trip should preserve original message")
		})
	}
}

func TestGetActiveCount_WithStorage(t *testing.T) {
	setupStorageTest(t)

	// Initialize storage first
	require.NoError(t, Init())

	// GetActiveCount should return 0 when there are no notifications
	count := GetActiveCount()
	assert.Equal(t, 0, count)
}

func TestAddNotification_WithStorage(t *testing.T) {
	setupStorageTest(t)

	require.NoError(t, Init())

	id, err := AddNotification("test message", "2025-01-01T12:00:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestListNotifications_WithStorage(t *testing.T) {
	setupStorageTest(t)

	require.NoError(t, Init())

	// Add a notification first
	_, err := AddNotification("test message", "2025-01-01T12:00:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)

	// List all notifications
	result, err := ListNotifications("", "", "", "", "", "", "", "")
	require.NoError(t, err)
	assert.Contains(t, result, "test message")
}

func TestGetNotificationByID_WithStorage(t *testing.T) {
	setupStorageTest(t)

	require.NoError(t, Init())

	// Add a notification first
	id, err := AddNotification("test message", "2025-01-01T12:00:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)

	// Get by ID
	result, err := GetNotificationByID(id)
	require.NoError(t, err)
	assert.Contains(t, result, "test message")
}

func TestDismissNotification_WithStorage(t *testing.T) {
	setupStorageTest(t)

	require.NoError(t, Init())

	// Add a notification first
	id, err := AddNotification("test message", "2025-01-01T12:00:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)

	// Dismiss it
	err = DismissNotification(id)
	require.NoError(t, err)

	// Verify it's dismissed (should appear in "all" but not "active")
	activeResult, err := ListNotifications("active", "", "", "", "", "", "", "")
	require.NoError(t, err)
	assert.NotContains(t, activeResult, id)
}

func TestDismissAll_WithStorage(t *testing.T) {
	setupStorageTest(t)

	require.NoError(t, Init())

	// Add multiple notifications
	_, err := AddNotification("message 1", "2025-01-01T12:00:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)
	_, err = AddNotification("message 2", "2025-01-01T12:01:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)

	// Dismiss all
	err = DismissAll()
	require.NoError(t, err)

	// Verify count is 0
	count := GetActiveCount()
	assert.Equal(t, 0, count)
}

func TestMarkNotificationRead_WithStorage(t *testing.T) {
	setupStorageTest(t)

	require.NoError(t, Init())

	// Add a notification first
	id, err := AddNotification("test message", "2025-01-01T12:00:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)

	// Mark as read
	err = MarkNotificationRead(id)
	require.NoError(t, err)

	// Verify it's read by listing unread
	unreadResult, err := ListNotifications("", "", "", "", "", "", "", "unread")
	require.NoError(t, err)
	assert.NotContains(t, unreadResult, id)
}

func TestMarkNotificationUnread_WithStorage(t *testing.T) {
	setupStorageTest(t)

	require.NoError(t, Init())

	// Add a notification first
	id, err := AddNotification("test message", "2025-01-01T12:00:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)

	// Mark as read first
	err = MarkNotificationRead(id)
	require.NoError(t, err)

	// Mark as unread
	err = MarkNotificationUnread(id)
	require.NoError(t, err)

	// Verify it's unread
	unreadResult, err := ListNotifications("", "", "", "", "", "", "", "unread")
	require.NoError(t, err)
	assert.Contains(t, unreadResult, id)
}

func TestMarkNotificationReadWithTimestamp_WithStorage(t *testing.T) {
	setupStorageTest(t)

	require.NoError(t, Init())

	// Add a notification first
	id, err := AddNotification("test message", "2025-01-01T12:00:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)

	// Mark as read with specific timestamp
	err = MarkNotificationReadWithTimestamp(id, "2025-01-02T10:00:00Z")
	require.NoError(t, err)

	// Verify it's read
	readResult, err := ListNotifications("", "", "", "", "", "", "", "read")
	require.NoError(t, err)
	assert.Contains(t, readResult, id)
}

func TestMarkNotificationUnreadWithTimestamp_WithStorage(t *testing.T) {
	setupStorageTest(t)

	require.NoError(t, Init())

	// Add a notification first
	id, err := AddNotification("test message", "2025-01-01T12:00:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)

	// Mark as read first
	err = MarkNotificationRead(id)
	require.NoError(t, err)

	// Mark as unread with timestamp (timestamp is ignored per implementation)
	err = MarkNotificationUnreadWithTimestamp(id, "2025-01-02T10:00:00Z")
	require.NoError(t, err)

	// Verify we can still get the notification (should be active)
	notif, err := GetNotificationByID(id)
	require.NoError(t, err)
	assert.Contains(t, notif, "test message")
}

func TestCleanupOldNotifications_WithStorage(t *testing.T) {
	setupStorageTest(t)

	require.NoError(t, Init())

	// Add a notification
	_, err := AddNotification("test message", "2025-01-01T12:00:00Z", "session1", "window0", "pane0", "123456", "info")
	require.NoError(t, err)

	// Cleanup with dry-run (won't actually delete)
	err = CleanupOldNotifications(30, true)
	require.NoError(t, err)

	// Verify notification still exists (dry-run)
	count := GetActiveCount()
	assert.Equal(t, 1, count)
}

func TestGetActiveCount_ReturnsZeroOnStorageError(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	// Create a file at the path where we'll try to create a directory
	// This will cause mkdir to fail
	tmpFile := filepath.Join(t.TempDir(), "blocked")
	require.NoError(t, os.WriteFile(tmpFile, []byte("test"), 0644))

	// Set state_dir to a path that cannot be created (file exists where dir should be)
	t.Setenv("TMUX_INTRAY_STATE_DIR", filepath.Join(tmpFile, "subdir"))
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.toml"))

	// GetActiveCount should return 0 when storage fails to initialize
	count := GetActiveCount()
	assert.Equal(t, 0, count)
}
