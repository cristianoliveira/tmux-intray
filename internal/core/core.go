// Package core provides core tmux interaction and tray management.
package core

import (
	"errors"
	"fmt"
	"strings"
)

// GetTrayItems returns tray items for a given state filter.
// Returns newline-separated messages (unescaped).
func (c *Core) GetTrayItems(stateFilter string) (string, error) {
	// Use storage.ListNotifications with only state filter
	lines, err := c.store.ListNotifications(stateFilter, "", "", "", "", "", "")
	if err != nil {
		return "", err
	}
	if lines == "" {
		return "", nil
	}
	var messages []string
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < storage.NumFields {
			for len(fields) < storage.NumFields {
				fields = append(fields, "")
			}
		}
		// Bounds check for fieldMessage
		if len(fields) <= storage.FieldMessage {
			continue
		}
		message := fields[storage.FieldMessage]
		// Unescape message
		message = unescapeMessage(message)
		messages = append(messages, message)
	}
	return strings.Join(messages, "\n"), nil
}

// GetTrayItems returns tray items for a given state filter using the default core.
func GetTrayItems(stateFilter string) (string, error) {
	return defaultCore.GetTrayItems(stateFilter)
}

// AddTrayItem adds a tray item.
// If session, window, pane are empty and noAuto is false, current tmux context is used.
// Returns the notification ID or an error if validation fails.
func (c *Core) AddTrayItem(item, session, window, pane, paneCreated string, noAuto bool, level string) (string, error) {
	// Treat empty/whitespace context same as not provided for resilience
	// This handles cases where plugin passes empty strings as flags
	session = strings.TrimSpace(session)
	window = strings.TrimSpace(window)
	pane = strings.TrimSpace(pane)

	// If auto context allowed and session/window/pane empty, get current tmux context
	if !noAuto && session == "" && window == "" && pane == "" {
		ctx := c.GetCurrentTmuxContext()
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
	id, err := c.store.AddNotification(item, "", session, window, pane, paneCreated, level)
	if err != nil {
		return "", fmt.Errorf("add tray item: failed to add notification: %w", err)
	}
	return id, nil
}

// AddTrayItem adds a tray item using the default client.
func AddTrayItem(item, session, window, pane, paneCreated string, noAuto bool, level string) (string, error) {
	return defaultCore.AddTrayItem(item, session, window, pane, paneCreated, noAuto, level)
}

// ClearTrayItems dismisses all active tray items.
func (c *Core) ClearTrayItems() error {
	return c.store.DismissAll()
}

// ClearTrayItems dismisses all active tray items using the default core.
func ClearTrayItems() error {
	return defaultCore.ClearTrayItems()
}

// MarkNotificationRead marks a notification as read.
func MarkNotificationRead(id string) error {
	return storage.MarkNotificationRead(id)
}

// MarkNotificationUnread marks a notification as unread.
func MarkNotificationUnread(id string) error {
	return storage.MarkNotificationUnread(id)
}

// GetVisibility returns the visibility state as "0" or "1".
func (c *Core) GetVisibility() string {
	return c.GetTmuxVisibility()
}

// GetVisibility returns the visibility state as "0" or "1" using the default client.
func GetVisibility() string {
	return defaultCore.GetVisibility()
}

// SetVisibility sets the visibility state.
func (c *Core) SetVisibility(visible bool) error {
	value := "0"
	if visible {
		value = "1"
	}
	_, err := c.SetTmuxVisibility(value)
	if err != nil {
		return fmt.Errorf("set visibility: %w", err)
	}
	return nil
}

// SetVisibility sets the visibility state using the default client.
func SetVisibility(visible bool) error {
	return defaultCore.SetVisibility(visible)
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
