package storage

import (
	"path/filepath"
	"testing"

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

func TestNewFromConfigSelectsSQLiteByDefault(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	setupIsolatedNewFromConfigTest(t)

	stor, err := NewFromConfig()
	require.NoError(t, err)
	require.IsType(t, &sqlite.SQLiteStorage{}, stor)

	if sqliteStorage, ok := stor.(*sqlite.SQLiteStorage); ok {
		require.NoError(t, sqliteStorage.Close())
	}
}

func TestNewForBackendReturnsErrorForInvalidBackend(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	setupIsolatedNewFromConfigTest(t)

	stor, err := NewForBackend("unknown")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown storage backend")
	require.Nil(t, stor)
}

func TestNewForBackendReturnsErrorForEmptyBackend(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	setupIsolatedNewFromConfigTest(t)

	stor, err := NewForBackend("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown storage backend")
	require.Nil(t, stor)
}
