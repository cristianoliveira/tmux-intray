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

func TestApplyFiltersAndSearchRecentsForcesUnreadView(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "unread", Timestamp: "2024-01-01T09:00:00Z", State: "active", Level: "info", Session: "$1", Window: "@1", Pane: "%1", ReadTimestamp: ""},
		{ID: 2, Message: "read", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info", Session: "$1", Window: "@1", Pane: "%1", ReadTimestamp: "2024-01-01T10:05:00Z"},
	}

	svc.SetNotifications(notifications)

	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "read", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 1, filtered[0].ID)

	svc.ApplyFiltersAndSearch(settings.TabAll, "", "", "", "", "", "", "read", "timestamp", "desc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 2, filtered[0].ID)
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
		{ID: 1, Message: "Error one", Level: "error", Timestamp: "2024-01-01T10:00:00Z", State: "active", Session: "$1", Window: "@1", Pane: "%1"},
		{ID: 2, Message: "Warning one", Level: "warning", Timestamp: "2024-01-01T10:00:00Z", State: "active", Session: "$1", Window: "@1", Pane: "%2"},
		{ID: 3, Message: "Error two", Level: "error", Timestamp: "2024-01-01T10:00:00Z", State: "active", Session: "$1", Window: "@1", Pane: "%3"},
		{ID: 4, Message: "Info one", Level: "info", Timestamp: "2024-01-01T10:00:00Z", State: "active", Session: "$1", Window: "@1", Pane: "%4"},
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

func TestApplyFiltersAndSearchActiveOnlyAcrossTabs(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "active 1", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info"},
		{ID: 2, Message: "dismissed", Timestamp: "2024-01-02T10:00:00Z", State: "dismissed", Level: "info"},
		{ID: 3, Message: "active 2", Timestamp: "2024-01-03T10:00:00Z", State: "active", Level: "warning"},
	}
	svc.SetNotifications(notifications)

	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 2)
	assert.Equal(t, []int{3, 1}, []int{filtered[0].ID, filtered[1].ID})

	svc.ApplyFiltersAndSearch(settings.TabAll, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 2)
	assert.Equal(t, []int{3, 1}, []int{filtered[0].ID, filtered[1].ID})

	svc.ApplyFiltersAndSearch(settings.TabAll, "", "dismissed", "", "", "", "", "", "timestamp", "desc")
	assert.Empty(t, svc.GetFilteredNotifications())
}

func TestApplyFiltersAndSearchRecentsUsesLimitedDataset(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := make([]notification.Notification, 0, 25)
	for i := 1; i <= 25; i++ {
		session := "$1"
		window := "@1"
		pane := "%1"
		if i > 13 {
			session = "$2"
			window = "@2"
			pane = "%2"
		}

		notifications = append(notifications, notification.Notification{
			ID:        i,
			Message:   "msg",
			Timestamp: "2024-01-01T10:00:00Z",
			State:     "active",
			Level:     "info",
			Session:   session,
			Window:    window,
			Pane:      pane,
		})
	}
	svc.SetNotifications(notifications)

	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "id", "desc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 6)
	assert.Equal(t, 25, filtered[0].ID)
	assert.Equal(t, 24, filtered[1].ID)
	assert.Equal(t, 23, filtered[2].ID)
	assert.Equal(t, 13, filtered[3].ID)
	assert.Equal(t, 12, filtered[4].ID)
	assert.Equal(t, 11, filtered[5].ID)

	svc.ApplyFiltersAndSearch(settings.TabAll, "", "", "", "", "", "", "", "id", "desc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 25)
	assert.Equal(t, 25, filtered[0].ID)
	assert.Equal(t, 1, filtered[len(filtered)-1].ID)
}

func TestApplyFiltersAndSearchScopesFilteringToSelectedTabDataset(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := make([]notification.Notification, 0, 25)
	for i := 1; i <= 25; i++ {
		level := "info"
		if i == 2 {
			level = "error"
		}
		notifications = append(notifications, notification.Notification{
			ID:        i,
			Message:   "msg",
			Timestamp: "2024-01-01T10:00:00Z",
			State:     "active",
			Level:     level,
		})
	}
	svc.SetNotifications(notifications)

	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "error", "", "", "", "", "id", "desc")
	assert.Empty(t, svc.GetFilteredNotifications())

	svc.ApplyFiltersAndSearch(settings.TabAll, "", "", "error", "", "", "", "", "id", "desc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 2, filtered[0].ID)
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

// TestRecentsPerSourceLimitWithLevelFilter verifies that per-source limit is applied after level filter
func TestRecentsPerSourceLimitWithLevelFilter(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "error 1", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "error"},
		{ID: 2, Message: "error 2", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:01:00Z", State: "active", Level: "error"},
		{ID: 3, Message: "error 3", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:02:00Z", State: "active", Level: "error"},
		{ID: 4, Message: "error 4", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:03:00Z", State: "active", Level: "error"},
		{ID: 5, Message: "warning 1", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:04:00Z", State: "active", Level: "warning"},
	}
	svc.SetNotifications(notifications)

	// Apply level filter for errors in Recents tab
	// Should get max 3 per source, so only 3 out of 4 errors
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "error", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 3)
	// Should be the last 3 in timestamp order (highest IDs with desc sort)
	assert.Equal(t, 4, filtered[0].ID)
	assert.Equal(t, 3, filtered[1].ID)
	assert.Equal(t, 2, filtered[2].ID)
}

// TestRecentsPerSourceLimitWithSessionFilter verifies that per-source limit is applied after session filter
func TestRecentsPerSourceLimitWithSessionFilter(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "msg 1", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info"},
		{ID: 2, Message: "msg 2", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:01:00Z", State: "active", Level: "info"},
		{ID: 3, Message: "msg 3", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:02:00Z", State: "active", Level: "info"},
		{ID: 4, Message: "msg 4", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:03:00Z", State: "active", Level: "info"},
		{ID: 5, Message: "msg 5", Session: "$2", Window: "@2", Pane: "%2", Timestamp: "2024-01-01T10:04:00Z", State: "active", Level: "info"},
	}
	svc.SetNotifications(notifications)

	// Apply session filter for $1 in Recents tab
	// Should get all 4 from $1, but limited to 3 per source
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "$1", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 3)
	assert.Equal(t, 4, filtered[0].ID)
	assert.Equal(t, 3, filtered[1].ID)
	assert.Equal(t, 2, filtered[2].ID)
}

// TestRecentsPerSourceLimitWithMultipleSources verifies per-source limit across multiple sources
func TestRecentsPerSourceLimitWithMultipleSources(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		// Source 1: 4 messages
		{ID: 1, Message: "s1 msg 1", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info"},
		{ID: 2, Message: "s1 msg 2", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:01:00Z", State: "active", Level: "info"},
		{ID: 3, Message: "s1 msg 3", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:02:00Z", State: "active", Level: "info"},
		{ID: 4, Message: "s1 msg 4", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:03:00Z", State: "active", Level: "info"},
		// Source 2: 4 messages
		{ID: 5, Message: "s2 msg 1", Session: "$2", Window: "@2", Pane: "%2", Timestamp: "2024-01-01T10:04:00Z", State: "active", Level: "info"},
		{ID: 6, Message: "s2 msg 2", Session: "$2", Window: "@2", Pane: "%2", Timestamp: "2024-01-01T10:05:00Z", State: "active", Level: "info"},
		{ID: 7, Message: "s2 msg 3", Session: "$2", Window: "@2", Pane: "%2", Timestamp: "2024-01-01T10:06:00Z", State: "active", Level: "info"},
		{ID: 8, Message: "s2 msg 4", Session: "$2", Window: "@2", Pane: "%2", Timestamp: "2024-01-01T10:07:00Z", State: "active", Level: "info"},
	}
	svc.SetNotifications(notifications)

	// In Recents tab, should get 3 per source (max 6 total)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 6)
	// Should have 3 from source 1 and 3 from source 2 (in desc timestamp order)
	assert.Equal(t, 8, filtered[0].ID) // s2 msg 4
	assert.Equal(t, 7, filtered[1].ID) // s2 msg 3
	assert.Equal(t, 6, filtered[2].ID) // s2 msg 2
	assert.Equal(t, 4, filtered[3].ID) // s1 msg 4
	assert.Equal(t, 3, filtered[4].ID) // s1 msg 3
	assert.Equal(t, 2, filtered[5].ID) // s1 msg 2
}

// TestRecentsPerSourceLimitWithSearchFilter verifies per-source limit is applied after search
func TestRecentsPerSourceLimitWithSearchFilter(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "alpha error", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:00:00Z", State: "active", Level: "info"},
		{ID: 2, Message: "beta error", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:01:00Z", State: "active", Level: "info"},
		{ID: 3, Message: "gamma error", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:02:00Z", State: "active", Level: "info"},
		{ID: 4, Message: "delta error", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:03:00Z", State: "active", Level: "info"},
		{ID: 5, Message: "epsilon alpha", Session: "$1", Window: "@1", Pane: "%1", Timestamp: "2024-01-01T10:04:00Z", State: "active", Level: "info"},
	}
	svc.SetNotifications(notifications)

	// Search for "error" in Recents tab - should get max 3 results per source
	svc.ApplyFiltersAndSearch(settings.TabRecents, "error", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 3)
	assert.Equal(t, 4, filtered[0].ID)
	assert.Equal(t, 3, filtered[1].ID)
	assert.Equal(t, 2, filtered[2].ID)
}
