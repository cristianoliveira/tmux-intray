package core

import (
	"os"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/require"
)

func TestCore(t *testing.T) {
	// Set up a single temporary storage directory for all subtests
	tmpDir := t.TempDir()
	os.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_DEBUG", "true")
	colors.SetDebug(true)
	// Initialize storage once
	storage.Init()

	// Helper to clear all notifications before each subtest
	clearNotifications := func() {
		_ = storage.DismissAll()
	}

	t.Run("GetTrayItems", func(t *testing.T) {
		clearNotifications()
		// Use default core (tmux not needed for this test)
		c := NewCore(nil)

		// Add notifications with explicit session/window/pane
		id1, err := c.AddTrayItem("message 1", "$1", "%1", "@1", "123456", true, "info")
		require.NoError(t, err)
		require.NotEmpty(t, id1)
		id2, err := c.AddTrayItem("message 2", "$2", "%2", "@2", "123457", true, "warning")
		require.NoError(t, err)
		require.NotEmpty(t, id2)

		// Get active items
		items := GetTrayItems("active")
		require.Contains(t, items, "message 1")
		require.Contains(t, items, "message 2")
		lines := strings.Split(strings.TrimSpace(items), "\n")
		require.Len(t, lines, 2)

		// Filter by dismissed state returns empty
		items = GetTrayItems("dismissed")
		require.Empty(t, strings.TrimSpace(items))
	})

	t.Run("AddTrayItemAutoContext", func(t *testing.T) {
		clearNotifications()
		// Mock tmux client to return a known context
		mockClient := new(tmux.MockClient)
		mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{
			SessionID: "$session",
			WindowID:  "%window",
			PaneID:    "@pane",
			PanePID:   "1748987643",
		}, nil).Once()
		c := NewCore(mockClient)

		id, err := c.AddTrayItem("auto message", "", "", "", "", false, "info")
		require.NoError(t, err)
		require.NotEmpty(t, id)
		// Verify item added (message appears)
		items := GetTrayItems("active")
		require.Contains(t, items, "auto message")
		mockClient.AssertExpectations(t)
	})

	t.Run("AddTrayItemNoAuto", func(t *testing.T) {
		clearNotifications()
		// tmux client will fail, but noAuto true means we don't call it
		mockClient := new(tmux.MockClient)
		c := NewCore(mockClient)
		id, err := c.AddTrayItem("manual message", "$s", "%w", "@p", "123", true, "error")
		require.NoError(t, err)
		require.NotEmpty(t, id)
		items := GetTrayItems("active")
		require.Contains(t, items, "manual message")
	})

	t.Run("ClearTrayItems", func(t *testing.T) {
		clearNotifications()
		// Use default core
		c := NewCore(nil)

		_, err := c.AddTrayItem("msg1", "$1", "%1", "@1", "123", true, "info")
		require.NoError(t, err)
		_, err = c.AddTrayItem("msg2", "$2", "%2", "@2", "456", true, "warning")
		require.NoError(t, err)

		err = ClearTrayItems()
		require.NoError(t, err)

		items := GetTrayItems("active")
		require.Empty(t, strings.TrimSpace(items))
	})

	t.Run("Visibility", func(t *testing.T) {
		clearNotifications()
		// Mock tmux client for visibility operations
		mockClient := new(tmux.MockClient)
		mockClient.On("GetEnvironment", "TMUX_INTRAY_VISIBLE").Return("1", nil).Once()
		c := NewCore(mockClient)

		visible := c.GetVisibility()
		require.Equal(t, "1", visible)
		mockClient.AssertExpectations(t)

		// SetVisibility with true should call set-environment with "1"
		mockClient = new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "1").Return(nil).Once()
		c = NewCore(mockClient)
		err := c.SetVisibility(true)
		require.NoError(t, err)
		mockClient.AssertExpectations(t)

		// SetVisibility with false should set "0"
		mockClient = new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "0").Return(nil).Once()
		c = NewCore(mockClient)
		err = c.SetVisibility(false)
		require.NoError(t, err)
		mockClient.AssertExpectations(t)

		// Simulate tmux failure
		mockClient = new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "1").Return(tmux.ErrTmuxNotRunning).Once()
		c = NewCore(mockClient)
		err = c.SetVisibility(true)
		require.Error(t, err)
		require.Equal(t, ErrTmuxOperationFailed, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("escapeMessage", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "normal text",
				input:    "hello world",
				expected: "hello world",
			},
			{
				name:     "newline",
				input:    "line1\nline2",
				expected: "line1\\nline2",
			},
			{
				name:     "tab",
				input:    "col1\tcol2",
				expected: "col1\\tcol2",
			},
			{
				name:     "backslash",
				input:    "path\\to\\file",
				expected: "path\\\\to\\\\file",
			},
			{
				name:     "mixed special chars",
				input:    "line1\n\ttab\\end",
				expected: "line1\\n\\ttab\\\\end",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := escapeMessage(tt.input)
				require.Equal(t, tt.expected, result)
			})
		}
	})
}
