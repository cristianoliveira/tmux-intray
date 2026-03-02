/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"fmt"
	"strings"

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

type jumpDetails struct {
	state   string
	session string
	window  string
	pane    string
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
		RunE: makeJumpRunE(client, &noMarkReadFlag),
	}

	jumpCmd.Flags().BoolVar(&noMarkReadFlag, "no-mark-read", false, "do not mark notification as read after successful jump")
	return jumpCmd
}

func makeJumpRunE(client jumpClient, noMarkReadFlag *bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if !client.EnsureTmuxRunning() {
			return fmt.Errorf("tmux not running")
		}

		details, err := loadJumpDetails(client, id)
		if err != nil {
			return err
		}

		paneExists, err := performJump(client, id, details, *noMarkReadFlag)
		if err != nil {
			return err
		}

		reportJumpOutcome(id, details, paneExists)
		return nil
	}
}

func loadJumpDetails(client jumpClient, id string) (jumpDetails, error) {
	line, err := client.GetNotificationByID(id)
	if err != nil {
		return jumpDetails{}, fmt.Errorf("jump: %w", err)
	}

	return parseJumpDetails(id, line)
}

func parseJumpDetails(id, line string) (jumpDetails, error) {
	fields := strings.Split(line, "\t")
	// Ensure at least 7 fields (up to pane)
	if len(fields) <= storage.FieldPane {
		return jumpDetails{}, fmt.Errorf("jump: invalid notification line format")
	}

	details := jumpDetails{
		state:   fields[storage.FieldState],
		session: fields[storage.FieldSession],
		window:  fields[storage.FieldWindow],
		pane:    fields[storage.FieldPane],
	}

	if err := validateJumpFields(id, details); err != nil {
		return jumpDetails{}, err
	}

	return details, nil
}

func validateJumpFields(id string, details jumpDetails) error {
	if details.session == "" || details.window == "" || details.pane == "" {
		var missingFields []string
		if details.session == "" {
			missingFields = append(missingFields, "session")
		}
		if details.window == "" {
			missingFields = append(missingFields, "window")
		}
		if details.pane == "" {
			missingFields = append(missingFields, "pane")
		}
		return fmt.Errorf(
			"jump: notification %s missing required fields:\n"+
				"  missing: %s\n"+
				"  required fields: session, window, pane\n"+
				"  hint: notifications must be created from within an active tmux session for jump to work",
			id, strings.Join(missingFields, ", "))
	}

	return nil
}

func performJump(client jumpClient, id string, details jumpDetails, noMarkRead bool) (bool, error) {
	paneExists := client.ValidatePaneExists(details.session, details.window, details.pane)

	if !client.JumpToPane(details.session, details.window, details.pane) {
		return paneExists, fmt.Errorf("jump: failed to jump because pane or window does not exist")
	}

	if !noMarkRead {
		if err := client.MarkNotificationRead(id); err != nil {
			return paneExists, fmt.Errorf("jump: failed to mark notification as read: %w", err)
		}
	}

	return paneExists, nil
}

func reportJumpOutcome(id string, details jumpDetails, paneExists bool) {
	if details.state == "dismissed" {
		colors.Info(fmt.Sprintf("Notification %s is dismissed, but jumping anyway", id))
	}

	if paneExists {
		colors.Success(fmt.Sprintf("Jumped to session %s, window %s, pane %s", details.session, details.window, details.pane))
		return
	}

	colors.Warning(fmt.Sprintf("Pane %s no longer exists (jumped to window %s:%s instead)", details.pane, details.session, details.window))
}
