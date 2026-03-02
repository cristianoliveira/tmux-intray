package state

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/require"
)

// TestTUIWithRealTmuxClient verifies TUI works with real TmuxClient.
// This is an integration test that requires tmux to be running.
func TestTUIWithRealTmuxClient(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setupStorage(t)

	// Create real TmuxClient
	client := tmux.NewDefaultClient()

	// Verify tmux is running
	hasSession, err := client.HasSession()
	if err != nil || !hasSession {
		t.Skip("tmux not running, skipping integration test")
	}

	// Add a test notification
	_, err = storage.AddNotification("Test notification for TUI", "", "$1", "@1", "%1", "", "info")
	require.NoError(t, err)

	// Create TUI model with real client and core
	model, err := NewModel(client)
	require.NoError(t, err)
	require.NotNil(t, model)

	// Verify uiState was initialized
	require.NotNil(t, model.uiState, "uiState should be initialized")

	// Verify session names were loaded through runtime coordinator
	names, err := model.runtimeCoordinator.ListSessions()
	require.NoError(t, err)
	require.NotEmpty(t, names, "session names should be loaded from tmux")

	// Verify notification was loaded
	require.NotEmpty(t, model.notifications, "notifications should be loaded from storage")
	require.NotEmpty(t, model.filtered, "filtered notifications should not be empty")

	// Verify session name lookup works
	sessionName := model.getSessionName("$1")
	require.NotEmpty(t, sessionName, "session name should be found")
}

// TestTUIClientNilDefaults tests that NewModel creates a default client when nil is passed.
func TestTUIClientNilDefaults(t *testing.T) {
	setupStorage(t)

	// Create TUI model with nil client (should create default)
	model, err := NewModel(nil)
	require.NoError(t, err)
	require.NotNil(t, model)

	// Verify uiState was initialized
	require.NotNil(t, model.uiState, "uiState should be initialized")

	// Verify runtime coordinator was created
	require.NotNil(t, model.runtimeCoordinator, "runtime coordinator should be created")
}
