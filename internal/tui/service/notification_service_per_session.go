// Package service provides implementations of TUI service interfaces.
package service

import "github.com/cristianoliveira/tmux-intray/internal/notification"

// selectBestNotificationPerSession groups notifications by session and selects
// the best representative from each session based on severity and recency.
// Returns the selected notifications up to the dataset limit, ordered by:
// 1. Most recent activity first
// 2. Then by severity (error > warning > info) for ties
func (s *DefaultNotificationService) selectBestNotificationPerSession(sorted []notification.Notification) []notification.Notification {
	// Group by session and select best per session
	sessionBest := make(map[string]notification.Notification)
	for _, notif := range sorted {
		sessionKey := notif.Session
		if current, exists := sessionBest[sessionKey]; !exists {
			// First notification for this session
			sessionBest[sessionKey] = notif
		} else if isBetterRepresentative(notif, current) {
			// Found better representative for this session
			sessionBest[sessionKey] = notif
		}
	}

	// Convert map to slice and re-sort by recency and severity
	result := make([]notification.Notification, 0, len(sessionBest))
	for _, notif := range sessionBest {
		result = append(result, notif)
	}

	// Sort by timestamp (descending) then by severity (descending) for ties
	result = s.SortNotifications(result, "timestamp", "desc")

	// Apply dataset limit
	if len(result) > recentsDatasetLimit {
		result = result[:recentsDatasetLimit]
	}

	return result
}
