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
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/formatter"
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
    --format=<format>    Output format: preset name or custom template (default: compact)
                         Presets: compact, detailed, json, count-only, levels, panes
                         Custom: ${variable} syntax, e.g., "${unread-count} unread"
    -h, --help           Show this help

EXAMPLES:
    tmux-intray status                           # Show compact format (default)
    tmux-intray status --format=detailed         # Show detailed format
    tmux-intray status --format=json             # Show JSON format
    tmux-intray status --format='${unread-count}' # Custom template`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure tmux is running (mirror bash script behavior)
			if !client.EnsureTmuxRunning() {
				return fmt.Errorf("tmux not running")
			}

			format := determineStatusFormat(cmd, formatFlag)

			w := cmd.OutOrStdout()
			return runStatusCommandWithFormat(client, format, w)
		},
	}

	statusCmd.Flags().StringVar(&formatFlag, "format", "compact", "Output format: preset name or custom template")
	return statusCmd
}

// determineStatusFormat determines the output format, preferring flag over env.
func determineStatusFormat(cmd *cobra.Command, formatFlag string) string {
	format := formatFlag
	if !cmd.Flag("format").Changed {
		if envFormat := os.Getenv("TMUX_INTRAY_STATUS_FORMAT"); envFormat != "" {
			format = envFormat
		}
	}
	if format == "" {
		format = "compact"
	}
	return format
}

// runStatusCommandWithFormat executes the status command with format support.
// It handles both preset names and custom templates.
func runStatusCommandWithFormat(client statusClient, format string, w io.Writer) error {
	// Handle legacy format names for backward compatibility
	switch format {
	case "summary":
		return formatSummary(client, w)
	case "levels":
		return formatLevels(client, w)
	case "panes":
		return formatPanes(client, w)
	case "json":
		return formatJSON(client, w)
	}

	// Check if it's a formatter preset (new system)
	registry := formatter.NewPresetRegistry()
	if preset, err := registry.Get(format); err == nil {
		// It's a preset, use the template from it
		return runStatusWithTemplate(client, preset.Template, w)
	}

	// Otherwise treat it as a custom template
	return runStatusWithTemplate(client, format, w)
}

// runStatusWithTemplate executes status with a template string.
func runStatusWithTemplate(client statusClient, template string, w io.Writer) error {
	// Create the variable context with current status data
	ctx := buildVariableContext(client)

	// Create template engine and substitute
	engine := formatter.NewTemplateEngine()
	result, err := engine.Substitute(template, ctx)
	if err != nil {
		return fmt.Errorf("template substitution error: %w", err)
	}

	_, err = fmt.Fprintln(w, result)
	return err
}

// buildVariableContext creates a VariableContext from current status data.
func buildVariableContext(client statusClient) formatter.VariableContext {
	active := countByState(client, "active")
	dismissed := countByState(client, "dismissed")
	read := countByState(client, "dismissed") // read items are dismissed
	infoCount, warningCount, errorCount, criticalCount := countByLevel(client)

	// Get latest message
	latestMsg := ""
	lines, _ := client.ListNotifications("active", "", "", "", "", "", "", "")
	if lines != "" {
		fields := strings.Split(strings.Split(lines, "\n")[0], "\t")
		if len(fields) > 6 {
			latestMsg = fields[6]
		}
	}

	// Determine highest severity
	highestSeverity := domain.LevelInfo
	if criticalCount > 0 {
		highestSeverity = domain.LevelCritical
	} else if errorCount > 0 {
		highestSeverity = domain.LevelError
	} else if warningCount > 0 {
		highestSeverity = domain.LevelWarning
	}

	return formatter.VariableContext{
		UnreadCount:     active,
		TotalCount:      active, // alias for unread
		ReadCount:       read,
		ActiveCount:     active,
		DismissedCount:  dismissed,
		InfoCount:       infoCount,
		WarningCount:    warningCount,
		ErrorCount:      errorCount,
		CriticalCount:   criticalCount,
		LatestMessage:   latestMsg,
		HasUnread:       active > 0,
		HasActive:       active > 0,
		HasDismissed:    dismissed > 0,
		HighestSeverity: highestSeverity,
		SessionList:     "",
		WindowList:      "",
		PaneList:        "",
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
