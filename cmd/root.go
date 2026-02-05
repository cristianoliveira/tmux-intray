/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/spf13/cobra"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tmux-intray",
	Short: "A quiet inbox for things that happen while you're not looking.",
	Long:  `A quiet inbox for things that happen while you're not looking.`,
	// Set custom error messages to match bash implementation
	SilenceErrors: true,
	SilenceUsage:  true,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			if args[0] == "--help" || args[0] == "-h" {
				cmd.Help()
				return
			}
			fmt.Fprintf(os.Stderr, "Unknown command '%s'\n", args[0])
		}
	},
}

// Override error handling to match bash messages
func init() {
	// Custom error handler
	cobra.OnInitialize(func() {
		cobra.EnableCommandSorting = false
	})
}

// Custom error handling to match bash messages
func handleCommandError(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Unknown command '%s'\n", args[0])
		return fmt.Errorf("")
	}
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteOriginal() error {
	defer hooks.WaitForPendingHooks()
	hooks.Init()
	return rootCmd.Execute()
}

func init() {
	// Set version for use in help output
	rootCmd.Version = Version

	// Hide the completion command
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Custom help is provided by the help command; default help template is used for --help

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tmux-intray.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	defer hooks.WaitForPendingHooks()
	hooks.Init()

	// Get the command line arguments
	args := os.Args[1:]

	// If no command provided or it's a help request, use cobra's default behavior
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		return rootCmd.Execute()
	}

	// Check if the command exists
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == args[0] {
			return rootCmd.Execute()
		}
	}

	// If command doesn't exist, output the error message and exit
	fmt.Fprintf(os.Stderr, "Unknown command '%s'\n", args[0])
	os.Exit(1)
	return nil // Never reached
}
