package state

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// handleKeyMsg processes keyboard input for the TUI.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle confirmation mode first
	if m.uiState.IsConfirmationMode() {
		return m.handleConfirmation(msg)
	}

	if handled, cmd := m.handlePendingKey(msg); handled {
		return m, cmd
	}

	if nextModel, cmd := m.handleKeyType(msg); cmd != nil || nextModel != nil {
		if nextModel == nil {
			nextModel = m
		}
		return nextModel, cmd
	}

	if !m.canProcessBinding() {
		// In search mode, only text input is handled; bindings are ignored.
		return m, nil
	}

	return m.handleKeyBinding(msg.String())
}

// handleConfirmation handles key input during confirmation mode.
func (m *Model) handleConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		// Cancel confirmation and quit
		m.uiState.SetConfirmationMode(false)
		return m.handleCtrlC()
	case tea.KeyEsc:
		// Cancel confirmation
		m.uiState.SetConfirmationMode(false)
		return m, nil
	case tea.KeyEnter:
		// Confirm action
		return m, m.executeConfirmedAction()
	case tea.KeyRunes:
		// Handle y/Y for yes, n/N for no
		if len(msg.Runes) == 0 {
			return m, nil
		}
		switch msg.Runes[0] {
		case 'y', 'Y':
			return m, m.executeConfirmedAction()
		case 'n', 'N':
			m.uiState.SetConfirmationMode(false)
			return m, nil
		}
	}
	return m, nil
}

// executeConfirmedAction executes the action that was confirmed.
func (m *Model) executeConfirmedAction() tea.Cmd {
	action := m.uiState.GetPendingAction()
	m.uiState.SetConfirmationMode(false)

	if action.Type == ActionDismissGroup {
		return m.handleDismissByFilter(action.Session, action.Window, action.Pane)
	} else {
		m.errorHandler.Error(fmt.Sprintf("Unknown action type: %s", action.Type))
		return nil
	}
}

// handleKeyType handles key type-based actions (Ctrl+C, Esc, Enter, etc.).
func (m *Model) handleKeyType(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m.handleCtrlC()
	case tea.KeyEsc:
		return m.handleEsc()
	case tea.KeyEnter:
		return m.handleEnter()
	case tea.KeyRunes:
		m.handleRunes(msg)
		return nil, nil
	case tea.KeyBackspace:
		m.handleBackspace()
		return nil, nil
	case tea.KeyUp, tea.KeyDown:
		return nil, nil
	case tea.KeyCtrlH:
		// In search mode, Ctrl+h moves cursor left (same as normal navigation)
		if m.uiState.IsSearchMode() {
			return m, nil // Left movement not needed for vertical navigation
		}
		return m, nil
	case tea.KeyCtrlJ:
		// In search mode, Ctrl+j moves cursor down
		if m.uiState.IsSearchMode() {
			m.handleMoveDown()
		}
		return m, nil
	case tea.KeyCtrlK:
		// In search mode, Ctrl+k moves cursor up
		if m.uiState.IsSearchMode() {
			m.handleMoveUp()
		}
		return m, nil
	case tea.KeyCtrlL:
		// In search mode, Ctrl+l moves cursor right (same as normal navigation)
		if m.uiState.IsSearchMode() {
			return m, nil // Right movement not needed for vertical navigation
		}
		return m, nil
	}
	return nil, nil
}

// canProcessBinding returns true if the current state allows processing mode-restricted bindings.
func (m *Model) canProcessBinding() bool {
	return !m.uiState.IsSearchMode()
}

// handleKeyBinding handles string-based key bindings.
func (m *Model) handleKeyBinding(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j":
		m.handleMoveDown()
		return m, nil
	case "k":
		m.handleMoveUp()
		return m, nil
	case "G":
		return m.handleBindingWithCheck(m.handleMoveBottom)
	case "g":
		return m.handleBindingWithCheck(func() {
			m.uiState.SetPendingKey("g")
		})
	case "/":
		m.handleSearchMode()
		return m, nil
	case "d":
		return m, m.handleDismiss()
	case "D":
		return m, m.handleDismissGroup()
	case "r":
		return m, m.markSelectedRead()
	case "u":
		return m, m.markSelectedUnread()
	case "v":
		return m.handleBindingWithCheck(m.cycleViewMode)
	case "h":
		m.handleCollapseNode()
		return m, nil
	case "l":
		m.handleExpandNode()
		return m, nil
	case "z":
		if m.canProcessBinding() && m.isGroupedView() {
			m.uiState.SetPendingKey("z")
		}
		return m, nil
	case "i":
		// In search mode, 'i' is handled by KeyRunes
		// This is a no-op but kept for documentation
		return m, nil
	case "q":
		return m.handleQuit()
	}
	return m, nil
}

// handleBindingWithCheck executes a binding if it can be processed.
func (m *Model) handleBindingWithCheck(fn func()) (tea.Model, tea.Cmd) {
	if m.canProcessBinding() {
		fn()
	}
	return m, nil
}

// handlePendingKey handles multi-key sequences (gg, za, zz, etc.).
func (m *Model) handlePendingKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !m.canProcessBinding() {
		m.uiState.ClearPendingKey()
	} else if m.uiState.GetPendingKey() != "" {
		if msg.String() == "a" && m.uiState.GetPendingKey() == "z" && m.isGroupedView() {
			m.uiState.ClearPendingKey()
			m.toggleFold()
			return true, nil
		}
		if msg.String() == "g" && m.uiState.GetPendingKey() == "g" {
			m.uiState.ClearPendingKey()
			m.handleMoveTop()
			return true, nil
		}
		if m.uiState.GetPendingKey() != "z" || msg.String() != "z" {
			m.uiState.ClearPendingKey()
		}
	}
	return false, nil
}
