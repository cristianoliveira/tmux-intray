/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cristianoliveira/tmux-intray/cmd"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/spf13/cobra"
)

// clearCmd represents the clear command
var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all items from the tray",
	Long: `Clear all active notifications from the tray.

This command dismisses all active notifications, running pre-clear and per-notification
hooks, and updates the tmux status option.

USAGE:
    tmux-intray clear

ALIAS:
    This command is an alias for 'tmux-intray dismiss --all'.

EXAMPLES:
    # Clear all active notifications
    tmux-intray clear`,
	Run: runClear,
}

var clearAllFunc = func() error {
	return storageStore.DismissAll()
}

func ClearAll() error {
	return clearAllFunc()
}

func init() {
	cmd.RootCmd.AddCommand(clearCmd)
}

func runClear(cmd *cobra.Command, args []string) {
	// Skip confirmation if running in CI or test environment
	if os.Getenv("CI") != "" || os.Getenv("BATS_TMPDIR") != "" {
		// In CI/test mode, proceed without confirmation
		err := ClearAll()
		if err != nil {
			colors.Error(err.Error())
			return
		}
		colors.Success("cleared")
		return
	}

	// Ask for confirmation
	if !confirmClearAll() {
		colors.Info("Operation cancelled")
		return
	}
	// Run clear operation
	err := ClearAll()
	if err != nil {
		colors.Error(err.Error())
		return
	}
	colors.Success("Tray cleared")
}

// confirmClearAll asks the user for confirmation before clearing all notifications.
func confirmClearAll() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Are you sure you want to clear all active notifications? (y/N): ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read, assume no
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}
