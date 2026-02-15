package service

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeServiceRootLifecycle(t *testing.T) {
	service := NewTreeService(model.GroupByPane).(*DefaultTreeService)

	assert.Nil(t, service.GetTreeRoot())

	err := service.BuildTree(sampleNotifications(), settings.GroupBySession)
	require.NoError(t, err)
	assert.NotNil(t, service.GetTreeRoot())

	service.ClearTree()

	assert.Nil(t, service.GetTreeRoot())
	assert.Nil(t, service.GetVisibleNodes())
	assert.True(t, service.cacheValid)
}

func TestBuildTreeInvalidatesVisibleNodesCache(t *testing.T) {
	service := NewTreeService(model.GroupByPane).(*DefaultTreeService)

	err := service.BuildTree(sampleNotifications()[:1], settings.GroupBySession)
	require.NoError(t, err)

	visibleFirst := service.GetVisibleNodes()
	assert.Len(t, visibleFirst, 1)
	assert.True(t, service.cacheValid)

	err = service.BuildTree(sampleNotifications(), settings.GroupBySession)
	require.NoError(t, err)

	assert.False(t, service.cacheValid)
	assert.Nil(t, service.visibleNodesCache)

	visibleAfterRebuild := service.GetVisibleNodes()
	assert.Len(t, visibleAfterRebuild, 2)
	assert.True(t, service.cacheValid)
}

func TestApplyExpansionStateInvalidatesVisibleNodesCache(t *testing.T) {
	service := NewTreeService(model.GroupByPane).(*DefaultTreeService)

	err := service.BuildTree(sampleNotifications()[:1], settings.GroupByPane)
	require.NoError(t, err)

	visibleBefore := service.GetVisibleNodes()
	assert.Len(t, visibleBefore, 1)
	assert.True(t, service.cacheValid)

	root := service.GetTreeRoot()
	require.NotNil(t, root)
	require.Len(t, root.Children, 1)
	session := root.Children[0]
	require.Len(t, session.Children, 1)
	window := session.Children[0]
	require.Len(t, window.Children, 1)
	pane := window.Children[0]

	service.ApplyExpansionState(map[string]bool{
		service.GetNodeIdentifier(session): true,
		service.GetNodeIdentifier(window):  true,
		service.GetNodeIdentifier(pane):    true,
	})

	assert.False(t, service.cacheValid)
	assert.Nil(t, service.visibleNodesCache)

	visibleAfter := service.GetVisibleNodes()
	assert.Len(t, visibleAfter, 4)
	assert.True(t, service.cacheValid)
}

func TestPruneEmptyGroupsInvalidatesVisibleNodesCache(t *testing.T) {
	service := NewTreeService(model.GroupByPane).(*DefaultTreeService)

	leaf := &model.TreeNode{
		Kind:    model.NodeKindNotification,
		Title:   "notification",
		Display: "notification",
		Notification: &notification.Notification{
			ID:      99,
			Message: "kept",
		},
	}

	emptySession := &model.TreeNode{
		Kind:    model.NodeKindSession,
		Title:   "empty",
		Display: "empty",
	}

	populatedSession := &model.TreeNode{
		Kind:     model.NodeKindSession,
		Title:    "with-notification",
		Display:  "with-notification",
		Expanded: true,
		Children: []*model.TreeNode{leaf},
	}

	service.treeRoot = &model.TreeNode{
		Kind:     model.NodeKindRoot,
		Title:    "root",
		Display:  "root",
		Expanded: true,
		Children: []*model.TreeNode{emptySession, populatedSession},
	}

	visibleBefore := service.GetVisibleNodes()
	assert.Len(t, visibleBefore, 3)
	assert.True(t, service.cacheValid)

	service.PruneEmptyGroups()

	assert.False(t, service.cacheValid)
	assert.Nil(t, service.visibleNodesCache)

	visibleAfter := service.GetVisibleNodes()
	assert.Len(t, visibleAfter, 2)
	assert.Equal(t, "with-notification", visibleAfter[0].Title)
}

func TestGetVisibleNodesCacheConsistencyAndRefreshAfterInvalidation(t *testing.T) {
	service := NewTreeService(model.GroupByPane).(*DefaultTreeService)

	err := service.BuildTree(sampleNotifications()[:1], settings.GroupBySession)
	require.NoError(t, err)

	first := service.GetVisibleNodes()
	second := service.GetVisibleNodes()

	assert.Equal(t, first, second)
	assert.Len(t, first, 1)
	assert.True(t, service.cacheValid)

	service.InvalidateCache()
	assert.False(t, service.cacheValid)

	root := service.GetTreeRoot()
	require.NotNil(t, root)
	require.Len(t, root.Children, 1)

	service.ExpandNode(root.Children[0])

	refreshed := service.GetVisibleNodes()
	assert.Len(t, refreshed, 2)
	assert.Equal(t, "session-a", refreshed[0].Title)
	assert.Equal(t, "first", refreshed[1].Title)
	assert.True(t, service.cacheValid)
}

func sampleNotifications() []notification.Notification {
	return []notification.Notification{
		{
			ID:        1,
			Timestamp: "2025-01-01T10:00:00Z",
			Session:   "session-a",
			Window:    "window-1",
			Pane:      "pane-1",
			Message:   "first",
		},
		{
			ID:        2,
			Timestamp: "2025-01-01T10:01:00Z",
			Session:   "session-b",
			Window:    "window-2",
			Pane:      "pane-2",
			Message:   "second",
		},
	}
}

func TestBuildTreeWithMessageGrouping(t *testing.T) {
	service := NewTreeService(model.GroupByPane).(*DefaultTreeService)

	// Create notifications with duplicate messages
	notifs := []notification.Notification{
		{
			ID:        1,
			Timestamp: "2025-01-01T10:00:00Z",
			Session:   "session-a",
			Window:    "window-1",
			Pane:      "pane-1",
			Message:   "error occurred",
		},
		{
			ID:        2,
			Timestamp: "2025-01-01T10:01:00Z",
			Session:   "session-b",
			Window:    "window-2",
			Pane:      "pane-2",
			Message:   "error occurred", // Same message as above
		},
		{
			ID:        3,
			Timestamp: "2025-01-01T10:02:00Z",
			Session:   "session-a",
			Window:    "window-1",
			Pane:      "pane-1",
			Message:   "warning issued",
		},
	}

	err := service.BuildTree(notifs, settings.GroupByMessage)
	require.NoError(t, err)

	root := service.GetTreeRoot()
	require.NotNil(t, root)
	require.Len(t, root.Children, 2) // Two unique messages

	// Find message groups
	var errorGroup *model.TreeNode
	var warningGroup *model.TreeNode
	for _, child := range root.Children {
		if child.Kind == model.NodeKindMessage {
			switch child.Title {
			case "error occurred":
				errorGroup = child
			case "warning issued":
				warningGroup = child
			}
		}
	}

	require.NotNil(t, errorGroup)
	require.NotNil(t, warningGroup)

	// Verify error group has 2 notifications
	assert.Equal(t, 2, errorGroup.Count)
	assert.Equal(t, 2, len(errorGroup.Children))
	assert.Equal(t, 2, errorGroup.UnreadCount)

	// Verify warning group has 1 notification
	assert.Equal(t, 1, warningGroup.Count)
	assert.Equal(t, 1, len(warningGroup.Children))
	assert.Equal(t, 1, warningGroup.UnreadCount)
}

func TestBuildTreeWithMessageGroupingAndReadNotifications(t *testing.T) {
	service := NewTreeService(model.GroupByPane).(*DefaultTreeService)

	// Create notifications with duplicate messages, some read
	notifs := []notification.Notification{
		{
			ID:            1,
			Timestamp:     "2025-01-01T10:00:00Z",
			Session:       "session-a",
			Window:        "window-1",
			Pane:          "pane-1",
			Message:       "error occurred",
			ReadTimestamp: "2025-01-01T11:00:00Z", // Read
		},
		{
			ID:        2,
			Timestamp: "2025-01-01T10:01:00Z",
			Session:   "session-b",
			Window:    "window-2",
			Pane:      "pane-2",
			Message:   "error occurred", // Same message, unread
		},
	}

	err := service.BuildTree(notifs, settings.GroupByMessage)
	require.NoError(t, err)

	root := service.GetTreeRoot()
	require.NotNil(t, root)
	require.Len(t, root.Children, 1) // One unique message

	errorGroup := root.Children[0]
	assert.Equal(t, model.NodeKindMessage, errorGroup.Kind)
	assert.Equal(t, 2, errorGroup.Count)
	assert.Equal(t, 1, errorGroup.UnreadCount) // Only one is unread
}

func TestTreeServiceTracksExtendedGroupStats(t *testing.T) {
	service := NewTreeService(model.GroupByPane).(*DefaultTreeService)

	notifs := []notification.Notification{
		{
			ID:        1,
			Timestamp: "2025-01-01T10:00:00Z",
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
			Level:     "error",
		},
		{
			ID:        2,
			Timestamp: "2025-01-01T09:30:00Z",
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
			Level:     "warning",
		},
	}

	require.NoError(t, service.BuildTree(notifs, settings.GroupBySession))
	root := service.GetTreeRoot()
	require.NotNil(t, root)
	require.NotNil(t, root.EarliestEvent)
	assert.Equal(t, "2025-01-01T09:30:00Z", root.EarliestEvent.Timestamp)
	require.NotNil(t, root.LevelCounts)
	assert.Equal(t, 1, root.LevelCounts["error"])
	assert.Equal(t, 1, root.LevelCounts["warning"])
	assert.Len(t, root.Sources, 2)
}

func TestGetTreeLevelWithMessageNode(t *testing.T) {
	service := NewTreeService(model.GroupByPane).(*DefaultTreeService)

	messageNode := &model.TreeNode{
		Kind: model.NodeKindMessage,
	}

	level := service.GetTreeLevel(messageNode)
	assert.Equal(t, 0, level)
}

func TestGroupByCommandHandlerWithMessage(t *testing.T) {
	mockModel := new(MockModelInterface)
	handler := &GroupByCommandHandler{model: mockModel}

	// Test Execute with message
	mockModel.On("GetGroupBy").Return("none")
	mockModel.On("SetGroupBy", "message").Return(nil)
	mockModel.On("ApplySearchFilter")
	mockModel.On("ResetCursor")
	mockModel.On("SaveSettings").Return(nil)
	result, err := handler.Execute([]string{"message"})
	assert.NoError(t, err)
	assert.Contains(t, result.Message, "Group by: message")

	// Test Validate
	err = handler.Validate([]string{"message"})
	assert.NoError(t, err)

	// Test Complete
	suggestions := handler.Complete([]string{})
	assert.Contains(t, suggestions, "message")
}
