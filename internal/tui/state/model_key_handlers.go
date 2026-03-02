package state

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// handleCtrlC handles Ctrl+C to exit the TUI.
func (m *Model) handleCtrlC() (tea.Model, tea.Cmd) {
	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		return m, tea.Batch(tea.Quit, errorMsgAfter(errorClearDuration))
	}
	return m, tea.Quit
}

// handleEsc handles Escape to exit search mode or quit.
func (m *Model) handleEsc() (tea.Model, tea.Cmd) {
	if m.uiState.IsSearchMode() {
		m.uiState.SetSearchMode(false)
		m.applySearchFilter()
		m.uiState.ResetCursor()
	} else {
		return m, tea.Quit
	}
	return m, nil
}

// handleEnter handles Enter to confirm search or jump to pane.
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.uiState.IsSearchMode() {
		// In search view mode, Enter should immediately perform jump.
		if m.uiState.GetViewMode() == model.ViewModeSearch {
			return m, m.handleJump()
		}
		// In other view modes, Enter confirms/exits search input.
		m.uiState.SetSearchMode(false)
		m.applySearchFilter()
		m.uiState.ResetCursor()
		return m, nil
	}
	return m, m.handleJump()
}

// handleRunes handles rune input (character keys and search text).
func (m *Model) handleRunes(msg tea.KeyMsg) {
	if m.uiState.IsSearchMode() {
		// In search mode, append runes to search query
		for _, r := range msg.Runes {
			m.uiState.AppendToSearchQuery(r)
		}
		m.applySearchFilter()
		m.uiState.ResetCursor()
	}
}

// handleBackspace handles backspace to delete characters in search mode.
func (m *Model) handleBackspace() {
	if m.uiState.IsSearchMode() {
		if len(m.uiState.GetSearchQuery()) > 0 {
			m.uiState.BackspaceSearchQuery()
			m.applySearchFilter()
			m.uiState.ResetCursor()
		}
	}
}

// handleQuit handles quit action, saving settings first.
func (m *Model) handleQuit() (tea.Model, tea.Cmd) {
	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		return m, tea.Batch(tea.Quit, errorMsgAfter(errorClearDuration))
	}
	// Quit
	return m, tea.Quit
}

func (m *Model) switchActiveTab(tab settings.Tab) {
	nextTab := settings.NormalizeTab(string(tab))
	if m.uiState.GetActiveTab() == nextTab {
		return
	}

	m.uiState.SetActiveTab(nextTab)
	m.applySearchFilter()
	m.resetCursor()

	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}
}

// handleSaveSettingsSuccess handles successful settings save (no-op).
func (m *Model) handleSaveSettingsSuccess(msg saveSettingsSuccessMsg) (tea.Model, tea.Cmd) {
	// Settings saved successfully - already displayed info message in saveSettings
	return m, nil
}

// handleSaveSettingsFailed handles failed settings save (no-op).
func (m *Model) handleSaveSettingsFailed(msg saveSettingsFailedMsg) (tea.Model, tea.Cmd) {
	// Settings save failed - already displayed warning message in saveSettings
	return m, nil
}

// handleWindowSizeMsg handles window resize events.
func (m *Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.uiState.SetWidth(msg.Width)
	m.uiState.SetHeight(msg.Height)
	// Initialize or update viewport dimensions
	m.uiState.UpdateViewportSize()
	// Update viewport content
	m.updateViewportContent()
	return m, nil
}
