// Package core provides core tmux interaction and tray management.
package core

import (
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
	format := "#{session_id} #{window_id} #{pane_id}"
	stdout, stderr, err := tmuxRunner("display", "-p", format)
	if err != nil {
		colors.Error("Failed to get tmux context: " + err.Error())
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return TmuxContext{}
	}
	// Split by whitespace
	parts := strings.Fields(stdout)
	if len(parts) != 3 {
		colors.Error("Unexpected tmux display output: " + stdout)
		return TmuxContext{}
	}
	ctx := TmuxContext{
		SessionID: parts[0],
		WindowID:  parts[1],
		PaneID:    parts[2],
	}
	// Try to get pane creation time (pane_created is not a standard tmux format variable)
	// We'll use an empty string since pane_created is not available in tmux format variables
	ctx.PaneCreated = ""
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
	for _, line := range lines {
		if line == paneID {
			return true
		}
	}
	return false
}

// JumpToPane jumps to a specific pane.
func JumpToPane(sessionID, windowID, paneID string) bool {
	// First try to jump to window
	targetWindow := sessionID + ":" + windowID
	_, stderr, err := tmuxRunner("select-window", "-t", targetWindow)
	if err != nil {
		colors.Warning("Window " + targetWindow + " does not exist")
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false
	}
	// Then select pane
	targetPane := sessionID + ":" + windowID + "." + paneID
	_, stderr, err = tmuxRunner("select-pane", "-t", targetPane)
	if err != nil {
		colors.Warning("Pane " + targetPane + " does not exist, but window selected")
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		// We still selected the window, return true? The bash script returns 0.
		// The function returns bool indicating success; we could consider window selection as partial success.
		// According to bash script, if pane doesn't exist, they still select window and return 0.
		// So we should return true.
		return true
	}
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
func SetTmuxVisibility(value string) bool {
	_, stderr, err := tmuxRunner("set-environment", "-g", "TMUX_INTRAY_VISIBLE", value)
	if err != nil {
		colors.Error("Failed to set tmux visibility: " + err.Error())
		if stderr != "" {
			colors.Debug("stderr: " + stderr)
		}
		return false
	}
	return true
}
