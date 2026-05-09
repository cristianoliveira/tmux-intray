// Package domain provides the domain layer for notifications.
// It contains business logic, value objects, and domain services.
package domain

import (
	"errors"
)

var (
	// ErrNotificationNotFound is returned when a notification is not found.
	ErrNotificationNotFound = errors.New("notification not found")

	// ErrInvalidNotificationID is returned when the notification ID is invalid.
	ErrInvalidNotificationID = errors.New("invalid notification ID")

	// ErrStorageFailed is returned when a storage operation fails.
	ErrStorageFailed = errors.New("storage operation failed")
)

// NotificationRepository defines the interface for notification persistence.
// This is the repository interface that storage implementations must follow.
type NotificationRepository interface {
	// Add adds a new notification and returns its ID.
	Add(message, timestamp, session, window, pane, paneCreated, level string) (int, error)

	// List retrieves notifications matching the given filters.
	// Filters: state, level, session, window, pane, olderThan, newerThan, readFilter.
	List(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) ([]Notification, error)

	// GetByID retrieves a notification by its ID.
	GetByID(id int) (*Notification, error)

	// Dismiss marks a notification as dismissed.
	Dismiss(id int) error

	// DismissAll marks all notifications as dismissed.
	DismissAll() error

	// DismissByFilter marks notifications matching the provided filters as dismissed.
	DismissByFilter(session, window, pane string) error

	// MarkRead marks a notification as read.
	MarkRead(id int) error

	// MarkUnread marks a notification as unread.
	MarkUnread(id int) error

	// CleanupOld removes notifications older than the specified days threshold.
	CleanupOld(daysThreshold int, dryRun bool) error

	// GetActiveCount returns the count of active notifications.
	GetActiveCount() int
}

// NotificationService provides business logic for notifications.
type NotificationService struct {
	repo NotificationRepository
}

// NewNotificationService creates a new notification service.
func NewNotificationService(repo NotificationRepository) *NotificationService {
	return &NotificationService{
		repo: repo,
	}
}

// Add adds a new notification with the given message and metadata.
// Returns the ID of the created notification.
func (s *NotificationService) Add(message, timestamp, session, window, pane, paneCreated, level string) (int, error) {
	return s.repo.Add(message, timestamp, session, window, pane, paneCreated, level)
}

// List retrieves notifications matching the given filters.
func (s *NotificationService) List(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) ([]Notification, error) {
	return s.repo.List(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter)
}

// GetByID retrieves a notification by its ID.
func (s *NotificationService) GetByID(id int) (*Notification, error) {
	return s.repo.GetByID(id)
}

// Dismiss marks a notification as dismissed by ID.
func (s *NotificationService) Dismiss(id int) error {
	return s.repo.Dismiss(id)
}

// DismissAll marks all notifications as dismissed.
func (s *NotificationService) DismissAll() error {
	return s.repo.DismissAll()
}

// DismissByFilter marks notifications matching the provided filters as dismissed.
func (s *NotificationService) DismissByFilter(session, window, pane string) error {
	return s.repo.DismissByFilter(session, window, pane)
}

// MarkRead marks a notification as read by ID.
func (s *NotificationService) MarkRead(id int) error {
	return s.repo.MarkRead(id)
}

// MarkUnread marks a notification as unread by ID.
func (s *NotificationService) MarkUnread(id int) error {
	return s.repo.MarkUnread(id)
}

// CleanupOld removes notifications older than the specified days threshold.
func (s *NotificationService) CleanupOld(daysThreshold int, dryRun bool) error {
	return s.repo.CleanupOld(daysThreshold, dryRun)
}

// GetActiveCount returns the count of active notifications.
func (s *NotificationService) GetActiveCount() int {
	return s.repo.GetActiveCount()
}

// QueryParams holds all parameters for querying notifications from storage.
type QueryParams struct {
	State       string
	Level       string
	Session     string
	Window      string
	Pane        string
	OlderThan   string // RFC3339 cutoff
	NewerThan   string // RFC3339 cutoff
	ReadFilter  string
	Search      string
	SortBy      string
	SortOrder   string
	UnreadFirst bool
}

// QueryResult holds the result of a notification query.
type QueryResult struct {
	Notifications []Notification
}

// Query fetches notifications from storage and applies filters, search, and sorting.
func (s *NotificationService) Query(params QueryParams) (*QueryResult, error) {
	notifs, err := s.repo.List(
		params.State,
		params.Level,
		params.Session,
		params.Window,
		params.Pane,
		params.OlderThan,
		params.NewerThan,
		params.ReadFilter,
	)
	if err != nil {
		return nil, err
	}

	if params.Search != "" {
		notifs = SearchNotifications(notifs, params.Search, true)
	}

	if params.SortBy != "" {
		field, err := ParseSortByField(params.SortBy)
		if err != nil {
			field = SortByTimestampField
		}
		order, err := ParseSortOrder(params.SortOrder)
		if err != nil {
			order = SortOrderDesc
		}
		opts := SortOptions{Field: field, Order: order}
		if params.UnreadFirst {
			notifs = SortWithUnreadFirst(notifs, opts)
		} else {
			notifs = SortNotifications(notifs, opts)
		}
	}

	return &QueryResult{Notifications: notifs}, nil
}
