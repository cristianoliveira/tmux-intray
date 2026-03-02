package tmuxintray

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

var coreInstance *core.Core = core.Default()

// SetCore sets the core instance for all tmuxintray functions.
// This should be called during initialization.
func SetCore(c *core.Core) {
	coreInstance = c
}

// GetCore returns the current core instance.
func GetCore() *core.Core {
	return coreInstance
}

// ListAllNotifications returns all notifications as TSV lines.
func ListAllNotifications() (string, error) {
	return storage.ListNotifications("", "", "", "", "", "", "", "")
}

// ListNotifications returns notifications with optional filters.
func ListNotifications(level, state string) (string, error) {
	return storage.ListNotifications(state, level, "", "", "", "", "", "")
}

// GetActiveCount returns the count of active notifications.
func GetActiveCount() int {
	return storage.GetActiveCount()
}

var getVisibilityFunc = func() (string, error) {
	return coreInstance.GetVisibility()
}

// GetVisibility returns current visibility state as "0" (hidden) or "1" (visible).
func GetVisibility() (string, error) {
	return getVisibilityFunc()
}

// SetVisibility sets tray visibility.
func SetVisibility(visible bool) error {
	return coreInstance.SetVisibility(visible)
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
		return 0, fmt.Errorf("invalid index: must be a number")
	}

	// Parse as integer
	num, err := strconv.Atoi(idx)
	if err != nil {
		return 0, fmt.Errorf("invalid index: %w", err)
	}

	// Check bounds
	if num <= 0 {
		return 0, fmt.Errorf("invalid index: must be greater than 0")
	}

	return num, nil
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
		return fmt.Errorf("invalid state: %s (valid: active, dismissed)", state)
	}
	return nil
}

// GetStateDir returns the state directory path.
func GetStateDir() string {
	return storage.GetStateDir()
}
