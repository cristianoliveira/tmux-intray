package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/require"
)

func TestCore(t *testing.T) {
	// Set up test environment
	tmpDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_DEBUG", "true")
	colors.SetDebug(true)
	// Reset storage state
	storage.Reset()

	// Create a SQLite storage instance for tests
	dbPath := filepath.Join(tmpDir, "notifications.db")
	sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqliteStorage.Close()
	})

	// Helper to clear all notifications before each subtest
	clearNotifications := func() {
		_ = sqliteStorage.DismissAll()
	}

	t.Run("GetTrayItems", func(t *testing.T) {
		clearNotifications()
		// Use default core (tmux not needed for this test)
		c := NewCore(nil, sqliteStorage)

		// Add notifications with explicit session/window/pane
		id1, err := c.AddTrayItem("message 1", "$1", "%1", "@1", "123456", true, "info")
		require.NoError(t, err)
		require.NotEmpty(t, id1)
		id2, err := c.AddTrayItem("message 2", "$2", "%2", "@2", "789012", true, "warning")
		require.NoError(t, err)
		require.NotEmpty(t, id2)

		// Get active items
		items, _ := c.GetTrayItems("active")
		require.Contains(t, items, "message 1")
		require.Contains(t, items, "message 2")
		lines := strings.Split(strings.TrimSpace(items), "\n")
		require.Len(t, lines, 2)

		// Filter by dismissed state returns empty
		items, _ = c.GetTrayItems("dismissed")
		require.Empty(t, strings.TrimSpace(items))
	})

	t.Run("AddTrayItem", func(t *testing.T) {
		clearNotifications()
		// Mock tmux client for auto-context
		mockClient := new(tmux.MockClient)
		mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{
			SessionID: "$0",
			WindowID:  "1",
			PaneID:    "%pane",
			PanePID:   "2024-01-01T12:00:00Z",
		}, nil).Once()
		c := NewCore(mockClient, sqliteStorage)

		id, err := c.AddTrayItem("auto message", "", "", "", "", false, "info")
		require.NoError(t, err)
		require.NotEmpty(t, id)
		// Verify item added (message appears)
		items, _ := c.GetTrayItems("active")
		require.Contains(t, items, "auto message")
		mockClient.AssertExpectations(t)
	})

	t.Run("AddTrayItemNoAuto", func(t *testing.T) {
		clearNotifications()
		// tmux client will fail, but noAuto true means we don't call it
		mockClient := new(tmux.MockClient)
		c := NewCore(mockClient, sqliteStorage)
		id, err := c.AddTrayItem("manual message", "$s", "%w", "@p", "123", true, "error")
		require.NoError(t, err)
		require.NotEmpty(t, id)
		items, _ := c.GetTrayItems("active")
		require.Contains(t, items, "manual message")
		mockClient.AssertExpectations(t)
	})

	t.Run("ClearTrayItems", func(t *testing.T) {
		clearNotifications()
		// Use default core
		c := NewCore(nil, sqliteStorage)

		_, err := c.AddTrayItem("msg1", "$1", "%1", "@1", "123", true, "info")
		require.NoError(t, err)
		_, err = c.AddTrayItem("msg2", "$2", "%2", "@2", "456", true, "warning")
		require.NoError(t, err)

		err = c.ClearTrayItems()
		require.NoError(t, err)

		items, _ := c.GetTrayItems("active")
		require.Empty(t, strings.TrimSpace(items))
	})

	t.Run("Visibility", func(t *testing.T) {
		// Mock tmux client for visibility operations
		mockClient := new(tmux.MockClient)
		mockClient.On("GetEnvironment", "TMUX_INTRAY_VISIBLE").Return("1", nil).Once()
		c := NewCore(mockClient, sqliteStorage)

		visible, err := c.GetVisibility()
		require.NoError(t, err)
		require.Equal(t, "1", visible)
		mockClient.AssertExpectations(t)

		// SetVisibility with true should call set-environment with "1"
		mockClient = new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "1").Return(nil).Once()
		c = NewCore(mockClient, sqliteStorage)
		err = c.SetVisibility(true)
		require.NoError(t, err)
		mockClient.AssertExpectations(t)

		// SetVisibility with false should set "0"
		mockClient = new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "0").Return(nil).Once()
		c = NewCore(mockClient, sqliteStorage)
		err = c.SetVisibility(false)
		require.NoError(t, err)
		mockClient.AssertExpectations(t)

		// Simulate tmux failure
		mockClient = new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "1").Return(tmux.ErrTmuxNotRunning).Once()
		c = NewCore(mockClient, sqliteStorage)
		err = c.SetVisibility(true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "set tmux visibility")
		mockClient.AssertExpectations(t)
	})
}
