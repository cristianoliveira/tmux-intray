package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) string {
	tmpDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_DEBUG", "true")
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")
	colors.SetDebug(true)
	// Reset package state
	notificationsFile = ""
	lockDir = ""
	initialized = false
	return tmpDir
}

func TestStorageInit(t *testing.T) {
	tmpDir := setupTest(t)
	Init()
	require.True(t, initialized)
	// Check notifications file exists
	require.FileExists(t, filepath.Join(tmpDir, "notifications.tsv"))
}

func TestAddNotification(t *testing.T) {
	setupTest(t)
	Init()
	id := AddNotification("test message", "", "session1", "window0", "pane0", "", "info")
	require.NotEmpty(t, id)
	// Should be numeric
	require.Regexp(t, `^\d+$`, id)
	// List notifications should contain one active
	list := ListNotifications("active", "", "", "", "", "", "")
	require.Contains(t, list, id)
	require.Contains(t, list, "test message")
}

func TestAddNotificationWithTimestamp(t *testing.T) {
	setupTest(t)
	Init()
	id := AddNotification("msg", "2025-01-01T12:00:00Z", "", "", "", "", "warning")
	require.NotEmpty(t, id)
	list := ListNotifications("all", "", "", "", "", "", "")
	require.Contains(t, list, "2025-01-01T12:00:00Z")
	require.Contains(t, list, "warning")
}

func TestListNotificationsFilters(t *testing.T) {
	setupTest(t)
	Init()
	// Add multiple notifications with different attributes
	id1 := AddNotification("error msg", "", "session1", "window1", "pane1", "", "error")
	id2 := AddNotification("info msg", "", "session2", "window2", "pane2", "", "info")
	require.NotEqual(t, id1, id2)

	// Helper to check IDs in list
	assertContainsID := func(list string, id string) {
		lines := strings.Split(strings.TrimSpace(list), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			fields := strings.Split(line, "\t")
			if len(fields) > 0 && fields[0] == id {
				return
			}
		}
		t.Errorf("list does not contain ID %s", id)
	}
	assertNotContainsID := func(list string, id string) {
		lines := strings.Split(strings.TrimSpace(list), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			fields := strings.Split(line, "\t")
			if len(fields) > 0 && fields[0] == id {
				t.Errorf("list contains ID %s", id)
			}
		}
	}

	// Filter by state active
	list := ListNotifications("active", "", "", "", "", "", "")
	assertContainsID(list, id1)
	assertContainsID(list, id2)

	// Filter by level
	list = ListNotifications("all", "error", "", "", "", "", "")
	assertContainsID(list, id1)
	assertNotContainsID(list, id2)

	// Filter by session
	list = ListNotifications("all", "", "session1", "", "", "", "")
	assertContainsID(list, id1)
	assertNotContainsID(list, id2)

	// Filter by window
	list = ListNotifications("all", "", "", "window2", "", "", "")
	assertContainsID(list, id2)
	assertNotContainsID(list, id1)

	// Filter by pane
	list = ListNotifications("all", "", "", "", "pane1", "", "")
	assertContainsID(list, id1)
	assertNotContainsID(list, id2)
}

func TestDismissNotification(t *testing.T) {
	setupTest(t)
	Init()
	id := AddNotification("to dismiss", "", "", "", "", "", "info")
	require.NotEmpty(t, id)
	// Should be active
	list := ListNotifications("active", "", "", "", "", "", "")
	require.Contains(t, list, id)
	// Dismiss
	err := DismissNotification(id)
	require.NoError(t, err)
	// Should not appear in active
	list = ListNotifications("active", "", "", "", "", "", "")
	require.NotContains(t, list, id)
	// Should appear in dismissed
	list = ListNotifications("dismissed", "", "", "", "", "", "")
	require.Contains(t, list, id)
	// Dismissing again should return error
	err = DismissNotification(id)
	require.Error(t, err)
}

func TestDismissAllFromStorage(t *testing.T) {
	setupTest(t)
	Init()
	id1 := AddNotification("msg1", "", "", "", "", "", "info")
	id2 := AddNotification("msg2", "", "", "", "", "", "warning")
	require.Equal(t, 2, GetActiveCount())
	err := DismissAll()
	require.NoError(t, err)
	require.Equal(t, 0, GetActiveCount())
	list := ListNotifications("dismissed", "", "", "", "", "", "")
	require.Contains(t, list, id1)
	require.Contains(t, list, id2)
}

func TestCleanupOldNotifications(t *testing.T) {
	setupTest(t)
	Init()
	// Add a notification with old timestamp
	id := AddNotification("old", "2000-01-01T00:00:00Z", "", "", "", "", "info")
	_ = DismissNotification(id)
	// Cleanup with threshold 1 day (dry run)
	CleanupOldNotifications(1, true)
	// Should still exist
	list := ListNotifications("all", "", "", "", "", "", "")
	require.Contains(t, list, id)
	// Real cleanup (should delete because timestamp is very old)
	CleanupOldNotifications(1, false)
	list = ListNotifications("all", "", "", "", "", "", "")
	require.NotContains(t, list, id)
}

func TestGetActiveCount(t *testing.T) {
	setupTest(t)
	Init()
	require.Equal(t, 0, GetActiveCount())
	id1 := AddNotification("msg1", "", "", "", "", "", "info")
	require.Equal(t, 1, GetActiveCount())
	_ = AddNotification("msg2", "", "", "", "", "", "warning")
	require.Equal(t, 2, GetActiveCount())
	// Dismiss one
	_ = DismissNotification(id1)
	require.Equal(t, 1, GetActiveCount())
	_ = DismissAll()
	require.Equal(t, 0, GetActiveCount())
}
