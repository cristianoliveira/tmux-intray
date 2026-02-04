/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// statusPanelCmd represents the status-panel command
var statusPanelCmd = &cobra.Command{
	Use:   "status-panel",
	Short: "Status bar indicator script (for tmux status-right)",
	Long:  `Status bar indicator script (for tmux status-right).`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("status-panel called")
	},
}

func init() {
	rootCmd.AddCommand(statusPanelCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statusPanelCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusPanelCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
