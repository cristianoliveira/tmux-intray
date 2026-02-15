package state

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	uimodel "github.com/cristianoliveira/tmux-intray/internal/tui/model"
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
		if treeIsGroupNode(node) {
			nodeLevel := treeNodeLevel(node) + 1
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

	if treeIsGroupNode(node) {
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

	session, window, pane, message := nodePathSegments(path)

	switch node.Kind {
	case NodeKindSession:
		return serializeNodeExpansionPath(uimodel.NodeKind(NodeKindSession), session)
	case NodeKindWindow:
		if session == "" {
			return ""
		}
		return serializeNodeExpansionPath(uimodel.NodeKind(NodeKindWindow), session, window)
	case NodeKindPane:
		if session == "" || window == "" {
			return ""
		}
		return serializeNodeExpansionPath(uimodel.NodeKind(NodeKindPane), session, window, pane)
	case NodeKindMessage:
		if session == "" || window == "" || pane == "" {
			return ""
		}
		return serializeNodeExpansionPath(uimodel.NodeKind(NodeKindMessage), session, window, pane, message)
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

	session, window, pane, message := nodePathSegments(path)

	switch node.Kind {
	case NodeKindSession:
		return serializeLegacyNodeExpansionPath(uimodel.NodeKind(NodeKindSession), session)
	case NodeKindWindow:
		if session == "" {
			return ""
		}
		return serializeLegacyNodeExpansionPath(uimodel.NodeKind(NodeKindWindow), session, window)
	case NodeKindPane:
		if session == "" || window == "" {
			return ""
		}
		return serializeLegacyNodeExpansionPath(uimodel.NodeKind(NodeKindPane), session, window, pane)
	case NodeKindMessage:
		if session == "" || window == "" || pane == "" {
			return ""
		}
		return serializeLegacyNodeExpansionPath(uimodel.NodeKind(NodeKindMessage), session, window, pane, message)
	default:
		return ""
	}
}

func treeIsGroupNode(node *Node) bool {
	if node == nil {
		return false
	}
	return node.Kind != NodeKindNotification && node.Kind != NodeKindRoot
}

func treeNodeLevel(node *Node) int {
	if node == nil {
		return 0
	}
	switch node.Kind {
	case NodeKindSession:
		return 0
	case NodeKindWindow:
		return 1
	case NodeKindPane:
		return 2
	case NodeKindMessage:
		return 3
	default:
		return 0
	}
}

func expandTree(node *Node) {
	if node == nil {
		return
	}
	if node.Kind != NodeKindNotification {
		node.Expanded = true
	}
	for _, child := range node.Children {
		expandTree(child)
	}
}

func findNodePath(root *Node, target *Node) ([]*Node, bool) {
	if root == nil || target == nil {
		return nil, false
	}
	if root == target {
		return []*Node{root}, true
	}
	for _, child := range root.Children {
		childPath, ok := findNodePath(child, target)
		if !ok {
			continue
		}
		return append([]*Node{root}, childPath...), true
	}
	return nil, false
}

func nodePathSegments(path []*Node) (session string, window string, pane string, message string) {
	for _, current := range path {
		switch current.Kind {
		case NodeKindSession:
			session = current.Title
		case NodeKindWindow:
			window = current.Title
		case NodeKindPane:
			pane = current.Title
		case NodeKindMessage:
			message = current.Title
		}
	}
	return session, window, pane, message
}
