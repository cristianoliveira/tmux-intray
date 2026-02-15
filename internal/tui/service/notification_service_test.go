package service

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortNotificationsUnreadFirst(t *testing.T) {
	svc := NewNotificationService(nil, nil)

	notifications := []notification.Notification{
		{ID: 1, Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info", Message: "one", ReadTimestamp: "2024-01-01T11:00:00Z"},
		{ID: 2, Timestamp: "2024-01-02T10:00:00Z", State: "active", Level: "warning", Message: "two", ReadTimestamp: ""},
		{ID: 3, Timestamp: "2024-01-03T10:00:00Z", State: "dismissed", Level: "error", Message: "three", ReadTimestamp: ""},
		{ID: 4, Timestamp: "2024-01-04T10:00:00Z", State: "dismissed", Level: "critical", Message: "four", ReadTimestamp: "2024-01-04T11:00:00Z"},
	}

	sorted := svc.SortNotifications(notifications, "timestamp", "desc")

	assert.Equal(t, []int{3, 2, 4, 1}, []int{sorted[0].ID, sorted[1].ID, sorted[2].ID, sorted[3].ID})
}

func TestSortNotificationsUnreadFirstKeepsSortOrderWithinBuckets(t *testing.T) {
	svc := NewNotificationService(nil, nil)

	notifications := []notification.Notification{
		{ID: 10, Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info", Message: "ten", ReadTimestamp: ""},
		{ID: 11, Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "warning", Message: "eleven", ReadTimestamp: ""},
		{ID: 12, Timestamp: "2024-01-01T10:00:00Z", State: "dismissed", Level: "error", Message: "twelve", ReadTimestamp: "2024-01-01T11:00:00Z"},
		{ID: 13, Timestamp: "2024-01-01T10:00:00Z", State: "dismissed", Level: "critical", Message: "thirteen", ReadTimestamp: "2024-01-01T12:00:00Z"},
	}

	sorted := svc.SortNotifications(notifications, "id", "asc")

	assert.Equal(t, []int{10, 11, 12, 13}, []int{sorted[0].ID, sorted[1].ID, sorted[2].ID, sorted[3].ID})
}

func TestFilterByReadStatus(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "one", Timestamp: "2024-01-01T09:00:00Z", State: "active", Level: "info", ReadTimestamp: ""},
		{ID: 2, Message: "two", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info", ReadTimestamp: "2024-01-01T10:05:00Z"},
	}

	unread := svc.FilterByReadStatus(notifications, "unread")
	require.Len(t, unread, 1)
	assert.Equal(t, 1, unread[0].ID)

	read := svc.FilterByReadStatus(notifications, "read")
	require.Len(t, read, 1)
	assert.Equal(t, 2, read[0].ID)
}

func TestApplyFiltersAndSearchRespectsReadFilter(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "alpha", Timestamp: "2024-01-01T09:00:00Z", State: "active", Level: "info", ReadTimestamp: ""},
		{ID: 2, Message: "beta", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info", ReadTimestamp: "2024-01-01T10:05:00Z"},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch("", "", "", "", "", "", "unread", "timestamp", "asc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 1, filtered[0].ID)
}
