package state

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// handleMoveDown moves the cursor down by one position.
func (m *Model) handleMoveDown() {
	listLen := m.currentListLen()
	m.uiState.MoveCursorDown(listLen)
	m.updateViewportContent()
	m.uiState.EnsureCursorVisible(listLen)
}

// handleMoveUp moves the cursor up by one position.
func (m *Model) handleMoveUp() {
	listLen := m.currentListLen()
	m.uiState.MoveCursorUp(listLen)
	m.updateViewportContent()
	m.uiState.EnsureCursorVisible(listLen)
}

// handleMoveTop moves the cursor to the top of the list.
func (m *Model) handleMoveTop() {
	listLen := m.currentListLen()
	if listLen == 0 {
		return
	}
	m.uiState.SetCursor(0)
	m.updateViewportContent()
	m.uiState.EnsureCursorVisible(listLen)
}

// handleMoveBottom moves the cursor to the bottom of the list.
func (m *Model) handleMoveBottom() {
	listLen := m.currentListLen()
	if listLen == 0 {
		return
	}
	m.uiState.SetCursor(listLen - 1)
	m.updateViewportContent()
	m.uiState.EnsureCursorVisible(listLen)
}

// handleSearchMode enters or exits search mode.
func (m *Model) handleSearchMode() {
	m.uiState.SetSearchMode(true)
	m.applySearchFilter()
	m.uiState.ResetCursor()
}

// handleSearchViewMode switches to search view mode and focuses the search input.
func (m *Model) handleSearchViewMode() {
	m.uiState.SetViewMode(model.ViewModeSearch)
	m.uiState.SetSearchMode(true)

	// Reset cursor before rendering filtered list, so highlight stays in sync.
	m.resetCursor()
	m.applySearchFilter()

	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}
}

// handleCollapseNode collapses the currently selected tree node.
func (m *Model) handleCollapseNode() {
	node := m.selectedVisibleNode()
	if node != nil {
		m.treeService.CollapseNode(node)
		m.invalidateCache()
		m.updateViewportContent()
	}
}

// handleExpandNode expands the currently selected tree node.
func (m *Model) handleExpandNode() {
	node := m.selectedVisibleNode()
	if node != nil {
		m.treeService.ExpandNode(node)
		m.invalidateCache()
		m.updateViewportContent()
	}
}
