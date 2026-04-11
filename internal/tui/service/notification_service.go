// Package service provides implementations of TUI service interfaces.
package service

import (
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
	"github.com/cristianoliveira/tmux-intray/internal/views"
)

// DefaultNotificationService implements the NotificationService interface.
type DefaultNotificationService struct {
	searchProvider search.Provider
	nameResolver   model.NameResolver
	settings       *settings.Settings
	notifications  []notification.Notification
	filtered       []notification.Notification
}

const (
	recentsDatasetLimit    = 20
	recentsPerSourceLimit  = 3
	recentsPerSessionLimit = 1  // Per-session smart selection for Story 2
	filteredListLimit      = 10 // Max items for filtered views (drill-down) per Story 3
)

// getRecentsTimeWindow returns the configured time window for the Recents tab.
// Parses the recents_time_window config value and returns as time.Duration.
// Defaults to 1 hour if config is missing or invalid.
func getRecentsTimeWindow() time.Duration {
	windowStr := config.Get("recents_time_window", "1h")
	duration, err := time.ParseDuration(windowStr)
	if err != nil {
		// This should not happen if validation is working, but fallback to 1 hour
		return time.Hour
	}
	return duration
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

// selectDataset filters active notifications and applies tab-specific logic.
func (s *DefaultNotificationService) selectDataset(activeTab settings.Tab, sortBy, sortOrder string) []notification.Notification {
	normalizedTab := settings.NormalizeTab(string(activeTab))

	if normalizedTab == settings.TabAll {
		domainDataset := s.convertToDomain(s.notifications)
		if len(domainDataset) == 0 && len(s.notifications) > 0 {
			// Fallback for partial/legacy in-memory fixtures that may not fully
			// convert to domain entities during UI-level tests.
			activeOnly := make([]notification.Notification, 0, len(s.notifications))
			for _, n := range s.notifications {
				if n.State == "" || n.State == "active" {
					activeOnly = append(activeOnly, n)
				}
			}
			return activeOnly
		}

		view := views.NewOrchestrator().Build(views.Options{
			Kind:   views.KindActiveNotificationTimeline,
			SortBy: sortBy,
			Order:  sortOrder,
		}, domainDataset)
		return s.convertFromDomain(view.Notifications)
	}

	// Sessions tab: shared per-session orchestration (all-time, per-session representatives).
	if normalizedTab == settings.TabSessions {
		view := views.NewOrchestrator().Build(views.Options{
			Kind:   views.KindSessionHistory,
			SortBy: sortBy,
			Order:  sortOrder,
		}, s.convertToDomain(s.notifications))
		return s.convertFromDomain(view.Notifications)
	}

	activeOnly := make([]notification.Notification, 0, len(s.notifications))
	for _, n := range s.notifications {
		if n.State == "" || n.State == "active" {
			activeOnly = append(activeOnly, n)
		}
	}

	// For Recents tab, apply configurable time window filter
	if normalizedTab == settings.TabRecents {
		domainNotifs := s.convertToDomain(activeOnly)
		filtered := domain.FilterByTimeDuration(domainNotifs, getRecentsTimeWindow())
		activeOnly = s.convertFromDomain(filtered)

		unreadOnly := make([]notification.Notification, 0, len(activeOnly))
		for _, n := range activeOnly {
			if !n.IsRead() {
				unreadOnly = append(unreadOnly, n)
			}
		}

		sorted := s.SortNotifications(unreadOnly, sortBy, sortOrder)

		// Apply per-session smart selection for Recents tab
		// This ensures max 1 notification per session with intelligent selection
		return s.selectBestNotificationPerSession(sorted)
	}

	return activeOnly
}

// ApplyFiltersAndSearch applies tab scope, then filters/search/sorting and stores filtered results.
func (s *DefaultNotificationService) ApplyFiltersAndSearch(tab settings.Tab, query, state, level, sessionID, windowID, paneID, readFilter, sortBy, sortOrder string) {
	normalizedTab := settings.NormalizeTab(string(tab))
	if normalizedTab == settings.TabRecents {
		readFilter = "unread"
	}

	result := s.selectDataset(tab, sortBy, sortOrder)

	// Check if this is a filtered view (drilling down into a specific session/window/pane)
	isFilteredView := sessionID != "" || windowID != "" || paneID != ""

	// If Recents tab with specific filters, show all notifications matching filters
	// (not just the per-session smart selection). Re-fetch without per-session limiting.
	// Apply 10-item limit for filtered views (Story 3).
	if isFilteredView && normalizedTab == settings.TabRecents {
		view := views.NewOrchestrator().Build(views.Options{
			Kind:   views.KindRecentUnreadTimeline,
			SortBy: sortBy,
			Order:  sortOrder,
			Limit:  filteredListLimit,
		}, s.convertToDomain(s.notifications))
		result = s.convertFromDomain(view.Notifications)
	}

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
