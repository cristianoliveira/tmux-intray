// Package service provides implementations of TUI service interfaces.
package service

import (
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// DefaultNotificationService implements the NotificationService interface.
type DefaultNotificationService struct {
	searchProvider search.Provider
	nameResolver   model.NameResolver
	settings       *settings.Settings
	notifications  []notification.Notification
	filtered       []notification.Notification
}

// NewNotificationService creates a new DefaultNotificationService.
func NewNotificationService(provider search.Provider, resolver model.NameResolver) model.NotificationService {
	return &DefaultNotificationService{
		searchProvider: provider,
		nameResolver:   resolver,
		settings:       nil, // Will be set later
		notifications:  []notification.Notification{},
		filtered:       []notification.Notification{},
	}
}

// SetSettings updates the settings used by the service.
// This allows the service to apply configurable sorting based on user settings.
func (s *DefaultNotificationService) SetSettings(setts *settings.Settings) {
	s.settings = setts
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
// Uses the service's settings to determine whether to apply unread-first grouping.
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
	if len(domainNotifs) != len(notifications) {
		result := make([]notification.Notification, len(notifications))
		copy(result, notifications)
		return result
	}

	// Apply sorting with or without unread-first based on settings
	var sorted []domain.Notification
	unreadFirst := s.shouldApplyUnreadFirst()
	if unreadFirst {
		sorted = domain.SortWithUnreadFirst(domainNotifs, opts)
	} else {
		sorted = domain.SortNotifications(domainNotifs, opts)
	}

	return s.convertFromDomain(sorted)
}

// shouldApplyUnreadFirst determines whether to apply unread-first grouping.
// Returns true if settings are configured to use unread-first sorting.
func (s *DefaultNotificationService) shouldApplyUnreadFirst() bool {
	if s.settings == nil {
		// Default to true for backward compatibility
		return true
	}
	return s.settings.UnreadFirst
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

// SetNotifications updates the underlying notification dataset.
func (s *DefaultNotificationService) SetNotifications(notifications []notification.Notification) {
	s.notifications = notifications
	// Initialize filtered to match notifications (no filters applied by default)
	s.filtered = notifications
}

// GetNotifications returns all notifications currently tracked by the service.
func (s *DefaultNotificationService) GetNotifications() []notification.Notification {
	return s.notifications
}

// GetFilteredNotifications returns the latest filtered notification view.
func (s *DefaultNotificationService) GetFilteredNotifications() []notification.Notification {
	return s.filtered
}

// FilterByReadStatus filters notifications by read status.
func (s *DefaultNotificationService) FilterByReadStatus(notifications []notification.Notification, readFilter string) []notification.Notification {
	if readFilter == "" {
		return notifications
	}
	domainNotifs := s.convertToDomain(notifications)
	filtered := domain.FilterByReadStatus(domainNotifs, readFilter)
	return s.convertFromDomain(filtered)
}

// ApplyFiltersAndSearch applies filters/search/sorting and stores filtered results.
func (s *DefaultNotificationService) ApplyFiltersAndSearch(query, state, level, sessionID, windowID, paneID, readFilter, sortBy, sortOrder string) {
	result := s.notifications
	// Apply state filter
	if state != "" {
		result = s.FilterByState(result, state)
	}
	// Apply level filter
	if level != "" {
		result = s.FilterByLevel(result, level)
	}
	// Apply session filter
	if sessionID != "" {
		result = s.FilterBySession(result, sessionID)
	}
	// Apply window filter
	if windowID != "" {
		result = s.FilterByWindow(result, windowID)
	}
	// Apply pane filter
	if paneID != "" {
		result = s.FilterByPane(result, paneID)
	}
	// Apply read filter
	if readFilter != "" {
		result = s.FilterByReadStatus(result, readFilter)
	}
	// Apply search query filter
	if query != "" {
		result = s.FilterNotifications(result, query)
	}
	// Apply sorting
	result = s.SortNotifications(result, sortBy, sortOrder)
	s.filtered = result
}
