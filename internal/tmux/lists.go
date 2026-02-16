// Package tmux provides a unified abstraction layer for tmux operations.
package tmux

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

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

// ListWindows returns all tmux windows as a map of window ID to name.
func (c *DefaultClient) ListWindows() (map[string]string, error) {
	stdout, stderr, err := c.Run("list-windows", "-a", "-F", "#{window_id}\t#{window_name}")
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}

	windows := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			windows[parts[0]] = parts[1]
		}
	}
	return windows, nil
}

// ListPanes returns all tmux panes as a map of pane ID to name.
func (c *DefaultClient) ListPanes() (map[string]string, error) {
	stdout, stderr, err := c.Run("list-panes", "-a", "-F", "#{pane_id}\t#{pane_title}")
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return nil, fmt.Errorf("failed to list panes: %w", err)
	}

	panes := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			panes[parts[0]] = parts[1]
		}
	}
	return panes, nil
}
