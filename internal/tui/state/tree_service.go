package state

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

type treeService struct{}

func newTreeService() *treeService {
	return &treeService{}
}

func (s *treeService) buildFilteredTree(notifications []notification.Notification, groupBy string, expansionState map[string]bool) *Node {
	if len(notifications) == 0 {
		return nil
	}

	root := BuildTree(notifications, groupBy)
	s.pruneEmptyGroups(root)

	if expansionState != nil {
		s.applyExpansionState(root, root, expansionState)
	} else {
		expandTree(root)
	}

	return root
}

func (s *treeService) computeVisibleNodes(root *Node) []*Node {
	if root == nil {
		return nil
	}

	visible := make([]*Node, 0)
	var walk func(node *Node)
	walk = func(node *Node) {
		if node == nil {
			return
		}
		if node.Kind != NodeKindRoot {
			visible = append(visible, node)
		}
		if node.Kind == NodeKindNotification {
			return
		}
		if node.Kind != NodeKindRoot && !node.Expanded {
			return
		}
		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)
	return visible
}

func (s *treeService) getNodeIdentifier(root *Node, node *Node) string {
	if node == nil {
		return ""
	}
	if node.Kind == NodeKindNotification && node.Notification != nil {
		return fmt.Sprintf("notif:%d", node.Notification.ID)
	}
	if node.Kind == NodeKindRoot {
		return "root"
	}
	path, ok := findNodePath(root, node)
	if !ok || len(path) == 0 {
		return ""
	}
	parts := make([]string, 0, len(path)*2)
	for _, current := range path {
		if current.Kind == NodeKindRoot {
			continue
		}
		parts = append(parts, string(current.Kind), current.Title)
	}
	return strings.Join(parts, ":")
}

func (s *treeService) findNodeByIdentifier(root *Node, visibleNodes []*Node, identifier string) *Node {
	for _, node := range visibleNodes {
		if s.getNodeIdentifier(root, node) == identifier {
			return node
		}
	}
	return nil
}

func (s *treeService) applyDefaultExpansion(root *Node, level int, expansionState map[string]bool) {
	if root == nil {
		return
	}

	var walk func(node *Node)
	walk = func(node *Node) {
		if node == nil {
			return
		}
		if isGroupNode(node) {
			nodeLevel := getTreeLevel(node) + 1
			expanded := nodeLevel <= level
			node.Expanded = expanded
			s.updateExpansionState(root, expansionState, node, expanded)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)
}

func (s *treeService) updateExpansionState(root *Node, expansionState map[string]bool, node *Node, expanded bool) {
	if expansionState == nil {
		return
	}

	key := s.nodeExpansionKey(root, node)
	if key == "" {
		return
	}
	legacyKey := s.nodeExpansionLegacyKey(root, node)
	if legacyKey != "" && legacyKey != key {
		delete(expansionState, legacyKey)
	}
	expansionState[key] = expanded
}

func (s *treeService) pruneEmptyGroups(node *Node) {
	if node == nil {
		return
	}

	filteredChildren := make([]*Node, 0, len(node.Children))
	for _, child := range node.Children {
		s.pruneEmptyGroups(child)
		if len(child.Children) > 0 || child.Kind == NodeKindNotification {
			filteredChildren = append(filteredChildren, child)
		}
	}
	node.Children = filteredChildren
}

func (s *treeService) applyExpansionState(root *Node, node *Node, expansionState map[string]bool) {
	if node == nil {
		return
	}

	if isGroupNode(node) {
		if expanded, ok := s.expansionStateValue(root, node, expansionState); ok {
			node.Expanded = expanded
		} else {
			node.Expanded = true
		}
	}

	for _, child := range node.Children {
		s.applyExpansionState(root, child, expansionState)
	}
}

func (s *treeService) expansionStateValue(root *Node, node *Node, expansionState map[string]bool) (bool, bool) {
	if expansionState == nil {
		return false, false
	}

	key := s.nodeExpansionKey(root, node)
	if key != "" {
		expanded, ok := expansionState[key]
		if ok {
			return expanded, true
		}
	}

	legacyKey := s.nodeExpansionLegacyKey(root, node)
	if legacyKey == "" {
		return false, false
	}

	expanded, ok := expansionState[legacyKey]
	if !ok {
		return false, false
	}
	if key != "" {
		expansionState[key] = expanded
		delete(expansionState, legacyKey)
	}
	return expanded, true
}

func (s *treeService) nodeExpansionKey(root *Node, node *Node) string {
	if node == nil || node.Kind == NodeKindNotification || node.Kind == NodeKindRoot {
		return ""
	}
	path, ok := findNodePath(root, node)
	if !ok || len(path) == 0 {
		return ""
	}

	session, window, pane := nodePathSegments(path)

	switch node.Kind {
	case NodeKindSession:
		return serializeNodeExpansionPath(NodeKindSession, session)
	case NodeKindWindow:
		if session == "" {
			return ""
		}
		return serializeNodeExpansionPath(NodeKindWindow, session, window)
	case NodeKindPane:
		if session == "" || window == "" {
			return ""
		}
		return serializeNodeExpansionPath(NodeKindPane, session, window, pane)
	default:
		return ""
	}
}

func (s *treeService) nodeExpansionLegacyKey(root *Node, node *Node) string {
	if node == nil || node.Kind == NodeKindNotification || node.Kind == NodeKindRoot {
		return ""
	}
	path, ok := findNodePath(root, node)
	if !ok || len(path) == 0 {
		return ""
	}

	session, window, pane := nodePathSegments(path)

	switch node.Kind {
	case NodeKindSession:
		return serializeLegacyNodeExpansionPath(NodeKindSession, session)
	case NodeKindWindow:
		if session == "" {
			return ""
		}
		return serializeLegacyNodeExpansionPath(NodeKindWindow, session, window)
	case NodeKindPane:
		if session == "" || window == "" {
			return ""
		}
		return serializeLegacyNodeExpansionPath(NodeKindPane, session, window, pane)
	default:
		return ""
	}
}
