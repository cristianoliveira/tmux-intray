package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sqlitebackend "github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorageOperationParityAgainstTSV(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	tsvStore, tsvMock := newTSVBackendForIntegration(t)
	sqliteStore, sqliteMock := newSQLiteBackendForIntegration(t)

	idsTSV := addFixtureNotifications(t, tsvStore)
	idsSQLite := addFixtureNotifications(t, sqliteStore)
	require.Equal(t, idsTSV, idsSQLite)

	listTSV, err := tsvStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	listSQLite, err := sqliteStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(listTSV), normalizeTSVOutput(listSQLite))

	getTSV, err := tsvStore.GetNotificationByID(idsTSV[0])
	require.NoError(t, err)
	getSQLite, err := sqliteStore.GetNotificationByID(idsSQLite[0])
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(getTSV), normalizeTSVOutput(getSQLite))

	require.NoError(t, tsvStore.MarkNotificationRead(idsTSV[1]))
	require.NoError(t, sqliteStore.MarkNotificationRead(idsSQLite[1]))
	readTSV, err := tsvStore.GetNotificationByID(idsTSV[1])
	require.NoError(t, err)
	readSQLite, err := sqliteStore.GetNotificationByID(idsSQLite[1])
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(readTSV), normalizeTSVOutput(readSQLite))

	require.NoError(t, tsvStore.MarkNotificationUnread(idsTSV[1]))
	require.NoError(t, sqliteStore.MarkNotificationUnread(idsSQLite[1]))
	unreadTSV, err := tsvStore.GetNotificationByID(idsTSV[1])
	require.NoError(t, err)
	unreadSQLite, err := sqliteStore.GetNotificationByID(idsSQLite[1])
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(unreadTSV), normalizeTSVOutput(unreadSQLite))

	require.NoError(t, tsvStore.DismissNotification(idsTSV[0]))
	require.NoError(t, sqliteStore.DismissNotification(idsSQLite[0]))
	require.Equal(t, tsvStore.GetActiveCount(), sqliteStore.GetActiveCount())

	require.NoError(t, tsvStore.DismissAll())
	require.NoError(t, sqliteStore.DismissAll())
	require.Equal(t, 0, tsvStore.GetActiveCount())
	require.Equal(t, 0, sqliteStore.GetActiveCount())

	require.NoError(t, tsvStore.CleanupOldNotifications(1, true))
	require.NoError(t, sqliteStore.CleanupOldNotifications(1, true))

	preCleanupTSV, err := tsvStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	preCleanupSQLite, err := sqliteStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(preCleanupTSV), normalizeTSVOutput(preCleanupSQLite))

	require.NoError(t, tsvStore.CleanupOldNotifications(1, false))
	require.NoError(t, sqliteStore.CleanupOldNotifications(1, false))

	postCleanupTSV, err := tsvStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	postCleanupSQLite, err := sqliteStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, normalizeTSVOutput(postCleanupTSV), normalizeTSVOutput(postCleanupSQLite))
	require.Contains(t, postCleanupSQLite, "2099-01-01T00:00:00Z")
	require.NotContains(t, postCleanupSQLite, "2000-01-01T00:00:00Z")

	tsvMock.AssertNumberOfCalls(t, "SetStatusOption", 5)
	sqliteMock.AssertNumberOfCalls(t, "SetStatusOption", 5)
}

func TestSQLiteStorageMigrationIntegration(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	baseDir := t.TempDir()
	tsvPath := filepath.Join(baseDir, "notifications.tsv")
	sqlitePath := filepath.Join(baseDir, "notifications.db")

	data := strings.Join([]string{
		"1\t2026-01-01T00:00:00Z\tactive\tsession-a\twindow-a\tpane-a\tfirst\\nline\t\tinfo\t",
		"1\t2026-01-02T00:00:00Z\tdismissed\tsession-a\twindow-a\tpane-a\tfirst updated\t\terror\t2026-01-02T00:00:01Z",
		"2\t2026-01-03T00:00:00Z\tactive\tsession-b\twindow-b\tpane-b\tsecond\t\twarning\t",
		"invalid\t2026-01-03T00:00:00Z\tactive\tsession\twindow\tpane\tbad row\t\tinfo\t",
	}, "\n") + "\n"
	require.NoError(t, os.WriteFile(tsvPath, []byte(data), 0o644))

	statsDryRun, err := sqlitebackend.MigrateTSVToSQLite(sqlitebackend.MigrationOptions{
		TSVPath:    tsvPath,
		SQLitePath: sqlitePath,
		DryRun:     true,
	})
	require.NoError(t, err)
	require.Equal(t, 4, statsDryRun.TotalRows)
	require.Equal(t, 1, statsDryRun.SkippedRows)
	require.Equal(t, 1, statsDryRun.DuplicateRows)
	require.Equal(t, 2, statsDryRun.MigratedRows)

	stats, err := sqlitebackend.MigrateTSVToSQLite(sqlitebackend.MigrationOptions{
		TSVPath:    tsvPath,
		SQLitePath: sqlitePath,
	})
	require.NoError(t, err)
	require.True(t, stats.BackupCreated)
	require.FileExists(t, stats.BackupPath)

	store, err := sqlitebackend.NewSQLiteStorage(sqlitePath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	lineOne, err := store.GetNotificationByID("1")
	require.NoError(t, err)
	require.Contains(t, lineOne, "\tdismissed\t")
	require.Contains(t, lineOne, "first updated")
	require.Contains(t, lineOne, "\terror\t")

	lineTwo, err := store.GetNotificationByID("2")
	require.NoError(t, err)
	require.Contains(t, lineTwo, "\tactive\t")
	require.Contains(t, lineTwo, "\twarning\t")
	count := store.GetActiveCount()
	require.Equal(t, 1, count)
}

func TestSQLiteStorageIntegrationHooksAndTmux(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "abort")
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")

	hooksDir := filepath.Join(t.TempDir(), "hooks")
	hookLog := filepath.Join(t.TempDir(), "hooks.log")
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", hooksDir)
	t.Setenv("HOOK_LOG", hookLog)

	hookBody := "#!/bin/sh\necho \"$HOOK_POINT:$NOTIFICATION_ID:$DELETED_COUNT\" >> \"$HOOK_LOG\"\n"
	writeIntegrationHook(t, hooksDir, "pre-add", "pre-add.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "post-add", "post-add.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "pre-dismiss", "pre-dismiss.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "post-dismiss", "post-dismiss.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "pre-clear", "pre-clear.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "cleanup", "cleanup.sh", hookBody)
	writeIntegrationHook(t, hooksDir, "post-cleanup", "post-cleanup.sh", hookBody)

	store, tmuxMock := newSQLiteBackendForIntegration(t)

	id1, err := store.AddNotification("old", "2000-01-01T00:00:00Z", "", "", "", "", "info")
	require.NoError(t, err)
	_, err = store.AddNotification("new", "2099-01-01T00:00:00Z", "", "", "", "", "warning")
	require.NoError(t, err)
	require.NoError(t, store.DismissNotification(id1))
	require.NoError(t, store.DismissAll())
	require.NoError(t, store.CleanupOldNotifications(1, false))

	content, err := os.ReadFile(hookLog)
	require.NoError(t, err)
	logOutput := string(content)
	require.Contains(t, logOutput, "pre-add:1:")
	require.Contains(t, logOutput, "post-add:1:")
	require.Contains(t, logOutput, "pre-dismiss:1:")
	require.Contains(t, logOutput, "post-dismiss:1:")
	require.Contains(t, logOutput, "pre-clear::")
	require.Contains(t, logOutput, "cleanup::")
	require.Contains(t, logOutput, "post-cleanup::1")

	tmuxMock.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "1")
	tmuxMock.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "2")
	tmuxMock.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "1")
	tmuxMock.AssertCalled(t, "SetStatusOption", "@tmux_intray_active_count", "0")
}

func addFixtureNotifications(t *testing.T, s Storage) []string {
	t.Helper()

	fixtures := []struct {
		message     string
		timestamp   string
		session     string
		window      string
		pane        string
		paneCreated string
		level       string
	}{
		{"first line\nwith tab\tvalue", "2000-01-01T00:00:00Z", "session-a", "window-1", "pane-1", "", "info"},
		{"second", "2026-01-03T04:05:06Z", "session-b", "window-2", "pane-2", "", "warning"},
		{"third", "2099-01-01T00:00:00Z", "session-c", "window-3", "pane-3", "", "error"},
	}

	ids := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		id, err := s.AddNotification(
			fixture.message,
			fixture.timestamp,
			fixture.session,
			fixture.window,
			fixture.pane,
			fixture.paneCreated,
			fixture.level,
		)
		require.NoError(t, err)
		ids = append(ids, id)
	}

	return ids
}

func newTSVBackendForIntegration(t *testing.T) (Storage, *tmux.MockClient) {
	t.Helper()

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	Reset()
	t.Cleanup(Reset)

	tmuxMock := new(tmux.MockClient)
	tmuxMock.On("HasSession").Return(true, nil)
	tmuxMock.On("SetStatusOption", "@tmux_intray_active_count", mock.Anything).Return(nil)
	SetTmuxClient(tmuxMock)

	store, err := NewFileStorage()
	require.NoError(t, err)
	return store, tmuxMock
}

func newSQLiteBackendForIntegration(t *testing.T) (Storage, *tmux.MockClient) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "notifications.db")
	store, err := sqlitebackend.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	tmuxMock := new(tmux.MockClient)
	tmuxMock.On("HasSession").Return(true, nil)
	tmuxMock.On("SetStatusOption", "@tmux_intray_active_count", mock.Anything).Return(nil)
	sqlitebackend.SetTmuxClient(tmuxMock)
	t.Cleanup(func() {
		sqlitebackend.SetTmuxClient(tmux.NewDefaultClient())
	})

	return store, tmuxMock
}

func normalizeTSVOutput(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(content), "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		for len(parts) < 10 {
			parts = append(parts, "")
		}
		if parts[9] != "" {
			parts[9] = "<read_timestamp>"
		}
		normalized = append(normalized, strings.Join(parts, "\t"))
	}
	return strings.Join(normalized, "\n")
}

func writeIntegrationHook(t *testing.T, hooksDir, hookPoint, name, body string) {
	t.Helper()
	hookPointDir := filepath.Join(hooksDir, hookPoint)
	require.NoError(t, os.MkdirAll(hookPointDir, 0o755))
	hookPath := filepath.Join(hookPointDir, name)
	require.NoError(t, os.WriteFile(hookPath, []byte(body), 0o755))
}

func TestSQLiteStorageLargeDatasetIntegration(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	store, _ := newSQLiteBackendForIntegration(t)

	for i := 0; i < 1000; i++ {
		_, err := store.AddNotification(
			fmt.Sprintf("message-%d", i),
			fmt.Sprintf("2026-01-01T00:00:%02dZ", i%60),
			"load-session",
			"load-window",
			"load-pane",
			"",
			"info",
		)
		require.NoError(t, err)
	}

	require.Equal(t, 1000, store.GetActiveCount())

	list, err := store.ListNotifications("active", "info", "load-session", "", "", "", "")
	require.NoError(t, err)
	require.Len(t, strings.Split(strings.TrimSpace(list), "\n"), 1000)
}

func TestDualWriterIntegrationConsistency(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	tsvStore, _ := newTSVBackendForIntegration(t)
	sqliteStore, _ := newSQLiteBackendForIntegration(t)

	dw, err := NewDualWriter(tsvStore, sqliteStore, DualWriterOptions{
		ReadFromSQLite:        true,
		ConsistencySampleSize: 10,
		VerifyEveryNWrites:    1,
	})
	require.NoError(t, err)

	id, err := dw.AddNotification("dual writer", "2026-01-01T00:00:00Z", "s", "w", "p", "", "info")
	require.NoError(t, err)
	require.Equal(t, "1", id)

	require.NoError(t, dw.MarkNotificationRead(id))
	require.NoError(t, dw.MarkNotificationUnread(id))
	require.NoError(t, dw.DismissNotification(id))

	report, err := dw.VerifyConsistency()
	require.NoError(t, err)
	require.False(t, report.HasCriticalDifferences())
	require.Equal(t, report.TSVRecordCount, report.SQLiteRecordCount)
}
