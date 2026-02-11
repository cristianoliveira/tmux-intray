// Package storage provides file-based TSV storage with locking.
package storage

// FileStorage implements the Storage interface using file-based TSV storage.
type FileStorage struct{}

// NewFileStorage creates a new FileStorage instance.
func NewFileStorage() (*FileStorage, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	return &FileStorage{}, nil
}

// AddNotification adds a notification and returns its ID.
func (fs *FileStorage) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	return AddNotification(message, timestamp, session, window, pane, paneCreated, level)
}

// ListNotifications returns TSV lines for notifications matching the specified filters.
func (fs *FileStorage) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	return ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter)
}

// GetNotificationByID retrieves a single notification by its ID.
func (fs *FileStorage) GetNotificationByID(id string) (string, error) {
	return GetNotificationByID(id)
}

// DismissNotification dismisses a notification by ID.
func (fs *FileStorage) DismissNotification(id string) error {
	return DismissNotification(id)
}

// DismissAll dismisses all active notifications.
func (fs *FileStorage) DismissAll() error {
	return DismissAll()
}

// MarkNotificationRead marks a notification as read by setting read_timestamp.
func (fs *FileStorage) MarkNotificationRead(id string) error {
	return MarkNotificationRead(id)
}

// MarkNotificationUnread marks a notification as unread by clearing read_timestamp.
func (fs *FileStorage) MarkNotificationUnread(id string) error {
	return MarkNotificationUnread(id)
}

// MarkNotificationReadWithTimestamp marks a notification as read by setting read_timestamp to the provided timestamp.
func (fs *FileStorage) MarkNotificationReadWithTimestamp(id, timestamp string) error {
	return MarkNotificationReadWithTimestamp(id, timestamp)
}

// MarkNotificationUnreadWithTimestamp marks a notification as unread by setting read_timestamp to the provided value (typically empty string).
func (fs *FileStorage) MarkNotificationUnreadWithTimestamp(id, timestamp string) error {
	return MarkNotificationUnreadWithTimestamp(id, timestamp)
}

// CleanupOldNotifications cleans up notifications older than the threshold.
func (fs *FileStorage) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	return CleanupOldNotifications(daysThreshold, dryRun)
}

// GetActiveCount returns the active notification count.
func (fs *FileStorage) GetActiveCount() int {
	return GetActiveCount()
}
