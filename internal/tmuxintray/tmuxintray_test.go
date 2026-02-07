package tmuxintray

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/stretchr/testify/require"
)

// setupTest prepares a temporary environment for testing.
// It sets environment variables and resets package state.
// Returns the temporary directory path.
func setupTest(t *testing.T) string {
	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	t.Setenv("TMUX_INTRAY_CONFIG_DIR", tmpDir)
	t.Setenv("TMUX_INTRAY_DEBUG", "true")
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")
	// Reset package states
	storage.Reset()
	hooks.ResetForTesting()
	colors.SetDebug(true)
	return tmpDir
}

func TestInit(t *testing.T) {
	tmpDir := setupTest(t)
	err := Init()
	require.NoError(t, err)
	// Verify that storage is initialized (notifications file exists)
	// Since we set TMUX_INTRAY_STATE_DIR to tmpDir, storage should have created notifications.tsv
	require.FileExists(t, filepath.Join(tmpDir, "notifications.tsv"))
}

func TestAddNotification(t *testing.T) {
	setupTest(t)
	Init()
	id, err := AddNotification("test message", "", "", "", "", false, "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	require.Regexp(t, `^\d+$`, id)
}

func TestListNotifications(t *testing.T) {
	setupTest(t)
	Init()
	id, err := AddNotification("test message", "", "", "", "", false, "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	list := ListNotifications("active", "", "", "", "", "", "")
	require.Contains(t, list, id)
	require.Contains(t, list, "test message")
}

func TestDismissNotification(t *testing.T) {
	setupTest(t)
	Init()
	id, err := AddNotification("test message", "", "", "", "", false, "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	err = DismissNotification(id)
	require.NoError(t, err)
	// Should be dismissed
	list := ListNotifications("active", "", "", "", "", "", "")
	require.NotContains(t, list, id)
	listDismissed := ListNotifications("dismissed", "", "", "", "", "", "")
	require.Contains(t, listDismissed, id)
}

func TestDismissAllNotifications(t *testing.T) {
	setupTest(t)
	Init()
	_, err := AddNotification("msg1", "", "", "", "", false, "info")
	require.NoError(t, err)
	_, err = AddNotification("msg2", "", "", "", "", false, "warning")
	require.NoError(t, err)
	err = DismissAllNotifications()
	require.NoError(t, err)
	list := ListNotifications("active", "", "", "", "", "", "")
	require.Empty(t, strings.TrimSpace(list))
}

func TestCleanupOldNotifications(t *testing.T) {
	setupTest(t)
	Init()
	// Add and dismiss a notification (should be removed after cleanup)
	id, err := AddNotification("old", "", "", "", "", false, "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	err = DismissNotification(id)
	require.NoError(t, err)
	// Cleanup with 0 days threshold (should remove dismissed)
	CleanupOldNotifications(0, false)
	// Verify no dismissed notifications remain
	list := ListNotifications("dismissed", "", "", "", "", "", "")
	require.Empty(t, strings.TrimSpace(list))
}

func TestGetActiveCount(t *testing.T) {
	setupTest(t)
	Init()
	require.Equal(t, 0, GetActiveCount())
	_, err := AddNotification("msg1", "", "", "", "", false, "info")
	require.NoError(t, err)
	require.Equal(t, 1, GetActiveCount())
	_, err = AddNotification("msg2", "", "", "", "", false, "warning")
	require.NoError(t, err)
	require.Equal(t, 2, GetActiveCount())
	// Dismiss one
	id, err := AddNotification("msg3", "", "", "", "", false, "info")
	require.NoError(t, err)
	DismissNotification(id)
	require.Equal(t, 2, GetActiveCount())
}

func TestGetVisibility(t *testing.T) {
	setupTest(t)
	Init()
	// Mock tmux context? This will call core.GetVisibility which calls tmux commands.
	// For now, just ensure no panic.
	visibility := GetVisibility()
	require.Contains(t, []string{"0", "1"}, visibility)
}

func TestSetVisibility(t *testing.T) {
	setupTest(t)
	Init()
	// This will attempt to set tmux option; might fail if tmux not running.
	// We'll just call it and ignore error (should not panic).
	_ = SetVisibility(true)
	_ = SetVisibility(false)
}

func TestShutdown(t *testing.T) {
	setupTest(t)
	Init()
	// Should not panic
	Shutdown()
}

func TestParseNotification(t *testing.T) {
	tsvLine := "1\t2025-01-01T12:00:00Z\tactive\tsess\twin\tpane\ttest message\t123\tinfo"
	notif, err := ParseNotification(tsvLine)
	require.NoError(t, err)
	require.Equal(t, "1", notif.ID)
	require.Equal(t, "2025-01-01T12:00:00Z", notif.Timestamp)
	require.Equal(t, "active", notif.State)
	require.Equal(t, "sess", notif.Session)
	require.Equal(t, "win", notif.Window)
	require.Equal(t, "pane", notif.Pane)
	require.Equal(t, "test message", notif.Message)
	require.Equal(t, "123", notif.PaneCreated)
	require.Equal(t, "info", notif.Level)

	// Test with escaped characters
	escapedLine := "2\t2025-01-01T12:00:00Z\tdismissed\t\tt\t\tline1\\nline2\\ttab\t0\twarning"
	notif2, err := ParseNotification(escapedLine)
	require.NoError(t, err)
	require.Equal(t, "2", notif2.ID)
	require.Equal(t, "line1\nline2\ttab", notif2.Message)
}
