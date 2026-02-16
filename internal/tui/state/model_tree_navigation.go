package state

import (
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// getNodeIdentifier returns a stable identifier for a node.
// For notification nodes, this is the notification ID.
// For group nodes, this is a combination of the node kind and title.
func (m *Model) getNodeIdentifier(node *model.TreeNode) string {
	return m.treeService.GetNodeIdentifier(node)
}

// findNodeByIdentifier finds a node by its identifier in the visible nodes list.
func (m *Model) findNodeByIdentifier(identifier string) *model.TreeNode {
	for _, node := range m.treeService.GetVisibleNodes() {
		if m.treeService.GetNodeIdentifier(node) == identifier {
			return node
		}
	}
	return nil
}

// restoreCursor restores the cursor to the node with the given identifier.
// If the node is not found, it adjusts the cursor to be within bounds.
func (m *Model) restoreCursor(identifier string) {
	if identifier == "" {
		m.adjustCursorBounds()
		return
	}

	targetNode := m.findNodeByIdentifier(identifier)
	if targetNode != nil {
		visibleNodes := m.ensureTreeService().GetVisibleNodes()
		for i, node := range visibleNodes {
			if node == targetNode {
				m.uiState.SetCursor(i)
				m.uiState.EnsureCursorVisible(len(visibleNodes))
				return
			}
		}
	}

	// If we couldn't find the exact node, adjust to bounds
	m.adjustCursorBounds()
}

// adjustCursorBounds ensures the cursor is within valid bounds.
func (m *Model) adjustCursorBounds() {
	listLen := m.currentListLen()
	m.uiState.AdjustCursorBounds(listLen)
	m.uiState.EnsureCursorVisible(listLen)
}

// getSessionName returns the session name for a session ID.
// Uses RuntimeCoordinator for name resolution.
func (m *Model) getSessionName(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	if m.runtimeCoordinator == nil {
		return sessionID
	}

	name, err := m.runtimeCoordinator.GetSessionName(sessionID)
	if err == nil && name != "" {
		return name
	}

	return m.runtimeCoordinator.ResolveSessionName(sessionID)
}

// getWindowName returns the window name for a window ID.
// Uses RuntimeCoordinator for name resolution.
func (m *Model) getWindowName(windowID string) string {
	if windowID == "" {
		return ""
	}
	if m.runtimeCoordinator == nil {
		return windowID
	}

	name, err := m.runtimeCoordinator.GetWindowName(windowID)
	if err == nil && name != "" {
		return name
	}

	return m.runtimeCoordinator.ResolveWindowName(windowID)
}

// getPaneName returns the pane name for a pane ID.
// Uses RuntimeCoordinator for name resolution.
func (m *Model) getPaneName(paneID string) string {
	if paneID == "" {
		return ""
	}
	if m.runtimeCoordinator == nil {
		return paneID
	}

	name, err := m.runtimeCoordinator.GetPaneName(paneID)
	if err == nil && name != "" {
		return name
	}

	return m.runtimeCoordinator.ResolvePaneName(paneID)
}
