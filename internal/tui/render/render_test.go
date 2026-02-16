package render

import (
	"regexp"
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
	options := disabledGroupHeaderOptions()

	row := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:    "session-one",
			Display:  "session-one",
			Expanded: true,
			Count:    3,
		},
		Level:   1,
		Width:   80,
		Styles:  &styles,
		Options: options,
	})

	assert.True(t, strings.HasPrefix(row, "  ▾ session-one (3)"))

	row = RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:    "win-1",
			Expanded: false,
			Count:    2,
		},
		Level:   2,
		Width:   80,
		Styles:  &styles,
		Options: options,
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
		Level:   0,
		Width:   10,
		Styles:  &styles,
		Options: disabledGroupHeaderOptions(),
	})

	assert.Equal(t, 10, utf8.RuneCountInString(row))
}

func TestRenderGroupRowDisplaysTimeRange(t *testing.T) {
	styles := GroupRowStyles{Base: lipgloss.NewStyle(), Selected: lipgloss.NewStyle()}
	fixedNow := time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC)
	row := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:    "session",
			Expanded: true,
			Count:    2,
		},
		Level:             0,
		Width:             80,
		Styles:            &styles,
		Now:               fixedNow,
		EarliestTimestamp: "2025-01-02T08:00:00Z",
		LatestTimestamp:   "2025-01-02T11:00:00Z",
	})
	assert.Contains(t, stripANSI(row), "4h – 1h")
}

func TestRenderGroupRowDisplaysBadges(t *testing.T) {
	styles := GroupRowStyles{Base: lipgloss.NewStyle(), Selected: lipgloss.NewStyle()}
	row := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:    "message",
			Expanded: true,
			Count:    5,
		},
		Level:       0,
		Width:       120,
		Styles:      &styles,
		LevelCounts: map[string]int{"warning": 2, "info": 3},
	})
	clean := stripANSI(row)
	assert.Contains(t, clean, "⚠2")
	assert.Contains(t, clean, "ℹ3")
}

func TestRenderGroupRowDisplaysSources(t *testing.T) {
	styles := GroupRowStyles{Base: lipgloss.NewStyle(), Selected: lipgloss.NewStyle()}
	options := settings.DefaultGroupHeaderOptions()
	options.ShowSourceAggregation = true
	row := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:    "group",
			Expanded: true,
			Count:    1,
		},
		Level:   0,
		Width:   120,
		Styles:  &styles,
		Sources: []string{"pane1", "pane2"},
		Options: options,
	})
	assert.Contains(t, stripANSI(row), "src: pane1,pane2")
}

func TestFooterGroupedHelpText(t *testing.T) {
	footer := Footer(FooterState{Grouped: true, ViewMode: settings.ViewModeGrouped, ShowHelp: true})

	assert.Contains(t, footer, "mode: [G]")
	assert.Contains(t, footer, "read: all")
	assert.Contains(t, footer, "v: cycle view mode")
	assert.Contains(t, footer, "gg/G: top/bottom")
	assert.Contains(t, footer, "h/l: collapse/expand")
	assert.Contains(t, footer, "za: toggle fold")
	assert.Contains(t, footer, "Enter: toggle/jump")
}

func TestFooterSearchModeHelpText(t *testing.T) {
	footer := Footer(FooterState{SearchMode: true, SearchQuery: "test", ViewMode: settings.ViewModeDetailed, ShowHelp: true})

	assert.Contains(t, footer, "mode: [D]")
	assert.Contains(t, footer, "read: all")
	assert.Contains(t, footer, "ESC: exit search")
	assert.Contains(t, footer, "Ctrl+j/k: navigate")
	assert.Contains(t, footer, "Search: test")
}

func TestFooterSearchModeWithoutHelp(t *testing.T) {
	footer := Footer(FooterState{SearchMode: true, SearchQuery: "test", ViewMode: settings.ViewModeDetailed, ShowHelp: false})

	// Should contain search query and search help
	assert.Contains(t, footer, "Search: test")
	assert.Contains(t, footer, "ESC: exit search")
	assert.Contains(t, footer, "Ctrl+j/k: navigate")
	// Should contain mode indicator
	assert.Contains(t, footer, "mode: [D]")
	// Should NOT contain regular help items
	assert.NotContains(t, footer, "read:")
	assert.NotContains(t, footer, "j/k: move")
	assert.NotContains(t, footer, "gg/G:")
	assert.NotContains(t, footer, "r: read")
	assert.NotContains(t, footer, "u: unread")
	assert.NotContains(t, footer, "d: dismiss")
	assert.NotContains(t, footer, "Enter:")
	assert.NotContains(t, footer, "q: quit")
	assert.Contains(t, footer, "?: toggle help")

}

func TestFooterReadFilterIndicator(t *testing.T) {
	footer := Footer(FooterState{ViewMode: settings.ViewModeGrouped, ReadFilter: settings.ReadFilterUnread, ShowHelp: true})
	assert.Contains(t, footer, "read: unread")

	footer = Footer(FooterState{ViewMode: settings.ViewModeGrouped, ReadFilter: settings.ReadFilterRead, ShowHelp: true})
	assert.Contains(t, footer, "read: read")
}

func TestFooterClampsToWidthAndClearsLine(t *testing.T) {
	footer := Footer(FooterState{Grouped: true, ViewMode: settings.ViewModeGrouped, Width: 24, ShowHelp: true})
	assert.Equal(t, 27, len(footer))
	assert.True(t, strings.HasSuffix(footer, "\x1b[K"))
}

func TestFooterMinimalHelp(t *testing.T) {
	footer := Footer(FooterState{ViewMode: settings.ViewModeCompact, ShowHelp: false})
	assert.Contains(t, footer, "mode: [C]")
	assert.Contains(t, footer, "j/k: move")
	assert.Contains(t, footer, "?: toggle help")
	// Should not contain other help items
	assert.NotContains(t, footer, "read:")
	assert.NotContains(t, footer, "gg/G:")
	assert.NotContains(t, footer, "v:")
	assert.NotContains(t, footer, "r:")
	assert.NotContains(t, footer, "u:")
	assert.NotContains(t, footer, "d:")
	assert.NotContains(t, footer, "Enter:")
	assert.NotContains(t, footer, "q:")
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
	options := disabledGroupHeaderOptions()

	// Test group with no unread items (should show only total)
	row := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:       "session-one",
			Display:     "session-one",
			Expanded:    true,
			Count:       5,
			UnreadCount: 0,
		},
		Level:   0,
		Width:   80,
		Styles:  &styles,
		Options: options,
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
		Level:   1,
		Width:   80,
		Styles:  &styles,
		Options: options,
	})

	assert.Contains(t, row, "session-two (10/3)")
}

func TestRenderGroupRowWithUnreadHighlighting(t *testing.T) {
	styles := GroupRowStyles{
		Base:     lipgloss.NewStyle(),
		Selected: lipgloss.NewStyle(),
	}
	options := disabledGroupHeaderOptions()

	// Test that groups with unread items use different styling
	rowWithUnread := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:       "session-with-unread",
			Display:     "session-with-unread",
			Expanded:    true,
			Count:       5,
			UnreadCount: 2,
		},
		Level:   0,
		Width:   80,
		Styles:  &styles,
		Options: options,
	})

	rowAllRead := RenderGroupRow(GroupRow{
		Node: &GroupNode{
			Title:       "session-all-read",
			Display:     "session-all-read",
			Expanded:    true,
			Count:       5,
			UnreadCount: 0,
		},
		Level:   0,
		Width:   80,
		Styles:  &styles,
		Options: options,
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

func disabledGroupHeaderOptions() settings.GroupHeaderOptions {
	options := settings.DefaultGroupHeaderOptions()
	options.ShowTimeRange = false
	options.ShowLevelBadges = false
	options.ShowSourceAggregation = false
	return options
}

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(input string) string {
	return ansiRegexp.ReplaceAllString(input, "")
}
