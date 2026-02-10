package state

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
	"github.com/cristianoliveira/tmux-intray/internal/tui/render"
	"github.com/cristianoliveira/tmux-intray/internal/tui/service"
)

const (
	viewModeCompact       = settings.ViewModeCompact
	viewModeDetailed      = settings.ViewModeDetailed
	viewModeGrouped       = settings.ViewModeGrouped
	headerFooterLines     = 2
	defaultViewportWidth  = 80
	defaultViewportHeight = 22
)

// Model represents the TUI model for bubbletea.
type Model struct {
	// Core state
	notifications []notification.Notification
	filtered      []notification.Notification
	uiState       *UIState // Extracted UI state management

	// Settings fields (non-UI state)
	sortBy         string
	sortOrder      string
	columns        []string
	filters        settings.Filter
	loadedSettings *settings.Settings // Track loaded settings for comparison

	// Services - implementing BubbleTea nested model pattern
	treeService         model.TreeService
	notificationService model.NotificationService
	runtimeCoordinator  model.RuntimeCoordinator
	commandService      model.CommandService
	// Legacy fields for backward compatibility
	client            tmux.TmuxClient
	sessionNames      map[string]string
	windowNames       map[string]string
	paneNames         map[string]string
	ensureTmuxRunning func() bool
	jumpToPane        func(sessionID, windowID, paneID string) bool
	searchProvider    search.Provider
}

// Init initializes the TUI model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.uiState.IsSearchMode() || m.uiState.IsCommandMode() {
			m.uiState.ClearPendingKey()
		} else if m.uiState.GetPendingKey() != "" {
			if msg.String() == "a" && m.uiState.GetPendingKey() == "z" && m.isGroupedView() {
				m.uiState.ClearPendingKey()
				m.toggleFold()
				return m, nil
			}
			if msg.String() != "z" {
				m.uiState.ClearPendingKey()
			}
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			// Save settings before exiting
			if err := m.saveSettings(); err != nil {
				colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
			}
			// Exit
			return m, tea.Quit
		case tea.KeyEsc:
			if m.uiState.IsSearchMode() {
				m.uiState.SetSearchMode(false)
				m.applySearchFilter()
				m.uiState.ResetCursor()
			} else if m.uiState.IsCommandMode() {
				m.uiState.SetCommandMode(false)
			} else {
				return m, tea.Quit
			}

		case tea.KeyEnter:
			if m.uiState.IsSearchMode() {
				m.uiState.SetSearchMode(false)
				return m, nil
			}
			if m.uiState.IsCommandMode() {
				// Execute command using CommandService
				cmd := m.executeCommandViaService()
				m.uiState.SetCommandMode(false)
				return m, cmd
			}
			if m.isGroupedView() && m.toggleNodeExpansion() {
				return m, nil
			}
			// Jump to pane of selected notification
			return m, m.handleJump()

		case tea.KeyRunes:
			if m.uiState.IsSearchMode() {
				// In search mode, append runes to search query
				for _, r := range msg.Runes {
					m.uiState.AppendToSearchQuery(r)
				}
				m.applySearchFilter()
				m.uiState.ResetCursor()
			} else if m.uiState.IsCommandMode() {
				// In command mode, append runes to command query
				for _, r := range msg.Runes {
					m.uiState.AppendToCommandQuery(r)
				}
			}

		case tea.KeyBackspace:
			if m.uiState.IsSearchMode() {
				if len(m.uiState.GetSearchQuery()) > 0 {
					m.uiState.BackspaceSearchQuery()
					m.applySearchFilter()
					m.uiState.ResetCursor()
				}
			} else if m.uiState.IsCommandMode() {
				if len(m.uiState.GetCommandQuery()) > 0 {
					m.uiState.BackspaceCommandQuery()
				}
			}

		case tea.KeyUp, tea.KeyDown:
			// Navigation handled below
			break
		}

		// If we're in command mode, don't process other key bindings
		if m.uiState.IsCommandMode() {
			return m, nil
		}

		// Handle specific key bindings
		switch msg.String() {
		case "j":
			// Move cursor down
			listLen := m.currentListLen()
			m.uiState.MoveCursorDown(listLen)
			m.updateViewportContent()
			// Auto-scroll viewport if needed
			m.uiState.EnsureCursorVisible(listLen)
		case "k":
			// Move cursor up
			listLen := m.currentListLen()
			m.uiState.MoveCursorUp(listLen)
			m.updateViewportContent()
			// Auto-scroll viewport if needed
			m.uiState.EnsureCursorVisible(listLen)
		case "/":
			// Enter search mode
			m.uiState.SetSearchMode(true)
			m.applySearchFilter()
			m.uiState.ResetCursor()
		case ":":
			// Enter command mode
			if !m.uiState.IsSearchMode() && !m.uiState.IsCommandMode() {
				m.uiState.SetCommandMode(true)
			}
		case "d":
			// Dismiss selected notification
			return m, m.handleDismiss()
		case "r":
			// Mark selected notification as read
			return m, m.markSelectedRead()
		case "u":
			// Mark selected notification as unread
			return m, m.markSelectedUnread()
		case "v":
			if !m.uiState.IsSearchMode() && !m.uiState.IsCommandMode() {
				m.cycleViewMode()
			}
		case "h":
			// Collapse selected group node
			node := m.selectedVisibleNode()
			if node != nil {
				m.treeService.CollapseNode(node)
				m.invalidateCache()
				m.updateViewportContent()
			}
		case "l":
			// Expand selected group node
			node := m.selectedVisibleNode()
			if node != nil {
				m.treeService.ExpandNode(node)
				m.invalidateCache()
				m.updateViewportContent()
			}
		case "z":
			if !m.uiState.IsSearchMode() && m.isGroupedView() {
				m.uiState.SetPendingKey("z")
			}
		case "i":
			// In search mode, 'i' is handled by KeyRunes
			// This is a no-op but kept for documentation
		case "q":
			if err := m.saveSettings(); err != nil {
				colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
			}
			// Quit
			return m, tea.Quit
		}

	case saveSettingsSuccessMsg:
		// Settings saved successfully - already displayed info message in saveSettings
		return m, nil

	case saveSettingsFailedMsg:
		// Settings save failed - already displayed warning message in saveSettings
		return m, nil

	case tea.WindowSizeMsg:
		m.uiState.SetWidth(msg.Width)
		m.uiState.SetHeight(msg.Height)
		// Initialize or update viewport dimensions
		m.uiState.UpdateViewportSize()
		// Update viewport content
		m.updateViewportContent()
	}

	return m, nil
}

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
	}))

	return s.String()
}

// SetLoadedSettings stores the loaded settings reference for later comparison.
func (m *Model) SetLoadedSettings(loaded *settings.Settings) {
	m.loadedSettings = loaded
}

// ToState converts the Model to a TUIState DTO for settings persistence.
// Only persists user-configurable settings (columns, sort, filters, view mode).
func (m *Model) ToState() settings.TUIState {
	dto := m.uiState.ToDTO()
	return settings.TUIState{
		Columns:               m.columns,
		SortBy:                m.sortBy,
		SortOrder:             m.sortOrder,
		Filters:               m.filters,
		ViewMode:              string(dto.ViewMode),
		GroupBy:               string(dto.GroupBy),
		DefaultExpandLevel:    dto.ExpandLevel,
		DefaultExpandLevelSet: true,
		ExpansionState:        dto.ExpansionState,
	}
}

// FromState applies settings from TUIState to the Model.
// Supports partial updates - only updates non-empty fields.
// Returns an error if the settings are invalid.
func (m *Model) FromState(state settings.TUIState) error {
	if state.GroupBy != "" && !settings.IsValidGroupBy(state.GroupBy) {
		return fmt.Errorf("invalid groupBy value: %s", state.GroupBy)
	}
	if state.DefaultExpandLevelSet {
		if state.DefaultExpandLevel < settings.MinExpandLevel || state.DefaultExpandLevel > settings.MaxExpandLevel {
			return fmt.Errorf("invalid defaultExpandLevel value: %d", state.DefaultExpandLevel)
		}
	}

	// Apply non-empty fields only (support partial updates)
	if len(state.Columns) > 0 {
		m.columns = state.Columns
	}
	if state.SortBy != "" {
		m.sortBy = state.SortBy
	}
	if state.SortOrder != "" {
		m.sortOrder = state.SortOrder
	}

	// Update UI state via DTO
	dto := model.UIDTO{}
	if state.ViewMode != "" {
		dto.ViewMode = model.ViewMode(state.ViewMode)
	}
	if state.GroupBy != "" {
		dto.GroupBy = model.GroupBy(state.GroupBy)
	}
	if state.DefaultExpandLevelSet {
		dto.ExpandLevel = state.DefaultExpandLevel
		dto.ExpandLevelSet = true
	}
	if state.ExpansionState != nil {
		dto.ExpansionState = state.ExpansionState
	}

	// Apply UI state - preserve current values if not explicitly set
	if err := m.uiState.FromDTO(dto); err != nil {
		return err
	}

	// Apply filters - only update non-empty fields
	if state.Filters.Level != "" ||
		state.Filters.State != "" ||
		state.Filters.Session != "" ||
		state.Filters.Window != "" ||
		state.Filters.Pane != "" {
		if state.Filters.Level != "" {
			m.filters.Level = state.Filters.Level
		}
		if state.Filters.State != "" {
			m.filters.State = state.Filters.State
		}
		if state.Filters.Session != "" {
			m.filters.Session = state.Filters.Session
		}
		if state.Filters.Window != "" {
			m.filters.Window = state.Filters.Window
		}
		if state.Filters.Pane != "" {
			m.filters.Pane = state.Filters.Pane
		}
	}

	m.applySearchFilter()
	m.resetCursor()
	return nil
}

// NewModel creates a new TUI model.
// If client is nil, a new DefaultClient is created.
func NewModel(client tmux.TmuxClient) (*Model, error) {
	if client == nil {
		client = tmux.NewDefaultClient()
	}

	// Initialize UI state
	uiState := NewUIState()

	// Initialize runtime coordinator (handles tmux integration and name resolution)
	runtimeCoordinator := service.NewRuntimeCoordinator(client)

	// Initialize tree service
	treeService := service.NewTreeService(uiState.GetGroupBy())

	// Initialize notification service with default search provider
	searchProvider := search.NewTokenProvider(
		search.WithCaseInsensitive(true),
		search.WithSessionNames(runtimeCoordinator.GetSessionNames()),
		search.WithWindowNames(runtimeCoordinator.GetWindowNames()),
		search.WithPaneNames(runtimeCoordinator.GetPaneNames()),
	)
	notificationService := service.NewNotificationService(searchProvider, runtimeCoordinator)

	m := Model{
		uiState:             uiState,
		runtimeCoordinator:  runtimeCoordinator,
		treeService:         treeService,
		notificationService: notificationService,
		// Legacy fields kept for backward compatibility but now using services
		client:            client,
		sessionNames:      runtimeCoordinator.GetSessionNames(),
		windowNames:       runtimeCoordinator.GetWindowNames(),
		paneNames:         runtimeCoordinator.GetPaneNames(),
		ensureTmuxRunning: core.EnsureTmuxRunning,
		jumpToPane:        core.JumpToPane,
	}

	// Initialize command service after model creation (needs ModelInterface)
	m.commandService = service.NewCommandService(&m)

	// Load initial notifications
	err := m.loadNotifications(false)
	if err != nil {
		return &Model{}, err
	}

	return &m, nil
}

func (m *Model) ensureTreeService() model.TreeService {
	if m.treeService != nil {
		return m.treeService
	}

	groupBy := model.GroupByNone
	if m.uiState != nil {
		groupBy = m.uiState.GetGroupBy()
	}

	m.treeService = service.NewTreeService(groupBy)
	return m.treeService
}

// applySearchFilter filters notifications based on the search query.
// This function only updates the filtered notifications; cursor management
// should be handled separately by resetCursor() or restoreCursor().
func (m *Model) applySearchFilter() {
	treeService := m.ensureTreeService()
	treeService.InvalidateCache()

	query := strings.TrimSpace(m.uiState.GetSearchQuery())
	if query == "" {
		m.filtered = m.notifications
	} else {
		// Use NotificationService to filter notifications
		m.filtered = m.notificationService.FilterNotifications(m.notifications, query)
	}
	if m.isGroupedView() {
		m.buildFilteredTree(m.filtered)
	} else {
		treeService.ClearTree()
	}
	m.updateViewportContent()
}

// ApplySearchFilter is the public version of applySearchFilter.
func (m *Model) ApplySearchFilter() {
	m.applySearchFilter()
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

// getNodeIdentifier returns a stable identifier for a node.
// For notification nodes, this is the notification ID.
// For group nodes, this is a combination of the node kind and title.
func (m *Model) getNodeIdentifier(node *model.TreeNode) string {
	return m.treeService.GetNodeIdentifier(node)
}

// findNodeByIdentifier finds a node by its identifier in the visible nodes list.
func (m *Model) findNodeByIdentifier(identifier string) *model.TreeNode {
	for _, node := range m.treeService.GetVisibleNodes() {
		if m.treeService.GetNodeIdentifier(node) == identifier {
			return node
		}
	}
	return nil
}

// restoreCursor restores the cursor to the node with the given identifier.
// If the node is not found, it adjusts the cursor to be within bounds.
func (m *Model) restoreCursor(identifier string) {
	if identifier == "" {
		m.adjustCursorBounds()
		return
	}

	targetNode := m.findNodeByIdentifier(identifier)
	if targetNode != nil {
		visibleNodes := m.ensureTreeService().GetVisibleNodes()
		for i, node := range visibleNodes {
			if node == targetNode {
				m.uiState.SetCursor(i)
				m.uiState.EnsureCursorVisible(len(visibleNodes))
				return
			}
		}
	}

	// If we couldn't find the exact node, adjust to bounds
	m.adjustCursorBounds()
}

// adjustCursorBounds ensures the cursor is within valid bounds.
func (m *Model) adjustCursorBounds() {
	listLen := m.currentListLen()
	m.uiState.AdjustCursorBounds(listLen)
	m.uiState.EnsureCursorVisible(listLen)
}

// executeCommandViaService executes the current command query using the CommandService and returns a command to run.
func (m *Model) executeCommandViaService() tea.Cmd {
	// If commandService is not initialized, fall back to legacy implementation
	if m.commandService == nil {
		return m.executeCommand()
	}

	cmd := strings.TrimSpace(m.uiState.GetCommandQuery())
	if cmd == "" {
		colors.Warning("Command is empty")
		return nil
	}

	// Parse and execute command using CommandService
	name, args, err := m.commandService.ParseCommand(cmd)
	if err != nil {
		colors.Warning(fmt.Sprintf("Failed to parse command: %v", err))
		return nil
	}

	result, err := m.commandService.ExecuteCommand(name, args)
	if err != nil {
		colors.Error(fmt.Sprintf("Failed to execute command: %v", err))
		return nil
	}

	// Handle result
	if result.Message != "" {
		if result.Error {
			colors.Warning(result.Message)
		} else {
			colors.Info(result.Message)
		}
	}

	if result.Quit {
		return tea.Quit
	}

	return result.Cmd
}

// executeCommand executes the current command query and returns a command to run.
// This is the legacy implementation kept for reference.
func (m *Model) executeCommand() tea.Cmd {
	cmd := strings.TrimSpace(m.uiState.GetCommandQuery())
	if cmd == "" {
		colors.Warning("Command is empty")
		return nil
	}

	parts := strings.Fields(cmd)
	command := strings.ToLower(parts[0])
	args := parts[1:]

	switch command {
	case "q":
		if len(args) > 0 {
			colors.Warning("Invalid usage: q")
			return nil
		}
		if err := m.saveSettings(); err != nil {
			colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		}
		return tea.Quit
	case "w":
		if len(args) > 0 {
			colors.Warning("Invalid usage: w")
			return nil
		}
		return func() tea.Msg {
			if err := m.saveSettings(); err != nil {
				return saveSettingsFailedMsg{err: err}
			}
			return saveSettingsSuccessMsg{}
		}
	case "group-by":
		if len(args) != 1 {
			colors.Warning("Invalid usage: group-by <none|session|window|pane>")
			return nil
		}

		groupBy := strings.ToLower(args[0])
		if !settings.IsValidGroupBy(groupBy) {
			colors.Warning(fmt.Sprintf("Invalid group-by value: %s (expected one of: none, session, window, pane)", args[0]))
			return nil
		}

		if string(m.uiState.GetGroupBy()) == groupBy {
			return nil
		}

		m.uiState.SetGroupBy(model.GroupBy(groupBy))
		m.applySearchFilter()
		m.resetCursor()
		if err := m.saveSettings(); err != nil {
			colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
			return nil
		}
		colors.Info(fmt.Sprintf("Group by: %s", groupBy))
		return nil
	case "expand-level":
		if len(args) != 1 {
			colors.Warning("Invalid usage: expand-level <0|1|2|3>")
			return nil
		}

		level, err := strconv.Atoi(args[0])
		if err != nil || level < settings.MinExpandLevel || level > settings.MaxExpandLevel {
			colors.Warning(fmt.Sprintf("Invalid expand-level value: %s (expected %d-%d)", args[0], settings.MinExpandLevel, settings.MaxExpandLevel))
			return nil
		}

		if m.uiState.GetExpandLevel() == level {
			return nil
		}

		m.uiState.SetExpandLevel(level)
		if m.isGroupedView() {
			m.applyDefaultExpansion()
		}
		if err := m.saveSettings(); err != nil {
			colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
			return nil
		}
		colors.Info(fmt.Sprintf("Default expand level: %d", m.uiState.GetExpandLevel()))
		return nil

	case "toggle-view":
		if len(args) > 0 {
			colors.Warning("Invalid usage: toggle-view")
			return nil
		}

		if m.uiState.IsGroupedView() {
			m.uiState.SetViewMode(model.ViewModeDetailed)
		} else {
			m.uiState.SetViewMode(model.ViewModeGrouped)
		}
		m.applySearchFilter()
		m.resetCursor()
		if err := m.saveSettings(); err != nil {
			colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		}
		colors.Info(fmt.Sprintf("View mode: %s", m.uiState.GetViewMode()))
		return nil
	default:
		colors.Warning(fmt.Sprintf("Unknown command: %s", command))
		return nil
	}
}

// saveSettings extracts current settings from model and saves to disk.
func (m *Model) saveSettings() error {
	// Extract current settings state
	state := m.ToState()
	colors.Debug("Saving settings from TUI state")
	if err := settings.Save(state.ToSettings()); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	colors.Info("Settings saved")
	return nil
}

// SaveSettings is the public version of saveSettings.
func (m *Model) SaveSettings() error {
	return m.saveSettings()
}

// updateViewportContent updates the viewport with the current filtered notifications.
func (m *Model) updateViewportContent() {
	var content strings.Builder
	width := m.uiState.GetWidth()
	cursor := m.uiState.GetCursor()

	if m.isGroupedView() {
		visibleNodes := m.treeService.GetVisibleNodes()
		if len(visibleNodes) == 0 {
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No notifications found"))
		} else {
			now := time.Now()
			for rowIndex, node := range visibleNodes {
				if node == nil {
					continue
				}
				if rowIndex > 0 {
					content.WriteString("\n")
				}
				if m.isGroupNode(node) {
					content.WriteString(render.RenderGroupRow(render.GroupRow{
						Node: &render.GroupNode{
							Title:    node.Title,
							Display:  node.Display,
							Expanded: node.Expanded,
							Count:    node.Count,
						},
						Selected: rowIndex == cursor,
						Level:    m.treeService.GetTreeLevel(node),
						Width:    width,
					}))
					continue
				}
				if node.Notification == nil {
					continue
				}
				notif := *node.Notification
				content.WriteString(render.Row(render.RowState{
					Notification: notif,
					SessionName:  m.getSessionName(notif.Session),
					Width:        width,
					Selected:     rowIndex == cursor,
					Now:          now,
				}))
			}
		}

		(*m.uiState.GetViewport()).SetContent(content.String())
		return
	}

	if len(m.filtered) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No notifications found"))
	} else {
		now := time.Now()
		for i, notif := range m.filtered {
			if i > 0 {
				content.WriteString("\n")
			}
			content.WriteString(render.Row(render.RowState{
				Notification: notif,
				SessionName:  m.getSessionName(notif.Session),
				Width:        width,
				Selected:     i == cursor,
				Now:          now,
			}))
		}
	}

	(*m.uiState.GetViewport()).SetContent(content.String())
}

// ensureCursorVisible ensures the cursor is visible in the viewport.
func (m *Model) ensureCursorVisible() {
	listLen := m.currentListLen()
	if listLen == 0 {
		return
	}

	m.uiState.EnsureCursorVisible(listLen)
}

// handleDismiss handles the dismiss action for the selected notification.
func (m *Model) handleDismiss() tea.Cmd {
	if m.currentListLen() == 0 {
		return nil
	}

	// Get the selected notification
	selected, ok := m.selectedNotification()
	if !ok {
		return nil
	}

	// Dismiss the notification using storage
	id := strconv.Itoa(selected.ID)
	if err := storage.DismissNotification(id); err != nil {
		colors.Error(fmt.Sprintf("Failed to dismiss notification: %v", err))
		return nil
	}

	// Save the current cursor position before reload
	oldCursor := m.uiState.GetCursor()

	// Reload notifications to get updated state (preserve cursor)
	if err := m.loadNotifications(true); err != nil {
		colors.Error(fmt.Sprintf("Failed to reload notifications: %v", err))
		return nil
	}

	// Restore cursor to the saved position, adjusting for bounds
	listLen := m.currentListLen()
	if listLen == 0 {
		m.uiState.SetCursor(0)
	} else {
		m.uiState.SetCursor(oldCursor)
		// Ensure cursor is within bounds
		m.adjustCursorBounds()
	}

	// Update viewport content
	m.updateViewportContent()

	return nil
}

// markSelectedRead marks the selected notification as read.
func (m *Model) markSelectedRead() tea.Cmd {
	if m.currentListLen() == 0 {
		return nil
	}

	selected, ok := m.selectedNotification()
	if !ok {
		return nil
	}

	// Save the notification ID to restore cursor later
	selectedID := selected.ID

	id := strconv.Itoa(selected.ID)
	if err := storage.MarkNotificationRead(id); err != nil {
		colors.Error(fmt.Sprintf("Failed to mark notification read: %v", err))
		return nil
	}

	if err := m.loadNotifications(true); err != nil {
		colors.Error(fmt.Sprintf("Failed to reload notifications: %v", err))
		return nil
	}

	// Restore cursor to the selected notification
	identifier := fmt.Sprintf("notif:%d", selectedID)
	m.restoreCursor(identifier)

	m.updateViewportContent()
	return nil
}

// markSelectedUnread marks the selected notification as unread.
func (m *Model) markSelectedUnread() tea.Cmd {
	if m.currentListLen() == 0 {
		return nil
	}

	selected, ok := m.selectedNotification()
	if !ok {
		return nil
	}

	// Save the notification ID to restore cursor later
	selectedID := selected.ID

	id := strconv.Itoa(selected.ID)
	if err := storage.MarkNotificationUnread(id); err != nil {
		colors.Error(fmt.Sprintf("tui: failed to mark notification unread: %v", err))
		return nil
	}

	if err := m.loadNotifications(true); err != nil {
		colors.Error(fmt.Sprintf("tui: failed to reload notifications: %v", err))
		return nil
	}

	// Restore cursor to the selected notification
	identifier := fmt.Sprintf("notif:%d", selectedID)
	m.restoreCursor(identifier)

	m.updateViewportContent()
	return nil
}

// handleJump handles the jump action for the selected notification.
func (m *Model) handleJump() tea.Cmd {
	if m.currentListLen() == 0 {
		return nil
	}

	// Get the selected notification
	selected, ok := m.selectedNotification()
	if !ok {
		return nil
	}

	// Check if notification has valid session, window, pane
	if selected.Session == "" || selected.Window == "" || selected.Pane == "" {
		colors.Error("jump: notification missing session, window, or pane information")
		return nil
	}

	// Ensure tmux is running
	if !m.runtimeCoordinator.EnsureTmuxRunning() {
		colors.Error("tmux not running")
		return nil
	}

	// Jump to the pane using RuntimeCoordinator
	if !m.runtimeCoordinator.JumpToPane(selected.Session, selected.Window, selected.Pane) {
		colors.Error("jump: failed to jump to pane")
		return nil
	}

	// Exit TUI after successful jump
	return tea.Quit
}

// resetCursor resets the cursor to the first item.
func (m *Model) resetCursor() {
	m.uiState.ResetCursor()
}

// ResetCursor is the public version of resetCursor.
func (m *Model) ResetCursor() {
	m.resetCursor()
}

// loadNotifications loads notifications from storage.
// If preserveCursor is true, attempts to maintain the current cursor position.
func (m *Model) loadNotifications(preserveCursor bool) error {
	var savedCursorPos int
	var savedNodeID string

	if preserveCursor {
		// Save current cursor state
		savedCursorPos = m.uiState.GetCursor()
		cursor := m.uiState.GetCursor()
		visibleNodes := m.treeService.GetVisibleNodes()
		if m.isGroupedView() && cursor < len(visibleNodes) {
			savedNodeID = m.getNodeIdentifier(visibleNodes[cursor])
		} else if !m.isGroupedView() && cursor < len(m.filtered) {
			savedNodeID = fmt.Sprintf("notif:%d", m.filtered[cursor].ID)
		}
	}

	lines, err := storage.ListNotifications("active", "", "", "", "", "", "", "")
	if err != nil {
		return fmt.Errorf("failed to load notifications: %w", err)
	}
	if lines == "" {
		m.notifications = []notification.Notification{}
		m.filtered = []notification.Notification{}
		m.treeService.ClearTree()
		if preserveCursor {
			m.adjustCursorBounds()
		} else {
			m.resetCursor()
		}
		m.updateViewportContent()
		return nil
	}

	var notifications []notification.Notification
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, err := notification.ParseNotification(line)
		if err != nil {
			continue
		}
		notifications = append(notifications, notif)
	}

	// Sort notifications by timestamp descending (most recent first)
	sort.Slice(notifications, func(i, j int) bool {
		return notifications[i].Timestamp > notifications[j].Timestamp
	})

	m.notifications = notifications
	m.applySearchFilter()

	if preserveCursor {
		if savedNodeID != "" {
			// Try to restore cursor to the same notification
			m.restoreCursor(savedNodeID)
		} else {
			// If we couldn't save the node ID, just adjust to bounds
			m.uiState.SetCursor(savedCursorPos)
			m.adjustCursorBounds()
		}
	} else {
		m.resetCursor()
	}

	return nil
}

func (m *Model) isGroupedView() bool {
	return m.uiState.IsGroupedView()
}

// IsGroupedView is the public version of isGroupedView.
func (m *Model) IsGroupedView() bool {
	return m.isGroupedView()
}

// cycleViewMode cycles through available view modes (compact → detailed → grouped).
func (m *Model) cycleViewMode() {
	m.uiState.CycleViewMode()
	m.applySearchFilter()
	m.resetCursor()

	if err := m.saveSettings(); err != nil {
		colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}
	colors.Info(fmt.Sprintf("View mode: %s", m.uiState.GetViewMode()))
}

func (m *Model) computeVisibleNodes() []*model.TreeNode {
	return m.treeService.GetVisibleNodes()
}

func (m *Model) invalidateCache() {
	m.ensureTreeService().InvalidateCache()
}

func isGroupNode(node *model.TreeNode) bool {
	if node == nil {
		return false
	}
	return node.Kind != model.NodeKindNotification && node.Kind != model.NodeKindRoot
}

// isGroupNode checks if a model.TreeNode is a group node.
func (m *Model) isGroupNode(node *model.TreeNode) bool {
	return node.Kind != model.NodeKindNotification && node.Kind != model.NodeKindRoot
}

func getTreeLevel(node *model.TreeNode) int {
	if node == nil {
		return 0
	}
	switch node.Kind {
	case model.NodeKindSession:
		return 0
	case model.NodeKindWindow:
		return 1
	case model.NodeKindPane:
		return 2
	default:
		return 0
	}
}
func (m *Model) currentListLen() int {
	if m.isGroupedView() {
		return len(m.treeService.GetVisibleNodes())
	}
	return len(m.filtered)
}

func (m *Model) selectedNotification() (notification.Notification, bool) {
	cursor := m.uiState.GetCursor()
	if m.isGroupedView() {
		visibleNodes := m.treeService.GetVisibleNodes()
		if cursor < 0 || cursor >= len(visibleNodes) {
			return notification.Notification{}, false
		}
		node := visibleNodes[cursor]
		if node == nil || node.Notification == nil {
			return notification.Notification{}, false
		}
		return *node.Notification, true
	}

	if cursor < 0 || cursor >= len(m.filtered) {
		return notification.Notification{}, false
	}
	return m.filtered[cursor], true
}

func (m *Model) selectedVisibleNode() *model.TreeNode {
	if !m.isGroupedView() {
		return nil
	}
	cursor := m.uiState.GetCursor()
	visibleNodes := m.treeService.GetVisibleNodes()
	if cursor < 0 || cursor >= len(visibleNodes) {
		return nil
	}
	return visibleNodes[cursor]
}

func (m *Model) toggleNodeExpansion() bool {
	node := m.selectedVisibleNode()
	if node == nil || node.Kind == model.NodeKindNotification {
		return false
	}
	if node.Expanded {
		m.treeService.CollapseNode(node)
	} else {
		m.treeService.ExpandNode(node)
	}
	m.invalidateCache()
	return true
}

func (m *Model) toggleFold() {
	if !m.isGroupedView() {
		return
	}
	node := m.selectedVisibleNode()
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	if m.allGroupsCollapsed() {
		m.applyDefaultExpansion()
		return
	}
	if node.Expanded {
		m.treeService.CollapseNode(node)
		m.invalidateCache()
		m.updateViewportContent()
		return
	}
	m.treeService.ExpandNode(node)
	m.invalidateCache()
	m.updateViewportContent()
}

func (m *Model) allGroupsCollapsed() bool {
	treeRoot := m.treeService.GetTreeRoot()
	if treeRoot == nil {
		return false
	}
	collapsed := true
	seen := false
	var walk func(node *model.TreeNode)
	walk = func(node *model.TreeNode) {
		if node == nil || !collapsed {
			return
		}
		if m.isGroupNode(node) {
			seen = true
			if node.Expanded {
				collapsed = false
				return
			}
		}
		for _, child := range node.Children {
			walk(child)
			if !collapsed {
				return
			}
		}
	}
	walk(treeRoot)
	return seen && collapsed
}

func (m *Model) applyDefaultExpansion() {
	treeRoot := m.treeService.GetTreeRoot()
	if treeRoot == nil {
		return
	}

	// Save selected node identifier before modifying tree
	selectedID := ""
	if selected := m.selectedVisibleNode(); selected != nil {
		selectedID = m.treeService.GetNodeIdentifier(selected)
	}

	level := m.uiState.GetExpandLevel()
	if level < settings.MinExpandLevel {
		level = settings.MinExpandLevel
	}
	if level > settings.MaxExpandLevel {
		level = settings.MaxExpandLevel
	}

	var walk func(node *model.TreeNode)
	walk = func(node *model.TreeNode) {
		if node == nil {
			return
		}
		if m.isGroupNode(node) {
			nodeLevel := m.treeService.GetTreeLevel(node) + 1
			expanded := nodeLevel <= level
			node.Expanded = expanded
			m.updateExpansionState(node, expanded)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(treeRoot)

	m.invalidateCache()

	// Restore cursor to the selected node using identifier
	if selectedID != "" {
		m.restoreCursor(selectedID)
	}

	// Ensure cursor is within bounds
	visibleNodes := m.treeService.GetVisibleNodes()
	if m.uiState.GetCursor() >= len(visibleNodes) {
		m.uiState.SetCursor(len(visibleNodes) - 1)
	}
	if m.uiState.GetCursor() < 0 {
		m.uiState.SetCursor(0)
	}
	m.updateViewportContent()
	m.ensureCursorVisible()
}

// ApplyDefaultExpansion is the public version of applyDefaultExpansion.
func (m *Model) ApplyDefaultExpansion() {
	m.applyDefaultExpansion()
}

// GetViewMode returns the current view mode.
func (m *Model) GetViewMode() string {
	return string(m.uiState.GetViewMode())
}

// ToggleViewMode toggles between view modes.
func (m *Model) ToggleViewMode() error {
	m.cycleViewMode()
	return nil
}

func (m *Model) expandNode(node *model.TreeNode) {
	if !m.isGroupedView() {
		return
	}
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	if node.Expanded {
		return
	}

	// Save node identifier before modifying tree to avoid using stale references
	nodeID := m.treeService.GetNodeIdentifier(node)

	m.treeService.ExpandNode(node)
	m.updateExpansionState(node, true)

	// Restore cursor to the same node using identifier
	m.restoreCursor(nodeID)

	m.updateViewportContent()
	m.ensureCursorVisible()
}

func (m *Model) collapseNode(node *model.TreeNode) {
	if !m.isGroupedView() {
		return
	}
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	if !node.Expanded {
		return
	}

	// Save node identifiers before modifying tree to avoid using stale references
	selectedID := ""
	if selected := m.selectedVisibleNode(); selected != nil {
		selectedID = m.treeService.GetNodeIdentifier(selected)
	}
	nodeID := m.treeService.GetNodeIdentifier(node)

	m.treeService.CollapseNode(node)
	m.updateExpansionState(node, false)
	visibleNodes := m.treeService.GetVisibleNodes()

	// If selected node was inside the collapsed node, move cursor to the collapsed node
	if selectedID != "" {
		// Check if the selected node is contained within the collapsed node
		// by comparing paths
		treeRoot := m.treeService.GetTreeRoot()
		if selectedNode := m.treeService.FindNodeByID(treeRoot, selectedID); selectedNode != nil {
			if collapsedNode := m.treeService.FindNodeByID(treeRoot, nodeID); collapsedNode != nil {
				if m.nodeContains(collapsedNode, selectedNode) {
					// Move cursor to the collapsed node
					if index := indexOfTreeNode(visibleNodes, collapsedNode); index >= 0 {
						m.uiState.SetCursor(index)
					}
				}
			}
		}
	}

	// Ensure cursor is within bounds
	if m.uiState.GetCursor() >= len(visibleNodes) {
		m.uiState.SetCursor(len(visibleNodes) - 1)
	}
	if m.uiState.GetCursor() < 0 {
		m.uiState.SetCursor(0)
	}
	m.updateViewportContent()
	m.ensureCursorVisible()
}

// nodeContains checks if targetNode is contained within root node.
func (m *Model) nodeContains(root, target *model.TreeNode) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	for _, child := range root.Children {
		if m.nodeContains(child, target) {
			return true
		}
	}
	return false
}

// indexOfTreeNode finds the index of a target node in a slice.
func indexOfTreeNode(nodes []*model.TreeNode, target *model.TreeNode) int {
	for i, node := range nodes {
		if node == target {
			return i
		}
	}
	return -1
}

func (m *Model) updateExpansionState(node *model.TreeNode, expanded bool) {
	key := m.nodeExpansionKey(node)
	if key == "" {
		return
	}
	expansionState := m.uiState.GetExpansionState()
	if expansionState == nil {
		expansionState = map[string]bool{}
		m.uiState.SetExpansionState(expansionState)
	}
	legacyKey := m.nodeExpansionLegacyKey(node)
	if legacyKey != "" && legacyKey != key {
		delete(expansionState, legacyKey)
	}
	m.uiState.UpdateExpansionState(key, expanded)
}

func (m *Model) nodeExpansionKey(node *model.TreeNode) string {
	if node == nil || node.Kind == model.NodeKindNotification || node.Kind == model.NodeKindRoot {
		return ""
	}
	// For group nodes, construct the key from the node's own properties
	// This is simpler than traversing the tree for each node
	switch node.Kind {
	case model.NodeKindSession:
		return serializeNodeExpansionPath(model.NodeKindSession, node.Title)
	case model.NodeKindWindow:
		// For window nodes, we need the session name too
		// This is a simplified approach - the full implementation would track parent references
		return serializeNodeExpansionPath(model.NodeKindWindow, node.Title)
	case model.NodeKindPane:
		// Similar to window nodes
		return serializeNodeExpansionPath(model.NodeKindPane, node.Title)
	default:
		return ""
	}
}

func (m *Model) nodeExpansionLegacyKey(node *model.TreeNode) string {
	if node == nil || node.Kind == model.NodeKindNotification || node.Kind == model.NodeKindRoot {
		return ""
	}
	// Use the same logic as the new key for now
	return m.nodeExpansionKey(node)
}

func serializeNodeExpansionPath(kind model.NodeKind, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	encoded := make([]string, 0, len(parts))
	for _, part := range parts {
		encoded = append(encoded, escapeExpansionPathSegment(part))
	}
	return fmt.Sprintf("%s:%s", kind, strings.Join(encoded, ":"))
}

func escapeExpansionPathSegment(value string) string {
	replacer := strings.NewReplacer(
		"%", "%25",
		":", "%3A",
	)
	return replacer.Replace(value)
}

func serializeLegacyNodeExpansionPath(kind model.NodeKind, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%s", kind, strings.Join(parts, ":"))
}

func (m *Model) nodePathSegments(path []*model.TreeNode) (session string, window string, pane string) {
	for _, current := range path {
		switch current.Kind {
		case model.NodeKindSession:
			session = current.Title
		case model.NodeKindWindow:
			window = current.Title
		case model.NodeKindPane:
			pane = current.Title
		}
	}
	return session, window, pane
}

// buildFilteredTree builds a tree from filtered notifications and applies saved expansion state.
// Returns a tree where group counts reflect only matching notifications.
func (m *Model) buildFilteredTree(notifications []notification.Notification) *model.TreeNode {
	m.invalidateCache()

	if len(notifications) == 0 {
		m.treeService.ClearTree()
		return nil
	}

	// Use TreeService to build the tree
	err := m.treeService.BuildTree(notifications, string(m.uiState.GetGroupBy()))
	if err != nil {
		m.treeService.ClearTree()
		return nil
	}

	// Prune empty groups (groups with no matching notifications)
	m.treeService.PruneEmptyGroups()

	// Apply saved expansion state where possible
	expansionState := m.uiState.GetExpansionState()
	if expansionState != nil {
		m.treeService.ApplyExpansionState(expansionState)
	} else {
		// If no saved state, expand all by default
		m.expandTreeRecursive(m.treeService.GetTreeRoot())
	}
	return m.treeService.GetTreeRoot()
}

// expandTreeRecursive is a helper that expands all group nodes.
func (m *Model) expandTreeRecursive(node *model.TreeNode) {
	if node == nil {
		return
	}
	if node.Kind != model.NodeKindNotification {
		node.Expanded = true
	}
	for _, child := range node.Children {
		m.expandTreeRecursive(child)
	}
}

// pruneEmptyGroups removes groups from the tree that have no children or count of 0.
// This ensures that empty groups created by filtering don't appear in the UI.
func (m *Model) pruneEmptyGroups(node *Node) {
	if node == nil {
		return
	}

	// Recursively prune children first
	var filteredChildren []*Node
	for _, child := range node.Children {
		m.pruneEmptyGroups(child)
		// Keep the child if it has children (even if it's a leaf with notifications)
		// or if it's a notification node
		if len(child.Children) > 0 || child.Kind == NodeKindNotification {
			filteredChildren = append(filteredChildren, child)
		}
	}
	node.Children = filteredChildren
}

// applyExpansionState applies the saved expansion state to the tree nodes.
// Only applies state to nodes that still exist in the tree (after pruning).
func (m *Model) applyExpansionState(node *model.TreeNode) {
	if node == nil {
		return
	}

	// Apply expansion state to group nodes
	if m.isGroupNode(node) {
		if expanded, ok := m.expansionStateValue(node); ok {
			node.Expanded = expanded
		} else {
			// Default to expanded for nodes without saved state
			node.Expanded = true
		}

	}

	// Recursively apply to children
	for _, child := range node.Children {
		m.applyExpansionState(child)
	}
}

func (m *Model) expansionStateValue(node *model.TreeNode) (bool, bool) {
	expansionState := m.uiState.GetExpansionState()
	if expansionState == nil {
		return false, false
	}

	key := m.nodeExpansionKey(node)
	if key != "" {
		expanded, ok := expansionState[key]
		if ok {
			return expanded, true
		}
	}

	legacyKey := m.nodeExpansionLegacyKey(node)
	if legacyKey == "" {
		return false, false
	}

	expanded, ok := expansionState[legacyKey]
	if !ok {
		return false, false
	}
	if key != "" {
		m.uiState.UpdateExpansionState(key, expanded)
		delete(expansionState, legacyKey)
	}
	return expanded, true
}

// getSessionName returns the session name for a session ID.
// Uses RuntimeCoordinator for name resolution.
func (m *Model) getSessionName(sessionID string) string {
	return m.runtimeCoordinator.ResolveSessionName(sessionID)
}

// getWindowName returns the window name for a window ID.
// Uses RuntimeCoordinator for name resolution.
func (m *Model) getWindowName(windowID string) string {
	return m.runtimeCoordinator.ResolveWindowName(windowID)
}

// getPaneName returns the pane name for a pane ID.
// Uses RuntimeCoordinator for name resolution.
func (m *Model) getPaneName(paneID string) string {
	return m.runtimeCoordinator.ResolvePaneName(paneID)
}

// getTreeRootForTest returns the tree root for testing purposes.
func (m *Model) getTreeRootForTest() *model.TreeNode {
	return m.treeService.GetTreeRoot()
}

// getVisibleNodesForTest returns the visible nodes for testing purposes.
func (m *Model) getVisibleNodesForTest() []*model.TreeNode {
	return m.treeService.GetVisibleNodes()
}
