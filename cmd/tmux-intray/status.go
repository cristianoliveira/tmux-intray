/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"io"
	"os"

	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
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
		Long: `Show notification status summary with template-based formatting.

USAGE:
    tmux-intray status [OPTIONS]

OPTIONS:
    --format=<format>    Output format: preset name or custom template (default: compact)

PRESETS / FORMATS (6):
    compact      [{{unread-count}}] {{latest-message}}
    detailed     {{unread-count}} unread, {{read-count}} read | Latest: {{latest-message}}
    json         Special JSON output with counts and pane breakdown
    count-only   {{unread-count}}
    levels       Special multi-line severity count output
    panes        Special pane-count output

VARIABLES (13):
    {{unread-count}}      Number of active notifications
    {{active-count}}      Alias for unread-count
    {{total-count}}       Alias for unread-count
    {{read-count}}        Number of dismissed notifications
    {{dismissed-count}}   Number of dismissed notifications
    {{latest-message}}    Text of most recent active notification
    {{has-unread}}        true/false if any active exist
    {{has-active}}        true/false if any active exist
    {{has-dismissed}}     true/false if any dismissed exist
    {{highest-severity}}  Severity level (1=critical, 2=error, 3=warning, 4=info)
    {{session-list}}      Sessions with active notifications
    {{window-list}}       Windows with active notifications
    {{pane-list}}         Panes with active notifications

LEVEL VARIABLES (4):
    {{critical-count}}    Number of critical notifications
    {{error-count}}       Number of error notifications
    {{warning-count}}     Number of warning notifications
    {{info-count}}        Number of info notifications

EXAMPLES:
    tmux-intray status                    # compact: [0] message
    tmux-intray status --format=detailed  # detailed: 0 unread, 0 read | Latest: ...
    tmux-intray status --format=json      # JSON: {"unread":0,"total":0,...}
    tmux-intray status --format='Alerts: {{critical-count}}'
    tmux-intray status --format='{{unread-count}} new messages'
    tmux-intray status --format='C:{{critical-count}} E:{{error-count}} W:{{warning-count}}'
    tmux-intray status --format='Level {{highest-severity}}'

See docs/status-guide.md for detailed documentation and more examples.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
	return appcore.DetermineStatusFormat(formatFlag, os.Getenv("TMUX_INTRAY_STATUS_FORMAT"), cmd.Flag("format").Changed)
}

// runStatusCommandWithFormat executes the status command with format support.
func runStatusCommandWithFormat(client statusClient, format string, w io.Writer) error {
	useCase := appcore.NewStatusUseCase(client)
	return useCase.Execute(format, w)
}

func countByLevel(client statusClient) (info, warning, errCount, critical int) {
	return appcore.CountByLevel(client)
}

func paneCounts(client statusClient) map[string]int {
	return appcore.PaneCounts(client)
}
