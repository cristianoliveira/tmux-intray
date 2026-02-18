// Package tmux provides a unified abstraction layer for tmux operations.
package tmux

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/errors"
)

// jumpToPane is the private implementation that accepts a custom error handler.
func (c *DefaultClient) jumpToPane(sessionID, windowID, paneID string, handler errors.ErrorHandler) (ok bool, err error) {
	fields := map[string]interface{}{
		"session_id": sessionID,
		"window_id":  windowID,
		"pane_id":    paneID,
	}
	colors.StructuredInfo("tmux", "jump", "started", nil, "", fields)
	defer func() {
		if err != nil {
			colors.StructuredError("tmux", "jump", "failed", err, "", fields)
			return
		}
		colors.StructuredInfo("tmux", "jump", "completed", nil, "", fields)
	}()

	// Validate required fields (sessionID and windowID)
	if sessionID == "" || windowID == "" {
		return false, ErrInvalidTarget
	}

	currentCtx, err := c.GetCurrentContext()
	if err != nil {
		colors.Debug("JumpToPane: failed to get current tmux context: " + err.Error())
	} else if currentCtx.SessionID != sessionID {
		colors.Debug(fmt.Sprintf("JumpToPane: switching client to session %s", sessionID))
		_, stderr, err := c.Run("switch-client", "-t", sessionID)
		if err != nil {
			if stderr != "" {
				colors.Debug("JumpToPane: stderr: " + stderr)
			}
			return false, fmt.Errorf("switch client to session %s: %w", sessionID, err)
		}
	}

	// paneID is optional - if empty, jump to window only
	if paneID == "" {
		return c.jumpToWindowOnly(sessionID, windowID)
	}

	return c.jumpToPaneWithValidation(sessionID, windowID, paneID, handler)
}

// jumpToWindowOnly switches to the specified session and selects the window.
func (c *DefaultClient) jumpToWindowOnly(sessionID, windowID string) (bool, error) {
	colors.Debug(fmt.Sprintf("JumpToPane: switching client to session %s (window-only jump)", sessionID))
	_, stderr, err := c.Run("switch-client", "-t", sessionID)
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false, fmt.Errorf("switch client to session %s: %w", sessionID, err)
	}

	targetWindow := sessionID + ":" + windowID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting window %s (window-only jump)", targetWindow))
	_, stderr, err = c.Run("select-window", "-t", targetWindow)
	if err != nil {
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false, fmt.Errorf("window %s does not exist: %w", targetWindow, err)
	}
	colors.Debug(fmt.Sprintf("JumpToPane: successfully selected window %s", targetWindow))
	return true, nil
}

// jumpToPaneWithValidation switches to the specified session, window, and pane after validation.
func (c *DefaultClient) jumpToPaneWithValidation(sessionID, windowID, paneID string, handler errors.ErrorHandler) (bool, error) {
	paneExists, err := c.ValidatePaneExists(sessionID, windowID, paneID)
	if err != nil {
		return false, fmt.Errorf("pane validation failed: %w", err)
	}

	colors.Debug(fmt.Sprintf("JumpToPane: pane validation for %s:%s - exists: %v", sessionID, windowID, paneExists))

	// Switch client to target session before selecting window/pane
	colors.Debug(fmt.Sprintf("JumpToPane: switching client to session %s", sessionID))
	_, stderr, err := c.Run("switch-client", "-t", sessionID)
	if err != nil {
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false, fmt.Errorf("switch client to session %s: %w", sessionID, err)
	}

	// Select the window first
	targetWindow := sessionID + ":" + windowID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting window %s", targetWindow))
	_, stderr, err = c.Run("select-window", "-t", targetWindow)
	if err != nil {
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false, fmt.Errorf("window %s does not exist: %w", targetWindow, err)
	}

	// If pane doesn't exist, show warning and fall back to window selection
	if !paneExists {
		handler.Warning("Pane " + paneID + " does not exist in window " + targetWindow + ", jumping to window instead")
		colors.Debug(fmt.Sprintf("JumpToPane: falling back to window selection (pane %s not found)", paneID))
		return true, nil
	}

	// Pane exists, select it using the correct tmux pane syntax: "sessionID:windowID.paneID"
	targetPane := sessionID + ":" + windowID + "." + paneID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting pane %s", targetPane))
	_, stderr, err = c.Run("select-pane", "-t", targetPane)
	if err != nil {
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false, fmt.Errorf("failed to select pane %s: %w", targetPane, err)
	}

	colors.Debug(fmt.Sprintf("JumpToPane: successfully selected pane %s", targetPane))
	return true, nil
}

// JumpToPane jumps to the specified pane or window.
// If paneID is empty, jumps to the window only.
// Returns true if jump succeeded, false if failed.
// Preconditions: sessionID and windowID must be non-empty; paneID is optional.
func (c *DefaultClient) JumpToPane(sessionID, windowID, paneID string) (bool, error) {
	return c.jumpToPane(sessionID, windowID, paneID, defaultCLIHandler)
}

// JumpToPaneWithHandler jumps to the specified pane or window with a custom error handler.
// This allows TUI and other UIs to handle errors differently than the CLI.
// If paneID is empty, jumps to the window only.
// Returns true if jump succeeded, false if failed.
// Preconditions: sessionID and windowID must be non-empty; paneID is optional.
func (c *DefaultClient) JumpToPaneWithHandler(sessionID, windowID, paneID string, handler errors.ErrorHandler) (bool, error) {
	return c.jumpToPane(sessionID, windowID, paneID, handler)
}
