// Package core provides core tmux interaction and tray management.
package core

import (
	"fmt"

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
	return c.tmuxRuntime().ensureTmuxRunning()
}

// EnsureTmuxRunning verifies that tmux is running using the default client.
func EnsureTmuxRunning() bool {
	return defaultCore.EnsureTmuxRunning()
}

// GetCurrentTmuxContext returns the current tmux context.
func (c *Core) GetCurrentTmuxContext() TmuxContext {
	return c.tmuxRuntime().currentContext()
}

// GetCurrentTmuxContext returns the current tmux context using the default client.
func GetCurrentTmuxContext() TmuxContext {
	return defaultCore.GetCurrentTmuxContext()
}

// ValidatePaneExists checks if a pane exists.
func (c *Core) ValidatePaneExists(sessionID, windowID, paneID string) bool {
	return c.tmuxRuntime().validatePaneExists(sessionID, windowID, paneID)
}

// ValidatePaneExists checks if a pane exists using the default client.
func ValidatePaneExists(sessionID, windowID, paneID string) bool {
	return defaultCore.ValidatePaneExists(sessionID, windowID, paneID)
}

// jumpToPane is the private implementation that accepts a custom error handler.
// sessionID and windowID are required; paneID is optional for explicit window jump.
func (c *Core) jumpToPane(sessionID, windowID, paneID string, handler errors.ErrorHandler) bool {
	coordinator := newTmuxJumpCoordinator(c.client)
	return coordinator.jumpToPane(sessionID, windowID, paneID, handler)
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
	return c.tmuxRuntime().getVisibility()
}

// GetTmuxVisibility returns the value of TMUX_INTRAY_VISIBLE global tmux variable using the default client.
func GetTmuxVisibility() string {
	return defaultCore.GetTmuxVisibility()
}

// SetTmuxVisibility sets the TMUX_INTRAY_VISIBLE global tmux variable.
// Returns (true, nil) on success, (false, error) on failure.
func (c *Core) SetTmuxVisibility(value string) (bool, error) {
	return c.tmuxRuntime().setVisibility(value)
}

// SetTmuxVisibility sets the TMUX_INTRAY_VISIBLE global tmux variable using the default client.
// Returns (true, nil) on success, (false, error) on failure.
func SetTmuxVisibility(value string) (bool, error) {
	return defaultCore.SetTmuxVisibility(value)
}

func (c *Core) tmuxRuntime() *tmuxRuntime {
	return newTmuxRuntime(c.client)
}
