// Package model provides interface contracts for TUI components.
// These interfaces define the contracts between different parts of the TUI system.
package model

import (
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// NotificationRepository defines the interface for notification data access operations.
// It provides CRUD operations on notifications specifically for the TUI context.
type NotificationRepository interface {
	// LoadNotifications loads all active notifications.
	// Returns a slice of notifications or an error if loading fails.
	LoadNotifications() ([]notification.Notification, error)

	// LoadFilteredNotifications loads notifications with optional filters.
	// Supported filters: state, level, session, window, pane.
	// Returns a slice of matching notifications or an error.
	LoadFilteredNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter string) ([]notification.Notification, error)

	// DismissNotification marks a notification as dismissed by ID.
	// Returns an error if the operation fails.
	DismissNotification(id string) error

	// MarkAsRead marks a notification as read by ID.
	// Returns an error if the operation fails.
	MarkAsRead(id string) error

	// MarkAsUnread marks a notification as unread by ID.
	// Returns an error if the operation fails.
	MarkAsUnread(id string) error

	// GetByID retrieves a notification by its ID.
	// Returns the notification or an error if not found.
	GetByID(id string) (notification.Notification, error)

	// GetActiveCount returns the number of active (non-dismissed) notifications.
	GetActiveCount() int
}
