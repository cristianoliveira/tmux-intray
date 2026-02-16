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
)

// Field indices matching storage package (some constants defined in jump.go)
const (
	fieldPaneCreated = 7
	fieldLevel       = 8
)

// StatusPanelOptions holds parameters for status panel.
type StatusPanelOptions struct {
	Format  string // "compact", "detailed", "count-only"
	Enabled bool   // true to enable output
}

type statusPanelClient interface {
	EnsureTmuxRunning() bool
	GetActiveCount() int
	ListNotifications(stateFilter string) string
	GetConfigBool(key string, defaultValue bool) bool
	GetConfigString(key, defaultValue string) string
}

var (
	statusPanelFormat  string
	statusPanelEnabled string
)

// defaultStatusPanelClient is the default implementation.
type defaultStatusPanelClient struct{}

func (d *defaultStatusPanelClient) EnsureTmuxRunning() bool {
	return core.EnsureTmuxRunning()
}

func (d *defaultStatusPanelClient) GetActiveCount() int {
	return fileStorage.GetActiveCount()
}

func (d *defaultStatusPanelClient) ListNotifications(stateFilter string) string {
	result, _ := fileStorage.ListNotifications(stateFilter, "", "", "", "", "", "", "")
	return result
}

func (d *defaultStatusPanelClient) GetConfigBool(key string, defaultValue bool) bool {
	return config.GetBool(key, defaultValue)
}

func (d *defaultStatusPanelClient) GetConfigString(key, defaultValue string) string {
	return config.Get(key, defaultValue)
}

// RunStatusPanel executes the status-panel command with given options.
// Returns the formatted output string (may be empty) and any error.
func RunStatusPanel(client statusPanelClient, opts StatusPanelOptions) (string, error) {
	// If disabled, return empty output
	if !opts.Enabled {
		return "", nil
	}

	// Ensure tmux is running (silently fail if not)
	if !client.EnsureTmuxRunning() {
		return "", nil
	}

	// Get active count
	total := client.GetActiveCount()
	if total == 0 {
		return "", nil
	}

	// Get counts by level
	info, warning, error, critical, err := getCountsByLevelWithClient(client)
	if err != nil {
		return "", err
	}

	// Determine format (default to compact if empty)
	format := opts.Format
	if format == "" {
		format = client.GetConfigString("status_format", "compact")
	}

	// Format output
	switch format {
	case "compact":
		return formatCompactWithColors(client, total, info, warning, error, critical), nil
	case "detailed":
		return formatDetailedWithColors(client, total, info, warning, error, critical), nil
	case "count-only":
		return formatCountOnly(total), nil
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}

// getCountsByLevelWithClient returns counts of active notifications per level using the client.
func getCountsByLevelWithClient(client statusPanelClient) (info, warning, error, critical int, err error) {
	lines := client.ListNotifications("active")
	if lines == "" {
		return 0, 0, 0, 0, nil
	}
	return format.ParseCountsByLevel(lines)
}

// parseLevelColorsWithClient parses the level_colors config using the client.
func parseLevelColorsWithClient(client statusPanelClient) map[string]string {
	colorsStr := client.GetConfigString("level_colors", "info:green,warning:yellow,error:red,critical:magenta")
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

// getLevelColorWithClient returns the tmux color code for a level using the client.
func getLevelColorWithClient(client statusPanelClient, level string) string {
	m := parseLevelColorsWithClient(client)
	color, ok := m[level]
	if !ok {
		return ""
	}
	return color
}

// formatCompactWithColors returns compact format output using client for colors.
func formatCompactWithColors(client statusPanelClient, total, info, warning, error, critical int) string {
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
	color := getLevelColorWithClient(client, highestLevel)
	icon := "🔔"
	if color != "" {
		return fmt.Sprintf("#[fg=%s]%s %d#[default]", color, icon, total)
	}
	return fmt.Sprintf("%s %d", icon, total)
}

// formatDetailedWithColors returns detailed format output using client for colors.
func formatDetailedWithColors(client statusPanelClient, total, info, warning, error, critical int) string {
	if total == 0 {
		return ""
	}
	var output strings.Builder
	if info > 0 {
		color := getLevelColorWithClient(client, "info")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]i:%d#[default] ", color, info))
		} else {
			output.WriteString(fmt.Sprintf("i:%d ", info))
		}
	}
	if warning > 0 {
		color := getLevelColorWithClient(client, "warning")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]w:%d#[default] ", color, warning))
		} else {
			output.WriteString(fmt.Sprintf("w:%d ", warning))
		}
	}
	if error > 0 {
		color := getLevelColorWithClient(client, "error")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]e:%d#[default] ", color, error))
		} else {
			output.WriteString(fmt.Sprintf("e:%d ", error))
		}
	}
	if critical > 0 {
		color := getLevelColorWithClient(client, "critical")
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

// formatCountOnly returns count-only format output.
func formatCountOnly(total int) string {
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("%d", total)
}
