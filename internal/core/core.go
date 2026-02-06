// Package core provides core tmux interaction and tray management.
package core

import (
	"errors"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

// Field indices matching storage package
const (
	fieldID          = 0
	fieldTimestamp   = 1
	fieldState       = 2
	fieldSession     = 3
	fieldWindow      = 4
	fieldPane        = 5
	fieldMessage     = 6
	fieldPaneCreated = 7
	fieldLevel       = 8
)

// GetTrayItems returns tray items for a given state filter.
// Returns newline-separated messages (unescaped).
func GetTrayItems(stateFilter string) string {
	// Use storage.ListNotifications with only state filter
	lines := storage.ListNotifications(stateFilter, "", "", "", "", "", "")
	if lines == "" {
		return ""
	}
	var messages []string
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) <= fieldMessage {
			continue
		}
		message := fields[fieldMessage]
		// Unescape message
		message = unescapeMessage(message)
		messages = append(messages, message)
	}
	return strings.Join(messages, "\n")
}

// AddTrayItem adds a tray item.
// If session, window, pane are empty and noAuto is false, current tmux context is used.
// Returns the notification ID.
func AddTrayItem(item, session, window, pane, paneCreated string, noAuto bool, level string) string {
	// Treat empty/whitespace context same as not provided for resilience
	// This handles cases where plugin passes empty strings as flags
	session = strings.TrimSpace(session)
	window = strings.TrimSpace(window)
	pane = strings.TrimSpace(pane)

	// If auto context allowed and session/window/pane empty, get current tmux context
	if !noAuto && session == "" && window == "" && pane == "" {
		ctx := GetCurrentTmuxContext()
		if ctx.SessionID != "" {
			session = ctx.SessionID
			window = ctx.WindowID
			pane = ctx.PaneID
			if paneCreated == "" {
				paneCreated = ctx.PaneCreated
			}
		}
	}
	// Add notification with empty timestamp (auto-generated)
	id := storage.AddNotification(item, "", session, window, pane, paneCreated, level)
	if id == "" {
		colors.Error("Failed to add tray item")
	}
	return id
}

// ClearTrayItems dismisses all active tray items.
func ClearTrayItems() error {
	return storage.DismissAll()
}

// GetVisibility returns the visibility state as "0" or "1".
func GetVisibility() string {
	return GetTmuxVisibility()
}

// SetVisibility sets the visibility state.
func SetVisibility(visible bool) error {
	value := "0"
	if visible {
		value = "1"
	}
	success := SetTmuxVisibility(value)
	if !success {
		return ErrTmuxOperationFailed
	}
	return nil
}

// Helper functions copied from storage package (since they're not exported)

func escapeMessage(msg string) string {
	// Escape backslashes first
	msg = strings.ReplaceAll(msg, "\\", "\\\\")
	// Escape tabs
	msg = strings.ReplaceAll(msg, "\t", "\\t")
	// Escape newlines
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	return msg
}

func unescapeMessage(msg string) string {
	// Unescape newlines first
	msg = strings.ReplaceAll(msg, "\\n", "\n")
	// Unescape tabs
	msg = strings.ReplaceAll(msg, "\\t", "\t")
	// Unescape backslashes
	msg = strings.ReplaceAll(msg, "\\\\", "\\")
	return msg
}

// ErrTmuxOperationFailed is returned when tmux operation fails.
var ErrTmuxOperationFailed = errors.New("tmux operation failed")
