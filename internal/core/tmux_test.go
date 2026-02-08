package core

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/require"
)

func TestTmuxFunctions(t *testing.T) {
	t.Run("EnsureTmuxRunning", func(t *testing.T) {
		mockClient := new(tmux.MockClient)

		// Test when tmux is running
		mockClient.On("HasSession").Return(true, nil).Once()
		c := NewCore(mockClient, nil)
		result := c.EnsureTmuxRunning()
		require.True(t, result)
		mockClient.AssertExpectations(t)

		// Test when tmux is not running (returns error)
		mockClient = new(tmux.MockClient)
		mockClient.On("HasSession").Return(false, tmux.ErrTmuxNotRunning).Once()
		c = NewCore(mockClient, nil)
		result = c.EnsureTmuxRunning()
		require.False(t, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("ValidatePaneExists", func(t *testing.T) {
		mockClient := new(tmux.MockClient)

		// Test when pane exists
		mockClient.On("ValidatePaneExists", "1", "1", "%1").Return(true, nil).Once()
		c := NewCore(mockClient, nil)
		result := c.ValidatePaneExists("1", "1", "%1")
		require.True(t, result)
		mockClient.AssertExpectations(t)

		// Test when pane doesn't exist
		mockClient = new(tmux.MockClient)
		mockClient.On("ValidatePaneExists", "1", "1", "%999").Return(false, nil).Once()
		c = NewCore(mockClient, nil)
		result = c.ValidatePaneExists("1", "1", "%999")
		require.False(t, result)
		mockClient.AssertExpectations(t)

		// Test when command fails
		mockClient = new(tmux.MockClient)
		mockClient.On("ValidatePaneExists", "1", "1", "%1").Return(false, tmux.ErrTmuxNotRunning).Once()
		c = NewCore(mockClient, nil)
		result = c.ValidatePaneExists("1", "1", "%1")
		require.False(t, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("JumpToPane", func(t *testing.T) {
		// Tiger Style: Test successful jump to existing pane
		mockClient := new(tmux.MockClient)
		mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{SessionID: "$0"}, nil).Once()
		mockClient.On("ValidatePaneExists", "$0", "1", "%1").Return(true, nil).Once()
		mockClient.On("Run", []string{"select-window", "-t", "$0:1"}).Return("", "", nil).Once()
		mockClient.On("Run", []string{"select-pane", "-t", "$0:1.%1"}).Return("", "", nil).Once()
		c := NewCore(mockClient, nil)
		result := c.JumpToPane("$0", "1", "%1")
		require.True(t, result)
		mockClient.AssertExpectations(t)

		// Tiger Style: Test jump to non-existing pane (should fall back to window)
		mockClient = new(tmux.MockClient)
		mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{SessionID: "$0"}, nil).Once()
		mockClient.On("ValidatePaneExists", "$0", "1", "%1").Return(false, nil).Once()
		mockClient.On("Run", []string{"select-window", "-t", "$0:1"}).Return("", "", nil).Once()
		c = NewCore(mockClient, nil)
		result = c.JumpToPane("$0", "1", "%1")
		require.True(t, result)
		mockClient.AssertExpectations(t)

		// Tiger Style: Test switching client session for cross-session jump
		mockClient = new(tmux.MockClient)
		mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{SessionID: "$1"}, nil).Once()
		mockClient.On("Run", []string{"switch-client", "-t", "$2"}).Return("", "", nil).Once()
		mockClient.On("ValidatePaneExists", "$2", "1", "%1").Return(true, nil).Once()
		mockClient.On("Run", []string{"select-window", "-t", "$2:1"}).Return("", "", nil).Once()
		mockClient.On("Run", []string{"select-pane", "-t", "$2:1.%1"}).Return("", "", nil).Once()
		c = NewCore(mockClient, nil)
		result = c.JumpToPane("$2", "1", "%1")
		require.True(t, result)
		mockClient.AssertExpectations(t)

		// Tiger Style: Test that select-window failure returns false (fail-fast)
		mockClient = new(tmux.MockClient)
		mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{SessionID: "$999"}, nil).Once()
		mockClient.On("ValidatePaneExists", "$999", "1", "%1").Return(true, nil).Once()
		mockClient.On("Run", []string{"select-window", "-t", "$999:1"}).Return("", "invalid target", tmux.ErrInvalidTarget).Once()
		c = NewCore(mockClient, nil)
		result = c.JumpToPane("$999", "1", "%1")
		require.False(t, result)
		mockClient.AssertExpectations(t)

		// Tiger Style: Test that select-pane failure returns false (error not swallowed)
		mockClient = new(tmux.MockClient)
		mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{SessionID: "$0"}, nil).Once()
		mockClient.On("ValidatePaneExists", "$0", "1", "%1").Return(true, nil).Once()
		mockClient.On("Run", []string{"select-window", "-t", "$0:1"}).Return("", "", nil).Once()
		mockClient.On("Run", []string{"select-pane", "-t", "$0:1.%1"}).Return("", "invalid pane", tmux.ErrInvalidTarget).Once()
		c = NewCore(mockClient, nil)
		result = c.JumpToPane("$0", "1", "%1")
		require.False(t, result)
		mockClient.AssertExpectations(t)

		// Tiger Style: Test that empty parameters are rejected (input validation)
		mockClient = new(tmux.MockClient)
		c = NewCore(mockClient, nil)
		result = c.JumpToPane("", "1", "%1")
		require.False(t, result)

		result = c.JumpToPane("$0", "", "%1")
		require.False(t, result)

		result = c.JumpToPane("$0", "1", "")
		require.False(t, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("GetTmuxVisibility", func(t *testing.T) {
		// Test when variable is set to 1
		mockClient := new(tmux.MockClient)
		mockClient.On("GetEnvironment", "TMUX_INTRAY_VISIBLE").Return("1", nil).Once()
		c := NewCore(mockClient, nil)
		result := c.GetTmuxVisibility()
		require.Equal(t, "1", result)
		mockClient.AssertExpectations(t)

		// Test when variable is set to 0
		mockClient = new(tmux.MockClient)
		mockClient.On("GetEnvironment", "TMUX_INTRAY_VISIBLE").Return("0", nil).Once()
		c = NewCore(mockClient, nil)
		result = c.GetTmuxVisibility()
		require.Equal(t, "0", result)
		mockClient.AssertExpectations(t)

		// Test when variable is not set
		mockClient = new(tmux.MockClient)
		mockClient.On("GetEnvironment", "TMUX_INTRAY_VISIBLE").Return("", tmux.ErrTmuxNotRunning).Once()
		c = NewCore(mockClient, nil)
		result = c.GetTmuxVisibility()
		require.Equal(t, "0", result)
		mockClient.AssertExpectations(t)
	})

	t.Run("SetTmuxVisibility", func(t *testing.T) {
		// Test successful set
		mockClient := new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "1").Return(nil).Once()
		c := NewCore(mockClient, nil)
		success, err := c.SetTmuxVisibility("1")
		require.True(t, success)
		require.NoError(t, err)
		mockClient.AssertExpectations(t)

		// Test failed set
		mockClient = new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "1").Return(tmux.ErrTmuxNotRunning).Once()
		c = NewCore(mockClient, nil)
		success, err = c.SetTmuxVisibility("1")
		require.False(t, success)
		require.Error(t, err)
		require.Contains(t, err.Error(), "tmux server is not running")
		mockClient.AssertExpectations(t)
	})

	t.Run("GetCurrentTmuxContext", func(t *testing.T) {
		// Tiger Style: Test successful parsing of 4-part format
		mockClient := new(tmux.MockClient)
		mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{
			SessionID: "$3",
			WindowID:  "@16",
			PaneID:    "%21",
			PanePID:   "8443",
		}, nil).Once()
		c := NewCore(mockClient, nil)
		ctx := c.GetCurrentTmuxContext()
		require.Equal(t, "$3", ctx.SessionID)
		require.Equal(t, "@16", ctx.WindowID)
		require.Equal(t, "%21", ctx.PaneID)
		require.Equal(t, "8443", ctx.PaneCreated)
		mockClient.AssertExpectations(t)

		// Tiger Style: Test tmux command failure
		mockClient = new(tmux.MockClient)
		mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{}, tmux.ErrTmuxNotRunning).Once()
		c = NewCore(mockClient, nil)
		ctx = c.GetCurrentTmuxContext()
		require.Equal(t, "", ctx.SessionID)
		mockClient.AssertExpectations(t)
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
