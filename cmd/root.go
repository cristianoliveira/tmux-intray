/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/cristianoliveira/tmux-intray/internal/logging"
	"github.com/cristianoliveira/tmux-intray/internal/version"
	"github.com/spf13/cobra"
)

var logFilePath string

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
				_ = cmd.Help()
				return
			}
			_, _ = fmt.Fprintf(os.Stderr, "Unknown command '%s'\n", args[0])
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
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Unknown command '%s'\n", args[0])
		return fmt.Errorf("")
	}
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func ExecuteOriginal() error {
	defer hooks.WaitForPendingHooks()
	if err := hooks.Init(); err != nil {
		return err
	}
	return RootCmd.Execute()
}

func init() {
	// Set version for use in help output and --version flag
	RootCmd.Version = version.String()

	// Hide the completion command
	RootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Ensure default help command is enabled (since we removed custom help)
	RootCmd.InitDefaultHelpCmd()

	// Log file flag
	RootCmd.PersistentFlags().StringVar(&logFilePath, "log-file", "", "log file path (default empty, logs to stderr)")
	RootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if logFilePath != "" {
			os.Setenv("TMUX_INTRAY_LOG_FILE", logFilePath)
		}
	}

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
	if err := hooks.Init(); err != nil {
		return err
	}
	// Initialize structured logging (if enabled via config)
	if err := logging.InitGlobal(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
	}
	defer logging.ShutdownGlobal()

	args := os.Args[1:]

	// DEBUG: print commands
	// fmt.Fprintf(os.Stderr, "DEBUG: commands: %v\n", RootCmd.Commands())

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
