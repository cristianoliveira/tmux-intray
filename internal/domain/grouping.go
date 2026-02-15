// Package domain provides the domain layer for notifications.
// It contains business logic, value objects, and domain services.
package domain

import (
	"sort"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/dedup"
)

// GroupByMode specifies how notifications should be grouped.
type GroupByMode string

const (
	GroupByNone    GroupByMode = "none"
	GroupBySession GroupByMode = "session"
	GroupByWindow  GroupByMode = "window"
	GroupByPane    GroupByMode = "pane"
	GroupByLevel   GroupByMode = "level"
	GroupByMessage GroupByMode = "message"
)

// IsValid checks if the group by mode is valid.
func (g GroupByMode) IsValid() bool {
	switch g {
	case GroupByNone, GroupBySession, GroupByWindow, GroupByPane, GroupByLevel, GroupByMessage:
		return true
	default:
		return false
	}
}

// String returns the string representation of the group by mode.
func (g GroupByMode) String() string {
	return string(g)
}

// Group represents a group of notifications.
type Group struct {
	Key           string
	DisplayName   string
	Count         int
	UnreadCount   int
	Notifications []Notification
}

// GroupResult represents the result of grouping notifications.
type GroupResult struct {
	Mode        GroupByMode
	Groups      []Group
	TotalCount  int
	TotalUnread int
}

// GroupNotifications groups notifications by the specified mode using default dedup options.
func GroupNotifications(notifs []Notification, mode GroupByMode) GroupResult {
	return GroupNotificationsWithDedup(notifs, mode, dedup.Options{})
}

// GroupNotificationsWithDedup groups notifications, honoring deduplication options when grouping by message.
func GroupNotificationsWithDedup(notifs []Notification, mode GroupByMode, dedupOpts dedup.Options) GroupResult {
	if !mode.IsValid() {
		mode = GroupByNone
	}

	if mode == GroupByNone || len(notifs) == 0 {
		return GroupResult{
			Mode:        mode,
			Groups:      []Group{},
			TotalCount:  len(notifs),
			TotalUnread: countUnread(notifs),
		}
	}

	groupsMap := make(map[string][]Notification)
	var messageKeys []string
	if mode == GroupByMessage {
		records := buildDedupRecords(notifs)
		messageKeys = dedup.BuildKeys(records, dedupOpts)
	}

	for idx, n := range notifs {
		var key string

		switch mode {
		case GroupBySession:
			key = n.Session
		case GroupByWindow:
			key = n.Session + "\x00" + n.Window
		case GroupByPane:
			key = n.Session + "\x00" + n.Window + "\x00" + n.Pane
		case GroupByLevel:
			key = n.Level.String()
		case GroupByMessage:
			key = messageKeys[idx]
		}

		groupsMap[key] = append(groupsMap[key], n)
	}

	// Convert map to slice and sort by key
	groups := make([]Group, 0, len(groupsMap))
	for key, groupNotifs := range groupsMap {
		groups = append(groups, Group{
			Key:           key,
			DisplayName:   extractDisplayName(key, mode),
			Count:         len(groupNotifs),
			UnreadCount:   countUnread(groupNotifs),
			Notifications: groupNotifs,
		})
	}

	// Sort groups by display name
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].DisplayName != groups[j].DisplayName {
			return groups[i].DisplayName < groups[j].DisplayName
		}
		return groups[i].Key < groups[j].Key
	})

	return GroupResult{
		Mode:        mode,
		Groups:      groups,
		TotalCount:  len(notifs),
		TotalUnread: countUnread(notifs),
	}
}

// extractDisplayName extracts the display name from a group key.
func extractDisplayName(key string, mode GroupByMode) string {
	if key == "" {
		return "(empty)"
	}

	switch mode {
	case GroupBySession:
		return key
	case GroupByWindow:
		parts := splitKey(key)
		if len(parts) >= 2 {
			return parts[1]
		}
		return key
	case GroupByPane:
		parts := splitKey(key)
		if len(parts) >= 3 {
			return parts[2]
		}
		return key
	case GroupByLevel:
		return key
	case GroupByMessage:
		stripped := dedup.StripBucketSuffix(key)
		if idx := strings.IndexRune(stripped, '\x00'); idx >= 0 {
			stripped = stripped[:idx]
		}
		if stripped == "" {
			return "(empty)"
		}
		return stripped
	default:
		return key
	}
}

func buildDedupRecords(notifs []Notification) []dedup.Record {
	records := make([]dedup.Record, len(notifs))
	for i, n := range notifs {
		records[i] = dedup.Record{
			Message:   n.Message,
			Level:     n.Level.String(),
			Session:   n.Session,
			Window:    n.Window,
			Pane:      n.Pane,
			State:     n.State.String(),
			Timestamp: n.Timestamp,
		}
	}
	return records
}

// splitKey splits a composite key into its parts.
func splitKey(key string) []string {
	return splitWithNull(key)
}

// splitWithNull splits a string by null bytes.
func splitWithNull(s string) []string {
	var result []string
	current := ""
	for _, r := range s {
		if r == '\x00' {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// countUnread counts unread notifications in a slice.
func countUnread(notifs []Notification) int {
	count := 0
	for _, n := range notifs {
		if n.ReadTimestamp == "" {
			count++
		}
	}
	return count
}

// GetNotificationsBySession groups notifications by session.
func GetNotificationsBySession(notifs []Notification) []Group {
	result := GroupNotifications(notifs, GroupBySession)
	return result.Groups
}

// GetNotificationsByWindow groups notifications by window.
func GetNotificationsByWindow(notifs []Notification) []Group {
	result := GroupNotifications(notifs, GroupByWindow)
	return result.Groups
}

// GetNotificationsByPane groups notifications by pane.
func GetNotificationsByPane(notifs []Notification) []Group {
	result := GroupNotifications(notifs, GroupByPane)
	return result.Groups
}

// GetNotificationsByLevel groups notifications by level.
func GetNotificationsByLevel(notifs []Notification) []Group {
	result := GroupNotifications(notifs, GroupByLevel)
	return result.Groups
}

// GetNotificationsByMessage groups notifications by message.
func GetNotificationsByMessage(notifs []Notification) []Group {
	result := GroupNotifications(notifs, GroupByMessage)
	return result.Groups
}

// GetGroupCounts returns a map of group keys to their counts.
func GetGroupCounts(notifs []Notification, mode GroupByMode) map[string]int {
	if !mode.IsValid() {
		return nil
	}

	result := GroupNotifications(notifs, mode)
	counts := make(map[string]int, len(result.Groups))
	for _, group := range result.Groups {
		counts[group.Key] = group.Count
	}
	return counts
}
