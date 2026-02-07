// Package core provides core tmux interaction and tray management.
package core

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

// TmuxContext captures the current tmux session/window/pane context.
type TmuxContext struct {
	SessionID   string
	WindowID    string
	PaneID      string
	PaneCreated string
}

// tmuxRunner is a variable that can be replaced for testing.
var tmuxRunner = func(args ...string) (string, string, error) {
	cmd := exec.Command("tmux", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// EnsureTmuxRunning verifies that tmux is running.
func EnsureTmuxRunning() bool {
	_, stderr, err := tmuxRunner("has-session")
	if err != nil {
		colors.Debug("tmux has-session failed: " + err.Error())
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false
	}
	return true
}

// GetCurrentTmuxContext returns the current tmux context.
func GetCurrentTmuxContext() TmuxContext {
	format := "#{session_id} #{window_id} #{pane_id} #{pane_pid}"
	stdout, stderr, err := tmuxRunner("display", "-p", format)
	if err != nil {
		colors.Error("Failed to get tmux context: " + err.Error())
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return TmuxContext{}
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

	// ASSERTION: Check if we have exactly 4 parts (session_id, window_id, pane_id, pane_pid)
	// All 4 should always be present in tmux (Power of 10 Rule 5)
	if len(filteredParts) != 4 {
		colors.Error(fmt.Sprintf("GetCurrentTmuxContext: unexpected format - expected 4 parts, got %d", len(filteredParts)))
		return TmuxContext{}
	}

	// ASSERTION: Validate that captured context values are non-empty
	if filteredParts[0] == "" || filteredParts[1] == "" || filteredParts[2] == "" {
		colors.Error("GetCurrentTmuxContext: invalid context - session_id, window_id, or pane_id is empty")
		return TmuxContext{}
	}

	ctx := TmuxContext{
		SessionID:   filteredParts[0], // e.g., "$3"
		WindowID:    filteredParts[1], // e.g., "@16"
		PaneID:      filteredParts[2], // e.g., "%21"
		PaneCreated: filteredParts[3], // e.g., "8443" (pane PID)
	}
	return ctx
}

// ValidatePaneExists checks if a pane exists.
func ValidatePaneExists(sessionID, windowID, paneID string) bool {
	target := sessionID + ":" + windowID
	stdout, stderr, err := tmuxRunner("list-panes", "-t", target, "-F", "#{pane_id}")
	if err != nil {
		colors.Debug("tmux list-panes failed: " + err.Error())
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false
	}
	// Each pane ID is on a separate line
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	// Trim paneID input in case it has whitespace
	paneID = strings.TrimSpace(paneID)
	for _, line := range lines {
		// Trim each pane ID from output in case of trailing whitespace
		if strings.TrimSpace(line) == paneID {
			return true
		}
	}
	return false
}

// JumpToPane jumps to a specific pane. It returns true if the jump succeeded
// (either to the pane or fallback to window), false if the jump completely failed.
// INVARIANTS:
//   - SessionID, windowID, and paneID must be non-empty (Power of 10 Rule 5)
//   - Pane reference format must be "sessionID:windowID.paneID" (tmux pane target syntax)
//   - If select-window fails, return false immediately (fail-fast)
//   - If select-pane fails, return false (don't swallow errors)
func JumpToPane(sessionID, windowID, paneID string) bool {
	// ASSERTION: Validate input parameters are non-empty
	if sessionID == "" || windowID == "" || paneID == "" {
		colors.Error("JumpToPane: invalid parameters (empty sessionID, windowID, or paneID)")
		return false
	}

	// First validate if the pane exists
	paneExists := ValidatePaneExists(sessionID, windowID, paneID)
	colors.Debug(fmt.Sprintf("JumpToPane: pane validation for %s:%s in window %s:%s - exists: %v", sessionID, paneID, sessionID, windowID, paneExists))

	// Select the window (this happens regardless of whether the pane exists)
	targetWindow := sessionID + ":" + windowID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting window %s", targetWindow))
	_, stderr, err := tmuxRunner("select-window", "-t", targetWindow)
	if err != nil {
		colors.Error("Window " + targetWindow + " does not exist")
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false
	}

	// If pane doesn't exist, show warning and fall back to window selection
	if !paneExists {
		colors.Warning("Pane " + paneID + " does not exist in window " + targetWindow + ", jumping to window instead")
		colors.Debug(fmt.Sprintf("JumpToPane: falling back to window selection (pane %s not found)", paneID))
		return true
	}

	// Pane exists, select it using the correct tmux pane syntax: "sessionID:windowID.paneID"
	// ASSERTION: targetPane must follow tmux pane reference format
	targetPane := sessionID + ":" + windowID + "." + paneID
	colors.Debug(fmt.Sprintf("JumpToPane: selecting pane %s", targetPane))
	_, stderr, err = tmuxRunner("select-pane", "-t", targetPane)
	if err != nil {
		// Fail-fast: don't swallow errors, return false to indicate failure
		colors.Error("Failed to select pane " + targetPane)
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false
	}

	colors.Debug(fmt.Sprintf("JumpToPane: successfully selected pane %s", targetPane))
	return true
}

// GetTmuxVisibility returns the value of TMUX_INTRAY_VISIBLE global tmux variable.
// Returns "0" if variable is not set.
func GetTmuxVisibility() string {
	stdout, stderr, err := tmuxRunner("show-environment", "-g", "TMUX_INTRAY_VISIBLE")
	if err != nil {
		colors.Debug("tmux show-environment failed: " + err.Error())
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return "0"
	}
	// Output could be "TMUX_INTRAY_VISIBLE=1" or empty line
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "TMUX_INTRAY_VISIBLE=") {
			return strings.TrimPrefix(line, "TMUX_INTRAY_VISIBLE=")
		}
	}
	return "0"
}

// SetTmuxVisibility sets the TMUX_INTRAY_VISIBLE global tmux variable.
func SetTmuxVisibility(value string) error {
	_, stderr, err := tmuxRunner("set-environment", "-g", "TMUX_INTRAY_VISIBLE", value)
	if err != nil {
		errMsg := "Failed to set tmux visibility: " + err.Error()
		colors.Error(errMsg)
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}
