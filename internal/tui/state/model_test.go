package state

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
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

	return mockClient
}

func TestNewModelInitialState(t *testing.T) {
	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	model, err := NewModel(mockClient)

	require.NoError(t, err)
	assert.Equal(t, 0, model.width)
	assert.Equal(t, 0, model.height)
	assert.Equal(t, 0, model.cursor)
	assert.False(t, model.searchMode)
	assert.Equal(t, "", model.searchQuery)
	assert.Empty(t, model.notifications)
	assert.Empty(t, model.filtered)
	assert.NotNil(t, model.expansionState)
	assert.Empty(t, model.expansionState)
	assert.Nil(t, model.treeRoot)
	assert.Empty(t, model.visibleNodes)
}

func TestModelGroupedModeBuildsVisibleNodes(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
			{ID: 2, Session: "$2", Window: "@1", Pane: "%2", Message: "Two"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

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

func TestModelSwitchesViewModes(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()
	require.NotNil(t, model.treeRoot)
	require.NotEmpty(t, model.visibleNodes)

	model.viewMode = "flat"
	model.applySearchFilter()
	assert.Nil(t, model.treeRoot)
	assert.Empty(t, model.visibleNodes)
}

func TestToggleNodeExpansionGroupedView(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

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
	model.cursor = groupIndex

	require.True(t, groupNode.Expanded)

	handled := model.toggleNodeExpansion()
	require.True(t, handled)
	assert.False(t, groupNode.Expanded)
	assert.Len(t, model.visibleNodes, 1)
	assert.Equal(t, 0, model.cursor)

	handled = model.toggleNodeExpansion()
	require.True(t, handled)
	assert.True(t, groupNode.Expanded)
	assert.Greater(t, len(model.visibleNodes), 1)
}

func TestToggleFoldTogglesGroupNode(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

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
	model.cursor = groupIndex

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
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

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
	model.cursor = paneIndex

	require.True(t, paneNode.Expanded)

	model.toggleFold()
	assert.False(t, paneNode.Expanded)

	model.toggleFold()
	assert.True(t, paneNode.Expanded)
}

func TestCollapseNodeMovesCursorToParent(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

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

	model.cursor = leafIndex
	model.collapseNode(paneNode)

	paneIndex := indexOfNode(model.visibleNodes, paneNode)
	require.NotEqual(t, -1, paneIndex)
	assert.Equal(t, paneIndex, model.cursor)
}

func TestToggleNodeExpansionIgnoresLeafNodes(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

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

	model.cursor = leafIndex
	visibleBefore := len(model.visibleNodes)

	handled := model.toggleNodeExpansion()

	assert.False(t, handled)
	assert.Len(t, model.visibleNodes, visibleBefore)
}

func TestToggleFoldIgnoresLeafNodes(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

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

	model.cursor = leafIndex
	visibleBefore := len(model.visibleNodes)

	model.toggleFold()

	assert.Len(t, model.visibleNodes, visibleBefore)
}

func TestToggleFoldExpandsDefaultWhenAllCollapsed(t *testing.T) {
	model := &Model{
		viewMode:           viewModeGrouped,
		defaultExpandLevel: 2,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

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
	model.cursor = 0

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
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "b", Window: "@1", Pane: "%1", Message: "B"},
			{ID: 2, Session: "a", Window: "@1", Pane: "%1", Message: "A"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()
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
	model.cursor = cursorIndex

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
		cursor:   0,
		width:    80,
		viewport: viewport.New(80, 22),
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 0, model.cursor)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 1, model.cursor)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 2, model.cursor)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 2, model.cursor)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 1, model.cursor)
}

func TestModelUpdateHandlesSearch(t *testing.T) {
	stubSessionFetchers(t)

	model := &Model{
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
		width:    80,
		viewport: viewport.New(80, 22),
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.True(t, model.searchMode)
	assert.Equal(t, "", model.searchQuery)
	assert.Equal(t, 0, model.cursor)
	assert.Len(t, model.filtered, 3)

	model.searchQuery = "error"
	model.applySearchFilter()

	require.Len(t, model.filtered, 2)
	assert.True(t, strings.Contains(model.filtered[0].Message, "Error"))

	model.searchQuery = "not found"
	model.applySearchFilter()

	require.Len(t, model.filtered, 1)
	assert.True(t, strings.Contains(strings.ToLower(model.filtered[0].Message), "not found"))

	model.searchQuery = ""
	model.applySearchFilter()

	assert.Len(t, model.filtered, 3)
}

func TestApplySearchFilterReadStatus(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{
			{ID: 1, Message: "Alpha", ReadTimestamp: "2024-01-01T12:00:00Z"},
			{ID: 2, Message: "Beta"},
		},
		filtered: []notification.Notification{},
		width:    80,
		viewport: viewport.New(80, 22),
	}

	model.searchQuery = "read"
	model.applySearchFilter()
	require.Len(t, model.filtered, 1)
	assert.True(t, model.filtered[0].IsRead())

	model.searchQuery = "unread"
	model.applySearchFilter()
	require.Len(t, model.filtered, 1)
	assert.False(t, model.filtered[0].IsRead())

	model.searchQuery = "unread beta"
	model.applySearchFilter()
	require.Len(t, model.filtered, 1)
	assert.Equal(t, "Beta", model.filtered[0].Message)

	model.searchQuery = "read alpha"
	model.applySearchFilter()
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
		notifications:  notifications,
		filtered:       []notification.Notification{},
		searchQuery:    "test",
		searchProvider: mockProvider,
		width:          80,
		viewport:       viewport.New(80, 22),
	}

	model.applySearchFilter()

	require.Len(t, model.filtered, 2)
	assert.Equal(t, notifications[0].ID, model.filtered[0].ID)
	assert.Equal(t, notifications[2].ID, model.filtered[1].ID)

	mockProvider.AssertExpectations(t)
}

// TestApplySearchFilterUsesDefaultTokenProvider tests that applySearchFilter
// falls back to TokenProvider when no custom provider is set.
func TestApplySearchFilterUsesDefaultTokenProvider(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{
			{ID: 1, Message: "Error: file not found", Level: "error"},
			{ID: 2, Message: "Warning: low memory", Level: "warning"},
			{ID: 3, Message: "Error: connection failed", Level: "error"},
		},
		filtered: []notification.Notification{},
		width:    80,
		viewport: viewport.New(80, 22),
	}

	// No custom searchProvider set, should use default TokenProvider
	assert.Nil(t, model.searchProvider)

	// Test case-insensitive matching (default behavior)
	model.searchQuery = "error"
	model.applySearchFilter()

	require.Len(t, model.filtered, 2)
	assert.Contains(t, model.filtered[0].Message, "Error")
	assert.Contains(t, model.filtered[1].Message, "Error")

	// Test token-based matching (all tokens must match)
	model.searchQuery = "error file"
	model.applySearchFilter()

	require.Len(t, model.filtered, 1)
	assert.Contains(t, model.filtered[0].Message, "file not found")

	// Test read/unread filtering
	model.notifications[0].ReadTimestamp = "2024-01-01T12:00:00Z"
	model.searchQuery = "read error"
	model.applySearchFilter()

	require.Len(t, model.filtered, 1)
	assert.Equal(t, 1, model.filtered[0].ID)
}

func TestModelUpdateHandlesQuit(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{}

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
	model := &Model{searchMode: true, searchQuery: "test"}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, cmd := model.Update(msg)
	model = updated.(*Model)

	assert.Nil(t, cmd)
	assert.False(t, model.searchMode)
	assert.Equal(t, "", model.searchQuery)
}

// TestApplySearchFilterGroupedView tests that search filtering works correctly
// in grouped view mode, including tree rebuilding and empty group pruning.
func TestApplySearchFilterGroupedView(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "Error: connection failed", Timestamp: "2024-01-03T10:00:00Z"},
			{ID: 2, Session: "$1", Window: "@1", Pane: "%2", Message: "Warning: low memory", Timestamp: "2024-01-02T10:00:00Z"},
			{ID: 3, Session: "$2", Window: "@1", Pane: "%1", Message: "Error: file not found", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: 4, Session: "$2", Window: "@2", Pane: "%1", Message: "Info: task completed", Timestamp: "2024-01-04T10:00:00Z"},
		},
		viewport:       viewport.New(80, 22),
		width:          80,
		expansionState: map[string]bool{},
	}

	// Search for "Error"
	model.searchQuery = "Error"
	model.applySearchFilter()

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
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "Unique message here", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: 2, Session: "$2", Window: "@1", Pane: "%1", Message: "Different message", Timestamp: "2024-01-02T10:00:00Z"},
		},
		viewport:       viewport.New(80, 22),
		width:          80,
		expansionState: map[string]bool{},
	}

	// Search for "Unique"
	model.searchQuery = "Unique"
	model.applySearchFilter()

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
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "Test message 1", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: 2, Session: "$1", Window: "@2", Pane: "%1", Message: "Test message 2", Timestamp: "2024-01-02T10:00:00Z"},
			{ID: 3, Session: "$2", Window: "@1", Pane: "%1", Message: "Test message 3", Timestamp: "2024-01-03T10:00:00Z"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	// First search - build initial tree
	model.searchQuery = ""
	model.applySearchFilter()
	require.NotNil(t, model.treeRoot)

	// Collapse session $2
	sessionNode := findChildByTitle(model.treeRoot, NodeKindSession, "$2")
	require.NotNil(t, sessionNode)
	sessionNode.Expanded = false
	model.updateExpansionState(sessionNode, false)

	// Second search - should preserve expansion state
	model.searchQuery = "message"
	model.applySearchFilter()

	// Find session $2 again in new tree
	sessionNode = findChildByTitle(model.treeRoot, NodeKindSession, "$2")
	require.NotNil(t, sessionNode)
	assert.False(t, sessionNode.Expanded, "expansion state should be preserved")
}

// TestBuildFilteredTreeHandlesNoMatches tests the edge case where search
// returns no matches.
func TestBuildFilteredTreeHandlesNoMatches(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "Test message", Timestamp: "2024-01-01T10:00:00Z"},
		},
		viewport:       viewport.New(80, 22),
		width:          80,
		expansionState: map[string]bool{},
	}

	// Search for something that doesn't exist
	model.searchQuery = "nonexistent"
	model.applySearchFilter()

	require.Empty(t, model.filtered)
	assert.Nil(t, model.treeRoot)
	assert.Empty(t, model.visibleNodes)

	// Verify viewport shows "No notifications found"
	view := model.viewport.View()
	assert.Contains(t, view, "No notifications found")
}

// TestBuildFilteredTreeWithEmptyQuery tests that empty query
// shows all notifications.
func TestBuildFilteredTreeWithEmptyQuery(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "First", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: 2, Session: "$1", Window: "@2", Pane: "%1", Message: "Second", Timestamp: "2024-01-02T10:00:00Z"},
		},
		viewport:       viewport.New(80, 22),
		width:          80,
		expansionState: map[string]bool{},
	}

	// Empty search
	model.searchQuery = ""
	model.applySearchFilter()

	require.Len(t, model.filtered, 2)
	require.NotNil(t, model.treeRoot)
	assert.Equal(t, 2, model.treeRoot.Count)
}

// TestBuildFilteredTreeGroupCounts tests that group counts reflect
// only matching notifications.
func TestBuildFilteredTreeGroupCounts(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "Error: connection failed", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: 2, Session: "$1", Window: "@1", Pane: "%1", Message: "Warning: low memory", Timestamp: "2024-01-02T10:00:00Z"},
			{ID: 3, Session: "$1", Window: "@1", Pane: "%2", Message: "Error: timeout", Timestamp: "2024-01-03T10:00:00Z"},
		},
		viewport:       viewport.New(80, 22),
		width:          80,
		expansionState: map[string]bool{},
	}

	// Search for "Error"
	model.searchQuery = "Error"
	model.applySearchFilter()

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

	model := &Model{}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.True(t, model.commandMode)
	assert.Equal(t, "", model.commandQuery)

	model.commandMode = true
	model.commandQuery = "test"

	msg = tea.KeyMsg{Type: tea.KeyEsc}
	updated, cmd := model.Update(msg)
	model = updated.(*Model)

	assert.Nil(t, cmd)
	assert.False(t, model.commandMode)
	assert.Equal(t, "", model.commandQuery)

	model.commandMode = true
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, "q", model.commandQuery)

	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, "", model.commandQuery)

	model.commandMode = true
	model.commandQuery = "q"
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd = model.Update(msg)

	assert.NotNil(t, cmd)
}

func TestModelUpdateHandlesWindowSize(t *testing.T) {
	model := &Model{}

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 100, model.width)
	assert.Equal(t, 30, model.height)
	assert.Equal(t, 28, model.viewport.Height)
}

func TestModelViewRendersContent(t *testing.T) {
	stubSessionFetchers(t)

	model := &Model{
		notifications: []notification.Notification{
			{ID: 1, Message: "Test notification", Timestamp: "2024-01-01T12:00:00Z", Level: "info", State: "active"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "Test notification", Timestamp: "2024-01-01T12:00:00Z", Level: "info", State: "active"},
		},
		cursor:   0,
		width:    84,
		height:   24,
		viewport: viewport.New(84, 22),
	}
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
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
		width:         80,
		height:        24,
		viewport:      viewport.New(80, 22),
	}
	model.updateViewportContent()

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "No notifications found")
}

func TestUpdateViewportContentGroupedViewWithEmptyTree(t *testing.T) {
	model := &Model{
		viewMode:      viewModeGrouped,
		notifications: []notification.Notification{},
		viewport:      viewport.New(80, 22),
		width:         80,
	}

	model.applySearchFilter()

	assert.Contains(t, model.viewport.View(), "No notifications found")
}

func TestUpdateViewportContentGroupedViewRendersMixedNodes(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One", Level: "info", State: "active"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()
	require.NotEmpty(t, model.visibleNodes)
	model.cursor = 0
	model.updateViewportContent()

	content := model.viewport.View()
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
		Width:    model.width,
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
		Width:        model.width,
		Selected:     leafIndex == model.cursor,
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
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "First", Level: "info", State: "active"},
			{ID: 2, Session: "$1", Window: "@1", Pane: "%1", Message: "Second", Level: "info", State: "active"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

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
	model.cursor = leafIndex
	model.updateViewportContent()

	content := model.viewport.View()
	expectedLeafRow := render.Row(render.RowState{
		Notification: *leafNode.Notification,
		SessionName:  model.getSessionName(leafNode.Notification.Session),
		Width:        model.width,
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
		Width:    model.width,
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

	lines, err := storage.ListNotifications("active", "", "", "", "", "", "")
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

	lines, err := storage.ListNotifications("active", "", "", "", "", "", "")
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
	model.cursor = cursorIndex

	cmd := model.handleDismiss()

	assert.Nil(t, cmd)

	lines, err := storage.ListNotifications("active", "", "", "", "", "", "")
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
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
		cursor:        0,
	}

	cmd := model.handleDismiss()

	assert.Nil(t, cmd)
}

func TestHandleJumpWithMissingContext(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{
			{ID: 1, Message: "Test"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "Test"},
		},
		cursor: 0,
	}

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
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "b", Window: "@1", Pane: "%1", Message: "B"},
			{ID: 2, Session: "a", Window: "", Pane: "%1", Message: "A"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
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
	model.cursor = 0

	cmd := model.handleJump()

	assert.Nil(t, cmd)
}

func TestHandleJumpWithEmptyList(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
		cursor:        0,
	}

	cmd := model.handleJump()

	assert.Nil(t, cmd)
}

func TestModelUpdateHandlesDismissKey(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
		cursor:        0,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := model.Update(msg)

	assert.NotNil(t, updated.(*Model))
}

func TestModelUpdateHandlesZaToggleFold(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

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
	model.cursor = groupIndex

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
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
		cursor:        0,
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)

	assert.NotNil(t, updated.(*Model))
}

func TestGetSessionNameCachesFetcher(t *testing.T) {
	model := &Model{
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
			model:   &Model{},
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
			model: &Model{},
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
				filters: settings.Filter{
					Level:   settings.LevelFilterError,
					State:   settings.StateFilterActive,
					Session: "old-session",
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
				assert.Empty(t, m.filters.Window)
				assert.Empty(t, m.filters.Pane)
				assert.Equal(t, settings.GroupByPane, m.groupBy)
				assert.Equal(t, 2, m.defaultExpandLevel)
				assert.Equal(t, map[string]bool{}, m.expansionState)
			},
		},
		{
			name:    "invalid groupBy",
			model:   &Model{},
			state:   settings.TUIState{GroupBy: "invalid"},
			wantErr: true,
		},
		{
			name:    "invalid defaultExpandLevel",
			model:   &Model{},
			state:   settings.TUIState{DefaultExpandLevel: 4, DefaultExpandLevelSet: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.FromState(tt.state)

			if (err != nil) != tt.wantErr {
				t.Fatalf("FromState() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.verifyFn != nil {
				tt.verifyFn(t, tt.model)
			}
		})
	}
}

func TestRoundTripSettings(t *testing.T) {
	tests := []struct {
		name  string
		model *Model
	}{
		{
			name:  "empty model",
			model: &Model{},
		},
		{
			name: "model with all settings",
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
				groupBy:            settings.GroupByWindow,
				defaultExpandLevel: 2,
				expansionState: map[string]bool{
					"window:@1": true,
				},
			},
		},
		{
			name: "model with partial settings",
			model: &Model{
				sortBy:   settings.SortByTimestamp,
				viewMode: settings.ViewModeCompact,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.model.ToState()

			newModel := &Model{}
			err := newModel.FromState(state)
			require.NoError(t, err)

			assert.Equal(t, tt.model.sortBy, newModel.sortBy)
			assert.Equal(t, tt.model.sortOrder, newModel.sortOrder)
			assert.Equal(t, tt.model.columns, newModel.columns)
			assert.Equal(t, tt.model.filters, newModel.filters)
			assert.Equal(t, tt.model.viewMode, newModel.viewMode)
			assert.Equal(t, tt.model.groupBy, newModel.groupBy)
			assert.Equal(t, tt.model.defaultExpandLevel, newModel.defaultExpandLevel)
			assert.Equal(t, tt.model.expansionState, newModel.expansionState)
		})
	}
}

func TestSaveSettings(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:             settings.SortByLevel,
		sortOrder:          settings.SortOrderAsc,
		columns:            []string{settings.ColumnID, settings.ColumnMessage},
		viewMode:           settings.ViewModeDetailed,
		groupBy:            settings.GroupBySession,
		defaultExpandLevel: 2,
		expansionState: map[string]bool{
			"session:$1": true,
		},
	}

	err := model.saveSettings()
	require.NoError(t, err)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, loaded.SortOrder)
	assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage}, loaded.Columns)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
	assert.Equal(t, settings.GroupBySession, loaded.GroupBy)
	assert.Equal(t, 2, loaded.DefaultExpandLevel)
	assert.Equal(t, map[string]bool{"session:$1": true}, loaded.ExpansionState)
}

func TestModelSaveOnQuit(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:    settings.SortByLevel,
		sortOrder: settings.SortOrderAsc,
		viewMode:  settings.ViewModeDetailed,
		groupBy:   settings.GroupBySession,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, loaded.SortOrder)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
	assert.Equal(t, settings.GroupBySession, loaded.GroupBy)
}

func TestTUISaveOnExit(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:    settings.SortByLevel,
		sortOrder: settings.SortOrderAsc,
		columns:   []string{settings.ColumnID, settings.ColumnMessage},
		viewMode:  settings.ViewModeDetailed,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)
	assert.NotNil(t, cmd)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, loaded.SortOrder)
	assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage}, loaded.Columns)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
}

func TestModelSaveOnCtrlC(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:   settings.SortByTimestamp,
		viewMode: settings.ViewModeDetailed,
		groupBy:  settings.GroupByWindow,
	}

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByTimestamp, loaded.SortBy)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
	assert.Equal(t, settings.GroupByWindow, loaded.GroupBy)
}

func TestTUILoadOnStart(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	original := &settings.Settings{
		Columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
		SortBy:    settings.SortByLevel,
		SortOrder: settings.SortOrderAsc,
		Filters: settings.Filter{
			Level:   settings.LevelFilterWarning,
			State:   settings.StateFilterActive,
			Session: "session-1",
		},
		ViewMode: settings.ViewModeDetailed,
	}

	require.NoError(t, settings.Save(original))

	loaded, err := settings.Load()
	require.NoError(t, err)

	model := &Model{}
	model.SetLoadedSettings(loaded)
	require.NoError(t, model.FromState(settings.FromSettings(loaded)))

	assert.Equal(t, original.Columns, model.columns)
	assert.Equal(t, original.SortBy, model.sortBy)
	assert.Equal(t, original.SortOrder, model.sortOrder)
	assert.Equal(t, original.Filters, model.filters)
	assert.Equal(t, original.ViewMode, model.viewMode)
}

func TestModelSaveOnCommandQ(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:   settings.SortByLevel,
		viewMode: settings.ViewModeDetailed,
		groupBy:  settings.GroupByPane,
	}

	model.commandMode = true
	model.commandQuery = "q"
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
	assert.Equal(t, settings.GroupByPane, loaded.GroupBy)
}

func TestModelSaveCommandW(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:   settings.SortByLevel,
		viewMode: settings.ViewModeDetailed,
		groupBy:  settings.GroupBySession,
	}

	model.commandMode = true
	model.commandQuery = "w"
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := model.Update(msg)

	require.NotNil(t, cmd)
	_ = cmd()

	model = updated.(*Model)
	assert.False(t, model.commandMode)
	assert.Equal(t, "", model.commandQuery)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
	assert.Equal(t, settings.GroupBySession, loaded.GroupBy)
}

func TestModelMissingSettingsFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	settingsPath := tmpDir + "/settings.json"
	_, err := os.Stat(settingsPath)
	assert.True(t, os.IsNotExist(err))

	model, err := NewModel(mockClient)
	require.NoError(t, err)
	assert.NotNil(t, model)
}

func TestModelCorruptedSettingsFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	setupStorage(t)
	stubSessionFetchers(t)

	settingsPath := tmpDir + "/settings.json"
	err := os.WriteFile(settingsPath, []byte("invalid json {{{"), 0644)
	require.NoError(t, err)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.NotEmpty(t, loaded.SortBy)
}

func TestModelSettingsLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	setupStorage(t)
	stubSessionFetchers(t)

	loaded, err := settings.Load()
	require.NoError(t, err)
	require.NotNil(t, loaded)

	model := &Model{}
	state := settings.FromSettings(loaded)
	err = model.FromState(state)
	require.NoError(t, err)

	model.sortBy = settings.SortByLevel
	model.sortOrder = settings.SortOrderAsc
	model.viewMode = settings.ViewModeDetailed
	model.groupBy = settings.GroupByWindow
	model.defaultExpandLevel = 2
	model.expansionState = map[string]bool{"window:@1": true}

	err = model.saveSettings()
	require.NoError(t, err)

	reloaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, reloaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, reloaded.SortOrder)
	assert.Equal(t, settings.ViewModeDetailed, reloaded.ViewMode)
	assert.Equal(t, settings.GroupByWindow, reloaded.GroupBy)
	assert.Equal(t, 2, reloaded.DefaultExpandLevel)
	assert.Equal(t, map[string]bool{"window:@1": true}, reloaded.ExpansionState)

	newModel := &Model{}
	newState := settings.FromSettings(reloaded)
	err = newModel.FromState(newState)
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, newModel.sortBy)
	assert.Equal(t, settings.SortOrderAsc, newModel.sortOrder)
	assert.Equal(t, settings.ViewModeDetailed, newModel.viewMode)
	assert.Equal(t, settings.GroupByWindow, newModel.groupBy)
	assert.Equal(t, 2, newModel.defaultExpandLevel)
	assert.Equal(t, map[string]bool{"window:@1": true}, newModel.expansionState)
}

func TestGroupedViewWithGroupByNone(t *testing.T) {
	t.Run("default settings", func(t *testing.T) {
		setupStorage(t)
		mockClient := stubSessionFetchers(t)
		_, err := storage.AddNotification("test message", "", "", "", "", "", "info")
		require.NoError(t, err)
		model, err := NewModel(mockClient)
		require.NoError(t, err)
		model.viewMode = settings.ViewModeGrouped
		model.groupBy = settings.GroupByNone
		model.applySearchFilter()
		t.Logf("visibleNodes count: %d", len(model.visibleNodes))
		for i, n := range model.visibleNodes {
			t.Logf("  [%d] kind=%s title=%s expanded=%v", i, n.Kind, n.Title, n.Expanded)
		}
		require.NotEmpty(t, model.visibleNodes)
	})
	t.Run("defaultExpandLevel=1", func(t *testing.T) {
		setupStorage(t)
		mockClient := stubSessionFetchers(t)
		_, err := storage.AddNotification("test message", "", "", "", "", "", "info")
		require.NoError(t, err)
		model, err := NewModel(mockClient)
		require.NoError(t, err)
		model.viewMode = settings.ViewModeGrouped
		model.groupBy = settings.GroupByNone
		model.defaultExpandLevel = 1
		model.applySearchFilter()
		t.Logf("visibleNodes count: %d", len(model.visibleNodes))
		for i, n := range model.visibleNodes {
			t.Logf("  [%d] kind=%s title=%s expanded=%v", i, n.Kind, n.Title, n.Expanded)
		}
		require.NotEmpty(t, model.visibleNodes)
	})
}
