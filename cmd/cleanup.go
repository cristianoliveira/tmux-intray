/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
)

// Flag variables for cleanup command
var (
	daysFlag   int
	dryRunFlag bool
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up old dismissed notifications",
	Long: `Clean up old dismissed notifications.

Automatically cleans up notifications that have been dismissed and are older
than the configured auto-cleanup days. This helps prevent storage bloat.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config.Load()

		// Ensure tmux is running (matches bash behavior)
		if !core.EnsureTmuxRunning() {
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

		storage.Init()
		err := storage.CleanupOldNotifications(days, dryRunFlag)
		if err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}

		cmd.Println("Cleanup completed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	// Default days 0 means "use config value"
	cleanupCmd.Flags().IntVar(&daysFlag, "days", 0, "Clean up notifications dismissed more than N days ago (default: TMUX_INTRAY_AUTO_CLEANUP_DAYS config value)")
	cleanupCmd.Flags().BoolVar(&dryRunFlag, "dryrun", false, "Show what would be deleted without actually deleting")
}
