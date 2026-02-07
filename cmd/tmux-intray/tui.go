/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
)

// ansiColorNumber extracts the color number from an ANSI escape sequence.
// Example: "\033[0;34m" -> "34"
func ansiColorNumber(ansi string) string {
	// Remove escape sequence prefix and suffix
	if len(ansi) < 2 {
		return ""
	}
	// Find the last ';' before the 'm'
	lastSemicolon := strings.LastIndex(ansi, ";")
	if lastSemicolon == -1 {
		return ""
	}
	// Extract number between ';' and 'm'
	color := ansi[lastSemicolon+1 : len(ansi)-1]
	return color
}

// sessionNameFetcher fetches session name from tmux for a given session ID.
// Can be replaced for testing.
var sessionNameFetcher = func(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	cmd := exec.Command("tmux", "display-message", "-t", sessionID, "-p", "#S")
	stdout, err := cmd.Output()
	if err != nil {
		return sessionID // fallback to session ID on error
	}
	return strings.TrimSpace(string(stdout))
}

// fetchAllSessionNames fetches all session IDs and names from tmux with a single call.
// Returns a map from session ID to session name.
// Can be replaced for testing.
var fetchAllSessionNames = func() map[string]string {
	names := make(map[string]string)
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_id}\t#{session_name}")
	stdout, err := cmd.Output()
	if err != nil {
		return names // empty map on error
	}

	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			names[parts[0]] = parts[1]
		}
	}
	return names
}

// tuiModel represents the TUI model for bubbletea.
type tuiModel struct {
	notifications []Notification
	filtered      []Notification
	cursor        int
	searchQuery   string
	searchMode    bool
	commandMode   bool
	commandQuery  string
	viewport      viewport.Model
	width         int
	height        int
	sessionNames  map[string]string
}

// Init initializes the TUI model.
func (m *tuiModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state.
func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
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
			// Quit
			return m, tea.Quit
		}

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

// applySearchFilter filters notifications based on the search query.
func (m *tuiModel) applySearchFilter() {
	if m.searchQuery == "" {
		m.filtered = m.notifications
		m.cursor = 0
		m.updateViewportContent()
		return
	}

	query := strings.ToLower(m.searchQuery)
	m.filtered = []Notification{}
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
func (m *tuiModel) executeCommand() tea.Cmd {
	cmd := strings.TrimSpace(m.commandQuery)
	switch cmd {
	case "q":
		return tea.Quit
	default:
		// Unknown command - ignore
		return nil
	}
}

// View renders the TUI.
func (m *tuiModel) View() string {
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
	s.WriteString(m.renderHeader())

	// Viewport with table rows
	s.WriteString("\n")
	s.WriteString(m.viewport.View())

	// Footer
	s.WriteString("\n")
	s.WriteString(m.renderFooter())

	return s.String()
}

// updateViewportContent updates the viewport with the current filtered notifications.
func (m *tuiModel) updateViewportContent() {
	var content strings.Builder

	if len(m.filtered) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No notifications found"))
	} else {
		for i, notif := range m.filtered {
			if i > 0 {
				content.WriteString("\n")
			}
			content.WriteString(m.renderRow(notif, i == m.cursor))
		}
	}

	m.viewport.SetContent(content.String())
}

// ensureCursorVisible ensures the cursor is visible in the viewport.
func (m *tuiModel) ensureCursorVisible() {
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

// renderHeader renders the table header.
func (m tuiModel) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ansiColorNumber(colors.Blue))) // Use ANSI color number

	// Column widths: TYPE=8, STATUS=8, SESSION=25, MESSAGE=variable, PANE=7, AGE=5
	typeWidth := 8
	statusWidth := 8
	sessionWidth := 25
	paneWidth := 7
	ageWidth := 5
	totalFixedWidth := typeWidth + statusWidth + sessionWidth + paneWidth + ageWidth
	spacesBetweenColumns := 10 // (6 columns - 1) * 2 spaces
	messageWidth := m.width - totalFixedWidth - spacesBetweenColumns

	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
		typeWidth, "TYPE",
		statusWidth, "STATUS",
		sessionWidth, "SESSION",
		messageWidth, "MESSAGE",
		paneWidth, "PANE",
		ageWidth, "AGE",
	)

	return headerStyle.Render(header)
}

// renderRow renders a single notification row.
func (m tuiModel) renderRow(notif Notification, isSelected bool) string {
	rowStyle := lipgloss.NewStyle()
	if isSelected {
		rowStyle = rowStyle.Background(lipgloss.Color(ansiColorNumber(colors.Blue))).Foreground(lipgloss.Color("0"))
	}

	// Get level icon
	levelIcon := m.getLevelIcon(notif.Level)

	// Get status icon
	statusIcon := getStatusIcon(notif.State)

	// Truncate message
	message := notif.Message
	if len(message) > 50 {
		message = message[:47] + "..."
	}

	// Calculate age
	age := calculateAge(notif.Timestamp)

	// Session column
	session := m.getSessionName(notif.Session)
	// Pane column (just pane ID)
	pane := notif.Pane

	// Column widths
	typeWidth := 8
	statusWidth := 8
	sessionWidth := 25
	paneWidth := 7
	ageWidth := 5
	totalFixedWidth := typeWidth + statusWidth + sessionWidth + paneWidth + ageWidth
	spacesBetweenColumns := 10 // (6 columns - 1) * 2 spaces
	messageWidth := m.width - totalFixedWidth - spacesBetweenColumns

	// Use default width if not set or too small
	if m.width == 0 || messageWidth < 10 {
		messageWidth = 50
	}

	// Truncate session if needed
	if len(session) > sessionWidth {
		session = session[:sessionWidth-3] + "..."
	}

	// Truncate pane if needed
	if len(pane) > paneWidth {
		pane = pane[:paneWidth-3] + "..."
	}

	// Truncate message to fit

	if len(message) > messageWidth {
		message = message[:messageWidth-3] + "..."
	}

	row := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
		typeWidth, levelIcon,
		statusWidth, statusIcon,
		sessionWidth, session,
		messageWidth, message,
		paneWidth, pane,
		ageWidth, age,
	)

	return rowStyle.Render(row)
}

// getLevelIcon returns the icon for a notification level.
func (m tuiModel) getLevelIcon(level string) string {
	switch level {
	case "error":
		return "❌ err"
	case "warning":
		return "⚠️ wrn"
	case "critical":
		return "‼️ crt"
	case "info", "":
		return "ℹ️ inf"
	default:
		return "ℹ️ " + level[:3]
	}
}

// getStatusIcon returns the icon for a notification state.
func getStatusIcon(state string) string {
	switch state {
	case "active", "":
		return "●"
	case "dismissed":
		return "○"
	default:
		return "?"
	}
}

// calculateAge calculates the relative age from a timestamp.
func calculateAge(timestamp string) string {
	if timestamp == "" {
		return ""
	}

	// Parse the timestamp
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return ""
	}

	// Calculate duration
	duration := time.Since(t)

	// Format the age
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
}

// renderFooter renders the footer with help text.
func (m tuiModel) renderFooter() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var help []string
	help = append(help, "j/k: move")
	if m.searchMode {
		help = append(help, "ESC: exit search")
		help = append(help, fmt.Sprintf("Search: %s", m.searchQuery))
	} else if m.commandMode {
		help = append(help, "ESC: cancel")
		help = append(help, fmt.Sprintf(":%s", m.commandQuery))
	} else {
		help = append(help, "/: search")
		help = append(help, ":: command")
	}
	help = append(help, "d: dismiss")
	enterHelp := "Enter: jump"
	if m.commandMode {
		enterHelp = "Enter: execute"
	}
	help = append(help, enterHelp)
	help = append(help, "q: quit")

	return helpStyle.Render(strings.Join(help, "  |  "))
}

// handleDismiss handles the dismiss action for the selected notification.
func (m *tuiModel) handleDismiss() tea.Cmd {
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
func (m *tuiModel) handleJump() tea.Cmd {
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
func (m *tuiModel) loadNotifications() error {
	lines := storage.ListNotifications("active", "", "", "", "", "", "")
	if lines == "" {
		m.notifications = []Notification{}
		m.filtered = []Notification{}
		m.updateViewportContent()
		return nil
	}

	var notifications []Notification
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, err := parseNotification(line)
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

// getSessionName returns the session name for a session ID, fetching from tmux if not cached.
func (m *tuiModel) getSessionName(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	if m.sessionNames == nil {
		m.sessionNames = make(map[string]string)
	}
	if name, ok := m.sessionNames[sessionID]; ok {
		return name
	}
	name := sessionNameFetcher(sessionID)
	m.sessionNames[sessionID] = name
	return name
}

// NewTUIModel creates a new TUI model.
func NewTUIModel() (*tuiModel, error) {
	m := tuiModel{
		viewport:     viewport.New(80, 22), // Default dimensions, will be updated on WindowSizeMsg
		sessionNames: fetchAllSessionNames(),
	}
	err := m.loadNotifications()
	if err != nil {
		return &tuiModel{}, err
	}
	return &m, nil
}

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive terminal UI for notifications",
	Long: `Interactive terminal UI for notifications.

USAGE:
    tmux-intray tui

KEY BINDINGS:
    j/k         Move up/down in the list
    /           Enter search mode
    :           Enter command mode
    ESC         Exit search/command mode, or quit TUI
    d           Dismiss selected notification
    Enter       Jump to pane (or execute command in command mode)
    q           Quit TUI`,
	Run: runTUI,
}

func init() {
	cmd.RootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) {
	// Initialize storage
	storage.Init()

	// Create TUI model
	model, err := NewTUIModel()
	if err != nil {
		colors.Error(fmt.Sprintf("Failed to create TUI model: %v", err))
		os.Exit(1)
	}

	// Create and run the bubbletea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Start the program
	if _, err := p.Run(); err != nil {
		colors.Error(fmt.Sprintf("Error running TUI: %v", err))
		os.Exit(1)
	}
}
