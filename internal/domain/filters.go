// Package domain provides the domain layer for notifications.
// It contains business logic, value objects, and domain services.
package domain

import (
	"fmt"
	"strings"
	"time"
)

// Read filter constants.
const (
	ReadFilterRead   = "read"
	ReadFilterUnread = "unread"
)

// Filter holds filter criteria for notifications.
type Filter struct {
	Level      NotificationLevel
	State      NotificationState
	Session    string
	Window     string
	Pane       string
	OlderThan  string // RFC3339 timestamp (>=)
	NewerThan  string // RFC3339 timestamp (<=)
	ReadFilter string // "read", "unread", or "" (no filter)
}

// FilterOptions holds filter parameters similar to CLI options.
type FilterOptions struct {
	State      string
	Level      string
	Session    string
	Window     string
	Pane       string
	OlderThan  int    // days
	NewerThan  int    // days
	ReadFilter string // "read", "unread", or ""
}

// ToFilter converts FilterOptions to a Filter struct.
func (fo FilterOptions) ToFilter() (Filter, error) {
	var level NotificationLevel
	var err error

	if fo.Level != "" {
		level, err = ParseNotificationLevel(fo.Level)
		if err != nil {
			return Filter{}, err
		}
	}

	var state NotificationState
	if fo.State != "" {
		state, err = ParseNotificationState(fo.State)
		if err != nil {
			return Filter{}, err
		}
	}

	// Validate read filter
	if fo.ReadFilter != "" && fo.ReadFilter != ReadFilterRead && fo.ReadFilter != ReadFilterUnread {
		return Filter{}, fmt.Errorf("invalid read filter: %s", fo.ReadFilter)
	}

	// Convert days to RFC3339 timestamps
	var olderThan, newerThan string
	if fo.OlderThan > 0 {
		t := time.Now().UTC().AddDate(0, 0, -fo.OlderThan)
		olderThan = t.Format(time.RFC3339)
	}
	if fo.NewerThan > 0 {
		t := time.Now().UTC().AddDate(0, 0, -fo.NewerThan)
		newerThan = t.Format(time.RFC3339)
	}

	return Filter{
		Level:      level,
		State:      state,
		Session:    fo.Session,
		Window:     fo.Window,
		Pane:       fo.Pane,
		OlderThan:  olderThan,
		NewerThan:  newerThan,
		ReadFilter: fo.ReadFilter,
	}, nil
}

// FilterNotifications filters a slice of notifications based on the given filter.
// Returns a new slice containing only matching notifications.
func FilterNotifications(notifs []Notification, filter Filter) []Notification {
	if filter.IsEmpty() {
		return notifs
	}

	result := make([]Notification, 0, len(notifs))
	for _, n := range notifs {
		if n.MatchesFilter(filter) {
			result = append(result, n)
		}
	}
	return result
}

// IsEmpty returns true if the filter has no criteria set.
func (f Filter) IsEmpty() bool {
	return f.Level == "" &&
		f.State == "" &&
		f.Session == "" &&
		f.Window == "" &&
		f.Pane == "" &&
		f.OlderThan == "" &&
		f.NewerThan == "" &&
		f.ReadFilter == ""
}

// FilterByLevel filters notifications by level.
func FilterByLevel(notifs []Notification, level string) []Notification {
	if level == "" {
		return notifs
	}
	return FilterNotifications(notifs, Filter{Level: NotificationLevel(level)})
}

// FilterByState filters notifications by state.
func FilterByState(notifs []Notification, state string) []Notification {
	if state == "" {
		return notifs
	}
	return FilterNotifications(notifs, Filter{State: NotificationState(state)})
}

// FilterBySession filters notifications by session ID.
func FilterBySession(notifs []Notification, session string) []Notification {
	if session == "" {
		return notifs
	}
	return FilterNotifications(notifs, Filter{Session: session})
}

// FilterByWindow filters notifications by window ID.
func FilterByWindow(notifs []Notification, window string) []Notification {
	if window == "" {
		return notifs
	}
	return FilterNotifications(notifs, Filter{Window: window})
}

// FilterByPane filters notifications by pane ID.
func FilterByPane(notifs []Notification, pane string) []Notification {
	if pane == "" {
		return notifs
	}
	return FilterNotifications(notifs, Filter{Pane: pane})
}

// FilterByReadStatus filters notifications by read status.
func FilterByReadStatus(notifs []Notification, readFilter string) []Notification {
	if readFilter == "" {
		return notifs
	}
	return FilterNotifications(notifs, Filter{ReadFilter: readFilter})
}

// FilterByTimeRange filters notifications by time range.
func FilterByTimeRange(notifs []Notification, olderThan, newerThan int) []Notification {
	if olderThan == 0 && newerThan == 0 {
		return notifs
	}
	filter := Filter{OlderThan: "", NewerThan: ""}
	if olderThan > 0 {
		t := time.Now().UTC().AddDate(0, 0, -olderThan)
		filter.OlderThan = t.Format(time.RFC3339)
	}
	if newerThan > 0 {
		t := time.Now().UTC().AddDate(0, 0, -newerThan)
		filter.NewerThan = t.Format(time.RFC3339)
	}
	return FilterNotifications(notifs, filter)
}

// SearchNotifications filters notifications by searching for a pattern.
// This is a simple substring search across multiple fields.
func SearchNotifications(notifs []Notification, query string, caseInsensitive bool) []Notification {
	if query == "" {
		return notifs
	}

	searchQuery := query
	if caseInsensitive {
		searchQuery = strings.ToLower(query)
	}

	result := make([]Notification, 0)
	for _, n := range notifs {
		text := n.Message + " " + n.Session + " " + n.Window + " " + n.Pane + " " + n.Level.String()
		searchText := text
		if caseInsensitive {
			searchText = strings.ToLower(text)
		}

		if strings.Contains(searchText, searchQuery) {
			result = append(result, n)
		}
	}
	return result
}
