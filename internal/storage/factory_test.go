package storage

import (
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
