// Package service provides implementations of TUI service interfaces.
package service

import (
	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

// selectBestNotificationPerSession groups notifications by session and selects
// the best representative from each session based on severity and recency.
func (s *DefaultNotificationService) selectBestNotificationPerSession(sorted []domain.Notification) []domain.Notification {
	sessionBest := make(map[string]domain.Notification)
	for _, notif := range sorted {
		sessionKey := notif.Session
		if current, exists := sessionBest[sessionKey]; !exists {
			sessionBest[sessionKey] = notif
		} else if isBetterRepresentative(notif, current) {
			sessionBest[sessionKey] = notif
		}
	}

	result := make([]domain.Notification, 0, len(sessionBest))
	for _, notif := range sessionBest {
		result = append(result, notif)
	}

	result = s.SortNotifications(result, "timestamp", "desc")

	if len(result) > recentsDatasetLimit {
		result = result[:recentsDatasetLimit]
	}

	return result
}

// getUnfilteredRecentsDataset returns active, unread notifications from the configured
// time window without per-session limiting (used for filtered views).
func (s *DefaultNotificationService) getUnfilteredRecentsDataset(sortBy, sortOrder string, limit int) []domain.Notification {
	activeOnly := make([]domain.Notification, 0, len(s.notifications))
	for _, n := range s.notifications {
		if n.State == "" || n.State == domain.StateActive {
			activeOnly = append(activeOnly, n)
		}
	}

	filtered := domain.FilterByTimeDuration(activeOnly, getRecentsTimeWindow())

	unreadOnly := make([]domain.Notification, 0, len(filtered))
	for _, n := range filtered {
		if !n.IsRead() {
			unreadOnly = append(unreadOnly, n)
		}
	}

	sorted := s.SortNotifications(unreadOnly, sortBy, sortOrder)

	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	return sorted
}

// getMostRecentPerSession returns the most recent notification for each unique session.
func (s *DefaultNotificationService) getMostRecentPerSession(notifications []domain.Notification, sortBy, sortOrder string) []domain.Notification {
	if len(notifications) == 0 {
		return notifications
	}

	sessionGroups := domain.GroupBySessionKeepMostRecent(notifications)

	result := make([]domain.Notification, 0, len(sessionGroups))
	for _, sn := range sessionGroups {
		result = append(result, sn.Notification)
	}

	if sortBy == "" {
		sortBy = "timestamp"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}
	result = s.SortNotifications(result, sortBy, sortOrder)

	return result
}
