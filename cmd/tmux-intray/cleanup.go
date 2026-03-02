/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"errors"
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/spf13/cobra"
)

type cleanupClient interface {
	EnsureTmuxRunning() bool
	CleanupOldNotifications(days int, dryRun bool) error
}

// NewCleanupCmd creates the cleanup command with explicit dependencies.
func NewCleanupCmd(client cleanupClient) *cobra.Command {
	if client == nil {
		panic("NewCleanupCmd: client dependency cannot be nil")
	}

	var daysFlag int
	var dryRunFlag bool

	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up old dismissed notifications",
		Long: `Clean up old dismissed notifications.

Automatically cleans up notifications that have been dismissed and are older
than the configured auto-cleanup days. This helps prevent storage bloat.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Load()

			// Ensure tmux is running (matches bash behavior)
			if !client.EnsureTmuxRunning() {
				return errors.New("no tmux session running")
			}

			// Get flags
			days := daysFlag
			if days == 0 {
				days = config.GetInt("auto_cleanup_days", 30)
			}

			if days <= 0 {
				return fmt.Errorf("days must be a positive integer")
			}

			cmd.Printf("Starting cleanup of notifications dismissed more than %d days ago\n", days)

			err := client.CleanupOldNotifications(days, dryRunFlag)
			if err != nil {
				return fmt.Errorf("cleanup failed: %w", err)
			}

			cmd.Println("Cleanup completed")
			return nil
		},
	}

	// Default days 0 means "use config value"
	cleanupCmd.Flags().IntVar(&daysFlag, "days", 0, "Clean up notifications dismissed more than N days ago (default: TMUX_INTRAY_AUTO_CLEANUP_DAYS config value)")
	cleanupCmd.Flags().BoolVar(&dryRunFlag, "dryrun", false, "Show what would be deleted without actually deleting")

	return cleanupCmd
}
