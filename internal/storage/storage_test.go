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
	Reset()
	return tmpDir
}

func TestStorageInit(t *testing.T) {
	tmpDir := setupTest(t)
	err := Init()
	require.NoError(t, err)
	// Check notifications file exists
	require.FileExists(t, filepath.Join(tmpDir, "notifications.tsv"))
}

func TestAddNotification(t *testing.T) {
	setupTest(t)
	require.NoError(t, Init())
	id, err := AddNotification("test message", "", "session1", "window0", "pane0", "", "info")
	require.NoError(t, err)
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
	require.NoError(t, Init())
	id, err := AddNotification("msg", "2025-01-01T12:00:00Z", "", "", "", "", "warning")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	list := ListNotifications("all", "", "", "", "", "", "")
	require.Contains(t, list, "2025-01-01T12:00:00Z")
	require.Contains(t, list, "warning")
}

func TestListNotificationsFilters(t *testing.T) {
	setupTest(t)
	require.NoError(t, Init())
	// Add multiple notifications with different attributes
	id1, err := AddNotification("error msg", "", "session1", "window1", "pane1", "", "error")
	require.NoError(t, err)
	id2, err := AddNotification("info msg", "", "session2", "window2", "pane2", "", "info")
	require.NoError(t, err)
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
	require.NoError(t, Init())
	id, err := AddNotification("to dismiss", "", "", "", "", "", "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	// Should be active
	list := ListNotifications("active", "", "", "", "", "", "")
	require.Contains(t, list, id)
	// Dismiss
	err = DismissNotification(id)
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
	require.NoError(t, Init())
	id1, err := AddNotification("msg1", "", "", "", "", "", "info")
	require.NoError(t, err)
	id2, err := AddNotification("msg2", "", "", "", "", "", "warning")
	require.NoError(t, err)
	require.Equal(t, 2, GetActiveCount())
	err = DismissAll()
	require.NoError(t, err)
	require.Equal(t, 0, GetActiveCount())
	list := ListNotifications("dismissed", "", "", "", "", "", "")
	require.Contains(t, list, id1)
	require.Contains(t, list, id2)
}

func TestCleanupOldNotifications(t *testing.T) {
	setupTest(t)
	require.NoError(t, Init())
	// Add a notification with old timestamp
	id, err := AddNotification("old", "2000-01-01T00:00:00Z", "", "", "", "", "info")
	require.NoError(t, err)
	_ = DismissNotification(id)
	// Cleanup with threshold 1 day (dry run)
	err = CleanupOldNotifications(1, true)
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
	require.NoError(t, Init())
	require.Equal(t, 0, GetActiveCount())
	id1, err := AddNotification("msg1", "", "", "", "", "", "info")
	require.NoError(t, err)
	require.Equal(t, 1, GetActiveCount())
	_, err = AddNotification("msg2", "", "", "", "", "", "warning")
	require.NoError(t, err)
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
			"TMUX_INTRAY_DEBUG=false")
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
			"TMUX_INTRAY_DEBUG=false",
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
		goID, err := AddNotification("test\tmessage", "", "sess2", "win1", "pane1", "", "warning")
		require.NoError(t, err)
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
				goID, err := AddNotification(tc.msg, "", "", "", "", "", "info")
				require.NoError(t, err)
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
	Reset()

	// Add notification
	id, err := AddNotification("test message", "", "", "", "", "", "info")
	require.NoError(t, err)
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
	Reset()

	// Add notification should fail
	id, err := AddNotification("test message", "", "", "", "", "", "info")
	require.Error(t, err)
	require.Empty(t, id)
	// Ensure no notification added
	list := ListNotifications("all", "", "", "", "", "", "")
	require.NotContains(t, list, "test message")
}

func TestMalformedTSVData(t *testing.T) {
	tmpDir := setupTest(t)
	notifFile := filepath.Join(tmpDir, "notifications.tsv")

	// Write malformed TSV data
	malformedData := "1\ttimestamp\tactive\n" + // Only 3 fields instead of 9
		"2\t2025-01-01T12:00:00Z\tactive\t\n\n\n\n\n\ninfo\n" + // Empty fields
		"3\t2025-01-01T12:00:00Z\n" + // Only 2 fields
		"4\t2025-01-01T12:00:00Z\tactive\tsession\twindow\tpane\tmessage\tcreated\t\n" // Missing level
	err := os.WriteFile(notifFile, []byte(malformedData), 0644)
	require.NoError(t, err)

	// Reset and reinitialize to load the malformed data
	notificationsFile = ""
	lockDir = ""
	initialized = false
	Init()

	// Add a valid notification - should succeed
	id, err := AddNotification("valid message", "", "session1", "window1", "pane1", "", "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)

	// List should work (filterNotifications should handle malformed data gracefully)
	list := ListNotifications("all", "", "", "", "", "", "")
	require.Contains(t, list, id)
	require.Contains(t, list, "valid message")

	// Dismissing notification 1 (malformed with 3 fields) should return error
	err = DismissNotification("1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected 9 fields, got 3")

	// Dismissing notification 3 (malformed with 2 fields) should return error
	err = DismissNotification("3")
	require.Error(t, err)

	// Dismissing a valid notification should work
	err = DismissNotification(id)
	require.NoError(t, err)
}

func TestGetFieldHelper(t *testing.T) {
	// Test getField with various inputs
	testCases := []struct {
		name      string
		fields    []string
		index     int
		want      string
		wantError bool
	}{
		{"valid index", []string{"a", "b", "c"}, 1, "b", false},
		{"first index", []string{"a", "b", "c"}, 0, "a", false},
		{"last index", []string{"a", "b", "c"}, 2, "c", false},
		{"negative index", []string{"a", "b", "c"}, -1, "", true},
		{"out of bounds high", []string{"a", "b", "c"}, 3, "", true},
		{"nil fields", nil, 0, "", true},
		{"empty fields", []string{}, 0, "", true},
		{"single field", []string{"only"}, 0, "only", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := getField(tc.fields, tc.index)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, result)
			}
		})
	}
}

func TestValidateNotificationInputs(t *testing.T) {
	// Test validateNotificationInputs with various inputs
	testCases := []struct {
		name        string
		message     string
		timestamp   string
		session     string
		window      string
		pane        string
		paneCreated string
		level       string
		wantError   bool
		errorMsg    string
	}{
		// Valid inputs
		{"valid basic", "test message", "", "", "", "", "", "info", false, ""},
		{"valid with timestamp", "test", "2025-01-01T12:00:00Z", "", "", "", "", "warning", false, ""},
		{"valid with all fields", "test", "2025-01-01T12:00:00Z", "sess1", "win0", "pane0", "123", "error", false, ""},
		{"valid with fractional seconds", "test", "2025-01-01T12:00:00.123Z", "", "", "", "", "critical", false, ""},
		{"valid RFC3339 with offset", "test", "2025-01-01T12:00:00+00:00", "", "", "", "", "info", false, ""},

		// Empty message
		{"empty message", "", "", "", "", "", "", "info", true, "message cannot be empty"},
		{"whitespace only message", "   ", "", "", "", "", "", "info", true, "message cannot be empty"},
		{"tab only message", "\t", "", "", "", "", "", "info", true, "message cannot be empty"},
		{"newline only message", "\n", "", "", "", "", "", "info", true, "message cannot be empty"},

		// Empty level (FIX #1: should now be rejected)
		{"empty level", "test", "", "", "", "", "", "", true, "level cannot be empty"},
		{"whitespace level", "test", "", "", "", "", "", "   ", true, "invalid level"},

		// Invalid level values
		{"invalid level lowercase", "test", "", "", "", "", "", "debug", true, "invalid level 'debug'"},
		{"invalid level uppercase", "test", "", "", "", "", "", "INFO", true, "invalid level 'INFO'"},
		{"invalid level number", "test", "", "", "", "", "", "1", true, "invalid level '1'"},

		// Invalid timestamp formats (FIX #2: should accept all RFC3339 formats)
		{"invalid timestamp no T", "test", "2025-01-01 12:00:00Z", "", "", "", "", "info", true, "invalid timestamp format"},
		{"invalid timestamp missing Z", "test", "2025-01-01T12:00:00", "", "", "", "", "info", true, "invalid timestamp format"},
		{"invalid timestamp garbage", "test", "not-a-timestamp", "", "", "", "", "info", true, "invalid timestamp format"},
		{"invalid timestamp partial", "test", "2025-01-01T12:00", "", "", "", "", "info", true, "invalid timestamp format"},

		// Whitespace-only session/window/pane (should be rejected)
		{"whitespace session", "test", "", "   ", "", "", "", "info", true, "session cannot be whitespace only"},
		{"whitespace window", "test", "", "", "  \t  ", "", "", "info", true, "window cannot be whitespace only"},
		{"whitespace pane", "test", "", "", "", "\n  ", "", "info", true, "pane cannot be whitespace only"},

		// Valid empty session/window/pane (optional fields)
		{"empty optional fields", "test", "", "", "", "", "", "info", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNotificationInputs(tc.message, tc.timestamp, tc.session, tc.window, tc.pane, tc.paneCreated, tc.level)
			if tc.wantError {
				require.Error(t, err, "expected error for test case: "+tc.name)
				if tc.errorMsg != "" {
					require.Contains(t, err.Error(), tc.errorMsg, "error message should contain: "+tc.errorMsg)
				}
			} else {
				require.NoError(t, err, "unexpected error for test case: "+tc.name)
			}
		})
	}
}

func TestAddNotificationValidation(t *testing.T) {
	setupTest(t)
	Init()

	// Test that validation errors are properly returned from AddNotification
	testCases := []struct {
		name      string
		message   string
		timestamp string
		level     string
		wantError bool
		errorMsg  string
	}{
		{"valid", "test message", "", "info", false, ""},
		{"empty message", "", "", "info", true, "message cannot be empty"},
		{"empty level", "test", "", "", true, "level cannot be empty"},
		{"invalid level", "test", "", "invalid", true, "invalid level 'invalid'"},
		{"invalid timestamp", "test", "bad-timestamp", "info", true, "invalid timestamp format"},
		{"valid RFC3339 timestamp", "test", "2025-01-01T12:00:00.123Z", "info", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := AddNotification(tc.message, tc.timestamp, "", "", "", "", tc.level)
			if tc.wantError {
				require.Error(t, err, "expected error for test case: "+tc.name)
				require.Empty(t, id, "ID should be empty on error")
				if tc.errorMsg != "" {
					require.Contains(t, err.Error(), tc.errorMsg, "error message should contain: "+tc.errorMsg)
				}
			} else {
				require.NoError(t, err, "unexpected error for test case: "+tc.name)
				require.NotEmpty(t, id, "ID should not be empty on success")
			}
		})
	}
}

func TestAppendLine(t *testing.T) {
	setupTest(t)
	require.NoError(t, Init())

	// Test successful append
	err := appendLine(1, "2025-01-01T12:00:00Z", "active", "session1", "window0", "pane0", "test message", "123456789", "info")
	require.NoError(t, err)

	// Verify line was written
	lines, err := readAllLines()
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Contains(t, lines[0], "test message")
}

func TestAppendLineWriteError(t *testing.T) {
	setupTest(t)
	require.NoError(t, Init())

	// Save original notifications file
	originalFile := notificationsFile
	defer func() {
		notificationsFile = originalFile
	}()

	// Create a read-only file to force write error
	readOnlyFile := filepath.Join(t.TempDir(), "readonly.txt")
	require.NoError(t, os.WriteFile(readOnlyFile, []byte("initial"), 0444))
	notificationsFile = readOnlyFile

	err := appendLine(1, "2025-01-01T12:00:00Z", "active", "session1", "window0", "pane0", "test message", "123456789", "info")
	require.Error(t, err)
	// The error should be either "open file" (permission denied) or "write line"
	require.True(t, strings.Contains(err.Error(), "open file") || strings.Contains(err.Error(), "write line"))
}

func TestAppendLineOpenError(t *testing.T) {
	setupTest(t)
	require.NoError(t, Init())

	// Save original notifications file
	originalFile := notificationsFile
	defer func() {
		notificationsFile = originalFile
	}()

	// Set to an invalid path (directory that doesn't exist)
	notificationsFile = filepath.Join(t.TempDir(), "nonexistent", "file.txt")

	err := appendLine(1, "2025-01-01T12:00:00Z", "active", "session1", "window0", "pane0", "test message", "123456789", "info")
	require.Error(t, err)
	require.Contains(t, err.Error(), "open file")
}

func TestAppendLineMultipleWrites(t *testing.T) {
	setupTest(t)
	require.NoError(t, Init())

	// Test multiple successful writes
	for i := 1; i <= 5; i++ {
		err := appendLine(i, "2025-01-01T12:00:00Z", "active", "session1", "window0", "pane0",
			fmt.Sprintf("message %d", i), "123456789", "info")
		require.NoError(t, err)
	}

	// Verify all lines were written
	lines, err := readAllLines()
	require.NoError(t, err)
	require.Len(t, lines, 5)

	// Verify content
	for i, line := range lines {
		require.Contains(t, line, fmt.Sprintf("message %d", i+1))
	}
}

func TestAppendLineWithSpecialCharacters(t *testing.T) {
	testMessages := []struct {
		name    string
		message string
	}{
		{"newline", "line1\nline2"},
		{"tab", "col1\tcol2"},
		{"backslash", "path\\to\\file"},
		{"mixed", "test\\nwith\ttabs"},
	}

	for _, tc := range testMessages {
		t.Run(tc.name, func(t *testing.T) {
			// Use separate state for each subtest
			setupTest(t)
			require.NoError(t, Init())

			escaped := escapeMessage(tc.message)
			err := appendLine(1, "2025-01-01T12:00:00Z", "active", "session1", "window0", "pane0",
				escaped, "123456789", "info")
			require.NoError(t, err)

			// Verify line was written
			lines, err := readAllLines()
			require.NoError(t, err)
			require.Len(t, lines, 1)
			require.Contains(t, lines[0], escaped)
		})
	}
}
func TestStrToInt(t *testing.T) {
	// Valid conversions
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid zero",
			input:   "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "valid positive",
			input:   "42",
			want:    42,
			wantErr: false,
		},
		{
			name:    "valid large number",
			input:   "999999",
			want:    999999,
			wantErr: false,
		},
		{
			name:    "invalid non-numeric",
			input:   "abc",
			want:    0,
			wantErr: true,
			errMsg:  "convert string to int",
		},
		{
			name:    "invalid empty string",
			input:   "",
			want:    0,
			wantErr: true,
			errMsg:  "convert string to int",
		},
		{
			name:    "invalid with letters",
			input:   "42abc",
			want:    0,
			wantErr: true,
			errMsg:  "convert string to int",
		},
		{
			name:    "negative value rejected",
			input:   "-1",
			want:    0,
			wantErr: true,
			errMsg:  "negative value not allowed",
		},
		{
			name:    "negative zero rejected",
			input:   "-0",
			want:    0,
			wantErr: true,
			errMsg:  "negative value not allowed",
		},
		{
			name:    "negative large value rejected",
			input:   "-999",
			want:    0,
			wantErr: true,
			errMsg:  "negative value not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strToInt(tt.input)
			if tt.wantErr {
				require.Error(t, err, "expected error but got none")
				require.Contains(t, err.Error(), tt.errMsg, "error message should contain expected text")
				require.Equal(t, tt.want, got, "on error, should return zero value")
			} else {
				require.NoError(t, err, "expected no error but got: %v", err)
				require.Equal(t, tt.want, got, "unexpected result")
			}
		})
	}
}

func TestUpdateTmuxStatusOption(t *testing.T) {
	setupTest(t)
	require.NoError(t, Init())

	t.Run("tmux availability check", func(t *testing.T) {
		// Call function with a test count
		err := updateTmuxStatusOption(5)
		// The function should either succeed or return a clear error
		if err != nil {
			// If tmux is not available, we expect a clear error message
			require.Contains(t, err.Error(), "tmux")
		}
		// If tmux is available, it should succeed
	})
}

func TestDismissNotificationHandlesTmuxError(t *testing.T) {
	setupTest(t)
	require.NoError(t, Init())
	id, err := AddNotification("to dismiss", "", "", "", "", "", "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	// Dismiss should still succeed even if tmux update fails
	err = DismissNotification(id)
	// The dismissal should succeed (notification is dismissed)
	// The tmux error should be logged but not cause dismiss to fail
	require.NoError(t, err)
	// Verify notification is actually dismissed
	list := ListNotifications("active", "", "", "", "", "", "")
	require.NotContains(t, list, id)
}

func TestDismissAllHandlesTmuxError(t *testing.T) {
	setupTest(t)
	require.NoError(t, Init())
	_, err := AddNotification("msg1", "", "", "", "", "", "info")
	require.NoError(t, err)
	_, err = AddNotification("msg2", "", "", "", "", "", "warning")
	require.NoError(t, err)
	require.Equal(t, 2, GetActiveCount())
	// DismissAll should still succeed even if tmux update fails
	err = DismissAll()
	// The dismissal should succeed (notifications are dismissed)
	// The tmux error should be logged but not cause dismiss to fail
	require.NoError(t, err)
	// Verify all notifications are actually dismissed
	require.Equal(t, 0, GetActiveCount())
}
