// Package service provides implementations of TUI service interfaces.
package service

import (
	"sort"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// DefaultNotificationService implements the NotificationService interface.
type DefaultNotificationService struct {
	searchProvider search.Provider
	nameResolver   model.NameResolver
	notifications  []notification.Notification
	filtered       []notification.Notification
}

// NewNotificationService creates a new DefaultNotificationService.
func NewNotificationService(provider search.Provider, resolver model.NameResolver) model.NotificationService {
	return &DefaultNotificationService{
		searchProvider: provider,
		nameResolver:   resolver,
	}
}

// SetNotifications updates the underlying notification dataset.
func (s *DefaultNotificationService) SetNotifications(notifications []notification.Notification) {
	s.notifications = append([]notification.Notification(nil), notifications...)
	s.filtered = append([]notification.Notification(nil), notifications...)
}

// GetNotifications returns all notifications currently tracked by the service.
func (s *DefaultNotificationService) GetNotifications() []notification.Notification {
	return append([]notification.Notification(nil), s.notifications...)
}

// GetFilteredNotifications returns the latest filtered notification view.
func (s *DefaultNotificationService) GetFilteredNotifications() []notification.Notification {
	return append([]notification.Notification(nil), s.filtered...)
}

// ApplyFiltersAndSearch applies filters, search and sorting to tracked notifications.
func (s *DefaultNotificationService) ApplyFiltersAndSearch(query, state, level, sessionID, windowID, paneID, sortBy, sortOrder string) {
	working := append([]notification.Notification(nil), s.notifications...)
	working = s.FilterByState(working, state)
	working = s.FilterByLevel(working, level)
	working = s.FilterBySession(working, sessionID)
	working = s.FilterByWindow(working, windowID)
	working = s.FilterByPane(working, paneID)

	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery != "" {
		working = s.FilterNotifications(working, trimmedQuery)
	}

	working = s.SortNotifications(working, sortBy, sortOrder)
	s.filtered = working
}

// FilterNotifications filters notifications based on a search query.
func (s *DefaultNotificationService) FilterNotifications(notifications []notification.Notification, query string) []notification.Notification {
	if query == "" {
		return notifications
	}

	var filtered []notification.Notification
	for _, n := range notifications {
		if s.searchProvider != nil {
			if s.searchProvider.Match(n, query) {
				filtered = append(filtered, n)
			}
		} else {
			// Default: simple string matching
			if s.simpleMatch(n, query) {
				filtered = append(filtered, n)
			}
		}
	}
	return filtered
}

// FilterByState filters notifications by state (active/dismissed).
func (s *DefaultNotificationService) FilterByState(notifications []notification.Notification, state string) []notification.Notification {
	if state == "" {
		return notifications
	}

	var filtered []notification.Notification
	for _, n := range notifications {
		if state == "active" && n.State == "active" {
			filtered = append(filtered, n)
		} else if state == "dismissed" && n.State == "dismissed" {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// FilterByLevel filters notifications by level (info/warning/error).
func (s *DefaultNotificationService) FilterByLevel(notifications []notification.Notification, level string) []notification.Notification {
	if level == "" {
		return notifications
	}

	var filtered []notification.Notification
	for _, n := range notifications {
		if strings.EqualFold(n.Level, level) {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// FilterBySession filters notifications by session ID.
func (s *DefaultNotificationService) FilterBySession(notifications []notification.Notification, sessionID string) []notification.Notification {
	if sessionID == "" {
		return notifications
	}

	var filtered []notification.Notification
	for _, n := range notifications {
		if n.Session == sessionID {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// FilterByWindow filters notifications by window ID.
func (s *DefaultNotificationService) FilterByWindow(notifications []notification.Notification, windowID string) []notification.Notification {
	if windowID == "" {
		return notifications
	}

	var filtered []notification.Notification
	for _, n := range notifications {
		if n.Window == windowID {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// FilterByPane filters notifications by pane ID.
func (s *DefaultNotificationService) FilterByPane(notifications []notification.Notification, paneID string) []notification.Notification {
	if paneID == "" {
		return notifications
	}

	var filtered []notification.Notification
	for _, n := range notifications {
		if n.Pane == paneID {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// SortNotifications sorts notifications by the specified field and order.
func (s *DefaultNotificationService) SortNotifications(notifications []notification.Notification, sortBy, sortOrder string) []notification.Notification {
	if len(notifications) == 0 {
		return notifications
	}

	// Create a copy to avoid modifying the original slice
	sorted := make([]notification.Notification, len(notifications))
	copy(sorted, notifications)

	sort.SliceStable(sorted, func(i, j int) bool {
		var cmp int
		switch sortBy {
		case "timestamp":
			if sorted[i].Timestamp < sorted[j].Timestamp {
				cmp = -1
			} else if sorted[i].Timestamp > sorted[j].Timestamp {
				cmp = 1
			}
		case "level":
			cmp = strings.Compare(sorted[i].Level, sorted[j].Level)
		case "message":
			cmp = strings.Compare(sorted[i].Message, sorted[j].Message)
		case "session":
			cmp = strings.Compare(sorted[i].Session, sorted[j].Session)
		default:
			// Default: sort by timestamp descending
			if sorted[i].Timestamp < sorted[j].Timestamp {
				cmp = 1
			} else if sorted[i].Timestamp > sorted[j].Timestamp {
				cmp = -1
			}
		}

		if sortOrder == "desc" {
			cmp = -cmp
		}
		return cmp < 0
	})

	return sorted
}

// GetUnreadCount returns the number of unread notifications.
func (s *DefaultNotificationService) GetUnreadCount(notifications []notification.Notification) int {
	count := 0
	for _, n := range notifications {
		if !n.IsRead() {
			count++
		}
	}
	return count
}

// GetReadCount returns the number of read notifications.
func (s *DefaultNotificationService) GetReadCount(notifications []notification.Notification) int {
	count := 0
	for _, n := range notifications {
		if n.IsRead() {
			count++
		}
	}
	return count
}

// GetCountsByLevel returns a map of notification counts by level.
func (s *DefaultNotificationService) GetCountsByLevel(notifications []notification.Notification) map[string]int {
	counts := make(map[string]int)
	for _, n := range notifications {
		counts[n.Level]++
	}
	return counts
}

// Search performs a full-text search on notifications.
func (s *DefaultNotificationService) Search(notifications []notification.Notification, query string, caseSensitive bool) []notification.Notification {
	if query == "" {
		return notifications
	}

	searchQuery := query
	if !caseSensitive {
		searchQuery = strings.ToLower(query)
	}

	var results []notification.Notification
	for _, n := range notifications {
		var message string
		if !caseSensitive {
			message = strings.ToLower(n.Message)
		} else {
			message = n.Message
		}

		if strings.Contains(message, searchQuery) {
			results = append(results, n)
		}
	}
	return results
}

// simpleMatch performs a simple string matching check.
func (s *DefaultNotificationService) simpleMatch(notif notification.Notification, query string) bool {
	lowerQuery := strings.ToLower(query)
	lowerMessage := strings.ToLower(notif.Message)

	if strings.Contains(lowerMessage, lowerQuery) {
		return true
	}

	if s.nameResolver != nil {
		if name := s.nameResolver.ResolveSessionName(notif.Session); name != "" {
			if strings.Contains(strings.ToLower(name), lowerQuery) {
				return true
			}
		}
		if name := s.nameResolver.ResolveWindowName(notif.Window); name != "" {
			if strings.Contains(strings.ToLower(name), lowerQuery) {
				return true
			}
		}
		if name := s.nameResolver.ResolvePaneName(notif.Pane); name != "" {
			if strings.Contains(strings.ToLower(name), lowerQuery) {
				return true
			}
		}
	}

	return false
}
