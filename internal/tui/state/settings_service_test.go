package state

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsServiceToStateIncludesNormalizedActiveTab(t *testing.T) {
	svc := newSettingsService()
	ui := NewUIState()
	ui.SetActiveTab(settings.TabAll)

	state := svc.toState(ui, []string{settings.ColumnID}, settings.SortByTimestamp, settings.SortOrderDesc, true, settings.Filter{})

	assert.Equal(t, settings.TabAll, state.ActiveTab)
	assert.Equal(t, []string{settings.ColumnID}, state.Columns)
	assert.Equal(t, settings.SortByTimestamp, state.SortBy)
	assert.Equal(t, true, state.UnreadFirst)
}

func TestSettingsServiceFromStateNormalizesActiveTab(t *testing.T) {
	tests := []struct {
		name   string
		tab    settings.Tab
		expect settings.Tab
	}{
		{name: "valid", tab: settings.TabAll, expect: settings.TabAll},
		{name: "missing", tab: "", expect: settings.TabRecents},
		{name: "invalid", tab: settings.Tab("bad"), expect: settings.TabRecents},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newSettingsService()
			ui := NewUIState()
			ui.SetActiveTab(settings.TabAll)

			columns := []string{}
			sortBy := ""
			sortOrder := ""
			unreadFirst := true
			filters := settings.Filter{}

			err := svc.fromState(settings.TUIState{ActiveTab: tt.tab}, ui, &columns, &sortBy, &sortOrder, &unreadFirst, &filters)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, ui.GetActiveTab())
		})
	}
}

func TestSettingsServicePreservesUnreadFirst(t *testing.T) {
	svc := newSettingsService()
	ui := NewUIState()
	ui.SetActiveTab(settings.TabAll)

	// Test with unreadFirst = true
	state := svc.toState(ui, []string{settings.ColumnID}, settings.SortByTimestamp, settings.SortOrderDesc, true, settings.Filter{})
	assert.Equal(t, true, state.UnreadFirst)

	// Test with unreadFirst = false
	state = svc.toState(ui, []string{settings.ColumnID}, settings.SortByTimestamp, settings.SortOrderDesc, false, settings.Filter{})
	assert.Equal(t, false, state.UnreadFirst)

	// Test fromState preserves UnreadFirst
	ui = NewUIState()
	columns := []string{}
	sortBy := ""
	sortOrder := ""
	unreadFirst := false
	filters := settings.Filter{}

	err := svc.fromState(settings.TUIState{UnreadFirst: true}, ui, &columns, &sortBy, &sortOrder, &unreadFirst, &filters)
	require.NoError(t, err)
	assert.Equal(t, true, unreadFirst)
}
