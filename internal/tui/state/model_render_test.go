package state

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
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

func TestViewRendersTabControlsAndActiveIndicator(t *testing.T) {
	model := newTestModel(t, []notification.Notification{{ID: 1, Message: "First", State: "active"}})
	model.uiState.SetWidth(80)
	model.uiState.SetHeight(24)
	model.uiState.UpdateViewportSize()
	model.applySearchFilter()

	view := model.View()
	assert.Contains(t, view, "Tabs:")
	assert.Contains(t, view, "[Recents]")
	assert.Contains(t, view, "All")

	model.uiState.SetActiveTab(settings.TabAll)
	model.applySearchFilter()

	view = model.View()
	assert.Contains(t, view, "Recents")
	assert.Contains(t, view, "[All]")
}
