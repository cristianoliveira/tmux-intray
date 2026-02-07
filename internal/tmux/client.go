// Package tmux provides a unified abstraction layer for tmux operations.
// It defines interfaces and types for interacting with tmux sessions, windows, and panes.
package tmux

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

// TmuxContext captures the current tmux session/window/pane context.
type TmuxContext struct {
	SessionID string
	WindowID  string
	PaneID    string
	PanePID   string
}

// TmuxClient is an interface that abstracts all tmux operations.
type TmuxClient interface {
	// GetCurrentContext returns the current tmux session/window/pane context.
	GetCurrentContext() (TmuxContext, error)

	// ValidatePaneExists checks if a pane exists in a given session and window.
	ValidatePaneExists(sessionID, windowID, paneID string) (bool, error)

	// JumpToPane jumps to a specific pane. Returns true if successful.
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

	// GetTmuxVisibility gets the tmux visibility state from environment variable.
	GetTmuxVisibility() (bool, error)

	// SetTmuxVisibility sets the tmux visibility state via environment variable.
	SetTmuxVisibility(visible bool) error

	// Run executes a tmux command with the given arguments.
	Run(args ...string) (string, string, error)
}

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
	return stdout.String(), stderr.String(), err
}

// GetCurrentContext returns the current tmux session/window/pane context.
func (c *DefaultClient) GetCurrentContext() (TmuxContext, error) {
	format := "#{session_id} #{window_id} #{pane_id} #{pane_pid}"
	stdout, stderr, err := c.Run("display", "-p", format)
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return TmuxContext{}, fmt.Errorf("failed to get tmux context: %w", err)
	}

	// Trim whitespace from the output
	stdout = strings.TrimSpace(stdout)

	// Split by space - tmux format string produces 4 parts
	// Format: #{session_id} #{window_id} #{pane_id} #{pane_pid}
	parts := strings.Split(stdout, " ")

	// Filter out empty strings that might result from multiple spaces
	var filteredParts []string
	for _, part := range parts {
		if part != "" {
			filteredParts = append(filteredParts, part)
		}
	}

	// Validate that we have exactly 4 parts
	if len(filteredParts) != 4 {
		return TmuxContext{}, fmt.Errorf("unexpected format - expected 4 parts, got %d", len(filteredParts))
	}

	// Validate that captured context values are non-empty
	if filteredParts[0] == "" || filteredParts[1] == "" || filteredParts[2] == "" {
		return TmuxContext{}, fmt.Errorf("invalid context - session_id, window_id, or pane_id is empty")
	}

	ctx := TmuxContext{
		SessionID: filteredParts[0],
		WindowID:  filteredParts[1],
		PaneID:    filteredParts[2],
		PanePID:   filteredParts[3],
	}
	return ctx, nil
}

// ValidatePaneExists checks if a pane exists in a given session and window.
func (c *DefaultClient) ValidatePaneExists(sessionID, windowID, paneID string) (bool, error) {
	target := sessionID + ":" + windowID
	stdout, stderr, err := c.Run("list-panes", "-t", target, "-F", "#{pane_id}")
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false, fmt.Errorf("failed to list panes: %w", err)
	}

	// Each pane ID is on a separate line
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	// Trim paneID input in case it has whitespace
	paneID = strings.TrimSpace(paneID)
	for _, line := range lines {
		// Trim each pane ID from output in case of trailing whitespace
		if strings.TrimSpace(line) == paneID {
			return true, nil
		}
	}
	return false, nil
}

// JumpToPane jumps to a specific pane. Returns true if successful.
func (c *DefaultClient) JumpToPane(sessionID, windowID, paneID string) (bool, error) {
	// Validate input parameters are non-empty
	if sessionID == "" || windowID == "" || paneID == "" {
		return false, ErrInvalidTarget
	}

	// First validate if the pane exists
	paneExists, err := c.ValidatePaneExists(sessionID, windowID, paneID)
	if err != nil {
		return false, fmt.Errorf("pane validation failed: %w", err)
	}

	colors.Debug(fmt.Sprintf("JumpToPane: pane validation for %s:%s - exists: %v", sessionID, paneID, paneExists))

	// Select the window (this happens regardless of whether the pane exists)
	targetWindow := sessionID + ":" + windowID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting window %s", targetWindow))
	_, stderr, err := c.Run("select-window", "-t", targetWindow)
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false, fmt.Errorf("window %s does not exist: %w", targetWindow, err)
	}

	// If pane doesn't exist, show warning and fall back to window selection
	if !paneExists {
		colors.Warning("Pane " + paneID + " does not exist in window " + targetWindow + ", jumping to window instead")
		colors.Debug(fmt.Sprintf("JumpToPane: falling back to window selection (pane %s not found)", paneID))
		return true, nil
	}

	// Pane exists, select it using the correct tmux pane syntax: "sessionID:windowID.paneID"
	targetPane := sessionID + ":" + windowID + "." + paneID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting pane %s", targetPane))
	_, stderr, err = c.Run("select-pane", "-t", targetPane)
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false, fmt.Errorf("failed to select pane %s: %w", targetPane, err)
	}

	colors.Debug(fmt.Sprintf("JumpToPane: successfully selected pane %s", targetPane))
	return true, nil
}

// SetEnvironment sets a tmux environment variable.
func (c *DefaultClient) SetEnvironment(name, value string) error {
	_, stderr, err := c.Run("set-environment", "-g", name, value)
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return fmt.Errorf("failed to set environment variable %s: %w", name, err)
	}
	return nil
}

// GetEnvironment gets a tmux environment variable value.
func (c *DefaultClient) GetEnvironment(name string) (string, error) {
	stdout, stderr, err := c.Run("show-environment", "-g", name)
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return "", fmt.Errorf("failed to get environment variable %s: %w", name, err)
	}

	// Output could be "NAME=value" or empty line
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, name+"=") {
			return strings.TrimPrefix(line, name+"="), nil
		}
	}
	return "", fmt.Errorf("environment variable %s not found", name)
}

// HasSession checks if tmux server is running.
func (c *DefaultClient) HasSession() (bool, error) {
	_, stderr, err := c.Run("has-session")
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false, ErrTmuxNotRunning
	}
	return true, nil
}

// SetStatusOption sets a tmux status option.
func (c *DefaultClient) SetStatusOption(name, value string) error {
	// First check if tmux is running
	running, err := c.HasSession()
	if err != nil {
		return err
	}
	if !running {
		return ErrTmuxNotRunning
	}

	_, stderr, err := c.Run("set", "-g", name, value)
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return fmt.Errorf("failed to set status option %s: %w", name, err)
	}
	return nil
}

// ListSessions returns all tmux sessions as a map of session ID to name.
func (c *DefaultClient) ListSessions() (map[string]string, error) {
	stdout, stderr, err := c.Run("list-sessions", "-F", "#{session_id}\t#{session_name}")
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessions := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			sessions[parts[0]] = parts[1]
		}
	}
	return sessions, nil
}

// GetSessionName returns the name of a session by its ID.
func (c *DefaultClient) GetSessionName(sessionID string) (string, error) {
	stdout, stderr, err := c.Run("display-message", "-t", sessionID, "-p", "#S")
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return "", fmt.Errorf("get session name: %w", err)
	}

	// Trim whitespace from the output
	sessionName := strings.TrimSpace(stdout)

	// Validate that session name is non-empty
	if sessionName == "" {
		return "", ErrSessionNotFound
	}

	return sessionName, nil
}

// GetTmuxVisibility gets the tmux visibility state from environment variable.
func (c *DefaultClient) GetTmuxVisibility() (bool, error) {
	value, err := c.GetEnvironment("TMUX_INTRAY_VISIBLE")
	if err != nil {
		// Environment variable not set is not an error - return false
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, fmt.Errorf("get tmux visibility: %w", err)
	}

	// Parse boolean value from string
	return value == "true", nil
}

// SetTmuxVisibility sets the tmux visibility state via environment variable.
func (c *DefaultClient) SetTmuxVisibility(visible bool) error {
	var value string
	if visible {
		value = "true"
	} else {
		value = "false"
	}

	if err := c.SetEnvironment("TMUX_INTRAY_VISIBLE", value); err != nil {
		return fmt.Errorf("set tmux visibility: %w", err)
	}

	return nil
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
