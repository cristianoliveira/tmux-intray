// Package service provides implementations of TUI service interfaces.
package service

import (
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// severityRank returns a numeric rank for severity comparison (higher = more severe).
// Used for per-session selection: error > warning > info.
func severityRank(level string) int {
	switch level {
	case "error":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// isBetterRepresentative returns true if candidate is a better representative
// for a session than current. Selection priority:
// 1. Higher severity (error > warning > info)
// 2. More recent timestamp (if severity ties)
func isBetterRepresentative(candidate, current notification.Notification) bool {
	candidateRank := severityRank(candidate.Level)
	currentRank := severityRank(current.Level)

	if candidateRank != currentRank {
		return candidateRank > currentRank
	}

	// Severity ties: prefer more recent
	candidateTime, errC := time.Parse(time.RFC3339, candidate.Timestamp)
	currentTime, errCurr := time.Parse(time.RFC3339, current.Timestamp)

	if errC != nil || errCurr != nil {
		// If we can't parse, keep current
		return false
	}

	return candidateTime.After(currentTime)
}

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

// getUnfilteredRecentsDataset returns active, unread notifications from the last hour
// without per-session limiting (used for filtered views).
// The limit parameter controls the maximum number of items returned (e.g., 10 for filtered views, 20 for unfiltered).
func (s *DefaultNotificationService) getUnfilteredRecentsDataset(sortBy, sortOrder string, limit int) []notification.Notification {
	activeOnly := make([]notification.Notification, 0, len(s.notifications))
	for _, n := range s.notifications {
		if n.State == "" || n.State == "active" {
			activeOnly = append(activeOnly, n)
		}
	}

	// Apply configurable time window filter
	domainNotifs := s.convertToDomain(activeOnly)
	filtered := domain.FilterByTimeDuration(domainNotifs, getRecentsTimeWindow())
	activeOnly = s.convertFromDomain(filtered)

	// Filter to unread only
	unreadOnly := make([]notification.Notification, 0, len(activeOnly))
	for _, n := range activeOnly {
		if !n.IsRead() {
			unreadOnly = append(unreadOnly, n)
		}
	}

	// Sort without per-session limiting
	sorted := s.SortNotifications(unreadOnly, sortBy, sortOrder)

	// Apply dataset limit without per-session logic
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	return sorted
}
