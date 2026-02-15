// Package service provides implementations of TUI service interfaces.
package service

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// DefaultTreeService implements the TreeService interface.
type DefaultTreeService struct {
	groupBy           model.GroupBy
	treeRoot          *model.TreeNode
	visibleNodes      []*model.TreeNode
	visibleNodesCache []*model.TreeNode
	cacheValid        bool
}

// NewTreeService creates a new DefaultTreeService.
func NewTreeService(groupBy model.GroupBy) model.TreeService {
	return &DefaultTreeService{
		groupBy: groupBy,
	}
}

// BuildTree creates a tree structure from a list of notifications.
func (s *DefaultTreeService) BuildTree(notifications []notification.Notification, groupBy string) error {
	resolvedGroupBy := s.resolveGroupBy(groupBy)

	root := &model.TreeNode{
		Kind:     model.NodeKindRoot,
		Title:    "root",
		Display:  "root",
		Expanded: true,
	}

	sessionNodes := make(map[string]*model.TreeNode)
	windowNodes := make(map[string]*model.TreeNode)
	paneNodes := make(map[string]*model.TreeNode)
	messageNodes := make(map[string]*model.TreeNode)

	for _, notif := range notifications {
		current := notif
		parent := root

		if resolvedGroupBy == settings.GroupBySession || resolvedGroupBy == settings.GroupByWindow || resolvedGroupBy == settings.GroupByPane {
			sessionNode := s.getOrCreateGroupNode(root, sessionNodes, model.NodeKindSession, current.Session)
			s.incrementGroupStats(sessionNode, current)
			parent = sessionNode
		}

		if resolvedGroupBy == settings.GroupByWindow || resolvedGroupBy == settings.GroupByPane {
			windowKey := current.Session + "\x00" + current.Window
			windowNode := s.getOrCreateGroupNode(parent, windowNodes, model.NodeKindWindow, windowKey, current.Window)
			s.incrementGroupStats(windowNode, current)
			parent = windowNode
		}

		if resolvedGroupBy == settings.GroupByPane {
			paneKey := current.Session + "\x00" + current.Window + "\x00" + current.Pane
			paneNode := s.getOrCreateGroupNode(parent, paneNodes, model.NodeKindPane, paneKey, current.Pane)
			s.incrementGroupStats(paneNode, current)
			parent = paneNode
		}

		if resolvedGroupBy == settings.GroupByMessage {
			messageNode := s.getOrCreateGroupNode(root, messageNodes, model.NodeKindMessage, current.Message, current.Message)
			s.incrementGroupStats(messageNode, current)
			parent = messageNode
		}

		leaf := &model.TreeNode{
			Kind:         model.NodeKindNotification,
			Title:        current.Message,
			Display:      current.Message,
			Notification: &current,
		}
		parent.Children = append(parent.Children, leaf)

		s.incrementGroupStats(root, current)
	}

	s.sortTree(root)
	s.treeRoot = root
	s.InvalidateCache()
	return nil
}

// RebuildTreeForFilter rebuilds tree and applies filtering-oriented behavior.
func (s *DefaultTreeService) RebuildTreeForFilter(notifications []notification.Notification, groupBy string, expansionState map[string]bool) error {
	if len(notifications) == 0 {
		s.ClearTree()
		return nil
	}

	if err := s.BuildTree(notifications, groupBy); err != nil {
		s.ClearTree()
		return err
	}

	s.PruneEmptyGroups()
	if len(expansionState) > 0 {
		s.ApplyExpansionState(expansionState)
		return nil
	}

	s.expandAllGroups(s.treeRoot)
	s.InvalidateCache()
	return nil
}

// ClearTree clears all internally managed tree state and cache.
func (s *DefaultTreeService) ClearTree() {
	s.treeRoot = nil
	s.visibleNodes = nil
	s.visibleNodesCache = nil
	s.cacheValid = false
}

// GetTreeRoot returns the current internally managed tree root.
func (s *DefaultTreeService) GetTreeRoot() *model.TreeNode {
	return s.treeRoot
}

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

// GetVisibleNodes returns all nodes that should be visible in the UI.
func (s *DefaultTreeService) GetVisibleNodes() []*model.TreeNode {
	if s.cacheValid {
		return s.visibleNodesCache
	}

	if s.treeRoot == nil {
		s.visibleNodes = nil
		s.visibleNodesCache = nil
		s.cacheValid = true
		return nil
	}

	var visible []*model.TreeNode
	var walk func(node *model.TreeNode)
	walk = func(node *model.TreeNode) {
		if node == nil {
			return
		}
		if node.Kind != model.NodeKindRoot {
			visible = append(visible, node)
		}
		if node.Kind == model.NodeKindNotification {
			return
		}
		if node.Kind != model.NodeKindRoot && !node.Expanded {
			return
		}
		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(s.treeRoot)
	s.visibleNodes = visible
	s.visibleNodesCache = visible
	s.cacheValid = true
	return s.visibleNodesCache
}

// InvalidateCache invalidates visible nodes cache.
func (s *DefaultTreeService) InvalidateCache() {
	s.visibleNodes = nil
	s.visibleNodesCache = nil
	s.cacheValid = false
}

// GetNodeIdentifier returns a stable identifier for a node.
func (s *DefaultTreeService) GetNodeIdentifier(node *model.TreeNode) string {
	if node == nil {
		return ""
	}
	if node.Kind == model.NodeKindNotification && node.Notification != nil {
		return fmt.Sprintf("notif:%d", node.Notification.ID)
	}
	// For group nodes, use the node kind and title
	// This is a simplified version - the full implementation needs path tracking
	if node.Kind == model.NodeKindRoot {
		return "root"
	}
	return fmt.Sprintf("%s:%s", node.Kind, node.Title)
}

// PruneEmptyGroups removes group nodes with no children from the tree.
func (s *DefaultTreeService) PruneEmptyGroups() {
	if s.treeRoot == nil {
		return
	}

	// Recursively prune children first
	var filteredChildren []*model.TreeNode
	for _, child := range s.treeRoot.Children {
		s.pruneEmptyGroupsNode(child)
		// Keep the child if it has children (even if it's a leaf with notifications)
		// or if it's a notification node
		if len(child.Children) > 0 || child.Kind == model.NodeKindNotification {
			filteredChildren = append(filteredChildren, child)
		}
	}
	s.treeRoot.Children = filteredChildren
	s.InvalidateCache()
}

func (s *DefaultTreeService) pruneEmptyGroupsNode(root *model.TreeNode) {
	if root == nil {
		return
	}

	var filteredChildren []*model.TreeNode
	for _, child := range root.Children {
		s.pruneEmptyGroupsNode(child)
		if len(child.Children) > 0 || child.Kind == model.NodeKindNotification {
			filteredChildren = append(filteredChildren, child)
		}
	}
	root.Children = filteredChildren
}

// ApplyExpansionState applies saved expansion state to tree nodes.
func (s *DefaultTreeService) ApplyExpansionState(expansionState map[string]bool) {
	if s.treeRoot == nil {
		return
	}
	s.applyExpansionStateNode(s.treeRoot, expansionState)
	s.InvalidateCache()
}

func (s *DefaultTreeService) applyExpansionStateNode(root *model.TreeNode, expansionState map[string]bool) {
	if root == nil {
		return
	}

	// Apply expansion state to group nodes
	if s.isGroupNode(root) {
		if expanded, ok := s.expansionStateValue(root, expansionState); ok {
			root.Expanded = expanded
		} else {
			// Default to expanded for nodes without saved state
			root.Expanded = true
		}
	}

	// Recursively apply to children
	for _, child := range root.Children {
		s.applyExpansionStateNode(child, expansionState)
	}
}

// ExpandNode expands a group node.
func (s *DefaultTreeService) ExpandNode(node *model.TreeNode) {
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	if node.Expanded {
		return
	}
	node.Expanded = true
	s.InvalidateCache()
}

// CollapseNode collapses a group node.
func (s *DefaultTreeService) CollapseNode(node *model.TreeNode) {
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	if !node.Expanded {
		return
	}
	node.Expanded = false
	s.InvalidateCache()
}

// ToggleNodeExpansion toggles the expansion state of a group node.
func (s *DefaultTreeService) ToggleNodeExpansion(node *model.TreeNode) {
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	node.Expanded = !node.Expanded
	s.InvalidateCache()
}

// GetTreeLevel returns the depth level of a node in the tree.
func (s *DefaultTreeService) GetTreeLevel(node *model.TreeNode) int {
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
		return 0
	default:
		return 0
	}
}

// Helper methods

func (s *DefaultTreeService) resolveGroupBy(groupBy string) string {
	if !settings.IsValidGroupBy(groupBy) {
		return settings.GroupByPane
	}
	return groupBy
}

func (s *DefaultTreeService) getOrCreateGroupNode(parent *model.TreeNode, cache map[string]*model.TreeNode, kind model.NodeKind, key string, titles ...string) *model.TreeNode {
	if node, ok := cache[key]; ok {
		return node
	}

	title := key
	if len(titles) > 0 {
		title = titles[0]
	}

	node := &model.TreeNode{
		Kind:    kind,
		Title:   title,
		Display: title,
	}
	parent.Children = append(parent.Children, node)
	cache[key] = node
	return node
}

func (s *DefaultTreeService) incrementGroupStats(node *model.TreeNode, notif notification.Notification) {
	node.Count++
	if !notif.IsRead() {
		node.UnreadCount++
	}
	if node.LatestEvent == nil || s.isNewerTimestamp(notif.Timestamp, node.LatestEvent.Timestamp) {
		node.LatestEvent = &notif
	}
}

func (s *DefaultTreeService) isNewerTimestamp(current string, latest string) bool {
	if current == "" {
		return false
	}
	if latest == "" {
		return true
	}
	return current > latest
}

func (s *DefaultTreeService) sortTree(node *model.TreeNode) {
	if node == nil {
		return
	}

	if len(node.Children) > 1 {
		// Sort children alphabetically by title
		// In Go, we need to do this in-place
		for i := 0; i < len(node.Children); i++ {
			for j := i + 1; j < len(node.Children); j++ {
				if strings.ToLower(node.Children[i].Title) > strings.ToLower(node.Children[j].Title) {
					node.Children[i], node.Children[j] = node.Children[j], node.Children[i]
				}
			}
		}
	}

	for _, child := range node.Children {
		s.sortTree(child)
	}
}

func (s *DefaultTreeService) isGroupNode(node *model.TreeNode) bool {
	if node == nil {
		return false
	}
	return node.Kind != model.NodeKindNotification && node.Kind != model.NodeKindRoot
}

func (s *DefaultTreeService) expansionStateValue(node *model.TreeNode, expansionState map[string]bool) (bool, bool) {
	if expansionState == nil {
		return false, false
	}

	key := s.GetNodeIdentifier(node)
	if key != "" {
		expanded, ok := expansionState[key]
		if ok {
			return expanded, true
		}
	}

	return false, false
}

func (s *DefaultTreeService) expandAllGroups(node *model.TreeNode) {
	if node == nil {
		return
	}
	if s.isGroupNode(node) {
		node.Expanded = true
	}
	for _, child := range node.Children {
		s.expandAllGroups(child)
	}
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
