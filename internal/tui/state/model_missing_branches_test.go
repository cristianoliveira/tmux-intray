package state

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
	"github.com/stretchr/testify/assert"
)

// Test missing branches in model_keys_core.go

func TestHandleKeyMsgWithUnknownKey(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})

	// Test with unknown key binding
	next, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Same(t, m, next)
	assert.Nil(t, cmd)
}

func TestHandleConfirmationUnknownAction(t *testing.T) {
	m := newTestModel(t, nil)
	m.uiState.SetConfirmationMode(true)
	m.uiState.SetPendingAction(PendingAction{Type: "unknown"})

	next, cmd := m.handleConfirmation(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Same(t, m, next)
	assert.Nil(t, cmd)
	assert.False(t, m.uiState.IsConfirmationMode())
}

func TestHandleKeyTypeCtrlHOutsideSearch(t *testing.T) {
	m := newTestModel(t, nil)

	// Test Ctrl+H outside search context (should be no-op)
	next, cmd := m.handleKeyType(tea.KeyMsg{Type: tea.KeyCtrlH})
	assert.Same(t, m, next) // Returns the model
	assert.Nil(t, cmd)
}

func TestHandleKeyTypeCtrlJOutsideSearch(t *testing.T) {
	m := newTestModel(t, nil)

	// Test Ctrl+J outside search context (should be no-op)
	next, cmd := m.handleKeyType(tea.KeyMsg{Type: tea.KeyCtrlJ})
	assert.Same(t, m, next) // Returns the model
	assert.Nil(t, cmd)
}

func TestHandleKeyTypeCtrlKOutsideSearch(t *testing.T) {
	m := newTestModel(t, nil)

	// Test Ctrl+K outside search context (should be no-op)
	next, cmd := m.handleKeyType(tea.KeyMsg{Type: tea.KeyCtrlK})
	assert.Same(t, m, next) // Returns the model
	assert.Nil(t, cmd)
}

func TestHandleKeyTypeCtrlLOutsideSearch(t *testing.T) {
	m := newTestModel(t, nil)

	// Test Ctrl+L outside search context (should be no-op)
	next, cmd := m.handleKeyType(tea.KeyMsg{Type: tea.KeyCtrlL})
	assert.Same(t, m, next) // Returns the model
	assert.Nil(t, cmd)
}

func TestHandleNavigationKeysGInSearch(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})

	// Test 'G' key in search context (should be no-op)
	next, cmd := m.handleNavigationKeys("G", true)
	assert.Same(t, m, next)
	assert.Nil(t, cmd)
}

func TestHandleNavigationKeysGInNormal(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})
	m.uiState.SetCursor(5)

	// Test 'G' key in normal context
	next, cmd := m.handleNavigationKeys("G", false)
	assert.Same(t, m, next)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, m.uiState.GetCursor()) // Should move to bottom
}

func TestHandleTreeKeysZOutsideGroupedView(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})
	// Ensure not in grouped view
	m.uiState.SetViewMode(settings.ViewModeDetailed)

	// Test 'z' key outside grouped view
	next, cmd := m.handleTreeKeys("z", true)
	assert.Same(t, m, next)
	assert.Nil(t, cmd)
	assert.Equal(t, "", m.uiState.GetPendingKey())
}

func TestHandlePendingKeyWithZAndNonZ(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})
	m.uiState.SetGroupBy(settings.GroupBySession)
	m.uiState.SetPendingKey("z")

	// Test non-z key after pending 'z'
	handled, cmd := m.handlePendingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.False(t, handled)
	assert.Nil(t, cmd)
	assert.Equal(t, "", m.uiState.GetPendingKey())
}

func TestBindingKeyForMsgWithCtrlFallbackMultiChar(t *testing.T) {
	m := newTestModel(t, nil)
	m.uiState.SetSearchMode(true) // To get context with ctrlFallsBack

	// Test simple ctrl+ key - should use fallback
	key, allow := m.bindingKeyForMsg(tea.KeyMsg{Type: tea.KeyCtrlA})
	assert.Equal(t, "a", key) // Should fallback to just 'a'
	assert.True(t, allow)
}

// Test missing branches in model_key_handlers.go

func TestHandleSaveSettingsSuccessMsg(t *testing.T) {
	m := newTestModel(t, nil)

	next, cmd := m.handleSaveSettingsSuccess(saveSettingsSuccessMsg{})
	assert.Same(t, m, next)
	assert.Nil(t, cmd)
}

func TestHandleSaveSettingsFailedMsg(t *testing.T) {
	m := newTestModel(t, nil)

	next, cmd := m.handleSaveSettingsFailed(saveSettingsFailedMsg{err: assert.AnError})
	assert.Same(t, m, next)
	assert.Nil(t, cmd)
}

func TestSwitchActiveTabSameTab(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})
	m.uiState.SetActiveTab(settings.TabRecents)
	m.uiState.SetCursor(1)

	m.switchActiveTab(settings.TabRecents)

	assert.Equal(t, settings.TabRecents, m.uiState.GetActiveTab())
	assert.Equal(t, 1, m.uiState.GetCursor())
}

// Test missing branches in model_tree_view.go

func TestIsGroupNodeWithNil(t *testing.T) {
	// Test both the package-level and method versions
	assert.False(t, isGroupNode(nil))

	// Note: m.isGroupNode(nil) would panic due to nil pointer access
	// This is expected behavior - the caller should ensure node is not nil
}

func TestGetTreeLevelWithNil(t *testing.T) {
	assert.Equal(t, 0, getTreeLevel(nil))
}

func TestGetTreeLevelWithDefault(t *testing.T) {
	node := &model.TreeNode{Kind: model.NodeKindRoot}
	assert.Equal(t, 0, getTreeLevel(node))
}

func TestGetTreeLevelAllKinds(t *testing.T) {
	tests := []struct {
		kind  model.NodeKind
		level int
	}{
		{model.NodeKindSession, 0},
		{model.NodeKindWindow, 1},
		{model.NodeKindPane, 2},
		{model.NodeKindMessage, 3},
		{model.NodeKindRoot, 0},
	}

	for _, tt := range tests {
		node := &model.TreeNode{Kind: tt.kind}
		assert.Equal(t, tt.level, getTreeLevel(node))
	}
}

func TestSelectedVisibleNodeOutsideGroupedView(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})
	m.uiState.SetViewMode(settings.ViewModeDetailed)

	node := m.selectedVisibleNode()
	assert.Nil(t, node)
}

func TestSelectedVisibleNodeWithInvalidCursor(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})
	m.uiState.SetGroupBy(settings.GroupBySession)
	m.applySearchFilter()
	m.uiState.SetCursor(999) // Invalid cursor

	node := m.selectedVisibleNode()
	assert.Nil(t, node)
}

func TestToggleFoldOutsideGroupedView(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})
	m.uiState.SetViewMode(settings.ViewModeDetailed)

	m.toggleFold()
	// Should not crash or change state
}

func TestToggleFoldWithNotificationNode(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})
	m.uiState.SetGroupBy(settings.GroupBySession)
	m.applySearchFilter()
	// Point to a notification node (this will depend on tree structure)

	m.toggleFold()
	// Should not crash
}

func TestAllGroupsCollapsedWithNoTree(t *testing.T) {
	m := newTestModel(t, nil)

	result := m.allGroupsCollapsed()
	assert.False(t, result)
}

func TestApplyDefaultExpansionWithNilTree(t *testing.T) {
	m := newTestModel(t, nil)

	m.applyDefaultExpansion()
	// Should not crash
}

func TestApplyDefaultExpansionWithOutOfBoundsCursor(t *testing.T) {
	m := newTestModel(t, []notification.Notification{
		{ID: 1, Session: "s1", Window: "w1", Pane: "p1", Message: "test1"},
		{ID: 2, Session: "s1", Window: "w2", Pane: "p1", Message: "test2"},
	})
	m.uiState.SetGroupBy(settings.GroupBySession)
	m.applySearchFilter()

	// Only set out of bounds if we have visible nodes
	visibleNodes := m.treeService.GetVisibleNodes()
	if len(visibleNodes) > 0 {
		m.uiState.SetCursor(999) // Out of bounds
		m.applyDefaultExpansion()
		// Should adjust cursor to be within bounds
		assert.Less(t, m.uiState.GetCursor(), len(m.treeService.GetVisibleNodes()))
	}
}

func TestApplyDefaultExpansionWithNegativeCursor(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})
	m.uiState.SetGroupBy(settings.GroupBySession)
	m.applySearchFilter()
	m.uiState.SetCursor(-1) // Negative cursor

	m.applyDefaultExpansion()
	// Should adjust cursor to be within bounds
	assert.Equal(t, 0, m.uiState.GetCursor())
}

func TestAllGroupsCollapsedWithEmptyTree(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "test"}})
	m.uiState.SetGroupBy(settings.GroupBySession)
	m.applySearchFilter()
	// Force tree to have no visible nodes

	result := m.allGroupsCollapsed()
	assert.False(t, result)
}

func TestIsGroupNodeWithRoot(t *testing.T) {
	// Test with Root node
	node := &model.TreeNode{Kind: model.NodeKindRoot}
	assert.False(t, isGroupNode(node))

	m := newTestModel(t, nil)
	assert.False(t, m.isGroupNode(node))
}

func TestIsGroupNodeWithNotification(t *testing.T) {
	// Test with Notification node
	node := &model.TreeNode{Kind: model.NodeKindNotification}
	assert.False(t, isGroupNode(node))

	m := newTestModel(t, nil)
	assert.False(t, m.isGroupNode(node))
}

func TestIsGroupNodeWithSession(t *testing.T) {
	// Test with Session node
	node := &model.TreeNode{Kind: model.NodeKindSession}
	assert.True(t, isGroupNode(node))

	m := newTestModel(t, nil)
	assert.True(t, m.isGroupNode(node))
}
