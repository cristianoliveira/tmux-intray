package main

import (
	"fmt"
	"time"
)

// resolveSessionName resolves a session ID to its display name using tmux.
// Returns the resolved name, or the original ID if resolution fails.
func resolveSessionName(sessionID string, sessionNames map[string]string) string {
	if sessionID == "" {
		return ""
	}
	if name, ok := sessionNames[sessionID]; ok && name != "" {
		return name
	}
	return sessionID
}

// formatAge formats a timestamp as relative age (e.g., "2h").
func formatAge(timestamp string) string {
	if timestamp == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}

	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd", int(duration.Hours()/24))
}
