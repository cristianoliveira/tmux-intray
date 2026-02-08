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
	SessionName   string
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
	switch len(fields) {
	case 9:
		// Old format without ReadTimestamp
		fields = append(fields, "") // Add empty ReadTimestamp
		fields = append(fields, "") // Add empty SessionName
	case 10:
		// Format with ReadTimestamp but without SessionName
		fields = append(fields, "") // Add empty SessionName
	case 11:
		// OK - new format with SessionName
	default:
		return Notification{}, fmt.Errorf("invalid notification field count: %d", len(fields))
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
		SessionName:   fields[4],
		Window:        fields[5],
		Pane:          fields[6],
		Message:       unescapeMessage(fields[7]),
		PaneCreated:   fields[8],
		Level:         fields[9],
		ReadTimestamp: fields[10],
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
