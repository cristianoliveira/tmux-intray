package service

import (
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

func (s *DefaultTreeService) updateUnreadCount(node *model.TreeNode, notif notification.Notification) {
	if notif.IsRead() {
		return
	}
	node.UnreadCount++
}

func (s *DefaultTreeService) updateTimeRange(node *model.TreeNode, notif notification.Notification) {
	if notif.Timestamp == "" {
		return
	}
	if node.LatestEvent == nil || s.isNewerTimestamp(notif.Timestamp, node.LatestEvent.Timestamp) {
		node.LatestEvent = &notif
	}
	if node.EarliestEvent == nil || s.isOlderTimestamp(notif.Timestamp, node.EarliestEvent.Timestamp) {
		node.EarliestEvent = &notif
	}
}

func (s *DefaultTreeService) updateLevelCounts(node *model.TreeNode, notif notification.Notification) {
	if node.LevelCounts == nil {
		node.LevelCounts = make(map[string]int)
	}
	level := notif.Level
	if level == "" {
		level = settings.LevelFilterInfo
	}
	node.LevelCounts[level]++
}

func (s *DefaultTreeService) updateSourceSet(node *model.TreeNode, notif notification.Notification) {
	if notif.Session == "" && notif.Window == "" && notif.Pane == "" {
		return
	}
	if node.Sources == nil {
		node.Sources = make(map[string]model.NotificationSource)
	}
	src := model.NotificationSource{Session: notif.Session, Window: notif.Window, Pane: notif.Pane}
	node.Sources[src.SourceKey()] = src
}
