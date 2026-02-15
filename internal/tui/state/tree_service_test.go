package state

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeServiceBuildFilteredTreeRespectsExpansionState(t *testing.T) {
	svc := newTreeService()
	notifications := []notification.Notification{
		{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "one"},
		{ID: 2, Session: "$1", Window: "@1", Pane: "%2", Message: "two"},
	}

	expansionState := map[string]bool{"session:$1": false}
	root := svc.buildFilteredTree(notifications, settings.GroupByPane, expansionState)
	require.NotNil(t, root)

	visible := svc.computeVisibleNodes(root)
	require.NotEmpty(t, visible)
	assert.Equal(t, NodeKindSession, visible[0].Kind)
	assert.False(t, visible[0].Expanded)
	assert.Len(t, visible, 1)
}

func TestTreeServiceGetNodeIdentifier(t *testing.T) {
	svc := newTreeService()
	notifications := []notification.Notification{{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "one"}}
	root := svc.buildFilteredTree(notifications, settings.GroupByPane, map[string]bool{})
	visible := svc.computeVisibleNodes(root)

	var notifNode *Node
	for _, node := range visible {
		if node.Kind == NodeKindNotification {
			notifNode = node
			break
		}
	}
	require.NotNil(t, notifNode)

	assert.Equal(t, "notif:1", svc.getNodeIdentifier(root, notifNode))
}

func TestTreeServiceMessageNodeExpansionKeys(t *testing.T) {
	svc := newTreeService()
	notifs := []notification.Notification{
		{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "duplicate"},
		{ID: 2, Session: "$1", Window: "@1", Pane: "%2", Message: "duplicate"},
	}

	root := svc.buildFilteredTree(notifs, settings.GroupByMessage, map[string]bool{})
	require.NotNil(t, root)

	session := findStateNode(root, NodeKindSession, "$1")
	require.NotNil(t, session)
	window := findStateNode(session, NodeKindWindow, "@1")
	require.NotNil(t, window)
	paneOne := findStateNode(window, NodeKindPane, "%1")
	paneTwo := findStateNode(window, NodeKindPane, "%2")
	require.NotNil(t, paneOne)
	require.NotNil(t, paneTwo)
	messageOne := findStateNode(paneOne, NodeKindMessage, "duplicate")
	messageTwo := findStateNode(paneTwo, NodeKindMessage, "duplicate")
	require.NotNil(t, messageOne)
	require.NotNil(t, messageTwo)

	keyOne := svc.nodeExpansionKey(root, messageOne)
	keyTwo := svc.nodeExpansionKey(root, messageTwo)
	require.NotEmpty(t, keyOne)
	require.NotEmpty(t, keyTwo)
	assert.NotEqual(t, keyOne, keyTwo)

	expansionState := map[string]bool{keyOne: false, keyTwo: true}
	svc.applyExpansionState(root, root, expansionState)
	assert.False(t, messageOne.Expanded)
	assert.True(t, messageTwo.Expanded)
}

func findStateNode(node *Node, kind NodeKind, title string) *Node {
	if node == nil {
		return nil
	}
	for _, child := range node.Children {
		if child.Kind == kind && child.Title == title {
			return child
		}
	}
	return nil
}
