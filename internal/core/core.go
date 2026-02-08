package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

// GetTrayItems returns tray items for a given state filter.
// Returns newline-separated messages (unescaped).
func GetTrayItems(stateFilter string) (string, error) {
	return defaultCore.GetTrayItems(stateFilter)
}

// GetTrayItems returns tray items for a given state filter using this Core instance.
// Returns newline-separated messages (unescaped).
func (c *Core) GetTrayItems(stateFilter string) (string, error) {
	// Use storage.ListNotifications with only state filter
	lines, err := c.storage.ListNotifications(stateFilter, "", "", "", "", "", "")
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
		// TODO: Duplicate of storage.normalizeFields logic. Consider exporting helper.
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

// AddTrayItem adds a tray item.
// If session, window, pane are empty and noAuto is false, current tmux context is used.
// Returns the notification ID or an error if validation fails.
func (c *Core) AddTrayItem(item, session, window, pane, paneCreated string, noAuto bool, level string) (string, error) {
	// Treat empty/whitespace context same as not provided for resilience
	item = strings.TrimSpace(item)
	if item == "" {
		return "", errors.New("add tray item: message cannot be empty")
	}

	// Get current tmux context if needed
	sessionName := ""
	if !noAuto && (session == "" || window == "" || pane == "") {
		ctx := c.GetCurrentTmuxContext()
		if session == "" {
			session = ctx.SessionID
		}
		if window == "" {
			window = ctx.WindowID
		}
		if pane == "" {
			pane = ctx.PaneID
		}
		if paneCreated == "" {
			paneCreated = ctx.PaneCreated
		}
		// Capture session name for searchability
		if c.client != nil && session != "" {
			if name, err := c.client.GetSessionName(session); err == nil && name != "" {
				sessionName = name
			}
		}
	}

	// Add notification with empty timestamp (auto-generated)
	id, err := c.storage.AddNotification(item, "", session, sessionName, window, pane, paneCreated, level)
	if err != nil {
		return "", fmt.Errorf("add tray item: failed to add notification: %w", err)
	}
	return id, nil
}

// AddTrayItem adds a tray item using the default core instance.
// Returns the notification ID or an error if validation fails.
func AddTrayItem(item, session, window, pane, paneCreated string, noAuto bool, level string) (string, error) {
	return defaultCore.AddTrayItem(item, session, window, pane, paneCreated, noAuto, level)
}

// ClearTrayItems dismisses all active tray items.
func ClearTrayItems() error {
	return defaultCore.ClearTrayItems()
}

// ClearTrayItems dismisses all active tray items using this Core instance.
func (c *Core) ClearTrayItems() error {
	return c.storage.DismissAll()
}

// MarkNotificationRead marks a notification as read.
func MarkNotificationRead(id string) error {
	return defaultCore.MarkNotificationRead(id)
}

// MarkNotificationRead marks a notification as read using this Core instance.
func (c *Core) MarkNotificationRead(id string) error {
	return c.storage.MarkNotificationRead(id)
}

// MarkNotificationUnread marks a notification as unread.
func MarkNotificationUnread(id string) error {
	return defaultCore.MarkNotificationUnread(id)
}

// MarkNotificationUnread marks a notification as unread using this Core instance.
func (c *Core) MarkNotificationUnread(id string) error {
	return c.storage.MarkNotificationUnread(id)
}

// GetVisibility returns the visibility state as "0" or "1".
func GetVisibility() (string, error) {
	return defaultCore.GetVisibility()
}

// GetVisibility returns the visibility state using this Core instance.
func (c *Core) GetVisibility() (string, error) {
	// Try tmux first
	if visible := c.GetTmuxVisibility(); visible != "" {
		return visible, nil
	}

	// Fall back to "0" (default)
	return "0", nil
}

// SetVisibility sets the visibility state.
// Returns true on success, false on failure.
func SetVisibility(visible bool) error {
	return defaultCore.SetVisibility(visible)
}

// SetVisibility sets the visibility state using this Core instance.
// Returns true on success, false on failure.
func (c *Core) SetVisibility(visible bool) error {
	// Set in tmux for other processes
	visibleStr := "0"
	if visible {
		visibleStr = "1"
	}
	_, err := c.SetTmuxVisibility(visibleStr)
	return err
}

// Helper functions copied from storage package (since they're not exported)
// TODO: Duplicate of storage.escapeMessage and storage.unescapeMessage. Consider exporting them from storage package.

func escapeMessage(msg string) string {
	// Escape backslashes first
	msg = strings.ReplaceAll(msg, "\\", "\\\\")
	// Escape newlines
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	// Escape tabs
	msg = strings.ReplaceAll(msg, "\t", "\\t")
	return msg
}

func unescapeMessage(msg string) string {
	// Unescape tabs first (to avoid unescaping \n in \t)
	msg = strings.ReplaceAll(msg, "\\t", "\t")
	// Unescape newlines
	msg = strings.ReplaceAll(msg, "\\n", "\n")
	// Unescape backslashes last
	msg = strings.ReplaceAll(msg, "\\\\", "\\")
	return msg
}
