package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings *Settings
		want     TUIState
	}{
		{
			name:     "nil settings",
			settings: nil,
			want:     TUIState{},
		},
		{
			name:     "default settings",
			settings: DefaultSettings(),
			want: TUIState{
				Columns:            DefaultColumns,
				SortBy:             SortByTimestamp,
				SortOrder:          SortOrderDesc,
				Filters:            Filter{},
				ViewMode:           ViewModeCompact,
				GroupBy:            GroupByNone,
				DefaultExpandLevel: 1,
				ExpansionState:     map[string]bool{},
			},
		},
		{
			name: "custom settings",
			settings: &Settings{
				Columns:   []string{ColumnID, ColumnMessage, ColumnLevel},
				SortBy:    SortByLevel,
				SortOrder: SortOrderAsc,
				Filters: Filter{
					Level:   LevelFilterWarning,
					State:   StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode:           ViewModeDetailed,
				GroupBy:            GroupBySession,
				DefaultExpandLevel: 2,
				ExpansionState: map[string]bool{
					"session:$1": true,
				},
			},
			want: TUIState{
				Columns:   []string{ColumnID, ColumnMessage, ColumnLevel},
				SortBy:    SortByLevel,
				SortOrder: SortOrderAsc,
				Filters: Filter{
					Level:   LevelFilterWarning,
					State:   StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode:           ViewModeDetailed,
				GroupBy:            GroupBySession,
				DefaultExpandLevel: 2,
				ExpansionState: map[string]bool{
					"session:$1": true,
				},
			},
		},
		{
			name: "empty settings",
			settings: &Settings{
				Columns:   []string{},
				SortBy:    "",
				SortOrder: "",
				Filters:   Filter{},
				ViewMode:  "",
				GroupBy:   "",
			},
			want: TUIState{
				Columns:   []string{},
				SortBy:    "",
				SortOrder: "",
				Filters:   Filter{},
				ViewMode:  "",
				GroupBy:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromSettings(tt.settings)
			assert.Equal(t, tt.want.Columns, got.Columns)
			assert.Equal(t, tt.want.SortBy, got.SortBy)
			assert.Equal(t, tt.want.SortOrder, got.SortOrder)
			assert.Equal(t, tt.want.Filters, got.Filters)
			assert.Equal(t, tt.want.ViewMode, got.ViewMode)
			assert.Equal(t, tt.want.GroupBy, got.GroupBy)
			assert.Equal(t, tt.want.DefaultExpandLevel, got.DefaultExpandLevel)
			assert.Equal(t, tt.want.ExpansionState, got.ExpansionState)
		})
	}
}

func TestToSettings(t *testing.T) {
	tests := []struct {
		name  string
		state TUIState
		want  *Settings
	}{
		{
			name:  "empty state",
			state: TUIState{},
			want: &Settings{
				Columns:            nil, // Empty state has nil columns
				SortBy:             "",
				SortOrder:          "",
				Filters:            Filter{},
				ViewMode:           "",
				GroupBy:            "",
				DefaultExpandLevel: 0,
				ExpansionState:     nil,
			},
		},
		{
			name: "custom state",
			state: TUIState{
				Columns:   []string{ColumnID, ColumnMessage, ColumnLevel},
				SortBy:    SortByLevel,
				SortOrder: SortOrderAsc,
				Filters: Filter{
					Level:   LevelFilterWarning,
					State:   StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode:           ViewModeDetailed,
				GroupBy:            GroupByWindow,
				DefaultExpandLevel: 3,
				ExpansionState: map[string]bool{
					"window:@1": true,
				},
			},
			want: &Settings{
				Columns:   []string{ColumnID, ColumnMessage, ColumnLevel},
				SortBy:    SortByLevel,
				SortOrder: SortOrderAsc,
				Filters: Filter{
					Level:   LevelFilterWarning,
					State:   StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode:           ViewModeDetailed,
				GroupBy:            GroupByWindow,
				DefaultExpandLevel: 3,
				ExpansionState: map[string]bool{
					"window:@1": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.ToSettings()
			assert.Equal(t, tt.want.Columns, got.Columns)
			assert.Equal(t, tt.want.SortBy, got.SortBy)
			assert.Equal(t, tt.want.SortOrder, got.SortOrder)
			assert.Equal(t, tt.want.Filters, got.Filters)
			assert.Equal(t, tt.want.ViewMode, got.ViewMode)
			assert.Equal(t, tt.want.GroupBy, got.GroupBy)
			assert.Equal(t, tt.want.DefaultExpandLevel, got.DefaultExpandLevel)
			assert.Equal(t, tt.want.ExpansionState, got.ExpansionState)
		})
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		state TUIState
		want  bool
	}{
		{
			name:  "completely empty",
			state: TUIState{},
			want:  true,
		},
		{
			name: "has columns",
			state: TUIState{
				Columns: []string{ColumnID, ColumnMessage},
			},
			want: false,
		},
		{
			name: "has sortBy",
			state: TUIState{
				SortBy: SortByLevel,
			},
			want: false,
		},
		{
			name: "has sortOrder",
			state: TUIState{
				SortOrder: SortOrderAsc,
			},
			want: false,
		},
		{
			name: "has viewMode",
			state: TUIState{
				ViewMode: ViewModeDetailed,
			},
			want: false,
		},
		{
			name: "has groupBy",
			state: TUIState{
				GroupBy: GroupByWindow,
			},
			want: false,
		},
		{
			name: "has defaultExpandLevel",
			state: TUIState{
				DefaultExpandLevel: 2,
			},
			want: false,
		},
		{
			name: "has expansion state",
			state: TUIState{
				ExpansionState: map[string]bool{"session:$1": true},
			},
			want: false,
		},
		{
			name: "has filter level",
			state: TUIState{
				Filters: Filter{Level: LevelFilterWarning},
			},
			want: false,
		},
		{
			name: "has filter state",
			state: TUIState{
				Filters: Filter{State: StateFilterActive},
			},
			want: false,
		},
		{
			name: "has filter session",
			state: TUIState{
				Filters: Filter{Session: "my-session"},
			},
			want: false,
		},
		{
			name: "has filter window",
			state: TUIState{
				Filters: Filter{Window: "@1"},
			},
			want: false,
		},
		{
			name: "has filter pane",
			state: TUIState{
				Filters: Filter{Pane: "%1"},
			},
			want: false,
		},
		{
			name: "all fields populated",
			state: TUIState{
				Columns:            []string{ColumnID, ColumnMessage},
				SortBy:             SortByLevel,
				SortOrder:          SortOrderAsc,
				ViewMode:           ViewModeDetailed,
				GroupBy:            GroupByPane,
				DefaultExpandLevel: 3,
				ExpansionState:     map[string]bool{"pane:%1": true},
				Filters: Filter{
					Level:   LevelFilterWarning,
					State:   StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.IsEmpty()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	tests := []struct {
		name     string
		settings *Settings
	}{
		{
			name:     "default settings",
			settings: DefaultSettings(),
		},
		{
			name: "custom settings",
			settings: &Settings{
				Columns:   []string{ColumnID, ColumnMessage, ColumnLevel},
				SortBy:    SortByLevel,
				SortOrder: SortOrderAsc,
				Filters: Filter{
					Level:   LevelFilterWarning,
					State:   StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode:           ViewModeDetailed,
				GroupBy:            GroupBySession,
				DefaultExpandLevel: 2,
				ExpansionState: map[string]bool{
					"session:$1": true,
				},
			},
		},
		{
			name: "partial settings",
			settings: &Settings{
				SortBy:   SortByLevel,
				ViewMode: ViewModeDetailed,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Settings -> TUIState -> Settings
			state := FromSettings(tt.settings)
			result := state.ToSettings()

			assert.Equal(t, tt.settings.Columns, result.Columns)
			assert.Equal(t, tt.settings.SortBy, result.SortBy)
			assert.Equal(t, tt.settings.SortOrder, result.SortOrder)
			assert.Equal(t, tt.settings.Filters, result.Filters)
			assert.Equal(t, tt.settings.ViewMode, result.ViewMode)
			assert.Equal(t, tt.settings.GroupBy, result.GroupBy)
			assert.Equal(t, tt.settings.DefaultExpandLevel, result.DefaultExpandLevel)
			assert.Equal(t, tt.settings.ExpansionState, result.ExpansionState)
		})
	}
}

func TestPartialTUIStateConversion(t *testing.T) {
	// Test that partial settings (empty fields) are preserved
	original := &Settings{
		Columns:   []string{ColumnID, ColumnMessage, ColumnLevel},
		SortBy:    SortByLevel,
		SortOrder: SortOrderAsc,
		Filters: Filter{
			Level:   LevelFilterWarning,
			State:   StateFilterActive,
			Session: "my-session",
		},
		ViewMode:           ViewModeDetailed,
		GroupBy:            GroupBySession,
		DefaultExpandLevel: 2,
		ExpansionState: map[string]bool{
			"session:$1": true,
		},
	}

	// Convert to TUIState
	state := FromSettings(original)
	require.Equal(t, original.Columns, state.Columns)
	require.Equal(t, original.SortBy, state.SortBy)
	require.Equal(t, original.SortOrder, state.SortOrder)
	require.Equal(t, original.Filters, state.Filters)
	require.Equal(t, original.ViewMode, state.ViewMode)
	require.Equal(t, original.GroupBy, state.GroupBy)
	require.Equal(t, original.DefaultExpandLevel, state.DefaultExpandLevel)
	require.Equal(t, original.ExpansionState, state.ExpansionState)

	// Partially clear some fields to simulate partial updates
	state.Filters.Session = ""
	state.Filters.Window = "@1"
	state.Filters.Pane = "%1"
	state.ExpansionState = map[string]bool{}

	// Convert back to Settings
	result := state.ToSettings()

	// Verify partial updates are preserved
	assert.Equal(t, original.Columns, result.Columns)
	assert.Equal(t, original.SortBy, result.SortBy)
	assert.Equal(t, original.SortOrder, result.SortOrder)
	assert.Equal(t, original.Filters.Level, result.Filters.Level)
	assert.Equal(t, original.Filters.State, result.Filters.State)
	assert.Equal(t, "", result.Filters.Session)
	assert.Equal(t, "@1", result.Filters.Window)
	assert.Equal(t, "%1", result.Filters.Pane)
	assert.Equal(t, original.ViewMode, result.ViewMode)
	assert.Equal(t, original.GroupBy, result.GroupBy)
	assert.Equal(t, original.DefaultExpandLevel, result.DefaultExpandLevel)
	assert.Equal(t, map[string]bool{}, result.ExpansionState)
}
