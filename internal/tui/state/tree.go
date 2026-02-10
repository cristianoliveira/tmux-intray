package state

import (
	"sort"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// NodeKind represents the type of a tree node.
type NodeKind string

const (
	NodeKindRoot         NodeKind = "root"
	NodeKindSession      NodeKind = "session"
	NodeKindWindow       NodeKind = "window"
	NodeKindPane         NodeKind = "pane"
	NodeKindNotification NodeKind = "notification"
)

// Node represents a tree node for hierarchical notification grouping.
type Node struct {
	Kind         NodeKind
	Title        string
	Display      string
	Expanded     bool
	Children     []*Node
	Notification *notification.Notification
	Count        int
	UnreadCount  int
	LatestEvent  *notification.Notification
}

// BuildTree groups notifications according to the configured groupBy depth.
func BuildTree(notifications []notification.Notification, groupBy string) *Node {
	resolvedGroupBy := resolveGroupBy(groupBy)

	root := &Node{
		Kind:     NodeKindRoot,
		Title:    "root",
		Display:  "root",
		Expanded: true,
	}

	sessionNodes := make(map[string]*Node)
	windowNodes := make(map[string]*Node)
	paneNodes := make(map[string]*Node)

	for _, notif := range notifications {
		current := notif
		parent := root

		if resolvedGroupBy == settings.GroupBySession || resolvedGroupBy == settings.GroupByWindow || resolvedGroupBy == settings.GroupByPane {
			sessionNode := getOrCreateGroupNode(root, sessionNodes, NodeKindSession, current.Session)
			incrementGroupStats(sessionNode, current)
			parent = sessionNode
		}

		if resolvedGroupBy == settings.GroupByWindow || resolvedGroupBy == settings.GroupByPane {
			windowKey := current.Session + "\x00" + current.Window
			windowNode := getOrCreateGroupNode(parent, windowNodes, NodeKindWindow, windowKey, current.Window)
			incrementGroupStats(windowNode, current)
			parent = windowNode
		}

		if resolvedGroupBy == settings.GroupByPane {
			paneKey := current.Session + "\x00" + current.Window + "\x00" + current.Pane
			paneNode := getOrCreateGroupNode(parent, paneNodes, NodeKindPane, paneKey, current.Pane)
			incrementGroupStats(paneNode, current)
			parent = paneNode
		}

		leaf := &Node{
			Kind:         NodeKindNotification,
			Title:        current.Message,
			Display:      current.Message,
			Notification: &current,
		}
		parent.Children = append(parent.Children, leaf)

		incrementGroupStats(root, current)
	}

	sortTree(root)
	return root
}

func resolveGroupBy(groupBy string) string {
	if !settings.IsValidGroupBy(groupBy) {
		return settings.GroupByPane
	}
	return groupBy
}

// FindNotificationPath locates the notification node and returns the path.
func FindNotificationPath(root *Node, notif notification.Notification) ([]*Node, bool) {
	if root == nil {
		return nil, false
	}

	path, ok := findNotificationPath(root, notif)
	if !ok {
		return nil, false
	}

	return path, true
}

func findNotificationPath(node *Node, notif notification.Notification) ([]*Node, bool) {
	if node == nil {
		return nil, false
	}
	if node.Kind == NodeKindNotification && notificationNodeMatches(node, notif) {
		return []*Node{node}, true
	}
	for _, child := range node.Children {
		childPath, ok := findNotificationPath(child, notif)
		if !ok {
			continue
		}
		return append([]*Node{node}, childPath...), true
	}
	return nil, false
}

func getOrCreateGroupNode(parent *Node, cache map[string]*Node, kind NodeKind, key string, titles ...string) *Node {
	if node, ok := cache[key]; ok {
		return node
	}

	title := key
	if len(titles) > 0 {
		title = titles[0]
	}

	node := &Node{
		Kind:    kind,
		Title:   title,
		Display: title,
	}
	parent.Children = append(parent.Children, node)
	cache[key] = node
	return node
}

// incrementGroupStats updates node statistics with notification data.
func incrementGroupStats(node *Node, notif notification.Notification) {
	node.Count++
	if !notif.IsRead() {
		node.UnreadCount++
	}
	if node.LatestEvent == nil || isNewerTimestamp(notif.Timestamp, node.LatestEvent.Timestamp) {
		node.LatestEvent = &notif
	}
}

// Assumes timestamps are ISO 8601 strings lexicographically comparable (e.g., 2024-01-02T10:00:00Z).
func isNewerTimestamp(current string, latest string) bool {
	if current == "" {
		return false
	}
	if latest == "" {
		return true
	}
	return current > latest
}

func sortTree(node *Node) {
	if node == nil {
		return
	}

	if len(node.Children) > 1 {
		sort.SliceStable(node.Children, func(i, j int) bool {
			return strings.ToLower(node.Children[i].Title) < strings.ToLower(node.Children[j].Title)
		})
	}

	for _, child := range node.Children {
		sortTree(child)
	}
}

func findChildByTitle(node *model.TreeNode, kind model.NodeKind, title string) *model.TreeNode {
	for _, child := range node.Children {
		if child.Kind == kind && child.Title == title {
			return child
		}
	}
	return nil
}

func findChildByNotification(node *Node, notif notification.Notification) *Node {
	for _, child := range node.Children {
		if notificationNodeMatches(child, notif) {
			return child
		}
	}
	return nil
}

func notificationNodeMatches(node *Node, notif notification.Notification) bool {
	if node == nil || node.Kind != NodeKindNotification || node.Notification == nil {
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
