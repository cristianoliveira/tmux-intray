package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/settings"
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
	lines, err := c.storage.ListNotifications(stateFilter, "", "", "", "", "", "", "")
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
		// Normalize fields to ensure proper padding
		normalized, err := storage.NormalizeFields(fields)
		if err != nil {
			// Skip malformed lines
			continue
		}
		fields = normalized
		// Bounds check for fieldMessage
		if len(fields) <= storage.FieldMessage {
			continue
		}
		message := fields[storage.FieldMessage]
		// Unescape message
		message = storage.UnescapeMessage(message)
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
	}

	// Add notification with empty timestamp (auto-generated)
	id, err := c.storage.AddNotification(item, "", session, window, pane, paneCreated, level)
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

// CleanupOldNotifications cleans up old dismissed notifications.
func CleanupOldNotifications(days int, dryRun bool) error {
	return defaultCore.CleanupOldNotifications(days, dryRun)
}

// CleanupOldNotifications cleans up old dismissed notifications using this Core instance.
func (c *Core) CleanupOldNotifications(days int, dryRun bool) error {
	return c.storage.CleanupOldNotifications(days, dryRun)
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

// ListNotifications lists notifications with filters.
func (c *Core) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	return c.storage.ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter)
}

// ListNotifications lists notifications with filters using the default core instance.
func ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	return defaultCore.ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter)
}

// DismissNotification dismisses a single notification by ID.
func (c *Core) DismissNotification(id string) error {
	return c.storage.DismissNotification(id)
}

// DismissNotification dismisses a single notification by ID using the default core instance.
func DismissNotification(id string) error {
	return defaultCore.DismissNotification(id)
}

// DismissAll dismisses all active notifications.
func (c *Core) DismissAll() error {
	return c.storage.DismissAll()
}

// DismissAll dismisses all active notifications using the default core instance.
func DismissAll() error {
	return defaultCore.DismissAll()
}

// ResetSettings resets settings to defaults.
func (c *Core) ResetSettings() (*settings.Settings, error) {
	return settings.Reset()
}

// ResetSettings resets settings to defaults using the default core instance.
func ResetSettings() (*settings.Settings, error) {
	return defaultCore.ResetSettings()
}

// LoadSettings loads current settings.
func (c *Core) LoadSettings() (*settings.Settings, error) {
	return settings.Load()
}

// LoadSettings loads current settings using the default core instance.
func LoadSettings() (*settings.Settings, error) {
	return defaultCore.LoadSettings()
}
