package state

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

type keyBindingContext int

const (
	keyBindingContextDefault keyBindingContext = iota
	keyBindingContextSearchView
	keyBindingContextSearchInput
)

type keyBindingPolicy struct {
	allowBindings bool
	ctrlFallsBack bool
}

// handleKeyMsg processes keyboard input for the TUI.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle confirmation mode first
	if m.uiState.IsConfirmationMode() {
		return m.handleConfirmation(msg)
	}

	// In search view mode we want `v` to keep cycling view modes.
	// This is a special case because normal search mode treats runes as input.
	if m.shouldCycleViewModeInSearchInput(msg) {
		m.cycleViewMode()
		return m, nil
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

	bindingKey, allowBindings := m.bindingKeyForMsg(msg)
	if !allowBindings {
		// In search mode, only text input is handled unless Ctrl is held.
		return m, nil
	}

	return m.handleKeyBinding(bindingKey, m.uiState.IsSearchMode())
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
		// In search contexts, Ctrl+h moves cursor left (same as normal navigation)
		if m.isSearchContext() {
			return m, nil // Left movement not needed for vertical navigation
		}
		return m, nil
	case tea.KeyCtrlJ:
		// In search contexts, Ctrl+j moves cursor down
		if m.isSearchContext() {
			m.handleMoveDown()
		}
		return m, nil
	case tea.KeyCtrlK:
		// In search contexts, Ctrl+k moves cursor up
		if m.isSearchContext() {
			m.handleMoveUp()
		}
		return m, nil
	case tea.KeyCtrlL:
		// In search contexts, Ctrl+l moves cursor right (same as normal navigation)
		if m.isSearchContext() {
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
func (m *Model) handleKeyBinding(key string, allowInSearch bool) (tea.Model, tea.Cmd) {
	switch key {
	case "j":
		m.handleMoveDown()
		return m, nil
	case "k":
		m.handleMoveUp()
		return m, nil
	case "G":
		return m.handleBindingWithCheck(m.handleMoveBottom, allowInSearch)
	case "g":
		return m.handleBindingWithCheck(func() {
			m.uiState.SetPendingKey("g")
		}, allowInSearch)
	case "/":
		viewModeBeforeSearch := m.uiState.GetViewMode()
		m.handleSearchViewMode()
		m.uiState.SetViewMode(viewModeBeforeSearch)
		return m, nil
	case "?":
		m.uiState.SetShowHelp(!m.uiState.ShowHelp())
		return m, nil
	case "d":
		return m, m.handleDismiss()
	case "D":
		return m, m.handleDismissGroup()
	case "r":
		m.switchActiveTab(settings.TabRecents)
		return m, nil
	case "a":
		m.switchActiveTab(settings.TabAll)
		return m, nil
	case "R":
		return m, m.markSelectedRead()
	case "u":
		return m, m.markSelectedUnread()
	case "v":
		return m.handleBindingWithCheck(m.cycleViewMode, allowInSearch)
	case "h":
		m.handleCollapseNode()
		return m, nil
	case "l":
		m.handleExpandNode()
		return m, nil
	case "z":
		if (allowInSearch || m.canProcessBinding()) && m.isGroupedView() {
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
func (m *Model) handleBindingWithCheck(fn func(), allowInSearch bool) (tea.Model, tea.Cmd) {
	if allowInSearch || m.canProcessBinding() {
		fn()
	}
	return m, nil
}

// handlePendingKey handles multi-key sequences (gg, za, zz, etc.).
func (m *Model) handlePendingKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	key, allowBindings := m.bindingKeyForMsg(msg)
	if !allowBindings {
		m.uiState.ClearPendingKey()
		return false, nil
	}
	if m.uiState.GetPendingKey() != "" {
		if key == "a" && m.uiState.GetPendingKey() == "z" && m.isGroupedView() {
			m.uiState.ClearPendingKey()
			m.toggleFold()
			return true, nil
		}
		if key == "g" && m.uiState.GetPendingKey() == "g" {
			m.uiState.ClearPendingKey()
			m.handleMoveTop()
			return true, nil
		}
		if m.uiState.GetPendingKey() != "z" || key != "z" {
			m.uiState.ClearPendingKey()
		}
	}
	return false, nil
}

func (m *Model) bindingKeyForMsg(msg tea.KeyMsg) (string, bool) {
	key := msg.String()
	policy := m.keyBindingPolicyForContext(m.currentKeyBindingContext())

	if policy.ctrlFallsBack && strings.HasPrefix(key, "ctrl+") {
		fallback := strings.TrimPrefix(key, "ctrl+")
		if len([]rune(fallback)) == 1 {
			return fallback, true
		}
	}

	return key, policy.allowBindings
}

func (m *Model) isSearchContext() bool {
	return m.currentKeyBindingContext() != keyBindingContextDefault
}

func (m *Model) shouldCycleViewModeInSearchInput(msg tea.KeyMsg) bool {
	if m.currentKeyBindingContext() != keyBindingContextSearchInput {
		return false
	}
	if m.uiState.GetViewMode() != model.ViewModeSearch {
		return false
	}
	return msg.Type == tea.KeyRunes && msg.String() == "v"
}

func (m *Model) currentKeyBindingContext() keyBindingContext {
	if m.uiState.IsSearchMode() {
		return keyBindingContextSearchInput
	}
	if m.uiState.GetViewMode() == model.ViewModeSearch {
		return keyBindingContextSearchView
	}
	return keyBindingContextDefault
}

func (m *Model) keyBindingPolicyForContext(context keyBindingContext) keyBindingPolicy {
	switch context {
	case keyBindingContextSearchInput:
		return keyBindingPolicy{allowBindings: false, ctrlFallsBack: true}
	case keyBindingContextSearchView:
		return keyBindingPolicy{allowBindings: true, ctrlFallsBack: true}
	default:
		return keyBindingPolicy{allowBindings: true, ctrlFallsBack: false}
	}
}
