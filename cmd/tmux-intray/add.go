/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"fmt"
	"strings"

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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddCmd(client, args, sessionFlag, windowFlag, paneFlag, paneCreatedFlag, noAssociateFlag, levelFlag)
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

// runAddCmd executes the add command logic.
func runAddCmd(client addClient, args []string, sessionFlag, windowFlag, paneFlag, paneCreatedFlag string, noAssociateFlag bool, levelFlag string) error {
	sessionFlag, windowFlag, paneFlag = normalizeAssociationFlags(sessionFlag, windowFlag, paneFlag)

	resolvedNoAssociate, err := resolveAssociationMode(client, noAssociateFlag, sessionFlag, windowFlag, paneFlag)
	if err != nil {
		return err
	}

	message := strings.Join(args, " ")
	if err := validateMessage(message); err != nil {
		return err
	}

	level := resolveLevel(levelFlag)

	_, err = client.AddTrayItem(message, sessionFlag, windowFlag, paneFlag, paneCreatedFlag, resolvedNoAssociate, level)
	if err != nil {
		return fmt.Errorf("add: failed to add tray item: %w", err)
	}

	colors.Success("added")
	return nil
}

func normalizeAssociationFlags(sessionFlag, windowFlag, paneFlag string) (string, string, string) {
	return strings.TrimSpace(sessionFlag), strings.TrimSpace(windowFlag), strings.TrimSpace(paneFlag)
}

func needsAutoAssociation(noAssociate bool, sessionFlag, windowFlag, paneFlag string) bool {
	return !noAssociate && sessionFlag == "" && windowFlag == "" && paneFlag == ""
}

func resolveAssociationMode(client addClient, noAssociate bool, sessionFlag, windowFlag, paneFlag string) (bool, error) {
	if !needsAutoAssociation(noAssociate, sessionFlag, windowFlag, paneFlag) {
		return noAssociate, nil
	}

	if client.EnsureTmuxRunning() {
		return noAssociate, nil
	}

	if allowTmuxlessMode() {
		colors.Warning("tmux not running; adding notification without pane association")
		return true, nil
	}

	return false, fmt.Errorf("tmux not running")
}

func resolveLevel(levelFlag string) string {
	if levelFlag == "" {
		return "info"
	}
	return levelFlag
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
