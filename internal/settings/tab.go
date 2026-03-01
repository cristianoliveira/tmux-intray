package settings

import "strings"

// Tab identifies the active notifications lane in the TUI.
type Tab string

const (
	// TabRecents shows recently active notifications.
	TabRecents Tab = "recents"

	// TabAll shows all notifications.
	TabAll Tab = "all"
)

// IsValid returns whether the tab is one of the supported values.
func (t Tab) IsValid() bool {
	switch t {
	case TabRecents, TabAll:
		return true
	default:
		return false
	}
}

// DefaultTab returns the default tab used when value is missing or invalid.
func DefaultTab() Tab {
	return TabRecents
}

// NormalizeTab converts arbitrary persisted input to a valid tab value.
// Missing or invalid values always resolve to the default tab.
func NormalizeTab(raw string) Tab {
	tab := Tab(strings.ToLower(strings.TrimSpace(raw)))
	if tab.IsValid() {
		return tab
	}

	return DefaultTab()
}
