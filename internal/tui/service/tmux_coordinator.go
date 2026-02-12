// Package service provides implementations of TUI service interfaces.
package service

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/errors"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// DefaultRuntimeCoordinator implements the RuntimeCoordinator interface.
type DefaultRuntimeCoordinator struct {
	client       tmux.TmuxClient
	sessionNames map[string]string
	windowNames  map[string]string
	paneNames    map[string]string

	// Function pointers for testability
	ensureTmuxRunning func() bool
	jumpToPane        func(sessionID, windowID, paneID string) bool

	// errorHandler is used for jump operations. If nil, uses default CLI handler.
	errorHandler errors.ErrorHandler
}

// NewRuntimeCoordinator creates a new DefaultRuntimeCoordinator.
func NewRuntimeCoordinator(client tmux.TmuxClient) model.RuntimeCoordinator {
	if client == nil {
		client = tmux.NewDefaultClient()
	}

	coordinator := &DefaultRuntimeCoordinator{
		client:            client,
		sessionNames:      make(map[string]string),
		windowNames:       make(map[string]string),
		paneNames:         make(map[string]string),
		ensureTmuxRunning: core.EnsureTmuxRunning,
		jumpToPane:        core.JumpToPane,
	}

	// Initialize name caches
	coordinator.RefreshNames()

	return coordinator
}

// EnsureTmuxRunning ensures the tmux server is running.
func (c *DefaultRuntimeCoordinator) EnsureTmuxRunning() bool {
	if c.ensureTmuxRunning == nil {
		return core.EnsureTmuxRunning()
	}
	return c.ensureTmuxRunning()
}

// SetErrorHandler sets the error handler for jump operations.
// If set, the error handler will be used instead of the default CLI handler.
func (c *DefaultRuntimeCoordinator) SetErrorHandler(handler errors.ErrorHandler) {
	c.errorHandler = handler
}

// JumpToPane jumps to a specific pane in tmux.
func (c *DefaultRuntimeCoordinator) JumpToPane(sessionID, windowID, paneID string) bool {
	if c.jumpToPane != nil {
		return c.jumpToPane(sessionID, windowID, paneID)
	}
	// If an error handler is set, use JumpToPaneWithHandler
	if c.errorHandler != nil {
		return core.JumpToPaneWithHandler(sessionID, windowID, paneID, c.errorHandler)
	}
	return core.JumpToPane(sessionID, windowID, paneID)
}

// ValidatePaneExists checks if a pane exists in the specified session and window.
func (c *DefaultRuntimeCoordinator) ValidatePaneExists(sessionID, windowID, paneID string) (bool, error) {
	if sessionID == "" || windowID == "" || paneID == "" {
		return false, fmt.Errorf("missing required parameters")
	}

	// Check if pane exists in name cache
	if _, ok := c.paneNames[paneID]; !ok {
		return false, fmt.Errorf("pane %s not found", paneID)
	}

	return true, nil
}

// GetCurrentContext returns the current tmux context.
func (c *DefaultRuntimeCoordinator) GetCurrentContext() (*model.TmuxContext, error) {
	ctx := core.GetCurrentTmuxContext()
	return &model.TmuxContext{
		SessionID:   ctx.SessionID,
		WindowID:    ctx.WindowID,
		PaneID:      ctx.PaneID,
		SessionName: c.ResolveSessionName(ctx.SessionID),
		WindowName:  c.ResolveWindowName(ctx.WindowID),
		PaneName:    c.ResolvePaneName(ctx.PaneID),
	}, nil
}

// ListSessions returns a map of session IDs to session names.
func (c *DefaultRuntimeCoordinator) ListSessions() (map[string]string, error) {
	sessions, err := c.client.ListSessions()
	if err != nil {
		return nil, err
	}
	c.sessionNames = sessions
	return sessions, nil
}

// ListWindows returns a map of window IDs to window names.
func (c *DefaultRuntimeCoordinator) ListWindows() (map[string]string, error) {
	windows, err := c.client.ListWindows()
	if err != nil {
		return nil, err
	}
	c.windowNames = windows
	return windows, nil
}

// ListPanes returns a map of pane IDs to pane names.
func (c *DefaultRuntimeCoordinator) ListPanes() (map[string]string, error) {
	panes, err := c.client.ListPanes()
	if err != nil {
		return nil, err
	}
	c.paneNames = panes
	return panes, nil
}

// GetSessionName returns the name of a session by its ID.
func (c *DefaultRuntimeCoordinator) GetSessionName(sessionID string) (string, error) {
	if name, ok := c.sessionNames[sessionID]; ok {
		return name, nil
	}
	return sessionID, nil
}

// GetWindowName returns the name of a window by its ID.
func (c *DefaultRuntimeCoordinator) GetWindowName(windowID string) (string, error) {
	if name, ok := c.windowNames[windowID]; ok {
		return name, nil
	}
	return windowID, nil
}

// GetPaneName returns the name of a pane by its ID.
func (c *DefaultRuntimeCoordinator) GetPaneName(paneID string) (string, error) {
	if name, ok := c.paneNames[paneID]; ok {
		return name, nil
	}
	return paneID, nil
}

// RefreshNames refreshes cached session, window, and pane names.
func (c *DefaultRuntimeCoordinator) RefreshNames() error {
	var err error

	// Refresh session names
	c.sessionNames, err = c.client.ListSessions()
	if err != nil {
		c.sessionNames = make(map[string]string)
	}

	// Refresh window names
	c.windowNames, err = c.client.ListWindows()
	if err != nil {
		c.windowNames = make(map[string]string)
	}

	// Refresh pane names
	c.paneNames, err = c.client.ListPanes()
	if err != nil {
		c.paneNames = make(map[string]string)
	}

	return nil
}

// GetTmuxVisibility returns the visibility state from tmux environment.
func (c *DefaultRuntimeCoordinator) GetTmuxVisibility() (bool, error) {
	value := core.GetTmuxVisibility()
	return value == "1", nil
}

// SetTmuxVisibility sets the visibility state in tmux environment.
func (c *DefaultRuntimeCoordinator) SetTmuxVisibility(visible bool) error {
	value := "0"
	if visible {
		value = "1"
	}
	_, err := core.SetTmuxVisibility(value)
	return err
}

// GetSessionNames returns the full map of session ID to name.
func (c *DefaultRuntimeCoordinator) GetSessionNames() map[string]string {
	return c.sessionNames
}

// GetWindowNames returns the full map of window ID to name.
func (c *DefaultRuntimeCoordinator) GetWindowNames() map[string]string {
	return c.windowNames
}

// GetPaneNames returns the full map of pane ID to name.
func (c *DefaultRuntimeCoordinator) GetPaneNames() map[string]string {
	return c.paneNames
}

// SetSessionNames sets the session name map.
func (c *DefaultRuntimeCoordinator) SetSessionNames(names map[string]string) {
	c.sessionNames = names
}

// SetWindowNames sets the window name map.
func (c *DefaultRuntimeCoordinator) SetWindowNames(names map[string]string) {
	c.windowNames = names
}

// SetPaneNames sets the pane name map.
func (c *DefaultRuntimeCoordinator) SetPaneNames(names map[string]string) {
	c.paneNames = names
}

// ResolveSessionName converts a session ID to a name.
func (c *DefaultRuntimeCoordinator) ResolveSessionName(sessionID string) string {
	if name, ok := c.sessionNames[sessionID]; ok {
		return name
	}
	return sessionID
}

// ResolveWindowName converts a window ID to a name.
func (c *DefaultRuntimeCoordinator) ResolveWindowName(windowID string) string {
	if name, ok := c.windowNames[windowID]; ok {
		return name
	}
	return windowID
}

// ResolvePaneName converts a pane ID to a name.
func (c *DefaultRuntimeCoordinator) ResolvePaneName(paneID string) string {
	if name, ok := c.paneNames[paneID]; ok {
		return name
	}
	return paneID
}
