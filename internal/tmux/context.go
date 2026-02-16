// Package tmux provides a unified abstraction layer for tmux operations.
package tmux

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

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
