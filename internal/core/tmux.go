// Package core provides core tmux interaction and tray management.
package core

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/errors"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
)

// defaultCLIHandler is the default CLI error handler for backward compatibility.
var defaultCLIHandler = errors.NewDefaultCLIHandler()

// TmuxContext captures the current tmux session/window/pane context.
type TmuxContext struct {
	SessionID   string
	WindowID    string
	PaneID      string
	PaneCreated string
}

// Core provides core tmux interaction functionality with injected TmuxClient and Storage.
type Core struct {
	client  tmux.TmuxClient
	storage storage.Storage
}

// NewCore creates a new Core instance with the given TmuxClient and Storage.
// If client is nil, a default client will be created.
// If storage is nil, a default file storage will be created.
// Panics if storage initialization fails, which is safer than continuing with nil storage.
func NewCore(client tmux.TmuxClient, stor storage.Storage) *Core {
	if client == nil {
		client = tmux.NewDefaultClient()
	}
	if stor == nil {
		fileStor, err := storage.NewFromConfig()
		if err != nil {
			panic(fmt.Sprintf("failed to initialize storage: %v", err))
		}
		stor = fileStor
	}
	return &Core{client: client, storage: stor}
}

// defaultCore is the default instance for backward compatibility.
var defaultCore = NewCore(nil, nil)

// EnsureTmuxRunning verifies that tmux is running.
func (c *Core) EnsureTmuxRunning() bool {
	running, err := c.client.HasSession()
	if err != nil {
		colors.Debug("EnsureTmuxRunning: tmux has-session failed: " + err.Error())
		return false
	}
	return running
}

// EnsureTmuxRunning verifies that tmux is running using the default client.
func EnsureTmuxRunning() bool {
	return defaultCore.EnsureTmuxRunning()
}

// GetCurrentTmuxContext returns the current tmux context.
func (c *Core) GetCurrentTmuxContext() TmuxContext {
	ctx, err := c.client.GetCurrentContext()
	if err != nil {
		colors.Error("get current tmux context: failed to get tmux context: " + err.Error())
		return TmuxContext{}
	}

	// Convert tmux.TmuxContext to core.TmuxContext
	return TmuxContext{
		SessionID:   ctx.SessionID,
		WindowID:    ctx.WindowID,
		PaneID:      ctx.PaneID,
		PaneCreated: ctx.PanePID,
	}
}

// GetCurrentTmuxContext returns the current tmux context using the default client.
func GetCurrentTmuxContext() TmuxContext {
	return defaultCore.GetCurrentTmuxContext()
}

// ValidatePaneExists checks if a pane exists.
func (c *Core) ValidatePaneExists(sessionID, windowID, paneID string) bool {
	exists, err := c.client.ValidatePaneExists(sessionID, windowID, paneID)
	if err != nil {
		colors.Debug(fmt.Sprintf("ValidatePaneExists: tmux list-panes failed for %s:%s.%s: %v", sessionID, windowID, paneID, err))
		return false
	}
	return exists
}

// ValidatePaneExists checks if a pane exists using the default client.
func ValidatePaneExists(sessionID, windowID, paneID string) bool {
	return defaultCore.ValidatePaneExists(sessionID, windowID, paneID)
}

// jumpToPane is the private implementation that accepts a custom error handler.
func (c *Core) jumpToPane(sessionID, windowID, paneID string, handler errors.ErrorHandler) bool {
	// ASSERTION: Validate input parameters are non-empty
	if sessionID == "" || windowID == "" || paneID == "" {
		var missing []string
		if sessionID == "" {
			missing = append(missing, "sessionID")
		}
		if windowID == "" {
			missing = append(missing, "windowID")
		}
		if paneID == "" {
			missing = append(missing, "paneID")
		}
		handler.Error(fmt.Sprintf("jump to pane: invalid parameters (empty %s)", strings.Join(missing, ", ")))
		return false
	}

	shouldSwitch := true
	currentCtx, err := c.client.GetCurrentContext()
	if err != nil {
		colors.Debug("JumpToPane: failed to get tmux context: " + err.Error())
	} else if currentCtx.SessionID == sessionID {
		shouldSwitch = false
	}
	var stderr string

	// First validate if the pane exists
	paneExists := c.ValidatePaneExists(sessionID, windowID, paneID)
	colors.Debug(fmt.Sprintf("JumpToPane: pane validation for %s:%s in window %s:%s - exists: %v", sessionID, paneID, sessionID, windowID, paneExists))

	// Switch client to target session before selecting window/pane
	if shouldSwitch {
		colors.Debug(fmt.Sprintf("JumpToPane: switching client to session %s", sessionID))
		_, stderr, err = c.client.Run("switch-client", "-t", sessionID)
		if err != nil {
			handler.Error(fmt.Sprintf("jump to pane: failed to switch client to session %s: %v", sessionID, err))
			if stderr != "" {
				colors.Debug("JumpToPane: stderr: " + stderr)
			}
			return false
		}
	}

	// Select the window (this happens regardless of whether the pane exists)
	targetWindow := sessionID + ":" + windowID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting window %s", targetWindow))
	_, stderr, err = c.client.Run("select-window", "-t", targetWindow)
	if err != nil {
		handler.Error(fmt.Sprintf("jump to pane: failed to select window %s: %v", targetWindow, err))
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false
	}

	// If pane doesn't exist, show warning and fall back to window selection
	if !paneExists {
		handler.Warning("Pane " + paneID + " does not exist in window " + targetWindow + ", jumping to window instead")
		colors.Debug(fmt.Sprintf("JumpToPane: falling back to window selection (pane %s not found)", paneID))
		return true
	}

	// Pane exists, select it using the correct tmux pane syntax: "sessionID:windowID.paneID"
	// ASSERTION: targetPane must follow tmux pane reference format
	targetPane := sessionID + ":" + windowID + "." + paneID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting pane %s", targetPane))
	_, stderr, err = c.client.Run("select-pane", "-t", targetPane)
	if err != nil {
		// Fail-fast: don't swallow errors, return false to indicate failure
		handler.Error(fmt.Sprintf("jump to pane: failed to select pane %s: %v", targetPane, err))
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false
	}

	colors.Debug(fmt.Sprintf("JumpToPane: successfully selected pane %s", targetPane))
	return true
}

// JumpToPane jumps to a specific pane. It returns true if the jump succeeded
// (either to the pane or fallback to window), false if the jump completely failed.
// Preconditions: sessionID, windowID, and paneID must be non-empty.
func (c *Core) JumpToPane(sessionID, windowID, paneID string) bool {
	return c.jumpToPane(sessionID, windowID, paneID, defaultCLIHandler)
}

// JumpToPane jumps to a specific pane using the default client.
func JumpToPane(sessionID, windowID, paneID string) bool {
	return defaultCore.JumpToPane(sessionID, windowID, paneID)
}

// JumpToPaneWithHandler jumps to a specific pane using the default client with a custom error handler.
// This allows TUI and other UIs to handle errors differently than the CLI.
func JumpToPaneWithHandler(sessionID, windowID, paneID string, handler errors.ErrorHandler) bool {
	return defaultCore.jumpToPane(sessionID, windowID, paneID, handler)
}

// GetTmuxVisibility returns the value of TMUX_INTRAY_VISIBLE global tmux variable.
// Returns "0" if variable is not set.
func (c *Core) GetTmuxVisibility() string {
	value, err := c.client.GetEnvironment("TMUX_INTRAY_VISIBLE")
	if err != nil {
		colors.Debug("GetTmuxVisibility: tmux show-environment failed: " + err.Error())
		return "0"
	}
	return value
}

// GetTmuxVisibility returns the value of TMUX_INTRAY_VISIBLE global tmux variable using the default client.
func GetTmuxVisibility() string {
	return defaultCore.GetTmuxVisibility()
}

// SetTmuxVisibility sets the TMUX_INTRAY_VISIBLE global tmux variable.
// Returns (true, nil) on success, (false, error) on failure.
func (c *Core) SetTmuxVisibility(value string) (bool, error) {
	err := c.client.SetEnvironment("TMUX_INTRAY_VISIBLE", value)
	if err != nil {
		colors.Error(fmt.Sprintf("set tmux visibility: failed to set TMUX_INTRAY_VISIBLE to '%s': %v", value, err))
		return false, fmt.Errorf("set tmux visibility: %w", err)
	}
	return true, nil
}

// SetTmuxVisibility sets the TMUX_INTRAY_VISIBLE global tmux variable using the default client.
// Returns (true, nil) on success, (false, error) on failure.
func SetTmuxVisibility(value string) (bool, error) {
	return defaultCore.SetTmuxVisibility(value)
}
