package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/version"

	"github.com/spf13/cobra"
)

type versionClient interface {
	GetVersion() string
}

type defaultVersionClient struct{}

func (d *defaultVersionClient) GetVersion() string {
	return version.String()
}

// NewVersionCmd creates the version command with explicit dependencies.
func NewVersionCmd(client versionClient) *cobra.Command {
	if client == nil {
		panic("NewVersionCmd: client dependency cannot be nil")
	}

	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Show the current version of tmux-intray.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "tmux-intray version %s\n", client.GetVersion())
			return nil
		},
	}
}

// versionOutputWriter is the writer used by PrintVersion. Can be changed for testing.
var versionOutputWriter io.Writer = os.Stdout

// PrintVersion prints the version information to stdout.
func PrintVersion() {
	fmt.Fprintf(versionOutputWriter, "tmux-intray version %s\n", version.String())
}

// versionCmd represents the version command
var versionCmd = NewVersionCmd(&defaultVersionClient{})

func init() {
	cmd.RootCmd.AddCommand(versionCmd)
}
