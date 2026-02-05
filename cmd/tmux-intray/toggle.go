/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/cmd"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/spf13/cobra"
)

// toggleGetVisibilityFunc is the function used to get visibility. Can be changed for testing.
var toggleGetVisibilityFunc = func() string {
	return core.GetVisibility()
}

// toggleSetVisibilityFunc is the function used to set visibility. Can be changed for testing.
var toggleSetVisibilityFunc = func(visible bool) error {
	return core.SetVisibility(visible)
}

// GetCurrentVisibility returns the current visibility as a boolean.
// Returns true if tray is visible, false if hidden.
func GetCurrentVisibility() bool {
	return toggleGetVisibilityFunc() == "1"
}

// Toggle toggles the tray visibility and returns the new visibility state.
// Returns true if tray is now visible, false if hidden, and any error.
func Toggle() (bool, error) {
	visible := toggleGetVisibilityFunc()
	newVisible := visible != "1"
	err := toggleSetVisibilityFunc(newVisible)
	return newVisible, err
}

// toggleCmd represents the toggle command
var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle the tray visibility",
	Long: `Toggle the visibility of the tmux-intray tray.

This command shows or hides the tray by setting the global tmux environment
variable TMUX_INTRAY_VISIBLE to "1" (visible) or "0" (hidden).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure tmux is running
		if !core.EnsureTmuxRunning() {
			if allowTmuxlessMode() {
				colors.Warning("tmux not running; skipping toggle")
				return nil
			}
			return fmt.Errorf("No tmux session running")
		}

		// Get current visibility
		oldVisible := core.GetVisibility()
		var newVisible bool
		var msg string
		if oldVisible == "1" {
			newVisible = false
			msg = "Tray hidden"
		} else {
			newVisible = true
			msg = "Tray visible"
		}
		newVisibleStr := "0"
		if newVisible {
			newVisibleStr = "1"
		}

		// Run pre-toggle hooks
		envVars := []string{
			"OLD_VISIBLE=" + oldVisible,
			"VISIBLE=" + newVisibleStr,
		}
		if err := hooks.Run("pre-toggle", envVars...); err != nil {
			colors.Error(err.Error())
			return err
		}

		// Set new visibility
		if err := core.SetVisibility(newVisible); err != nil {
			colors.Error(err.Error())
			return err
		}

		// Run post-toggle hooks
		if err := hooks.Run("post-toggle", envVars...); err != nil {
			colors.Error(err.Error())
			return err
		}

		colors.Info(msg)
		return nil
	},
}

func init() {
	cmd.RootCmd.AddCommand(toggleCmd)
}
