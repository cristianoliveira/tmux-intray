package render

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
)

const (
	readStatusWidth      = 2
	typeWidth            = 8
	statusWidth          = 8
	sessionWidth         = 25
	paneWidth            = 7
	ageWidth             = 5
	spacesBetweenColumns = 12
	defaultMessageWidth  = 50
	groupIndentSize      = 2
	groupCollapsedSymbol = "▸"
	groupExpandedSymbol  = "▾"
)

// FooterState defines the inputs needed to render footer help text.
type FooterState struct {
	SearchMode  bool
	SearchQuery string

	Grouped      bool
	ViewMode     string
	Width        int
	ErrorMessage string
	ReadFilter   string
	ShowHelp     bool
}

// RowState defines the inputs needed to render a notification row.
type RowState struct {
	Notification notification.Notification
	SessionName  string
	Width        int
	Selected     bool
	Now          time.Time
}

// Header renders the table header.
func Header(width int) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ansiColorNumber(colors.Blue)))

	messageWidth := calculateMessageWidth(width)

	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
		readStatusWidth, "RD",
		typeWidth, "TYPE",
		statusWidth, "STATUS",
		sessionWidth, "SESSION",
		messageWidth, "MESSAGE",
		paneWidth, "PANE",
		ageWidth, "AGE",
	)

	return headerStyle.Render(header)
}

// Row renders a single notification row.
func Row(state RowState) string {
	levelIcon := levelIcon(state.Notification.Level)
	statusIcon := statusIcon(state.Notification.State)
	readIndicator := ReadStatusIndicator(state.Notification.IsRead(), state.Selected)

	message := state.Notification.Message
	if len(message) > defaultMessageWidth {
		message = message[:defaultMessageWidth-3] + "..."
	}

	age := calculateAge(state.Notification.Timestamp, state.Now)

	session := state.SessionName
	pane := state.Notification.Pane

	messageWidth := calculateMessageWidth(state.Width)
	if state.Width == 0 || messageWidth < 10 {
		messageWidth = defaultMessageWidth
	}

	if len(session) > sessionWidth {
		session = session[:sessionWidth-3] + "..."
	}

	if len(pane) > paneWidth {
		pane = pane[:paneWidth-3] + "..."
	}

	if len(message) > messageWidth {
		message = message[:messageWidth-3] + "..."
	}

	columns := []string{
		readIndicator,
		fmt.Sprintf("%-*s", typeWidth, levelIcon),
		fmt.Sprintf("%-*s", statusWidth, statusIcon),
		fmt.Sprintf("%-*s", sessionWidth, session),
		fmt.Sprintf("%-*s", messageWidth, message),
		fmt.Sprintf("%-*s", paneWidth, pane),
		fmt.Sprintf("%-*s", ageWidth, age),
	}

	if !state.Selected {
		return strings.Join(columns, "  ")
	}

	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color(ansiColorNumber(colors.Blue))).Foreground(lipgloss.Color("0"))
	var row strings.Builder
	for index, column := range columns {
		if index > 0 {
			row.WriteString(selectedStyle.Render("  "))
		}
		if index == 0 {
			row.WriteString(column)
			continue
		}
		row.WriteString(selectedStyle.Render(column))
	}

	return row.String()
}

// buildFullHelpSearchModeItems returns the help items for full help mode when searching.
func buildFullHelpSearchModeItems(state FooterState) []string {
	var items []string
	items = append(items, fmt.Sprintf("Search: %s", state.SearchQuery))
	items = append(items, fmt.Sprintf("mode: %s", viewModeIndicator(state.ViewMode)))
	items = append(items, fmt.Sprintf("read: %s", readFilterIndicator(state.ReadFilter)))
	items = append(items, "ESC: exit search")
	items = append(items, "Ctrl+j/k: navigate")
	items = append(items, "j/k: move")
	items = append(items, "gg/G: top/bottom")
	items = append(items, "r: read")
	items = append(items, "u: unread")
	items = append(items, "d: dismiss")
	enterHelp := "Enter: jump"
	if state.Grouped {
		enterHelp = "Enter: toggle/jump"
	}
	items = append(items, enterHelp)
	items = append(items, "q: quit")
	items = append(items, "?: toggle help")
	return items
}

// buildFullHelpNormalModeItems returns the help items for full help mode when not searching.
func buildFullHelpNormalModeItems(state FooterState) []string {
	var items []string
	items = append(items, fmt.Sprintf("mode: %s", viewModeIndicator(state.ViewMode)))
	items = append(items, fmt.Sprintf("read: %s", readFilterIndicator(state.ReadFilter)))
	items = append(items, "j/k: move")
	items = append(items, "gg/G: top/bottom")
	items = append(items, "/: search")
	items = append(items, "v: cycle view mode")
	if state.Grouped {
		items = append(items, "h/l: collapse/expand")
		items = append(items, "za: toggle fold")
		items = append(items, "D: dismiss group")
	}
	items = append(items, "r: read")
	items = append(items, "u: unread")
	items = append(items, "d: dismiss")
	enterHelp := "Enter: jump"
	if state.Grouped {
		enterHelp = "Enter: toggle/jump"
	}
	items = append(items, enterHelp)
	items = append(items, "q: quit")
	items = append(items, "?: toggle help")
	return items
}

// buildMinimalSearchModeItems returns the help items for minimal help mode when searching.
func buildMinimalSearchModeItems(state FooterState) []string {
	var items []string
	items = append(items, fmt.Sprintf("Search: %s", state.SearchQuery))
	items = append(items, "ESC: exit search")
	items = append(items, "Ctrl+j/k: navigate")
	items = append(items, fmt.Sprintf("mode: %s", viewModeIndicator(state.ViewMode)))
	return items
}

// buildMinimalNormalModeItems returns the help items for minimal help mode when not searching.
func buildMinimalNormalModeItems(state FooterState) []string {
	var items []string
	items = append(items, fmt.Sprintf("mode: %s", viewModeIndicator(state.ViewMode)))
	items = append(items, "j/k: move")
	return items
}

// Footer renders the footer with help text.
func Footer(state FooterState) string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	searchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ansiColorNumber(colors.Blue))).Bold(true)

	var items []string
	// Error message is rendered above the footer, not included here

	switch {
	case state.ShowHelp && state.SearchMode:
		items = buildFullHelpSearchModeItems(state)
	case state.ShowHelp && !state.SearchMode:
		items = buildFullHelpNormalModeItems(state)
	case !state.ShowHelp && state.SearchMode:
		items = buildMinimalSearchModeItems(state)
	default: // !state.ShowHelp && !state.SearchMode
		items = buildMinimalNormalModeItems(state)
	}

	// Apply styling to each item
	var styledParts []string
	for _, item := range items {
		if strings.HasPrefix(item, "Search: ") {
			styledParts = append(styledParts, searchStyle.Render(item))
		} else {
			styledParts = append(styledParts, helpStyle.Render(item))
		}
	}

	footer := strings.Join(styledParts, "  |  ")
	footer = truncateFooter(footer, state.Width)

	return footer + "\x1b[K"
}

func truncateFooter(value string, width int) string {
	if width <= 0 {
		return value
	}
	if utf8.RuneCountInString(value) <= width {
		return value
	}
	return string([]rune(value)[:width])
}

func viewModeIndicator(mode string) string {
	switch mode {
	case settings.ViewModeCompact:
		return "[C]"
	case settings.ViewModeDetailed:
		return "[D]"
	case settings.ViewModeGrouped:
		return "[G]"
	default:
		return "[?]"
	}
}

func readFilterIndicator(filter string) string {
	switch filter {
	case settings.ReadFilterRead:
		return "read"
	case settings.ReadFilterUnread:
		return "unread"
	default:
		return "all"
	}
}

func calculateMessageWidth(width int) int {
	totalFixedWidth := readStatusWidth + typeWidth + statusWidth + sessionWidth + paneWidth + ageWidth
	return width - totalFixedWidth - spacesBetweenColumns
}

func levelIcon(level string) string {
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

func statusIcon(state string) string {
	switch state {
	case "active", "":
		return "●"
	case "dismissed":
		return "○"
	default:
		return "?"
	}
}

// ReadStatusIndicator renders the read/unread indicator with color.
func ReadStatusIndicator(isRead bool, isSelected bool) string {
	symbol := "●"
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(ansiColorNumber(colors.Red)))
	if isRead {
		symbol = "○"
		style = style.Foreground(lipgloss.Color("241"))
	}
	if isSelected {
		style = style.Background(lipgloss.Color(ansiColorNumber(colors.Blue))).Bold(true)
	}
	return style.Width(readStatusWidth).Align(lipgloss.Left).Render(symbol)
}

func calculateAge(timestamp string, now time.Time) string {
	if timestamp == "" {
		return ""
	}

	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return ""
	}

	if now.IsZero() {
		now = time.Now()
	}

	duration := now.Sub(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd", int(duration.Hours()/24))
}

// ansiColorNumber extracts the color number from an ANSI escape sequence.
// Example: "\033[0;34m" -> "34"
func ansiColorNumber(ansi string) string {
	if len(ansi) < 2 {
		return ""
	}
	lastSemicolon := strings.LastIndex(ansi, ";")
	if lastSemicolon == -1 {
		return ""
	}
	return ansi[lastSemicolon+1 : len(ansi)-1]
}
