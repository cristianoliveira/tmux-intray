// Package service provides implementations of TUI service interfaces.
package service

import (
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

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

// getMostRecentPerSession returns the most recent notification for each unique session.
// This is used for the Sessions tab to show sessions that have messages,
// sorted by recency (most recent message first).
//
// This method delegates to the shared domain function domain.GroupBySessionKeepMostRecent
// to ensure consistent behavior between CLI and TUI.
func (s *DefaultNotificationService) getMostRecentPerSession(notifications []notification.Notification, sortBy, sortOrder string) []notification.Notification {
	if len(notifications) == 0 {
		return notifications
	}

	// Convert to domain notifications using unsafe version (data from storage is pre-validated)
	domainNotifPtrs := notification.ToDomainSliceUnsafe(notifications)
	domainNotifs := make([]domain.Notification, 0, len(domainNotifPtrs))
	for _, n := range domainNotifPtrs {
		if n != nil {
			domainNotifs = append(domainNotifs, *n)
		}
	}

	// Use shared domain function to get most recent per session (no time limit)
	sessionGroups := domain.GroupBySessionKeepMostRecent(domainNotifs)

	// Convert back to notification.Notification
	result := s.convertSessionNotificationsFromDomain(sessionGroups)

	// Sort by recency (most recent first)
	if sortBy == "" {
		sortBy = "timestamp"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}
	result = s.SortNotifications(result, sortBy, sortOrder)

	return result
}

// convertSessionNotificationsFromDomain converts []domain.SessionNotification to []notification.Notification.
func (s *DefaultNotificationService) convertSessionNotificationsFromDomain(sessions []domain.SessionNotification) []notification.Notification {
	result := make([]notification.Notification, 0, len(sessions))
	for _, sn := range sessions {
		result = append(result, s.convertFromDomainSingle(sn.Notification))
	}
	return result
}
