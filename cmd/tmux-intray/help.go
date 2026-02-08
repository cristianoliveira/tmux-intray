/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/cristianoliveira/tmux-intray/cmd"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/spf13/cobra"
)

// helpCmd represents the help command
// outputWriter is the writer used by PrintHelp. Can be changed for testing.
var outputWriter io.Writer = io.Writer(nil)

// PrintHelp prints the help information for the given root command.
func PrintHelp(cmd *cobra.Command) {
	if outputWriter == nil {
		outputWriter = cmd.OutOrStdout()
	}
	printHelp(cmd, outputWriter)
}

func printHelp(cmd *cobra.Command, w io.Writer) {
	// DEBUG: log which command we're helping
	// fmt.Fprintf(os.Stderr, "DEBUG: cmd.Use=%s, parent=%v\n", cmd.Use, cmd.Parent())

	// Order of commands as in bash help
	commandOrder := []string{
		"add",
		"list",
		"dismiss",
		"clear",
		"cleanup",
		"migrate",
		"toggle",
		"jump",
		"status",
		"status-panel",
		"follow",
		"help",
		"version",
	}

	// Build command descriptions with colors
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
		// Colorize command name in cyan, description in green
		cmdLines = append(cmdLines, fmt.Sprintf("    %s%-16s%s %s%s%s", colors.Cyan, use, colors.Reset, colors.Green, short, colors.Reset))
	}

	// Colorized headers
	headerColor := colors.Blue
	reset := colors.Reset

	// Get version from root command
	version := cmd.Version
	if version == "" {
		version = "0.0.0"
	}

	helpText := fmt.Sprintf(`%stmux-intray v%s%s

%sA quiet inbox for things that happen while you're not looking.%s

%sUSAGE:%s
    tmux-intray [COMMAND] [OPTIONS]

%sCOMMANDS:%s
%s

%sOPTIONS:%s
    -h, --help      Show help message
`, headerColor, version, reset, colors.Cyan, reset, headerColor, reset, headerColor, reset, strings.Join(cmdLines, "\n"), headerColor, reset)
	fmt.Fprint(w, helpText)
}

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Show this help message",
	Long:  `Show this help message.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			PrintHelp(cmd.Root())
			return
		}
		// Find the subcommand
		targetCmd, _, err := cmd.Root().Find(args)
		if err != nil || targetCmd == nil {
			// fallback to root help
			PrintHelp(cmd.Root())
			return
		}
		// Call help for that command (will show its Long description)
		targetCmd.Help()
	},
}

func init() {
	cmd.RootCmd.SetHelpCommand(helpCmd)
}
