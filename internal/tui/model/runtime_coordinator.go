// Package model provides interface contracts for TUI components.
// These interfaces define the contracts between different parts of the TUI system.
package model

// RuntimeCoordinator defines the interface for tmux integration and coordination.
// It handles communication between the TUI and tmux server.
type RuntimeCoordinator interface {
	// EnsureTmuxRunning ensures the tmux server is running.
	// Returns true if tmux is running, false otherwise.
	EnsureTmuxRunning() bool

	// JumpToPane jumps to a specific pane in tmux.
	// Requires session ID and window ID; pane ID is optional.
	// Returns true on success, false on failure.
	JumpToPane(sessionID, windowID, paneID string) bool

	// JumpToWindow jumps to a specific window in tmux.
	// Requires session ID and window ID.
	// Returns true on success, false on failure.
	JumpToWindow(sessionID, windowID string) bool

	// ValidatePaneExists checks if a pane exists in the specified session and window.
	// Returns true if the pane exists, false otherwise.
	ValidatePaneExists(sessionID, windowID, paneID string) (bool, error)

	// GetCurrentContext returns the current tmux context (session, window, pane).
	// Returns an error if not running in a tmux session.
	GetCurrentContext() (*TmuxContext, error)

	// ListSessions returns a map of session IDs to session names.
	ListSessions() (map[string]string, error)

	// ListWindows returns a map of window IDs to window names.
	ListWindows() (map[string]string, error)

	// ListPanes returns a map of pane IDs to pane names.
	ListPanes() (map[string]string, error)

	// GetSessionName returns the name of a session by its ID.
	GetSessionName(sessionID string) (string, error)

	// GetWindowName returns the name of a window by its ID.
	GetWindowName(windowID string) (string, error)

	// GetPaneName returns the name of a pane by its ID.
	GetPaneName(paneID string) (string, error)

	// RefreshNames refreshes cached session, window, and pane names.
	// Call this periodically to keep names up to date.
	RefreshNames() error

	// GetTmuxVisibility returns the visibility state from tmux environment.
	// Returns true if visible, false otherwise.
	GetTmuxVisibility() (bool, error)

	// SetTmuxVisibility sets the visibility state in tmux environment.
	SetTmuxVisibility(visible bool) error

	// NameResolver interface methods (embedded for convenience)
	NameResolver
}

// TmuxContext represents the current tmux session/window/pane context.
type TmuxContext struct {
	// SessionID is the unique identifier for the session.
	SessionID string

	// SessionName is the human-readable session name.
	SessionName string

	// WindowID is the unique identifier for the window.
	WindowID string

	// WindowName is the human-readable window name.
	WindowName string

	// PaneID is the unique identifier for the pane.
	PaneID string

	// PaneName is the human-readable pane title.
	PaneName string

	// PanePID is the process ID of the pane's process.
	PanePID string
}

// NameResolver provides name resolution for tmux entities.
type NameResolver interface {
	// ResolveSessionName converts a session ID to a name.
	ResolveSessionName(sessionID string) string

	// ResolveWindowName converts a window ID to a name.
	ResolveWindowName(windowID string) string

	// ResolvePaneName converts a pane ID to a name.
	ResolvePaneName(paneID string) string

	// GetSessionNames returns the full map of session ID to name.
	GetSessionNames() map[string]string

	// GetWindowNames returns the full map of window ID to name.
	GetWindowNames() map[string]string

	// GetPaneNames returns the full map of pane ID to name.
	GetPaneNames() map[string]string

	// SetSessionNames sets the session name map.
	SetSessionNames(names map[string]string)

	// SetWindowNames sets the window name map.
	SetWindowNames(names map[string]string)

	// SetPaneNames sets the pane name map.
	SetPaneNames(names map[string]string)
}
