// Package format provides output formatting functionality for CLI commands.
// It includes formatters for different output styles and notification display.
package format

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

// ParseCountsByLevel parses TSV lines and returns counts per notification level.
// Lines are expected to be in the storage TSV format with field indices defined
// in storage package. Unknown levels are counted as info (default).
func ParseCountsByLevel(lines string) (info, warning, error, critical int, err error) {
	if lines == "" {
		return 0, 0, 0, 0, nil
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
			// Default to info
			info++
		}
	}
	return info, warning, error, critical, nil
}

// ParsePaneCounts parses TSV lines and returns a map of pane keys to counts.
// Pane key format: session:window:pane
func ParsePaneCounts(lines string) map[string]int {
	counts := make(map[string]int)
	if lines == "" {
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

// FormatSummary writes a summary of notification counts to the writer.
// Format: "Active notifications: X\n  info: A, warning: B, error: C, critical: D\n"
// If active is 0, writes "No active notifications\n"
func FormatSummary(w io.Writer, active int, info, warning, error, critical int) error {
	if active == 0 {
		_, err := fmt.Fprintf(w, "No active notifications\n")
		return err
	}
	_, err := fmt.Fprintf(w, "Active notifications: %d\n", active)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  info: %d, warning: %d, error: %d, critical: %d\n",
		info, warning, error, critical)
	return err
}

// FormatLevels writes level counts in key:value format, one per line.
func FormatLevels(w io.Writer, info, warning, error, critical int) error {
	_, err := fmt.Fprintf(w, "info:%d\nwarning:%d\nerror:%d\ncritical:%d\n",
		info, warning, error, critical)
	return err
}

// FormatPanes writes pane counts in paneKey:count format, one per line.
// Pane keys are sorted alphabetically for deterministic output.
func FormatPanes(w io.Writer, paneCounts map[string]int) error {
	// Sort keys for deterministic output
	keys := make([]string, 0, len(paneCounts))
	for k := range paneCounts {
		keys = append(keys, k)
	}
	// simple alphabetical sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, key := range keys {
		_, err := fmt.Fprintf(w, "%s:%d\n", key, paneCounts[key])
		if err != nil {
			return err
		}
	}
	return nil
}

// StatusData holds aggregated status information for JSON output.
type StatusData struct {
	Active   int            `json:"active"`
	Info     int            `json:"info"`
	Warning  int            `json:"warning"`
	Error    int            `json:"error"`
	Critical int            `json:"critical"`
	Panes    map[string]int `json:"panes"`
}

// FormatJSON writes status data as JSON to the writer.
func FormatJSON(w io.Writer, active int, info, warning, error, critical int, paneCounts map[string]int) error {
	data := StatusData{
		Active:   active,
		Info:     info,
		Warning:  warning,
		Error:    error,
		Critical: critical,
		Panes:    paneCounts,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
