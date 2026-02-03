// Package storage provides file-based TSV storage with locking.
package storage

// Init initializes storage.
func Init() {
}

// AddNotification adds a notification and returns its ID.
func AddNotification(message, timestamp, session, window, pane, paneCreated, level string) string {
	_ = message
	_ = timestamp
	_ = session
	_ = window
	_ = pane
	_ = paneCreated
	_ = level
	return ""
}

// ListNotifications returns TSV lines for notifications.
func ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) string {
	_ = stateFilter
	_ = levelFilter
	_ = sessionFilter
	_ = windowFilter
	_ = paneFilter
	_ = olderThanCutoff
	_ = newerThanCutoff
	return ""
}

// DismissNotification dismisses a notification by ID.
func DismissNotification(id string) {
	_ = id
}

// DismissAll dismisses all active notifications.
func DismissAll() {
}

// CleanupOldNotifications cleans up notifications older than the threshold.
func CleanupOldNotifications(daysThreshold int, dryRun bool) {
	_ = daysThreshold
	_ = dryRun
}

// GetActiveCount returns the active notification count.
func GetActiveCount() int {
	return 0
}
