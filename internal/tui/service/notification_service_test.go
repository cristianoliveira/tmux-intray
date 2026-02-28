package service

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
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
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "unread", "timestamp", "asc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 1, filtered[0].ID)
}

// TestSearchFunction tests the Search method with token matching.
func TestSearchFunction(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "Error: file not found", Level: "error", Timestamp: "2024-01-01T10:00:00Z", State: "active"},
		{ID: 2, Message: "Warning: low memory", Level: "warning", Timestamp: "2024-01-01T10:00:00Z", State: "active"},
		{ID: 3, Message: "Error: connection failed", Level: "error", Timestamp: "2024-01-01T10:00:00Z", State: "active"},
		{ID: 4, Message: "Info: task completed", Level: "info", Timestamp: "2024-01-01T10:00:00Z", State: "active"},
	}
	svc.SetNotifications(notifications)
	t.Logf("notifications: %+v", notifications)

	// Search for "error" (case-insensitive by default)
	results := svc.Search(notifications, "error", false)
	t.Logf("results for 'error': %+v", results)
	require.Len(t, results, 2)
	assert.Equal(t, 1, results[0].ID)
	assert.Equal(t, 3, results[1].ID)

	// Search for "file not found"
	results = svc.Search(notifications, "file not found", false)
	t.Logf("results for 'file not found': %+v", results)
	require.Len(t, results, 1)
	assert.Equal(t, 1, results[0].ID)

	// Search for "Warning" with case-sensitive (should match exact case)
	results = svc.Search(notifications, "Warning", true)
	t.Logf("results for 'Warning': %+v", results)
	require.Len(t, results, 1)
	assert.Equal(t, 2, results[0].ID)

	// Empty query returns all
	results = svc.Search(notifications, "", false)
	t.Logf("results for empty query: %+v", results)
	require.Len(t, results, 4)
}

// TestApplyFiltersAndSearchLevelFilter tests filtering by level.
func TestApplyFiltersAndSearchLevelFilter(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "Error one", Level: "error", Timestamp: "2024-01-01T10:00:00Z", State: "active"},
		{ID: 2, Message: "Warning one", Level: "warning", Timestamp: "2024-01-01T10:00:00Z", State: "active"},
		{ID: 3, Message: "Error two", Level: "error", Timestamp: "2024-01-01T10:00:00Z", State: "active"},
		{ID: 4, Message: "Info one", Level: "info", Timestamp: "2024-01-01T10:00:00Z", State: "active"},
	}
	svc.SetNotifications(notifications)

	// Filter by level "error"
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "error", "", "", "", "", "timestamp", "asc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 2)
	assert.Equal(t, 1, filtered[0].ID)
	assert.Equal(t, 3, filtered[1].ID)

	// Filter by level "warning"
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "warning", "", "", "", "", "timestamp", "asc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 2, filtered[0].ID)

	// Filter by level "info"
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "info", "", "", "", "", "timestamp", "asc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 4, filtered[0].ID)
}

// TestApplyFiltersAndSearchSessionWindowPaneFilter tests filtering by session, window, pane.
func TestApplyFiltersAndSearchSessionWindowPaneFilter(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "Msg 1", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info"},
		{ID: 2, Message: "Msg 2", Session: "$1", Window: "@1", Pane: "%2", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info"},
		{ID: 3, Message: "Msg 3", Session: "$2", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info"},
		{ID: 4, Message: "Msg 4", Session: "$2", Window: "@2", Pane: "%1", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info"},
	}
	svc.SetNotifications(notifications)

	// Filter by session "$1"
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "$1", "", "", "", "timestamp", "asc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 2)
	assert.Equal(t, 1, filtered[0].ID)
	assert.Equal(t, 2, filtered[1].ID)

	// Filter by window "@1"
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "@1", "", "", "timestamp", "asc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 3)
	assert.Equal(t, 1, filtered[0].ID)
	assert.Equal(t, 2, filtered[1].ID)
	assert.Equal(t, 3, filtered[2].ID)

	// Filter by pane "%1"
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "%1", "", "timestamp", "asc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 3)
	assert.Equal(t, 1, filtered[0].ID)
	assert.Equal(t, 3, filtered[1].ID)
	assert.Equal(t, 4, filtered[2].ID)

	// Combined session and pane
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "$1", "", "%1", "", "timestamp", "asc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 1, filtered[0].ID)
}

func TestApplyFiltersAndSearchTabScopeUsesActiveNotificationsOnly(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "active newest", Timestamp: "2024-01-03T10:00:00Z", State: "active", Level: "info"},
		{ID: 2, Message: "dismissed newest", Timestamp: "2024-01-04T10:00:00Z", State: "dismissed", Level: "warning"},
		{ID: 3, Message: "active older", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "error"},
	}
	svc.SetNotifications(notifications)

	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	recents := svc.GetFilteredNotifications()
	require.Len(t, recents, 2)
	assert.Equal(t, []int{1, 3}, []int{recents[0].ID, recents[1].ID})

	svc.ApplyFiltersAndSearch(settings.TabAll, "", "", "", "", "", "", "", "timestamp", "desc")
	all := svc.GetFilteredNotifications()
	require.Len(t, all, 2)
	assert.Equal(t, []int{1, 3}, []int{all[0].ID, all[1].ID})

	svc.ApplyFiltersAndSearch(settings.TabAll, "", "dismissed", "", "", "", "", "", "timestamp", "desc")
	assert.Empty(t, svc.GetFilteredNotifications())
}

func TestApplyFiltersAndSearchTabAllPhaseConstraintActiveOnly(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "active", Timestamp: "2024-01-03T10:00:00Z", State: "active", Level: "info"},
		{ID: 2, Message: "dismissed", Timestamp: "2024-01-04T10:00:00Z", State: "dismissed", Level: "warning"},
	}
	svc.SetNotifications(notifications)

	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	recents := svc.GetFilteredNotifications()
	require.Len(t, recents, 1)
	assert.Equal(t, 1, recents[0].ID)

	svc.ApplyFiltersAndSearch(settings.TabAll, "", "", "", "", "", "", "", "timestamp", "desc")
	all := svc.GetFilteredNotifications()
	require.Len(t, all, 1)
	assert.Equal(t, 1, all[0].ID)
}

func TestApplyFiltersAndSearchTabScopeSearchesWithinTabDataset(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "active alpha", Timestamp: "2024-01-03T10:00:00Z", State: "active", Level: "info"},
		{ID: 2, Message: "dismissed alpha", Timestamp: "2024-01-04T10:00:00Z", State: "dismissed", Level: "warning"},
	}
	svc.SetNotifications(notifications)

	svc.ApplyFiltersAndSearch(settings.TabAll, "alpha", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 1, filtered[0].ID)
}
