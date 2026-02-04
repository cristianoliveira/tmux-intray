/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
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
	Long: `Toggle the tray visibility.

USAGE:
    tmux-intray toggle

DESCRIPTION:
    Toggles the global visibility flag for the tray. When hidden, notifications
    are still stored but may not appear in status bar indicators. This command
    is primarily used by the tmux plugin (bound to 'prefix+i').

EXAMPLES:
    # Toggle tray visibility
    tmux-intray toggle`,
	Run: runToggle,
}

func init() {
	rootCmd.AddCommand(toggleCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// toggleCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// toggleCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runToggle(cmd *cobra.Command, args []string) {
	// Ensure tmux is running (mirror bash script behavior)
	if !core.EnsureTmuxRunning() {
		colors.Error("tmux is not running")
		return
	}

	// Get current visibility before toggle (optional display)
	current := GetCurrentVisibility()
	status := "hidden"
	if current {
		status = "visible"
	}
	colors.Info("Current visibility: " + status)

	// Toggle visibility
	newVisible, err := Toggle()
	if err != nil {
		colors.Error(err.Error())
		return
	}

	// Display result
	result := "hidden"
	if newVisible {
		result = "visible"
	}
	colors.Success("Tray " + result)
}
