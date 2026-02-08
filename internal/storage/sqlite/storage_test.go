package sqlite

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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

	list, err := s.ListNotifications("all", "error", "", "", "", "", "")
	require.NoError(t, err)
	require.Contains(t, list, "\terror\t")
	require.NotContains(t, list, "\twarning\t")

	list, err = s.ListNotifications("all", "", "sess-b", "", "", "", "")
	require.NoError(t, err)
	require.Contains(t, list, "sess-b")
	require.NotContains(t, list, "sess-a")

	list, err = s.ListNotifications("all", "", "", "", "", "2026-01-02T00:00:00Z", "")
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

	_, err = s.ListNotifications("pending", "", "", "", "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid state")

	err = s.DismissNotification("999")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotificationNotFound))

	err = s.MarkNotificationRead("not-a-number")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInvalidNotificationID))
}
