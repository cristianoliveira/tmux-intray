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

func TestBuildTreePaneMessageGroupsWithoutLeaves(t *testing.T) {
	service := NewTreeService(model.GroupByPane).(*DefaultTreeService)

	notifs := []notification.Notification{
		{
			ID:        1,
			Timestamp: "2025-01-01T10:00:00Z",
			Session:   "session-a",
			Window:    "window-1",
			Pane:      "pane-1",
			Message:   "hello",
		},
		{
			ID:        2,
			Timestamp: "2025-01-01T10:01:00Z",
			Session:   "session-a",
			Window:    "window-1",
			Pane:      "pane-1",
			Message:   "hello",
		},
	}

	err := service.BuildTree(notifs, settings.GroupByPaneMessage)
	require.NoError(t, err)

	root := service.GetTreeRoot()
	require.NotNil(t, root)
	require.Len(t, root.Children, 1)

	session := root.Children[0]
	require.Len(t, session.Children, 1)

	window := session.Children[0]
	require.Len(t, window.Children, 1)

	pane := window.Children[0]
	require.Len(t, pane.Children, 1)

	message := pane.Children[0]
	assert.Equal(t, model.NodeKindMessage, message.Kind)
	assert.Equal(t, "hello", message.Title)
	assert.Equal(t, 2, message.Count)
	assert.Empty(t, message.Children)
	require.NotNil(t, message.LatestEvent)
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
