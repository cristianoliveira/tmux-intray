// Package settings provides TUI settings management and persistence.
package settings

// TUIState represents the TUI model state that can be persisted.
// This DTO pattern avoids tight coupling between internal/settings and cmd/tmux-intray packages.
type TUIState struct {
	// Columns defines which columns are displayed and their order.
	Columns []string `toml:"columns"`

	// SortBy specifies which column to sort by.
	SortBy string `toml:"sort_by"`

	// SortOrder specifies sort direction: "asc" or "desc".
	SortOrder string `toml:"sort_order"`

	// Filters contains active filter criteria.
	Filters Filter `toml:"filters"`

	// ViewMode specifies the display layout: "compact", "detailed", or "grouped".
	ViewMode string `toml:"view_mode"`

	// GroupBy specifies the grouping mode: "none", "session", "window", "pane", "message", or "pane_message".
	GroupBy string `toml:"group_by"`

	// DefaultExpandLevel controls the default grouping expansion level (0-3).
	DefaultExpandLevel int `toml:"default_expand_level"`

	// DefaultExpandLevelSet indicates DefaultExpandLevel was explicitly provided.
	DefaultExpandLevelSet bool `toml:"-"`

	// AutoExpandUnread controls whether groups with unread notifications are auto-expanded.
	AutoExpandUnread bool `toml:"auto_expand_unread"`

	// ShowHelp controls whether help text is shown in footer.
	ShowHelp bool `toml:"show_help"`

	// ExpansionState stores explicit expansion overrides by node path.
	ExpansionState map[string]bool `toml:"expansion_state"`
}

// FromSettings converts Settings to TUIState.
func FromSettings(s *Settings) TUIState {
	if s == nil {
		return TUIState{}
	}
	return TUIState{
		Columns:               s.Columns,
		SortBy:                s.SortBy,
		SortOrder:             s.SortOrder,
		Filters:               s.Filters,
		ViewMode:              s.ViewMode,
		GroupBy:               s.GroupBy,
		DefaultExpandLevel:    s.DefaultExpandLevel,
		DefaultExpandLevelSet: true,
		AutoExpandUnread:      s.AutoExpandUnread,
		ShowHelp:              s.ShowHelp,
		ExpansionState:        s.ExpansionState,
	}
}

// ToSettings converts TUIState to Settings.
// Returns a Settings struct with the values from TUIState.
// If values are empty, they will use defaults when loaded/saved.
func (t TUIState) ToSettings() *Settings {
	defaultExpandLevel := 0
	if t.DefaultExpandLevelSet {
		defaultExpandLevel = t.DefaultExpandLevel
	}
	return &Settings{
		Columns:            t.Columns,
		SortBy:             t.SortBy,
		SortOrder:          t.SortOrder,
		Filters:            t.Filters,
		ViewMode:           t.ViewMode,
		GroupBy:            t.GroupBy,
		DefaultExpandLevel: defaultExpandLevel,
		AutoExpandUnread:   t.AutoExpandUnread,
		ShowHelp:           t.ShowHelp,
		ExpansionState:     t.ExpansionState,
	}
}

// IsEmpty returns true if all fields in TUIState are empty or zero-length.
func (t TUIState) IsEmpty() bool {
	return len(t.Columns) == 0 &&
		t.SortBy == "" &&
		t.SortOrder == "" &&
		t.ViewMode == "" &&
		t.GroupBy == "" &&
		!t.DefaultExpandLevelSet &&
		len(t.ExpansionState) == 0 &&
		t.Filters.Level == "" &&
		t.Filters.State == "" &&
		t.Filters.Read == "" &&
		t.Filters.Session == "" &&
		t.Filters.Window == "" &&
		t.Filters.Pane == ""
}
