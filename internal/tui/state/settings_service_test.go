package state

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsServiceToStatePreservesUnreadFirstAndActiveTab(t *testing.T) {
	svc := newSettingsService()
	ui := NewUIState()
	ui.SetActiveTab(settings.TabAll)

	state := svc.toState(ui, []string{settings.ColumnID}, settings.SortByTimestamp, settings.SortOrderDesc, false, settings.Filter{})

	assert.Equal(t, settings.TabAll, state.ActiveTab)
	assert.False(t, state.UnreadFirst)
}

func TestSettingsServiceFromStateAppliesUnreadFirstAndNormalizesTab(t *testing.T) {
	svc := newSettingsService()
	ui := NewUIState()

	columns := []string{}
	sortBy := ""
	sortOrder := ""
	unreadFirst := true
	filters := settings.Filter{}

	err := svc.fromState(settings.TUIState{UnreadFirst: false, ActiveTab: settings.Tab("invalid")}, ui, &columns, &sortBy, &sortOrder, &unreadFirst, &filters)
	require.NoError(t, err)

	assert.False(t, unreadFirst)
	assert.Equal(t, settings.TabRecents, ui.GetActiveTab())
}
