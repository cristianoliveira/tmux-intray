package state

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStorage(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	storage.Reset()
	require.NoError(t, storage.Init())

	return tmpDir
}

func setupConfig(t *testing.T, dir string) {
	t.Helper()

	t.Setenv("TMUX_INTRAY_CONFIG_DIR", dir)
}

func stubSessionFetchers(t *testing.T) *tmux.MockClient {
	t.Helper()

	mockClient := new(tmux.MockClient)
	// Mock ListSessions to return empty map
	mockClient.On("ListSessions").Return(map[string]string{}, nil)
	// Mock ListWindows to return empty map
	mockClient.On("ListWindows").Return(map[string]string{}, nil)
	// Mock ListPanes to return empty map
	mockClient.On("ListPanes").Return(map[string]string{}, nil)

	return mockClient
}

func TestNewModelInitialState(t *testing.T) {
	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	model, err := NewModel(mockClient)

	require.NoError(t, err)
	// NewUIState initializes with default values
	assert.Equal(t, defaultViewportWidth, model.uiState.GetWidth())
	assert.Equal(t, defaultViewportHeight, model.uiState.GetHeight())
	assert.Equal(t, 0, model.uiState.GetCursor())
	assert.False(t, model.uiState.IsSearchMode())
	assert.Equal(t, "", model.uiState.GetSearchQuery())
	assert.Empty(t, model.notifications)
	assert.Empty(t, model.filtered)
	assert.NotNil(t, model.expansionState)
	assert.Empty(t, model.expansionState)
	assert.Nil(t, model.treeRoot)
	assert.Empty(t, model.visibleNodes)
}

func TestModelGroupedModeBuildsVisibleNodes(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		groupBy:  settings.GroupByPane,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
			{ID: 2, Session: "$2", Window: "@1", Pane: "%2", Message: "Two"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	require.NotNil(t, model.treeRoot)
	require.Len(t, model.visibleNodes, 8)
	assert.Equal(t, NodeKindSession, model.visibleNodes[0].Kind)
	assert.Equal(t, NodeKindWindow, model.visibleNodes[1].Kind)
	assert.Equal(t, NodeKindPane, model.visibleNodes[2].Kind)
	assert.Equal(t, NodeKindNotification, model.visibleNodes[3].Kind)
	assert.Equal(t, NodeKindSession, model.visibleNodes[4].Kind)
	assert.Equal(t, NodeKindWindow, model.visibleNodes[5].Kind)
	assert.Equal(t, NodeKindPane, model.visibleNodes[6].Kind)
	assert.Equal(t, NodeKindNotification, model.visibleNodes[7].Kind)
}

func TestComputeVisibleNodesUsesCacheWhenValid(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		groupBy:  settings.GroupByPane,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()
	require.True(t, model.cacheValid)
	require.Len(t, model.visibleNodes, 4)

	groupNode := model.visibleNodes[0]
	require.NotNil(t, groupNode)
	require.True(t, groupNode.Expanded)

	groupNode.Expanded = false
	visible := model.computeVisibleNodes()

	assert.Len(t, visible, 4)
	assert.Equal(t, NodeKindNotification, visible[len(visible)-1].Kind)
}

func TestInvalidateCacheForcesVisibleNodesRecompute(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		groupBy:  settings.GroupByPane,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()
	require.True(t, model.cacheValid)
	require.Len(t, model.visibleNodes, 4)

	groupNode := model.visibleNodes[0]
	groupNode.Expanded = false

	model.invalidateCache()
	assert.False(t, model.cacheValid)
	assert.Nil(t, model.visibleNodesCache)

	visible := model.computeVisibleNodes()
	assert.True(t, model.cacheValid)
	assert.Len(t, visible, 1)
	assert.Equal(t, NodeKindSession, visible[0].Kind)
}

func TestCacheInvalidationOnSearchAndExpansionChanges(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		groupBy:  settings.GroupBySession,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "alpha message"},
			{ID: 2, Session: "$2", Window: "@2", Pane: "%2", Message: "beta message"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()
	require.True(t, model.cacheValid)
	require.Len(t, model.visibleNodes, 4)

	model.uiState.SetSearchQuery("alpha")
	model.applySearchFilter()
	model.resetCursor()
	require.True(t, model.cacheValid)
	require.Len(t, model.visibleNodes, 2)

	groupNode := model.visibleNodes[0]
	model.uiState.SetCursor(0)
	require.True(t, groupNode.Expanded)

	handled := model.toggleNodeExpansion()
	require.True(t, handled)
	assert.False(t, groupNode.Expanded)
	assert.True(t, model.cacheValid)
	assert.Len(t, model.visibleNodes, 1)
}

func BenchmarkComputeVisibleNodesCache(b *testing.B) {
	notifications := make([]notification.Notification, 0, 1000)
	for i := 0; i < 1000; i++ {
		notifications = append(notifications, notification.Notification{
			ID:      i + 1,
			Session: "$1",
			Window:  "@1",
			Pane:    "%1",
			Message: "bench message",
		})
	}

	model := &Model{
		uiState:        NewUIState(),
		viewMode:       viewModeGrouped,
		groupBy:        settings.GroupByPane,
		notifications:  notifications,
		expansionState: map[string]bool{},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80
	model.applySearchFilter()
	model.resetCursor()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.computeVisibleNodes()
	}
}

func TestModelGroupedModeRespectsGroupByDepth(t *testing.T) {
	tests := []struct {
		name                 string
		groupBy              string
		expectedVisibleKinds []NodeKind
	}{
		{
			name:                 "session",
			groupBy:              settings.GroupBySession,
			expectedVisibleKinds: []NodeKind{NodeKindSession, NodeKindNotification, NodeKindSession, NodeKindNotification},
		},
		{
			name:                 "window",
			groupBy:              settings.GroupByWindow,
			expectedVisibleKinds: []NodeKind{NodeKindSession, NodeKindWindow, NodeKindNotification, NodeKindSession, NodeKindWindow, NodeKindNotification},
		},
		{
			name:                 "none",
			groupBy:              settings.GroupByNone,
			expectedVisibleKinds: []NodeKind{NodeKindNotification, NodeKindNotification},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &Model{
				uiState:  NewUIState(),
				viewMode: viewModeGrouped,
				groupBy:  tt.groupBy,
				notifications: []notification.Notification{
					{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
					{ID: 2, Session: "$2", Window: "@2", Pane: "%2", Message: "Two"},
				},
			}
			model.uiState.SetWidth(80)
			model.uiState.GetViewport().Width = 80

			model.applySearchFilter()
			model.resetCursor()

			require.Len(t, model.visibleNodes, len(tt.expectedVisibleKinds))
			for i, kind := range tt.expectedVisibleKinds {
				assert.Equal(t, kind, model.visibleNodes[i].Kind)
			}
		})
	}
}

func TestModelSwitchesViewModes(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()
	require.NotNil(t, model.treeRoot)
	require.NotEmpty(t, model.visibleNodes)

	model.viewMode = "flat"
	model.applySearchFilter()
	model.resetCursor()
	assert.Nil(t, model.treeRoot)
	assert.Empty(t, model.visibleNodes)
}

func TestToggleNodeExpansionGroupedView(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	var groupNode *Node
	groupIndex := -1
	for idx, node := range model.visibleNodes {
		if node != nil && isGroupNode(node) {
			groupNode = node
			groupIndex = idx
			break
		}
	}
	require.NotNil(t, groupNode)
	require.NotEqual(t, -1, groupIndex)
	model.uiState.SetCursor(groupIndex)

	require.True(t, groupNode.Expanded)

	handled := model.toggleNodeExpansion()
	require.True(t, handled)
	assert.False(t, groupNode.Expanded)
	assert.Len(t, model.visibleNodes, 1)
	assert.Equal(t, 0, model.uiState.GetCursor())

	handled = model.toggleNodeExpansion()
	require.True(t, handled)
	assert.True(t, groupNode.Expanded)
	assert.Greater(t, len(model.visibleNodes), 1)
}

func TestToggleFoldTogglesGroupNode(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	var groupNode *Node
	groupIndex := -1
	for idx, node := range model.visibleNodes {
		if node != nil && isGroupNode(node) {
			groupNode = node
			groupIndex = idx
			break
		}
	}
	require.NotNil(t, groupNode)
	require.NotEqual(t, -1, groupIndex)
	model.uiState.SetCursor(groupIndex)

	require.True(t, groupNode.Expanded)

	model.toggleFold()
	assert.False(t, groupNode.Expanded)
	assert.Len(t, model.visibleNodes, 1)

	model.toggleFold()
	assert.True(t, groupNode.Expanded)
	assert.Greater(t, len(model.visibleNodes), 1)
}

func TestToggleFoldWorksAtPaneDepth(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	var paneNode *Node
	paneIndex := -1
	for idx, node := range model.visibleNodes {
		if node != nil && node.Kind == NodeKindPane {
			paneNode = node
			paneIndex = idx
			break
		}
	}
	require.NotNil(t, paneNode)
	require.NotEqual(t, -1, paneIndex)
	model.uiState.SetCursor(paneIndex)

	require.True(t, paneNode.Expanded)

	model.toggleFold()
	assert.False(t, paneNode.Expanded)

	model.toggleFold()
	assert.True(t, paneNode.Expanded)
}

func TestCollapseNodeMovesCursorToParent(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	var leafNode *Node
	leafIndex := -1
	for idx, node := range model.visibleNodes {
		if node != nil && node.Kind == NodeKindNotification {
			leafNode = node
			leafIndex = idx
			break
		}
	}
	require.NotNil(t, leafNode)
	require.NotEqual(t, -1, leafIndex)

	path, ok := findNodePath(model.treeRoot, leafNode)
	require.True(t, ok)
	var paneNode *Node
	for _, node := range path {
		if node != nil && node.Kind == NodeKindPane {
			paneNode = node
			break
		}
	}
	require.NotNil(t, paneNode)

	model.uiState.SetCursor(leafIndex)
	model.collapseNode(paneNode)

	paneIndex := indexOfNode(model.visibleNodes, paneNode)
	require.NotEqual(t, -1, paneIndex)
	assert.Equal(t, paneIndex, model.uiState.GetCursor())
}

func TestToggleNodeExpansionIgnoresLeafNodes(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	var leafNode *Node
	leafIndex := -1
	for idx, node := range model.visibleNodes {
		if node != nil && node.Kind == NodeKindNotification {
			leafNode = node
			leafIndex = idx
			break
		}
	}
	require.NotNil(t, leafNode)
	require.NotEqual(t, -1, leafIndex)

	model.uiState.SetCursor(leafIndex)
	visibleBefore := len(model.visibleNodes)

	handled := model.toggleNodeExpansion()

	assert.False(t, handled)
	assert.Len(t, model.visibleNodes, visibleBefore)
}

func TestToggleFoldIgnoresLeafNodes(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	var leafNode *Node
	leafIndex := -1
	for idx, node := range model.visibleNodes {
		if node != nil && node.Kind == NodeKindNotification {
			leafNode = node
			leafIndex = idx
			break
		}
	}
	require.NotNil(t, leafNode)
	require.NotEqual(t, -1, leafIndex)

	model.uiState.SetCursor(leafIndex)
	visibleBefore := len(model.visibleNodes)

	model.toggleFold()

	assert.Len(t, model.visibleNodes, visibleBefore)
}

func TestToggleFoldExpandsDefaultWhenAllCollapsed(t *testing.T) {
	model := &Model{
		uiState:            NewUIState(),
		viewMode:           viewModeGrouped,
		defaultExpandLevel: 2,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	var collapseAll func(node *Node)
	collapseAll = func(node *Node) {
		if node == nil {
			return
		}
		if isGroupNode(node) {
			node.Expanded = false
		}
		for _, child := range node.Children {
			collapseAll(child)
		}
	}
	collapseAll(model.treeRoot)
	model.visibleNodes = model.computeVisibleNodes()
	model.uiState.SetCursor(0)

	require.True(t, model.allGroupsCollapsed())

	model.toggleFold()

	sessionNode := findChildByTitle(model.treeRoot, NodeKindSession, "$1")
	require.NotNil(t, sessionNode)
	windowNode := findChildByTitle(sessionNode, NodeKindWindow, "@1")
	require.NotNil(t, windowNode)
	paneNode := findChildByTitle(windowNode, NodeKindPane, "%1")
	require.NotNil(t, paneNode)

	assert.True(t, sessionNode.Expanded)
	assert.True(t, windowNode.Expanded)
	assert.False(t, paneNode.Expanded)
}

func TestModelSelectedNotificationGroupedView(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "b", Window: "@1", Pane: "%1", Message: "B"},
			{ID: 2, Session: "a", Window: "@1", Pane: "%1", Message: "A"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()
	cursorIndex := -1
	for idx, node := range model.visibleNodes {
		if node == nil || node.Kind != NodeKindNotification || node.Notification == nil {
			continue
		}
		if node.Notification.Session == "a" {
			cursorIndex = idx
			break
		}
	}
	require.NotEqual(t, -1, cursorIndex)
	model.uiState.SetCursor(cursorIndex)

	selected, ok := model.selectedNotification()

	require.True(t, ok)
	assert.Equal(t, "a", selected.Session)
}

func TestModelInitReturnsNil(t *testing.T) {
	model := &Model{}

	cmd := model.Init()

	assert.Nil(t, cmd)
}

func TestModelUpdateHandlesNavigation(t *testing.T) {
	stubSessionFetchers(t)

	model := &Model{
		uiState: NewUIState(),
		notifications: []notification.Notification{
			{ID: 1, Message: "First"},
			{ID: 2, Message: "Second"},
			{ID: 3, Message: "Third"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "First"},
			{ID: 2, Message: "Second"},
			{ID: 3, Message: "Third"},
		},
	}
	model.uiState.SetCursor(0)
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 0, model.uiState.GetCursor())

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 1, model.uiState.GetCursor())

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 2, model.uiState.GetCursor())

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 2, model.uiState.GetCursor())

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 1, model.uiState.GetCursor())
}

func TestModelUpdateHandlesSearch(t *testing.T) {
	stubSessionFetchers(t)

	model := &Model{
		uiState: NewUIState(),
		notifications: []notification.Notification{
			{ID: 1, Message: "Error: file not found"},
			{ID: 2, Message: "Warning: low memory"},
			{ID: 3, Message: "Error: connection failed"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "Error: file not found"},
			{ID: 2, Message: "Warning: low memory"},
			{ID: 3, Message: "Error: connection failed"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.True(t, model.uiState.IsSearchMode())
	assert.Equal(t, "", model.uiState.GetSearchQuery())
	assert.Equal(t, 0, model.uiState.GetCursor())
	assert.Len(t, model.filtered, 3)

	model.uiState.SetSearchQuery("error")
	model.applySearchFilter()
	model.resetCursor()

	require.Len(t, model.filtered, 2)
	assert.True(t, strings.Contains(model.filtered[0].Message, "Error"))

	model.uiState.SetSearchQuery("not found")
	model.applySearchFilter()
	model.resetCursor()

	require.Len(t, model.filtered, 1)
	assert.True(t, strings.Contains(strings.ToLower(model.filtered[0].Message), "not found"))

	model.uiState.SetSearchQuery("")
	model.applySearchFilter()
	model.resetCursor()

	assert.Len(t, model.filtered, 3)
}

func TestModelUpdateCyclesViewModesWithPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		uiState:  NewUIState(),
		viewMode: settings.ViewModeCompact,
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}

	updated, _ := model.Update(msg)
	model = updated.(*Model)
	assert.Equal(t, settings.ViewModeDetailed, model.viewMode)
	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)

	updated, _ = model.Update(msg)
	model = updated.(*Model)
	assert.Equal(t, settings.ViewModeGrouped, model.viewMode)
	loaded, err = settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.ViewModeGrouped, loaded.ViewMode)

	updated, _ = model.Update(msg)
	model = updated.(*Model)
	assert.Equal(t, settings.ViewModeCompact, model.viewMode)
	loaded, err = settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.ViewModeCompact, loaded.ViewMode)
}

func TestModelUpdateIgnoresViewModeCycleInSearchAndCommandModes(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: settings.ViewModeCompact,
	}
	model.uiState.SetSearchMode(true)
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
	updated, _ := model.Update(msg)
	model = updated.(*Model)
	assert.Equal(t, settings.ViewModeCompact, model.viewMode)
	assert.Equal(t, "v", model.uiState.GetSearchQuery())

	model.uiState.SetSearchMode(false)
	model.uiState.SetCommandMode(true)
	updated, _ = model.Update(msg)
	model = updated.(*Model)
	assert.Equal(t, settings.ViewModeCompact, model.viewMode)
	assert.Equal(t, "v", model.uiState.GetCommandQuery())
}

func TestApplySearchFilterReadStatus(t *testing.T) {
	model := &Model{
		uiState: NewUIState(),
		notifications: []notification.Notification{
			{ID: 1, Message: "Alpha", ReadTimestamp: "2024-01-01T12:00:00Z"},
			{ID: 2, Message: "Beta"},
		},
		filtered: []notification.Notification{},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.uiState.SetSearchQuery("read")
	model.applySearchFilter()
	model.resetCursor()
	require.Len(t, model.filtered, 1)
	assert.True(t, model.filtered[0].IsRead())

	model.uiState.SetSearchQuery("unread")
	model.applySearchFilter()
	model.resetCursor()
	require.Len(t, model.filtered, 1)
	assert.False(t, model.filtered[0].IsRead())

	model.uiState.SetSearchQuery("unread beta")
	model.applySearchFilter()
	model.resetCursor()
	require.Len(t, model.filtered, 1)
	assert.Equal(t, "Beta", model.filtered[0].Message)

	model.uiState.SetSearchQuery("read alpha")
	model.applySearchFilter()
	model.resetCursor()
	require.Len(t, model.filtered, 1)
	assert.Equal(t, "Alpha", model.filtered[0].Message)
}

// TestApplySearchFilterWithMockProvider tests that applySearchFilter correctly
// uses a custom mock search provider when set.
func TestApplySearchFilterWithMockProvider(t *testing.T) {
	mockProvider := new(search.MockProvider)

	notifications := []notification.Notification{
		{ID: 1, Message: "First notification"},
		{ID: 2, Message: "Second notification"},
		{ID: 3, Message: "Third notification"},
	}

	// Set up mock to match only ID 1 and 3
	mockProvider.On("Match", notifications[0], "test").Return(true)
	mockProvider.On("Match", notifications[1], "test").Return(false)
	mockProvider.On("Match", notifications[2], "test").Return(true)

	model := &Model{
		uiState:        NewUIState(),
		notifications:  notifications,
		filtered:       []notification.Notification{},
		searchProvider: mockProvider,
	}
	model.uiState.SetSearchQuery("test")
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	require.Len(t, model.filtered, 2)
	assert.Equal(t, notifications[0].ID, model.filtered[0].ID)
	assert.Equal(t, notifications[2].ID, model.filtered[1].ID)

	mockProvider.AssertExpectations(t)
}

// TestApplySearchFilterUsesDefaultTokenProvider tests that applySearchFilter
// falls back to TokenProvider when no custom provider is set.
func TestApplySearchFilterUsesDefaultTokenProvider(t *testing.T) {
	model := &Model{
		uiState: NewUIState(),
		notifications: []notification.Notification{
			{ID: 1, Message: "Error: file not found", Level: "error"},
			{ID: 2, Message: "Warning: low memory", Level: "warning"},
			{ID: 3, Message: "Error: connection failed", Level: "error"},
		},
		filtered: []notification.Notification{},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	// No custom searchProvider set, should use default TokenProvider
	assert.Nil(t, model.searchProvider)

	// Test case-insensitive matching (default behavior)
	model.uiState.SetSearchQuery("error")
	model.applySearchFilter()
	model.resetCursor()

	require.Len(t, model.filtered, 2)
	assert.Contains(t, model.filtered[0].Message, "Error")
	assert.Contains(t, model.filtered[1].Message, "Error")

	// Test token-based matching (all tokens must match)
	model.uiState.SetSearchQuery("error file")
	model.applySearchFilter()
	model.resetCursor()

	require.Len(t, model.filtered, 1)
	assert.Contains(t, model.filtered[0].Message, "file not found")

	// Test read/unread filtering
	model.notifications[0].ReadTimestamp = "2024-01-01T12:00:00Z"
	model.uiState.SetSearchQuery("read error")
	model.applySearchFilter()
	model.resetCursor()

	require.Len(t, model.filtered, 1)
	assert.Equal(t, 1, model.filtered[0].ID)
}

func TestModelUpdateHandlesQuit(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		uiState: NewUIState(),
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)
	assert.NotNil(t, cmd)

	msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd = model.Update(msg)
	assert.NotNil(t, cmd)

	msg = tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd = model.Update(msg)
	assert.NotNil(t, cmd)
}

func TestModelUpdateHandlesSearchEscape(t *testing.T) {
	model := &Model{uiState: NewUIState()}
	model.uiState.SetSearchMode(true)
	model.uiState.SetSearchQuery("test")

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, cmd := model.Update(msg)
	model = updated.(*Model)

	assert.Nil(t, cmd)
	assert.False(t, model.uiState.IsSearchMode())
	assert.Equal(t, "", model.uiState.GetSearchQuery())
}

func TestModelUpdateHandlesSearchEnter(t *testing.T) {
	model := &Model{uiState: NewUIState()}
	model.uiState.SetSearchMode(true)
	model.uiState.SetSearchQuery("test query")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := model.Update(msg)
	model = updated.(*Model)

	assert.Nil(t, cmd)
	assert.False(t, model.uiState.IsSearchMode())
	// In the new implementation, search query is cleared when exiting search mode
	assert.Equal(t, "", model.uiState.GetSearchQuery())
}

// TestApplySearchFilterGroupedView tests that search filtering works correctly
// in grouped view mode, including tree rebuilding and empty group pruning.
func TestApplySearchFilterGroupedView(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "Error: connection failed", Timestamp: "2024-01-03T10:00:00Z"},
			{ID: 2, Session: "$1", Window: "@1", Pane: "%2", Message: "Warning: low memory", Timestamp: "2024-01-02T10:00:00Z"},
			{ID: 3, Session: "$2", Window: "@1", Pane: "%1", Message: "Error: file not found", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: 4, Session: "$2", Window: "@2", Pane: "%1", Message: "Info: task completed", Timestamp: "2024-01-04T10:00:00Z"},
		},
		expansionState: map[string]bool{},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	// Search for "Error"
	model.uiState.SetSearchQuery("Error")
	model.applySearchFilter()
	model.resetCursor()

	require.Len(t, model.filtered, 2)
	require.NotNil(t, model.treeRoot)
	require.NotEmpty(t, model.visibleNodes)

	// Verify that only error notifications are in filtered list
	assert.Contains(t, model.filtered[0].Message, "Error")
	assert.Contains(t, model.filtered[1].Message, "Error")

	// Verify tree root count matches filtered count
	assert.Equal(t, 2, model.treeRoot.Count)

	// Verify only sessions with matching errors are in the tree
	sessionCount := 0
	for _, node := range model.visibleNodes {
		if node != nil && node.Kind == NodeKindSession {
			sessionCount++
		}
	}
	assert.Equal(t, 2, sessionCount)
}

// TestBuildFilteredTreePrunesEmptyGroups tests that empty groups are removed
// from the tree after filtering.
func TestBuildFilteredTreePrunesEmptyGroups(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "Unique message here", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: 2, Session: "$2", Window: "@1", Pane: "%1", Message: "Different message", Timestamp: "2024-01-02T10:00:00Z"},
		},
		expansionState: map[string]bool{},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	// Search for "Unique"
	model.uiState.SetSearchQuery("Unique")
	model.applySearchFilter()
	model.resetCursor()

	require.Len(t, model.filtered, 1)
	require.NotNil(t, model.treeRoot)

	// Verify tree has only one session (the one with matching notification)
	sessionCount := 0
	var sessionNode *Node
	for _, node := range model.treeRoot.Children {
		if node != nil && node.Kind == NodeKindSession {
			sessionCount++
			sessionNode = node
		}
	}
	assert.Equal(t, 1, sessionCount)
	require.NotNil(t, sessionNode)

	// Verify session count is 1 (only matching notification)
	assert.Equal(t, 1, sessionNode.Count)
}

// TestBuildFilteredTreePreservesExpansionState tests that expansion state
// is preserved across searches when possible.
func TestBuildFilteredTreePreservesExpansionState(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "Test message 1", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: 2, Session: "$1", Window: "@2", Pane: "%1", Message: "Test message 2", Timestamp: "2024-01-02T10:00:00Z"},
			{ID: 3, Session: "$2", Window: "@1", Pane: "%1", Message: "Test message 3", Timestamp: "2024-01-03T10:00:00Z"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	// First search - build initial tree
	model.uiState.SetSearchQuery("")
	model.applySearchFilter()
	model.resetCursor()
	require.NotNil(t, model.treeRoot)

	// Collapse session $2
	sessionNode := findChildByTitle(model.treeRoot, NodeKindSession, "$2")
	require.NotNil(t, sessionNode)
	sessionNode.Expanded = false
	model.updateExpansionState(sessionNode, false)

	// Second search - should preserve expansion state
	model.uiState.SetSearchQuery("message")
	model.applySearchFilter()
	model.resetCursor()

	// Find session $2 again in new tree
	sessionNode = findChildByTitle(model.treeRoot, NodeKindSession, "$2")
	require.NotNil(t, sessionNode)
	assert.False(t, sessionNode.Expanded, "expansion state should be preserved")
}

// TestBuildFilteredTreeHandlesNoMatches tests the edge case where search
// returns no matches.
func TestBuildFilteredTreeHandlesNoMatches(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "Test message", Timestamp: "2024-01-01T10:00:00Z"},
		},
		expansionState: map[string]bool{},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	// Search for something that doesn't exist
	model.uiState.SetSearchQuery("nonexistent")
	model.applySearchFilter()
	model.resetCursor()

	require.Empty(t, model.filtered)
	assert.Nil(t, model.treeRoot)
	assert.Empty(t, model.visibleNodes)

	// Verify viewport shows "No notifications found"
	view := model.uiState.GetViewport().View()
	assert.Contains(t, view, "No notifications found")
}

// TestBuildFilteredTreeWithEmptyQuery tests that empty query
// shows all notifications.
func TestBuildFilteredTreeWithEmptyQuery(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "First", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: 2, Session: "$1", Window: "@2", Pane: "%1", Message: "Second", Timestamp: "2024-01-02T10:00:00Z"},
		},
		expansionState: map[string]bool{},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	// Empty search
	model.uiState.SetSearchQuery("")
	model.applySearchFilter()
	model.resetCursor()

	require.Len(t, model.filtered, 2)
	require.NotNil(t, model.treeRoot)
	assert.Equal(t, 2, model.treeRoot.Count)
}

// TestBuildFilteredTreeGroupCounts tests that group counts reflect
// only matching notifications.
func TestBuildFilteredTreeGroupCounts(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "Error: connection failed", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: 2, Session: "$1", Window: "@1", Pane: "%1", Message: "Warning: low memory", Timestamp: "2024-01-02T10:00:00Z"},
			{ID: 3, Session: "$1", Window: "@1", Pane: "%2", Message: "Error: timeout", Timestamp: "2024-01-03T10:00:00Z"},
		},
		expansionState: map[string]bool{},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	// Search for "Error"
	model.uiState.SetSearchQuery("Error")
	model.applySearchFilter()
	model.resetCursor()

	require.Len(t, model.filtered, 2)
	require.NotNil(t, model.treeRoot)

	// Verify root count
	assert.Equal(t, 2, model.treeRoot.Count)

	// Verify session count
	sessionNode := findChildByTitle(model.treeRoot, NodeKindSession, "$1")
	require.NotNil(t, sessionNode)
	assert.Equal(t, 2, sessionNode.Count)

	// Verify window count
	windowNode := findChildByTitle(sessionNode, NodeKindWindow, "@1")
	require.NotNil(t, windowNode)
	assert.Equal(t, 2, windowNode.Count)

	// Pane %1 should have 1 error, Pane %2 should have 1 error
	pane1 := findChildByTitle(windowNode, NodeKindPane, "%1")
	pane2 := findChildByTitle(windowNode, NodeKindPane, "%2")
	require.NotNil(t, pane1)
	require.NotNil(t, pane2)
	assert.Equal(t, 1, pane1.Count)
	assert.Equal(t, 1, pane2.Count)
}

func TestModelUpdateHandlesCommandMode(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		uiState: NewUIState(),
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.True(t, model.uiState.IsCommandMode())
	assert.Equal(t, "", model.uiState.GetCommandQuery())

	model.uiState.SetCommandMode(true)
	model.uiState.SetCommandQuery("test")

	msg = tea.KeyMsg{Type: tea.KeyEsc}
	updated, cmd := model.Update(msg)
	model = updated.(*Model)

	assert.Nil(t, cmd)
	assert.False(t, model.uiState.IsCommandMode())
	assert.Equal(t, "", model.uiState.GetCommandQuery())

	model.uiState.SetCommandMode(true)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, "q", model.uiState.GetCommandQuery())

	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, "", model.uiState.GetCommandQuery())

	model.uiState.SetCommandMode(true)
	model.uiState.SetCommandQuery("q")
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd = model.Update(msg)

	assert.NotNil(t, cmd)
}

func TestModelUpdateHandlesWindowSize(t *testing.T) {
	model := &Model{
		uiState: NewUIState(),
	}

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 100, model.uiState.GetWidth())
	assert.Equal(t, 30, model.uiState.GetHeight())
	assert.Equal(t, 28, model.uiState.GetViewport().Height)
}

func TestModelViewRendersContent(t *testing.T) {
	stubSessionFetchers(t)

	model := &Model{
		uiState: NewUIState(),
		notifications: []notification.Notification{
			{ID: 1, Message: "Test notification", Timestamp: "2024-01-01T12:00:00Z", Level: "info", State: "active"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "Test notification", Timestamp: "2024-01-01T12:00:00Z", Level: "info", State: "active"},
		},
	}
	model.uiState.SetCursor(0)
	model.uiState.SetWidth(84)
	model.uiState.SetHeight(24)
	model.updateViewportContent()

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "TYPE")
	assert.Contains(t, view, "STATUS")
	assert.Contains(t, view, "SESSION")
	assert.Contains(t, view, "MESSAGE")
	assert.Contains(t, view, "PANE")
	assert.Contains(t, view, "AGE")
	assert.Contains(t, view, "Test notification")
	assert.Contains(t, view, "j/k: move")
	assert.Contains(t, view, "q: quit")
}

func TestModelViewWithNoNotifications(t *testing.T) {
	model := &Model{
		uiState:       NewUIState(),
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
	}
	model.uiState.SetWidth(80)
	model.uiState.SetHeight(24)
	model.updateViewportContent()

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "No notifications found")
}

func TestUpdateViewportContentGroupedViewWithEmptyTree(t *testing.T) {
	model := &Model{
		uiState:       NewUIState(),
		viewMode:      viewModeGrouped,
		notifications: []notification.Notification{},
	}

	model.applySearchFilter()
	model.resetCursor()

	assert.Contains(t, model.uiState.GetViewport().View(), "No notifications found")
}

func TestUpdateViewportContentGroupedViewRendersMixedNodes(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One", Level: "info", State: "active"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()
	require.NotEmpty(t, model.visibleNodes)
	model.uiState.SetCursor(0)
	model.updateViewportContent()

	content := model.uiState.GetViewport().View()
	groupNode := model.visibleNodes[0]
	require.NotNil(t, groupNode)

	expectedGroupRow := render.RenderGroupRow(render.GroupRow{
		Node: &render.GroupNode{
			Title:    groupNode.Title,
			Display:  groupNode.Display,
			Expanded: groupNode.Expanded,
			Count:    groupNode.Count,
		},
		Selected: true,
		Level:    getTreeLevel(groupNode),
		Width:    model.uiState.GetWidth(),
	})
	assert.Contains(t, content, expectedGroupRow)

	var leafNode *Node
	var leafIndex int
	for idx, node := range model.visibleNodes {
		if node != nil && node.Kind == NodeKindNotification && node.Notification != nil {
			leafNode = node
			leafIndex = idx
			break
		}
	}
	require.NotNil(t, leafNode)

	expectedLeafRow := render.Row(render.RowState{
		Notification: *leafNode.Notification,
		SessionName:  model.getSessionName(leafNode.Notification.Session),
		Width:        model.uiState.GetWidth(),
		Selected:     leafIndex == model.uiState.GetCursor(),
		Now:          time.Time{},
	})
	assert.Contains(t, content, expectedLeafRow)

	groupIndex := strings.Index(content, expectedGroupRow)
	leafRowIndex := strings.Index(content, expectedLeafRow)
	require.NotEqual(t, -1, groupIndex)
	require.NotEqual(t, -1, leafRowIndex)
	assert.Less(t, groupIndex, leafRowIndex)
}

func TestUpdateViewportContentGroupedViewHighlightsLeafRow(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "First", Level: "info", State: "active"},
			{ID: 2, Session: "$1", Window: "@1", Pane: "%1", Message: "Second", Level: "info", State: "active"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	var leafNode *Node
	var leafIndex int
	var groupNode *Node
	for idx, node := range model.visibleNodes {
		if node == nil {
			continue
		}
		if groupNode == nil && isGroupNode(node) {
			groupNode = node
		}
		if node.Kind == NodeKindNotification && node.Notification != nil {
			leafNode = node
			leafIndex = idx
			break
		}
	}
	require.NotNil(t, leafNode)
	require.NotNil(t, groupNode)
	model.uiState.SetCursor(leafIndex)
	model.updateViewportContent()

	content := model.uiState.GetViewport().View()
	expectedLeafRow := render.Row(render.RowState{
		Notification: *leafNode.Notification,
		SessionName:  model.getSessionName(leafNode.Notification.Session),
		Width:        model.uiState.GetWidth(),
		Selected:     true,
		Now:          time.Time{},
	})
	assert.Contains(t, content, expectedLeafRow)

	expectedGroupRow := render.RenderGroupRow(render.GroupRow{
		Node: &render.GroupNode{
			Title:    groupNode.Title,
			Display:  groupNode.Display,
			Expanded: groupNode.Expanded,
			Count:    groupNode.Count,
		},
		Selected: false,
		Level:    getTreeLevel(groupNode),
		Width:    model.uiState.GetWidth(),
	})
	assert.Contains(t, content, expectedGroupRow)
}

func TestHandleDismiss(t *testing.T) {
	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	id, err := storage.AddNotification("Test message", "2024-01-01T12:00:00Z", "", "", "", "1234", "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)

	model, err := NewModel(mockClient)
	require.NoError(t, err)
	require.Len(t, model.filtered, 1)

	cmd := model.handleDismiss()

	assert.Nil(t, cmd)

	model, err = NewModel(mockClient)
	require.NoError(t, err)
	assert.Empty(t, model.filtered)
}

func TestMarkSelectedRead(t *testing.T) {
	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	id, err := storage.AddNotification("Test message", "2024-01-01T12:00:00Z", "", "", "", "", "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)

	model, err := NewModel(mockClient)
	require.NoError(t, err)
	require.Len(t, model.filtered, 1)

	cmd := model.markSelectedRead()
	assert.Nil(t, cmd)

	lines, err := storage.ListNotifications("active", "", "", "", "", "", "", "")
	require.NoError(t, err)

	parts := strings.Split(lines, "\n")
	require.Len(t, parts, 1)
	loaded, err := notification.ParseNotification(parts[0])
	require.NoError(t, err)
	assert.True(t, loaded.IsRead())
	assert.True(t, model.filtered[0].IsRead())
}

func TestMarkSelectedUnread(t *testing.T) {
	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	id, err := storage.AddNotification("Test message", "2024-01-01T12:00:00Z", "", "", "", "", "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	require.NoError(t, storage.MarkNotificationRead(id))

	model, err := NewModel(mockClient)
	require.NoError(t, err)
	require.Len(t, model.filtered, 1)
	require.True(t, model.filtered[0].IsRead())

	cmd := model.markSelectedUnread()
	assert.Nil(t, cmd)

	lines, err := storage.ListNotifications("active", "", "", "", "", "", "", "")
	require.NoError(t, err)

	parts := strings.Split(lines, "\n")
	require.Len(t, parts, 1)
	loaded, err := notification.ParseNotification(parts[0])
	require.NoError(t, err)
	assert.False(t, loaded.IsRead())
	assert.False(t, model.filtered[0].IsRead())
}

func TestHandleDismissGroupedViewUsesVisibleNodes(t *testing.T) {
	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	_, err := storage.AddNotification("B msg", "2024-02-02T12:00:00Z", "b", "@1", "%1", "", "info")
	require.NoError(t, err)
	_, err = storage.AddNotification("A msg", "2024-01-01T12:00:00Z", "a", "@1", "%1", "", "info")
	require.NoError(t, err)

	model, err := NewModel(mockClient)
	require.NoError(t, err)
	model.viewMode = viewModeGrouped
	model.applySearchFilter()
	model.resetCursor()
	cursorIndex := -1
	for idx, node := range model.visibleNodes {
		if node == nil || node.Kind != NodeKindNotification || node.Notification == nil {
			continue
		}
		if node.Notification.Session == "a" {
			cursorIndex = idx
			break
		}
	}
	require.NotEqual(t, -1, cursorIndex)
	model.uiState.SetCursor(cursorIndex)

	cmd := model.handleDismiss()

	assert.Nil(t, cmd)

	lines, err := storage.ListNotifications("active", "", "", "", "", "", "", "")
	require.NoError(t, err)

	remainingSessions := []string{}
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, err := notification.ParseNotification(line)
		require.NoError(t, err)
		remainingSessions = append(remainingSessions, notif.Session)
	}

	require.Len(t, remainingSessions, 1)
	assert.Equal(t, "b", remainingSessions[0])
}

func TestHandleDismissWithEmptyList(t *testing.T) {
	model := &Model{
		uiState:       NewUIState(),
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
	}
	model.uiState.SetCursor(0)

	cmd := model.handleDismiss()

	assert.Nil(t, cmd)
}

func TestHandleJumpWithMissingContext(t *testing.T) {
	model := &Model{
		uiState: NewUIState(),
		notifications: []notification.Notification{
			{ID: 1, Message: "Test"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "Test"},
		},
	}
	model.uiState.SetCursor(0)

	cmd := model.handleJump()
	assert.Nil(t, cmd)

	model.filtered[0].Session = "$1"
	cmd = model.handleJump()
	assert.Nil(t, cmd)

	model.filtered[0].Window = "@2"
	model.filtered[0].Pane = ""
	cmd = model.handleJump()
	assert.Nil(t, cmd)
}

func TestHandleJumpGroupedViewUsesVisibleNodes(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "b", Window: "@1", Pane: "%1", Message: "B"},
			{ID: 2, Session: "a", Window: "", Pane: "%1", Message: "A"},
		},
		ensureTmuxRunning: func() bool {
			t.Fatal("ensureTmuxRunning should not be called")
			return true
		},
		jumpToPane: func(sessionID, windowID, paneID string) bool {
			t.Fatal("jumpToPane should not be called")
			return true
		},
	}

	model.applySearchFilter()
	model.resetCursor()
	model.uiState.SetCursor(0)

	cmd := model.handleJump()

	assert.Nil(t, cmd)
}

func TestHandleJumpWithEmptyList(t *testing.T) {
	model := &Model{
		uiState:       NewUIState(),
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
	}
	model.uiState.SetCursor(0)

	cmd := model.handleJump()

	assert.Nil(t, cmd)
}

func TestModelUpdateHandlesDismissKey(t *testing.T) {
	model := &Model{
		uiState:       NewUIState(),
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
	}
	model.uiState.SetCursor(0)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := model.Update(msg)

	assert.NotNil(t, updated.(*Model))
}

func TestModelUpdateHandlesZaToggleFold(t *testing.T) {
	model := &Model{
		uiState:  NewUIState(),
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.GetViewport().Width = 80

	model.applySearchFilter()
	model.resetCursor()

	var groupNode *Node
	groupIndex := -1
	for idx, node := range model.visibleNodes {
		if node != nil && isGroupNode(node) {
			groupNode = node
			groupIndex = idx
			break
		}
	}
	require.NotNil(t, groupNode)
	require.NotEqual(t, -1, groupIndex)
	model.uiState.SetCursor(groupIndex)

	require.True(t, groupNode.Expanded)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}
	updated, _ := model.Update(msg)
	require.NotNil(t, updated.(*Model))

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updated, _ = model.Update(msg)
	require.NotNil(t, updated.(*Model))

	assert.False(t, groupNode.Expanded)
}

func TestModelUpdateHandlesEnterKey(t *testing.T) {
	model := &Model{
		uiState:       NewUIState(),
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
	}
	model.uiState.SetCursor(0)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)

	assert.NotNil(t, updated.(*Model))
}

func TestGetSessionNameCachesFetcher(t *testing.T) {
	model := &Model{
		uiState: NewUIState(),
		sessionNames: map[string]string{
			"$1": "$1-name",
		},
	}

	name := model.getSessionName("$1")
	assert.Equal(t, "$1-name", name)

	// Call again - should return cached value
	name = model.getSessionName("$1")
	assert.Equal(t, "$1-name", name)
}

func TestToState(t *testing.T) {
	tests := []struct {
		name  string
		model *Model
		want  settings.TUIState
	}{
		{
			name:  "empty model",
			model: &Model{},
			want: settings.TUIState{
				DefaultExpandLevelSet: true,
			},
		},
		{
			name: "model with settings",
			model: &Model{
				sortBy:    settings.SortByLevel,
				sortOrder: settings.SortOrderAsc,
				columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				viewMode:           settings.ViewModeDetailed,
				groupBy:            settings.GroupBySession,
				defaultExpandLevel: 2,
				expansionState: map[string]bool{
					"session:$1": true,
				},
			},
			want: settings.TUIState{
				SortBy:    settings.SortByLevel,
				SortOrder: settings.SortOrderAsc,
				Columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				Filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode:              settings.ViewModeDetailed,
				GroupBy:               settings.GroupBySession,
				DefaultExpandLevel:    2,
				DefaultExpandLevelSet: true,
				ExpansionState: map[string]bool{
					"session:$1": true,
				},
			},
		},
		{
			name: "model with partial settings",
			model: &Model{
				sortBy:   settings.SortByTimestamp,
				viewMode: settings.ViewModeCompact,
				groupBy:  settings.GroupByNone,
			},
			want: settings.TUIState{
				SortBy:                settings.SortByTimestamp,
				ViewMode:              settings.ViewModeCompact,
				GroupBy:               settings.GroupByNone,
				DefaultExpandLevelSet: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.ToState()

			assert.Equal(t, tt.want.SortBy, got.SortBy)
			assert.Equal(t, tt.want.SortOrder, got.SortOrder)
			assert.Equal(t, tt.want.Columns, got.Columns)
			assert.Equal(t, tt.want.Filters, got.Filters)
			assert.Equal(t, tt.want.ViewMode, got.ViewMode)
			assert.Equal(t, tt.want.GroupBy, got.GroupBy)
			assert.Equal(t, tt.want.DefaultExpandLevel, got.DefaultExpandLevel)
			assert.Equal(t, tt.want.DefaultExpandLevelSet, got.DefaultExpandLevelSet)
			assert.Equal(t, tt.want.ExpansionState, got.ExpansionState)
		})
	}
}

func TestFromState(t *testing.T) {
	tests := []struct {
		name     string
		model    *Model
		state    settings.TUIState
		wantErr  bool
		verifyFn func(*testing.T, *Model)
	}{
		{
			name:    "empty state - no changes",
			model:   &Model{uiState: NewUIState()},
			state:   settings.TUIState{},
			wantErr: false,
			verifyFn: func(t *testing.T, m *Model) {
				assert.Equal(t, "", m.sortBy)
				assert.Equal(t, "", m.sortOrder)
				assert.Empty(t, m.columns)
				assert.Equal(t, "", m.viewMode)
				assert.Equal(t, "", m.groupBy)
				assert.Equal(t, 0, m.defaultExpandLevel)
				assert.Nil(t, m.expansionState)
				assert.Equal(t, settings.Filter{}, m.filters)
			},
		},
		{
			name:  "full state - all fields set",
			model: &Model{uiState: NewUIState()},
			state: settings.TUIState{
				SortBy:    settings.SortByLevel,
				SortOrder: settings.SortOrderAsc,
				Columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				Filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode:              settings.ViewModeDetailed,
				GroupBy:               settings.GroupByWindow,
				DefaultExpandLevel:    2,
				DefaultExpandLevelSet: true,
				ExpansionState: map[string]bool{
					"window:@1": true,
				},
			},
			wantErr: false,
			verifyFn: func(t *testing.T, m *Model) {
				assert.Equal(t, settings.SortByLevel, m.sortBy)
				assert.Equal(t, settings.SortOrderAsc, m.sortOrder)
				assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel}, m.columns)
				assert.Equal(t, settings.ViewModeDetailed, m.viewMode)
				assert.Equal(t, settings.GroupByWindow, m.groupBy)
				assert.Equal(t, 2, m.defaultExpandLevel)
				assert.Equal(t, map[string]bool{"window:@1": true}, m.expansionState)
				assert.Equal(t, settings.LevelFilterWarning, m.filters.Level)
				assert.Equal(t, settings.StateFilterActive, m.filters.State)
				assert.Equal(t, "my-session", m.filters.Session)
				assert.Equal(t, "@1", m.filters.Window)
				assert.Equal(t, "%1", m.filters.Pane)
			},
		},
		{
			name: "partial state - only some fields set",
			model: &Model{
				uiState:   NewUIState(),
				sortBy:    settings.SortByTimestamp,
				sortOrder: settings.SortOrderDesc,
				columns:   []string{settings.ColumnID},
				filters: settings.Filter{
					Level: settings.LevelFilterError,
				},
				viewMode:           settings.ViewModeCompact,
				groupBy:            settings.GroupBySession,
				defaultExpandLevel: 3,
			},
			state: settings.TUIState{
				SortBy:                settings.SortByLevel,
				Columns:               []string{settings.ColumnID, settings.ColumnMessage},
				DefaultExpandLevel:    0,
				DefaultExpandLevelSet: true,
			},
			wantErr: false,
			verifyFn: func(t *testing.T, m *Model) {
				assert.Equal(t, settings.SortByLevel, m.sortBy)
				assert.Equal(t, settings.SortOrderDesc, m.sortOrder)
				assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage}, m.columns)
				assert.Equal(t, settings.LevelFilterError, m.filters.Level)
				assert.Equal(t, settings.ViewModeCompact, m.viewMode)
				assert.Equal(t, settings.GroupBySession, m.groupBy)
				assert.Equal(t, 0, m.defaultExpandLevel)
			},
		},
		{
			name: "partial filters - only some filter fields set",
			model: &Model{
				uiState: NewUIState(),
				filters: settings.Filter{
					Level:   settings.LevelFilterError,
					State:   settings.StateFilterActive,
					Session: "old-session",
					Window:  "old-session",
					Pane:    "old-session",
				},
				groupBy:            settings.GroupByPane,
				defaultExpandLevel: 2,
				expansionState: map[string]bool{
					"pane:%1": true,
				},
			},
			state: settings.TUIState{
				Filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					Session: "new-session",
				},
				ExpansionState: map[string]bool{},
			},
			wantErr: false,
			verifyFn: func(t *testing.T, m *Model) {
				assert.Equal(t, settings.LevelFilterWarning, m.filters.Level)
				assert.Equal(t, settings.StateFilterActive, m.filters.State)
				assert.Equal(t, "new-session", m.filters.Session)
				// Fields not set in state preserve their old values
				assert.Equal(t, "old-session", m.filters.Window)
				assert.Equal(t, "old-session", m.filters.Pane)
				assert.Equal(t, settings.GroupByPane, m.groupBy)
				assert.Equal(t, 2, m.defaultExpandLevel)
				assert.Equal(t, map[string]bool{}, m.expansionState)
			},
		},
		{
			name:  "invalid groupBy value",
			model: &Model{uiState: NewUIState()},
			state: settings.TUIState{
				GroupBy: "invalid",
			},
			wantErr: true,
		},
		{
			name:  "invalid defaultExpandLevel value",
			model: &Model{uiState: NewUIState()},
			state: settings.TUIState{
				DefaultExpandLevel:    999,
				DefaultExpandLevelSet: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.FromState(tt.state)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.verifyFn != nil {
				tt.verifyFn(t, tt.model)
			}
		})
	}
}

func TestModelWithNegativeDimensions(t *testing.T) {
	// Test that negative dimensions are handled correctly
	model := &Model{
		uiState: NewUIState(),
	}
	model.uiState.SetWidth(-10)
	model.uiState.SetHeight(-5)

	// Should clamp to default values
	assert.Equal(t, defaultViewportWidth, model.uiState.GetWidth())
	assert.Equal(t, defaultViewportHeight, model.uiState.GetHeight())
}

func TestModelCursorBoundsWithEmptyList(t *testing.T) {
	model := &Model{
		uiState:       NewUIState(),
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
	}
	model.uiState.SetCursor(5)

	// Adjust cursor should handle empty list
	model.adjustCursorBounds()
	assert.Equal(t, 0, model.uiState.GetCursor())
}

func TestModelCursorBoundsWithSingleItem(t *testing.T) {
	model := &Model{
		uiState: NewUIState(),
		notifications: []notification.Notification{
			{ID: 1, Message: "Test"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "Test"},
		},
	}

	// Cursor at 0 should work
	model.uiState.SetCursor(0)
	model.adjustCursorBounds()
	assert.Equal(t, 0, model.uiState.GetCursor())

	// Cursor beyond bounds should be adjusted
	model.uiState.SetCursor(10)
	model.adjustCursorBounds()
	assert.Equal(t, 0, model.uiState.GetCursor())

	// Negative cursor should be adjusted
	model.uiState.SetCursor(-5)
	model.adjustCursorBounds()
	assert.Equal(t, 0, model.uiState.GetCursor())
}

func TestModelCursorBoundsWithMultipleItems(t *testing.T) {
	model := &Model{
		uiState: NewUIState(),
		notifications: []notification.Notification{
			{ID: 1, Message: "First"},
			{ID: 2, Message: "Second"},
			{ID: 3, Message: "Third"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "First"},
			{ID: 2, Message: "Second"},
			{ID: 3, Message: "Third"},
		},
	}

	// Cursor at valid position should work
	model.uiState.SetCursor(1)
	model.adjustCursorBounds()
	assert.Equal(t, 1, model.uiState.GetCursor())

	// Cursor at max valid position should work
	model.uiState.SetCursor(2)
	model.adjustCursorBounds()
	assert.Equal(t, 2, model.uiState.GetCursor())

	// Cursor beyond bounds should be adjusted
	model.uiState.SetCursor(10)
	model.adjustCursorBounds()
	assert.Equal(t, 2, model.uiState.GetCursor())

	// Negative cursor should be adjusted
	model.uiState.SetCursor(-1)
	model.adjustCursorBounds()
	assert.Equal(t, 0, model.uiState.GetCursor())
}

func TestModelViewportEdgeConditions(t *testing.T) {
	model := &Model{
		uiState: NewUIState(),
		notifications: []notification.Notification{
			{ID: 1, Message: "First"},
			{ID: 2, Message: "Second"},
			{ID: 3, Message: "Third"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "First"},
			{ID: 2, Message: "Second"},
			{ID: 3, Message: "Third"},
		},
	}
	model.uiState.SetWidth(80)
	model.uiState.SetHeight(24)
	model.updateViewportContent()

	// Test viewport top edge
	model.uiState.SetCursor(0)
	model.uiState.EnsureCursorVisible(3)
	assert.Equal(t, 0, model.uiState.GetCursor())

	// Test viewport bottom edge
	model.uiState.SetCursor(2)
	model.uiState.EnsureCursorVisible(3)
	assert.Equal(t, 2, model.uiState.GetCursor())

	// Test with zero items
	model.filtered = []notification.Notification{}
	model.uiState.EnsureCursorVisible(0)
	// Should not panic
}
