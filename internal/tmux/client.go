// Package tmux provides a unified abstraction layer for tmux operations.
// It defines interfaces and types for interacting with tmux sessions, windows, and panes.
package tmux

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/errors"
	"github.com/cristianoliveira/tmux-intray/internal/ports"
)

// TmuxContext captures the current tmux session/window/pane context.
type TmuxContext = ports.TmuxContext

// TmuxClient is an interface that abstracts all tmux operations.
type TmuxClient interface {
	// GetCurrentContext returns the current tmux session/window/pane context.
	GetCurrentContext() (TmuxContext, error)

	// ValidatePaneExists checks if a pane exists in a given session and window.
	ValidatePaneExists(sessionID, windowID, paneID string) (bool, error)

	// JumpToPane jumps to the specified pane or window.
	// If paneID is empty, jumps to the window only.
	// Returns true if jump succeeded, false if failed.
	JumpToPane(sessionID, windowID, paneID string) (bool, error)

	// SetEnvironment sets a tmux environment variable.
	SetEnvironment(name, value string) error

	// GetEnvironment gets a tmux environment variable value.
	GetEnvironment(name string) (string, error)

	// HasSession checks if tmux server is running.
	HasSession() (bool, error)

	// SetStatusOption sets a tmux status option.
	SetStatusOption(name, value string) error

	// ListSessions returns all tmux sessions as a map of session ID to name.
	ListSessions() (map[string]string, error)

	// GetSessionName returns the name of a session by its ID.
	GetSessionName(sessionID string) (string, error)

	// ListWindows returns all tmux windows as a map of window ID to name.
	ListWindows() (map[string]string, error)

	// ListPanes returns all tmux panes as a map of pane ID to name.
	ListPanes() (map[string]string, error)

	// GetTmuxVisibility gets the tmux visibility state from environment variable.
	GetTmuxVisibility() (bool, error)

	// SetTmuxVisibility sets the tmux visibility state via environment variable.
	SetTmuxVisibility(visible bool) error

	// Run executes a tmux command with the given arguments.
	Run(args ...string) (string, string, error)
}

// defaultCLIHandler is the default CLI error handler for backward compatibility.
var defaultCLIHandler = errors.NewDefaultCLIHandler()

// DefaultClient implements TmuxClient using exec.Command to run tmux.
type DefaultClient struct {
	socketPath string
	timeout    time.Duration
}

// NewDefaultClient creates a new DefaultClient with the given options.
func NewDefaultClient(opts ...ClientOption) *DefaultClient {
	client := &DefaultClient{
		timeout: DefaultTimeout,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// runCommand executes a tmux command with the given arguments.
// It returns stdout, stderr, and any error that occurred.
func (c *DefaultClient) runCommand(args ...string) (string, string, error) {
	start := time.Now()
	command := ""
	if len(args) > 0 {
		command = args[0]
	}
	colors.StructuredDebug("tmux", "run", "started", nil, command, map[string]interface{}{"args_count": len(args)})
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	cmdArgs := []string{}
	if c.socketPath != "" {
		cmdArgs = append(cmdArgs, "-L", c.socketPath)
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "tmux", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start).Seconds()
	if err != nil {
		colors.StructuredError("tmux", "run", "failed", err, command, map[string]interface{}{"args_count": len(args), "duration_seconds": duration})
	} else {
		colors.StructuredDebug("tmux", "run", "completed", nil, command, map[string]interface{}{"args_count": len(args), "duration_seconds": duration})
	}
	return stdout.String(), stderr.String(), err
}

// Run executes a tmux command with the given arguments.
// It returns stdout, stderr, and any error that occurred.
func (c *DefaultClient) Run(args ...string) (string, string, error) {
	stdout, stderr, err := c.runCommand(args...)
	if err != nil {
		return stdout, stderr, fmt.Errorf("tmux command %v failed: %w", args, err)
	}
	return stdout, stderr, nil
}
