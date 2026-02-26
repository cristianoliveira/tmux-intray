package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/format"
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
		result = "summary"
	}
	return result
}

// ValidateStatusFormat validates status output format.
func ValidateStatusFormat(formatValue string) error {
	validFormats := map[string]bool{
		"summary": true,
		"levels":  true,
		"panes":   true,
		"json":    true,
	}

	if !validFormats[formatValue] {
		return fmt.Errorf("status: unknown format: %s", formatValue)
	}

	return nil
}

// Execute runs status behavior for a validated format.
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
	default:
		return fmt.Errorf("status: unknown format: %s", formatValue)
	}
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
