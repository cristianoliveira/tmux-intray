// Package model provides interface contracts for TUI components.
package model

import "github.com/cristianoliveira/tmux-intray/internal/notification"

// InteractionController coordinates side-effectful TUI interactions.
// It encapsulates tmux/core interactions and notification persistence operations.
type InteractionController interface {
	LoadActiveNotifications() ([]notification.Notification, error)
	DismissNotification(id string) error
	DismissByFilter(session, window, pane string) error
	MarkNotificationRead(id string) error
	MarkNotificationUnread(id string) error
	EnsureTmuxRunning() bool
	JumpToPane(sessionID, windowID, paneID string) bool
	JumpToWindow(sessionID, windowID string) bool
}
