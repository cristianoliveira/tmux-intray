package state

import (
	"testing"

	tuimodel "github.com/cristianoliveira/tmux-intray/internal/tui/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupSourcesForNodePrefersPaneNames(t *testing.T) {
	m := &Model{}
	node := &tuimodel.TreeNode{
		Sources: map[string]tuimodel.NotificationSource{
			"pane-a": {Pane: "%2"},
			"pane-b": {Pane: "%1"},
		},
	}

	sources := m.groupSourcesForNode(node)
	require.Len(t, sources, 2)
	assert.Equal(t, []string{"%1", "%2"}, sources)
}

func TestGroupSourcesForNodeBuildsCompositeLabels(t *testing.T) {
	m := &Model{}
	node := &tuimodel.TreeNode{
		Sources: map[string]tuimodel.NotificationSource{
			"session-only":   {Session: "$1"},
			"session-window": {Session: "$2", Window: "@3"},
		},
	}

	sources := m.groupSourcesForNode(node)
	require.Len(t, sources, 2)
	assert.Contains(t, sources, "$1")
	assert.Contains(t, sources, "$2:@3")
}
