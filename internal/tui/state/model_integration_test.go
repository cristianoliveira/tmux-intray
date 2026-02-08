package state

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
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
	_, err = storage.AddNotification("Test notification for TUI", "", "$1", "", "@1", "%1", "", "info")
	require.NoError(t, err)

	// Create TUI model with real client
	model, err := NewModel(client)
	require.NoError(t, err)
	require.NotNil(t, model)

	// Verify notification was loaded
	require.NotEmpty(t, model.notifications, "notifications should be loaded from storage")
	require.NotEmpty(t, model.filtered, "filtered notifications should not be empty")

	// Find the notification we added
	var foundNotif *notification.Notification
	for i := range model.notifications {
		if model.notifications[i].ID == 1 {
			foundNotif = &model.notifications[i]
			break
		}
	}
	require.NotNil(t, foundNotif, "notification should be found")

	// Verify session name lookup works (returns session ID as fallback since SessionName is empty)
	sessionName := model.getSessionName(*foundNotif)
	// Session name should return session ID as fallback when SessionName is empty
	require.Equal(t, "$1", sessionName, "session name should return session ID when SessionName is empty")
}

// TestTUIClientNilDefaults tests that NewModel creates a default client when nil is passed.
func TestTUIClientNilDefaults(t *testing.T) {
	setupStorage(t)

	// Create TUI model with nil client (should create default)
	model, err := NewModel(nil)
	require.NoError(t, err)
	require.NotNil(t, model)

	// Verify client was created
	require.NotNil(t, model.client, "default client should be created")
}
