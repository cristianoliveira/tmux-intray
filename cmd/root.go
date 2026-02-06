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

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
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
// This is called by main.main(). It only needs to happen once to the RootCmd.
func ExecuteOriginal() error {
	defer hooks.WaitForPendingHooks()
	hooks.Init()
	return RootCmd.Execute()
}

func init() {
	// Set version for use in help output
	RootCmd.Version = Version

	// Hide the completion command
	RootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Custom help is provided by the help command; default help template is used for --help

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags which, if defined here,
	// will be global for your application.

	// RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tmux-intray.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	defer hooks.WaitForPendingHooks()
	hooks.Init()

	args := os.Args[1:]

	// No args? show help by routing through cobra help command
	if len(args) == 0 {
		RootCmd.SetArgs([]string{"help"})
		return RootCmd.Execute()
	}

	// Pass explicit help flags through cobra
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		RootCmd.SetArgs(args)
		return RootCmd.Execute()
	}

	// Check if the command exists (including aliases)
	targetCmd, _, err := RootCmd.Find(args)
	if err == nil && targetCmd != nil {
		RootCmd.SetArgs(args)
		return RootCmd.Execute()
	}

	fmt.Fprintf(os.Stderr, "Unknown command '%s'\n", args[0])
	os.Exit(1)
	return nil
}
