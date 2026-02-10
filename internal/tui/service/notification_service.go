// Package service provides implementations of TUI service interfaces.
package service

import (
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
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

// convertToDomain converts a slice of notification.Notification to domain.Notification values.
func (s *DefaultNotificationService) convertToDomain(notifs []notification.Notification) []domain.Notification {
	domainPtrs, err := notification.ToDomainSlice(notifs)
	if err != nil {
		// Should not happen with valid data; return empty slice
		return []domain.Notification{}
	}
	result := make([]domain.Notification, 0, len(domainPtrs))
	for _, n := range domainPtrs {
		if n != nil {
			result = append(result, *n)
		}
	}
	return result
}

// convertFromDomain converts a slice of domain.Notification values to notification.Notification.
func (s *DefaultNotificationService) convertFromDomain(notifs []domain.Notification) []notification.Notification {
	ptrs := make([]*domain.Notification, len(notifs))
	for i := range notifs {
		ptrs[i] = &notifs[i]
	}
	return notification.FromDomainSlice(ptrs)
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
	domainNotifs := s.convertToDomain(notifications)
	filtered := domain.FilterByState(domainNotifs, state)
	return s.convertFromDomain(filtered)
}

// FilterByLevel filters notifications by level (info/warning/error).
func (s *DefaultNotificationService) FilterByLevel(notifications []notification.Notification, level string) []notification.Notification {
	if level == "" {
		return notifications
	}
	domainNotifs := s.convertToDomain(notifications)
	filtered := domain.FilterByLevel(domainNotifs, level)
	return s.convertFromDomain(filtered)
}

// FilterBySession filters notifications by session ID.
func (s *DefaultNotificationService) FilterBySession(notifications []notification.Notification, sessionID string) []notification.Notification {
	if sessionID == "" {
		return notifications
	}
	domainNotifs := s.convertToDomain(notifications)
	filtered := domain.FilterBySession(domainNotifs, sessionID)
	return s.convertFromDomain(filtered)
}

// FilterByWindow filters notifications by window ID.
func (s *DefaultNotificationService) FilterByWindow(notifications []notification.Notification, windowID string) []notification.Notification {
	if windowID == "" {
		return notifications
	}
	domainNotifs := s.convertToDomain(notifications)
	filtered := domain.FilterByWindow(domainNotifs, windowID)
	return s.convertFromDomain(filtered)
}

// FilterByPane filters notifications by pane ID.
func (s *DefaultNotificationService) FilterByPane(notifications []notification.Notification, paneID string) []notification.Notification {
	if paneID == "" {
		return notifications
	}
	domainNotifs := s.convertToDomain(notifications)
	filtered := domain.FilterByPane(domainNotifs, paneID)
	return s.convertFromDomain(filtered)
}

// SortNotifications sorts notifications by the specified field and order.
func (s *DefaultNotificationService) SortNotifications(notifications []notification.Notification, sortBy, sortOrder string) []notification.Notification {
	if len(notifications) == 0 {
		return notifications
	}

	// Parse sort field and order with defaults
	field, err := domain.ParseSortByField(sortBy)
	if err != nil {
		field = domain.SortByTimestampField
	}
	order, err := domain.ParseSortOrder(sortOrder)
	if err != nil {
		order = domain.SortOrderDesc
	}

	opts := domain.SortOptions{
		Field: field,
		Order: order,
		// CaseInsensitive: false (default)
	}

	domainNotifs := s.convertToDomain(notifications)
	sorted := domain.SortNotifications(domainNotifs, opts)
	return s.convertFromDomain(sorted)
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
