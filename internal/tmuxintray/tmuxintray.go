// Package tmuxintray provides tmux-intray library initialization and orchestration.
package tmuxintray

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
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
	if err := hooks.Init(); err != nil {
		return fmt.Errorf("hooks initialization failed: %w", err)
	}

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
// Returns the notification ID or an error if validation fails.
func AddNotification(message, session, window, pane, paneCreated string, noAuto bool, level string) (string, error) {
	return core.AddTrayItem(message, session, window, pane, paneCreated, noAuto, level)
}

// ListNotifications returns a list of notifications matching the filters.
// Filters that are empty strings are ignored.
// Returns TSV lines (same format as storage.ListNotifications) and an error if validation fails.
func ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error) {
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

// MarkNotificationRead marks a notification as read.
func MarkNotificationRead(id string) error {
	return storage.MarkNotificationRead(id)
}

// MarkNotificationUnread marks a notification as unread.
func MarkNotificationUnread(id string) error {
	return storage.MarkNotificationUnread(id)
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

var getVisibilityFunc = func() string {
	return core.GetVisibility()
}

// GetVisibility returns the current visibility state as "0" (hidden) or "1" (visible).
func GetVisibility() string {
	return getVisibilityFunc()
}

// SetVisibility sets the tray visibility.
func SetVisibility(visible bool) error {
	return core.SetVisibility(visible)
}

// Notification represents a single notification in the tray.
type Notification struct {
	ID            string
	Timestamp     string
	State         string
	Session       string
	Window        string
	Pane          string
	Message       string
	PaneCreated   string
	Level         string
	ReadTimestamp string
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
// Preconditions: tsvLine must be a valid TSV line with NumFields or NumFields-1 fields.
func ParseNotification(tsvLine string) (Notification, error) {
	fields := strings.Split(tsvLine, "\t")
	switch len(fields) {
	case storage.NumFields - 1:
		fields = append(fields, "")
	case storage.NumFields:
		// OK
	default:
		return Notification{}, fmt.Errorf("invalid notification field count: %d", len(fields))
	}
	return Notification{
		ID:            fields[storage.FieldID],
		Timestamp:     fields[storage.FieldTimestamp],
		State:         fields[storage.FieldState],
		Session:       fields[storage.FieldSession],
		Window:        fields[storage.FieldWindow],
		Pane:          fields[storage.FieldPane],
		Message:       unescapeMessage(fields[storage.FieldMessage]),
		PaneCreated:   fields[storage.FieldPaneCreated],
		Level:         fields[storage.FieldLevel],
		ReadTimestamp: fields[storage.FieldReadTimestamp],
	}, nil
}
