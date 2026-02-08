// Package storage provides file-based TSV storage with locking.
package storage

// Store defines the interface for storage operations.
type Store interface {
	Init() error
	AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error)
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error)
	GetNotificationByID(id string) (string, error)
	DismissNotification(id string) error
	DismissAll() error
	CleanupOldNotifications(daysThreshold int, dryRun bool) error
	GetActiveCount() int
}

// DefaultStore implements Store using the package-level functions.
type DefaultStore struct{}

// NewDefaultStore creates a new default Store implementation.
func NewDefaultStore() Store {
	return &DefaultStore{}
}

// Init initializes storage directories and files.
func (s *DefaultStore) Init() error {
	return Init()
}

// AddNotification adds a notification and returns its ID.
func (s *DefaultStore) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	return AddNotification(message, timestamp, session, window, pane, paneCreated, level)
}

// ListNotifications returns TSV lines for notifications matching the specified filters.
func (s *DefaultStore) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error) {
	return ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
}

// GetNotificationByID retrieves a notification by its ID.
func (s *DefaultStore) GetNotificationByID(id string) (string, error) {
	return GetNotificationByID(id)
}

// DismissNotification dismisses a notification by ID.
func (s *DefaultStore) DismissNotification(id string) error {
	return DismissNotification(id)
}

// DismissAll dismisses all active notifications.
func (s *DefaultStore) DismissAll() error {
	return DismissAll()
}

// CleanupOldNotifications cleans up notifications older than the threshold.
func (s *DefaultStore) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	return CleanupOldNotifications(daysThreshold, dryRun)
}

// GetActiveCount returns the active notification count.
func (s *DefaultStore) GetActiveCount() int {
	return GetActiveCount()
}
