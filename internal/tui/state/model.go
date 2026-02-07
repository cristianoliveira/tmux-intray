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
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/render"
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
	viewport      viewport.Model
	width         int
	height        int
	sessionNames  map[string]string
	client        tmux.TmuxClient // TmuxClient for tmux operations

	// Settings fields
	sortBy             string
	sortOrder          string
	columns            []string
	filters            settings.Filter
	viewMode           string
	groupBy            string
	defaultExpandLevel int
	expansionState     map[string]bool
	loadedSettings     *settings.Settings // Track loaded settings for comparison
}

// Init initializes the TUI model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
				m.applySearchFilter()
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
			// Jump to pane of selected notification
			return m, m.handleJump()

		case tea.KeyRunes:
			if m.searchMode {
				// In search mode, append runes to search query
				for _, r := range msg.Runes {
					m.searchQuery += string(r)
				}
				m.applySearchFilter()
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
					m.applySearchFilter()
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
			if m.cursor < len(m.filtered)-1 {
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
			m.applySearchFilter()
		case ":":
			// Enter command mode
			if !m.searchMode && !m.commandMode {
				m.commandMode = true
				m.commandQuery = ""
			}
		case "d":
			// Dismiss selected notification
			return m, m.handleDismiss()
		case "i":
			// In search mode, 'i' is handled by KeyRunes
			// This is a no-op but kept for documentation
		case "q":
			// Save settings before quitting
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
		viewportHeight := m.height - 2 // Reserve 1 line for header, 1 line for footer
		m.viewport = viewport.New(msg.Width, viewportHeight)
		// Update viewport content
		m.updateViewportContent()
	}

	return m, nil
}

// View renders the TUI.
func (m *Model) View() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}

	// Ensure viewport is initialized
	if m.viewport.Height == 0 {
		viewportHeight := m.height - 2 // Reserve 1 line for header, 1 line for footer
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
		ExpansionState:        m.expansionState,
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

	m := Model{
		viewport:       viewport.New(80, 22), // Default dimensions, will be updated on WindowSizeMsg
		sessionNames:   fetchAllSessionNames(),
		client:         client,
		expansionState: map[string]bool{},
	}
	err = m.loadNotifications()
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
func (m *Model) applySearchFilter() {
	if m.searchQuery == "" {
		m.filtered = m.notifications
		m.cursor = 0
		m.updateViewportContent()
		return
	}

	query := strings.ToLower(m.searchQuery)
	m.filtered = []notification.Notification{}
	for _, n := range m.notifications {
		if strings.Contains(strings.ToLower(n.Message), query) ||
			strings.Contains(strings.ToLower(n.Session), query) ||
			strings.Contains(strings.ToLower(n.Window), query) ||
			strings.Contains(strings.ToLower(n.Pane), query) {
			m.filtered = append(m.filtered, n)
		}
	}
	m.cursor = 0
	m.updateViewportContent()
}

// executeCommand executes the current command query and returns a command to run.
func (m *Model) executeCommand() tea.Cmd {
	cmd := strings.TrimSpace(m.commandQuery)
	switch cmd {
	case "q":
		// Save settings before quitting
		if err := m.saveSettings(); err != nil {
			colors.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		}
		return tea.Quit
	case "w":
		// Save settings and continue TUI
		return func() tea.Msg {
			if err := m.saveSettings(); err != nil {
				return saveSettingsFailedMsg{err: err}
			}
			return saveSettingsSuccessMsg{}
		}
	default:
		// Unknown command - ignore
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
	if len(m.filtered) == 0 {
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
	if len(m.filtered) == 0 {
		return nil
	}

	// Get the selected notification
	selected := m.filtered[m.cursor]

	// Dismiss the notification using storage
	id := strconv.Itoa(selected.ID)
	if err := storage.DismissNotification(id); err != nil {
		colors.Error(fmt.Sprintf("Failed to dismiss notification: %v", err))
		return nil
	}

	// Reload notifications to get updated state
	if err := m.loadNotifications(); err != nil {
		colors.Error(fmt.Sprintf("Failed to reload notifications: %v", err))
		return nil
	}

	// Adjust cursor if necessary
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Update viewport content
	m.updateViewportContent()

	return nil
}

// handleJump handles the jump action for the selected notification.
func (m *Model) handleJump() tea.Cmd {
	if len(m.filtered) == 0 {
		return nil
	}

	// Get the selected notification
	selected := m.filtered[m.cursor]

	// Check if notification has valid session, window, pane
	if selected.Session == "" || selected.Window == "" || selected.Pane == "" {
		colors.Error("Cannot jump: notification is missing session, window, or pane information")
		return nil
	}

	// Ensure tmux is running
	if !core.EnsureTmuxRunning() {
		colors.Error("tmux is not running")
		return nil
	}

	// Jump to the pane
	if !core.JumpToPane(selected.Session, selected.Window, selected.Pane) {
		colors.Error("Failed to jump to pane")
		return nil
	}

	// Exit TUI after successful jump
	return tea.Quit
}

// loadNotifications loads notifications from storage.
func (m *Model) loadNotifications() error {
	lines, err := storage.ListNotifications("active", "", "", "", "", "", "")
	if err != nil {
		return fmt.Errorf("failed to load notifications: %w", err)
	}
	if lines == "" {
		m.notifications = []notification.Notification{}
		m.filtered = []notification.Notification{}
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
	return nil
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
