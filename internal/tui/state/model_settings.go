package state

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// saveSettings extracts current settings from model and saves to disk.
func (m *Model) saveSettings() error {
	// Extract current settings state
	state := m.ToState()
	colors.Debug("Saving settings from TUI state")
	if err := m.ensureSettingsService().save(state); err != nil {
		return err
	}
	m.loadedSettings = state.ToSettings()
	return nil
}

// SaveSettings is the public version of saveSettings.
func (m *Model) SaveSettings() error {
	return m.saveSettings()
}

// GetGroupBy returns the current group-by setting.
func (m *Model) GetGroupBy() string {
	return string(m.uiState.GetGroupBy())
}

// SetGroupBy sets the group-by setting.
func (m *Model) SetGroupBy(groupBy string) error {
	if !settings.IsValidGroupBy(groupBy) {
		return fmt.Errorf("invalid group-by value: %s", groupBy)
	}

	if m.GetGroupBy() == groupBy {
		return nil // Already set
	}

	m.uiState.SetGroupBy(model.GroupBy(groupBy))
	return nil
}

// GetExpandLevel returns the current expand level setting.
func (m *Model) GetExpandLevel() int {
	return m.uiState.GetExpandLevel()
}

// SetExpandLevel sets the expand level setting.
func (m *Model) SetExpandLevel(level int) error {
	if level < settings.MinExpandLevel || level > settings.MaxExpandLevel {
		return fmt.Errorf("invalid expand level value: %d (expected %d-%d)", level, settings.MinExpandLevel, settings.MaxExpandLevel)
	}

	if m.uiState.GetExpandLevel() == level {
		return nil // Already set
	}

	m.uiState.SetExpandLevel(level)
	return nil
}

// GetReadFilter returns the current persisted read filter value.
func (m *Model) GetReadFilter() string {
	return m.filters.Read
}

// SetReadFilter updates the read filter preference.
func (m *Model) SetReadFilter(value string) error {
	normalized := strings.ToLower(value)
	if normalized != "" && normalized != settings.ReadFilterRead && normalized != settings.ReadFilterUnread {
		return fmt.Errorf("invalid read filter value: %s", value)
	}
	if m.filters.Read == normalized {
		return nil
	}
	m.filters.Read = normalized
	return nil
}
