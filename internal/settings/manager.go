// Package settings provides TUI settings management and persistence.
package settings

// TUIState represents the TUI model state that can be persisted.
// This DTO pattern avoids tight coupling between internal/settings and cmd/tmux-intray packages.
type TUIState struct {
	// Columns defines which columns are displayed and their order.
	Columns []string `json:"columns"`

	// SortBy specifies which column to sort by.
	SortBy string `json:"sortBy"`

	// SortOrder specifies sort direction: "asc" or "desc".
	SortOrder string `json:"sortOrder"`

	// Filters contains active filter criteria.
	Filters Filter `json:"filters"`

	// ViewMode specifies the display layout: "compact", "detailed", or "grouped".
	ViewMode string `json:"viewMode"`

	// GroupBy specifies the grouping mode: "none", "session", "window", or "pane".
	GroupBy string `json:"groupBy"`

	// DefaultExpandLevel controls the default grouping expansion level (0-3).
	DefaultExpandLevel int `json:"defaultExpandLevel"`

	// DefaultExpandLevelSet indicates DefaultExpandLevel was explicitly provided.
	DefaultExpandLevelSet bool `json:"-"`

	// AutoExpandUnread controls whether groups with unread notifications are auto-expanded.
	AutoExpandUnread bool `json:"autoExpandUnread"`

	// ExpansionState stores explicit expansion overrides by node path.
	ExpansionState map[string]bool `json:"expansionState"`
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
		t.Filters.Session == "" &&
		t.Filters.Window == "" &&
		t.Filters.Pane == ""
}
