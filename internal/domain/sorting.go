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
	SortByIDField         SortByField = "id"
	SortByTimestampField  SortByField = "timestamp"
	SortByStateField      SortByField = "state"
	SortByLevelField      SortByField = "level"
	SortBySessionField    SortByField = "session"
	SortByMessageField    SortByField = "message"
	SortByReadStatusField SortByField = "read_status"
)

// IsValid checks if the sort by field is valid.
func (s SortByField) IsValid() bool {
	switch s {
	case SortByIDField, SortByTimestampField, SortByStateField,
		SortByLevelField, SortBySessionField, SortByMessageField,
		SortByReadStatusField:
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

	opts = normalizeSortOptions(opts)

	// Create a copy to avoid modifying the original
	sorted := make([]Notification, len(notifs))
	copy(sorted, notifs)

	sort.SliceStable(sorted, func(i, j int) bool {
		return compareNotifications(sorted[i], sorted[j], opts) < 0
	})

	return sorted
}

// normalizeSortOptions normalizes sort options by setting defaults.
func normalizeSortOptions(opts SortOptions) SortOptions {
	if !opts.Field.IsValid() {
		opts.Field = SortByTimestampField
	}
	if !opts.Order.IsValid() {
		opts.Order = SortOrderDesc
	}
	return opts
}

// compareNotifications compares two notifications based on the sort options.
// Returns -1 if i < j, 1 if i > j, 0 if equal.
func compareNotifications(i, j Notification, opts SortOptions) int {
	// Read status field handles order directly in compareByField
	if opts.Field == SortByReadStatusField {
		less := compareByField(i, j, opts)
		if less {
			return -1
		}
		return 1
	}

	less := compareByField(i, j, opts)

	// Apply order (flip for descending)
	if opts.Order == SortOrderDesc {
		if less {
			return 1
		}
		return -1
	}

	if less {
		return -1
	}
	return 1
}

// compareByField compares two notifications by the specified field.
func compareByField(i, j Notification, opts SortOptions) bool {
	switch opts.Field {
	case SortByIDField:
		return i.ID < j.ID
	case SortByTimestampField:
		return i.Timestamp < j.Timestamp
	case SortByStateField:
		return i.State.String() < j.State.String()
	case SortByLevelField:
		return i.Level.String() < j.Level.String()
	case SortBySessionField:
		return i.Session < j.Session
	case SortByMessageField:
		msgI := i.Message
		msgJ := j.Message
		if opts.CaseInsensitive {
			return strings.ToLower(msgI) < strings.ToLower(msgJ)
		}
		return msgI < msgJ
	case SortByReadStatusField:
		// For read status, handle order directly in the comparison
		iRead := i.IsRead()
		jRead := j.IsRead()
		if opts.Order == SortOrderAsc {
			// Ascending: unread first, then read
			return !iRead && jRead
		}
		// Descending: read first, then unread
		return iRead && !jRead
	default:
		return i.Timestamp < j.Timestamp
	}
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

// SortByReadStatus sorts notifications by read status (unread first).
func SortByReadStatus(notifs []Notification, order SortOrder) []Notification {
	return SortNotifications(notifs, SortOptions{Field: SortByReadStatusField, Order: order})
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

// SortWithUnreadFirst sorts notifications with unread messages first, then applies the given sort options.
// This function first partitions notifications into unread and read groups, then sorts each group
// according to the provided options before recombining them.
// Returns a new sorted slice without modifying the original.
func SortWithUnreadFirst(notifs []Notification, opts SortOptions) []Notification {
	if len(notifs) == 0 {
		return notifs
	}

	opts = normalizeSortOptions(opts)

	// Create a copy to avoid modifying the original
	sorted := make([]Notification, len(notifs))
	copy(sorted, notifs)

	// Partition into unread and read notifications
	unread := make([]Notification, 0, len(sorted))
	read := make([]Notification, 0, len(sorted))

	for _, n := range sorted {
		if n.IsRead() {
			read = append(read, n)
		} else {
			unread = append(unread, n)
		}
	}

	// Sort each partition independently
	sort.SliceStable(unread, func(i, j int) bool {
		return compareNotifications(unread[i], unread[j], opts) < 0
	})

	sort.SliceStable(read, func(i, j int) bool {
		return compareNotifications(read[i], read[j], opts) < 0
	})

	// Recombine: unread first, then read
	result := make([]Notification, 0, len(sorted))
	result = append(result, unread...)
	result = append(result, read...)

	return result
}
