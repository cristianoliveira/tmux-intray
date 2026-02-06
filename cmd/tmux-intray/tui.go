/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
)

// tuiModel represents the TUI model for bubbletea.
type tuiModel struct {
	notifications []Notification
	filtered      []Notification
	cursor        int
	searchQuery   string
	searchMode    bool
	viewport      tea.Model
	width         int
	height        int
}

// Init initializes the TUI model.
func (m tuiModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state.
func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// Exit
			return m, tea.Quit

		case tea.KeyEnter:
			// Jump to pane of selected notification
			return m, m.handleJump()

		case tea.KeyRunes:
			if m.searchMode {
				// In search mode, append runes to search query
				for _, r := range msg.Runes {
					m.searchQuery += string(r)
				}
				m.applySearchFilter()
			}

		case tea.KeyBackspace:
			if m.searchMode {
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.applySearchFilter()
				}
			}

		case tea.KeyUp, tea.KeyDown:
			// Navigation handled below
			break
		}

		// Handle specific key bindings
		switch msg.String() {
		case "j":
			// Move cursor down
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "k":
			// Move cursor up
			if m.cursor > 0 {
				m.cursor--
			}
		case "/":
			// Enter search mode
			m.searchMode = true
			m.searchQuery = ""
			m.applySearchFilter()
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
	}

	return m, nil
}

// applySearchFilter filters notifications based on the search query.
func (m *tuiModel) applySearchFilter() {
	if m.searchQuery == "" {
		m.filtered = m.notifications
		m.cursor = 0
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
}

// View renders the TUI.
func (m tuiModel) View() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}

	var s strings.Builder

	// Header
	s.WriteString(m.renderHeader())

	// Table rows
	if len(m.filtered) == 0 {
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No notifications found"))
	} else {
		for i, notif := range m.filtered {
			if i >= m.height-5 { // Reserve space for header and footer
				s.WriteString(fmt.Sprintf("\n%s ... %d more", lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" "), len(m.filtered)-i))
				break
			}
			s.WriteString("\n")
			s.WriteString(m.renderRow(notif, i == m.cursor))
		}
	}

	// Footer
	s.WriteString("\n\n")
	s.WriteString(m.renderFooter())

	return s.String()
}

// renderHeader renders the table header.
func (m tuiModel) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colors.Blue[1:])) // Strip escape code prefix

	// Column widths: TYPE=6, STATUS=7, SUMMARY=variable, SOURCE=15, AGE=8
	typeWidth := 6
	statusWidth := 7
	sourceWidth := 15
	ageWidth := 8
	summaryWidth := m.width - typeWidth - statusWidth - sourceWidth - ageWidth - 13 // 13 = spaces between columns

	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s",
		typeWidth, "TYPE",
		statusWidth, "STATUS",
		summaryWidth, "SUMMARY",
		sourceWidth, "SOURCE",
		ageWidth, "AGE",
	)

	return headerStyle.Render(header)
}

// renderRow renders a single notification row.
func (m tuiModel) renderRow(notif Notification, isSelected bool) string {
	rowStyle := lipgloss.NewStyle()
	if isSelected {
		rowStyle = rowStyle.Background(lipgloss.Color(colors.Blue[1:])).Foreground(lipgloss.Color("0"))
	}

	// Get level icon
	levelIcon := m.getLevelIcon(notif.Level)

	// Get status icon
	statusIcon := getStatusIcon(notif.State)

	// Truncate summary
	summary := notif.Message
	if len(summary) > 50 {
		summary = summary[:47] + "..."
	}

	// Calculate age
	age := calculateAge(notif.Timestamp)

	// Format source as Session:Window:Pane
	source := fmt.Sprintf("%s:%s:%s", notif.Session, notif.Window, notif.Pane)

	// Column widths
	typeWidth := 6
	statusWidth := 7
	sourceWidth := 15
	ageWidth := 8
	summaryWidth := m.width - typeWidth - statusWidth - sourceWidth - ageWidth - 13

	// Truncate source if needed
	if len(source) > sourceWidth {
		source = source[:sourceWidth-3] + "..."
	}

	// Truncate summary to fit
	if len(summary) > summaryWidth {
		summary = summary[:summaryWidth-3] + "..."
	}

	row := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s",
		typeWidth, levelIcon,
		statusWidth, statusIcon,
		summaryWidth, summary,
		sourceWidth, source,
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
	} else {
		help = append(help, "/: search")
	}
	help = append(help, "d: dismiss")
	help = append(help, "Enter: jump")
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

	m.notifications = notifications
	m.applySearchFilter()
	return nil
}

// NewTUIModel creates a new TUI model.
func NewTUIModel() (tuiModel, error) {
	m := tuiModel{}
	err := m.loadNotifications()
	if err != nil {
		return tuiModel{}, err
	}
	return m, nil
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
    ESC         Exit search mode
    d           Dismiss selected notification
    Enter       Jump to pane of selected notification
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
