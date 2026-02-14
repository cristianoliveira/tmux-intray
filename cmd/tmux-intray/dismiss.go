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

type dismissClient interface {
	DismissNotification(id string) error
	DismissAll() error
}

// NewDismissCmd creates the dismiss command with explicit dependencies.
func NewDismissCmd(client dismissClient) *cobra.Command {
	if client == nil {
		panic("NewDismissCmd: client dependency cannot be nil")
	}

	var dismissAll bool

	dismissCmd := &cobra.Command{
		Use:   "dismiss [ID]",
		Short: "Dismiss a notification",
		Long: `Dismiss a specific notification by ID or all active notifications.

USAGE:
    tmux-intray dismiss <id>      Dismiss a specific notification
    tmux-intray dismiss --all     Dismiss all active notifications

OPTIONS:
    -h, --help           Show this help`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate arguments
			if dismissAll && len(args) > 0 {
				return fmt.Errorf("dismiss: cannot specify both --all and id")
			}
			if !dismissAll && len(args) == 0 {
				return fmt.Errorf("dismiss: either specify an id or use --all")
			}
			if len(args) > 1 {
				return fmt.Errorf("dismiss: too many arguments")
			}

			if dismissAll {
				return dismissAllWithConfirmation(client)
			}
			return dismissSingleNotification(client, args[0])
		},
	}

	dismissCmd.Flags().BoolVar(&dismissAll, "all", false, "Dismiss all active notifications")
	return dismissCmd
}

// dismissAllWithConfirmation handles dismissing all notifications with confirmation if needed.
func dismissAllWithConfirmation(client dismissClient) error {
	if !isCIOrTestEnv() {
		if !confirmDismissAllFunc() {
			colors.Info("Operation cancelled")
			return nil
		}
	} else {
		colors.Debug("skipping confirmation due to CI/test environment")
	}

	err := client.DismissAll()
	if err != nil {
		return fmt.Errorf("dismiss: failed to dismiss all: %w", err)
	}
	colors.Success("All active notifications dismissed")
	return nil
}

// isCIOrTestEnv checks if running in CI or test environment.
func isCIOrTestEnv() bool {
	return os.Getenv("CI") != "" || os.Getenv("BATS_TMPDIR") != ""
}

// dismissSingleNotification handles dismissing a single notification.
func dismissSingleNotification(client dismissClient, id string) error {
	err := client.DismissNotification(id)
	if err != nil {
		return fmt.Errorf("dismiss: failed to dismiss notification: %w", err)
	}
	colors.Success("Notification " + id + " dismissed")
	return nil
}

// dismissCmd represents the dismiss command
var dismissCmd = NewDismissCmd(coreClient)

var dismissFunc = func(id string) error {
	return coreClient.DismissNotification(id)
}

var dismissAllFunc = func() error {
	return coreClient.DismissAll()
}

var confirmDismissAllFunc = func() bool {
	return confirmDismissAll()
}

func Dismiss(id string) error {
	return dismissFunc(id)
}

func DismissAll() error {
	return dismissAllFunc()
}

func init() {
	cmd.RootCmd.AddCommand(dismissCmd)
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
