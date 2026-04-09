// Package domain provides the domain layer for notifications.
// It contains business logic, value objects, and domain services.
package domain

// SessionNotification holds a session's most recent notification.
type SessionNotification struct {
	Session      string
	Notification Notification
}

// GroupBySessionKeepMostRecent groups notifications by session, keeping only the most recent
// notification per session. Returns a slice sorted by timestamp descending (most recent first).
func GroupBySessionKeepMostRecent(notifications []Notification) []SessionNotification {
	if len(notifications) == 0 {
		return nil
	}

	// Group by session, keeping the most recent
	sessionMap := make(map[string]Notification)
	for _, notif := range notifications {
		session := notif.Session
		if session == "" {
			continue // Skip notifications without session
		}

		existing, exists := sessionMap[session]
		if !exists || notif.Timestamp > existing.Timestamp {
			sessionMap[session] = notif
		}
	}

	// Convert to slice
	result := make([]SessionNotification, 0, len(sessionMap))
	for session, notif := range sessionMap {
		result = append(result, SessionNotification{
			Session:      session,
			Notification: notif,
		})
	}

	// Sort by timestamp descending (most recent first)
	SortByTimestampDesc(result)

	return result
}

// SortByTimestampDesc sorts SessionNotifications by timestamp descending.
func SortByTimestampDesc(sessions []SessionNotification) {
	SortSlice(sessions, func(i, j int) bool {
		left := sessions[i]
		right := sessions[j]

		if left.Notification.Timestamp == right.Notification.Timestamp {
			return left.Session < right.Session
		}

		return left.Notification.Timestamp > right.Notification.Timestamp
	})
}

// SortSlice is a generic slice sorter for SessionNotification.
func SortSlice(sessions []SessionNotification, less func(i, j int) bool) {
	for i := 1; i < len(sessions); i++ {
		for j := i; j > 0 && less(j, j-1); j-- {
			sessions[j], sessions[j-1] = sessions[j-1], sessions[j]
		}
	}
}
