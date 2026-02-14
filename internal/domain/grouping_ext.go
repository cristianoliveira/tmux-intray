// Package domain provides the domain layer for notifications.
// It contains business logic, value objects, and domain services.
package domain

import (
	"fmt"
	"sort"
	"time"
)

// GroupingCriteria represents the criteria used to group and deduplicate notifications.
type GroupingCriteria string

const (
	// GroupByExactMatch groups notifications that are exactly the same.
	GroupByExactMatch GroupingCriteria = "exact_match"
	// GroupByMessageAndLevel groups notifications with the same message and level.
	GroupByMessageAndLevel GroupingCriteria = "message_and_level"
	// GroupByMessageAndSource groups notifications with the same message and source (session+window+pane).
	GroupByMessageAndSource GroupingCriteria = "message_and_source"
	// GroupByTimeWindow groups notifications within a time window that have similar content.
	GroupByTimeWindow GroupingCriteria = "time_window"
)

// IsValid checks if the grouping criteria is valid.
func (g GroupingCriteria) IsValid() bool {
	switch g {
	case GroupByExactMatch, GroupByMessageAndLevel, GroupByMessageAndSource, GroupByTimeWindow:
		return true
	default:
		return false
	}
}

// String returns the string representation of the grouping criteria.
func (g GroupingCriteria) String() string {
	return string(g)
}

// NotificationGroup represents a group of notifications that are considered duplicates or similar.
type NotificationGroup struct {
	ID             string
	Criteria       GroupingCriteria
	Representative *Notification   // The representative notification for this group
	Notifications  []*Notification // All notifications in this group
	Count          int             // Total number of notifications in the group
	FirstSeen      time.Time       // Timestamp of the first notification in the group
	LastSeen       time.Time       // Timestamp of the last notification in the group
	Aggregated     bool            // Whether this group has been aggregated for display
}

// NewNotificationGroup creates a new notification group.
func NewNotificationGroup(criteria GroupingCriteria, representative *Notification) *NotificationGroup {
	now := time.Now()
	return &NotificationGroup{
		ID:             generateGroupID(criteria, representative),
		Criteria:       criteria,
		Representative: representative,
		Notifications:  []*Notification{representative},
		Count:          1,
		FirstSeen:      now,
		LastSeen:       now,
		Aggregated:     false,
	}
}

// AddNotification adds a notification to the group.
func (ng *NotificationGroup) AddNotification(notification *Notification) {
	ng.Notifications = append(ng.Notifications, notification)
	ng.Count++
	if notification.Timestamp != "" {
		t, _ := time.Parse(time.RFC3339, notification.Timestamp)
		if t.Before(ng.FirstSeen) {
			ng.FirstSeen = t
		}
		if t.After(ng.LastSeen) {
			ng.LastSeen = t
		}
	}
}

// GetDisplayName returns a human-readable display name for the group.
func (ng *NotificationGroup) GetDisplayName() string {
	switch ng.Criteria {
	case GroupByExactMatch:
		return fmt.Sprintf("Exact Match: %s", ng.Representative.Message)
	case GroupByMessageAndLevel:
		return fmt.Sprintf("%s (%s)", ng.Representative.Message, ng.Representative.Level)
	case GroupByMessageAndSource:
		return fmt.Sprintf("%s (%s/%s/%s)", ng.Representative.Message, ng.Representative.Session, ng.Representative.Window, ng.Representative.Pane)
	case GroupByTimeWindow:
		return fmt.Sprintf("%s (Time Window)", ng.Representative.Message)
	default:
		return ng.Representative.Message
	}
}

// GetUnreadCount returns the number of unread notifications in the group.
func (ng *NotificationGroup) GetUnreadCount() int {
	count := 0
	for _, n := range ng.Notifications {
		if n.ReadTimestamp == "" {
			count++
		}
	}
	return count
}

// GroupingConfig represents the configuration for notification grouping.
type GroupingConfig struct {
	Criteria          GroupingCriteria
	TimeWindow        time.Duration // For time window grouping
	MaxGroupSize      int           // Maximum number of notifications to keep in a group
	EnableAggregation bool          // Whether to aggregate groups for display
}

// GroupingResult represents the result of grouping notifications.
type GroupingResult struct {
	Config      GroupingConfig
	Groups      []*NotificationGroup
	TotalCount  int
	TotalUnread int
}

// GroupNotificationsByCriteria groups notifications based on the specified criteria.
func GroupNotificationsByCriteria(notifications []*Notification, config GroupingConfig) (*GroupingResult, error) {
	if !config.Criteria.IsValid() {
		return nil, fmt.Errorf("invalid grouping criteria: %s", config.Criteria)
	}

	groupsMap := make(map[string]*NotificationGroup)

	for _, notification := range notifications {
		key := generateGroupKey(config.Criteria, notification)

		if group, exists := groupsMap[key]; exists {
			group.AddNotification(notification)
		} else {
			group := NewNotificationGroup(config.Criteria, notification)
			groupsMap[key] = group
		}
	}

	// Convert map to slice
	groups := make([]*NotificationGroup, 0, len(groupsMap))
	for _, group := range groupsMap {
		// Apply aggregation if enabled and group size exceeds max
		if config.EnableAggregation && config.MaxGroupSize > 0 && group.Count > config.MaxGroupSize {
			group.Aggregated = true
		}
		groups = append(groups, group)
	}

	// Sort groups by first seen time (newest first)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].FirstSeen.After(groups[j].FirstSeen)
	})

	// Calculate totals
	totalCount := len(notifications)
	totalUnread := 0
	for _, group := range groups {
		totalUnread += group.GetUnreadCount()
	}

	return &GroupingResult{
		Config:      config,
		Groups:      groups,
		TotalCount:  totalCount,
		TotalUnread: totalUnread,
	}, nil
}

// generateGroupID generates a unique ID for a notification group.
func generateGroupID(criteria GroupingCriteria, notification *Notification) string {
	switch criteria {
	case GroupByExactMatch:
		return fmt.Sprintf("%s:%s", criteria, notification.Message)
	case GroupByMessageAndLevel:
		return fmt.Sprintf("%s:%s:%s", criteria, notification.Message, notification.Level)
	case GroupByMessageAndSource:
		return fmt.Sprintf("%s:%s:%s:%s:%s", criteria, notification.Message, notification.Session, notification.Window, notification.Pane)
	case GroupByTimeWindow:
		return fmt.Sprintf("%s:%s", criteria, notification.Message)
	default:
		return fmt.Sprintf("%s:%d", criteria, notification.ID)
	}
}

// generateGroupKey generates a key for grouping notifications.
func generateGroupKey(criteria GroupingCriteria, notification *Notification) string {
	switch criteria {
	case GroupByExactMatch:
		return notification.Message
	case GroupByMessageAndLevel:
		return fmt.Sprintf("%s:%s", notification.Message, notification.Level)
	case GroupByMessageAndSource:
		return fmt.Sprintf("%s:%s:%s:%s", notification.Message, notification.Session, notification.Window, notification.Pane)
	case GroupByTimeWindow:
		// For time window, we use message and a time-based key
		return notification.Message
	default:
		return fmt.Sprintf("%d", notification.ID)
	}
}

// IsDuplicate checks if a notification is a duplicate based on the grouping criteria.
func IsDuplicate(criteria GroupingCriteria, existing, new *Notification) bool {
	switch criteria {
	case GroupByExactMatch:
		return existing.Message == new.Message
	case GroupByMessageAndLevel:
		return existing.Message == new.Message && existing.Level == new.Level
	case GroupByMessageAndSource:
		return existing.Message == new.Message &&
			existing.Session == new.Session &&
			existing.Window == new.Window &&
			existing.Pane == new.Pane
	case GroupByTimeWindow:
		// Check if messages are similar and within time window
		if existing.Message != new.Message {
			return false
		}
		// For time window, we'd need to check timestamps, but this is a simplified version
		return true
	default:
		return false
	}
}

// GetGroupedNotificationsByCriteria returns grouped notifications for a specific criteria.
func GetGroupedNotificationsByCriteria(notifications []*Notification, criteria GroupingCriteria) ([]*NotificationGroup, error) {
	config := GroupingConfig{
		Criteria:          criteria,
		TimeWindow:        5 * time.Minute, // Default 5 minute window
		MaxGroupSize:      10,              // Default max group size
		EnableAggregation: true,
	}

	result, err := GroupNotificationsByCriteria(notifications, config)
	if err != nil {
		return nil, err
	}

	return result.Groups, nil
}

// GetGroupedNotificationsByExactMatch groups notifications by exact match.
func GetGroupedNotificationsByExactMatch(notifications []*Notification) ([]*NotificationGroup, error) {
	return GetGroupedNotificationsByCriteria(notifications, GroupByExactMatch)
}

// GetGroupedNotificationsByMessageAndLevel groups notifications by message and level.
func GetGroupedNotificationsByMessageAndLevel(notifications []*Notification) ([]*NotificationGroup, error) {
	return GetGroupedNotificationsByCriteria(notifications, GroupByMessageAndLevel)
}

// GetGroupedNotificationsByMessageAndSource groups notifications by message and source.
func GetGroupedNotificationsByMessageAndSource(notifications []*Notification) ([]*NotificationGroup, error) {
	return GetGroupedNotificationsByCriteria(notifications, GroupByMessageAndSource)
}

// GetGroupedNotificationsByTimeWindow groups notifications by time window.
func GetGroupedNotificationsByTimeWindow(notifications []*Notification, window time.Duration) ([]*NotificationGroup, error) {
	config := GroupingConfig{
		Criteria:          GroupByTimeWindow,
		TimeWindow:        window,
		MaxGroupSize:      10,
		EnableAggregation: true,
	}

	result, err := GroupNotificationsByCriteria(notifications, config)
	if err != nil {
		return nil, err
	}

	return result.Groups, nil
}
