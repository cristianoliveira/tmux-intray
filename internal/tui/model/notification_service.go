// Package model provides interface contracts for TUI components.
// These interfaces define the contracts between different parts of the TUI system.
package model

import (
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// NotificationService defines the interface for notification business logic operations.
// It handles filtering, searching, and managing notification data.
type NotificationService interface {
	// FilterNotifications filters notifications based on a search query.
	// Returns a list of matching notifications.
	FilterNotifications(notifications []notification.Notification, query string) []notification.Notification

	// FilterByState filters notifications by state (active/dismissed).
	FilterByState(notifications []notification.Notification, state string) []notification.Notification

	// FilterByLevel filters notifications by level (info/warning/error).
	FilterByLevel(notifications []notification.Notification, level string) []notification.Notification

	// FilterBySession filters notifications by session ID.
	FilterBySession(notifications []notification.Notification, sessionID string) []notification.Notification

	// FilterByWindow filters notifications by window ID.
	FilterByWindow(notifications []notification.Notification, windowID string) []notification.Notification

	// FilterByPane filters notifications by pane ID.
	FilterByPane(notifications []notification.Notification, paneID string) []notification.Notification

	// SortNotifications sorts notifications by the specified field and order.
	SortNotifications(notifications []notification.Notification, sortBy, sortOrder string) []notification.Notification

	// GetUnreadCount returns the number of unread notifications.
	GetUnreadCount(notifications []notification.Notification) int

	// GetReadCount returns the number of read notifications.
	GetReadCount(notifications []notification.Notification) int

	// GetCountsByLevel returns a map of notification counts by level.
	GetCountsByLevel(notifications []notification.Notification) map[string]int

	// Search performs a full-text search on notifications.
	Search(notifications []notification.Notification, query string, caseSensitive bool) []notification.Notification
}
