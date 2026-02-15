// Package model provides interface contracts for TUI components.
// These interfaces define the contracts between different parts of the TUI system.
package model

import (
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// TreeService defines the interface for tree building and management operations.
// It handles the hierarchical organization of notifications by session/window/pane.
type TreeService interface {
	// BuildTree creates a tree structure from a list of notifications.
	// The groupBy parameter determines the grouping depth (session, window, or pane).
	// The resulting tree is stored internally by the service.
	BuildTree(notifications []notification.Notification, groupBy string) error

	// RebuildTreeForFilter rebuilds the tree for filtered notifications, pruning empty
	// groups and applying expansion state (or expanding all nodes when no state exists).
	RebuildTreeForFilter(notifications []notification.Notification, groupBy string, expansionState map[string]bool) error

	// ClearTree clears all internally stored tree state and cache.
	ClearTree()

	// GetTreeRoot returns the current tree root managed by the service.
	GetTreeRoot() *TreeNode

	// FindNotificationPath locates a notification in the tree and returns the path.
	// Returns a slice of nodes from root to the notification, or an error if not found.
	FindNotificationPath(root *TreeNode, notif notification.Notification) ([]*TreeNode, error)

	// FindNodeByID finds a tree node by its unique identifier.
	// Returns the node or nil if not found.
	FindNodeByID(root *TreeNode, identifier string) *TreeNode

	// GetVisibleNodes returns all nodes that should be visible in the UI.
	// This respects expansion state of group nodes and uses an internal cache.
	GetVisibleNodes() []*TreeNode

	// InvalidateCache invalidates the internal visible nodes cache.
	InvalidateCache()

	// GetNodeIdentifier returns a stable identifier for a node.
	// Used for cursor restoration after tree updates.
	GetNodeIdentifier(node *TreeNode) string

	// PruneEmptyGroups removes group nodes with no children from the tree.
	// Updates the internally stored root.
	PruneEmptyGroups()

	// ApplyExpansionState applies saved expansion state to tree nodes.
	// Takes a map of node identifiers to expanded state.
	ApplyExpansionState(expansionState map[string]bool)

	// ExpandNode expands a group node.
	// Does nothing if the node is already expanded or is a notification node.
	ExpandNode(node *TreeNode)

	// CollapseNode collapses a group node.
	// Does nothing if the node is already collapsed or is a notification node.
	CollapseNode(node *TreeNode)

	// ToggleNodeExpansion toggles the expansion state of a group node.
	ToggleNodeExpansion(node *TreeNode)

	// GetTreeLevel returns the depth level of a node in the tree.
	// Root is level 0, session nodes are level 0 in their context, etc.
	GetTreeLevel(node *TreeNode) int
}

// TreeNode represents a node in the notification tree hierarchy.
type TreeNode struct {
	// Kind is the type of node (root, session, window, pane, or notification).
	Kind NodeKind

	// Title is the display title of the node.
	Title string

	// Display is an alternative display string (e.g., for sorting).
	Display string

	// Expanded indicates whether a group node is expanded in the UI.
	Expanded bool

	// Children are the child nodes of this node.
	Children []*TreeNode

	// Notification is the notification data for leaf nodes.
	Notification *notification.Notification

	// Count is the total number of notifications under this group node.
	Count int

	// UnreadCount is the number of unread notifications under this group node.
	UnreadCount int

	// LatestEvent is the most recent notification under this group node.
	LatestEvent *notification.Notification
}

// NodeKind represents the type of a tree node.
type NodeKind string

const (
	// NodeKindRoot represents the root of the tree.
	NodeKindRoot NodeKind = "root"

	// NodeKindSession represents a session group node.
	NodeKindSession NodeKind = "session"

	// NodeKindWindow represents a window group node.
	NodeKindWindow NodeKind = "window"

	// NodeKindPane represents a pane group node.
	NodeKindPane NodeKind = "pane"

	// NodeKindMessage represents a message group node.
	NodeKindMessage NodeKind = "message"

	// NodeKindNotification represents a leaf node containing a notification.
	NodeKindNotification NodeKind = "notification"
)
