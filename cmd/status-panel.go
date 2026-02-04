/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
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

// statusPanelEnsureTmuxRunningFunc is the function used to ensure tmux is running. Can be changed for testing.
var statusPanelEnsureTmuxRunningFunc = func() bool {
	return core.EnsureTmuxRunning()
}

// statusPanelGetActiveCountFunc is the function used to get active notification count.
var statusPanelGetActiveCountFunc = func() int {
	return storage.GetActiveCount()
}

// statusPanelListNotificationsFunc is the function used to list notifications.
var statusPanelListNotificationsFunc = func(stateFilter string) string {
	return storage.ListNotifications(stateFilter, "", "", "", "", "", "")
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
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) <= fieldLevel {
			continue
		}
		level := fields[fieldLevel]
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
			// Default to info
			info++
		}
	}
	return info, warning, error, critical, nil
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

// formatCountOnly returns count-only format output.
func formatCountOnly(total int) string {
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("%d", total)
}

// Run executes the status-panel command with given options.
// Returns the formatted output string (may be empty) and any error.
func Run(opts StatusPanelOptions) (string, error) {
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

var (
	statusPanelFormat  string
	statusPanelEnabled string
)

// statusPanelCmd represents the status-panel command
var statusPanelCmd = &cobra.Command{
	Use:   "status-panel",
	Short: "Status bar indicator script (for tmux status-right)",
	Long: `Status bar indicator script (for tmux status-right).

USAGE:
    tmux-intray status-panel [OPTIONS]

OPTIONS:
    --format=<format>    Output format: compact, detailed, count-only (default: compact)
    --enabled=<0|1>      Enable/disable status indicator (default: 1)
    -h, --help           Show this help

DESCRIPTION:
    This script is designed to be used in tmux status-right configuration.
    Example: set -g status-right "#(tmux-intray status-panel) %H:%M"

    The script outputs a formatted string showing notification counts.
    When clicked, it can trigger the list command (via tmux bindings).`,
	Run: runStatusPanel,
}

func init() {
	rootCmd.AddCommand(statusPanelCmd)

	statusPanelCmd.Flags().StringVar(&statusPanelFormat, "format", "", "Output format: compact, detailed, count-only")
	statusPanelCmd.Flags().StringVar(&statusPanelEnabled, "enabled", "", "Enable/disable status indicator (0 or 1)")
}

func runStatusPanel(cmd *cobra.Command, args []string) {
	// Determine format
	format := statusPanelFormat
	if format == "" {
		// Get from config via environment variable (config package already loaded)
		// We'll rely on internal command to use config defaults.
		// Pass empty string to let internal command decide.
	}

	// Determine enabled
	enabled := true // default
	if statusPanelEnabled != "" {
		val := strings.ToLower(statusPanelEnabled)
		if val == "0" || val == "false" || val == "no" || val == "off" {
			enabled = false
		} else if val == "1" || val == "true" || val == "yes" || val == "on" {
			enabled = true
		} else {
			colors.Error("invalid value for --enabled, must be 0 or 1")
			return
		}
	}

	opts := StatusPanelOptions{
		Format:  format,
		Enabled: enabled,
	}
	output, err := Run(opts)
	if err != nil {
		colors.Error(err.Error())
		os.Exit(1)
	}
	if output != "" {
		fmt.Print(output)
	}
	// No output means empty string (tmux will show nothing).
}
