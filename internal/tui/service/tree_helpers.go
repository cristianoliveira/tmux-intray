package service

import (
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/dedup"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

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
	if node == nil {
		return
	}
	node.Count++
	s.updateUnreadCount(node, notif)
	s.updateTimeRange(node, notif)
	s.updateLevelCounts(node, notif)
	s.updateSourceSet(node, notif)
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

func (s *DefaultTreeService) isOlderTimestamp(current string, earliest string) bool {
	if current == "" {
		return false
	}
	if earliest == "" {
		return true
	}
	return current < earliest
}

func (s *DefaultTreeService) sortTree(node *model.TreeNode) {
	if node == nil {
		return
	}

	if len(node.Children) > 1 {
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

func buildNotificationDedupRecords(notifs []notification.Notification) []dedup.Record {
	records := make([]dedup.Record, len(notifs))
	for i, n := range notifs {
		records[i] = dedup.Record{
			Message:   n.Message,
			Level:     n.Level,
			Session:   n.Session,
			Window:    n.Window,
			Pane:      n.Pane,
			State:     n.State,
			Timestamp: n.Timestamp,
		}
	}
	return records
}
