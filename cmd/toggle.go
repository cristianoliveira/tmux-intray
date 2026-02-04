/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/spf13/cobra"
)

// toggleCmd represents the toggle command
var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle the tray visibility",
	Long: `Toggle the visibility of the tmux-intray tray.

This command shows or hides the tray by setting the global tmux environment
variable TMUX_INTRAY_VISIBLE to "1" (visible) or "0" (hidden).`,
	Run: func(cmd *cobra.Command, args []string) {
		// Ensure tmux is running
		if !core.EnsureTmuxRunning() {
			colors.Error("No tmux session running")
			os.Exit(1)
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
			os.Exit(1)
		}

		// Set new visibility
		if err := core.SetVisibility(newVisible); err != nil {
			colors.Error(err.Error())
			os.Exit(1)
		}

		// Run post-toggle hooks
		if err := hooks.Run("post-toggle", envVars...); err != nil {
			colors.Error(err.Error())
			os.Exit(1)
		}

		colors.Info(msg)
	},
}

func init() {
	rootCmd.AddCommand(toggleCmd)
}
