package render

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

const (
	typeWidth            = 8
	statusWidth          = 8
	sessionWidth         = 25
	paneWidth            = 7
	ageWidth             = 5
	spacesBetweenColumns = 10
	defaultMessageWidth  = 50
	groupIndentSize      = 2
	groupCollapsedSymbol = "▸"
	groupExpandedSymbol  = "▾"
)

// FooterState defines the inputs needed to render footer help text.
type FooterState struct {
	SearchMode   bool
	CommandMode  bool
	SearchQuery  string
	CommandQuery string
	Grouped      bool
}

// RowState defines the inputs needed to render a notification row.
type RowState struct {
	Notification notification.Notification
	SessionName  string
	Width        int
	Selected     bool
	Now          time.Time
}

// GroupNode defines the inputs needed to render a grouped tree node.
type GroupNode struct {
	Title    string
	Display  string
	Expanded bool
	Count    int
}

// GroupRow defines the inputs needed to render a group row.
type GroupRow struct {
	Node     *GroupNode
	Selected bool
	Level    int
	Width    int
	Styles   *GroupRowStyles
}

// GroupRowStyles defines styles for group rows.
type GroupRowStyles struct {
	Base     lipgloss.Style
	Selected lipgloss.Style
}

// Header renders the table header.
func Header(width int) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ansiColorNumber(colors.Blue)))

	messageWidth := calculateMessageWidth(width)

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

// Row renders a single notification row.
func Row(state RowState) string {
	rowStyle := lipgloss.NewStyle()
	if state.Selected {
		rowStyle = rowStyle.Background(lipgloss.Color(ansiColorNumber(colors.Blue))).Foreground(lipgloss.Color("0"))
	}

	levelIcon := levelIcon(state.Notification.Level)
	statusIcon := statusIcon(state.Notification.State)

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

// RenderGroupRow renders a single group row.
func RenderGroupRow(row GroupRow) string {
	if row.Node == nil {
		return ""
	}

	styles := row.Styles
	if styles == nil {
		defaultStyles := defaultGroupRowStyles()
		styles = &defaultStyles
	}

	indent := strings.Repeat(" ", groupIndentSize*row.Level)
	symbol := groupCollapsedSymbol
	if row.Node.Expanded {
		symbol = groupExpandedSymbol
	}

	title := row.Node.Display
	if title == "" {
		title = row.Node.Title
	}

	label := fmt.Sprintf("%s%s %s (%d)", indent, symbol, title, row.Node.Count)
	label = truncateGroupRow(label, row.Width)

	if row.Selected {
		return styles.Selected.Render(label)
	}
	return styles.Base.Render(label)
}

// Footer renders the footer with help text.
func Footer(state FooterState) string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var help []string
	help = append(help, "j/k: move")
	if state.SearchMode {
		help = append(help, "ESC: exit search")
		help = append(help, fmt.Sprintf("Search: %s", state.SearchQuery))
	} else if state.CommandMode {
		help = append(help, "ESC: cancel")
		help = append(help, fmt.Sprintf(":%s", state.CommandQuery))
	} else {
		help = append(help, "/: search")
		help = append(help, ":: command")
		if state.Grouped {
			help = append(help, "h/l: collapse/expand")
			help = append(help, "za: toggle fold")
		}
	}
	help = append(help, "d: dismiss")
	enterHelp := "Enter: jump"
	if state.Grouped {
		enterHelp = "Enter: toggle/jump"
	}
	if state.CommandMode {
		enterHelp = "Enter: execute"
	}
	help = append(help, enterHelp)
	help = append(help, "q: quit")
	help = append(help, ":w: save")

	return helpStyle.Render(strings.Join(help, "  |  "))
}

func calculateMessageWidth(width int) int {
	totalFixedWidth := typeWidth + statusWidth + sessionWidth + paneWidth + ageWidth
	return width - totalFixedWidth - spacesBetweenColumns
}

func defaultGroupRowStyles() GroupRowStyles {
	base := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ansiColorNumber(colors.Blue)))
	selected := lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color(ansiColorNumber(colors.Blue))).
		Foreground(lipgloss.Color("0"))
	return GroupRowStyles{
		Base:     base,
		Selected: selected,
	}
}

func truncateGroupRow(value string, width int) string {
	if width <= 0 {
		return value
	}
	if utf8.RuneCountInString(value) <= width {
		return value
	}
	return string([]rune(value)[:width])
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
