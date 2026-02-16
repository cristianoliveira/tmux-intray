package state

import (
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// expandTreeRecursive is a helper that expands all group nodes.
func (m *Model) expandTreeRecursive(node *model.TreeNode) {
	if node == nil {
		return
	}
	if node.Kind != model.NodeKindNotification {
		node.Expanded = true
	}
	for _, child := range node.Children {
		m.expandTreeRecursive(child)
	}
}

func (m *Model) nodePathSegments(path []*model.TreeNode) (session string, window string, pane string) {
	for _, current := range path {
		switch current.Kind {
		case model.NodeKindSession:
			session = current.Title
		case model.NodeKindWindow:
			window = current.Title
		case model.NodeKindPane:
			pane = current.Title
		}
	}
	return session, window, pane
}

// findNodePath returns the path from root to target node (inclusive).
// Returns nil if target not found.
func (m *Model) findNodePath(root, target *model.TreeNode) []*model.TreeNode {
	if root == nil || target == nil {
		return nil
	}
	if root == target {
		return []*model.TreeNode{root}
	}
	for _, child := range root.Children {
		if path := m.findNodePath(child, target); path != nil {
			return append([]*model.TreeNode{root}, path...)
		}
	}
	return nil
}

// getAncestorTitles returns session, window, pane titles for a node by finding its path from root.
func (m *Model) getAncestorTitles(node *model.TreeNode) (session, window, pane string, ok bool) {
	treeRoot := m.treeService.GetTreeRoot()
	if treeRoot == nil {
		return "", "", "", false
	}
	path := m.findNodePath(treeRoot, node)
	if path == nil {
		return "", "", "", false
	}
	session, window, pane = m.nodePathSegments(path)
	return session, window, pane, true
}
