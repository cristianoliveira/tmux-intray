// Package tmux provides a unified abstraction layer for tmux operations.
package tmux

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

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
