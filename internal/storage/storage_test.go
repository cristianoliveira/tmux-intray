package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorageStubs(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")
	os.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	Init()

	require.Equal(t, "", AddNotification("msg", "", "", "", "", "", "info"))
	require.Equal(t, "", ListNotifications("active", "", "", "", "", "", ""))

	DismissNotification("1")
	DismissAll()
	CleanupOldNotifications(30, true)

	require.Equal(t, 0, GetActiveCount())
}
