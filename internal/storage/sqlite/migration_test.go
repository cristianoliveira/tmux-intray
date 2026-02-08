package sqlite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrateTSVToSQLiteMigratesLatestRowsAndSkipsMalformed(t *testing.T) {
	tmpDir := t.TempDir()
	tsvPath := filepath.Join(tmpDir, "notifications.tsv")
	dbPath := filepath.Join(tmpDir, "notifications.db")
	backupPath := filepath.Join(tmpDir, "notifications.tsv.sqlite-migration.bak")

	content := strings.Join([]string{
		"1\t2026-01-01T01:00:00Z\tactive\ts1\tw1\tp1\thello\\nworld\t\tinfo\t",
		"1\t2026-01-01T02:00:00Z\tdismissed\ts1\tw1\tp1\toverride\t\twarning\t",
		"2\t2026-01-01T03:00:00Z\tactive\ts2\tw2\tp2\tgood\t\terror",
		"bad\t2026-01-01T04:00:00Z\tactive\ts2\tw2\tp2\tbad id\t\terror\t",
		"3\tinvalid\tactive\ts2\tw2\tp2\tbad timestamp\t\terror\t",
		"4\t2026-01-01T05:00:00Z\tinvalid\ts2\tw2\tp2\tbad state\t\terror\t",
		"5\t2026-01-01T06:00:00Z\tactive\ts2\tw2\tp2\tbad level\t\tunknown\t",
	}, "\n") + "\n"
	require.NoError(t, os.WriteFile(tsvPath, []byte(content), 0o644))

	stats, err := MigrateTSVToSQLite(MigrationOptions{
		TSVPath:    tsvPath,
		SQLitePath: dbPath,
		BackupPath: backupPath,
	})
	require.NoError(t, err)
	require.Equal(t, 7, stats.TotalRows)
	require.Equal(t, 2, stats.MigratedRows)
	require.Equal(t, 4, stats.SkippedRows)
	require.Equal(t, 0, stats.FailedRows)
	require.Equal(t, 1, stats.DuplicateRows)
	require.True(t, stats.BackupCreated)
	require.Equal(t, backupPath, stats.BackupPath)
	require.Len(t, stats.Warnings, 4)

	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	require.Equal(t, content, string(backupContent))

	s, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	lines, err := s.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	parts := strings.Split(lines, "\n")
	require.Len(t, parts, 2)
	require.Contains(t, lines, "1\t2026-01-01T02:00:00Z\tdismissed")
	require.Contains(t, lines, "2\t2026-01-01T03:00:00Z\tactive")
}

func TestMigrateTSVToSQLiteDryRunDoesNotWriteFiles(t *testing.T) {
	tmpDir := t.TempDir()
	tsvPath := filepath.Join(tmpDir, "notifications.tsv")
	dbPath := filepath.Join(tmpDir, "notifications.db")
	backupPath := filepath.Join(tmpDir, "notifications.tsv.sqlite-migration.bak")

	content := "1\t2026-01-01T01:00:00Z\tactive\ts1\tw1\tp1\tmsg\t\tinfo\t\n"
	require.NoError(t, os.WriteFile(tsvPath, []byte(content), 0o644))

	stats, err := MigrateTSVToSQLite(MigrationOptions{
		TSVPath:    tsvPath,
		SQLitePath: dbPath,
		BackupPath: backupPath,
		DryRun:     true,
	})
	require.NoError(t, err)
	require.Equal(t, 1, stats.TotalRows)
	require.Equal(t, 1, stats.MigratedRows)
	require.Equal(t, 0, stats.SkippedRows)
	require.False(t, stats.BackupCreated)

	_, err = os.Stat(dbPath)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(backupPath)
	require.True(t, os.IsNotExist(err))
}

func TestMigrateTSVToSQLiteIsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	tsvPath := filepath.Join(tmpDir, "notifications.tsv")
	dbPath := filepath.Join(tmpDir, "notifications.db")
	backup1 := filepath.Join(tmpDir, "backup1.tsv")
	backup2 := filepath.Join(tmpDir, "backup2.tsv")

	content := strings.Join([]string{
		"1\t2026-01-01T01:00:00Z\tactive\ts1\tw1\tp1\tmsg1\t\tinfo\t",
		"2\t2026-01-01T02:00:00Z\tdismissed\ts2\tw2\tp2\tmsg2\t\terror\t2026-01-01T03:00:00Z",
	}, "\n") + "\n"
	require.NoError(t, os.WriteFile(tsvPath, []byte(content), 0o644))

	stats, err := MigrateTSVToSQLite(MigrationOptions{TSVPath: tsvPath, SQLitePath: dbPath, BackupPath: backup1})
	require.NoError(t, err)
	require.Equal(t, 2, stats.MigratedRows)

	stats, err = MigrateTSVToSQLite(MigrationOptions{TSVPath: tsvPath, SQLitePath: dbPath, BackupPath: backup2})
	require.NoError(t, err)
	require.Equal(t, 2, stats.MigratedRows)

	s, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	lines, err := s.ListNotifications("all", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Len(t, strings.Split(lines, "\n"), 2)
}

func TestRollbackTSVMigrationRestoresBackupAndRemovesSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	tsvPath := filepath.Join(tmpDir, "notifications.tsv")
	dbPath := filepath.Join(tmpDir, "notifications.db")
	backupPath := filepath.Join(tmpDir, "notifications.tsv.sqlite-migration.bak")

	require.NoError(t, os.WriteFile(tsvPath, []byte("current\n"), 0o644))
	require.NoError(t, os.WriteFile(backupPath, []byte("backup\n"), 0o644))
	require.NoError(t, os.WriteFile(dbPath, []byte("db"), 0o644))

	err := RollbackTSVMigration(tsvPath, dbPath, backupPath)
	require.NoError(t, err)

	restored, err := os.ReadFile(tsvPath)
	require.NoError(t, err)
	require.Equal(t, "backup\n", string(restored))

	_, err = os.Stat(dbPath)
	require.True(t, os.IsNotExist(err))
}
