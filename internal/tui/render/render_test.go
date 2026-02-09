package render

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/stretchr/testify/assert"
)

func TestLevelIcon(t *testing.T) {
	tests := []struct {
		level    string
		expected string
	}{
		{"error", "❌ err"},
		{"warning", "⚠️ wrn"},
		{"critical", "‼️ crt"},
		{"info", "ℹ️ inf"},
		{"", "ℹ️ inf"},
		{"notice", "ℹ️ not"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := levelIcon(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{"active", "●"},
		{"", "●"},
		{"dismissed", "○"},
		{"paused", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := statusIcon(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadStatusIndicator(t *testing.T) {
	expectedUnread := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ansiColorNumber(colors.Red))).
		Width(readStatusWidth).
		Align(lipgloss.Left).
		Render("●")
	assert.Equal(t, expectedUnread, ReadStatusIndicator(false, false))

	expectedRead := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(readStatusWidth).
		Align(lipgloss.Left).
		Render("○")
	assert.Equal(t, expectedRead, ReadStatusIndicator(true, false))

	expectedUnreadSelected := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ansiColorNumber(colors.Red))).
		Background(lipgloss.Color(ansiColorNumber(colors.Blue))).
		Bold(true).
		Width(readStatusWidth).
		Align(lipgloss.Left).
		Render("●")
	assert.Equal(t, expectedUnreadSelected, ReadStatusIndicator(false, true))
	expectedReadSelected := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color(ansiColorNumber(colors.Blue))).
		Bold(true).
		Width(readStatusWidth).
		Align(lipgloss.Left).
		Render("○")
	assert.Equal(t, expectedReadSelected, ReadStatusIndicator(true, true))
}

func TestCalculateAge(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 30, 0, time.UTC)

	assert.Equal(t, "30s", calculateAge("2024-01-01T12:00:00Z", now))
	assert.Equal(t, "", calculateAge("", now))
	assert.Equal(t, "", calculateAge("invalid", now))
}

func TestRowSessionAndPaneColumns(t *testing.T) {
	row := Row(RowState{
		Notification: notification.Notification{
			ID:        1,
			Session:   "$1",
			Window:    "@2",
			Pane:      "%3",
			Message:   "Test message",
			Timestamp: "2024-01-01T12:00:00Z",
			Level:     "info",
			State:     "active",
		},
		SessionName: "main-session",
		Width:       100,
		Selected:    false,
		Now:         time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})

	assert.True(t, strings.HasPrefix(row, ReadStatusIndicator(false, false)))
	assert.True(t, strings.Contains(row, "main-session"))
	assert.True(t, strings.Contains(row, "%3"))
	assert.False(t, strings.Contains(row, "@2:%3"))
}

func TestRenderGroupRowIndentationAndSymbol(t *testing.T) {
	styles := GroupRowStyles{
		Base:     lipgloss.NewStyle(),
		Selected: lipgloss.NewStyle(),
	}

	row := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:    "session-one",
			Display:  "session-one",
			Expanded: true,
			Count:    3,
		},
		Level:  1,
		Width:  80,
		Styles: &styles,
	})

	assert.True(t, strings.HasPrefix(row, "  ▾ session-one (3)"))

	row = RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:    "win-1",
			Expanded: false,
			Count:    2,
		},
		Level:  2,
		Width:  80,
		Styles: &styles,
	})

	assert.True(t, strings.HasPrefix(row, "    ▸ win-1 (2)"))
}

func TestRenderGroupRowTruncatesToWidth(t *testing.T) {
	styles := GroupRowStyles{
		Base:     lipgloss.NewStyle(),
		Selected: lipgloss.NewStyle(),
	}

	row := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:    "session-long-title",
			Display:  "session-long-title",
			Expanded: true,
			Count:    12,
		},
		Level:  0,
		Width:  10,
		Styles: &styles,
	})

	assert.Equal(t, 10, utf8.RuneCountInString(row))
}

func TestFooterGroupedHelpText(t *testing.T) {
	footer := Footer(FooterState{Grouped: true, ViewMode: settings.ViewModeGrouped})

	assert.Contains(t, footer, "mode: [G]")
	assert.Contains(t, footer, "v: cycle view mode")
	assert.Contains(t, footer, "h/l: collapse/expand")
	assert.Contains(t, footer, "za: toggle fold")
	assert.Contains(t, footer, "Enter: toggle/jump")
}

func TestFooterCommandModeHelpText(t *testing.T) {
	footer := Footer(FooterState{CommandMode: true, CommandQuery: "group-by window", ViewMode: settings.ViewModeGrouped})

	assert.Contains(t, footer, "ESC: cancel")
	assert.Contains(t, footer, "cmds: "+commandList)
	assert.Contains(t, footer, ":group-by window")
	assert.Contains(t, footer, "Enter: execute")
}

func TestViewModeIndicator(t *testing.T) {
	assert.Equal(t, "[C]", viewModeIndicator(settings.ViewModeCompact))
	assert.Equal(t, "[D]", viewModeIndicator(settings.ViewModeDetailed))
	assert.Equal(t, "[G]", viewModeIndicator(settings.ViewModeGrouped))
	assert.Equal(t, "[?]", viewModeIndicator("unknown"))
}

func TestRenderGroupRowWithUnreadCounts(t *testing.T) {
	styles := GroupRowStyles{
		Base:     lipgloss.NewStyle(),
		Selected: lipgloss.NewStyle(),
	}

	// Test group with no unread items (should show only total)
	row := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:       "session-one",
			Display:     "session-one",
			Expanded:    true,
			Count:       5,
			UnreadCount: 0,
		},
		Level:  0,
		Width:  80,
		Styles: &styles,
	})

	assert.Contains(t, row, "session-one (5)")
	assert.NotContains(t, row, "session-one (5/0)")

	// Test group with unread items (should show total/unread)
	row = RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:       "session-two",
			Display:     "session-two",
			Expanded:    false,
			Count:       10,
			UnreadCount: 3,
		},
		Level:  1,
		Width:  80,
		Styles: &styles,
	})

	assert.Contains(t, row, "session-two (10/3)")
}

func TestRenderGroupRowWithUnreadHighlighting(t *testing.T) {
	styles := GroupRowStyles{
		Base:     lipgloss.NewStyle(),
		Selected: lipgloss.NewStyle(),
	}

	// Test that groups with unread items use different styling
	rowWithUnread := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:       "session-with-unread",
			Display:     "session-with-unread",
			Expanded:    true,
			Count:       5,
			UnreadCount: 2,
		},
		Level:  0,
		Width:  80,
		Styles: &styles,
	})

	rowAllRead := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:       "session-all-read",
			Display:     "session-all-read",
			Expanded:    true,
			Count:       5,
			UnreadCount: 0,
		},
		Level:  0,
		Width:  80,
		Styles: &styles,
	})

	// Both should render successfully
	assert.NotEmpty(t, rowWithUnread)
	assert.NotEmpty(t, rowAllRead)

	// Both should contain their respective titles
	assert.Contains(t, rowWithUnread, "session-with-unread")
	assert.Contains(t, rowAllRead, "session-all-read")

	// The unread row should show the count format
	assert.Contains(t, rowWithUnread, "(5/2)")

	// The all-read row should show only the total
	assert.Contains(t, rowAllRead, "(5)")
}
