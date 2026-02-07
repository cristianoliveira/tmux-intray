package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitWithInvalidStateDir(t *testing.T) {
	// Reset and set invalid state_dir (path that cannot be created)
	Reset()
	os.Setenv("TMUX_INTRAY_STATE_DIR", "/nonexistent/that/cannot/be/created")
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Init() should fail when state_dir cannot be created
	err := Init()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create state directory")
}

func TestInitSuccess(t *testing.T) {
	// Reset and set state_dir
	Reset()
	tmpDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Init() should succeed with valid state_dir
	err := Init()
	require.NoError(t, err)
}

func TestInitConcurrent(t *testing.T) {
	// Reset and set state_dir
	Reset()
	tmpDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Test concurrent calls to Init()
	done := make(chan error, 100)
	for i := 0; i < 100; i++ {
		go func() {
			done <- Init()
		}()
	}

	// All should succeed
	for i := 0; i < 100; i++ {
		err := <-done
		require.NoError(t, err)
	}
}

func TestInitIdempotent(t *testing.T) {
	// Reset and set state_dir
	Reset()
	tmpDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Init() should be idempotent
	for i := 0; i < 10; i++ {
		err := Init()
		require.NoError(t, err)
	}
}

func TestInitOpenFileFailure(t *testing.T) {
	// Reset and set up state_dir
	Reset()
	tmpDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")

	// Create the state directory first (to pass MkdirAll check)
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	// Make the directory read-only (prevents file creation)
	// This should cause OpenFile with O_CREATE flag to fail
	err = os.Chmod(tmpDir, 0555) // read and execute only
	require.NoError(t, err)
	defer os.Chmod(tmpDir, 0755) // restore permissions for cleanup

	// Init() should fail when OpenFile fails (cannot create file in read-only directory)
	err = Init()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create notifications file")
}
