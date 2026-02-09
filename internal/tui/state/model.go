package state

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/render"
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
	notifications []notification.Notification
	filtered      []notification.Notification
	cursor        int
	searchQuery   string
	searchMode    bool
	commandMode   bool
	commandQuery  string
	pendingKey    string
	viewport      viewport.Model
	width         int
	height        int
	sessionNames  map[string]string
	windowNames   map[string]string
	paneNames     map[string]string
	client        tmux.TmuxClient // TmuxClient for tmux operations

	// Settings fields
	sortBy             string
	sortOrder          string
	columns            []string
	filters            settings.Filter
	viewMode           string
	groupBy            string
	defaultExpandLevel int

	expansionState map[string]bool
	loadedSettings *settings.Settings // Track loaded settings for comparison

	treeRoot     *Node
	visibleNodes []*Node

	visibleNodesCache []*Node
	cacheValid        bool

	ensureTmuxRunning func() bool
	jumpToPane        func(sessionID, windowID, paneID string) bool
	searchProvider    search.Provider // Optional custom search provider (defaults to token-based search)
}

// Init initializes the TUI model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchMode || m.commandMode {
			m.pendingKey = ""
		} else if m.pendingKey != "" {
			if msg.String() == "a" && m.pendingKey == "z" && m.isGroupedView() {
				m.pendingKey = ""
				m.toggleFold()
				return m, nil
			}
			if msg.String() != "z" {
				m.pendingKey = ""
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
			if m.searchMode {
				m.searchMode = false
				m.searchQuery = ""
				m.applySearchFilter(false)
			} else if m.commandMode {
				m.commandMode = false
				m.commandQuery = ""
			} else {
				return m, tea.Quit
			}

		case tea.KeyEnter:
			if m.commandMode {
				// Execute command
				cmd := m.executeCommand()
				m.commandMode = false
				m.commandQuery = ""
				return m, cmd
			}
			if m.isGroupedView() && m.toggleNodeExpansion() {
				return m, nil
			}
			// Jump to pane of selected notification
			return m, m.handleJump()

		case tea.KeyRunes:
			if m.searchMode {
				// In search mode, append runes to search query
				for _, r := range msg.Runes {
					m.searchQuery += string(r)
				}
				m.applySearchFilter(false)
			} else if m.commandMode {
				// In command mode, append runes to command query
				for _, r := range msg.Runes {
					m.commandQuery += string(r)
				}
			}

		case tea.KeyBackspace:
			if m.searchMode {
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.applySearchFilter(false)
				}
			} else if m.commandMode {
				if len(m.commandQuery) > 0 {
					m.commandQuery = m.commandQuery[:len(m.commandQuery)-1]
				}
			}

		case tea.KeyUp, tea.KeyDown:
			// Navigation handled below
			break
		}

		// If we're in command mode, don't process other key bindings
		if m.commandMode {
			return m, nil
		}

		// Handle specific key bindings
		switch msg.String() {
		case "j":
			// Move cursor down
			listLen := m.currentListLen()
			if m.cursor < listLen-1 {
				m.cursor++
				m.updateViewportContent()
			}
			// Auto-scroll viewport if needed
			m.ensureCursorVisible()
		case "k":
			// Move cursor up
			if m.cursor > 0 {
				m.cursor--
				m.updateViewportContent()
			}
			// Auto-scroll viewport if needed
			m.ensureCursorVisible()
		case "/":
			// Enter search mode
			m.searchMode = true
			m.searchQuery = ""
			m.applySearchFilter(false)
		case ":":
			// Enter command mode
			if !m.searchMode && !m.commandMode {
				m.commandMode = true
				m.commandQuery = ""
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
			if !m.searchMode && !m.commandMode {
				m.cycleViewMode()
			}
		case "h":
			// Collapse selected group node
			m.collapseNode(m.selectedVisibleNode())
		case "l":
			// Expand selected group node
			m.expandNode(m.selectedVisibleNode())
		case "z":
			if !m.searchMode && m.isGroupedView() {
				m.pendingKey = "z"
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
		m.width = msg.Width
		m.height = msg.Height
		// Initialize or update viewport dimensions
		viewportHeight := m.height - headerFooterLines // Reserve 1 line for header, 1 line for footer
		m.viewport = viewport.New(msg.Width, viewportHeight)
		// Update viewport content
		m.updateViewportContent()
	}

	return m, nil
}

// View renders the TUI.
func (m *Model) View() string {
	if m.width == 0 {
		m.width = defaultViewportWidth
	}
	if m.height == 0 {
		m.height = 24
	}

	// Ensure viewport is initialized
	if m.viewport.Height == 0 {
		viewportHeight := m.height - headerFooterLines // Reserve 1 line for header, 1 line for footer
		m.viewport = viewport.New(m.width, viewportHeight)
		m.updateViewportContent()
	}

	var s strings.Builder

	// Header
	s.WriteString(render.Header(m.width))

	// Viewport with table rows
	s.WriteString("\n")
	s.WriteString(m.viewport.View())

	// Footer
	s.WriteString("\n")
	s.WriteString(render.Footer(render.FooterState{
		SearchMode:   m.searchMode,
		CommandMode:  m.commandMode,
		SearchQuery:  m.searchQuery,
		CommandQuery: m.commandQuery,
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
	return settings.TUIState{
		Columns:               m.columns,
		SortBy:                m.sortBy,
		SortOrder:             m.sortOrder,
		Filters:               m.filters,
		ViewMode:              m.viewMode,
		GroupBy:               m.groupBy,
		DefaultExpandLevel:    m.defaultExpandLevel,
		DefaultExpandLevelSet: true,

		ExpansionState: m.expansionState,
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
	if state.ViewMode != "" {
		m.viewMode = state.ViewMode
	}
	if state.GroupBy != "" {
		m.groupBy = state.GroupBy
	}
	if state.DefaultExpandLevelSet {
		m.defaultExpandLevel = state.DefaultExpandLevel
	}

	if state.ExpansionState != nil {
		m.expansionState = state.ExpansionState
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

	m.applySearchFilter(false)
	return nil
}

// NewModel creates a new TUI model.
// If client is nil, a new DefaultClient is created.
func NewModel(client tmux.TmuxClient) (*Model, error) {
	if client == nil {
		client = tmux.NewDefaultClient()
	}

	// Fetch all session names from tmux
	sessionNames, err := client.ListSessions()
	if err != nil {
		sessionNames = make(map[string]string)
	}

	// Fetch all window names from tmux
	windowNames, err := client.ListWindows()
	if err != nil {
		windowNames = make(map[string]string)
	}

	// Fetch all pane names from tmux
	paneNames, err := client.ListPanes()
	if err != nil {
		paneNames = make(map[string]string)
	}

	m := Model{
		viewport:          viewport.New(defaultViewportWidth, defaultViewportHeight), // Default dimensions, will be updated on WindowSizeMsg
		sessionNames:      sessionNames,
		windowNames:       windowNames,
		paneNames:         paneNames,
		client:            client,
		expansionState:    map[string]bool{},
		ensureTmuxRunning: core.EnsureTmuxRunning,
		jumpToPane:        core.JumpToPane,
	}
	err = m.loadNotifications(false)
	if err != nil {
		return &Model{}, err
	}
	return &m, nil
}

// saveSettingsSuccessMsg is sent when settings are saved successfully.
type saveSettingsSuccessMsg struct{}

// saveSettingsFailedMsg is sent when settings save fails.
type saveSettingsFailedMsg struct {
	err error
}

// applySearchFilter filters notifications based on the search query.
// If preserveCursor is true, the current cursor position is preserved.
// Otherwise, the cursor is reset to 0.
func (m *Model) applySearchFilter(preserveCursor bool) {
	m.invalidateCache()

	query := strings.TrimSpace(m.searchQuery)
	if query == "" {
		m.filtered = m.notifications
	} else {
		// Use custom provider if set, otherwise use default token provider
		var provider search.Provider
		if m.searchProvider != nil {
			provider = m.searchProvider
		} else {
			// Default: token-based search with case-insensitivity and name maps (backward compatible)
			provider = search.NewTokenProvider(
				search.WithCaseInsensitive(true),
				search.WithSessionNames(m.sessionNames),
				search.WithWindowNames(m.windowNames),
				search.WithPaneNames(m.paneNames),
			)
		}

		// Filter using the provider
		m.filtered = []notification.Notification{}
		for _, n := range m.notifications {
			if provider.Match(n, query) {
				m.filtered = append(m.filtered, n)
			}
		}
	}
	if m.isGroupedView() {
		m.treeRoot = m.buildFilteredTree(m.filtered)
		m.visibleNodes = m.computeVisibleNodes()
	} else {
		m.treeRoot = nil
		m.invalidateCache()
		m.visibleNodes = nil
	}
	if !preserveCursor {
		m.cursor = 0
	}
	m.updateViewportContent()
}

// getNodeIdentifier returns a stable identifier for a node.
// For notification nodes, this is the notification ID.
// For group nodes, this is a combination of the node kind and title.
func (m *Model) getNodeIdentifier(node *Node) string {
	if node == nil {
		return ""
	}
	if node.Kind == NodeKindNotification && node.Notification != nil {
		return fmt.Sprintf("notif:%d", node.Notification.ID)
	}
	// For group nodes, use the node kind and path
	if node.Kind == NodeKindRoot {
		return "root"
	}
	path, ok := findNodePath(m.treeRoot, node)
	if !ok || len(path) == 0 {
		return ""
	}
	var parts []string
	for _, n := range path {
		if n.Kind == NodeKindRoot {
			continue
		}
		parts = append(parts, string(n.Kind), n.Title)
	}
	return strings.Join(parts, ":")
}

// findNodeByIdentifier finds a node by its identifier in the visible nodes list.
func (m *Model) findNodeByIdentifier(identifier string) *Node {
	for _, node := range m.visibleNodes {
		if m.getNodeIdentifier(node) == identifier {
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
		for i, node := range m.visibleNodes {
			if node == targetNode {
				m.cursor = i
				m.ensureCursorVisible()
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
	if listLen == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= listLen {
		m.cursor = listLen - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureCursorVisible()
}

// executeCommand executes the current command query and returns a command to run.
func (m *Model) executeCommand() tea.Cmd {
	cmd := strings.TrimSpace(m.commandQuery)
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

		if m.groupBy == groupBy {
			return nil
		}

		m.groupBy = groupBy
		m.applySearchFilter(false)
		if err := m.saveSettings(); err != nil {
			colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
			return nil
		}
		colors.Info(fmt.Sprintf("Group by: %s", m.groupBy))
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

		if m.defaultExpandLevel == level {
			return nil
		}

		m.defaultExpandLevel = level
		if m.isGroupedView() {
			m.applyDefaultExpansion()
		}
		if err := m.saveSettings(); err != nil {
			colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
			return nil
		}
		colors.Info(fmt.Sprintf("Default expand level: %d", m.defaultExpandLevel))
		return nil

	case "toggle-view":
		if len(args) > 0 {
			colors.Warning("Invalid usage: toggle-view")
			return nil
		}

		if m.isGroupedView() {
			m.viewMode = viewModeDetailed
		} else {
			m.viewMode = viewModeGrouped
		}
		m.applySearchFilter(false)
		if err := m.saveSettings(); err != nil {
			colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		}
		colors.Info(fmt.Sprintf("View mode: %s", m.viewMode))
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

// updateViewportContent updates the viewport with the current filtered notifications.
func (m *Model) updateViewportContent() {
	var content strings.Builder

	if m.isGroupedView() {
		if len(m.visibleNodes) == 0 {
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No notifications found"))
		} else {
			now := time.Now()
			for rowIndex, node := range m.visibleNodes {
				if node == nil {
					continue
				}
				if rowIndex > 0 {
					content.WriteString("\n")
				}
				if isGroupNode(node) {
					content.WriteString(render.RenderGroupRow(render.GroupRow{
						Node: &render.GroupNode{
							Title:    node.Title,
							Display:  node.Display,
							Expanded: node.Expanded,
							Count:    node.Count,
						},
						Selected: rowIndex == m.cursor,
						Level:    getTreeLevel(node),
						Width:    m.width,
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
					Width:        m.width,
					Selected:     rowIndex == m.cursor,
					Now:          now,
				}))
			}
		}

		m.viewport.SetContent(content.String())
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
				Width:        m.width,
				Selected:     i == m.cursor,
				Now:          now,
			}))
		}
	}

	m.viewport.SetContent(content.String())
}

// ensureCursorVisible ensures the cursor is visible in the viewport.
func (m *Model) ensureCursorVisible() {
	if m.currentListLen() == 0 {
		return
	}

	// Get the current viewport line offset
	lineOffset := m.viewport.YOffset

	// Calculate the viewport height
	viewportHeight := m.viewport.Height

	// If cursor is above viewport, scroll up
	if m.cursor < lineOffset {
		m.viewport.LineUp(lineOffset - m.cursor)
	}

	// If cursor is below viewport, scroll down
	if m.cursor >= lineOffset+viewportHeight {
		m.viewport.LineDown(m.cursor - (lineOffset + viewportHeight) + 1)
	}
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
	oldCursor := m.cursor

	// Reload notifications to get updated state (preserve cursor)
	if err := m.loadNotifications(true); err != nil {
		colors.Error(fmt.Sprintf("Failed to reload notifications: %v", err))
		return nil
	}

	// Restore cursor to the saved position, adjusting for bounds
	listLen := m.currentListLen()
	if listLen == 0 {
		m.cursor = 0
	} else {
		m.cursor = oldCursor
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
	ensureTmuxRunning := m.ensureTmuxRunning
	if ensureTmuxRunning == nil {
		ensureTmuxRunning = core.EnsureTmuxRunning
	}
	if !ensureTmuxRunning() {
		colors.Error("tmux not running")
		return nil
	}

	// Jump to the pane
	jumpToPane := m.jumpToPane
	if jumpToPane == nil {
		jumpToPane = core.JumpToPane
	}
	if !jumpToPane(selected.Session, selected.Window, selected.Pane) {
		colors.Error("jump: failed to jump to pane")
		return nil
	}

	// Exit TUI after successful jump
	return tea.Quit
}

// loadNotifications loads notifications from storage.
// If preserveCursor is true, the current cursor position is preserved.
func (m *Model) loadNotifications(preserveCursor bool) error {
	lines, err := storage.ListNotifications("active", "", "", "", "", "", "", "")
	if err != nil {
		return fmt.Errorf("failed to load notifications: %w", err)
	}
	if lines == "" {
		m.notifications = []notification.Notification{}
		m.filtered = []notification.Notification{}
		m.treeRoot = nil
		m.invalidateCache()
		m.visibleNodes = nil
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
	m.applySearchFilter(preserveCursor)
	return nil
}

func (m *Model) isGroupedView() bool {
	return m.viewMode == viewModeGrouped
}

// cycleViewMode cycles through available view modes (compact → detailed → grouped).
func (m *Model) cycleViewMode() {
	nextMode := nextViewMode(m.viewMode)
	if nextMode == m.viewMode {
		return
	}

	m.viewMode = nextMode
	m.applySearchFilter(false)

	if err := m.saveSettings(); err != nil {
		colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}
	colors.Info(fmt.Sprintf("View mode: %s", m.viewMode))
}

// nextViewMode returns the next view mode in the cycle.
func nextViewMode(current string) string {
	availableModes := []string{viewModeCompact, viewModeDetailed}
	if isViewModeAvailable(viewModeGrouped) {
		availableModes = append(availableModes, viewModeGrouped)
	}

	for i, mode := range availableModes {
		if mode == current {
			return availableModes[(i+1)%len(availableModes)]
		}
	}

	return availableModes[0]
}

// isViewModeAvailable returns true if the given mode is a valid view mode.

func isViewModeAvailable(mode string) bool {
	return mode == viewModeCompact || mode == viewModeDetailed || mode == viewModeGrouped
}

func (m *Model) computeVisibleNodes() []*Node {
	if m.cacheValid {
		return m.visibleNodesCache
	}

	if m.treeRoot == nil {
		m.visibleNodesCache = nil
		m.cacheValid = true
		return nil
	}

	var visible []*Node
	var walk func(node *Node)
	walk = func(node *Node) {
		if node == nil {
			return
		}
		if node.Kind != NodeKindRoot {
			visible = append(visible, node)
		}
		if node.Kind == NodeKindNotification {
			return
		}
		if node.Kind != NodeKindRoot && !node.Expanded {
			return
		}
		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(m.treeRoot)
	m.visibleNodesCache = visible
	m.cacheValid = true
	return visible
}

func (m *Model) invalidateCache() {
	m.visibleNodesCache = nil
	m.cacheValid = false
}

func isGroupNode(node *Node) bool {
	if node == nil {
		return false
	}
	return node.Kind != NodeKindNotification && node.Kind != NodeKindRoot
}

func getTreeLevel(node *Node) int {
	if node == nil {
		return 0
	}
	switch node.Kind {
	case NodeKindSession:
		return 0
	case NodeKindWindow:
		return 1
	case NodeKindPane:
		return 2
	default:
		return 0
	}
}
func (m *Model) currentListLen() int {
	if m.isGroupedView() {
		return len(m.visibleNodes)
	}
	return len(m.filtered)
}

func (m *Model) selectedNotification() (notification.Notification, bool) {
	if m.isGroupedView() {
		if m.cursor < 0 || m.cursor >= len(m.visibleNodes) {
			return notification.Notification{}, false
		}
		node := m.visibleNodes[m.cursor]
		if node == nil || node.Notification == nil {
			return notification.Notification{}, false
		}
		return *node.Notification, true
	}

	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return notification.Notification{}, false
	}
	return m.filtered[m.cursor], true
}

func (m *Model) selectedVisibleNode() *Node {
	if !m.isGroupedView() {
		return nil
	}
	if m.cursor < 0 || m.cursor >= len(m.visibleNodes) {
		return nil
	}
	return m.visibleNodes[m.cursor]
}

func (m *Model) toggleNodeExpansion() bool {
	node := m.selectedVisibleNode()
	if node == nil || node.Kind == NodeKindNotification {
		return false
	}
	if node.Expanded {
		m.collapseNode(node)
		return true
	}
	m.expandNode(node)
	return true
}

func (m *Model) toggleFold() {
	if !m.isGroupedView() {
		return
	}
	node := m.selectedVisibleNode()
	if node == nil || node.Kind == NodeKindNotification {
		return
	}
	if m.allGroupsCollapsed() {
		m.applyDefaultExpansion()
		return
	}
	if node.Expanded {
		m.collapseNode(node)
		return
	}
	m.expandNode(node)
}

func (m *Model) allGroupsCollapsed() bool {
	if m.treeRoot == nil {
		return false
	}
	collapsed := true
	seen := false
	var walk func(node *Node)
	walk = func(node *Node) {
		if node == nil || !collapsed {
			return
		}
		if isGroupNode(node) {
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
	walk(m.treeRoot)
	return seen && collapsed
}

func (m *Model) applyDefaultExpansion() {
	if m.treeRoot == nil {
		return
	}
	selected := m.selectedVisibleNode()
	level := m.defaultExpandLevel
	if level < settings.MinExpandLevel {
		level = settings.MinExpandLevel
	}
	if level > settings.MaxExpandLevel {
		level = settings.MaxExpandLevel
	}

	var walk func(node *Node)
	walk = func(node *Node) {
		if node == nil {
			return
		}
		if isGroupNode(node) {
			nodeLevel := getTreeLevel(node) + 1
			expanded := nodeLevel <= level
			node.Expanded = expanded
			m.updateExpansionState(node, expanded)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(m.treeRoot)

	m.invalidateCache()
	m.visibleNodes = m.computeVisibleNodes()
	if selected != nil {
		if index := indexOfNode(m.visibleNodes, selected); index >= 0 {
			m.cursor = index
		}
	}
	if m.cursor >= len(m.visibleNodes) {
		m.cursor = len(m.visibleNodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.updateViewportContent()
	m.ensureCursorVisible()
}
func (m *Model) expandNode(node *Node) {
	if !m.isGroupedView() {
		return
	}
	if node == nil || node.Kind == NodeKindNotification {
		return
	}
	if node.Expanded {
		return
	}

	node.Expanded = true
	m.updateExpansionState(node, true)
	m.invalidateCache()
	m.visibleNodes = m.computeVisibleNodes()
	m.updateViewportContent()
	m.ensureCursorVisible()
}

func (m *Model) collapseNode(node *Node) {
	if !m.isGroupedView() {
		return
	}
	if node == nil || node.Kind == NodeKindNotification {
		return
	}
	if !node.Expanded {
		return
	}

	selected := m.selectedVisibleNode()
	node.Expanded = false
	m.updateExpansionState(node, false)
	m.invalidateCache()
	m.visibleNodes = m.computeVisibleNodes()
	if selected != nil && nodeContains(node, selected) {
		if index := indexOfNode(m.visibleNodes, node); index >= 0 {
			m.cursor = index
		}
	}
	if m.cursor >= len(m.visibleNodes) {
		m.cursor = len(m.visibleNodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.updateViewportContent()
	m.ensureCursorVisible()
}

func (m *Model) updateExpansionState(node *Node, expanded bool) {
	key := m.nodeExpansionKey(node)
	if key == "" {
		return
	}
	if m.expansionState == nil {
		m.expansionState = map[string]bool{}
	}
	legacyKey := m.nodeExpansionLegacyKey(node)
	if legacyKey != "" && legacyKey != key {
		delete(m.expansionState, legacyKey)
	}
	m.expansionState[key] = expanded
}

func (m *Model) nodeExpansionKey(node *Node) string {
	if node == nil || node.Kind == NodeKindNotification || node.Kind == NodeKindRoot {
		return ""
	}
	path, ok := findNodePath(m.treeRoot, node)
	if !ok || len(path) == 0 {
		return ""
	}

	session, window, pane := nodePathSegments(path)

	switch node.Kind {
	case NodeKindSession:
		return serializeNodeExpansionPath(NodeKindSession, session)
	case NodeKindWindow:
		if session == "" {
			return ""
		}
		return serializeNodeExpansionPath(NodeKindWindow, session, window)
	case NodeKindPane:
		if session == "" || window == "" {
			return ""
		}
		return serializeNodeExpansionPath(NodeKindPane, session, window, pane)
	default:
		return ""
	}
}

func (m *Model) nodeExpansionLegacyKey(node *Node) string {
	if node == nil || node.Kind == NodeKindNotification || node.Kind == NodeKindRoot {
		return ""
	}
	path, ok := findNodePath(m.treeRoot, node)
	if !ok || len(path) == 0 {
		return ""
	}

	session, window, pane := nodePathSegments(path)

	switch node.Kind {
	case NodeKindSession:
		return serializeLegacyNodeExpansionPath(NodeKindSession, session)
	case NodeKindWindow:
		if session == "" {
			return ""
		}
		return serializeLegacyNodeExpansionPath(NodeKindWindow, session, window)
	case NodeKindPane:
		if session == "" || window == "" {
			return ""
		}
		return serializeLegacyNodeExpansionPath(NodeKindPane, session, window, pane)
	default:
		return ""
	}
}

func serializeNodeExpansionPath(kind NodeKind, parts ...string) string {
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

func serializeLegacyNodeExpansionPath(kind NodeKind, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%s", kind, strings.Join(parts, ":"))
}

func nodePathSegments(path []*Node) (session string, window string, pane string) {
	for _, current := range path {
		switch current.Kind {
		case NodeKindSession:
			session = current.Title
		case NodeKindWindow:
			window = current.Title
		case NodeKindPane:
			pane = current.Title
		}
	}
	return session, window, pane
}

func findNodePath(root *Node, target *Node) ([]*Node, bool) {
	if root == nil || target == nil {
		return nil, false
	}
	if root == target {
		return []*Node{root}, true
	}
	for _, child := range root.Children {
		path, ok := findNodePath(child, target)
		if ok {
			return append([]*Node{root}, path...), true
		}
	}
	return nil, false
}

func nodeContains(root *Node, target *Node) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	for _, child := range root.Children {
		if nodeContains(child, target) {
			return true
		}
	}
	return false
}

func indexOfNode(nodes []*Node, target *Node) int {
	for i, node := range nodes {
		if node == target {
			return i
		}
	}
	return -1
}

func expandTree(node *Node) {
	if node == nil {
		return
	}
	if node.Kind != NodeKindNotification {
		node.Expanded = true
	}
	for _, child := range node.Children {
		expandTree(child)
	}
}

// buildFilteredTree builds a tree from filtered notifications and applies saved expansion state.
// Returns a tree where group counts reflect only matching notifications.
func (m *Model) buildFilteredTree(notifications []notification.Notification) *Node {
	m.invalidateCache()

	if len(notifications) == 0 {
		return nil
	}

	root := BuildTree(notifications, m.groupBy)

	// Prune empty groups (groups with no matching notifications)
	m.pruneEmptyGroups(root)

	// FIX: Set treeRoot before applying expansion state
	// to ensure consistent key generation
	m.treeRoot = root

	// Apply saved expansion state where possible
	if m.expansionState != nil {
		m.applyExpansionState(root)
	} else {
		// If no saved state, expand all by default
		expandTree(root)
	}

	return root
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
func (m *Model) applyExpansionState(node *Node) {
	if node == nil {
		return
	}

	// Apply expansion state to group nodes
	if isGroupNode(node) {
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

func (m *Model) expansionStateValue(node *Node) (bool, bool) {
	if m.expansionState == nil {
		return false, false
	}

	key := m.nodeExpansionKey(node)
	if key != "" {
		expanded, ok := m.expansionState[key]
		if ok {
			return expanded, true
		}
	}

	legacyKey := m.nodeExpansionLegacyKey(node)
	if legacyKey == "" {
		return false, false
	}

	expanded, ok := m.expansionState[legacyKey]
	if !ok {
		return false, false
	}
	if key != "" {
		m.expansionState[key] = expanded
		delete(m.expansionState, legacyKey)
	}
	return expanded, true
}

// getSessionName returns the session name for a session ID.
// Uses cached session names from initial fetch.
func (m *Model) getSessionName(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	if m.sessionNames == nil {
		return sessionID
	}
	if name, ok := m.sessionNames[sessionID]; ok {
		return name
	}
	return sessionID // fallback to session ID if not found
}

// getWindowName returns the window name for a window ID.
// Uses cached window names from initial fetch.
func (m *Model) getWindowName(windowID string) string {
	if windowID == "" {
		return ""
	}
	if m.windowNames == nil {
		return windowID
	}
	if name, ok := m.windowNames[windowID]; ok {
		return name
	}
	return windowID // fallback to window ID if not found
}

// getPaneName returns the pane name for a pane ID.
// Uses cached pane names from initial fetch.
func (m *Model) getPaneName(paneID string) string {
	if paneID == "" {
		return ""
	}
	if m.paneNames == nil {
		return paneID
	}
	if name, ok := m.paneNames[paneID]; ok {
		return name
	}
	return paneID // fallback to pane ID if not found
}
