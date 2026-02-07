// Package settings provides TUI settings management and persistence.
package settings

// TUIState represents the TUI model state that can be persisted.
// This DTO pattern avoids tight coupling between internal/settings and cmd/tmux-intray packages.
type TUIState struct {
	// Columns defines which columns are displayed and their order.
	Columns []string

	// SortBy specifies which column to sort by.
	SortBy string

	// SortOrder specifies sort direction: "asc" or "desc".
	SortOrder string

	// Filters contains active filter criteria.
	Filters Filter

	// ViewMode specifies the display layout: "compact" or "detailed".
	ViewMode string
}

// FromSettings converts Settings to TUIState.
func FromSettings(s *Settings) TUIState {
	if s == nil {
		return TUIState{}
	}
	return TUIState{
		Columns:   s.Columns,
		SortBy:    s.SortBy,
		SortOrder: s.SortOrder,
		Filters:   s.Filters,
		ViewMode:  s.ViewMode,
	}
}

// ToSettings converts TUIState to Settings.
// Returns a Settings struct with the values from TUIState.
// If values are empty, they will use defaults when loaded/saved.
func (t TUIState) ToSettings() *Settings {
	return &Settings{
		Columns:   t.Columns,
		SortBy:    t.SortBy,
		SortOrder: t.SortOrder,
		Filters:   t.Filters,
		ViewMode:  t.ViewMode,
	}
}

// IsEmpty returns true if all fields in TUIState are empty or zero-length.
func (t TUIState) IsEmpty() bool {
	return len(t.Columns) == 0 &&
		t.SortBy == "" &&
		t.SortOrder == "" &&
		t.ViewMode == "" &&
		t.Filters.Level == "" &&
		t.Filters.State == "" &&
		t.Filters.Session == "" &&
		t.Filters.Window == "" &&
		t.Filters.Pane == ""
}
