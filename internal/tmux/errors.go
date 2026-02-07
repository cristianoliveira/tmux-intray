// Package tmux provides a unified abstraction layer for tmux operations.
// It defines interfaces and types for interacting with tmux sessions, windows, and panes.
package tmux

import "errors"

// Custom error types for tmux-specific failures.
var (
	// ErrTmuxNotRunning is returned when tmux server is not available.
	ErrTmuxNotRunning = errors.New("tmux server is not running")

	// ErrSessionNotFound is returned when a tmux session cannot be found.
	ErrSessionNotFound = errors.New("tmux session not found")

	// ErrPaneNotFound is returned when a tmux pane cannot be found.
	ErrPaneNotFound = errors.New("tmux pane not found")

	// ErrInvalidTarget is returned when a tmux target specification is invalid.
	ErrInvalidTarget = errors.New("invalid tmux target specification")

	// ErrTmuxCommandFailed is returned when a tmux command execution fails.
	ErrTmuxCommandFailed = errors.New("tmux command failed")
)
