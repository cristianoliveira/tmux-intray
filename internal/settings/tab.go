package settings

import "strings"

// Tab defines available top-level TUI notification tabs.
type Tab string

const (
	TabRecents Tab = "recents"
	TabAll     Tab = "all"
)

// IsValid returns true when the tab value is supported.
func (t Tab) IsValid() bool {
	switch t {
	case TabRecents, TabAll:
		return true
	default:
		return false
	}
}

// DefaultTab returns the default tab selection.
func DefaultTab() Tab {
	return TabRecents
}

// NormalizeTab converts arbitrary persisted input to a valid tab value.
func NormalizeTab(raw string) Tab {
	tab := Tab(strings.ToLower(strings.TrimSpace(raw)))
	if tab.IsValid() {
		return tab
	}

	return DefaultTab()
}
