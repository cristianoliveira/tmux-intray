package state

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTreeGroupsAndSorts(t *testing.T) {
	notifications := []notification.Notification{
		{
			ID:        1,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Beta issue",
			Timestamp: "2024-01-02T10:00:00Z",
		},
		{
			ID:        2,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Alpha issue",
			Timestamp: "2024-01-03T10:00:00Z",
		},
		{
			ID:        3,
			Session:   "$1",
			Window:    "@2",
			Pane:      "%2",
			Message:   "Gamma issue",
			Timestamp: "2024-01-01T10:00:00Z",
		},
		{
			ID:        4,
			Session:   "$2",
			Window:    "@1",
			Pane:      "%3",
			Message:   "Delta issue",
			Timestamp: "2024-01-04T10:00:00Z",
		},
	}

	root := BuildTree(notifications, settings.GroupByPane)

	require.NotNil(t, root)
	assert.Equal(t, NodeKindRoot, root.Kind)
	assert.Equal(t, 4, root.Count)
	require.NotNil(t, root.LatestEvent)
	assert.Equal(t, 4, root.LatestEvent.ID)

	require.Len(t, root.Children, 2)
	assert.Equal(t, "$1", root.Children[0].Title)
	assert.Equal(t, "$2", root.Children[1].Title)

	sessionOne := root.Children[0]
	assert.Equal(t, NodeKindSession, sessionOne.Kind)
	assert.Equal(t, 3, sessionOne.Count)
	require.NotNil(t, sessionOne.LatestEvent)
	assert.Equal(t, 2, sessionOne.LatestEvent.ID)

	require.Len(t, sessionOne.Children, 2)
	assert.Equal(t, "@1", sessionOne.Children[0].Title)
	assert.Equal(t, "@2", sessionOne.Children[1].Title)

	windowOne := sessionOne.Children[0]
	assert.Equal(t, 2, windowOne.Count)
	require.NotNil(t, windowOne.LatestEvent)
	assert.Equal(t, 2, windowOne.LatestEvent.ID)

	require.Len(t, windowOne.Children, 1)
	paneOne := windowOne.Children[0]
	assert.Equal(t, "%1", paneOne.Title)
	assert.Equal(t, 2, paneOne.Count)
	require.NotNil(t, paneOne.LatestEvent)
	assert.Equal(t, 2, paneOne.LatestEvent.ID)

	require.Len(t, paneOne.Children, 2)
	assert.Equal(t, "Alpha issue", paneOne.Children[0].Title)
	assert.Equal(t, "Beta issue", paneOne.Children[1].Title)
}

func TestFindNotificationPath(t *testing.T) {
	notifications := []notification.Notification{
		{
			ID:        10,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Example notice",
			Timestamp: "2024-01-02T10:00:00Z",
		},
	}

	root := BuildTree(notifications, settings.GroupByPane)

	path, ok := FindNotificationPath(root, notifications[0])
	require.True(t, ok)
	require.Len(t, path, 5)
	assert.Equal(t, NodeKindRoot, path[0].Kind)
	assert.Equal(t, "$1", path[1].Title)
	assert.Equal(t, "@1", path[2].Title)
	assert.Equal(t, "%1", path[3].Title)
	assert.Equal(t, NodeKindNotification, path[4].Kind)
}

// TestBuildTreeWithEmptyNotifications tests that BuildTree handles
// empty notification list correctly.
func TestBuildTreeWithEmptyNotifications(t *testing.T) {
	notifications := []notification.Notification{}

	root := BuildTree(notifications, settings.GroupByPane)

	require.NotNil(t, root)
	assert.Equal(t, NodeKindRoot, root.Kind)
	assert.Equal(t, 0, root.Count)
	assert.Empty(t, root.Children)
}

// TestBuildTreeCountsMultipleNotificationsPerPane tests that group counts
// are correctly calculated when multiple notifications exist in the same pane.
func TestBuildTreeCountsMultipleNotificationsPerPane(t *testing.T) {
	notifications := []notification.Notification{
		{
			ID:        1,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "First message",
			Timestamp: "2024-01-01T10:00:00Z",
		},
		{
			ID:        2,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Second message",
			Timestamp: "2024-01-02T10:00:00Z",
		},
		{
			ID:        3,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Third message",
			Timestamp: "2024-01-03T10:00:00Z",
		},
	}

	root := BuildTree(notifications, settings.GroupByPane)

	require.NotNil(t, root)
	assert.Equal(t, 3, root.Count)

	// Check session count
	require.Len(t, root.Children, 1)
	sessionNode := root.Children[0]
	assert.Equal(t, NodeKindSession, sessionNode.Kind)
	assert.Equal(t, 3, sessionNode.Count)

	// Check window count
	require.Len(t, sessionNode.Children, 1)
	windowNode := sessionNode.Children[0]
	assert.Equal(t, NodeKindWindow, windowNode.Kind)
	assert.Equal(t, 3, windowNode.Count)

	// Check pane count
	require.Len(t, windowNode.Children, 1)
	paneNode := windowNode.Children[0]
	assert.Equal(t, NodeKindPane, paneNode.Kind)
	assert.Equal(t, 3, paneNode.Count)

	// Check that all three notifications are children
	require.Len(t, paneNode.Children, 3)
	for i, child := range paneNode.Children {
		assert.Equal(t, NodeKindNotification, child.Kind)
		assert.Equal(t, i+1, child.Notification.ID)
	}
}

// TestBuildTreeSortsAlphabetically tests that siblings are sorted
// alphabetically by title (case-insensitive).
func TestBuildTreeSortsAlphabetically(t *testing.T) {
	notifications := []notification.Notification{
		{
			ID:        1,
			Session:   "$3",
			Window:    "@3",
			Pane:      "%3",
			Message:   "Gamma",
			Timestamp: "2024-01-03T10:00:00Z",
		},
		{
			ID:        2,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Alpha",
			Timestamp: "2024-01-01T10:00:00Z",
		},
		{
			ID:        3,
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
			Message:   "Beta",
			Timestamp: "2024-01-02T10:00:00Z",
		},
	}

	root := BuildTree(notifications, settings.GroupByPane)

	require.NotNil(t, root)
	require.Len(t, root.Children, 3)

	// Sessions should be sorted: $1, $2, $3
	assert.Equal(t, "$1", root.Children[0].Title)
	assert.Equal(t, "$2", root.Children[1].Title)
	assert.Equal(t, "$3", root.Children[2].Title)
}

func TestBuildTreeRespectsGroupBy(t *testing.T) {
	notifications := []notification.Notification{
		{
			ID:        1,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Beta issue",
			Timestamp: "2024-01-02T10:00:00Z",
		},
		{
			ID:        2,
			Session:   "$1",
			Window:    "@2",
			Pane:      "%2",
			Message:   "Alpha issue",
			Timestamp: "2024-01-03T10:00:00Z",
		},
		{
			ID:        3,
			Session:   "$2",
			Window:    "@1",
			Pane:      "%3",
			Message:   "Gamma issue",
			Timestamp: "2024-01-01T10:00:00Z",
		},
	}

	tests := []struct {
		name            string
		groupBy         string
		firstLevelKinds []NodeKind
		secondLevelKind NodeKind
	}{
		{
			name:            "none",
			groupBy:         settings.GroupByNone,
			firstLevelKinds: []NodeKind{NodeKindNotification, NodeKindNotification, NodeKindNotification},
		},
		{
			name:            "session",
			groupBy:         settings.GroupBySession,
			firstLevelKinds: []NodeKind{NodeKindSession, NodeKindSession},
			secondLevelKind: NodeKindNotification,
		},
		{
			name:            "window",
			groupBy:         settings.GroupByWindow,
			firstLevelKinds: []NodeKind{NodeKindSession, NodeKindSession},
			secondLevelKind: NodeKindWindow,
		},
		{
			name:            "pane",
			groupBy:         settings.GroupByPane,
			firstLevelKinds: []NodeKind{NodeKindSession, NodeKindSession},
			secondLevelKind: NodeKindWindow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := BuildTree(notifications, tt.groupBy)

			require.NotNil(t, root)
			assert.Equal(t, 3, root.Count)
			require.Len(t, root.Children, len(tt.firstLevelKinds))

			for i, kind := range tt.firstLevelKinds {
				assert.Equal(t, kind, root.Children[i].Kind)
			}

			switch tt.groupBy {
			case settings.GroupByNone:
				assert.Equal(t, "Alpha issue", root.Children[0].Title)
				assert.Equal(t, "Beta issue", root.Children[1].Title)
				assert.Equal(t, "Gamma issue", root.Children[2].Title)
			case settings.GroupBySession:
				sessionOne := root.Children[0]
				assert.Equal(t, "$1", sessionOne.Title)
				assert.Equal(t, 2, sessionOne.Count)
				require.Len(t, sessionOne.Children, 2)
				for _, child := range sessionOne.Children {
					assert.Equal(t, tt.secondLevelKind, child.Kind)
				}
			case settings.GroupByWindow:
				sessionOne := root.Children[0]
				require.Len(t, sessionOne.Children, 2)
				windowOne := sessionOne.Children[0]
				assert.Equal(t, tt.secondLevelKind, windowOne.Kind)
				require.Len(t, windowOne.Children, 1)
				assert.Equal(t, NodeKindNotification, windowOne.Children[0].Kind)
			case settings.GroupByPane:
				sessionOne := root.Children[0]
				require.Len(t, sessionOne.Children, 2)
				windowOne := sessionOne.Children[0]
				require.Len(t, windowOne.Children, 1)
				paneOne := windowOne.Children[0]
				assert.Equal(t, NodeKindPane, paneOne.Kind)
				require.Len(t, paneOne.Children, 1)
				assert.Equal(t, NodeKindNotification, paneOne.Children[0].Kind)
			}
		})
	}
}

func TestBuildTreeInvalidGroupByFallsBackToPane(t *testing.T) {
	notifications := []notification.Notification{
		{
			ID:        1,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Alpha issue",
			Timestamp: "2024-01-03T10:00:00Z",
		},
	}

	root := BuildTree(notifications, "invalid")
	require.NotNil(t, root)
	require.Len(t, root.Children, 1)
	assert.Equal(t, NodeKindSession, root.Children[0].Kind)
	require.Len(t, root.Children[0].Children, 1)
	assert.Equal(t, NodeKindWindow, root.Children[0].Children[0].Kind)
	require.Len(t, root.Children[0].Children[0].Children, 1)
	assert.Equal(t, NodeKindPane, root.Children[0].Children[0].Children[0].Kind)
}

func TestBuildTreePaneMessageGroupsWithoutLeaves(t *testing.T) {
	notifications := []notification.Notification{
		{
			ID:        1,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Repeated",
			Timestamp: "2024-01-03T10:00:00Z",
		},
		{
			ID:        2,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Repeated",
			Timestamp: "2024-01-03T10:01:00Z",
		},
	}

	root := BuildTree(notifications, settings.GroupByPaneMessage)

	require.NotNil(t, root)
	require.Len(t, root.Children, 1)
	require.Len(t, root.Children[0].Children, 1)
	require.Len(t, root.Children[0].Children[0].Children, 1)
	require.Len(t, root.Children[0].Children[0].Children[0].Children, 1)

	messageNode := root.Children[0].Children[0].Children[0].Children[0]
	assert.Equal(t, NodeKindMessage, messageNode.Kind)
	assert.Equal(t, "Repeated", messageNode.Title)
	assert.Equal(t, 2, messageNode.Count)
	assert.Empty(t, messageNode.Children)
	require.NotNil(t, messageNode.LatestEvent)
}

func TestFindNotificationPathWithGroupByNone(t *testing.T) {
	notifications := []notification.Notification{
		{
			ID:        1,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Message:   "Alpha issue",
			Timestamp: "2024-01-03T10:00:00Z",
		},
	}

	root := BuildTree(notifications, settings.GroupByNone)
	path, ok := FindNotificationPath(root, notifications[0])

	require.True(t, ok)
	require.Len(t, path, 2)
	assert.Equal(t, NodeKindRoot, path[0].Kind)
	assert.Equal(t, NodeKindNotification, path[1].Kind)
}

// TestBuildTreeUnreadCounts tests that unread counts are correctly calculated
// at each grouping level.
func TestBuildTreeUnreadCounts(t *testing.T) {
	notifications := []notification.Notification{
		{
			ID:            1,
			Session:       "$1",
			Window:        "@1",
			Pane:          "%1",
			Message:       "Unread message 1",
			Timestamp:     "2024-01-01T10:00:00Z",
			ReadTimestamp: "", // Unread
		},
		{
			ID:            2,
			Session:       "$1",
			Window:        "@1",
			Pane:          "%1",
			Message:       "Read message 1",
			Timestamp:     "2024-01-02T10:00:00Z",
			ReadTimestamp: "2024-01-02T11:00:00Z", // Read
		},
		{
			ID:            3,
			Session:       "$1",
			Window:        "@2",
			Pane:          "%2",
			Message:       "Unread message 2",
			Timestamp:     "2024-01-03T10:00:00Z",
			ReadTimestamp: "", // Unread
		},
		{
			ID:            4,
			Session:       "$2",
			Window:        "@1",
			Pane:          "%3",
			Message:       "Read message 2",
			Timestamp:     "2024-01-04T10:00:00Z",
			ReadTimestamp: "2024-01-04T11:00:00Z", // Read
		},
		{
			ID:            5,
			Session:       "$2",
			Window:        "@1",
			Pane:          "%3",
			Message:       "Unread message 3",
			Timestamp:     "2024-01-05T10:00:00Z",
			ReadTimestamp: "", // Unread
		},
	}

	root := BuildTree(notifications, settings.GroupByPane)

	require.NotNil(t, root)
	assert.Equal(t, 5, root.Count)
	assert.Equal(t, 3, root.UnreadCount) // 3 unread notifications total

	// Check session $1
	require.Len(t, root.Children, 2)
	sessionOne := root.Children[0]
	assert.Equal(t, NodeKindSession, sessionOne.Kind)
	assert.Equal(t, "$1", sessionOne.Title)
	assert.Equal(t, 3, sessionOne.Count)
	assert.Equal(t, 2, sessionOne.UnreadCount) // 2 unread in session $1

	// Check window @1 in session $1
	require.Len(t, sessionOne.Children, 2)
	windowOne := sessionOne.Children[0]
	assert.Equal(t, NodeKindWindow, windowOne.Kind)
	assert.Equal(t, "@1", windowOne.Title)
	assert.Equal(t, 2, windowOne.Count)
	assert.Equal(t, 1, windowOne.UnreadCount) // 1 unread in window @1

	// Check pane %1 in window @1
	require.Len(t, windowOne.Children, 1)
	paneOne := windowOne.Children[0]
	assert.Equal(t, NodeKindPane, paneOne.Kind)
	assert.Equal(t, "%1", paneOne.Title)
	assert.Equal(t, 2, paneOne.Count)
	assert.Equal(t, 1, paneOne.UnreadCount) // 1 unread in pane %1

	// Check session $2
	sessionTwo := root.Children[1]
	assert.Equal(t, NodeKindSession, sessionTwo.Kind)
	assert.Equal(t, "$2", sessionTwo.Title)
	assert.Equal(t, 2, sessionTwo.Count)
	assert.Equal(t, 1, sessionTwo.UnreadCount) // 1 unread in session $2

	// Check window @1 in session $2
	require.Len(t, sessionTwo.Children, 1)
	windowTwoSession := sessionTwo.Children[0]
	assert.Equal(t, NodeKindWindow, windowTwoSession.Kind)
	assert.Equal(t, "@1", windowTwoSession.Title)
	assert.Equal(t, 2, windowTwoSession.Count)
	assert.Equal(t, 1, windowTwoSession.UnreadCount) // 1 unread in window @1

	// Check pane %3 in window @1
	require.Len(t, windowTwoSession.Children, 1)
	paneThree := windowTwoSession.Children[0]
	assert.Equal(t, NodeKindPane, paneThree.Kind)
	assert.Equal(t, "%3", paneThree.Title)
	assert.Equal(t, 2, paneThree.Count)
	assert.Equal(t, 1, paneThree.UnreadCount) // 1 unread in pane %3
}
