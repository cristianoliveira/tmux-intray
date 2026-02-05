// Package tmuxintray provides tmux-intray library initialization and orchestration.
package tmuxintray

import (
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

// Field indices matching storage package
const (
	fieldID          = 0
	fieldTimestamp   = 1
	fieldState       = 2
	fieldSession     = 3
	fieldWindow      = 4
	fieldPane        = 5
	fieldMessage     = 6
	fieldPaneCreated = 7
	fieldLevel       = 8
	numFields        = 9
)

// Init initializes all internal packages in the correct order.
// It loads configuration, sets up colors debugging, initializes storage,
// and starts the hooks subsystem.
// Returns an error if any initialization step fails.
func Init() error {
	// Load configuration first
	config.Load()

	// Set debug mode based on configuration
	debug := config.GetBool("debug", false)
	colors.SetDebug(debug)

	// Initialize storage
	storage.Init()

	// Initialize hooks subsystem
	hooks.Init()

	// TODO: verify initialization
	return nil
}

// Shutdown gracefully shuts down the library, cleaning up resources.
// It should be called before program exit.
func Shutdown() {
	hooks.Shutdown()
}

// AddNotification adds a new notification to the tray.
// It uses automatic tmux context detection unless noAuto is true.
// Returns the notification ID or empty string on error.
func AddNotification(message, session, window, pane, paneCreated string, noAuto bool, level string) string {
	return core.AddTrayItem(message, session, window, pane, paneCreated, noAuto, level)
}

// ListNotifications returns a list of notifications matching the filters.
// Filters that are empty strings are ignored.
// Returns TSV lines (same format as storage.ListNotifications).
func ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) string {
	return storage.ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
}

// DismissNotification dismisses a notification by ID.
func DismissNotification(id string) error {
	return storage.DismissNotification(id)
}

// DismissAllNotifications dismisses all active notifications.
func DismissAllNotifications() error {
	return storage.DismissAll()
}

// CleanupOldNotifications removes dismissed notifications older than the given days.
// If dryRun is true, only logs what would be removed.
func CleanupOldNotifications(days int, dryRun bool) {
	storage.CleanupOldNotifications(days, dryRun)
}

// GetActiveCount returns the number of active notifications.
func GetActiveCount() int {
	return storage.GetActiveCount()
}

// GetVisibility returns the current visibility state as "0" (hidden) or "1" (visible).
func GetVisibility() string {
	return core.GetVisibility()
}

// SetVisibility sets the tray visibility.
func SetVisibility(visible bool) error {
	return core.SetVisibility(visible)
}

// Notification represents a single notification in the tray.
type Notification struct {
	ID          string
	Timestamp   string
	State       string
	Session     string
	Window      string
	Pane        string
	Message     string
	PaneCreated string
	Level       string
}

// unescapeMessage reverses the escaping applied by storage package.
func unescapeMessage(msg string) string {
	// Unescape newlines first
	msg = strings.ReplaceAll(msg, "\\n", "\n")
	// Unescape tabs
	msg = strings.ReplaceAll(msg, "\\t", "\t")
	// Unescape backslashes
	msg = strings.ReplaceAll(msg, "\\\\", "\\")
	return msg
}

// ParseNotification parses a TSV line into a Notification struct.
func ParseNotification(tsvLine string) (Notification, error) {
	fields := strings.Split(tsvLine, "\t")
	if len(fields) < numFields {
		// Pad missing fields
		for len(fields) < numFields {
			fields = append(fields, "")
		}
	}
	return Notification{
		ID:          fields[fieldID],
		Timestamp:   fields[fieldTimestamp],
		State:       fields[fieldState],
		Session:     fields[fieldSession],
		Window:      fields[fieldWindow],
		Pane:        fields[fieldPane],
		Message:     unescapeMessage(fields[fieldMessage]),
		PaneCreated: fields[fieldPaneCreated],
		Level:       fields[fieldLevel],
	}, nil
}
