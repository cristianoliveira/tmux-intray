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

	tests := []struct {
		name        string
		unreadFirst bool
	}{
		{name: "unread first true", unreadFirst: true},
		{name: "unread first false", unreadFirst: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := svc.toState(ui, []string{settings.ColumnID}, settings.SortByTimestamp, settings.SortOrderDesc, tt.unreadFirst, settings.Filter{})

			assert.Equal(t, settings.TabAll, state.ActiveTab)
			assert.Equal(t, []string{settings.ColumnID}, state.Columns)
			assert.Equal(t, settings.SortByTimestamp, state.SortBy)
			assert.Equal(t, tt.unreadFirst, state.UnreadFirst)
		})
	}
}

func TestSettingsServiceFromStateAppliesUnreadFirstAndNormalizesTab(t *testing.T) {
	tests := []struct {
		name        string
		tab         settings.Tab
		unreadFirst bool
		expectTab   settings.Tab
		expectUF    bool
	}{
		{name: "valid tab with unreadFirst true", tab: settings.TabAll, unreadFirst: true, expectTab: settings.TabAll, expectUF: true},
		{name: "valid tab with unreadFirst false", tab: settings.TabAll, unreadFirst: false, expectTab: settings.TabAll, expectUF: false},
		{name: "missing tab with unreadFirst true", tab: "", unreadFirst: true, expectTab: settings.TabRecents, expectUF: true},
		{name: "missing tab with unreadFirst false", tab: "", unreadFirst: false, expectTab: settings.TabRecents, expectUF: false},
		{name: "invalid tab with unreadFirst true", tab: settings.Tab("bad"), unreadFirst: true, expectTab: settings.TabRecents, expectUF: true},
		{name: "invalid tab with unreadFirst false", tab: settings.Tab("bad"), unreadFirst: false, expectTab: settings.TabRecents, expectUF: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newSettingsService()
			ui := NewUIState()
			ui.SetActiveTab(settings.TabAll)

			columns := []string{}
			sortBy := ""
			sortOrder := ""
			unreadFirst := tt.unreadFirst
			filters := settings.Filter{}

			err := svc.fromState(settings.TUIState{ActiveTab: tt.tab, UnreadFirst: tt.unreadFirst}, ui, &columns, &sortBy, &sortOrder, &unreadFirst, &filters)
			require.NoError(t, err)
			assert.Equal(t, tt.expectTab, ui.GetActiveTab())
			assert.Equal(t, tt.expectUF, unreadFirst)
		})
	}
}
