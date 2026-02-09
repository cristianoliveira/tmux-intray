package service

import (
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

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
}

// CommandServiceAdapter adapts the state.Model to implement ModelInterface.
type CommandServiceAdapter struct {
	model interface{} // Using interface{} to avoid circular import
}

// NewCommandServiceAdapter creates a new adapter for the given model.
func NewCommandServiceAdapter(model interface{}) ModelInterface {
	return &CommandServiceAdapter{model: model}
}

// SaveSettings saves the current settings to disk.
func (a *CommandServiceAdapter) SaveSettings() error {
	// Use type assertion to access the unexported saveSettings method
	if m, ok := a.model.(interface{ saveSettings() error }); ok {
		return m.saveSettings()
	}
	return nil
}

// GetGroupBy returns the current group-by setting.
func (a *CommandServiceAdapter) GetGroupBy() string {
	if m, ok := a.model.(interface{ getGroupBy() model.GroupBy }); ok {
		return string(m.getGroupBy())
	}
	return ""
}

// SetGroupBy sets the group-by setting.
func (a *CommandServiceAdapter) SetGroupBy(groupBy string) error {
	// This would need to be implemented in the Model
	return nil
}

// ApplySearchFilter applies the current search filter.
func (a *CommandServiceAdapter) ApplySearchFilter() {
	if m, ok := a.model.(interface{ applySearchFilter() }); ok {
		m.applySearchFilter()
	}
}

// ResetCursor resets the cursor to the first item.
func (a *CommandServiceAdapter) ResetCursor() {
	if m, ok := a.model.(interface{ resetCursor() }); ok {
		m.resetCursor()
	}
}

// GetExpandLevel returns the current expand level setting.
func (a *CommandServiceAdapter) GetExpandLevel() int {
	if m, ok := a.model.(interface{ getExpandLevel() int }); ok {
		return m.getExpandLevel()
	}
	return 0
}

// SetExpandLevel sets the expand level setting.
func (a *CommandServiceAdapter) SetExpandLevel(level int) error {
	// This would need to be implemented in the Model
	return nil
}

// IsGroupedView returns true if the current view is grouped.
func (a *CommandServiceAdapter) IsGroupedView() bool {
	if m, ok := a.model.(interface{ isGroupedView() bool }); ok {
		return m.isGroupedView()
	}
	return false
}

// ApplyDefaultExpansion applies the default expansion to the tree.
func (a *CommandServiceAdapter) ApplyDefaultExpansion() {
	if m, ok := a.model.(interface{ applyDefaultExpansion() }); ok {
		m.applyDefaultExpansion()
	}
}

// ToggleViewMode toggles between view modes.
func (a *CommandServiceAdapter) ToggleViewMode() error {
	// This would need to be implemented in the Model
	return nil
}

// GetViewMode returns the current view mode.
func (a *CommandServiceAdapter) GetViewMode() string {
	if m, ok := a.model.(interface{ getViewMode() model.ViewMode }); ok {
		return string(m.getViewMode())
	}
	return ""
}
