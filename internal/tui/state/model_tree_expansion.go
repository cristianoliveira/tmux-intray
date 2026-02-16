package state

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

func (m *Model) collapseNode(node *model.TreeNode) {
	if !m.isGroupedView() {
		return
	}
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	if !node.Expanded {
		return
	}

	// Save node identifiers before modifying tree to avoid using stale references
	selectedID := ""
	if selected := m.selectedVisibleNode(); selected != nil {
		selectedID = m.treeService.GetNodeIdentifier(selected)
	}
	nodeID := m.treeService.GetNodeIdentifier(node)

	m.treeService.CollapseNode(node)
	m.updateExpansionState(node, false)
	visibleNodes := m.treeService.GetVisibleNodes()

	// If selected node was inside the collapsed node, move cursor to the collapsed node
	m.moveCursorToCollapsedNodeIfNeeded(selectedID, nodeID, visibleNodes)

	// Ensure cursor is within bounds
	m.clampCursorToBounds(len(visibleNodes))
	m.updateViewportContent()
	m.ensureCursorVisible()
}

// moveCursorToCollapsedNodeIfNeeded moves cursor to collapsed node if selected was inside it.
func (m *Model) moveCursorToCollapsedNodeIfNeeded(selectedID, nodeID string, visibleNodes []*model.TreeNode) {
	if selectedID == "" {
		return
	}

	treeRoot := m.treeService.GetTreeRoot()
	selectedNode := m.treeService.FindNodeByID(treeRoot, selectedID)
	if selectedNode == nil {
		return
	}

	collapsedNode := m.treeService.FindNodeByID(treeRoot, nodeID)
	if collapsedNode == nil {
		return
	}

	if !m.nodeContains(collapsedNode, selectedNode) {
		return
	}

	// Move cursor to the collapsed node
	if index := indexOfTreeNode(visibleNodes, collapsedNode); index >= 0 {
		m.uiState.SetCursor(index)
	}
}

// clampCursorToBounds ensures cursor is within valid bounds.
func (m *Model) clampCursorToBounds(listLen int) {
	if m.uiState.GetCursor() >= listLen {
		m.uiState.SetCursor(listLen - 1)
	}
	if m.uiState.GetCursor() < 0 {
		m.uiState.SetCursor(0)
	}
}

// nodeContains checks if targetNode is contained within root node.
func (m *Model) nodeContains(root, target *model.TreeNode) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	for _, child := range root.Children {
		if m.nodeContains(child, target) {
			return true
		}
	}
	return false
}

// indexOfTreeNode finds the index of a target node in a slice.
func indexOfTreeNode(nodes []*model.TreeNode, target *model.TreeNode) int {
	for i, node := range nodes {
		if node == target {
			return i
		}
	}
	return -1
}

func (m *Model) updateExpansionState(node *model.TreeNode, expanded bool) {
	key := m.nodeExpansionKey(node)
	if key == "" {
		return
	}
	expansionState := m.uiState.GetExpansionState()
	if expansionState == nil {
		expansionState = map[string]bool{}
		m.uiState.SetExpansionState(expansionState)
	}
	legacyKey := m.nodeExpansionLegacyKey(node)
	if legacyKey != "" && legacyKey != key {
		delete(expansionState, legacyKey)
	}
	m.uiState.UpdateExpansionState(key, expanded)
}

func (m *Model) nodeExpansionKey(node *model.TreeNode) string {
	if node == nil || node.Kind == model.NodeKindNotification || node.Kind == model.NodeKindRoot {
		return ""
	}
	return m.ensureTreeService().GetNodeIdentifier(node)
}

func (m *Model) nodeExpansionLegacyKey(node *model.TreeNode) string {
	if node == nil || node.Kind == model.NodeKindNotification || node.Kind == model.NodeKindRoot {
		return ""
	}
	switch node.Kind {
	case model.NodeKindSession:
		return serializeLegacyNodeExpansionPath(model.NodeKindSession, node.Title)
	case model.NodeKindWindow:
		return serializeLegacyNodeExpansionPath(model.NodeKindWindow, node.Title)
	case model.NodeKindPane:
		return serializeLegacyNodeExpansionPath(model.NodeKindPane, node.Title)
	case model.NodeKindMessage:
		return serializeLegacyNodeExpansionPath(model.NodeKindMessage, node.Title)
	default:
		return ""
	}
}

func serializeNodeExpansionPath(kind model.NodeKind, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	encoded := make([]string, 0, len(parts))
	for _, part := range parts {
		encoded = append(encoded, escapeExpansionPathSegment(part))
	}
	return fmt.Sprintf("%s:%s", kind, strings.Join(encoded, ":"))
}

func escapeExpansionPathSegment(value string) string {
	replacer := strings.NewReplacer(
		"%", "%25",
		":", "%3A",
	)
	return replacer.Replace(value)
}

func serializeLegacyNodeExpansionPath(kind model.NodeKind, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%s", kind, strings.Join(parts, ":"))
}

func (m *Model) expansionStateValue(node *model.TreeNode) (bool, bool) {
	expansionState := m.uiState.GetExpansionState()
	if expansionState == nil {
		return false, false
	}

	key := m.nodeExpansionKey(node)
	if key != "" {
		expanded, ok := expansionState[key]
		if ok {
			return expanded, true
		}
	}

	legacyKey := m.nodeExpansionLegacyKey(node)
	if legacyKey == "" {
		return false, false
	}

	expanded, ok := expansionState[legacyKey]
	if !ok {
		return false, false
	}
	if key != "" {
		m.uiState.UpdateExpansionState(key, expanded)
		delete(expansionState, legacyKey)
	}
	return expanded, true
}
