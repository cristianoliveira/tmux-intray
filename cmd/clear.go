/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
)

// clearCmd represents the clear command
var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all items from the tray",
	Long: `Clear all active notifications from the tray.

This command dismisses all active notifications, running pre-clear and per-notification
hooks, and updates the tmux status option.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize storage
		storage.Init()
		// Run clear operation
		err := storage.DismissAll()
		if err != nil {
			return fmt.Errorf("failed to clear tray: %w", err)
		}
		cmd.Println("Tray cleared")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clearCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clearCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
