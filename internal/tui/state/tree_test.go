package state

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
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

	root := BuildTree(notifications)

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

	root := BuildTree(notifications)

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

	root := BuildTree(notifications)

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

	root := BuildTree(notifications)

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

	root := BuildTree(notifications)

	require.NotNil(t, root)
	require.Len(t, root.Children, 3)

	// Sessions should be sorted: $1, $2, $3
	assert.Equal(t, "$1", root.Children[0].Title)
	assert.Equal(t, "$2", root.Children[1].Title)
	assert.Equal(t, "$3", root.Children[2].Title)
}
