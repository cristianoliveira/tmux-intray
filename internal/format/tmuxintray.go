// Package format provides output formatting functionality for CLI commands.
// This file handles formatting of tmuxintray.Notification types.
package format

import (
	"fmt"
	"io"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

// TmuxIntrayNotificationPrinter handles printing of tmuxintray.Notification objects.
type TmuxIntrayNotificationPrinter struct {
	writer io.Writer
}

// NewTmuxIntrayNotificationPrinter creates a new notification printer.
func NewTmuxIntrayNotificationPrinter(w io.Writer) *TmuxIntrayNotificationPrinter {
	return &TmuxIntrayNotificationPrinter{writer: w}
}

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

// PrintNotification prints a single notification in a human-readable format.
func (p *TmuxIntrayNotificationPrinter) PrintNotification(notification TmuxIntrayNotification, showIndex bool, index int) {
	var prefix string
	if showIndex {
		prefix = fmt.Sprintf("%d: ", index)
	}
	colors.Info("%s[%s] %s (%s:%s.%s)",
		prefix,
		notification.Level,
		notification.Message,
		notification.Session,
		notification.Window,
		notification.Pane,
	)
}

// PrintNotifications prints multiple notifications with optional index display.
func (p *TmuxIntrayNotificationPrinter) PrintNotifications(notifications []TmuxIntrayNotification, showIndex bool) {
	if len(notifications) == 0 {
		colors.Info("No notifications")
		return
	}

	for i, notif := range notifications {
		p.PrintNotification(notif, showIndex, i+1)
	}
}

// PrintError prints an error message using the error color.
func PrintError(msg string) {
	colors.Error(msg)
}

// PrintInfo prints an info message.
func PrintInfo(msg string) {
	colors.Info(msg)
}

// PrintDebug prints a debug message.
func PrintDebug(msg string) {
	colors.Debug(msg)
}

// FormatValidationError formats a validation error message.
func FormatValidationError(field, value, validOptions string) string {
	return fmt.Sprintf("invalid %s: %s (valid: %s)", field, value, validOptions)
}
