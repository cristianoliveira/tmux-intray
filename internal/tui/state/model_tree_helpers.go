package state

import (
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// getTreeRootForTest returns the tree root for testing purposes.
func (m *Model) getTreeRootForTest() *model.TreeNode {
	return m.treeService.GetTreeRoot()
}

// getVisibleNodesForTest returns the visible nodes for testing purposes.
func (m *Model) getVisibleNodesForTest() []*model.TreeNode {
	return m.treeService.GetVisibleNodes()
}

// collectNotificationsInGroup collects all notifications under a group node.
// Returns the session, window, pane filters and count of notifications.
func (m *Model) collectNotificationsInGroup(node *model.TreeNode) (session, window, pane string, count int) {
	if node == nil || node.Kind == model.NodeKindNotification {
		return "", "", "", 0
	}

	count = node.Count

	switch node.Kind {
	case model.NodeKindSession:
		return node.Title, "", "", count
	case model.NodeKindWindow, model.NodeKindPane:
		sess, win, pan, ok := m.getAncestorTitles(node)
		if !ok {
			return "", "", "", 0
		}
		return sess, win, pan, count
	default:
		return "", "", "", 0
	}
}

// getGroupTypeLabel returns a human-readable label for the group kind.
func getGroupTypeLabel(kind model.NodeKind) string {
	switch kind {
	case model.NodeKindSession:
		return "session"
	case model.NodeKindWindow:
		return "window"
	case model.NodeKindPane:
		return "pane"
	case model.NodeKindMessage:
		return "message"
	default:
		return "group"
	}
}
