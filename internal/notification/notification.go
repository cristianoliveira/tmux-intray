package notification

import (
	"fmt"
	"strings"
	"time"
)

// Notification represents a single notification record.
type Notification struct {
	ID            int
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

// ParseNotification parses a TSV line into a Notification.
func ParseNotification(line string) (Notification, error) {
	fields := strings.Split(line, "\t")
	if len(fields) < 9 {
		return Notification{}, fmt.Errorf("invalid notification field count: %d", len(fields))
	}
	// Ensure at least 10 fields
	for len(fields) < 10 {
		fields = append(fields, "")
	}
	id := 0
	if fields[0] != "" {
		fmt.Sscanf(fields[0], "%d", &id)
	}
	return Notification{
		ID:            id,
		Timestamp:     fields[1],
		State:         fields[2],
		Session:       fields[3],
		Window:        fields[4],
		Pane:          fields[5],
		Message:       unescapeMessage(fields[6]),
		PaneCreated:   fields[7],
		Level:         fields[8],
		ReadTimestamp: fields[9],
	}, nil
}

// IsRead reports whether the notification has a read timestamp.
func (n Notification) IsRead() bool {
	return n.ReadTimestamp != ""
}

// MarkRead returns a copy of the notification with a read timestamp set.
func (n Notification) MarkRead() Notification {
	n.ReadTimestamp = time.Now().UTC().Format(time.RFC3339)
	return n
}

// MarkUnread returns a copy of the notification with no read timestamp.
func (n Notification) MarkUnread() Notification {
	n.ReadTimestamp = ""
	return n
}

// unescapeMessage reverses the escaping done by storage.escapeMessage.
func unescapeMessage(msg string) string {
	// Unescape newlines first
	msg = strings.ReplaceAll(msg, "\\n", "\n")
	// Unescape tabs
	msg = strings.ReplaceAll(msg, "\\t", "\t")
	// Unescape backslashes
	msg = strings.ReplaceAll(msg, "\\\\", "\\")
	return msg
}
