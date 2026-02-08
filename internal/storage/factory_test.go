package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/stretchr/testify/require"
)

func TestNewFromConfigSelectsTSVByDefault(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	t.Setenv("TMUX_INTRAY_STATE_DIR", t.TempDir())

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &FileStorage{}, stor)
}

func TestNewFromConfigSelectsSQLiteBackend(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	t.Setenv("TMUX_INTRAY_STATE_DIR", t.TempDir())
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "sqlite")

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &sqlite.SQLiteStorage{}, stor)

	if sqliteStorage, ok := stor.(*sqlite.SQLiteStorage); ok {
		require.NoError(t, sqliteStorage.Close())
	}
}

func TestNewFromConfigFallsBackToTSVForInvalidBackend(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	t.Setenv("TMUX_INTRAY_STATE_DIR", t.TempDir())
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "unknown")

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &FileStorage{}, stor)
}

func TestNewFromConfigSQLiteOptInMigratesTSVData(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "sqlite")

	tsvPath := filepath.Join(stateDir, "notifications.tsv")
	tsvData := strings.Join([]string{
		"1\t2026-01-01T00:00:00Z\tactive\tsession-1\twindow-1\tpane-1\tfirst\t\tinfo\t",
		"1\t2026-01-02T00:00:00Z\tdismissed\tsession-1\twindow-1\tpane-1\tupdated\t\terror\t2026-01-02T00:00:10Z",
		"2\t2026-01-03T00:00:00Z\tactive\tsession-2\twindow-2\tpane-2\tsecond\t\twarning\t",
	}, "\n") + "\n"
	require.NoError(t, os.WriteFile(tsvPath, []byte(tsvData), 0o644))

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &sqlite.SQLiteStorage{}, stor)

	sqliteStorage := stor.(*sqlite.SQLiteStorage)
	t.Cleanup(func() {
		require.NoError(t, sqliteStorage.Close())
	})

	line, err := sqliteStorage.GetNotificationByID("1")
	require.NoError(t, err)
	require.Contains(t, line, "\tdismissed\t")
	require.Contains(t, line, "\tupdated\t")

	require.FileExists(t, tsvPath+migrationBackupSuffix)
}

func TestNewFromConfigSQLiteOptInSkipsMigrationWhenDBExists(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "sqlite")

	dbPath := filepath.Join(stateDir, "notifications.db")
	seedStorage, err := sqlite.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	_, err = seedStorage.AddNotification("existing", "2026-01-01T00:00:00Z", "", "", "", "", "info")
	require.NoError(t, err)
	require.NoError(t, seedStorage.Close())

	tsvPath := filepath.Join(stateDir, "notifications.tsv")
	tsvData := "42\t2026-01-03T00:00:00Z\tactive\tsession\twindow\tpane\tshould-not-migrate\t\twarning\t\n"
	require.NoError(t, os.WriteFile(tsvPath, []byte(tsvData), 0o644))

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &sqlite.SQLiteStorage{}, stor)

	sqliteStorage := stor.(*sqlite.SQLiteStorage)
	t.Cleanup(func() {
		require.NoError(t, sqliteStorage.Close())
	})

	_, err = sqliteStorage.GetNotificationByID("1")
	require.NoError(t, err)

	_, err = sqliteStorage.GetNotificationByID("42")
	require.Error(t, err)
	require.ErrorContains(t, err, "notification not found")

	require.NoFileExists(t, tsvPath+migrationBackupSuffix)
}

func TestNewFromConfigSQLiteOptInMigrationFailureRollsBackAndFallsBack(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "sqlite")

	tsvPath := filepath.Join(stateDir, "notifications.tsv")
	tsvData := "5\t2026-01-03T00:00:00Z\tactive\tsession\twindow\tpane\tstill-in-tsv\t\tinfo\t\n"
	require.NoError(t, os.WriteFile(tsvPath, []byte(tsvData), 0o644))

	originalMigrate := migrateTSVToSQLite
	originalRollback := rollbackTSVMigration
	t.Cleanup(func() {
		migrateTSVToSQLite = originalMigrate
		rollbackTSVMigration = originalRollback
	})

	rollbackCalled := false
	migrateTSVToSQLite = func(opts sqlite.MigrationOptions) (sqlite.MigrationStats, error) {
		require.NoError(t, os.WriteFile(opts.SQLitePath, []byte("partial"), 0o644))
		return sqlite.MigrationStats{}, fmt.Errorf("simulated migration failure")
	}
	rollbackTSVMigration = func(tsv, db, backup string) error {
		rollbackCalled = true
		require.Equal(t, tsvPath, tsv)
		require.Equal(t, filepath.Join(stateDir, "notifications.db"), db)
		require.Equal(t, tsvPath+migrationBackupSuffix, backup)
		return os.Remove(db)
	}

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &FileStorage{}, stor)
	require.True(t, rollbackCalled)

	_, err = os.Stat(filepath.Join(stateDir, "notifications.db"))
	require.True(t, os.IsNotExist(err))
}
