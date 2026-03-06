package core

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/errors"
	"github.com/cristianoliveira/tmux-intray/internal/ports"
)

type tmuxJumpCoordinator struct {
	client  ports.TmuxClient
	runtime *tmuxRuntime
}

func newTmuxJumpCoordinator(client ports.TmuxClient) *tmuxJumpCoordinator {
	return &tmuxJumpCoordinator{
		client:  client,
		runtime: newTmuxRuntime(client),
	}
}

func (c *tmuxJumpCoordinator) jumpToPane(sessionID, windowID, paneID string, handler errors.ErrorHandler) bool {
	if !validateJumpParams(sessionID, windowID, handler) {
		return false
	}

	if c.shouldSwitchSession(sessionID) {
		if !c.switchToSession(sessionID, handler) {
			return false
		}
	}

	targetWindow := sessionID + ":" + windowID
	if !c.selectWindow(targetWindow, handler) {
		return false
	}

	if paneID == "" {
		colors.Debug(fmt.Sprintf("JumpToPane: explicit window jump to %s", targetWindow))
		return true
	}

	paneExists := c.runtime.validatePaneExists(sessionID, windowID, paneID)
	colors.Debug(fmt.Sprintf("JumpToPane: pane validation for %s:%s in window %s:%s - exists: %v", sessionID, paneID, sessionID, windowID, paneExists))

	if !paneExists {
		handler.Warning("Pane " + paneID + " does not exist in window " + targetWindow + ", jumping to window instead")
		colors.Debug(fmt.Sprintf("JumpToPane: falling back to window selection (pane %s not found)", paneID))
		return true
	}

	return c.selectPane(sessionID, windowID, paneID, handler)
}

func validateJumpParams(sessionID, windowID string, handler errors.ErrorHandler) bool {
	if sessionID == "" || windowID == "" {
		var missing []string
		if sessionID == "" {
			missing = append(missing, "sessionID")
		}
		if windowID == "" {
			missing = append(missing, "windowID")
		}
		handler.Error(fmt.Sprintf("jump: invalid parameters (empty %s)", strings.Join(missing, ", ")))
		return false
	}
	return true
}

func (c *tmuxJumpCoordinator) shouldSwitchSession(sessionID string) bool {
	currentCtx, err := c.client.GetCurrentContext()
	if err != nil {
		colors.Debug("JumpToPane: failed to get tmux context: " + err.Error())
		return true
	}
	return currentCtx.SessionID != sessionID
}

func (c *tmuxJumpCoordinator) switchToSession(sessionID string, handler errors.ErrorHandler) bool {
	colors.Debug(fmt.Sprintf("JumpToPane: switching client to session %s", sessionID))
	_, stderr, err := c.client.Run("switch-client", "-t", sessionID)
	if err != nil {
		handler.Error(fmt.Sprintf("jump to pane: failed to switch client to session %s: %v", sessionID, err))
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false
	}
	return true
}

func (c *tmuxJumpCoordinator) selectWindow(targetWindow string, handler errors.ErrorHandler) bool {
	colors.Debug(fmt.Sprintf("JumpToPane: selecting window %s", targetWindow))
	_, stderr, err := c.client.Run("select-window", "-t", targetWindow)
	if err != nil {
		handler.Error(fmt.Sprintf("jump to pane: failed to select window %s: %v", targetWindow, err))
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false
	}
	return true
}

func (c *tmuxJumpCoordinator) selectPane(sessionID, windowID, paneID string, handler errors.ErrorHandler) bool {
	targetPane := sessionID + ":" + windowID + "." + paneID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting pane %s", targetPane))
	_, stderr, err := c.client.Run("select-pane", "-t", targetPane)
	if err != nil {
		handler.Error(fmt.Sprintf("jump to pane: failed to select pane %s: %v", targetPane, err))
		if stderr != "" {
			colors.Debug("JumpToPane: stderr: " + stderr)
		}
		return false
	}

	colors.Debug(fmt.Sprintf("JumpToPane: successfully selected pane %s", targetPane))
	return true
}
