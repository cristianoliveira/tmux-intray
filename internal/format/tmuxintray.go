// Package format provides output formatting functionality for CLI commands.
// This file handles formatting of tmuxintray.Notification types.
package format

// TmuxIntrayNotification represents a single notification in the tray.
// This type is defined in internal/tmuxintray package but is duplicated here
// to avoid circular dependencies.
type TmuxIntrayNotification struct {
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
