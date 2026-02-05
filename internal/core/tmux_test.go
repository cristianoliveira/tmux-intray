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
		// Test successful jump to existing pane
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
		result := JumpToPane("1", "1", "%1")
		if !result {
			t.Error("Expected true when jump succeeds")
		}
		if !selectWindowCalled {
			t.Error("Expected select-window to be called")
		}
		if !selectPaneCalled {
			t.Error("Expected select-pane to be called")
		}

		// Test jump to non-existing pane (should fall back to window)
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
		result = JumpToPane("1", "1", "%1") // Try to jump to %1
		if !result {
			t.Error("Expected true when window exists even if pane doesn't")
		}
		if !selectWindowCalled {
			t.Error("Expected select-window to be called even when pane doesn't exist")
		}
		if selectPaneCalled {
			t.Error("Did not expect select-pane to be called when pane doesn't exist")
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
