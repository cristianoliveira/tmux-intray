package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/formatter"
)

// StatusClient defines dependencies for status command.
type StatusClient interface {
	EnsureTmuxRunning() bool
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
}

// StatusUseCase coordinates status behavior.
type StatusUseCase struct {
	client StatusClient
}

// NewStatusUseCase creates a status use-case.
func NewStatusUseCase(client StatusClient) *StatusUseCase {
	if client == nil {
		panic("NewStatusUseCase: client dependency cannot be nil")
	}

	return &StatusUseCase{client: client}
}

// DetermineStatusFormat resolves effective format preserving CLI precedence.
func DetermineStatusFormat(formatFlag, envFormat string, flagChanged bool) string {
	result := formatFlag
	if !flagChanged && envFormat != "" {
		result = envFormat
	}
	if result == "" {
		result = "compact"
	}
	return result
}

// Execute runs status behavior for presets, legacy formats, and custom templates.
func (u *StatusUseCase) Execute(formatValue string, w io.Writer) error {
	if !u.client.EnsureTmuxRunning() {
		return fmt.Errorf("tmux not running")
	}

	switch formatValue {
	case "summary":
		return formatSummary(u.client, w)
	case "levels":
		return formatLevels(u.client, w)
	case "panes":
		return formatPanes(u.client, w)
	case "json":
		return formatJSON(u.client, w)
	}

	registry := formatter.NewPresetRegistry()
	if preset, err := registry.Get(formatValue); err == nil {
		return runStatusWithTemplate(u.client, preset.Template, w)
	}

	return runStatusWithTemplate(u.client, formatValue, w)
}

// CountByState counts notifications for a state.
func CountByState(client StatusClient, state string) int {
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

// CountByLevel counts active notifications by level.
func CountByLevel(client StatusClient) (info, warning, errCount, critical int) {
	lines, err := client.ListNotifications("active", "", "", "", "", "", "", "")
	if err != nil || lines == "" {
		return
	}

	info, warning, errCount, critical, _ = format.ParseCountsByLevel(lines)
	return
}

// PaneCounts returns active notification counts by pane key.
func PaneCounts(client StatusClient) map[string]int {
	lines, err := client.ListNotifications("active", "", "", "", "", "", "", "")
	if err != nil || lines == "" {
		return make(map[string]int)
	}

	return format.ParsePaneCounts(lines)
}

func buildVariableContext(client StatusClient) formatter.VariableContext {
	active := CountByState(client, "active")
	dismissed := CountByState(client, "dismissed")
	read := CountByState(client, "dismissed")
	infoCount, warningCount, errCount, criticalCount := CountByLevel(client)

	latestMsg := ""
	lines, _ := client.ListNotifications("active", "", "", "", "", "", "", "")
	if lines != "" {
		fields := strings.Split(strings.Split(lines, "\n")[0], "\t")
		if len(fields) > 6 {
			latestMsg = fields[6]
		}
	}

	highestSeverity := domain.LevelInfo
	if criticalCount > 0 {
		highestSeverity = domain.LevelCritical
	} else if errCount > 0 {
		highestSeverity = domain.LevelError
	} else if warningCount > 0 {
		highestSeverity = domain.LevelWarning
	}

	return formatter.VariableContext{
		UnreadCount:     active,
		TotalCount:      active,
		ReadCount:       read,
		ActiveCount:     active,
		DismissedCount:  dismissed,
		InfoCount:       infoCount,
		WarningCount:    warningCount,
		ErrorCount:      errCount,
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

func runStatusWithTemplate(client StatusClient, template string, w io.Writer) error {
	ctx := buildVariableContext(client)
	engine := formatter.NewTemplateEngine()
	result, err := engine.Substitute(template, ctx)
	if err != nil {
		return fmt.Errorf("template substitution error: %w", err)
	}
	_, err = fmt.Fprintln(w, result)
	return err
}

func formatSummary(client StatusClient, w io.Writer) error {
	active := CountByState(client, "active")
	if active == 0 {
		return format.FormatSummary(w, 0, 0, 0, 0, 0)
	}

	info, warning, errCount, critical := CountByLevel(client)
	return format.FormatSummary(w, active, info, warning, errCount, critical)
}

func formatLevels(client StatusClient, w io.Writer) error {
	info, warning, errCount, critical := CountByLevel(client)
	return format.FormatLevels(w, info, warning, errCount, critical)
}

func formatPanes(client StatusClient, w io.Writer) error {
	return format.FormatPanes(w, PaneCounts(client))
}

func formatJSON(client StatusClient, w io.Writer) error {
	active := CountByState(client, "active")
	info, warning, errCount, critical := CountByLevel(client)
	return format.FormatJSON(w, active, info, warning, errCount, critical, PaneCounts(client))
}
