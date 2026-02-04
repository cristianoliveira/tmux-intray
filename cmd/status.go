/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
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
	Run: runStatus,
}

var statusFormat string

// statusOutputWriter is the writer used by PrintStatus. Can be changed for testing.
var statusOutputWriter io.Writer = os.Stdout

// statusListFunc is the function used to retrieve notifications. Can be changed for testing.
var statusListFunc = func(state, level, session, window, pane, olderThan, newerThan string) string {
	return storage.ListNotifications(state, level, session, window, pane, olderThan, newerThan)
}

// statusActiveCountFunc is the function used to get active count. Can be changed for testing.
var statusActiveCountFunc = func() int {
	return storage.GetActiveCount()
}

// PrintStatus prints status summary according to the provided format.
func PrintStatus(format string) {
	if statusOutputWriter == nil {
		statusOutputWriter = os.Stdout
	}
	printStatus(format, statusOutputWriter)
}

func printStatus(format string, w io.Writer) {
	switch format {
	case "summary":
		formatSummary(w)
	case "levels":
		formatLevels(w)
	case "panes":
		formatPanes(w)
	case "json":
		formatJSON(w)
	default:
		fmt.Fprintf(w, "Unknown format: %s\n", format)
	}
}

// countByState returns the number of notifications with given state.
func countByState(state string) int {
	lines := statusListFunc(state, "", "", "", "", "", "")
	if lines == "" {
		return 0
	}
	// Count non-empty lines
	count := 0
	for _, line := range strings.Split(lines, "\n") {
		if line != "" {
			count++
		}
	}
	return count
}

// countByLevel returns counts per level for active notifications.
func countByLevel() (info, warning, error, critical int) {
	lines := statusListFunc("active", "", "", "", "", "", "")
	if lines == "" {
		return
	}
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) <= 8 {
			continue
		}
		level := fields[8]
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

// paneCounts returns map of pane key to count for active notifications.
func paneCounts() map[string]int {
	counts := make(map[string]int)
	lines := statusListFunc("active", "", "", "", "", "", "")
	if lines == "" {
		return counts
	}
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) <= 5 {
			continue
		}
		session := fields[3]
		window := fields[4]
		pane := fields[5]
		key := fmt.Sprintf("%s:%s:%s", session, window, pane)
		counts[key]++
	}
	return counts
}

func formatSummary(w io.Writer) {
	active := countByState("active")
	dismissed := countByState("dismissed")
	total := active + dismissed
	if total == 0 {
		fmt.Fprintf(w, "%sNo notifications%s\n", colors.Blue, colors.Reset)
		return
	}
	fmt.Fprintf(w, "%sActive notifications: %d%s\n", colors.Blue, active, colors.Reset)
	fmt.Fprintf(w, "%sDismissed notifications: %d%s\n", colors.Blue, dismissed, colors.Reset)
	fmt.Fprintf(w, "%sTotal notifications: %d%s\n", colors.Blue, total, colors.Reset)
	if active > 0 {
		info, warning, error, critical := countByLevel()
		fmt.Fprintf(w, "%s  info: %d, warning: %d, error: %d, critical: %d%s\n", colors.Blue, info, warning, error, critical, colors.Reset)
	}
}

func formatLevels(w io.Writer) {
	info, warning, error, critical := countByLevel()
	fmt.Fprintf(w, "info:%d\nwarning:%d\nerror:%d\ncritical:%d\n", info, warning, error, critical)
}

func formatPanes(w io.Writer) {
	counts := paneCounts()
	for pane, count := range counts {
		fmt.Fprintf(w, "%s:%d\n", pane, count)
	}
}

func formatJSON(w io.Writer) {
	fmt.Fprintln(w, "JSON format not yet implemented")
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().StringVar(&statusFormat, "format", "summary", "Output format: summary, levels, panes, json")
}

func runStatus(cmd *cobra.Command, args []string) {
	// Ensure tmux is running (mirror bash script behavior)
	if !core.EnsureTmuxRunning() {
		colors.Error("tmux is not running")
		return
	}

	PrintStatus(statusFormat)
}
