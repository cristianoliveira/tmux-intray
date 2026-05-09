// Package storage provides the storage interface for tmux-intray.
package storage

import "github.com/cristianoliveira/tmux-intray/internal/domain"

// Storage defines the interface for notification storage operations.
type Storage interface {
	AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error)
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
	ListNotificationValues(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) ([]domain.Notification, error)
	GetNotificationByID(id string) (string, error)
	DismissNotification(id string) error
	DismissAll() error
	DismissByFilter(session, window, pane string) error
	MarkNotificationRead(id string) error
	MarkNotificationUnread(id string) error
	CleanupOldNotifications(daysThreshold int, dryRun bool) error
	GetActiveCount() int
}
