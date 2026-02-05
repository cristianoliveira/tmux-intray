package core

import (
	"os"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
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

	// Backup original tmuxRunner
	origRunner := tmuxRunner
	defer func() { tmuxRunner = origRunner }()

	// Helper to clear all notifications before each subtest
	clearNotifications := func() {
		_ = storage.DismissAll()
	}

	t.Run("GetTrayItems", func(t *testing.T) {
		clearNotifications()
		// Mock tmuxRunner to simulate tmux not running
		tmuxRunner = func(args ...string) (string, string, error) {
			return "", "tmux not running", os.ErrNotExist
		}

		// Add notifications with explicit session/window/pane
		id1 := AddTrayItem("message 1", "$1", "%1", "@1", "123456", true, "info")
		require.NotEmpty(t, id1)
		id2 := AddTrayItem("message 2", "$2", "%2", "@2", "123457", true, "warning")
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
		// Mock tmuxRunner to return a known context
		tmuxRunner = func(args ...string) (string, string, error) {
			if len(args) >= 3 && args[0] == "display" && args[1] == "-p" {
				return "$session %window @pane 1748987643", "", nil
			}
			return "", "", os.ErrNotExist
		}

		id := AddTrayItem("auto message", "", "", "", "", false, "info")
		require.NotEmpty(t, id)
		// Verify item added (message appears)
		items := GetTrayItems("active")
		require.Contains(t, items, "auto message")
	})

	t.Run("AddTrayItemNoAuto", func(t *testing.T) {
		clearNotifications()
		// tmuxRunner will fail, but noAuto true means we don't call it
		tmuxRunner = func(args ...string) (string, string, error) {
			return "", "tmux not running", os.ErrNotExist
		}
		id := AddTrayItem("manual message", "$s", "%w", "@p", "123", true, "error")
		require.NotEmpty(t, id)
		items := GetTrayItems("active")
		require.Contains(t, items, "manual message")
	})

	t.Run("ClearTrayItems", func(t *testing.T) {
		clearNotifications()
		tmuxRunner = func(args ...string) (string, string, error) {
			return "", "tmux not running", os.ErrNotExist
		}

		AddTrayItem("msg1", "$1", "%1", "@1", "123", true, "info")
		AddTrayItem("msg2", "$2", "%2", "@2", "456", true, "warning")

		err := ClearTrayItems()
		require.NoError(t, err)

		items := GetTrayItems("active")
		require.Empty(t, strings.TrimSpace(items))
	})

	t.Run("Visibility", func(t *testing.T) {
		clearNotifications()
		// Mock tmuxRunner for visibility operations
		var lastArgs []string
		tmuxRunner = func(args ...string) (string, string, error) {
			lastArgs = args
			if args[0] == "show-environment" && len(args) >= 3 && args[2] == "TMUX_INTRAY_VISIBLE" {
				return "TMUX_INTRAY_VISIBLE=1", "", nil
			}
			return "", "", nil
		}

		visible := GetVisibility()
		require.Equal(t, "1", visible)

		// SetVisibility with true should call set-environment with "1"
		lastArgs = nil
		tmuxRunner = func(args ...string) (string, string, error) {
			lastArgs = args
			return "", "", nil
		}
		err := SetVisibility(true)
		require.NoError(t, err)
		require.Equal(t, []string{"set-environment", "-g", "TMUX_INTRAY_VISIBLE", "1"}, lastArgs)

		// SetVisibility with false should set "0"
		lastArgs = nil
		tmuxRunner = func(args ...string) (string, string, error) {
			lastArgs = args
			return "", "", nil
		}
		err = SetVisibility(false)
		require.NoError(t, err)
		require.Equal(t, []string{"set-environment", "-g", "TMUX_INTRAY_VISIBLE", "0"}, lastArgs)

		// Simulate tmux failure
		tmuxRunner = func(args ...string) (string, string, error) {
			return "", "error", os.ErrInvalid
		}
		err = SetVisibility(true)
		require.Error(t, err)
		require.Equal(t, ErrTmuxOperationFailed, err)
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
