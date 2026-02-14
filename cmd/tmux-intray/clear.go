/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
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

type clearClient interface {
	ClearTrayItems() error
}

// NewClearCmd creates the clear command with explicit dependencies.
func NewClearCmd(client clearClient) *cobra.Command {
	if client == nil {
		panic("NewClearCmd: client dependency cannot be nil")
	}

	clearCmd := &cobra.Command{
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
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Skip confirmation if running in CI or test environment
			if allowTmuxlessMode() {
				// In CI/test mode, proceed without confirmation
				err := client.ClearTrayItems()
				if err != nil {
					return fmt.Errorf("clear: failed to clear tray items: %w", err)
				}
				colors.Success("cleared")
				return nil
			}

			// Ask for confirmation
			if !confirmClearAll() {
				colors.Info("Operation cancelled")
				return nil
			}

			err := client.ClearTrayItems()
			if err != nil {
				return fmt.Errorf("clear: failed to clear tray items: %w", err)
			}
			colors.Success("Tray cleared")
			return nil
		},
	}

	return clearCmd
}

// clearCmd represents the clear command.
var clearCmd = NewClearCmd(coreClient)

func init() {
	cmd.RootCmd.AddCommand(clearCmd)
}

var clearAllFunc = func() error {
	return fileStorage.DismissAll()
}

func ClearAll() error {
	return clearAllFunc()
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
