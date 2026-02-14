/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
)

type jumpClient interface {
	EnsureTmuxRunning() bool
	GetNotificationByID(id string) (string, error)
	ValidatePaneExists(session, window, pane string) bool
	JumpToPane(session, window, pane string) bool
	MarkNotificationRead(id string) error
}

// NewJumpCmd creates the jump command with explicit dependencies.
func NewJumpCmd(client jumpClient) *cobra.Command {
	if client == nil {
		panic("NewJumpCmd: client dependency cannot be nil")
	}

	var noMarkReadFlag bool

	jumpCmd := &cobra.Command{
		Use:   "jump",
		Short: "Jump to the pane of a notification",
		Long: `Jump to the pane of a notification.

USAGE:
    tmux-intray jump <id>

DESCRIPTION:
    Navigates to the tmux pane where the notification originated. The pane
    must still exist; if it doesn't, the command falls back to the window.
    By default, a successful jump automatically marks the notification as read.
    Use --no-mark-read to disable this behavior.

ARGUMENTS:
    <id>    Notification ID (as shown in 'tmux-intray list --format=table')

OPTIONS:
    --no-mark-read    Do not mark the notification as read after a successful jump

EXAMPLES:
    # Jump to pane of notification with ID 42
    tmux-intray jump 42

    # Jump without marking notification as read
    tmux-intray jump --no-mark-read 42`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("jump: requires a notification id")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			if !client.EnsureTmuxRunning() {
				return fmt.Errorf("tmux not running")
			}

			line, err := client.GetNotificationByID(id)
			if err != nil {
				return fmt.Errorf("jump: %w", err)
			}

			fields := strings.Split(line, "\t")
			// Ensure at least 7 fields (up to pane)
			if len(fields) <= storage.FieldPane {
				return fmt.Errorf("jump: invalid notification line format")
			}
			state := fields[storage.FieldState]
			session := fields[storage.FieldSession]
			window := fields[storage.FieldWindow]
			pane := fields[storage.FieldPane]

			if session == "" || window == "" || pane == "" {
				var missingFields []string
				if session == "" {
					missingFields = append(missingFields, "session")
				}
				if window == "" {
					missingFields = append(missingFields, "window")
				}
				if pane == "" {
					missingFields = append(missingFields, "pane")
				}
				return fmt.Errorf(
					"jump: notification %s missing required fields:\n"+
						"  missing: %s\n"+
						"  required fields: session, window, pane\n"+
						"  hint: notifications must be created from within an active tmux session for jump to work",
					id, strings.Join(missingFields, ", "))
			}

			paneExists := client.ValidatePaneExists(session, window, pane)

			if !client.JumpToPane(session, window, pane) {
				return fmt.Errorf("jump: failed to jump because pane or window does not exist")
			}

			if !noMarkReadFlag {
				if err := client.MarkNotificationRead(id); err != nil {
					return fmt.Errorf("jump: failed to mark notification as read: %w", err)
				}
			}

			if state == "dismissed" {
				colors.Info(fmt.Sprintf("Notification %s is dismissed, but jumping anyway", id))
			}

			if paneExists {
				colors.Success(fmt.Sprintf("Jumped to session %s, window %s, pane %s", session, window, pane))
			} else {
				colors.Warning(fmt.Sprintf("Pane %s no longer exists (jumped to window %s:%s instead)", pane, session, window))
			}
			return nil
		},
	}

	jumpCmd.Flags().BoolVar(&noMarkReadFlag, "no-mark-read", false, "do not mark notification as read after successful jump")
	return jumpCmd
}

// jumpCmd represents the jump command.
var jumpCmd = NewJumpCmd(coreClient)

func init() {
	cmd.RootCmd.AddCommand(jumpCmd)
}
