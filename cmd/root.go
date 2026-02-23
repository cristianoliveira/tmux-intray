/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/cristianoliveira/tmux-intray/internal/logging"
	"github.com/cristianoliveira/tmux-intray/internal/version"
	"github.com/spf13/cobra"
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
	args := os.Args[1:]

	// Disable structured logging for TUI command to avoid JSON output interfering with display
	isTUICommand := len(args) > 0 && args[0] == "tui"
	if isTUICommand {
		colors.DisableStructuredLogging()
		logging.DisableStructuredLogging()
		defer colors.EnableStructuredLogging()
		defer logging.EnableStructuredLogging()
	}

	colors.StructuredInfo("cli/root", "execute", "started", nil, "", map[string]interface{}{"arg_count": len(args)})
	defer hooks.WaitForPendingHooks()
	if err := hooks.Init(); err != nil {
		colors.StructuredError("cli/root", "execute", "hooks_init_failed", err, "", nil)
		return err
	}

	// No args? show help by routing through cobra help command
	if len(args) == 0 {
		RootCmd.SetArgs([]string{"help"})
		colors.StructuredInfo("cli/command", "execute", "started", nil, "help", map[string]interface{}{"arg_count": 0})
		err := RootCmd.Execute()
		if err != nil {
			colors.StructuredError("cli/root", "execute", "help_command_failed", err, "", nil)
			colors.StructuredError("cli/command", "execute", "failed", err, "help", nil)
		} else {
			colors.StructuredInfo("cli/root", "execute", "help_completed", nil, "", nil)
			colors.StructuredInfo("cli/command", "execute", "completed", nil, "help", nil)
		}
		return err
	}

	// Pass explicit help flags through cobra
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		RootCmd.SetArgs(args)
		colors.StructuredInfo("cli/command", "execute", "started", nil, "help", map[string]interface{}{"arg_count": len(args)})
		err := RootCmd.Execute()
		if err != nil {
			colors.StructuredError("cli/root", "execute", "help_flag_failed", err, "", nil)
			colors.StructuredError("cli/command", "execute", "failed", err, "help", nil)
		} else {
			colors.StructuredInfo("cli/root", "execute", "help_flag_completed", nil, "", nil)
			colors.StructuredInfo("cli/command", "execute", "completed", nil, "help", nil)
		}
		return err
	}

	// Check if the command exists (including aliases)
	targetCmd, _, err := RootCmd.Find(args)
	if err == nil && targetCmd != nil {
		RootCmd.SetArgs(args)
		commandPath := targetCmd.CommandPath()
		colors.StructuredInfo("cli/command", "execute", "started", nil, commandPath, map[string]interface{}{"arg_count": len(args)})
		err := RootCmd.Execute()
		if err != nil {
			colors.StructuredError("cli/root", "execute", "command_failed", err, "", map[string]interface{}{"command": commandPath})
			colors.StructuredError("cli/command", "execute", "failed", err, commandPath, nil)
		} else {
			colors.StructuredInfo("cli/root", "execute", "command_completed", nil, "", map[string]interface{}{"command": commandPath})
			colors.StructuredInfo("cli/command", "execute", "completed", nil, commandPath, nil)
		}
		return err
	}

	// Unknown command
	colors.StructuredError("cli/root", "execute", "unknown_command", nil, "", map[string]interface{}{"command": args[0]})
	colors.StructuredError("cli/command", "execute", "unknown", nil, args[0], nil)
	fmt.Fprintf(os.Stderr, "Unknown command '%s'\n", args[0])
	os.Exit(1)
	return nil
}
