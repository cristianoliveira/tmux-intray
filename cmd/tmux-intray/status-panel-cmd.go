/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/spf13/cobra"
)

// NewStatusPanelCmd creates the status-panel command with explicit dependencies.
func NewStatusPanelCmd(client statusPanelClient) *cobra.Command {
	if client == nil {
		panic("NewStatusPanelCmd: client dependency cannot be nil")
	}

	cmd := &cobra.Command{
		Use:   "status-panel",
		Short: "Status bar indicator script (for tmux status-right)",
		Long: `Status bar indicator script (for tmux status-right).

USAGE:
    tmux-intray status-panel [OPTIONS]

OPTIONS:
    --format=<format>    Output format: compact, detailed, count-only (default: compact)
    --enabled=<0|1>      Enable/disable status indicator (default: 1)
    -h, --help           Show this help

DESCRIPTION:
    This script is designed to be used in tmux status-right configuration.
    Example: set -g status-right "#(tmux-intray status-panel) %H:%M"

    The script outputs a formatted string showing notification counts.
    When clicked, it can trigger the list command (via tmux bindings).`,
		RunE: func(c *cobra.Command, args []string) error {
			// Determine format
			format := statusPanelFormat

			// Determine enabled
			enabled := true // default
			if statusPanelEnabled != "" {
				val := strings.ToLower(statusPanelEnabled)
				switch val {
				case "0", "false", "no", "off":
					enabled = false
				case "1", "true", "yes", "on":
					enabled = true
				default:
					colors.Error("invalid value for --enabled, must be 0 or 1")
					os.Exit(1)
				}
			}

			opts := StatusPanelOptions{
				Format:  format,
				Enabled: enabled,
			}
			output, err := RunStatusPanel(client, opts)
			if err != nil {
				colors.Error(err.Error())
				os.Exit(1)
			}
			if output != "" {
				fmt.Print(output)
			}
			// No output means empty string (tmux will show nothing).
			return nil
		},
	}

	cmd.Flags().StringVar(&statusPanelFormat, "format", "", "Output format: compact, detailed, count-only")
	cmd.Flags().StringVar(&statusPanelEnabled, "enabled", "", "Enable/disable status indicator (0 or 1)")

	return cmd
}

// statusPanelCmd represents the status-panel command
var statusPanelCmd = NewStatusPanelCmd(&defaultStatusPanelClient{})

func init() {
	cmd.RootCmd.AddCommand(statusPanelCmd)
}
