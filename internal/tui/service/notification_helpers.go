// Package service provides implementations of TUI service interfaces.
package service

import (
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

// severityRank returns a numeric rank for severity comparison (higher = more severe).
// Used for per-session selection: error > warning > info.
func severityRank(level domain.NotificationLevel) int {
	switch level {
	case domain.LevelError:
		return 3
	case domain.LevelWarning:
		return 2
	case domain.LevelInfo:
		return 1
	default:
		return 0
	}
}

// isBetterRepresentative returns true if candidate is a better representative
// for a session than current. Selection priority:
// 1. Higher severity (error > warning > info)
// 2. More recent timestamp (if severity ties)
func isBetterRepresentative(candidate, current domain.Notification) bool {
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

// simpleMatch performs a simple string matching check.
func (s *DefaultNotificationService) simpleMatch(notif domain.Notification, query string) bool {
	lowerQuery := strings.ToLower(query)
	lowerMessage := strings.ToLower(notif.Message)

	if strings.Contains(lowerMessage, lowerQuery) {
		return true
	}

	return s.matchResolvedNames(notif, lowerQuery)
}

// matchResolvedNames checks if the query matches any resolved name.
func (s *DefaultNotificationService) matchResolvedNames(notif domain.Notification, lowerQuery string) bool {
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
