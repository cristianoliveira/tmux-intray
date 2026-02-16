/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/status"
)

// statusPanelEnsureTmuxRunningFunc is the function used to ensure tmux is running. Can be changed for testing.
var statusPanelEnsureTmuxRunningFunc = func() bool {
	return core.EnsureTmuxRunning()
}

// statusPanelGetActiveCountFunc is the function used to get active notification count.
var statusPanelGetActiveCountFunc = func() int {
	return fileStorage.GetActiveCount()
}

// statusPanelListNotificationsFunc is the function used to list notifications.
var statusPanelListNotificationsFunc = func(stateFilter string) string {
	result, _ := fileStorage.ListNotifications(stateFilter, "", "", "", "", "", "", "")
	return result
}

// statusPanelGetConfigBoolFunc is the function used to get boolean config.
var statusPanelGetConfigBoolFunc = func(key string, defaultValue bool) bool {
	return config.GetBool(key, defaultValue)
}

// statusPanelGetConfigStringFunc is the function used to get string config.
var statusPanelGetConfigStringFunc = func(key, defaultValue string) string {
	return config.Get(key, defaultValue)
}

// getCountsByLevel returns counts of active notifications per level.
func getCountsByLevel() (info, warning, error, critical int, err error) {
	lines := statusPanelListNotificationsFunc("active")
	if lines == "" {
		return 0, 0, 0, 0, nil
	}
	return format.ParseCountsByLevel(lines)
}

// parseLevelColors parses the level_colors config into a map.
func parseLevelColors() map[string]string {
	colorsStr := statusPanelGetConfigStringFunc("level_colors", "info:green,warning:yellow,error:red,critical:magenta")
	m := make(map[string]string)
	pairs := strings.Split(colorsStr, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) == 2 {
			level := strings.TrimSpace(parts[0])
			color := strings.TrimSpace(parts[1])
			m[level] = color
		}
	}
	return m
}

// getLevelColor returns the tmux color code for a level.
func getLevelColor(level string) string {
	m := parseLevelColors()
	color, ok := m[level]
	if !ok {
		return ""
	}
	return color
}

// formatCompact returns compact format output.
func formatCompact(total, info, warning, error, critical int) string {
	if total == 0 {
		return ""
	}
	// Determine highest severity level present
	highestLevel := "info"
	if critical > 0 {
		highestLevel = "critical"
	} else if error > 0 {
		highestLevel = "error"
	} else if warning > 0 {
		highestLevel = "warning"
	}
	color := getLevelColor(highestLevel)
	icon := "🔔"
	if color != "" {
		return fmt.Sprintf("#[fg=%s]%s %d#[default]", color, icon, total)
	}
	return fmt.Sprintf("%s %d", icon, total)
}

// formatDetailed returns detailed format output.
func formatDetailed(total, info, warning, error, critical int) string {
	if total == 0 {
		return ""
	}
	var output strings.Builder
	if info > 0 {
		color := getLevelColor("info")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]i:%d#[default] ", color, info))
		} else {
			output.WriteString(fmt.Sprintf("i:%d ", info))
		}
	}
	if warning > 0 {
		color := getLevelColor("warning")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]w:%d#[default] ", color, warning))
		} else {
			output.WriteString(fmt.Sprintf("w:%d ", warning))
		}
	}
	if error > 0 {
		color := getLevelColor("error")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]e:%d#[default] ", color, error))
		} else {
			output.WriteString(fmt.Sprintf("e:%d ", error))
		}
	}
	if critical > 0 {
		color := getLevelColor("critical")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]c:%d#[default] ", color, critical))
		} else {
			output.WriteString(fmt.Sprintf("c:%d ", critical))
		}
	}
	// Trim trailing space
	result := output.String()
	if len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}
	return result
}

// Run executes the status-panel command with given options.
// Returns the formatted output string (may be empty) and any error.
func Run(opts status.StatusPanelOptions) (string, error) {
	// If disabled, return empty output
	if !opts.Enabled {
		return "", nil
	}

	// Ensure tmux is running (silently fail if not)
	if !statusPanelEnsureTmuxRunningFunc() {
		return "", nil
	}

	// Get active count
	total := statusPanelGetActiveCountFunc()
	if total == 0 {
		return "", nil
	}

	// Get counts by level
	info, warning, error, critical, err := getCountsByLevel()
	if err != nil {
		return "", err
	}

	// Determine format (default to compact if empty)
	format := opts.Format
	if format == "" {
		format = statusPanelGetConfigStringFunc("status_format", "compact")
	}

	// Format output
	switch format {
	case "compact":
		return formatCompact(total, info, warning, error, critical), nil
	case "detailed":
		return formatDetailed(total, info, warning, error, critical), nil
	case "count-only":
		return formatCountOnly(total), nil
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}
