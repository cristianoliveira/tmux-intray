// Package storage provides the storage interface for tmux-intray.
package storage

// Storage defines the interface for notification storage operations.
type Storage interface {
	AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error)
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
	GetNotificationByID(id string) (string, error)
	DismissNotification(id string) error
	DismissAll() error
	MarkNotificationRead(id string) error
	MarkNotificationUnread(id string) error
	MarkNotificationReadWithTimestamp(id, timestamp string) error
	MarkNotificationUnreadWithTimestamp(id, timestamp string) error
	CleanupOldNotifications(daysThreshold int, dryRun bool) error
	GetActiveCount() int
}
