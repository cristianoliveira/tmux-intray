/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the version of tmux-intray.
const Version = "0.1.0"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Show the current version of tmux-intray.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tmux-intray v%s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
