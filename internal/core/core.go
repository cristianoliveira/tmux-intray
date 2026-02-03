// Package core provides core tmux interaction and tray management.
package core

// GetTrayItems returns tray items for a given state filter.
func GetTrayItems(stateFilter string) string {
	_ = stateFilter
	return ""
}

// AddTrayItem adds a tray item.
func AddTrayItem(item, session, window, pane, paneCreated string, noAuto bool, level string) {
	_ = item
	_ = session
	_ = window
	_ = pane
	_ = paneCreated
	_ = noAuto
	_ = level
}

// ClearTrayItems dismisses all active tray items.
func ClearTrayItems() {
}

// GetVisibility returns the visibility state.
func GetVisibility() string {
	return ""
}

// SetVisibility sets the visibility state.
func SetVisibility(visible string) {
	_ = visible
}
