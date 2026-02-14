/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/spf13/cobra"
)

type addClient interface {
	EnsureTmuxRunning() bool
	AddTrayItem(item, session, window, pane, paneCreated string, noAssociate bool, level string) (string, error)
}

// NewAddCmd creates the add command with explicit dependencies.
func NewAddCmd(client addClient) *cobra.Command {
	if client == nil {
		panic("NewAddCmd: client dependency cannot be nil")
	}

	var sessionFlag string
	var windowFlag string
	var paneFlag string
	var paneCreatedFlag string
	var noAssociateFlag bool
	var levelFlag string

	addCmd := &cobra.Command{
		Use:   "add [OPTIONS] <message>",
		Short: "Add a new item to the tray",
		Long: `tmux-intray add - Add a new item to the tray

USAGE:
    tmux-intray add [OPTIONS] <message>

OPTIONS:
    --session <id>          Associate with specific session ID
    --window <id>           Associate with specific window ID
    --pane <id>             Associate with specific pane ID
    --pane-created <time>   Pane creation timestamp (seconds since epoch)
    --no-associate          Do not associate with any pane
    --level <level>         Notification level: info, warning, error, critical (default: info)
    -h, --help              Show this help

If no pane association options are provided, automatically associates with
the current tmux pane (if inside tmux). Use --no-associate to skip.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "add requires a message\n")
				return fmt.Errorf("")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Treat empty strings same as not provided (e.g., --session="" from plugin)
			// This makes the CLI resilient to the plugin passing empty flag values
			sessionFlag = strings.TrimSpace(sessionFlag)
			windowFlag = strings.TrimSpace(windowFlag)
			paneFlag = strings.TrimSpace(paneFlag)

			needsAutoAssociation := !noAssociateFlag && sessionFlag == "" && windowFlag == "" && paneFlag == ""
			if needsAutoAssociation && !client.EnsureTmuxRunning() {
				if allowTmuxlessMode() {
					colors.Warning("tmux not running; adding notification without pane association")
					noAssociateFlag = true
				} else {
					return fmt.Errorf("tmux not running")
				}
			}

			// Join arguments as message (bash style)
			message := strings.Join(args, " ")

			// Validate message
			if err := validateMessage(message); err != nil {
				return err
			}

			// Message is stored as-is; storage layer handles timestamps
			formattedMessage := message

			// Determine level
			level := levelFlag
			if level == "" {
				level = "info"
			}

			// Run pre-add hooks (they will be run again by storage.AddNotification)
			// We could skip here, but storage already runs hooks; we still need to ensure
			// hooks are initialized. The root command already calls hooks.Init().
			// We'll rely on storage hooks.

			// Add tray item
			_, err := client.AddTrayItem(formattedMessage, sessionFlag, windowFlag, paneFlag, paneCreatedFlag, noAssociateFlag, level)
			if err != nil {
				return fmt.Errorf("add: failed to add tray item: %w", err)
			}

			colors.Success("added")
			return nil
		},
	}

	addCmd.Flags().StringVar(&sessionFlag, "session", "", "Associate with specific session ID")
	addCmd.Flags().StringVar(&windowFlag, "window", "", "Associate with specific window ID")
	addCmd.Flags().StringVar(&paneFlag, "pane", "", "Associate with specific pane ID")
	addCmd.Flags().StringVar(&paneCreatedFlag, "pane-created", "", "Pane creation timestamp (seconds since epoch)")
	addCmd.Flags().BoolVar(&noAssociateFlag, "no-associate", false, "Do not associate with any pane")
	addCmd.Flags().StringVar(&levelFlag, "level", "info", "Notification level: info, warning, error, critical")

	return addCmd
}

// addCmd represents the add command.
var addCmd = NewAddCmd(coreClient)

func init() {
	cmd.RootCmd.AddCommand(addCmd)
}

// validateMessage checks message length and emptiness (matches Bash validation)
func validateMessage(message string) error {
	// Check length
	if len(message) > 1000 {
		return fmt.Errorf("add: message too long (max 1000 characters)")
	}
	// Check if empty after stripping whitespace
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return fmt.Errorf("add: message cannot be empty")
	}
	return nil
}
