package sqlite

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockStatusPublisher struct {
	mock.Mock
}

func (m *mockStatusPublisher) HasSession() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *mockStatusPublisher) SetStatusOption(name, value string) error {
	args := m.Called(name, value)
	return args.Error(0)
}

func newTestStorage(t *testing.T) *SQLiteStorage {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "notifications.db")
	s, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.Close())
	})

	return s
}

func writeHookScript(t *testing.T, hooksDir, hookPoint, name, body string) {
	t.Helper()
	hookPointDir := filepath.Join(hooksDir, hookPoint)
	require.NoError(t, os.MkdirAll(hookPointDir, 0o755))
	scriptPath := filepath.Join(hookPointDir, name)
	require.NoError(t, os.WriteFile(scriptPath, []byte(body), 0o755))
}

func TestSQLiteStorageImplementsInterface(t *testing.T) {
	_ = newTestStorage(t)
}

func TestAddAndGetNotification(t *testing.T) {
	s := newTestStorage(t)

	id, err := s.AddNotification("line1\nline2\tend", "2026-01-02T03:04:05Z", "s1", "w1", "p1", "", "info")
	require.NoError(t, err)
	require.Equal(t, "1", id)

	line, err := s.GetNotificationByID(id)
	require.NoError(t, err)
	require.Contains(t, line, "\tactive\t")
	require.Contains(t, line, "line1\\nline2\\tend")
}

func TestListNotificationsFilters(t *testing.T) {
	s := newTestStorage(t)

	_, err := s.AddNotification("error", "2026-01-01T01:00:00Z", "sess-a", "win-a", "pane-a", "", "error")
	require.NoError(t, err)
	_, err = s.AddNotification("warning", "2026-01-02T01:00:00Z", "sess-b", "win-b", "pane-b", "", "warning")
	require.NoError(t, err)

	list, err := s.ListNotifications("all", "error", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Contains(t, list, "\terror\t")
	require.NotContains(t, list, "\twarning\t")

	list, err = s.ListNotifications("all", "", "sess-b", "", "", "", "", "")
	require.NoError(t, err)
	require.Contains(t, list, "sess-b")
	require.NotContains(t, list, "sess-a")

	list, err = s.ListNotifications("all", "", "", "", "", "2026-01-02T00:00:00Z", "", "")
	require.NoError(t, err)
	require.Contains(t, list, "2026-01-01T01:00:00Z")
	require.NotContains(t, list, "2026-01-02T01:00:00Z")
}

func TestDismissNotificationAndDismissAll(t *testing.T) {
	s := newTestStorage(t)

	id1, err := s.AddNotification("n1", "", "", "", "", "", "info")
	require.NoError(t, err)
	id2, err := s.AddNotification("n2", "", "", "", "", "", "warning")
	require.NoError(t, err)

	require.Equal(t, 2, s.GetActiveCount())
	require.NoError(t, s.DismissNotification(id1))
	require.Equal(t, 1, s.GetActiveCount())

	err = s.DismissNotification(id1)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotificationAlreadyDismissed))

	require.NoError(t, s.DismissAll())
	require.Equal(t, 0, s.GetActiveCount())

	line, err := s.GetNotificationByID(id2)
	require.NoError(t, err)
	require.Contains(t, line, "\tdismissed\t")
}

func TestMarkReadAndUnread(t *testing.T) {
	s := newTestStorage(t)

	id, err := s.AddNotification("n", "", "", "", "", "", "info")
	require.NoError(t, err)

	require.NoError(t, s.MarkNotificationRead(id))
	line, err := s.GetNotificationByID(id)
	require.NoError(t, err)
	fields := strings.Split(line, "\t")
	require.Len(t, fields, 10)
	require.NotEmpty(t, fields[9])
	_, err = time.Parse(time.RFC3339, fields[9])
	require.NoError(t, err)

	require.NoError(t, s.MarkNotificationUnread(id))
	line, err = s.GetNotificationByID(id)
	require.NoError(t, err)
	fields = strings.Split(line, "\t")
	require.Empty(t, fields[9])
}

func TestCleanupOldNotifications(t *testing.T) {
	s := newTestStorage(t)

	idOld, err := s.AddNotification("old", "2000-01-01T00:00:00Z", "", "", "", "", "info")
	require.NoError(t, err)
	idNew, err := s.AddNotification("new", "2099-01-01T00:00:00Z", "", "", "", "", "info")
	require.NoError(t, err)

	require.NoError(t, s.DismissNotification(idOld))
	require.NoError(t, s.DismissNotification(idNew))

	require.NoError(t, s.CleanupOldNotifications(1, true))
	_, err = s.GetNotificationByID(idOld)
	require.NoError(t, err)

	require.NoError(t, s.CleanupOldNotifications(1, false))
	_, err = s.GetNotificationByID(idOld)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotificationNotFound))

	_, err = s.GetNotificationByID(idNew)
	require.NoError(t, err)
}

func TestValidationAndNotFoundErrors(t *testing.T) {
	s := newTestStorage(t)

	_, err := s.AddNotification("", "", "", "", "", "", "info")
	require.Error(t, err)
	require.Contains(t, err.Error(), "message cannot be empty")

	_, err = s.ListNotifications("pending", "", "", "", "", "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid state")

	err = s.DismissNotification("999")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotificationNotFound))

	err = s.MarkNotificationRead("not-a-number")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInvalidNotificationID))
}

func TestHooksParityForAddDismissAndCleanup(t *testing.T) {
	hooksDir := filepath.Join(t.TempDir(), "hooks")
	hookLog := filepath.Join(t.TempDir(), "hooks.log")
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", hooksDir)
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "abort")

	scriptBody := "#!/bin/sh\necho \"$HOOK_POINT:$NOTIFICATION_ID:$DELETED_COUNT\" >> \"$HOOK_LOG\"\n"
	writeHookScript(t, hooksDir, "pre-add", "01-pre-add.sh", scriptBody)
	writeHookScript(t, hooksDir, "post-add", "01-post-add.sh", scriptBody)
	writeHookScript(t, hooksDir, "pre-dismiss", "01-pre-dismiss.sh", scriptBody)
	writeHookScript(t, hooksDir, "post-dismiss", "01-post-dismiss.sh", scriptBody)
	writeHookScript(t, hooksDir, "pre-clear", "01-pre-clear.sh", scriptBody)
	writeHookScript(t, hooksDir, "cleanup", "01-cleanup.sh", scriptBody)
	writeHookScript(t, hooksDir, "post-cleanup", "01-post-cleanup.sh", scriptBody)
	t.Setenv("HOOK_LOG", hookLog)

	s := newTestStorage(t)

	id, err := s.AddNotification("old", "2000-01-01T00:00:00Z", "", "", "", "", "info")
	require.NoError(t, err)
	require.NoError(t, s.DismissNotification(id))
	require.NoError(t, s.CleanupOldNotifications(1, false))
	_, err = s.AddNotification("new", "", "", "", "", "", "warning")
	require.NoError(t, err)
	require.NoError(t, s.DismissAll())

	content, err := os.ReadFile(hookLog)
	require.NoError(t, err)
	logOutput := string(content)
	require.Contains(t, logOutput, "pre-add:1:")
	require.Contains(t, logOutput, "post-add:1:")
	require.Contains(t, logOutput, "pre-dismiss:1:")
	require.Contains(t, logOutput, "post-dismiss:1:")
	require.Contains(t, logOutput, "pre-clear::")
	require.Equal(t, 2, strings.Count(logOutput, "pre-dismiss:"))
	require.Equal(t, 2, strings.Count(logOutput, "post-dismiss:"))
	require.Contains(t, logOutput, "cleanup::")
	require.Contains(t, logOutput, "post-cleanup::1")
}

func TestTmuxStatusParityForActiveCountChanges(t *testing.T) {
	s := newTestStorage(t)

	mockClient := new(mockStatusPublisher)
	mockClient.On("HasSession").Return(true, nil)
	mockClient.On("SetStatusOption", "@tmux_intray_active_count", "1").Return(nil).Once()
	mockClient.On("SetStatusOption", "@tmux_intray_active_count", "2").Return(nil).Once()
	mockClient.On("SetStatusOption", "@tmux_intray_active_count", "1").Return(nil).Once()
	mockClient.On("SetStatusOption", "@tmux_intray_active_count", "0").Return(nil).Once()

	SetTmuxClient(mockClient)
	t.Cleanup(func() {
		SetTmuxClient(noopStatusPublisher{})
	})

	id1, err := s.AddNotification("n1", "", "", "", "", "", "info")
	require.NoError(t, err)
	id2, err := s.AddNotification("n2", "", "", "", "", "", "warning")
	require.NoError(t, err)

	require.NoError(t, s.DismissNotification(id1))
	require.NoError(t, s.DismissAll())

	mockClient.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "1")
	mockClient.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "2")
	mockClient.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "1")
	mockClient.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "0")
	mockClient.AssertNumberOfCalls(t, "SetStatusOption", 4)
	mockClient.AssertNumberOfCalls(t, "HasSession", 4)

	line, err := s.GetNotificationByID(id2)
	require.NoError(t, err)
	require.Contains(t, line, "\tdismissed\t")
}

func TestDismissByFilter(t *testing.T) {
	s := newTestStorage(t)

	// Add notifications with different session/window/pane combinations
	_, err := s.AddNotification("n1", "", "sess1", "win1", "pane1", "", "info")
	require.NoError(t, err)
	_, err = s.AddNotification("n2", "", "sess1", "win1", "pane2", "", "warning")
	require.NoError(t, err)
	_, err = s.AddNotification("n3", "", "sess1", "win2", "pane1", "", "error")
	require.NoError(t, err)
	_, err = s.AddNotification("n4", "", "sess2", "win1", "pane1", "", "info")
	require.NoError(t, err)

	// Verify all are active
	require.Equal(t, 4, s.GetActiveCount())

	// Dismiss by session filter
	err = s.DismissByFilter("sess1", "", "")
	require.NoError(t, err)
	require.Equal(t, 1, s.GetActiveCount())

	// Verify sess1 notifications are dismissed
	list, err := s.ListNotifications("all", "", "sess1", "", "", "", "", "")
	require.NoError(t, err)
	lines := strings.Split(list, "\n")
	// Count non-empty lines
	notificationLines := 0
	for _, line := range lines {
		if line != "" {
			notificationLines++
			require.Contains(t, line, "\tdismissed\t")
		}
	}
	require.Equal(t, 3, notificationLines) // 3 notifications from sess1

	// Verify remaining notifications count
	activeCount := 0
	list, err = s.ListNotifications("active", "", "", "", "", "", "", "")
	require.NoError(t, err)
	lines = strings.Split(list, "\n")
	for _, line := range lines {
		if line != "" {
			activeCount++
			require.Contains(t, line, "\tactive\t")
			// Should be from sess2 (the one not dismissed)
			require.Contains(t, line, "sess2")
		}
	}
	require.Equal(t, 1, activeCount)
}

func TestDismissByFilterWithWindow(t *testing.T) {
	s := newTestStorage(t)

	// Add notifications with different windows
	_, err := s.AddNotification("n1", "", "sess1", "win1", "pane1", "", "info")
	require.NoError(t, err)
	_, err = s.AddNotification("n2", "", "sess1", "win1", "pane2", "", "warning")
	require.NoError(t, err)
	_, err = s.AddNotification("n3", "", "sess1", "win2", "pane1", "", "error")
	require.NoError(t, err)

	// Dismiss by window filter (within session)
	err = s.DismissByFilter("sess1", "win1", "")
	require.NoError(t, err)
	require.Equal(t, 1, s.GetActiveCount())

	// Verify win1 notifications are dismissed, win2 is still active
	list, err := s.ListNotifications("all", "", "sess1", "", "", "", "", "")
	require.NoError(t, err)
	lines := strings.Split(list, "\n")
	dismissedCount := 0
	activeCount := 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.Contains(line, "\tdismissed\t") {
			dismissedCount++
		}
		if strings.Contains(line, "\tactive\t") {
			activeCount++
		}
	}
	require.Equal(t, 2, dismissedCount)
	require.Equal(t, 1, activeCount)
}

func TestDismissByFilterWithPane(t *testing.T) {
	s := newTestStorage(t)

	// Add notifications with different panes
	_, err := s.AddNotification("n1", "", "sess1", "win1", "pane1", "", "info")
	require.NoError(t, err)
	_, err = s.AddNotification("n2", "", "sess1", "win1", "pane2", "", "warning")
	require.NoError(t, err)
	_, err = s.AddNotification("n3", "", "sess1", "win1", "pane3", "", "error")
	require.NoError(t, err)

	// Dismiss by pane filter (within session and window)
	err = s.DismissByFilter("sess1", "win1", "pane1")
	require.NoError(t, err)
	require.Equal(t, 2, s.GetActiveCount())

	// Verify pane1 notification is dismissed, others are still active
	list, err := s.ListNotifications("all", "", "sess1", "win1", "", "", "", "")
	require.NoError(t, err)
	lines := strings.Split(list, "\n")
	dismissedCount := 0
	activeCount := 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.Contains(line, "\tdismissed\t") {
			dismissedCount++
		}
		if strings.Contains(line, "\tactive\t") {
			activeCount++
		}
	}
	require.Equal(t, 1, dismissedCount)
	require.Equal(t, 2, activeCount)
}

func TestDismissByFilterWithEmptyFilters(t *testing.T) {
	s := newTestStorage(t)

	// Add notifications
	_, err := s.AddNotification("n1", "", "sess1", "win1", "pane1", "", "info")
	require.NoError(t, err)
	_, err = s.AddNotification("n2", "", "sess2", "win2", "pane2", "", "warning")
	require.NoError(t, err)

	// Dismiss with empty filters (should dismiss all active)
	err = s.DismissByFilter("", "", "")
	require.NoError(t, err)
	require.Equal(t, 0, s.GetActiveCount())

	// Verify all are dismissed
	list, err := s.ListNotifications("all", "", "", "", "", "", "", "")
	require.NoError(t, err)
	lines := strings.Split(list, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		require.Contains(t, line, "\tdismissed\t")
	}
}

func TestDismissByFilterWithNoMatches(t *testing.T) {
	s := newTestStorage(t)

	// Add notifications
	_, err := s.AddNotification("n1", "", "sess1", "win1", "pane1", "", "info")
	require.NoError(t, err)
	initialCount := s.GetActiveCount()

	// Dismiss with non-matching filters
	err = s.DismissByFilter("nonexistent", "", "")
	require.NoError(t, err)
	require.Equal(t, initialCount, s.GetActiveCount())
}

func TestDismissByFilterRunsHooks(t *testing.T) {
	hooksDir := filepath.Join(t.TempDir(), "hooks")
	hookLog := filepath.Join(t.TempDir(), "hooks.log")
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", hooksDir)
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "abort")

	scriptBody := "#!/bin/sh\necho \"$HOOK_POINT:$NOTIFICATION_ID\" >> \"$HOOK_LOG\"\n"
	writeHookScript(t, hooksDir, "pre-dismiss", "01-pre-dismiss.sh", scriptBody)
	writeHookScript(t, hooksDir, "post-dismiss", "01-post-dismiss.sh", scriptBody)
	t.Setenv("HOOK_LOG", hookLog)

	s := newTestStorage(t)

	// Add notifications with same session
	id1, err := s.AddNotification("n1", "", "sess1", "win1", "pane1", "", "info")
	require.NoError(t, err)
	id2, err := s.AddNotification("n2", "", "sess1", "win1", "pane2", "", "warning")
	require.NoError(t, err)

	// Dismiss by session filter
	err = s.DismissByFilter("sess1", "", "")
	require.NoError(t, err)

	// Verify hooks were called for both notifications
	content, err := os.ReadFile(hookLog)
	require.NoError(t, err)
	logOutput := string(content)
	require.Contains(t, logOutput, "pre-dismiss:"+id1)
	require.Contains(t, logOutput, "post-dismiss:"+id1)
	require.Contains(t, logOutput, "pre-dismiss:"+id2)
	require.Contains(t, logOutput, "post-dismiss:"+id2)
}
