package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/stretchr/testify/require"
)

func setupIsolatedNewFromConfigTest(t *testing.T) string {
	t.Helper()

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.toml"))

	return stateDir
}

func TestNewFromConfigSelectsTSVByDefault(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	setupIsolatedNewFromConfigTest(t)

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &FileStorage{}, stor)
}

func TestNewFromConfigSelectsSQLiteBackend(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	setupIsolatedNewFromConfigTest(t)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "sqlite")

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &sqlite.SQLiteStorage{}, stor)

	if sqliteStorage, ok := stor.(*sqlite.SQLiteStorage); ok {
		require.NoError(t, sqliteStorage.Close())
	}
}

func TestNewFromConfigFallsBackToTSVForNonCanonicalBackendValue(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	setupIsolatedNewFromConfigTest(t)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "  SQlite  ")

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &FileStorage{}, stor)
}

func TestNewFromConfigFallsBackToTSVForInvalidBackend(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	setupIsolatedNewFromConfigTest(t)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "unknown")

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &FileStorage{}, stor)
}

func TestNewFromConfigFallsBackToTSVForEmptyBackendValue(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	setupIsolatedNewFromConfigTest(t)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "")

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &FileStorage{}, stor)
}

func TestNewFromConfigSelectsDualBackend(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	setupIsolatedNewFromConfigTest(t)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "dual")

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &DualWriter{}, stor)
}

func TestNewFromConfigAutoMigratesTSVDataWhenSQLiteOptIn(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	stateDir := setupIsolatedNewFromConfigTest(t)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "sqlite")

	ts := time.Now().UTC().Format(time.RFC3339)
	tsvLine := fmt.Sprintf("1\t%s\tactive\ts\tw\tp\tmsg\t\tinfo\t", ts)
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "notifications.tsv"), []byte(tsvLine+"\n"), 0o644))

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &sqlite.SQLiteStorage{}, stor)

	sqliteStorage, ok := stor.(*sqlite.SQLiteStorage)
	require.True(t, ok)
	t.Cleanup(func() {
		require.NoError(t, sqliteStorage.Close())
	})

	list, err := sqliteStorage.ListNotifications("all", "", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Contains(t, list, "\tmsg\t")

	backupPath := filepath.Join(stateDir, "notifications.tsv.sqlite-migration.bak")
	require.FileExists(t, backupPath)
}

func TestNewFromConfigSkipsAutoMigrationWhenSQLiteAlreadyHasData(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	stateDir := setupIsolatedNewFromConfigTest(t)
	t.Setenv("TMUX_INTRAY_STORAGE_BACKEND", "sqlite")

	sqlitePath := filepath.Join(stateDir, "notifications.db")
	seedStore, err := sqlite.NewSQLiteStorage(sqlitePath)
	require.NoError(t, err)
	_, err = seedStore.AddNotification("sqlite-data", time.Now().UTC().Format(time.RFC3339), "", "", "", "", "info")
	require.NoError(t, err)
	require.NoError(t, seedStore.Close())

	ts := time.Now().UTC().Format(time.RFC3339)
	tsvLine := fmt.Sprintf("99\t%s\tactive\t\t\t\ttsv-data\t\tinfo\t", ts)
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "notifications.tsv"), []byte(tsvLine+"\n"), 0o644))

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &sqlite.SQLiteStorage{}, stor)

	sqliteStorage, ok := stor.(*sqlite.SQLiteStorage)
	require.True(t, ok)
	t.Cleanup(func() {
		require.NoError(t, sqliteStorage.Close())
	})

	list, err := sqliteStorage.ListNotifications("all", "", "", "", "", "", "", "")
	require.NoError(t, err)
	require.Contains(t, list, "sqlite-data")
	require.NotContains(t, list, "tsv-data")

	backupPath := filepath.Join(stateDir, "notifications.tsv.sqlite-migration.bak")
	_, err = os.Stat(backupPath)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
	require.Equal(t, 1, len(strings.Split(strings.TrimSpace(list), "\n")))
}
