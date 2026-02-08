package storage

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDualWriterVerifyConsistencyIntegration(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	tsvStore, _ := newTSVBackendForIntegration(t)
	sqliteStore, _ := newSQLiteBackendForIntegration(t)

	dw, err := NewDualWriterWithBackends(tsvStore, sqliteStore, DualWriterOptions{SampleSize: 50})
	require.NoError(t, err)

	id1, err := dw.AddNotification("first", "2026-01-01T00:00:00Z", "s", "w", "p", "", "info")
	require.NoError(t, err)
	id2, err := dw.AddNotification("second", "2026-01-02T00:00:00Z", "s", "w", "p", "", "warning")
	require.NoError(t, err)
	require.NoError(t, dw.MarkNotificationRead(id2))
	require.NoError(t, dw.DismissNotification(id1))

	report, err := dw.VerifyConsistency(0)
	require.NoError(t, err)
	require.True(t, report.Consistent)
	require.Equal(t, 2, report.TSVCount)
	require.Equal(t, 2, report.SQLiteCount)
	require.Equal(t, 1, report.TSVActiveCount)
	require.Equal(t, 1, report.SQLiteActiveCount)
}

func TestDualWriterVerifyOnlyIntegrationShowsMissingSQLiteRecords(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	tsvStore, _ := newTSVBackendForIntegration(t)
	sqliteStore, _ := newSQLiteBackendForIntegration(t)

	dw, err := NewDualWriterWithBackends(tsvStore, sqliteStore, DualWriterOptions{VerifyOnly: true})
	require.NoError(t, err)

	_, err = dw.AddNotification("tsv-only", "2026-01-01T00:00:00Z", "s", "w", "p", "", "info")
	require.NoError(t, err)

	report, err := dw.VerifyConsistency(10)
	require.NoError(t, err)
	require.False(t, report.Consistent)
	require.Equal(t, 1, report.TSVCount)
	require.Equal(t, 0, report.SQLiteCount)
	require.Equal(t, []string{"1"}, report.MissingInSQLite)
}

func TestDualWriterAutoMigratesTSVToSQLite(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")
	t.Setenv("TMUX_INTRAY_STATE_DIR", t.TempDir())

	// Setup tmux mock
	tmuxMock := new(tmux.MockClient)
	tmuxMock.On("HasSession").Return(true, nil)
	tmuxMock.On("SetStatusOption", "@tmux_intray_active_count", mock.Anything).Return(nil)
	SetTmuxClient(tmuxMock)
	sqlite.SetTmuxClient(tmuxMock)

	// Initialize storage to create the files
	Reset()
	defer Reset()

	// Create TSV storage and add data (this will be the data to migrate)
	tsvStore, err := NewFileStorage()
	require.NoError(t, err)

	_, err = tsvStore.AddNotification("first-msg", "2026-01-01T00:00:00Z", "s1", "w1", "p1", "", "info")
	require.NoError(t, err)
	_, err = tsvStore.AddNotification("second-msg", "2026-01-02T00:00:00Z", "s2", "w2", "p2", "", "warning")
	require.NoError(t, err)
	err = tsvStore.DismissNotification("1")
	require.NoError(t, err)

	// Verify TSV has data
	tsvCount := tsvStore.GetActiveCount()
	require.Equal(t, 1, tsvCount)

	// Now create DualWriter - this should detect TSV has data and SQLite is empty, and migrate
	dw, err := NewDualWriter(DualWriterOptions{ReadBackend: ReadBackendSQLite})
	require.NoError(t, err)

	// Verify data is now in SQLite via ListNotifications (uses SQLite backend by default)
	list, err := dw.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Contains(t, list, "first-msg")
	require.Contains(t, list, "second-msg")

	// Verify active count is correct
	require.Equal(t, 1, dw.GetActiveCount())

	// Verify consistency between backends
	report, err := dw.VerifyConsistency(0)
	require.NoError(t, err)
	require.True(t, report.Consistent)
	require.Equal(t, 2, report.TSVCount)
	require.Equal(t, 2, report.SQLiteCount)
}

func TestDualWriterMigrationSkipsWhenSQLiteHasData(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	// Create both backends with pre-existing data
	tsvStore, _ := newTSVBackendForIntegration(t)
	sqliteStore, _ := newSQLiteBackendForIntegration(t)

	// Add different data to both
	_, err := tsvStore.AddNotification("tsv-msg", "2026-01-01T00:00:00Z", "s1", "w1", "p1", "", "info")
	require.NoError(t, err)

	_, err = sqliteStore.AddNotification("sqlite-msg", "2026-01-02T00:00:00Z", "s2", "w2", "p2", "", "warning")
	require.NoError(t, err)

	// Create DualWriter - should NOT migrate (SQLite has data)
	_, err = NewDualWriterWithBackends(tsvStore, sqliteStore, DualWriterOptions{ReadBackend: ReadBackendSQLite})
	require.NoError(t, err)

	// Verify SQLite still only has its original data
	sqliteList, err := sqliteStore.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Contains(t, sqliteList, "sqlite-msg")
	require.NotContains(t, sqliteList, "tsv-msg")
}

func TestDualWriterMigrationHandlesVariousMessageTypes(t *testing.T) {
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")
	t.Setenv("TMUX_INTRAY_STATE_DIR", t.TempDir())

	// Setup tmux mock
	tmuxMock := new(tmux.MockClient)
	tmuxMock.On("HasSession").Return(true, nil)
	tmuxMock.On("SetStatusOption", "@tmux_intray_active_count", mock.Anything).Return(nil)
	SetTmuxClient(tmuxMock)
	sqlite.SetTmuxClient(tmuxMock)

	// Initialize storage
	Reset()
	defer Reset()

	// Create TSV storage and add various types of messages
	tsvStore, err := NewFileStorage()
	require.NoError(t, err)

	// Add messages with different content types
	_, err = tsvStore.AddNotification("simple message", "2026-01-01T00:00:00Z", "s", "w", "p", "", "info")
	require.NoError(t, err)
	_, err = tsvStore.AddNotification("message with special chars: !@#$%^&*()", "2026-01-02T00:00:00Z", "s", "w", "p", "", "warning")
	require.NoError(t, err)
	_, err = tsvStore.AddNotification("message with unicode: café 日本語", "2026-01-03T00:00:00Z", "s", "w", "p", "", "error")
	require.NoError(t, err)

	// Create DualWriter - should auto-migrate
	dw, err := NewDualWriter(DualWriterOptions{ReadBackend: ReadBackendSQLite})
	require.NoError(t, err)

	// Verify all data migrated correctly
	list, err := dw.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Contains(t, list, "simple message")
	require.Contains(t, list, "message with special chars")
	require.Contains(t, list, "message with unicode")

	// Verify consistency
	report, err := dw.VerifyConsistency(0)
	require.NoError(t, err)
	require.True(t, report.Consistent)
	require.Equal(t, 3, report.TSVCount)
	require.Equal(t, 3, report.SQLiteCount)
}
