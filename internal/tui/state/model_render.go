package state

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
	"github.com/cristianoliveira/tmux-intray/internal/tui/render"
)

// View renders the TUI.
func (m *Model) View() string {
	if m.uiState.GetWidth() == 0 {
		m.uiState.SetWidth(defaultViewportWidth)
	}
	if m.uiState.GetHeight() == 0 {
		m.uiState.SetHeight(24)
	}

	// Ensure viewport is initialized
	if m.uiState.GetViewport().Height == 0 {
		m.uiState.UpdateViewportSize()
		m.updateViewportContent()
	}

	var s strings.Builder

	// Header
	s.WriteString(render.Header(m.uiState.GetWidth()))

	// Viewport with table rows
	s.WriteString("\n")
	s.WriteString(m.uiState.GetViewport().View())

	// Footer
	s.WriteString("\n")
	s.WriteString(render.Footer(render.FooterState{
		SearchMode:   m.uiState.IsSearchMode(),
		CommandMode:  m.uiState.IsCommandMode(),
		SearchQuery:  m.uiState.GetSearchQuery(),
		CommandQuery: m.uiState.GetCommandQuery(),
		Grouped:      m.isGroupedView(),
		ViewMode:     string(m.uiState.GetViewMode()),
		Width:        m.uiState.GetWidth(),
		ErrorMessage: m.errorMessage,
	}))

	return s.String()
}

// updateViewportContent updates the viewport with the current filtered notifications.
func (m *Model) updateViewportContent() {
	var content strings.Builder
	width := m.uiState.GetWidth()
	cursor := m.uiState.GetCursor()

	if m.isGroupedView() {
		m.renderGroupedView(&content, width, cursor)
		(*m.uiState.GetViewport()).SetContent(content.String())
		return
	}

	m.renderFlatView(&content, width, cursor)
	(*m.uiState.GetViewport()).SetContent(content.String())
}

// renderGroupedView renders the grouped notification tree view.
func (m *Model) renderGroupedView(content *strings.Builder, width, cursor int) {
	visibleNodes := m.treeService.GetVisibleNodes()
	if len(visibleNodes) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No notifications found"))
		return
	}

	now := time.Now()
	for rowIndex, node := range visibleNodes {
		if node == nil {
			continue
		}
		if rowIndex > 0 {
			content.WriteString("\n")
		}
		if m.isGroupNode(node) {
			m.renderGroupNodeRow(content, node, rowIndex, cursor, width)
			continue
		}
		if node.Notification == nil {
			continue
		}
		m.renderNotificationRow(content, *node.Notification, rowIndex, cursor, width, now)
	}
}

// renderGroupNodeRow renders a single group node row.
func (m *Model) renderGroupNodeRow(content *strings.Builder, node *model.TreeNode, rowIndex, cursor, width int) {
	display := node.Display
	switch node.Kind {
	case model.NodeKindSession:
		display = m.getSessionName(node.Title)
	case model.NodeKindWindow:
		display = m.getWindowName(node.Title)
	case model.NodeKindPane:
		display = m.getPaneName(node.Title)
	}
	content.WriteString(render.RenderGroupRow(render.GroupRow{
		Node: &render.GroupNode{
			Title:    node.Title,
			Display:  display,
			Expanded: node.Expanded,
			Count:    node.Count,
		},
		Selected: rowIndex == cursor,
		Level:    m.treeService.GetTreeLevel(node),
		Width:    width,
	}))
}

// renderNotificationRow renders a single notification row.
func (m *Model) renderNotificationRow(content *strings.Builder, notif notification.Notification, rowIndex, cursor, width int, now time.Time) {
	notif.Pane = m.getPaneName(notif.Pane)
	content.WriteString(render.Row(render.RowState{
		Notification: notif,
		SessionName:  m.getSessionName(notif.Session),
		Width:        width,
		Selected:     rowIndex == cursor,
		Now:          now,
	}))
}

// renderFlatView renders the flat notification list view.
func (m *Model) renderFlatView(content *strings.Builder, width, cursor int) {
	filtered := m.filtered
	if len(filtered) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No notifications found"))
		return
	}

	now := time.Now()
	for i, notif := range filtered {
		notifCopy := notif
		notifCopy.Pane = m.getPaneName(notifCopy.Pane)
		if i > 0 {
			content.WriteString("\n")
		}
		content.WriteString(render.Row(render.RowState{
			Notification: notifCopy,
			SessionName:  m.getSessionName(notifCopy.Session),
			Width:        width,
			Selected:     i == cursor,
			Now:          now,
		}))
	}
}

// ensureCursorVisible ensures the cursor is visible in the viewport.
func (m *Model) ensureCursorVisible() {
	listLen := m.currentListLen()
	if listLen == 0 {
		return
	}

	m.uiState.EnsureCursorVisible(listLen)
}
