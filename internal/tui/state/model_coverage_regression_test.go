package state

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeViewPublicWrappersAndTabCycle(t *testing.T) {
	setupConfig(t, t.TempDir())

	m := newTestModel(t, []notification.Notification{
		{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "one"},
	})
	m.uiState.SetWidth(80)
	m.uiState.GetViewport().Width = 80

	assert.Equal(t, settings.ViewModeDetailed, m.GetViewMode())
	assert.False(t, m.IsGroupedView())

	require.NoError(t, m.ToggleViewMode())
	assert.Equal(t, settings.ViewModeGrouped, m.GetViewMode())
	assert.True(t, m.IsGroupedView())

	m.uiState.SetGroupBy(settings.GroupByPane)
	m.applySearchFilter()
	m.resetCursor()
	require.NotEmpty(t, m.computeVisibleNodes())

	m.ApplyDefaultExpansion()

	m.uiState.SetActiveTab(settings.TabRecents)
	m.cycleActiveTab()
	assert.Equal(t, settings.TabAll, m.uiState.GetActiveTab())
	m.cycleActiveTab()
	assert.Equal(t, settings.TabRecents, m.uiState.GetActiveTab())
}

func TestHandleConfirmationRunesBranches(t *testing.T) {
	m := newTestModel(t, nil)
	m.uiState.SetConfirmationMode(true)

	next, cmd := m.handleConfirmation(tea.KeyMsg{Type: tea.KeyRunes})
	assert.Same(t, m, next)
	assert.Nil(t, cmd)
	assert.True(t, m.uiState.IsConfirmationMode())

	next, cmd = m.handleConfirmation(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.Same(t, m, next)
	assert.Nil(t, cmd)
	assert.False(t, m.uiState.IsConfirmationMode())
}

func TestHandleTabSwitchingKeysToRecents(t *testing.T) {
	setupConfig(t, t.TempDir())

	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "one"}})
	m.uiState.SetActiveTab(settings.TabAll)

	next, cmd := m.handleTabSwitchingKeys("r")
	assert.Same(t, m, next)
	assert.Nil(t, cmd)
	assert.Equal(t, settings.TabRecents, m.uiState.GetActiveTab())
}

func TestHandleBackspaceSearchModeBranches(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "hello"}})
	m.uiState.SetSearchMode(true)
	m.uiState.SetSearchQuery("ab")
	m.uiState.SetCursor(1)

	m.handleBackspace()
	assert.Equal(t, "a", m.uiState.GetSearchQuery())
	assert.Equal(t, 0, m.uiState.GetCursor())

	m.uiState.SetSearchQuery("")
	m.uiState.SetCursor(1)
	m.handleBackspace()
	assert.Equal(t, "", m.uiState.GetSearchQuery())
	assert.Equal(t, 1, m.uiState.GetCursor())
}

func TestSwitchActiveTabNoopWhenSameTab(t *testing.T) {
	m := newTestModel(t, []notification.Notification{{ID: 1, Message: "one"}})
	m.uiState.SetActiveTab(settings.TabRecents)
	m.uiState.SetCursor(1)

	m.switchActiveTab(settings.TabRecents)

	assert.Equal(t, settings.TabRecents, m.uiState.GetActiveTab())
	assert.Equal(t, 1, m.uiState.GetCursor())
}
