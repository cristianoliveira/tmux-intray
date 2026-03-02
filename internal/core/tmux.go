// Package core provides core tmux interaction and tray management.
package core

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/errors"
	"github.com/cristianoliveira/tmux-intray/internal/ports"
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
	client   ports.TmuxClient
	storage  ports.NotificationRepository
	settings ports.SettingsStore
}

// NewCoreWithDeps creates a new Core instance with injected dependencies.
// If dependencies are nil, default implementations are used.
// Panics if storage initialization fails, which is safer than continuing with nil storage.
func NewCoreWithDeps(client ports.TmuxClient, stor ports.NotificationRepository, settingsStore ports.SettingsStore) *Core {
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
	if settingsStore == nil {
		settingsStore = defaultSettingsStore{}
	}
	return &Core{client: client, storage: stor, settings: settingsStore}
}

// NewCore creates a new Core instance with backward-compatible defaults.
func NewCore(client ports.TmuxClient, stor ports.NotificationRepository) *Core {
	return NewCoreWithDeps(client, stor, nil)
}

// defaultCore is the default instance for backward compatibility.
var defaultCore = NewCore(nil, nil)

// Default returns the default Core instance for backward compatibility.
func Default() *Core {
	return defaultCore
}


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
// sessionID and windowID are required; paneID is optional for explicit window jump.
func (c *Core) jumpToPane(sessionID, windowID, paneID string, handler errors.ErrorHandler) bool {
	if !validateJumpParams(sessionID, windowID, paneID, handler) {
		return false
	}

	shouldSwitch := c.shouldSwitchSession(sessionID)

	if shouldSwitch {
		if !c.switchToSession(sessionID, handler) {
			return false
		}
	}

	targetWindow := sessionID + ":" + windowID
	if !c.selectWindow(targetWindow, handler) {
		return false
	}

	if paneID == "" {
		colors.Debug(fmt.Sprintf("JumpToPane: explicit window jump to %s", targetWindow))
		return true
	}

	paneExists := c.ValidatePaneExists(sessionID, windowID, paneID)
	colors.Debug(fmt.Sprintf("JumpToPane: pane validation for %s:%s in window %s:%s - exists: %v", sessionID, paneID, sessionID, windowID, paneExists))

	if !paneExists {
		handler.Warning("Pane " + paneID + " does not exist in window " + targetWindow + ", jumping to window instead")
		colors.Debug(fmt.Sprintf("JumpToPane: falling back to window selection (pane %s not found)", paneID))
		return true
	}

	return c.selectPane(sessionID, windowID, paneID, handler)
}

// validateJumpParams validates the jump parameters.
func validateJumpParams(sessionID, windowID, paneID string, handler errors.ErrorHandler) bool {
	if sessionID == "" || windowID == "" {
		var missing []string
		if sessionID == "" {
			missing = append(missing, "sessionID")
		}
		if windowID == "" {
			missing = append(missing, "windowID")
		}
		handler.Error(fmt.Sprintf("jump: invalid parameters (empty %s)", strings.Join(missing, ", ")))
		return false
	}
	return true
}

// shouldSwitchSession determines if we need to switch to the target session.
func (c *Core) shouldSwitchSession(sessionID string) bool {
	currentCtx, err := c.client.GetCurrentContext()
	if err != nil {
		colors.Debug("JumpToPane: failed to get tmux context: " + err.Error())
		return true
	}
	return currentCtx.SessionID != sessionID
}

// switchToSession switches to the target session.
func (c *Core) switchToSession(sessionID string, handler errors.ErrorHandler) bool {
	colors.Debug(fmt.Sprintf("JumpToPane: switching client to session %s", sessionID))
	_, stderr, err := c.client.Run("switch-client", "-t", sessionID)
	if err != nil {
		handler.Error(fmt.Sprintf("jump to pane: failed to switch client to session %s: %v", sessionID, err))
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false
	}
	return true
}

// selectWindow selects the target window.
func (c *Core) selectWindow(targetWindow string, handler errors.ErrorHandler) bool {
	colors.Debug(fmt.Sprintf("JumpToPane: selecting window %s", targetWindow))
	_, stderr, err := c.client.Run("select-window", "-t", targetWindow)
	if err != nil {
		handler.Error(fmt.Sprintf("jump to pane: failed to select window %s: %v", targetWindow, err))
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false
	}
	return true
}

// selectPane selects the target pane.
func (c *Core) selectPane(sessionID, windowID, paneID string, handler errors.ErrorHandler) bool {
	targetPane := sessionID + ":" + windowID + "." + paneID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting pane %s", targetPane))
	_, stderr, err := c.client.Run("select-pane", "-t", targetPane)
	if err != nil {
		handler.Error(fmt.Sprintf("jump to pane: failed to select pane %s: %v", targetPane, err))
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false
	}

	colors.Debug(fmt.Sprintf("JumpToPane: successfully selected pane %s", targetPane))
	return true
}

// JumpToPane jumps to a specific pane. If paneID is empty, performs an explicit
// window jump to sessionID:windowID. It returns true if the jump succeeded
// (either to the pane or fallback to window), false if the jump completely failed.
// Preconditions: sessionID and windowID must be non-empty.
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
