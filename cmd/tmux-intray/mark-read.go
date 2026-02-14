/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/cmd"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/spf13/cobra"
)

type markReadClient interface {
	MarkNotificationRead(id string) error
}

// NewMarkReadCmd creates the mark-read command with explicit dependencies.
func NewMarkReadCmd(client markReadClient) *cobra.Command {
	if client == nil {
		panic("NewMarkReadCmd: client dependency cannot be nil")
	}

	markReadCmd := &cobra.Command{
		Use:   "mark-read <id>",
		Short: "Mark a notification as read",
		Long: `Mark a notification as read by ID.

USAGE:
    tmux-intray mark-read <id>

OPTIONS:
    -h, --help           Show this help`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			err := client.MarkNotificationRead(id)
			if err != nil {
				return fmt.Errorf("mark-read: %w", err)
			}
			colors.Success(fmt.Sprintf("Notification %s marked as read", id))
			return nil
		},
	}

	return markReadCmd
}

// markReadCmd represents the mark-read command
var markReadCmd = NewMarkReadCmd(coreClient)

func init() {
	cmd.RootCmd.AddCommand(markReadCmd)
}
