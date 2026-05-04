// Package model provides interface contracts for TUI components.
// These interfaces define the contracts between different parts of the TUI system.
package model

import (
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
)

// NotificationService defines the interface for notification business logic operations.
// It handles filtering, searching, and managing notification data.
type NotificationService interface {
	// SetNotifications updates the underlying notification dataset.
	SetNotifications(notifications []domain.Notification)

	// SetShowStale controls whether notifications for stale tmux targets remain visible.
	SetShowStale(show bool)

	// GetNotifications returns all notifications currently tracked by the service.
	GetNotifications() []domain.Notification

	// GetFilteredNotifications returns the latest filtered notification view.
	GetFilteredNotifications() []domain.Notification

	// ApplyFiltersAndSearch applies tab scope, then filters/search/sorting and stores filtered results.
	ApplyFiltersAndSearch(tab settings.Tab, query, state, level, sessionID, windowID, paneID, readFilter, sortBy, sortOrder string)

	// FilterNotifications filters notifications based on a search query.
	// Returns a list of matching notifications.
	FilterNotifications(notifications []domain.Notification, query string) []domain.Notification

	// FilterByState filters notifications by state (active/dismissed).
	FilterByState(notifications []domain.Notification, state string) []domain.Notification

	// FilterByLevel filters notifications by level (info/warning/error).
	FilterByLevel(notifications []domain.Notification, level string) []domain.Notification

	// FilterBySession filters notifications by session ID.
	FilterBySession(notifications []domain.Notification, sessionID string) []domain.Notification

	// FilterByWindow filters notifications by window ID.
	FilterByWindow(notifications []domain.Notification, windowID string) []domain.Notification

	// FilterByPane filters notifications by pane ID.
	FilterByPane(notifications []domain.Notification, paneID string) []domain.Notification

	// FilterByReadStatus filters notifications by read/unread status.
	FilterByReadStatus(notifications []domain.Notification, readFilter string) []domain.Notification

	// SortNotifications sorts notifications by the specified field and order.
	SortNotifications(notifications []domain.Notification, sortBy, sortOrder string) []domain.Notification

	// GetUnreadCount returns the number of unread notifications.
	GetUnreadCount(notifications []domain.Notification) int

	// GetReadCount returns the number of read notifications.
	GetReadCount(notifications []domain.Notification) int

	// GetCountsByLevel returns a map of notification counts by level.
	GetCountsByLevel(notifications []domain.Notification) map[string]int

	// Search performs a full-text search on notifications.
	Search(notifications []domain.Notification, query string, caseSensitive bool) []domain.Notification
}
