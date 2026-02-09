// Package domain provides the domain layer for notifications.
// It contains business logic, value objects, and domain services.
package domain

import (
	"fmt"
	"sort"
	"strings"
)

// SortByField specifies which field to sort notifications by.
type SortByField string

const (
	SortByIDField        SortByField = "id"
	SortByTimestampField SortByField = "timestamp"
	SortByStateField     SortByField = "state"
	SortByLevelField     SortByField = "level"
	SortBySessionField   SortByField = "session"
	SortByMessageField   SortByField = "message"
)

// IsValid checks if the sort by field is valid.
func (s SortByField) IsValid() bool {
	switch s {
	case SortByIDField, SortByTimestampField, SortByStateField,
		SortByLevelField, SortBySessionField, SortByMessageField:
		return true
	default:
		return false
	}
}

// String returns the string representation of the sort by field.
func (s SortByField) String() string {
	return string(s)
}

// SortOrder specifies the sort direction.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// IsValid checks if the sort order is valid.
func (s SortOrder) IsValid() bool {
	switch s {
	case SortOrderAsc, SortOrderDesc:
		return true
	default:
		return false
	}
}

// String returns the string representation of the sort order.
func (s SortOrder) String() string {
	return string(s)
}

// SortOptions holds sorting options for notifications.
type SortOptions struct {
	Field           SortByField
	Order           SortOrder
	CaseInsensitive bool
}

// SortNotifications sorts notifications based on the given options.
// Returns a new sorted slice without modifying the original.
func SortNotifications(notifs []Notification, opts SortOptions) []Notification {
	if len(notifs) == 0 {
		return notifs
	}

	if !opts.Field.IsValid() {
		opts.Field = SortByTimestampField
	}
	if !opts.Order.IsValid() {
		opts.Order = SortOrderDesc
	}

	// Create a copy to avoid modifying the original
	sorted := make([]Notification, len(notifs))
	copy(sorted, notifs)

	sort.SliceStable(sorted, func(i, j int) bool {
		var less bool

		switch opts.Field {
		case SortByIDField:
			less = sorted[i].ID < sorted[j].ID
		case SortByTimestampField:
			less = sorted[i].Timestamp < sorted[j].Timestamp
		case SortByStateField:
			less = sorted[i].State.String() < sorted[j].State.String()
		case SortByLevelField:
			less = sorted[i].Level.String() < sorted[j].Level.String()
		case SortBySessionField:
			less = sorted[i].Session < sorted[j].Session
		case SortByMessageField:
			msgI := sorted[i].Message
			msgJ := sorted[j].Message
			if opts.CaseInsensitive {
				less = strings.ToLower(msgI) < strings.ToLower(msgJ)
			} else {
				less = msgI < msgJ
			}
		default:
			less = sorted[i].Timestamp < sorted[j].Timestamp
		}

		if opts.Order == SortOrderDesc {
			return !less
		}
		return less
	})

	return sorted
}

// SortByID sorts notifications by ID.
func SortByID(notifs []Notification, order SortOrder) []Notification {
	return SortNotifications(notifs, SortOptions{Field: SortByIDField, Order: order})
}

// SortByTimestamp sorts notifications by timestamp.
func SortByTimestamp(notifs []Notification, order SortOrder) []Notification {
	return SortNotifications(notifs, SortOptions{Field: SortByTimestampField, Order: order})
}

// SortByState sorts notifications by state.
func SortByState(notifs []Notification, order SortOrder) []Notification {
	return SortNotifications(notifs, SortOptions{Field: SortByStateField, Order: order})
}

// SortByLevel sorts notifications by level.
func SortByLevel(notifs []Notification, order SortOrder) []Notification {
	return SortNotifications(notifs, SortOptions{Field: SortByLevelField, Order: order})
}

// SortBySession sorts notifications by session.
func SortBySession(notifs []Notification, order SortOrder) []Notification {
	return SortNotifications(notifs, SortOptions{Field: SortBySessionField, Order: order})
}

// SortByMessage sorts notifications by message.
func SortByMessage(notifs []Notification, order SortOrder, caseInsensitive bool) []Notification {
	return SortNotifications(notifs, SortOptions{Field: SortByMessageField, Order: order, CaseInsensitive: caseInsensitive})
}

// DefaultSortOptions returns the default sort options (timestamp descending).
func DefaultSortOptions() SortOptions {
	return SortOptions{
		Field:           SortByTimestampField,
		Order:           SortOrderDesc,
		CaseInsensitive: false,
	}
}

// ParseSortByField parses a string into a SortByField.
func ParseSortByField(field string) (SortByField, error) {
	f := SortByField(field)
	if !f.IsValid() {
		return "", fmt.Errorf("invalid sort field: %s", field)
	}
	return f, nil
}

// ParseSortOrder parses a string into a SortOrder.
func ParseSortOrder(order string) (SortOrder, error) {
	o := SortOrder(order)
	if !o.IsValid() {
		return "", fmt.Errorf("invalid sort order: %s", order)
	}
	return o, nil
}
