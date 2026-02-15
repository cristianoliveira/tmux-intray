// Package service provides implementations of TUI service interfaces.
package service

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// FindNotificationPath locates a notification in the tree and returns the path.
func (s *DefaultTreeService) FindNotificationPath(root *model.TreeNode, notif notification.Notification) ([]*model.TreeNode, error) {
	if root == nil {
		return nil, fmt.Errorf("root node cannot be nil")
	}

	path, ok := s.findNotificationPathRecursive(root, notif)
	if !ok {
		return nil, fmt.Errorf("notification not found in tree")
	}

	return path, nil
}

// findNotificationPathRecursive is a helper that recursively searches for a notification.
func (s *DefaultTreeService) findNotificationPathRecursive(node *model.TreeNode, notif notification.Notification) ([]*model.TreeNode, bool) {
	if node == nil {
		return nil, false
	}
	if node.Kind == model.NodeKindNotification && s.notificationNodeMatches(node, notif) {
		return []*model.TreeNode{node}, true
	}
	for _, child := range node.Children {
		childPath, ok := s.findNotificationPathRecursive(child, notif)
		if !ok {
			continue
		}
		return append([]*model.TreeNode{node}, childPath...), true
	}
	return nil, false
}

// FindNodeByID finds a tree node by its unique identifier.
func (s *DefaultTreeService) FindNodeByID(root *model.TreeNode, identifier string) *model.TreeNode {
	if root == nil {
		return nil
	}

	return s.findNodeByIDRecursive(root, identifier)
}

// findNodeByIDRecursive is a helper that recursively searches for a node by identifier.
func (s *DefaultTreeService) findNodeByIDRecursive(node *model.TreeNode, identifier string) *model.TreeNode {
	if node == nil {
		return nil
	}

	if s.GetNodeIdentifier(node) == identifier {
		return node
	}

	for _, child := range node.Children {
		if found := s.findNodeByIDRecursive(child, identifier); found != nil {
			return found
		}
	}

	return nil
}

// findNodePath finds the path from root to target node.
func (s *DefaultTreeService) findNodePath(root *model.TreeNode, target *model.TreeNode) ([]*model.TreeNode, bool) {
	if root == nil || target == nil {
		return nil, false
	}
	if root == target {
		return []*model.TreeNode{root}, true
	}
	for _, child := range root.Children {
		childPath, ok := s.findNodePath(child, target)
		if !ok {
			continue
		}
		return append([]*model.TreeNode{root}, childPath...), true
	}
	return nil, false
}

// GetNodeIdentifier returns a stable identifier for a node.
func (s *DefaultTreeService) GetNodeIdentifier(node *model.TreeNode) string {
	if node == nil {
		return ""
	}
	if node.Kind == model.NodeKindNotification && node.Notification != nil {
		return fmt.Sprintf("notif:%d", node.Notification.ID)
	}
	if node.Kind == model.NodeKindRoot {
		return "root"
	}
	path, ok := s.findNodePath(s.treeRoot, node)
	if !ok || len(path) == 0 {
		return fmt.Sprintf("%s:%s", node.Kind, node.Title)
	}
	parts := make([]string, 0, len(path)*2)
	for _, current := range path {
		if current.Kind == model.NodeKindRoot {
			continue
		}
		parts = append(parts, string(current.Kind), current.Title)
	}
	return strings.Join(parts, ":")
}

// notificationNodeMatches checks if a notification node matches the given notification.
func (s *DefaultTreeService) notificationNodeMatches(node *model.TreeNode, notif notification.Notification) bool {
	if node == nil || node.Kind != model.NodeKindNotification || node.Notification == nil {
		return false
	}
	if node.Notification.ID == notif.ID {
		return true
	}
	return node.Notification.Timestamp == notif.Timestamp &&
		node.Notification.Message == notif.Message &&
		node.Notification.Session == notif.Session &&
		node.Notification.Window == notif.Window &&
		node.Notification.Pane == notif.Pane
}
