/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/spf13/cobra"
)

type statusClient interface {
	EnsureTmuxRunning() bool
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
	GetActiveCount() int
}

// NewStatusCmd creates the status command with explicit dependencies.
func NewStatusCmd(client statusClient) *cobra.Command {
	if client == nil {
		panic("NewStatusCmd: client dependency cannot be nil")
	}

	var formatFlag string

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show notification status summary",
		Long: `Show notification status summary.

USAGE:
    tmux-intray status [OPTIONS]

OPTIONS:
    --format=<format>    Output format: summary, levels, panes, json (default: summary)
    -h, --help           Show this help

EXAMPLES:
    tmux-intray status               # Show summary
    tmux-intray status --format=levels # Show counts by level
    tmux-intray status --format=panes  # Show counts by pane`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure tmux is running (mirror bash script behavior)
			if !client.EnsureTmuxRunning() {
				return fmt.Errorf("tmux not running")
			}

			format := determineStatusFormat(cmd, formatFlag)
			if err := validateStatusFormat(format); err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			return runStatusCommand(client, format, w)
		},
	}

	statusCmd.Flags().StringVar(&formatFlag, "format", "summary", "Output format: summary, levels, panes, json")
	return statusCmd
}

// determineStatusFormat determines the output format.
func determineStatusFormat(cmd *cobra.Command, formatFlag string) string {
	format := formatFlag
	if !cmd.Flag("format").Changed {
		if envFormat := os.Getenv("TMUX_INTRAY_STATUS_FORMAT"); envFormat != "" {
			format = envFormat
		}
	}
	if format == "" {
		format = "summary"
	}
	return format
}

// validateStatusFormat validates the status format.
func validateStatusFormat(format string) error {
	validFormats := map[string]bool{
		"summary": true,
		"levels":  true,
		"panes":   true,
		"json":    true,
	}
	if !validFormats[format] {
		return fmt.Errorf("status: unknown format: %s", format)
	}
	return nil
}

// runStatusCommand executes the status command with the given format.
func runStatusCommand(client statusClient, format string, w io.Writer) error {
	switch format {
	case "summary":
		return formatSummary(client, w)
	case "levels":
		return formatLevels(client, w)
	case "panes":
		return formatPanes(client, w)
	case "json":
		return formatJSON(client, w)
	default:
		return fmt.Errorf("status: unknown format: %s", format)
	}
}

// statusCmd represents the status command.
var statusCmd = NewStatusCmd(coreClient)

func init() {
	cmd.RootCmd.AddCommand(statusCmd)
}

// Helper functions

func countByState(client statusClient, state string) int {
	lines, err := client.ListNotifications(state, "", "", "", "", "", "", "")
	if err != nil || lines == "" {
		return 0
	}
	count := 0
	for _, line := range strings.Split(lines, "\n") {
		if line != "" {
			count++
		}
	}
	return count
}

func countByLevel(client statusClient) (info, warning, error, critical int) {
	lines, err := client.ListNotifications("active", "", "", "", "", "", "", "")
	if err != nil || lines == "" {
		return
	}
	info, warning, error, critical, _ = format.ParseCountsByLevel(lines)
	return
}

func paneCounts(client statusClient) map[string]int {
	lines, err := client.ListNotifications("active", "", "", "", "", "", "", "")
	if err != nil || lines == "" {
		return make(map[string]int)
	}
	return format.ParsePaneCounts(lines)
}

func formatSummary(client statusClient, w io.Writer) error {
	active := countByState(client, "active")
	if active == 0 {
		return format.FormatSummary(w, 0, 0, 0, 0, 0)
	}
	info, warning, error, critical := countByLevel(client)
	return format.FormatSummary(w, active, info, warning, error, critical)
}

func formatLevels(client statusClient, w io.Writer) error {
	info, warning, error, critical := countByLevel(client)
	return format.FormatLevels(w, info, warning, error, critical)
}

func formatPanes(client statusClient, w io.Writer) error {
	counts := paneCounts(client)
	return format.FormatPanes(w, counts)
}

func formatJSON(client statusClient, w io.Writer) error {
	active := countByState(client, "active")
	info, warning, error, critical := countByLevel(client)
	counts := paneCounts(client)
	return format.FormatJSON(w, active, info, warning, error, critical, counts)
}
