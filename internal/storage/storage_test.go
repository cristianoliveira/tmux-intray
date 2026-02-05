package storage

import (
	"fmt"
	"os"
	"os/exec"
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
	err := CleanupOldNotifications(1, true)
	require.NoError(t, err)
	// Should still exist
	list := ListNotifications("all", "", "", "", "", "", "")
	require.Contains(t, list, id)
	// Real cleanup (should delete because timestamp is very old)
	err = CleanupOldNotifications(1, false)
	require.NoError(t, err)
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

func TestBashStorageCompatibility(t *testing.T) {
	tmpDir := setupTest(t)
	// Find lib directory (project root)
	libDir := ""
	cwd, _ := os.Getwd()
	absPath, _ := filepath.Abs(cwd)

	// Try from current dir and go up looking for lib directory
	currentDir := absPath
	for i := 0; i < 5; i++ { // Limit depth to avoid infinite loops
		testPath := filepath.Join(currentDir, "lib")
		if _, err := os.Stat(testPath); err == nil {
			libDir = testPath
			break
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir { // Reached root
			break
		}
		currentDir = parent
	}

	// If still not found, try relative paths as fallback
	if libDir == "" {
		candidates := []string{
			filepath.Join("lib"),
			filepath.Join("..", "lib"),
			filepath.Join("..", "..", "lib"),
			filepath.Join("../../../lib"),
		}
		for _, cand := range candidates {
			if _, err := os.Stat(cand); err == nil {
				abs, err := filepath.Abs(cand)
				if err == nil {
					libDir = abs
					break
				}
			}
		}
	}

	require.NotEmpty(t, libDir, "lib directory not found")
	require.DirExists(t, libDir)

	// Helper to write and run a bash script that sources storage.sh
	runBashStorageScript := func(scriptContent string) (string, error) {
		scriptFile := filepath.Join(tmpDir, "script.sh")
		err := os.WriteFile(scriptFile, []byte(scriptContent), 0755)
		if err != nil {
			return "", err
		}
		cmd := exec.Command("bash", scriptFile)
		cmd.Env = append(os.Environ(),
			"TMUX_INTRAY_STATE_DIR="+tmpDir,
			"TMUX_INTRAY_HOOKS_ENABLED=0",
			"TMUX_INTRAY_DEBUG=true")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("bash script failed: %w", err)
		}
		return strings.TrimSpace(string(output)), nil
	}

	// Helper to add notification via bash storage
	bashAddNotification := func(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
		// Escape single quotes in message for bash
		// We'll pass arguments via environment variables to avoid quoting hell
		script := fmt.Sprintf(`
set -e
source "%s/storage.sh"
storage_add_notification "$TMUX_INTRAY_MESSAGE" "$TMUX_INTRAY_TIMESTAMP" "$TMUX_INTRAY_SESSION" "$TMUX_INTRAY_WINDOW" "$TMUX_INTRAY_PANE" "$TMUX_INTRAY_PANE_CREATED" "$TMUX_INTRAY_LEVEL"
`, libDir)
		cmd := exec.Command("bash", "-c", script)
		cmd.Env = append(os.Environ(),
			"TMUX_INTRAY_STATE_DIR="+tmpDir,
			"TMUX_INTRAY_HOOKS_ENABLED=0",
			"TMUX_INTRAY_DEBUG=true",
			"TMUX_INTRAY_MESSAGE="+message,
			"TMUX_INTRAY_TIMESTAMP="+timestamp,
			"TMUX_INTRAY_SESSION="+session,
			"TMUX_INTRAY_WINDOW="+window,
			"TMUX_INTRAY_PANE="+pane,
			"TMUX_INTRAY_PANE_CREATED="+paneCreated,
			"TMUX_INTRAY_LEVEL="+level)
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("bash add failed: %w", err)
		}
		id := strings.TrimSpace(string(output))
		return id, nil
	}

	// Helper to list notifications via bash storage
	bashListNotifications := func(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error) {
		script := fmt.Sprintf(`
source "%s/storage.sh"
storage_list_notifications "%s" "%s" "%s" "%s" "%s" "%s" "%s"
`, libDir, stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
		return runBashStorageScript(script)
	}

	// Test 1: Add via bash, read via Go
	t.Run("BashAddGoList", func(t *testing.T) {
		bashID, err := bashAddNotification("hello\nworld", "", "sess1", "win0", "pane0", "", "info")
		require.NoError(t, err)
		require.NotEmpty(t, bashID)

		// List via Go
		Init()
		list := ListNotifications("all", "", "", "", "", "", "")
		require.Contains(t, list, bashID)
		// Check message is correct (unescaped)
		lines := strings.Split(strings.TrimSpace(list), "\n")
		found := false
		for _, line := range lines {
			fields := strings.Split(line, "\t")
			if fields[fieldID] == bashID {
				require.Equal(t, "hello\nworld", unescapeMessage(fields[fieldMessage]))
				found = true
				break
			}
		}
		require.True(t, found, "Notification not found in Go list")
	})

	// Test 2: Add via Go, read via bash (list via bash storage_list_notifications)
	t.Run("GoAddBashList", func(t *testing.T) {
		Init()
		goID := AddNotification("test\tmessage", "", "sess2", "win1", "pane1", "", "warning")
		require.NotEmpty(t, goID)

		// Use bash to list notifications
		bashList, err := bashListNotifications("all", "", "", "", "", "", "")
		require.NoError(t, err)
		require.Contains(t, bashList, goID)
		// Parse TSV lines and find message
		lines := strings.Split(bashList, "\n")
		found := false
		for _, line := range lines {
			fields := strings.Split(line, "\t")
			if len(fields) > fieldID && fields[fieldID] == goID {
				// Bash storage returns escaped message; need to unescape
				require.Equal(t, "test\tmessage", unescapeMessage(fields[fieldMessage]))
				found = true
				break
			}
		}
		require.True(t, found, "Notification not found in bash list")
	})

	t.Run("EscapeCompatibility", func(t *testing.T) {
		testCases := []struct {
			name string
			msg  string
		}{
			{"newline", "hello\nworld"},
			{"tab", "hello\tworld"},
			{"backslash", "hello\\world"},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Bash -> Go
				bashID, err := bashAddNotification(tc.msg, "", "", "", "", "", "info")
				require.NoError(t, err)
				Init()
				list := ListNotifications("all", "", "", "", "", "", "")
				require.Contains(t, list, bashID)
				lines := strings.Split(strings.TrimSpace(list), "\n")
				for _, line := range lines {
					fields := strings.Split(line, "\t")
					if fields[fieldID] == bashID {
						require.Equal(t, tc.msg, unescapeMessage(fields[fieldMessage]))
						break
					}
				}
				// Go -> Bash
				goID := AddNotification(tc.msg, "", "", "", "", "", "info")
				require.NotEmpty(t, goID)
				bashList, err := bashListNotifications("all", "", "", "", "", "", "")
				require.NoError(t, err)
				require.Contains(t, bashList, goID)
				lines = strings.Split(bashList, "\n")
				for _, line := range lines {
					fields := strings.Split(line, "\t")
					if len(fields) > fieldID && fields[fieldID] == goID {
						require.Equal(t, tc.msg, unescapeMessage(fields[fieldMessage]))
						break
					}
				}
			})
		}
	})
}

func TestAddNotificationWithHooks(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "hooks")
	preAddDir := filepath.Join(hookDir, "pre-add")
	require.NoError(t, os.MkdirAll(preAddDir, 0755))
	postAddDir := filepath.Join(hookDir, "post-add")
	require.NoError(t, os.MkdirAll(postAddDir, 0755))

	// Create a hook script that logs its execution
	script := filepath.Join(preAddDir, "test.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho 'pre-add hook executed'"), 0755))
	script2 := filepath.Join(postAddDir, "test.sh")
	require.NoError(t, os.WriteFile(script2, []byte("#!/bin/sh\necho 'post-add hook executed'"), 0755))

	// Set environment variables
	oldHookDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldHookDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", hookDir)
	oldEnabled := os.Getenv("TMUX_INTRAY_HOOKS_ENABLED")
	defer os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", oldEnabled)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
	defer os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
	// Ensure state directory is separate
	oldStateDir := os.Getenv("TMUX_INTRAY_STATE_DIR")
	defer os.Setenv("TMUX_INTRAY_STATE_DIR", oldStateDir)
	stateDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)

	// Reset storage state
	notificationsFile = ""
	lockDir = ""
	initialized = false

	// Add notification
	id := AddNotification("test message", "", "", "", "", "", "info")
	require.NotEmpty(t, id)
	// Verify notification exists
	list := ListNotifications("active", "", "", "", "", "", "")
	require.Contains(t, list, id)
	// Note: we cannot easily capture hook output; but if hook fails with abort mode, AddNotification would return empty.
}

func TestAddNotificationHookAbort(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "hooks")
	preAddDir := filepath.Join(hookDir, "pre-add")
	require.NoError(t, os.MkdirAll(preAddDir, 0755))
	script := filepath.Join(preAddDir, "abort.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexit 1"), 0755))

	oldHookDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldHookDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", hookDir)
	oldEnabled := os.Getenv("TMUX_INTRAY_HOOKS_ENABLED")
	defer os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", oldEnabled)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
	defer os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "abort")
	oldStateDir := os.Getenv("TMUX_INTRAY_STATE_DIR")
	defer os.Setenv("TMUX_INTRAY_STATE_DIR", oldStateDir)
	stateDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)

	// Reset storage state
	notificationsFile = ""
	lockDir = ""
	initialized = false

	// Add notification should fail
	id := AddNotification("test message", "", "", "", "", "", "info")
	require.Empty(t, id)
	// Ensure no notification added
	list := ListNotifications("all", "", "", "", "", "", "")
	require.NotContains(t, list, "test message")
}
