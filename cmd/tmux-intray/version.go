/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/spf13/cobra"
)

type versionClient interface {
	Version() string
}

// NewVersionCmd creates the version command with explicit dependencies.
func NewVersionCmd(client versionClient) *cobra.Command {
	if client == nil {
		panic("NewVersionCmd: client dependency cannot be nil")
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Show the current version of tmux-intray.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "tmux-intray version %s\n", client.Version())
			return nil
		},
	}

	return versionCmd
}

// versionCmd represents the version command
var versionCmd = NewVersionCmd(coreClient)

func init() {
	cmd.RootCmd.AddCommand(versionCmd)
}
