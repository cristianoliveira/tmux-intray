/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
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
func Execute() error {
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
