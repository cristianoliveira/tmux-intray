package service

// ModelInterface defines the methods that the CommandService needs from the Model.
// This allows us to avoid circular dependencies and keeps the CommandService decoupled.
type ModelInterface interface {
	// SaveSettings saves the current settings to disk.
	SaveSettings() error

	// GetGroupBy returns the current group-by setting.
	GetGroupBy() string

	// SetGroupBy sets the group-by setting.
	SetGroupBy(groupBy string) error

	// ApplySearchFilter applies the current search filter.
	ApplySearchFilter()

	// ResetCursor resets the cursor to the first item.
	ResetCursor()

	// GetExpandLevel returns the current expand level setting.
	GetExpandLevel() int

	// SetExpandLevel sets the expand level setting.
	SetExpandLevel(level int) error

	// IsGroupedView returns true if the current view is grouped.
	IsGroupedView() bool

	// ApplyDefaultExpansion applies the default expansion to the tree.
	ApplyDefaultExpansion()

	// ToggleViewMode toggles between view modes.
	ToggleViewMode() error

	// GetViewMode returns the current view mode.
	GetViewMode() string

	// GetReadFilter returns the current read filter value.
	GetReadFilter() string

	// SetReadFilter updates the read filter value.
	SetReadFilter(value string) error
}
