/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
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

			// Determine format: flag > environment variable > default
			format := formatFlag
			if !cmd.Flag("format").Changed {
				if envFormat := os.Getenv("TMUX_INTRAY_STATUS_FORMAT"); envFormat != "" {
					format = envFormat
				}
			}
			if format == "" {
				format = "summary"
			}

			// Validate format
			validFormats := map[string]bool{
				"summary": true,
				"levels":  true,
				"panes":   true,
				"json":    true,
			}
			if !validFormats[format] {
				return fmt.Errorf("status: unknown format: %s", format)
			}

			// Output writer
			w := cmd.OutOrStdout()

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
		},
	}

	statusCmd.Flags().StringVar(&formatFlag, "format", "summary", "Output format: summary, levels, panes, json")
	return statusCmd
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
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) <= storage.FieldLevel {
			continue
		}
		level := fields[storage.FieldLevel]
		switch level {
		case "info":
			info++
		case "warning":
			warning++
		case "error":
			error++
		case "critical":
			critical++
		default:
			info++
		}
	}
	return
}

func paneCounts(client statusClient) map[string]int {
	counts := make(map[string]int)
	lines, err := client.ListNotifications("active", "", "", "", "", "", "", "")
	if err != nil || lines == "" {
		return counts
	}
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) <= storage.FieldPane {
			continue
		}
		session := fields[storage.FieldSession]
		window := fields[storage.FieldWindow]
		pane := fields[storage.FieldPane]
		key := fmt.Sprintf("%s:%s:%s", session, window, pane)
		counts[key]++
	}
	return counts
}

func formatSummary(client statusClient, w io.Writer) error {
	active := countByState(client, "active")
	if active == 0 {
		fmt.Fprintf(w, "No active notifications\n")
		return nil
	}
	fmt.Fprintf(w, "Active notifications: %d\n", active)
	info, warning, error, critical := countByLevel(client)
	fmt.Fprintf(w, "  info: %d, warning: %d, error: %d, critical: %d\n", info, warning, error, critical)
	return nil
}

func formatLevels(client statusClient, w io.Writer) error {
	info, warning, error, critical := countByLevel(client)
	fmt.Fprintf(w, "info:%d\nwarning:%d\nerror:%d\ncritical:%d\n", info, warning, error, critical)
	return nil
}

func formatPanes(client statusClient, w io.Writer) error {
	counts := paneCounts(client)
	for pane, count := range counts {
		fmt.Fprintf(w, "%s:%d\n", pane, count)
	}
	return nil
}

func formatJSON(client statusClient, w io.Writer) error {
	fmt.Fprintln(w, "JSON format not yet implemented")
	return nil
}
