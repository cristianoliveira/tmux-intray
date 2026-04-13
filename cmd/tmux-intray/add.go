/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
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
	useCase := appcore.NewAddUseCase(client)
	return useCase.Execute(appcore.AddInput{
		Args:        args,
		Session:     sessionFlag,
		Window:      windowFlag,
		Pane:        paneFlag,
		PaneCreated: paneCreatedFlag,
		NoAssociate: noAssociateFlag,
		Level:       levelFlag,
		AllowTmuxless: func() bool {
			return allowTmuxlessMode()
		},
	})
}

// validateMessage checks message length and emptiness (matches Bash validation)
func validateMessage(message string) error {
	return appcore.ValidateAddMessage(message)
}
