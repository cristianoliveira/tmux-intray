package state

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/assert"
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

	// Create TUI model with real client
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

func TestCrossLaneRecentsAllAndJumpFlow(t *testing.T) {
	notifications := make([]notification.Notification, 0, 25)
	for i := 1; i <= 25; i++ {
		pane := "%1"
		if i == 24 {
			pane = ""
		}

		notifications = append(notifications, notification.Notification{
			ID:        i,
			Session:   "$1",
			Window:    "@1",
			Pane:      pane,
			Message:   "cross-lane",
			Timestamp: fmt.Sprintf("2024-01-%02dT10:00:00Z", i),
			State:     "active",
			Level:     "info",
		})
	}

	var paneJumpCalls int
	var windowJumpCalls int

	model := newTestModelWithOptions(t, notifications, func(m *Model) {
		m.runtimeCoordinator = &testRuntimeCoordinator{
			ensureTmuxRunningFn: func() bool { return true },
			jumpToPaneFn: func(sessionID, windowID, paneID string) bool {
				paneJumpCalls++
				return sessionID == "$1" && windowID == "@1" && paneID == "%1"
			},
			jumpToWindowFn: func(sessionID, windowID string) bool {
				windowJumpCalls++
				return sessionID == "$1" && windowID == "@1"
			},
		}
	})
	model.applySearchFilter()

	assert.Equal(t, settings.TabRecents, model.uiState.GetActiveTab())
	require.Len(t, model.filtered, 20)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(*Model)
	assert.Equal(t, settings.TabAll, model.uiState.GetActiveTab())
	require.Len(t, model.filtered, 25)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(*Model)
	assert.Equal(t, settings.TabRecents, model.uiState.GetActiveTab())
	require.Len(t, model.filtered, 20)

	model.uiState.SetCursor(0)
	require.NotNil(t, model.handleJump())
	assert.Equal(t, 1, paneJumpCalls)
	assert.Equal(t, 0, windowJumpCalls)

	model.uiState.SetCursor(1)
	require.NotNil(t, model.handleJump())
	assert.Equal(t, 1, paneJumpCalls)
	assert.Equal(t, 1, windowJumpCalls)
}
