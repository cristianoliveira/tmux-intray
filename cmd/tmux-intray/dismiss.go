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
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
)

// dismissCmd represents the dismiss command
var dismissCmd = &cobra.Command{
	Use:   "dismiss [ID]",
	Short: "Dismiss a notification",
	Long: `Dismiss a specific notification by ID or all active notifications.

USAGE:
    tmux-intray dismiss <id>      Dismiss a specific notification
    tmux-intray dismiss --all     Dismiss all active notifications

OPTIONS:
    -h, --help           Show this help`,
	Args: cobra.MaximumNArgs(1),
	Run:  runDismiss,
}

var (
	dismissAll bool
)

var dismissFunc = func(id string) error {
	return storage.DismissNotification(id)
}

var dismissAllFunc = func() error {
	return storage.DismissAll()
}

func Dismiss(id string) error {
	return dismissFunc(id)
}

func DismissAll() error {
	return dismissAllFunc()
}

func init() {
	cmd.RootCmd.AddCommand(dismissCmd)

	// Local flags
	dismissCmd.Flags().BoolVar(&dismissAll, "all", false, "Dismiss all active notifications")
}

func runDismiss(cmd *cobra.Command, args []string) {
	// Validate arguments
	if dismissAll && len(args) > 0 {
		colors.Error("dismiss: cannot specify both --all and id")
		return
	}
	if !dismissAll && len(args) == 0 {
		colors.Error("dismiss: either specify an id or use --all")
		return
	}
	if len(args) > 1 {
		colors.Error("dismiss: too many arguments")
		return
	}

	// Initialize storage (done by storage package automatically, but ensure)
	// storage.Init() called by Dismiss/DismissAll

	if dismissAll {
		// Ask for confirmation
		if !confirmDismissAll() {
			colors.Info("Operation cancelled")
			return
		}
		err := DismissAll()
		if err != nil {
			colors.Error(err.Error())
			return
		}
		colors.Success("All active notifications dismissed")
	} else {
		id := args[0]
		// Validate ID is numeric (optional, storage will validate)
		err := Dismiss(id)
		if err != nil {
			colors.Error(err.Error())
			return
		}
		colors.Success("Notification " + id + " dismissed")
	}
	return
}

// confirmDismissAll asks the user for confirmation before dismissing all notifications.
func confirmDismissAll() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Are you sure you want to dismiss all active notifications? (y/N): ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read, assume no
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}
