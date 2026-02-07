package state

import (
	"sort"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
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
	LatestEvent  *notification.Notification
}

// BuildTree groups notifications into a session/window/pane hierarchy.
func BuildTree(notifications []notification.Notification) *Node {
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

		sessionNode := getOrCreateGroupNode(root, sessionNodes, NodeKindSession, current.Session)
		incrementGroupStats(sessionNode, current)

		windowKey := current.Session + "\x00" + current.Window
		windowNode := getOrCreateGroupNode(sessionNode, windowNodes, NodeKindWindow, windowKey, current.Window)
		incrementGroupStats(windowNode, current)

		paneKey := current.Session + "\x00" + current.Window + "\x00" + current.Pane
		paneNode := getOrCreateGroupNode(windowNode, paneNodes, NodeKindPane, paneKey, current.Pane)
		incrementGroupStats(paneNode, current)

		leaf := &Node{
			Kind:         NodeKindNotification,
			Title:        current.Message,
			Display:      current.Message,
			Notification: &current,
		}
		paneNode.Children = append(paneNode.Children, leaf)

		incrementGroupStats(root, current)
	}

	sortTree(root)
	return root
}

// FindNotificationPath locates the notification node and returns the path.
func FindNotificationPath(root *Node, notif notification.Notification) ([]*Node, bool) {
	if root == nil {
		return nil, false
	}

	sessionNode := findChildByTitle(root, NodeKindSession, notif.Session)
	if sessionNode == nil {
		return nil, false
	}

	windowNode := findChildByTitle(sessionNode, NodeKindWindow, notif.Window)
	if windowNode == nil {
		return nil, false
	}

	paneNode := findChildByTitle(windowNode, NodeKindPane, notif.Pane)
	if paneNode == nil {
		return nil, false
	}

	leafNode := findChildByNotification(paneNode, notif)
	if leafNode == nil {
		return nil, false
	}

	return []*Node{root, sessionNode, windowNode, paneNode, leafNode}, true
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

// FIXME: Storing pointer to loop variable is safe due to escape analysis, but consider storing notification by value to avoid heap allocations.
func incrementGroupStats(node *Node, notif notification.Notification) {
	node.Count++
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

func findChildByTitle(node *Node, kind NodeKind, title string) *Node {
	for _, child := range node.Children {
		if child.Kind == kind && child.Title == title {
			return child
		}
	}
	return nil
}

func findChildByNotification(node *Node, notif notification.Notification) *Node {
	for _, child := range node.Children {
		if child.Kind != NodeKindNotification || child.Notification == nil {
			continue
		}
		if child.Notification.ID == notif.ID {
			return child
		}
		if child.Notification.Timestamp == notif.Timestamp &&
			child.Notification.Message == notif.Message &&
			child.Notification.Session == notif.Session &&
			child.Notification.Window == notif.Window &&
			child.Notification.Pane == notif.Pane {
			return child
		}
	}
	return nil
}
