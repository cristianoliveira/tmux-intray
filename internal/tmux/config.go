// Package tmux provides a unified abstraction layer for tmux operations.
// It defines interfaces and types for interacting with tmux sessions, windows, and panes.
package tmux

import "time"

const (
	// DefaultTimeout is the default timeout for tmux commands.
	DefaultTimeout = 5 * time.Second
)

// ClientOption is a functional option for configuring a TmuxClient.
type ClientOption func(*DefaultClient)

// WithSocketPath sets the tmux socket path for the client.
func WithSocketPath(socketPath string) ClientOption {
	return func(c *DefaultClient) {
		c.socketPath = socketPath
	}
}

// WithTimeout sets the timeout for tmux command execution.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *DefaultClient) {
		c.timeout = timeout
	}
}
