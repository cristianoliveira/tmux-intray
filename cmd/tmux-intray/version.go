package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/version"

	"github.com/spf13/cobra"
)

// versionOutputWriter is the writer used by PrintVersion. Can be changed for testing.
var versionOutputWriter io.Writer = os.Stdout

// PrintVersion prints the version information to stdout.
func PrintVersion() {
	fmt.Fprintf(versionOutputWriter, "tmux-intray version %s\n", version.String())
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Show the current version of tmux-intray.`,
	Run: func(cmd *cobra.Command, args []string) {
		PrintVersion()
	},
}

func init() {
	cmd.RootCmd.AddCommand(versionCmd)
}
