// Package service provides implementations of TUI service interfaces.
package service

import (
	"sort"

	"github.com/cristianoliveira/tmux-intray/internal/dedup"
	"github.com/cristianoliveira/tmux-intray/internal/dedupconfig"
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

// POC: Hardcoded limit for messages per pane.
const maxMessagesPerPane = 3

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

	includeSession := resolvedGroupBy == settings.GroupBySession ||
		resolvedGroupBy == settings.GroupByWindow ||
		resolvedGroupBy == settings.GroupByPane ||
		resolvedGroupBy == settings.GroupByMessage
	includeWindow := resolvedGroupBy == settings.GroupByWindow ||
		resolvedGroupBy == settings.GroupByPane ||
		resolvedGroupBy == settings.GroupByMessage
	includePane := resolvedGroupBy == settings.GroupByPane ||
		resolvedGroupBy == settings.GroupByMessage
	groupByMessage := resolvedGroupBy == settings.GroupByMessage

	var messageKeys []string
	if groupByMessage {
		records := buildNotificationDedupRecords(notifications)
		messageKeys = dedup.BuildKeys(records, dedupconfig.Load())
	}

	for idx, notif := range notifications {
		current := notif
		parent := root

		paneKey := ""

		if includeSession {
			sessionNode := s.getOrCreateGroupNode(root, sessionNodes, model.NodeKindSession, current.Session)
			s.incrementGroupStats(sessionNode, current)
			parent = sessionNode
		}

		if includeWindow {
			windowKey := current.Session + "\x00" + current.Window
			windowNode := s.getOrCreateGroupNode(parent, windowNodes, model.NodeKindWindow, windowKey, current.Window)
			s.incrementGroupStats(windowNode, current)
			parent = windowNode
		}

		if includePane {
			paneKey = current.Session + "\x00" + current.Window + "\x00" + current.Pane
			paneNode := s.getOrCreateGroupNode(parent, paneNodes, model.NodeKindPane, paneKey, current.Pane)
			s.incrementGroupStats(paneNode, current)
			parent = paneNode
		}

		if groupByMessage {
			if messageNode := s.attachMessageNode(parent, current, idx, messageKeys, paneKey, messageNodes); messageNode != nil {
				parent = messageNode
			}
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
	s.limitMessages(root)

	// Update all group counts after limiting children
	s.updateAllGroupCounts(root)

	s.treeRoot = root
	s.InvalidateCache()
	return nil
}

// updateAllGroupCounts updates counts for all group nodes after limiting.
func (s *DefaultTreeService) updateAllGroupCounts(root *model.TreeNode) {
	if root == nil {
		return
	}

	if root.Kind != model.NodeKindRoot && root.Kind != model.NodeKindNotification {
		// Recalculate from children
		newCount := 0
		newUnreadCount := 0
		for _, child := range root.Children {
			if child.Kind == model.NodeKindNotification {
				// Notification node: count as 1, check if unread
				newCount++
				if child.Notification != nil && !child.Notification.IsRead() {
					newUnreadCount++
				}
			} else {
				// Group node: use its counts
				newCount += child.Count
				newUnreadCount += child.UnreadCount
			}
		}
		root.Count = newCount
		root.UnreadCount = newUnreadCount
	}

	for _, child := range root.Children {
		s.updateAllGroupCounts(child)
	}
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
		return 3
	default:
		return 0
	}
}

// limitMessages limits the number of messages shown per pane to maxMessagesPerPane.
// Messages are sorted by unread status (unread first) then timestamp (newest first).
func (s *DefaultTreeService) limitMessages(node *model.TreeNode) {
	if node == nil {
		return
	}

	totalMessages := len(node.Children)

	// Recursively process ALL nodes
	for _, child := range node.Children {
		s.limitMessages(child)
	}

	// Only limit at window or pane level, not session/root
	if node.Kind != model.NodeKindWindow && node.Kind != model.NodeKindPane {
		return
	}

	if totalMessages <= maxMessagesPerPane {
		return
	}

	// Sort all children by timestamp desc (most recent first), prioritizing unread
	allChildren := make([]*model.TreeNode, len(node.Children))
	copy(allChildren, node.Children)

	sort.SliceStable(allChildren, func(i, j int) bool {
		// Unread always first
		iUnread := allChildren[i].Notification != nil && !allChildren[i].Notification.IsRead()
		jUnread := allChildren[j].Notification != nil && !allChildren[j].Notification.IsRead()
		if iUnread != jUnread {
			return iUnread
		}
		// Then sort by timestamp desc
		ti := allChildren[i].Notification.Timestamp
		tj := allChildren[j].Notification.Timestamp
		return ti > tj
	})

	// Take only the first maxMessagesPerPane messages
	if len(allChildren) > maxMessagesPerPane {
		allChildren = allChildren[:maxMessagesPerPane]
	}

	node.Children = allChildren

	// Update node counts to reflect limited children
	s.updateNodeCountAfterLimit(node)
}

// updateNodeCountAfterLimit updates Count and UnreadCount after limiting children.
func (s *DefaultTreeService) updateNodeCountAfterLimit(node *model.TreeNode) {
	if node == nil {
		return
	}

	// Only update for window/pane nodes
	if node.Kind != model.NodeKindWindow && node.Kind != model.NodeKindPane {
		return
	}

	// Recalculate counts from current children
	newCount := 0
	newUnreadCount := 0
	for _, child := range node.Children {
		if child.Kind == model.NodeKindNotification {
			newCount++
			if child.Notification != nil && !child.Notification.IsRead() {
				newUnreadCount++
			}
		} else {
			// For group children, add their counts
			newCount += child.Count
			newUnreadCount += child.UnreadCount
		}
	}

	node.Count = newCount
	node.UnreadCount = newUnreadCount
}
