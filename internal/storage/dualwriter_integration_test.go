package storage

import (
	"testing"

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
