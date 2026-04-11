// Package domain provides the domain layer for notifications.
// It contains business logic, value objects, and domain services.
package domain

import (
	"fmt"
	"time"
)

// Notification represents a single notification entity with business logic.
type Notification struct {
	ID            int
	Timestamp     string
	State         NotificationState
	Session       string
	Window        string
	Pane          string
	Message       string
	PaneCreated   string
	Level         NotificationLevel
	ReadTimestamp string
}

// NotificationState represents the state of a notification.
type NotificationState string

const (
	StateActive    NotificationState = "active"
	StateDismissed NotificationState = "dismissed"
)

// IsValid checks if the notification state is valid.
func (s NotificationState) IsValid() bool {
	switch s {
	case StateActive, StateDismissed:
		return true
	default:
		return false
	}
}

// String returns the string representation of the state.
func (s NotificationState) String() string {
	return string(s)
}

// NotificationLevel represents the severity level of a notification.
type NotificationLevel string

const (
	LevelInfo     NotificationLevel = "info"
	LevelWarning  NotificationLevel = "warning"
	LevelError    NotificationLevel = "error"
	LevelCritical NotificationLevel = "critical"
)

// IsValid checks if the notification level is valid.
func (l NotificationLevel) IsValid() bool {
	switch l {
	case LevelInfo, LevelWarning, LevelError, LevelCritical:
		return true
	default:
		return false
	}
}

// String returns the string representation of the level.
func (l NotificationLevel) String() string {
	return string(l)
}

// IsRead reports whether the notification has a read timestamp.
func (n *Notification) IsRead() bool {
	return n.ReadTimestamp != ""
}

// MarkRead returns a copy of the notification with a read timestamp set.
func (n *Notification) MarkRead() *Notification {
	n.ReadTimestamp = time.Now().UTC().Format(time.RFC3339)
	return n
}

// MarkUnread returns a copy of the notification with no read timestamp.
func (n *Notification) MarkUnread() *Notification {
	n.ReadTimestamp = ""
	return n
}

// Dismiss changes the notification state to dismissed.
func (n *Notification) Dismiss() *Notification {
	n.State = StateDismissed
	return n
}

// Validate validates the notification and returns an error if invalid.
func (n *Notification) Validate() error {
	if n.ID <= 0 {
		return fmt.Errorf("invalid notification ID: %d", n.ID)
	}

	if n.Timestamp == "" {
		return fmt.Errorf("notification timestamp cannot be empty")
	}

	// Validate RFC3339 timestamp format
	if _, err := time.Parse(time.RFC3339, n.Timestamp); err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	if !n.State.IsValid() {
		return fmt.Errorf("invalid notification state: %s", n.State)
	}

	if !n.Level.IsValid() {
		return fmt.Errorf("invalid notification level: %s", n.Level)
	}

	if n.Message == "" {
		return fmt.Errorf("notification message cannot be empty")
	}

	// Validate read timestamp if present
	if n.ReadTimestamp != "" {
		if _, err := time.Parse(time.RFC3339, n.ReadTimestamp); err != nil {
			return fmt.Errorf("invalid read timestamp format: %w", err)
		}
	}

	return nil
}

// MatchesFilter checks if the notification matches the given filter criteria.
func (n *Notification) MatchesFilter(filter Filter) bool {
	checks := []func() bool{
		func() bool { return matchesStringFilter(string(n.Level), string(filter.Level)) },
		func() bool { return matchesStringFilter(string(n.State), string(filter.State)) },
		func() bool { return matchesStringFilter(n.Session, filter.Session) },
		func() bool { return matchesStringFilter(n.Window, filter.Window) },
		func() bool { return matchesStringFilter(n.Pane, filter.Pane) },
		func() bool { return matchesOlderThanFilter(n.Timestamp, filter.OlderThan) },
		func() bool { return matchesNewerThanFilter(n.Timestamp, filter.NewerThan) },
		func() bool { return matchesReadFilter(n.IsRead(), filter.ReadFilter) },
	}

	for _, check := range checks {
		if !check() {
			return false
		}
	}

	return true
}

func matchesStringFilter(value, expected string) bool {
	return expected == "" || value == expected
}

func matchesOlderThanFilter(timestamp, olderThan string) bool {
	return olderThan == "" || timestamp <= olderThan
}

func matchesNewerThanFilter(timestamp, newerThan string) bool {
	return newerThan == "" || timestamp >= newerThan
}

func matchesReadFilter(isRead bool, readFilter string) bool {
	switch readFilter {
	case "", ReadFilterRead:
		return readFilter == "" || isRead
	case ReadFilterUnread:
		return !isRead
	default:
		return true
	}
}

// NewNotification creates a new notification with validation.
func NewNotification(
	id int,
	timestamp string,
	state NotificationState,
	session, window, pane, message, paneCreated string,
	level NotificationLevel,
	readTimestamp string,
) (*Notification, error) {
	notif := &Notification{
		ID:            id,
		Timestamp:     timestamp,
		State:         state,
		Session:       session,
		Window:        window,
		Pane:          pane,
		Message:       message,
		PaneCreated:   paneCreated,
		Level:         level,
		ReadTimestamp: readTimestamp,
	}

	if err := notif.Validate(); err != nil {
		return nil, err
	}

	return notif, nil
}

// ParseNotificationLevel parses a string into a NotificationLevel.
func ParseNotificationLevel(level string) (NotificationLevel, error) {
	nl := NotificationLevel(level)
	if !nl.IsValid() {
		return "", fmt.Errorf("invalid notification level: %s", level)
	}
	return nl, nil
}

// ParseNotificationState parses a string into a NotificationState.
func ParseNotificationState(state string) (NotificationState, error) {
	ns := NotificationState(state)
	if !ns.IsValid() {
		return "", fmt.Errorf("invalid notification state: %s", state)
	}
	return ns, nil
}
