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
