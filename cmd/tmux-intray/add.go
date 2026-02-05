/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/spf13/cobra"
)

var (
	sessionFlag     string
	windowFlag      string
	paneFlag        string
	paneCreatedFlag string
	noAssociateFlag bool
	levelFlag       string
)

// addCmd represents the add command
var addCmd = &cobra.Command{
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
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if message is provided to match bash error message
		if len(args) == 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "add requires a message\n")
			return fmt.Errorf("")
		}

		// Ensure tmux is running (required for auto-association unless --no-associate)
		if !noAssociateFlag && sessionFlag == "" && windowFlag == "" && paneFlag == "" {
			if !core.EnsureTmuxRunning() {
				return fmt.Errorf("No tmux session running")
			}
		}

		// Join arguments as message (bash style)
		message := strings.Join(args, " ")

		// Validate message
		if err := validateMessage(message); err != nil {
			return err
		}

		// Format message with timestamp (as Bash does)
		formattedMessage := formatMessage(message)

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
		id := core.AddTrayItem(formattedMessage, sessionFlag, windowFlag, paneFlag, paneCreatedFlag, noAssociateFlag, level)
		if id == "" {
			return fmt.Errorf("Failed to add tray item")
		}

		colors.Success("added")
		return nil
	},
}

func init() {
	cmd.RootCmd.AddCommand(addCmd)

	// Define flags
	addCmd.Flags().StringVar(&sessionFlag, "session", "", "Associate with specific session ID")
	addCmd.Flags().StringVar(&windowFlag, "window", "", "Associate with specific window ID")
	addCmd.Flags().StringVar(&paneFlag, "pane", "", "Associate with specific pane ID")
	addCmd.Flags().StringVar(&paneCreatedFlag, "pane-created", "", "Pane creation timestamp (seconds since epoch)")
	addCmd.Flags().BoolVar(&noAssociateFlag, "no-associate", false, "Do not associate with any pane")
	addCmd.Flags().StringVar(&levelFlag, "level", "info", "Notification level: info, warning, error, critical")
}

// validateMessage checks message length and emptiness (matches Bash validation)
func validateMessage(message string) error {
	// Check length
	if len(message) > 1000 {
		return fmt.Errorf("Message too long (max 1000 characters)")
	}
	// Check if empty after stripping whitespace
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return fmt.Errorf("Message cannot be empty")
	}
	return nil
}

// formatMessage adds timestamp prefix like Bash's format_message
func formatMessage(message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("[%s] %s", timestamp, message)
}
