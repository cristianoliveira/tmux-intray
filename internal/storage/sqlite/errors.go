package sqlite

import "errors"

var (
	// ErrInvalidNotificationID indicates an empty or malformed notification ID.
	ErrInvalidNotificationID = errors.New("invalid notification ID")
	// ErrNotificationNotFound indicates that a notification cannot be found.
	ErrNotificationNotFound = errors.New("notification not found")
	// ErrNotificationAlreadyDismissed indicates the notification is already dismissed.
	ErrNotificationAlreadyDismissed = errors.New("notification already dismissed")
)

var validLevels = map[string]bool{
	"info":     true,
	"warning":  true,
	"error":    true,
	"critical": true,
}

var validStates = map[string]bool{
	"active":    true,
	"dismissed": true,
	"all":       true,
}
