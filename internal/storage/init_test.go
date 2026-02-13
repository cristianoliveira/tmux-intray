package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_Success(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.toml"))

	err := Init()
	require.NoError(t, err)
	assert.Equal(t, stateDir, GetStateDir())
}

func TestInit_CreatesDirectories(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	stateDir := filepath.Join(t.TempDir(), "nested", "state", "dir")
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.toml"))

	// Directory doesn't exist yet
	_, err := os.Stat(stateDir)
	require.True(t, os.IsNotExist(err))

	err = Init()
	require.NoError(t, err)

	// Directory should now exist
	info, err := os.Stat(stateDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestInit_ErrorsWhenStateDirNotConfigured(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	// Clear both environment and config
	t.Setenv("TMUX_INTRAY_STATE_DIR", "")
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.toml"))

	err := Init()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TMUX_INTRAY_STATE_DIR not configured")
}

func TestInit_IsIdempotent(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.toml"))

	// Call Init multiple times
	err := Init()
	require.NoError(t, err)

	err = Init()
	require.NoError(t, err)

	err = Init()
	require.NoError(t, err)
}

func TestGetStateDir_FromEnvVarDirectly(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	// Don't call Init, test that GetStateDir reads from env directly
	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.toml"))

	// Without Init, GetStateDir should still read from env
	result := GetStateDir()
	assert.Equal(t, stateDir, result)
}

func TestReset_ClearsState(t *testing.T) {
	Reset()

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.toml"))

	// Initialize
	err := Init()
	require.NoError(t, err)
	require.Equal(t, stateDir, GetStateDir())

	// Reset
	Reset()

	// State should be cleared - GetStateDir now reads from env directly
	// but the internal stateDir variable should be empty
	// We can verify by checking that Init can be called again with a different dir
	newStateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", newStateDir)

	err = Init()
	require.NoError(t, err)
	assert.Equal(t, newStateDir, GetStateDir())
}

func TestReset_ClearsDefaultStorage(t *testing.T) {
	Reset()

	stateDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	t.Setenv("TMUX_INTRAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.toml"))

	require.NoError(t, Init())

	// Add a notification to initialize default storage
	_, err := AddNotification("test", "2025-01-01T12:00:00Z", "sess", "win", "pane", "", "info")
	require.NoError(t, err)

	// Verify it was added
	count := GetActiveCount()
	require.Equal(t, 1, count)

	// Reset should clear this
	Reset()

	// After reset, GetActiveCount returns 0 because the default storage is nil
	// (it returns 0 on error, and without a configured state dir, it will error)
	t.Setenv("TMUX_INTRAY_STATE_DIR", "")
	count = GetActiveCount()
	assert.Equal(t, 0, count)
}
