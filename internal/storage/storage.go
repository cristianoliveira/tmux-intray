// Package storage provides the storage interface for tmux-intray.
package storage

import (
	"fmt"
	"strings"
	"sync"

	"github.com/cristianoliveira/tmux-intray/internal/config"
)

var (
	defaultStorage Storage
	defaultOnce    sync.Once
	defaultErr     error
)

// getDefaultStorage returns the default storage instance, initializing it if necessary.
func getDefaultStorage() (Storage, error) {
	defaultOnce.Do(func() {
		config.Load()
		defaultStorage, defaultErr = NewFromConfig()
	})
	return defaultStorage, defaultErr
}

// AddNotification adds a notification using the default storage backend.
func AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	store, err := getDefaultStorage()
	if err != nil {
		return "", fmt.Errorf("failed to get storage: %w", err)
	}
	return store.AddNotification(message, timestamp, session, window, pane, paneCreated, level)
}

// ListNotifications returns TSV lines for notifications using the default storage backend.
func ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	store, err := getDefaultStorage()
	if err != nil {
		return "", fmt.Errorf("failed to get storage: %w", err)
	}
	return store.ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter)
}

// GetNotificationByID retrieves a notification by ID using the default storage backend.
func GetNotificationByID(id string) (string, error) {
	store, err := getDefaultStorage()
	if err != nil {
		return "", fmt.Errorf("failed to get storage: %w", err)
	}
	return store.GetNotificationByID(id)
}

// DismissNotification dismisses a notification using the default storage backend.
func DismissNotification(id string) error {
	store, err := getDefaultStorage()
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}
	return store.DismissNotification(id)
}

// DismissAll dismisses all active notifications using the default storage backend.
func DismissAll() error {
	store, err := getDefaultStorage()
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}
	return store.DismissAll()
}

// MarkNotificationRead marks a notification as read using the default storage backend.
func MarkNotificationRead(id string) error {
	store, err := getDefaultStorage()
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}
	return store.MarkNotificationRead(id)
}

// MarkNotificationUnread marks a notification as unread using the default storage backend.
func MarkNotificationUnread(id string) error {
	store, err := getDefaultStorage()
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}
	return store.MarkNotificationUnread(id)
}

// MarkNotificationReadWithTimestamp marks a notification as read with the provided timestamp.
func MarkNotificationReadWithTimestamp(id, timestamp string) error {
	store, err := getDefaultStorage()
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}
	return store.MarkNotificationReadWithTimestamp(id, timestamp)
}

// MarkNotificationUnreadWithTimestamp marks a notification as unread with the provided timestamp.
func MarkNotificationUnreadWithTimestamp(id, timestamp string) error {
	store, err := getDefaultStorage()
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}
	return store.MarkNotificationUnreadWithTimestamp(id, timestamp)
}

// CleanupOldNotifications cleans up old notifications using the default storage backend.
func CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	store, err := getDefaultStorage()
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}
	return store.CleanupOldNotifications(daysThreshold, dryRun)
}

// GetActiveCount returns the count of active notifications using the default storage backend.
func GetActiveCount() int {
	store, err := getDefaultStorage()
	if err != nil {
		return 0
	}
	return store.GetActiveCount()
}

// NormalizeFields ensures a TSV line has the correct number of fields.
// Pads with empty strings if fewer than expected, returns error if below minimum.
func NormalizeFields(fields []string) ([]string, error) {
	if len(fields) < MinFields {
		return nil, fmt.Errorf("expected at least %d fields, got %d", MinFields, len(fields))
	}
	if len(fields) < NumFields {
		for len(fields) < NumFields {
			fields = append(fields, "")
		}
	}
	return fields, nil
}

// EscapeMessage escapes special characters in a message for TSV storage.
// Escapes backslashes, tabs, and newlines to preserve message formatting.
func EscapeMessage(msg string) string {
	// Escape backslashes first
	msg = strings.ReplaceAll(msg, "\\", "\\\\")
	// Escape tabs
	msg = strings.ReplaceAll(msg, "\t", "\\t")
	// Escape newlines
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	return msg
}

// UnescapeMessage unescapes special characters from a TSV-stored message.
// Restores newlines, tabs, and backslashes to their original values.
func UnescapeMessage(msg string) string {
	// Unescape newlines first
	msg = strings.ReplaceAll(msg, "\\n", "\n")
	// Unescape tabs
	msg = strings.ReplaceAll(msg, "\\t", "\t")
	// Unescape backslashes
	msg = strings.ReplaceAll(msg, "\\\\", "\\")
	return msg
}
