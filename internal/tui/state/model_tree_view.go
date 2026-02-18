package state

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

func (m *Model) isGroupedView() bool {
	return m.uiState.IsGroupedView()
}

// IsGroupedView is the public version of isGroupedView.
func (m *Model) IsGroupedView() bool {
	return m.isGroupedView()
}

// cycleViewMode cycles through available view modes (compact → detailed → grouped → search).
func (m *Model) cycleViewMode() {
	prevMode := m.uiState.GetViewMode()
	m.uiState.CycleViewMode()
	nextMode := m.uiState.GetViewMode()

	// Search view mode implies search input should be active.
	if nextMode == model.ViewModeSearch {
		m.uiState.SetSearchMode(true)
	} else if prevMode == model.ViewModeSearch {
		// Leaving search view mode should exit the search input.
		m.uiState.SetSearchMode(false)
	}
	m.applySearchFilter()
	m.resetCursor()

	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}
}

func (m *Model) computeVisibleNodes() []*model.TreeNode {
	return m.treeService.GetVisibleNodes()
}

func (m *Model) invalidateCache() {
	m.ensureTreeService().InvalidateCache()
}

func isGroupNode(node *model.TreeNode) bool {
	if node == nil {
		return false
	}
	return node.Kind != model.NodeKindNotification && node.Kind != model.NodeKindRoot
}

func (m *Model) isGroupNode(node *model.TreeNode) bool {
	return node.Kind != model.NodeKindNotification && node.Kind != model.NodeKindRoot
}

func getTreeLevel(node *model.TreeNode) int {
	if node == nil {
		return 0
	}
	switch node.Kind {
	case model.NodeKindSession:
		return 0
	case model.NodeKindWindow:
		return 1
	case model.NodeKindPane:
		return 2
	case model.NodeKindMessage:
		return 3
	default:
		return 0
	}
}

func (m *Model) currentListLen() int {
	if m.isGroupedView() {
		return len(m.treeService.GetVisibleNodes())
	}
	return len(m.filtered)
}

func (m *Model) selectedVisibleNode() *model.TreeNode {
	if !m.isGroupedView() {
		return nil
	}
	cursor := m.uiState.GetCursor()
	visibleNodes := m.treeService.GetVisibleNodes()
	if cursor < 0 || cursor >= len(visibleNodes) {
		return nil
	}
	return visibleNodes[cursor]
}

func (m *Model) toggleNodeExpansion() bool {
	node := m.selectedVisibleNode()
	if node == nil || node.Kind == model.NodeKindNotification {
		return false
	}
	if node.Expanded {
		m.treeService.CollapseNode(node)
	} else {
		m.treeService.ExpandNode(node)
	}
	m.invalidateCache()
	return true
}

func (m *Model) toggleFold() {
	if !m.isGroupedView() {
		return
	}
	node := m.selectedVisibleNode()
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	if m.allGroupsCollapsed() {
		m.applyDefaultExpansion()
		return
	}
	if node.Expanded {
		m.treeService.CollapseNode(node)
		m.invalidateCache()
		m.updateViewportContent()
		return
	}
	m.treeService.ExpandNode(node)
	m.invalidateCache()
	m.updateViewportContent()
}

func (m *Model) allGroupsCollapsed() bool {
	treeRoot := m.treeService.GetTreeRoot()
	if treeRoot == nil {
		return false
	}
	collapsed := true
	seen := false
	var walk func(node *model.TreeNode)
	walk = func(node *model.TreeNode) {
		if node == nil || !collapsed {
			return
		}
		if m.isGroupNode(node) {
			seen = true
			if node.Expanded {
				collapsed = false
				return
			}
		}
		for _, child := range node.Children {
			walk(child)
			if !collapsed {
				return
			}
		}
	}
	walk(treeRoot)
	return seen && collapsed
}

func (m *Model) applyDefaultExpansion() {
	treeRoot := m.treeService.GetTreeRoot()
	if treeRoot == nil {
		return
	}

	// Save selected node identifier before modifying tree
	selectedID := ""
	if selected := m.selectedVisibleNode(); selected != nil {
		selectedID = m.treeService.GetNodeIdentifier(selected)
	}

	level := m.uiState.GetExpandLevel()
	if level < settings.MinExpandLevel {
		level = settings.MinExpandLevel
	}
	if level > settings.MaxExpandLevel {
		level = settings.MaxExpandLevel
	}

	var walk func(node *model.TreeNode)
	walk = func(node *model.TreeNode) {
		if node == nil {
			return
		}
		if m.isGroupNode(node) {
			nodeLevel := m.treeService.GetTreeLevel(node) + 1
			expanded := nodeLevel <= level
			node.Expanded = expanded
			m.updateExpansionState(node, expanded)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(treeRoot)

	m.invalidateCache()

	// Restore cursor to the selected node using identifier
	if selectedID != "" {
		m.restoreCursor(selectedID)
	}

	// Ensure cursor is within bounds
	visibleNodes := m.treeService.GetVisibleNodes()
	if m.uiState.GetCursor() >= len(visibleNodes) {
		m.uiState.SetCursor(len(visibleNodes) - 1)
	}
	if m.uiState.GetCursor() < 0 {
		m.uiState.SetCursor(0)
	}
	m.updateViewportContent()
	m.ensureCursorVisible()
}

// ApplyDefaultExpansion is the public version of applyDefaultExpansion.
func (m *Model) ApplyDefaultExpansion() {
	m.applyDefaultExpansion()
}

// GetViewMode returns the current view mode.
func (m *Model) GetViewMode() string {
	return string(m.uiState.GetViewMode())
}

// ToggleViewMode toggles between view modes.
func (m *Model) ToggleViewMode() error {
	m.cycleViewMode()
	return nil
}
