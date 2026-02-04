/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tmux-intray",
	Short: "A quiet inbox for things that happen while you're not looking.",
	Long:  `A quiet inbox for things that happen while you're not looking.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Set version for use in help output
	rootCmd.Version = Version

	// Hide the completion command
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Set custom help function that matches bash help output
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		printHelpText(cmd)
	})

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tmux-intray.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func printHelpText(cmd *cobra.Command) {
	// Order of commands as in bash help
	commandOrder := []string{
		"add",
		"list",
		"dismiss",
		"clear",
		"cleanup",
		"toggle",
		"jump",
		"status",
		"status-panel",
		"follow",
		"help",
		"version",
	}

	// Build command descriptions
	var cmdLines []string
	for _, name := range commandOrder {
		// Find command
		var found *cobra.Command
		for _, c := range cmd.Commands() {
			if c.Name() == name {
				found = c
				break
			}
		}
		if found == nil {
			continue
		}
		// Format: command use + padding + short description
		use := found.Use
		short := found.Short
		// Ensure proper spacing (bash help uses 4 spaces after command)
		cmdLines = append(cmdLines, fmt.Sprintf("    %-16s %s", use, short))
	}

	helpText := fmt.Sprintf(`tmux-intray v%s

A quiet inbox for things that happen while you're not looking.

USAGE:
    tmux-intray [COMMAND] [OPTIONS]

COMMANDS:
%s

OPTIONS:
    -h, --help      Show help message
`, Version, strings.Join(cmdLines, "\n"))
	fmt.Print(helpText)
}
