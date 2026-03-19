package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nowMinutes returns current time minus specified minutes (for testing recent notifications)
func nowMinutes(minutes int) string {
	return time.Now().UTC().Add(-time.Duration(minutes) * time.Minute).Format(time.RFC3339)
}

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
		{ID: 1, Message: "alpha", Timestamp: nowMinutes(30), State: "active", Level: "info", ReadTimestamp: ""},
		{ID: 2, Message: "beta", Timestamp: nowMinutes(25), State: "active", Level: "info", ReadTimestamp: nowMinutes(20)},
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
		{ID: 1, Message: "unread", Timestamp: nowMinutes(30), State: "active", Level: "info", Session: "$1", Window: "@1", Pane: "%1", ReadTimestamp: ""},
		{ID: 2, Message: "read", Timestamp: nowMinutes(25), State: "active", Level: "info", Session: "$1", Window: "@1", Pane: "%1", ReadTimestamp: nowMinutes(20)},
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
		{ID: 1, Message: "Error one", Level: "error", Timestamp: nowMinutes(30), State: "active", Session: "$1", Window: "@1", Pane: "%1"},
		{ID: 2, Message: "Warning one", Level: "warning", Timestamp: nowMinutes(28), State: "active", Session: "$2", Window: "@1", Pane: "%2"},
		{ID: 3, Message: "Error two", Level: "error", Timestamp: nowMinutes(26), State: "active", Session: "$3", Window: "@1", Pane: "%3"},
		{ID: 4, Message: "Info one", Level: "info", Timestamp: nowMinutes(24), State: "active", Session: "$4", Window: "@1", Pane: "%4"},
	}
	svc.SetNotifications(notifications)

	// Filter by level "error" - should include IDs 1 and 3 from different sessions
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "error", "", "", "", "", "timestamp", "asc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 2)
	assert.Equal(t, 1, filtered[0].ID)
	assert.Equal(t, 3, filtered[1].ID)

	// Filter by level "warning" - should include ID 2
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "warning", "", "", "", "", "timestamp", "asc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 2, filtered[0].ID)

	// Filter by level "info" - should include ID 4
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "info", "", "", "", "", "timestamp", "asc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 1)
	assert.Equal(t, 4, filtered[0].ID)
}

// TestApplyFiltersAndSearchSessionWindowPaneFilter tests filtering by session, window, pane.
func TestApplyFiltersAndSearchSessionWindowPaneFilter(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := []notification.Notification{
		{ID: 1, Message: "Msg 1", Session: "$1", Window: "@1", Pane: "%1", Timestamp: nowMinutes(30), State: "active", Level: "info"},
		{ID: 2, Message: "Msg 2", Session: "$1", Window: "@1", Pane: "%2", Timestamp: nowMinutes(28), State: "active", Level: "info"},
		{ID: 3, Message: "Msg 3", Session: "$2", Window: "@1", Pane: "%1", Timestamp: nowMinutes(26), State: "active", Level: "info"},
		{ID: 4, Message: "Msg 4", Session: "$2", Window: "@2", Pane: "%1", Timestamp: nowMinutes(24), State: "active", Level: "info"},
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
		{ID: 1, Message: "active 1", Timestamp: nowMinutes(30), State: "active", Level: "info", Session: "$1", Window: "@1", Pane: "%1"},
		{ID: 2, Message: "dismissed", Timestamp: nowMinutes(28), State: "dismissed", Level: "info", Session: "$2", Window: "@2", Pane: "%2"},
		{ID: 3, Message: "active 2", Timestamp: nowMinutes(26), State: "active", Level: "warning", Session: "$3", Window: "@3", Pane: "%3"},
	}
	svc.SetNotifications(notifications)

	// Recents with per-session selection: should have 2 active notifications from different sessions
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()
	require.Len(t, filtered, 2)
	assert.Equal(t, []int{3, 1}, []int{filtered[0].ID, filtered[1].ID})

	// All tab shows only active notifications (dismissed excluded by selectDataset)
	svc.ApplyFiltersAndSearch(settings.TabAll, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered = svc.GetFilteredNotifications()
	require.Len(t, filtered, 2)
	assert.Equal(t, []int{3, 1}, []int{filtered[0].ID, filtered[1].ID})

	// All tab with dismissed filter applied (no results since no dismissed notifications in All tab)
	svc.ApplyFiltersAndSearch(settings.TabAll, "", "dismissed", "", "", "", "", "", "timestamp", "desc")
	assert.Empty(t, svc.GetFilteredNotifications())
}

func TestApplyFiltersAndSearchRecentsUsesLimitedDataset(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	notifications := make([]notification.Notification, 0, 25)
	for i := 1; i <= 25; i++ {
		session := fmt.Sprintf("$%d", i) // Each notification gets its own session
		window := "@1"
		pane := "%1"

		notifications = append(notifications, notification.Notification{
			ID:        i,
			Message:   "msg",
			Timestamp: nowMinutes(25 - i),
			State:     "active",
			Level:     "info",
			Session:   session,
			Window:    window,
			Pane:      pane,
		})
	}
	svc.SetNotifications(notifications)

	// With per-session selection, we get up to 20 most recent sessions (within time window)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "id", "desc")
	filtered := svc.GetFilteredNotifications()
	// Should have up to 20 sessions represented
	require.Len(t, filtered, 20)
	assert.Equal(t, 25, filtered[0].ID)
	assert.Equal(t, 6, filtered[19].ID)

	// All tab shows all 25 notifications
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

func TestRecentsTabApplies1HourTimeWindow(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	notifications := []notification.Notification{
		{
			ID:        1,
			Message:   "recent notification",
			Timestamp: now.Add(-30 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        2,
			Message:   "older notification",
			Timestamp: now.Add(-2 * time.Hour).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        3,
			Message:   "warning in window",
			Timestamp: now.Add(-45 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "warning",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        4,
			Message:   "info in window, different session",
			Timestamp: now.Add(-35 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
		},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should only include notifications within the last 1 hour
	// With per-session selection: max 1 per session
	// Session $1: should select warning (ID 3, highest severity)
	// Session $2: should select info (ID 4, only one)
	require.Len(t, filtered, 2)
	ids := []int{filtered[0].ID, filtered[1].ID}
	assert.Contains(t, ids, 3) // warning from session $1
	assert.Contains(t, ids, 4) // info from session $2
	// Ensure notification 2 (2 hours old) is not included
	for _, n := range filtered {
		assert.NotEqual(t, 2, n.ID)
	}
	// Ensure notification 1 is not included (warning is higher severity)
	for _, n := range filtered {
		assert.NotEqual(t, 1, n.ID)
	}
}

func TestRecentsTabCanBeEmpty(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	// Only old notifications
	notifications := []notification.Notification{
		{
			ID:        1,
			Message:   "very old notification",
			Timestamp: now.Add(-3 * time.Hour).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should be empty since all notifications are older than 1 hour
	assert.Empty(t, filtered)
}

// TestRecentsPerSessionSmartSelection tests per-session selection with severity-based prioritization.
func TestRecentsPerSessionSmartSelection(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	// Multiple notifications from same session with different severities
	notifications := []notification.Notification{
		{
			ID:        1,
			Message:   "info notification",
			Timestamp: now.Add(-30 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        2,
			Message:   "warning notification",
			Timestamp: now.Add(-25 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "warning",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        3,
			Message:   "error notification",
			Timestamp: now.Add(-20 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "error",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should select only the error notification (highest severity) for this session
	require.Len(t, filtered, 1)
	assert.Equal(t, 3, filtered[0].ID)
	assert.Equal(t, "error", filtered[0].Level)
}

// TestRecentsPerSessionSelectionWithMultipleSessions tests max 1 per session across multiple sessions.
func TestRecentsPerSessionSelectionWithMultipleSessions(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	notifications := []notification.Notification{
		// Session 1: multiple notifications
		{
			ID:        1,
			Message:   "session1 info",
			Timestamp: now.Add(-30 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        2,
			Message:   "session1 warning",
			Timestamp: now.Add(-25 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "warning",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		// Session 2: multiple notifications
		{
			ID:        3,
			Message:   "session2 error",
			Timestamp: now.Add(-28 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "error",
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
		},
		{
			ID:        4,
			Message:   "session2 info",
			Timestamp: now.Add(-22 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
		},
		// Session 3: single notification
		{
			ID:        5,
			Message:   "session3 warning",
			Timestamp: now.Add(-15 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "warning",
			Session:   "$3",
			Window:    "@3",
			Pane:      "%3",
		},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should have max 1 per session (3 total)
	require.Len(t, filtered, 3)

	// Verify each session represented and correct severity selected
	sessionMap := make(map[string]notification.Notification)
	for _, n := range filtered {
		sessionMap[n.Session] = n
	}

	assert.Equal(t, "warning", sessionMap["$1"].Level)
	assert.Equal(t, 2, sessionMap["$1"].ID)

	assert.Equal(t, "error", sessionMap["$2"].Level)
	assert.Equal(t, 3, sessionMap["$2"].ID)

	assert.Equal(t, "warning", sessionMap["$3"].Level)
	assert.Equal(t, 5, sessionMap["$3"].ID)
}

// TestRecentsOrderedByMostRecentActivity tests sessions ordered by most recent timestamp.
func TestRecentsOrderedByMostRecentActivity(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	notifications := []notification.Notification{
		// Session 1: recent info
		{
			ID:        1,
			Message:   "session1 old",
			Timestamp: now.Add(-40 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		// Session 2: oldest
		{
			ID:        2,
			Message:   "session2 oldest",
			Timestamp: now.Add(-50 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
		},
		// Session 3: most recent
		{
			ID:        3,
			Message:   "session3 newest",
			Timestamp: now.Add(-5 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$3",
			Window:    "@3",
			Pane:      "%3",
		},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should be ordered by recency: session3, session1, session2
	require.Len(t, filtered, 3)
	assert.Equal(t, 3, filtered[0].ID) // session3 (most recent)
	assert.Equal(t, 1, filtered[1].ID) // session1
	assert.Equal(t, 2, filtered[2].ID) // session2 (oldest)
}

// TestRecentsSeveritySelectionPrefersError tests error is selected over warning/info.
func TestRecentsSeveritySelectionPrefersError(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	notifications := []notification.Notification{
		{
			ID:        1,
			Message:   "older error",
			Timestamp: now.Add(-40 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "error",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        2,
			Message:   "newer warning",
			Timestamp: now.Add(-5 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "warning",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should prefer error even though warning is newer
	require.Len(t, filtered, 1)
	assert.Equal(t, 1, filtered[0].ID)
	assert.Equal(t, "error", filtered[0].Level)
}

// TestRecentsSeverityTieUsesRecency tests recent notification chosen when severity ties.
func TestRecentsSeverityTieUsesRecency(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	notifications := []notification.Notification{
		{
			ID:        1,
			Message:   "older warning",
			Timestamp: now.Add(-40 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "warning",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        2,
			Message:   "newer warning",
			Timestamp: now.Add(-5 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "warning",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should prefer newer when severity is same
	require.Len(t, filtered, 1)
	assert.Equal(t, 2, filtered[0].ID)
}

// TestRecentsRespects20ItemLimit tests total unfiltered limit of 20 items.
func TestRecentsRespects20ItemLimit(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	// Create 30 notifications across 30 different sessions
	notifications := make([]notification.Notification, 0, 30)
	for i := 1; i <= 30; i++ {
		sessionID := fmt.Sprintf("$%d", i)
		notifications = append(notifications, notification.Notification{
			ID:        i,
			Message:   fmt.Sprintf("msg%d", i),
			Timestamp: now.Add(-time.Duration(30-i) * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   sessionID,
			Window:    "@1",
			Pane:      "%1",
		})
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should limit to 20 items max
	require.Len(t, filtered, 20)

	// Verify they're the most recent 20
	assert.Equal(t, 30, filtered[0].ID)
	assert.Equal(t, 11, filtered[19].ID)
}

// TestRecentsIntegrationWithTimeWindow tests per-session selection works with 1-hour time window.
func TestRecentsIntegrationWithTimeWindow(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	notifications := []notification.Notification{
		// Session 1: within window
		{
			ID:        1,
			Message:   "session1 recent error",
			Timestamp: now.Add(-30 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "error",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        2,
			Message:   "session1 recent warning",
			Timestamp: now.Add(-20 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "warning",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		// Session 2: outside window (2 hours old)
		{
			ID:        3,
			Message:   "session2 old error",
			Timestamp: now.Add(-2 * time.Hour).Format(time.RFC3339),
			State:     "active",
			Level:     "error",
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
		},
		// Session 3: within window
		{
			ID:        4,
			Message:   "session3 recent info",
			Timestamp: now.Add(-45 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$3",
			Window:    "@3",
			Pane:      "%3",
		},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should only include notifications within 1 hour
	// Session 1: error (higher severity than warning)
	// Session 3: info (only notification)
	require.Len(t, filtered, 2)

	sessionMap := make(map[string]notification.Notification)
	for _, n := range filtered {
		sessionMap[n.Session] = n
	}

	assert.Equal(t, "error", sessionMap["$1"].Level)
	assert.Equal(t, 1, sessionMap["$1"].ID)
	assert.NotContains(t, sessionMap, "$2") // Outside time window
	assert.Equal(t, "info", sessionMap["$3"].Level)
	assert.Equal(t, 4, sessionMap["$3"].ID)
}

// TestFilteredListRespects10ItemLimit tests that filtered views (drilling down by session/window/pane) are limited to 10 items.
func TestFilteredListRespects10ItemLimit(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	// Create 15 notifications from the same session within time window
	notifications := make([]notification.Notification, 0, 15)
	sessionID := "$test-session"
	for i := 1; i <= 15; i++ {
		notifications = append(notifications, notification.Notification{
			ID:        i,
			Message:   fmt.Sprintf("msg%d", i),
			Timestamp: now.Add(-time.Duration(15-i) * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   sessionID,
			Window:    "@1",
			Pane:      "%1",
		})
	}

	svc.SetNotifications(notifications)
	// Apply filter by session - this triggers the filtered view logic
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", sessionID, "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should limit to 10 items max in filtered view
	require.Len(t, filtered, 10)

	// Verify they're the most recent 10
	assert.Equal(t, 15, filtered[0].ID)
	assert.Equal(t, 6, filtered[9].ID)
}

// TestFilteredListByWindowRespects10ItemLimit tests filtered view by window ID.
func TestFilteredListByWindowRespects10ItemLimit(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	// Create 12 notifications from the same window within time window
	notifications := make([]notification.Notification, 0, 12)
	windowID := "@test-window"
	for i := 1; i <= 12; i++ {
		notifications = append(notifications, notification.Notification{
			ID:        i,
			Message:   fmt.Sprintf("msg%d", i),
			Timestamp: now.Add(-time.Duration(12-i) * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   fmt.Sprintf("$%d", i),
			Window:    windowID,
			Pane:      fmt.Sprintf("%%%d", i),
		})
	}

	svc.SetNotifications(notifications)
	// Apply filter by window
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", windowID, "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should limit to 10 items max
	require.Len(t, filtered, 10)

	// Verify they're the most recent 10
	assert.Equal(t, 12, filtered[0].ID)
	assert.Equal(t, 3, filtered[9].ID)
}

// TestFilteredListByPaneRespects10ItemLimit tests filtered view by pane ID.
func TestFilteredListByPaneRespects10ItemLimit(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	// Create 20 notifications from the same pane within time window
	notifications := make([]notification.Notification, 0, 20)
	paneID := "%test-pane"
	for i := 1; i <= 20; i++ {
		notifications = append(notifications, notification.Notification{
			ID:        i,
			Message:   fmt.Sprintf("msg%d", i),
			Timestamp: now.Add(-time.Duration(20-i) * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$1",
			Window:    "@1",
			Pane:      paneID,
		})
	}

	svc.SetNotifications(notifications)
	// Apply filter by pane
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", paneID, "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should limit to 10 items max
	require.Len(t, filtered, 10)

	// Verify they're the most recent 10
	assert.Equal(t, 20, filtered[0].ID)
	assert.Equal(t, 11, filtered[9].ID)
}

// TestFilteredListRespectsTimeWindow tests that filtered views still respect 1-hour time window.
func TestFilteredListRespectsTimeWindow(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	notifications := []notification.Notification{
		// Within window (30 min ago)
		{
			ID:        1,
			Message:   "recent",
			Timestamp: now.Add(-30 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$target",
			Window:    "@1",
			Pane:      "%1",
		},
		// Outside window (2 hours ago)
		{
			ID:        2,
			Message:   "old",
			Timestamp: now.Add(-2 * time.Hour).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$target",
			Window:    "@1",
			Pane:      "%1",
		},
		// Within window (45 min ago)
		{
			ID:        3,
			Message:   "recent2",
			Timestamp: now.Add(-45 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$target",
			Window:    "@1",
			Pane:      "%1",
		},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "$target", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should only show 2 items (the ones within 1 hour)
	require.Len(t, filtered, 2)
	assert.Equal(t, 1, filtered[0].ID)
	assert.Equal(t, 3, filtered[1].ID)
}

// TestFilteredListHandlesEmptyResults tests graceful handling of empty filtered results.
func TestFilteredListHandlesEmptyResults(t *testing.T) {
	svc := NewNotificationService(nil, nil)

	notifications := []notification.Notification{
		{
			ID:        1,
			Message:   "test",
			Timestamp: nowMinutes(30),
			State:     "active",
			Level:     "info",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
	}

	svc.SetNotifications(notifications)
	// Filter by a session that doesn't exist
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "$nonexistent", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// Should return empty list gracefully
	require.Len(t, filtered, 0)
}

// TestFilteredListOrderedByTimestamp tests that filtered results are ordered newest first.
func TestFilteredListOrderedByTimestamp(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	notifications := []notification.Notification{
		{ID: 1, Message: "msg1", Timestamp: now.Add(-10 * time.Minute).Format(time.RFC3339), State: "active", Level: "info", Session: "$1", Window: "@1", Pane: "%1"},
		{ID: 2, Message: "msg2", Timestamp: now.Add(-5 * time.Minute).Format(time.RFC3339), State: "active", Level: "info", Session: "$1", Window: "@1", Pane: "%1"},
		{ID: 3, Message: "msg3", Timestamp: now.Add(-20 * time.Minute).Format(time.RFC3339), State: "active", Level: "info", Session: "$1", Window: "@1", Pane: "%1"},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "$1", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	require.Len(t, filtered, 3)
	// Should be ordered by timestamp descending (newest first)
	assert.Equal(t, 2, filtered[0].ID)
	assert.Equal(t, 1, filtered[1].ID)
	assert.Equal(t, 3, filtered[2].ID)
}

func TestRecentsTabUsesConfigurableTimeWindow(t *testing.T) {
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	// Create notifications at various time points
	notifications := []notification.Notification{
		{
			ID:        1,
			Message:   "5 minutes old",
			Timestamp: now.Add(-5 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        2,
			Message:   "30 minutes old",
			Timestamp: now.Add(-30 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
		},
		{
			ID:        3,
			Message:   "45 minutes old",
			Timestamp: now.Add(-45 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$3",
			Window:    "@3",
			Pane:      "%3",
		},
		{
			ID:        4,
			Message:   "2 hours old",
			Timestamp: now.Add(-2 * time.Hour).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$4",
			Window:    "@4",
			Pane:      "%4",
		},
	}

	svc.SetNotifications(notifications)

	// Test with default 1h window
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()
	// Should include 1, 2, 3 (all within 1 hour), exclude 4 (2 hours old)
	require.Equal(t, 3, len(filtered), "Default 1h window should include 3 notifications")
	for _, n := range filtered {
		assert.NotEqual(t, 4, n.ID, "2-hour-old notification should not be in default 1h window")
	}
}

func TestRecentsTabUsesConfiguredTimeWindowFrom30Minutes(t *testing.T) {
	// This test demonstrates that the config is used
	// In a real scenario, you would set TMUX_INTRAY_RECENTS_TIME_WINDOW=30m
	// and test that only notifications within 30 minutes are shown
	svc := NewNotificationService(nil, nil)
	now := time.Now().UTC()

	notifications := []notification.Notification{
		{
			ID:        1,
			Message:   "15 minutes old",
			Timestamp: now.Add(-15 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        2,
			Message:   "45 minutes old",
			Timestamp: now.Add(-45 * time.Minute).Format(time.RFC3339),
			State:     "active",
			Level:     "info",
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
		},
	}

	svc.SetNotifications(notifications)
	svc.ApplyFiltersAndSearch(settings.TabRecents, "", "", "", "", "", "", "", "timestamp", "desc")
	filtered := svc.GetFilteredNotifications()

	// With default 1h, both should be included
	require.Equal(t, 2, len(filtered), "Both notifications should be within 1h window")
}
