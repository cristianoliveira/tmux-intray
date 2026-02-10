package storage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// DomainRepositoryAdapter implements domain.NotificationRepository by wrapping
// a Storage implementation and converting between types.
type DomainRepositoryAdapter struct {
	storage Storage
}

// NewDomainRepositoryAdapter creates a new domain repository adapter.
func NewDomainRepositoryAdapter(storage Storage) *DomainRepositoryAdapter {
	return &DomainRepositoryAdapter{
		storage: storage,
	}
}

// Add adds a new notification and returns its ID.
func (a *DomainRepositoryAdapter) Add(message, timestamp, session, window, pane, paneCreated, level string) (int, error) {
	idStr, err := a.storage.AddNotification(message, timestamp, session, window, pane, paneCreated, level)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", domain.ErrStorageFailed, err)
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("invalid ID format: %w", err)
	}

	return id, nil
}

// List retrieves notifications matching the given filters.
func (a *DomainRepositoryAdapter) List(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) ([]domain.Notification, error) {
	tsvData, err := a.storage.ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStorageFailed, err)
	}

	if tsvData == "" {
		return []domain.Notification{}, nil
	}

	var oldNotifs []notification.Notification
	lines := strings.Split(tsvData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		oldNotif, err := notification.ParseNotification(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse notification line: %w", err)
		}
		oldNotifs = append(oldNotifs, oldNotif)
	}

	domainNotifs, err := notification.ToDomainSlice(oldNotifs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to domain notifications: %w", err)
	}

	result := make([]domain.Notification, 0, len(domainNotifs))
	for _, n := range domainNotifs {
		if n != nil {
			result = append(result, *n)
		}
	}

	return result, nil
}

// GetByID retrieves a notification by its ID.
func (a *DomainRepositoryAdapter) GetByID(id int) (*domain.Notification, error) {
	idStr := strconv.Itoa(id)
	tsvData, err := a.storage.GetNotificationByID(idStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStorageFailed, err)
	}

	if tsvData == "" {
		return nil, domain.ErrNotificationNotFound
	}

	oldNotif, err := notification.ParseNotification(tsvData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse notification: %w", err)
	}

	domainNotif, err := notification.ToDomain(oldNotif)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to domain notification: %w", err)
	}

	return domainNotif, nil
}

// Dismiss marks a notification as dismissed.
func (a *DomainRepositoryAdapter) Dismiss(id int) error {
	idStr := strconv.Itoa(id)
	err := a.storage.DismissNotification(idStr)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrStorageFailed, err)
	}
	return nil
}

// DismissAll marks all notifications as dismissed.
func (a *DomainRepositoryAdapter) DismissAll() error {
	err := a.storage.DismissAll()
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrStorageFailed, err)
	}
	return nil
}

// MarkRead marks a notification as read.
func (a *DomainRepositoryAdapter) MarkRead(id int) error {
	idStr := strconv.Itoa(id)
	err := a.storage.MarkNotificationRead(idStr)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrStorageFailed, err)
	}
	return nil
}

// MarkUnread marks a notification as unread.
func (a *DomainRepositoryAdapter) MarkUnread(id int) error {
	idStr := strconv.Itoa(id)
	err := a.storage.MarkNotificationUnread(idStr)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrStorageFailed, err)
	}
	return nil
}

// CleanupOld removes notifications older than the specified days threshold.
func (a *DomainRepositoryAdapter) CleanupOld(daysThreshold int, dryRun bool) error {
	err := a.storage.CleanupOldNotifications(daysThreshold, dryRun)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrStorageFailed, err)
	}
	return nil
}

// GetActiveCount returns the count of active notifications.
func (a *DomainRepositoryAdapter) GetActiveCount() int {
	return a.storage.GetActiveCount()
}
