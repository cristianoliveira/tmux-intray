package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/require"
)

func TestUpdateTmuxStatusOptionWithRealTmux(t *testing.T) {
	// Skip test in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Create a temporary directory for the tmux socket
	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "tmux-test.sock")

	// Create a tmux client with custom socket
	client := tmux.NewDefaultClient(tmux.WithSocketPath(socketPath))

	// Start a new tmux server with the custom socket
	_, _, err := client.Run("-f", "/dev/null", "new-session", "-d", "-s", "test")
	if err != nil {
		t.Skipf("Failed to start tmux server: %v", err)
	}

	// Cleanup: Kill the tmux server after test
	defer func() {
		_, _, _ = client.Run("kill-server")
	}()

	// Wait a bit for the server to start
	time.Sleep(100 * time.Millisecond)

	// Set the tmux client for storage package
	SetTmuxClient(client)

	// Test updateTmuxStatusOption with active count of 5
	err = updateTmuxStatusOption(5)
	require.NoError(t, err)

	// Verify the status option was set correctly
	value, err := client.GetEnvironment("@tmux_intray_active_count")
	require.NoError(t, err)
	require.Equal(t, "5", value)

	// Test updateTmuxStatusOption with active count of 0
	err = updateTmuxStatusOption(0)
	require.NoError(t, err)

	// Verify the status option was updated
	value, err = client.GetEnvironment("@tmux_intray_active_count")
	require.NoError(t, err)
	require.Equal(t, "0", value)
}

func TestUpdateTmuxStatusOptionWithoutTmux(t *testing.T) {
	// Create a client with a non-existent socket
	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "non-existent.sock")
	client := tmux.NewDefaultClient(tmux.WithSocketPath(socketPath))

	// Set the tmux client for storage package
	SetTmuxClient(client)

	// Test updateTmuxStatusOption should fail
	err := updateTmuxStatusOption(5)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tmux not available")
}
