/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
)

// dismissCmd represents the dismiss command
var dismissCmd = &cobra.Command{
	Use:   "dismiss ID",
	Short: "Dismiss a notification",
	Long: `Dismiss a specific notification by ID.
Runs pre-dismiss and post-dismiss hooks, updates tmux status.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		storage.Init()
		err := storage.DismissNotification(id)
		if err != nil {
			return err
		}
		cmd.Printf("Notification %s dismissed\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dismissCmd)
}
