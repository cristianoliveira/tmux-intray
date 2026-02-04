/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up old dismissed notifications",
	Long: `Clean up old dismissed notifications.

USAGE:
    tmux-intray cleanup [OPTIONS]

OPTIONS:
    --days N          Clean up notifications dismissed more than N days ago
                      (default: TMUX_INTRAY_AUTO_CLEANUP_DAYS config value)
    --dry-run         Show what would be deleted without actually deleting
    -h, --help        Show this help

Automatically cleans up notifications that have been dismissed and are older
than the configured auto-cleanup days. This helps prevent storage bloat.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cleanup called")
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cleanupCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cleanupCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
