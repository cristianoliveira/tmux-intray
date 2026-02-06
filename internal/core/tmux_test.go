package core

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

func TestTmuxFunctions(t *testing.T) {
	// Backup original tmuxRunner
	origRunner := tmuxRunner
	defer func() { tmuxRunner = origRunner }()

	t.Run("EnsureTmuxRunning", func(t *testing.T) {
		// Test when tmux is running
		tmuxRunner = func(args ...string) (string, string, error) {
			return "has-session", "", nil
		}
		result := EnsureTmuxRunning()
		if !result {
			t.Error("Expected true when tmux is running")
		}

		// Test when tmux is not running (returns error)
		tmuxRunner = func(args ...string) (string, string, error) {
			return "", "", errors.New("exit status 1")
		}
		result = EnsureTmuxRunning()
		if result {
			t.Error("Expected false when tmux is not running")
		}
	})

	t.Run("ValidatePaneExists", func(t *testing.T) {
		// Test when pane exists
		tmuxRunner = func(args ...string) (string, string, error) {
			if args[0] == "list-panes" {
				return "%1\n%2", "", nil
			}
			return "", "", nil
		}
		result := ValidatePaneExists("1", "1", "%1")
		if !result {
			t.Error("Expected true when pane exists")
		}

		// Test when pane doesn't exist
		tmuxRunner = func(args ...string) (string, string, error) {
			if args[0] == "list-panes" {
				return "%1\n%2", "", nil
			}
			return "", "", nil
		}
		result = ValidatePaneExists("1", "1", "%999")
		if result {
			t.Error("Expected false when pane doesn't exist")
		}

		// Test when command fails
		tmuxRunner = func(args ...string) (string, string, error) {
			return "", "", errors.New("command failed")
		}
		result = ValidatePaneExists("1", "1", "%1")
		if result {
			t.Error("Expected false when command fails")
		}
	})

	t.Run("JumpToPane", func(t *testing.T) {
		// Tiger Style: Test successful jump to existing pane
		// ASSERTION 1: select-window must be called to change the active window
		// ASSERTION 2: select-pane must be called to highlight the target pane
		var selectWindowCalled, selectPaneCalled bool
		tmuxRunner = func(args ...string) (string, string, error) {
			if args[0] == "list-panes" {
				return "%1", "", nil
			} else if args[0] == "select-window" {
				selectWindowCalled = true
				return "", "", nil
			} else if args[0] == "select-pane" {
				selectPaneCalled = true
				return "", "", nil
			}
			return "", "", nil
		}
		result := JumpToPane("$0", "1", "%1")
		if !result {
			t.Error("Expected true when jump succeeds")
		}
		if !selectWindowCalled {
			t.Error("Expected select-window to be called")
		}
		if !selectPaneCalled {
			t.Error("Expected select-pane to be called")
		}

		// Tiger Style: Test jump to non-existing pane (should fall back to window)
		// ASSERTION 1: select-window must still be called for fallback to window
		// ASSERTION 2: select-pane must NOT be called if pane doesn't exist
		selectWindowCalled = false
		selectPaneCalled = false
		tmuxRunner = func(args ...string) (string, string, error) {
			if args[0] == "list-panes" {
				return "%2", "", nil // Different pane, so %1 doesn't exist
			} else if args[0] == "select-window" {
				selectWindowCalled = true
				return "", "", nil
			} else if args[0] == "select-pane" {
				selectPaneCalled = true
				return "", "", nil
			}
			return "", "", nil
		}
		result = JumpToPane("$0", "1", "%1") // Try to jump to %1
		if !result {
			t.Error("Expected true when window exists even if pane doesn't")
		}
		if !selectWindowCalled {
			t.Error("Expected select-window to be called even when pane doesn't exist")
		}
		if selectPaneCalled {
			t.Error("Did not expect select-pane to be called when pane doesn't exist")
		}

		// Tiger Style: Test that select-window failure returns false (fail-fast)
		// ASSERTION: When select-window fails, return false immediately without trying select-pane
		tmuxRunner = func(args ...string) (string, string, error) {
			if args[0] == "list-panes" {
				return "%1", "", nil
			} else if args[0] == "select-window" {
				return "", "invalid target", errors.New("invalid target")
			}
			return "", "", nil
		}
		result = JumpToPane("$999", "1", "%1")
		if result {
			t.Error("Expected false when select-window fails")
		}

		// Tiger Style: Test that select-pane failure returns false (error not swallowed)
		// ASSERTION: When select-pane fails, return false and don't hide the error
		selectPaneCalled = false
		tmuxRunner = func(args ...string) (string, string, error) {
			if args[0] == "list-panes" {
				return "%1", "", nil // Pane exists
			} else if args[0] == "select-window" {
				return "", "", nil // Window exists
			} else if args[0] == "select-pane" {
				selectPaneCalled = true
				return "", "invalid pane", errors.New("invalid pane")
			}
			return "", "", nil
		}
		result = JumpToPane("$0", "1", "%1")
		if result {
			t.Error("Expected false when select-pane fails (error not swallowed)")
		}
		if !selectPaneCalled {
			t.Error("Expected select-pane to be called even though it will fail")
		}

		// Tiger Style: Test that empty parameters are rejected (input validation)
		// ASSERTION: Empty session/window/pane should return false immediately
		result = JumpToPane("", "1", "%1")
		if result {
			t.Error("Expected false when sessionID is empty")
		}

		result = JumpToPane("$0", "", "%1")
		if result {
			t.Error("Expected false when windowID is empty")
		}

		result = JumpToPane("$0", "1", "")
		if result {
			t.Error("Expected false when paneID is empty")
		}

		// Tiger Style: Test pane reference format is correct (sessionID:paneID, not sessionID:windowID.paneID)
		// ASSERTION: select-pane must be called with correct format (sessionID:paneID)
		var capturedTarget string
		tmuxRunner = func(args ...string) (string, string, error) {
			if args[0] == "list-panes" {
				return "%95", "", nil
			} else if args[0] == "select-window" {
				return "", "", nil
			} else if args[0] == "select-pane" {
				// Capture the target argument
				for i, arg := range args {
					if arg == "-t" && i+1 < len(args) {
						capturedTarget = args[i+1]
						break
					}
				}
				return "", "", nil
			}
			return "", "", nil
		}
		JumpToPane("$2", "@6", "%95")
		// The format should be sessionID:paneID (not sessionID:windowID.paneID)
		if capturedTarget != "$2:%95" {
			t.Errorf("Expected pane target format '$2:%%95', got '%s'", capturedTarget)
		}
	})

	t.Run("GetTmuxVisibility", func(t *testing.T) {
		// Test when variable is set to 1
		tmuxRunner = func(args ...string) (string, string, error) {
			return "TMUX_INTRAY_VISIBLE=1", "", nil
		}
		result := GetTmuxVisibility()
		if result != "1" {
			t.Errorf("Expected '1', got '%s'", result)
		}

		// Test when variable is set to 0
		tmuxRunner = func(args ...string) (string, string, error) {
			return "TMUX_INTRAY_VISIBLE=0", "", nil
		}
		result = GetTmuxVisibility()
		if result != "0" {
			t.Errorf("Expected '0', got '%s'", result)
		}

		// Test when variable is not set
		tmuxRunner = func(args ...string) (string, string, error) {
			return "", "", errors.New("variable not found")
		}
		result = GetTmuxVisibility()
		if result != "0" {
			t.Errorf("Expected '0' when variable is not set, got '%s'", result)
		}
	})

	t.Run("SetTmuxVisibility", func(t *testing.T) {
		// Test successful set
		tmuxRunner = func(args ...string) (string, string, error) {
			return "", "", nil
		}
		result := SetTmuxVisibility("1")
		if !result {
			t.Error("Expected true when set succeeds")
		}

		// Test failed set
		tmuxRunner = func(args ...string) (string, string, error) {
			return "", "invalid option", errors.New("invalid option")
		}
		result = SetTmuxVisibility("1")
		if result {
			t.Error("Expected false when set fails")
		}
	})
}

func TestColorsErrorFallback(t *testing.T) {
	// Test error fallback function
	defer func() {
		// Reset colors after test
		colors.SetDebug(false)
	}()

	// This tests the errorFallback function in colors package
	// Since it's not directly exported, we trigger it by calling colors.Error
	// when color support is not available
	colors.Error("test error message")
}
