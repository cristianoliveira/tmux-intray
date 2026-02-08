package tmuxintray

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

// ListAllNotifications returns all notifications as TSV lines.
func ListAllNotifications() (string, error) {
	return storage.ListNotifications("", "", "", "", "", "", "")
}

// ListNotifications returns notifications with optional filters.
func ListNotifications(level, state string) (string, error) {
	return storage.ListNotifications(state, level, "", "", "", "", "")
}

// GetActiveCount returns the count of active notifications.
func GetActiveCount() int {
	return storage.GetActiveCount()
}

var getVisibilityFunc = func() (string, error) {
	return core.GetVisibility()
}

// GetVisibility returns current visibility state as "0" (hidden) or "1" (visible).
func GetVisibility() (string, error) {
	return getVisibilityFunc()
}

// SetVisibility sets tray visibility.
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

// unescapeMessage reverses the escaping applied by the storage package.
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

// ValidateIndex validates the index string format and converts to int.
// Returns the index as int or error if invalid format or out of bounds.
func ValidateIndex(idx string) (int, error) {
	// Check for invalid characters
	if !strings.ContainsAny(idx, "0123456789") {
		colors.Error("invalid index: must be a number")
		return 0, fmt.Errorf("invalid index: must be a number")
	}

	// Parse as integer
	num, err := strconv.Atoi(idx)
	if err != nil {
		colors.Error("invalid index: ", fmt.Sprintf("%v", err))
		return 0, fmt.Errorf("invalid index: %w", err)
	}

	// Check bounds
	if num <= 0 {
		colors.Error("invalid index: must be greater than 0")
		return 0, fmt.Errorf("invalid index: must be greater than 0")
	}

	return num, nil
}

// PrintNotification prints a notification in a human-readable format.
func PrintNotification(notification Notification, showIndex bool, index int) {
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

// PrintNotifications prints notifications with optional index display.
func PrintNotifications(notifications []Notification, showIndex bool) {
	if len(notifications) == 0 {
		colors.Info("No notifications")
		return
	}

	for i, notif := range notifications {
		PrintNotification(notif, showIndex, i+1)
	}
}

// ParseAndPrintNotifications parses TSV lines and prints them with optional index display.
func ParseAndPrintNotifications(tsvLines string, showIndex bool) error {
	if tsvLines == "" {
		colors.Info("No notifications")
		return nil
	}

	var notifications []Notification
	for _, line := range strings.Split(tsvLines, "\n") {
		if line == "" {
			continue
		}
		notif, err := ParseNotification(line)
		if err != nil {
			return fmt.Errorf("failed to parse notification: %w", err)
		}
		notifications = append(notifications, notif)
	}

	PrintNotifications(notifications, showIndex)
	return nil
}

// ValidateLevel checks if the provided notification level is valid.
func ValidateLevel(level string) error {
	if level == "" {
		level = "info"
	}
	validLevels := map[string]bool{
		"info":     true,
		"warning":  true,
		"error":    true,
		"critical": true,
	}
	if !validLevels[level] {
		colors.Error("invalid level: ", fmt.Sprintf("%s (valid: info, warning, error, critical)", level))
		return fmt.Errorf("invalid level: %s (valid: info, warning, error, critical)", level)
	}
	return nil
}

// ValidateState checks if the provided notification state is valid.
func ValidateState(state string) error {
	if state == "" {
		state = "active"
	}
	validStates := map[string]bool{
		"active":    true,
		"dismissed": true,
		"all":       true,
	}
	if !validStates[state] {
		colors.Error("invalid state: ", fmt.Sprintf("%s (valid: active, dismissed)", state))
		return fmt.Errorf("invalid state: %s (valid: active, dismissed)", state)
	}
	return nil
}

// DebugLog prints a debug message if debug mode is enabled.
func DebugLog(msg string) {
	colors.Debug(msg)
}

// GetStateDir returns the state directory path.
func GetStateDir() string {
	return storage.GetStateDir()
}
