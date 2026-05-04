// Package service provides implementations of TUI service interfaces.
package service

import (
	"strings"
	"time"

	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// DefaultNotificationService implements the NotificationService interface.
type DefaultNotificationService struct {
	searchProvider search.Provider
	nameResolver   model.NameResolver
	settings       *settings.Settings
	notifications  []domain.Notification
	filtered       []domain.Notification
	showStale      bool
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
		notifications:  []domain.Notification{},
		filtered:       []domain.Notification{},
	}
}

// SetShowStale controls whether notifications for stale tmux targets remain visible.
func (s *DefaultNotificationService) SetShowStale(show bool) {
	s.showStale = show
}

// SetSettings updates the settings used by the service.
func (s *DefaultNotificationService) SetSettings(setts *settings.Settings) {
	s.settings = setts
}

// FilterNotifications filters notifications based on a search query.
func (s *DefaultNotificationService) FilterNotifications(notifications []domain.Notification, query string) []domain.Notification {
	if query == "" {
		return notifications
	}

	var filtered []domain.Notification
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
func (s *DefaultNotificationService) FilterByState(notifications []domain.Notification, state string) []domain.Notification {
	if state == "" {
		return notifications
	}
	return domain.FilterByState(notifications, state)
}

// FilterByLevel filters notifications by level (info/warning/error).
func (s *DefaultNotificationService) FilterByLevel(notifications []domain.Notification, level string) []domain.Notification {
	if level == "" {
		return notifications
	}
	return domain.FilterByLevel(notifications, level)
}

// FilterBySession filters notifications by session ID.
func (s *DefaultNotificationService) FilterBySession(notifications []domain.Notification, sessionID string) []domain.Notification {
	if sessionID == "" {
		return notifications
	}
	return domain.FilterBySession(notifications, sessionID)
}

// FilterByWindow filters notifications by window ID.
func (s *DefaultNotificationService) FilterByWindow(notifications []domain.Notification, windowID string) []domain.Notification {
	if windowID == "" {
		return notifications
	}
	return domain.FilterByWindow(notifications, windowID)
}

// FilterByPane filters notifications by pane ID.
func (s *DefaultNotificationService) FilterByPane(notifications []domain.Notification, paneID string) []domain.Notification {
	if paneID == "" {
		return notifications
	}
	return domain.FilterByPane(notifications, paneID)
}

// SortNotifications sorts notifications by the specified field and order.
func (s *DefaultNotificationService) SortNotifications(notifications []domain.Notification, sortBy, sortOrder string) []domain.Notification {
	if len(notifications) == 0 {
		return notifications
	}

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
	}

	var sorted []domain.Notification
	unreadFirst := s.shouldApplyUnreadFirst()
	if unreadFirst {
		sorted = domain.SortWithUnreadFirst(notifications, opts)
	} else {
		sorted = domain.SortNotifications(notifications, opts)
	}

	return sorted
}

// shouldApplyUnreadFirst determines whether to apply unread-first grouping.
func (s *DefaultNotificationService) shouldApplyUnreadFirst() bool {
	if s.settings == nil {
		return true
	}
	return s.settings.UnreadFirst
}

// GetUnreadCount returns the number of unread notifications.
func (s *DefaultNotificationService) GetUnreadCount(notifications []domain.Notification) int {
	count := 0
	for _, n := range notifications {
		if !n.IsRead() {
			count++
		}
	}
	return count
}

// GetReadCount returns the number of read notifications.
func (s *DefaultNotificationService) GetReadCount(notifications []domain.Notification) int {
	count := 0
	for _, n := range notifications {
		if n.IsRead() {
			count++
		}
	}
	return count
}

// GetCountsByLevel returns a map of notification counts by level.
func (s *DefaultNotificationService) GetCountsByLevel(notifications []domain.Notification) map[string]int {
	counts := make(map[string]int)
	for _, n := range notifications {
		counts[n.Level.String()]++
	}
	return counts
}

// Search performs a full-text search on notifications.
func (s *DefaultNotificationService) Search(notifications []domain.Notification, query string, caseSensitive bool) []domain.Notification {
	if query == "" {
		return notifications
	}

	searchQuery := query
	if !caseSensitive {
		searchQuery = strings.ToLower(query)
	}

	var results []domain.Notification
	for _, n := range notifications {
		message := n.Message
		if !caseSensitive {
			message = strings.ToLower(n.Message)
		}

		if strings.Contains(message, searchQuery) {
			results = append(results, n)
		}
	}
	return results
}

// SetNotifications updates the underlying notification dataset.
func (s *DefaultNotificationService) SetNotifications(notifications []domain.Notification) {
	s.notifications = notifications
	s.filtered = notifications
}

// GetNotifications returns all notifications currently tracked by the service.
func (s *DefaultNotificationService) GetNotifications() []domain.Notification {
	return s.notifications
}

// GetFilteredNotifications returns the latest filtered notification view.
func (s *DefaultNotificationService) GetFilteredNotifications() []domain.Notification {
	return s.filtered
}

// FilterByReadStatus filters notifications by read status.
func (s *DefaultNotificationService) FilterByReadStatus(notifications []domain.Notification, readFilter string) []domain.Notification {
	if readFilter == "" {
		return notifications
	}
	return domain.FilterByReadStatus(notifications, readFilter)
}

// selectDataset filters active notifications and applies tab-specific logic.
func (s *DefaultNotificationService) selectDataset(activeTab settings.Tab, sortBy, sortOrder string) []domain.Notification {
	activeOnly := make([]domain.Notification, 0, len(s.notifications))
	for _, n := range s.notifications {
		if n.State == "" || n.State == domain.StateActive {
			activeOnly = append(activeOnly, n)
		}
	}

	normalizedTab := settings.NormalizeTab(string(activeTab))
	if normalizedTab == settings.TabAll {
		return activeOnly
	}

	if normalizedTab == settings.TabRecents {
		filtered := domain.FilterByTimeDuration(activeOnly, getRecentsTimeWindow())

		unreadOnly := make([]domain.Notification, 0, len(filtered))
		for _, n := range filtered {
			if !n.IsRead() {
				unreadOnly = append(unreadOnly, n)
			}
		}

		sorted := s.SortNotifications(unreadOnly, sortBy, sortOrder)
		return s.selectBestNotificationPerSession(sorted)
	}

	if normalizedTab == settings.TabSessions {
		return s.getMostRecentPerSession(s.notifications, sortBy, sortOrder)
	}

	return activeOnly
}

// FilterResolvableTmuxTargets hides notifications whose tmux session/window/pane no longer exists.
func (s *DefaultNotificationService) FilterResolvableTmuxTargets(notifications []domain.Notification) []domain.Notification {
	if s.nameResolver == nil {
		return notifications
	}

	return appcore.KeepOnlyResolvableNotifications(notifications, appcore.DisplayNames{
		Sessions: s.nameResolver.GetSessionNames(),
		Windows:  s.nameResolver.GetWindowNames(),
		Panes:    s.nameResolver.GetPaneNames(),
	}, s.showStale)
}

// ApplyFiltersAndSearch applies tab scope, then filters/search/sorting and stores filtered results.
func (s *DefaultNotificationService) ApplyFiltersAndSearch(tab settings.Tab, query, state, level, sessionID, windowID, paneID, readFilter, sortBy, sortOrder string) {
	if settings.NormalizeTab(string(tab)) == settings.TabRecents {
		readFilter = "unread"
	}

	result := s.selectDataset(tab, sortBy, sortOrder)
	result = s.FilterResolvableTmuxTargets(result)

	isFilteredView := sessionID != "" || windowID != "" || paneID != ""

	if isFilteredView && settings.NormalizeTab(string(tab)) == settings.TabRecents {
		result = s.getUnfilteredRecentsDataset(sortBy, sortOrder, filteredListLimit)
	}

	if state != "" {
		result = s.FilterByState(result, state)
	}
	if level != "" {
		result = s.FilterByLevel(result, level)
	}
	if sessionID != "" {
		result = s.FilterBySession(result, sessionID)
	}
	if windowID != "" {
		result = s.FilterByWindow(result, windowID)
	}
	if paneID != "" {
		result = s.FilterByPane(result, paneID)
	}
	if readFilter != "" {
		result = s.FilterByReadStatus(result, readFilter)
	}
	if query != "" {
		result = s.FilterNotifications(result, query)
	}
	result = s.SortNotifications(result, sortBy, sortOrder)
	s.filtered = result
}
