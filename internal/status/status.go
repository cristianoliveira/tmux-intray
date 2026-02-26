/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/

// Package status provides status panel functionality for tmux-intray.
package status

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/formatter"
	"github.com/cristianoliveira/tmux-intray/internal/ports"
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

// StatusPanelClient defines the interface for status panel operations.
type StatusPanelClient interface {
	EnsureTmuxRunning() bool
	GetActiveCount() int
	ListNotifications(stateFilter string) string
	ports.ConfigProvider
}

var (
	// Format holds the output format flag value
	Format string
	// Enabled holds the enabled flag value
	Enabled string
)

// DefaultClient is a placeholder that implements StatusPanelClient.
// Real implementations should be created with proper storage injection.
type DefaultClient struct{}

func (d *DefaultClient) EnsureTmuxRunning() bool {
	return core.EnsureTmuxRunning()
}

func (d *DefaultClient) GetActiveCount() int {
	return 0
}

func (d *DefaultClient) ListNotifications(stateFilter string) string {
	return ""
}

func (d *DefaultClient) GetConfigBool(key string, defaultValue bool) bool {
	return config.GetBool(key, defaultValue)
}

func (d *DefaultClient) GetConfigString(key, defaultValue string) string {
	return config.Get(key, defaultValue)
}

// RunStatusPanel executes the status-panel command with given options.
// Returns the formatted output string (may be empty) and any error.
func RunStatusPanel(client StatusPanelClient, opts StatusPanelOptions) (string, error) {
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

	// Determine format (default to compact if empty)
	formatName := opts.Format
	if formatName == "" {
		formatName = client.GetConfigString("status_format", "compact")
	}

	// Build variable context
	ctx := format.BuildVariableContextForStatusPanel(client)

	// Format output based on format name
	switch formatName {
	case "compact":
		return formatCompactWithColors(client, ctx), nil
	case "detailed":
		return formatDetailedWithColors(client, ctx), nil
	case "count-only":
		return formatCountOnly(ctx.ActiveCount), nil
	default:
		return "", fmt.Errorf("unknown format: %s", formatName)
	}
}

// parseLevelColorsWithClient parses the level_colors config using the client.
func parseLevelColorsWithClient(client StatusPanelClient) map[string]string {
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
func getLevelColorWithClient(client StatusPanelClient, level string) string {
	m := parseLevelColorsWithClient(client)
	color, ok := m[level]
	if !ok {
		return ""
	}
	return color
}

// formatCompactWithColors returns compact format output using context for colors.
func formatCompactWithColors(client StatusPanelClient, ctx formatter.VariableContext) string {
	if ctx.ActiveCount == 0 {
		return ""
	}
	// Determine highest severity level present
	highestLevel := "info"
	if ctx.CriticalCount > 0 {
		highestLevel = "critical"
	} else if ctx.ErrorCount > 0 {
		highestLevel = "error"
	} else if ctx.WarningCount > 0 {
		highestLevel = "warning"
	}
	color := getLevelColorWithClient(client, highestLevel)
	icon := "🔔"
	if color != "" {
		return fmt.Sprintf("#[fg=%s]%s %d#[default]", color, icon, ctx.ActiveCount)
	}
	return fmt.Sprintf("%s %d", icon, ctx.ActiveCount)
}

// formatDetailedWithColors returns detailed format output using context for colors.
func formatDetailedWithColors(client StatusPanelClient, ctx formatter.VariableContext) string {
	if ctx.ActiveCount == 0 {
		return ""
	}
	var output strings.Builder
	if ctx.InfoCount > 0 {
		color := getLevelColorWithClient(client, "info")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]i:%d#[default] ", color, ctx.InfoCount))
		} else {
			output.WriteString(fmt.Sprintf("i:%d ", ctx.InfoCount))
		}
	}
	if ctx.WarningCount > 0 {
		color := getLevelColorWithClient(client, "warning")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]w:%d#[default] ", color, ctx.WarningCount))
		} else {
			output.WriteString(fmt.Sprintf("w:%d ", ctx.WarningCount))
		}
	}
	if ctx.ErrorCount > 0 {
		color := getLevelColorWithClient(client, "error")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]e:%d#[default] ", color, ctx.ErrorCount))
		} else {
			output.WriteString(fmt.Sprintf("e:%d ", ctx.ErrorCount))
		}
	}
	if ctx.CriticalCount > 0 {
		color := getLevelColorWithClient(client, "critical")
		if color != "" {
			output.WriteString(fmt.Sprintf("#[fg=%s]c:%d#[default] ", color, ctx.CriticalCount))
		} else {
			output.WriteString(fmt.Sprintf("c:%d ", ctx.CriticalCount))
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
