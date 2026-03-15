// Package service provides implementations of TUI service interfaces.
package service

import (
	"strings"
	"time"

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

	candidateTime, errC := time.Parse(time.RFC3339, candidate.Timestamp)
	currentTime, errCurr := time.Parse(time.RFC3339, current.Timestamp)

	if errC != nil || errCurr != nil {
		return false
	}

	return candidateTime.After(currentTime)
}

// notificationSourceKey creates a unique key for a notification source.
func notificationSourceKey(notif notification.Notification) string {
	return notif.Session + "\x00" + notif.Window + "\x00" + notif.Pane
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// simpleMatch performs a simple string matching check.
func (s *DefaultNotificationService) simpleMatch(notif notification.Notification, query string) bool {
	lowerQuery := strings.ToLower(query)
	lowerMessage := strings.ToLower(notif.Message)

	if strings.Contains(lowerMessage, lowerQuery) {
		return true
	}

	return s.matchResolvedNames(notif, lowerQuery)
}

// matchResolvedNames checks if the query matches any resolved name.
func (s *DefaultNotificationService) matchResolvedNames(notif notification.Notification, lowerQuery string) bool {
	if s.nameResolver == nil {
		return false
	}

	if s.matchesName(s.nameResolver.ResolveSessionName(notif.Session), lowerQuery) {
		return true
	}
	if s.matchesName(s.nameResolver.ResolveWindowName(notif.Window), lowerQuery) {
		return true
	}
	if s.matchesName(s.nameResolver.ResolvePaneName(notif.Pane), lowerQuery) {
		return true
	}
	return false
}

// matchesName checks if the resolved name contains the query (case-insensitive).
func (s *DefaultNotificationService) matchesName(name, lowerQuery string) bool {
	if name == "" {
		return false
	}
	return strings.Contains(strings.ToLower(name), lowerQuery)
}
