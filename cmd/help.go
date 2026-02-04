/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// helpCmd represents the help command
var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Show this help message",
	Long:  `Show this help message.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Use the root command's help function
		cmd.Root().Help()
	},
}

func init() {
	rootCmd.SetHelpCommand(helpCmd)
}
