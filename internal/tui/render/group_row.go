package render

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
)

// GroupNode defines the inputs needed to render a grouped tree node.
type GroupNode struct {
	Title       string
	Display     string
	Expanded    bool
	Count       int
	UnreadCount int
}

// GroupRow defines the inputs needed to render a group row.
type GroupRow struct {
	Node     *GroupNode
	Selected bool
	Level    int
	Width    int
	Styles   *GroupRowStyles
	Now      time.Time
	// EarliestTimestamp and LatestTimestamp are RFC3339 times.
	EarliestTimestamp string
	LatestTimestamp   string
	LevelCounts       map[string]int
	Sources           []string
	Options           settings.GroupHeaderOptions
}

// GroupRowStyles defines styles for group rows.
type GroupRowStyles struct {
	Base     lipgloss.Style
	Selected lipgloss.Style
}

type groupRowSegment struct {
	text  string
	style *lipgloss.Style
}

func (s groupRowSegment) width() int {
	return utf8.RuneCountInString(s.text)
}

// RenderGroupRow renders a single group row.
func RenderGroupRow(row GroupRow) string {
	if row.Node == nil {
		return ""
	}

	styles := ensureGroupRowStyles(row.Styles)
	options := resolveGroupRowOptions(row.Options)

	segments := buildGroupRowSegments(row, options)
	segments = clampGroupRowSegments(segments, row.Width)
	plain := plainTextFromSegments(segments)
	if row.Selected {
		return styles.Selected.Render(plain)
	}
	return renderSegments(segments, groupBaseStyle(row, styles))
}

func ensureGroupRowStyles(styles *GroupRowStyles) *GroupRowStyles {
	if styles != nil {
		return styles
	}
	defaults := defaultGroupRowStyles()
	return &defaults
}

func buildGroupRowSegments(row GroupRow, options settings.GroupHeaderOptions) []groupRowSegment {
	segments := []groupRowSegment{buildGroupTitleSegment(row)}
	segments = appendTimeRangeSegment(segments, row, options)
	segments = appendBadgeSegments(segments, row, options)
	return appendSourceSegment(segments, row.Sources, options)
}

func buildGroupTitleSegment(row GroupRow) groupRowSegment {
	indent := strings.Repeat(" ", groupIndentSize*row.Level)
	symbol := groupCollapsedSymbol
	if row.Node.Expanded {
		symbol = groupExpandedSymbol
	}
	title := resolveGroupTitle(row.Node)
	countLabel := formatGroupCount(row.Node.Count, row.Node.UnreadCount)
	return groupRowSegment{text: fmt.Sprintf("%s%s %s (%s)", indent, symbol, title, countLabel)}
}

func resolveGroupTitle(node *GroupNode) string {
	if node == nil {
		return ""
	}
	if node.Display != "" {
		return node.Display
	}
	return node.Title
}

func formatGroupCount(total, unread int) string {
	if unread > 0 {
		return fmt.Sprintf("%d/%d", total, unread)
	}
	return fmt.Sprintf("%d", total)
}

func appendTimeRangeSegment(segments []groupRowSegment, row GroupRow, options settings.GroupHeaderOptions) []groupRowSegment {
	if timeRange := buildTimeRangeLabel(row, options); timeRange != "" {
		return appendSegmentWithGap(segments, groupRowSegment{text: timeRange}, "  ")
	}
	return segments
}

func appendBadgeSegments(segments []groupRowSegment, row GroupRow, options settings.GroupHeaderOptions) []groupRowSegment {
	badges := buildBadgeSegments(row, options)
	if len(badges) == 0 {
		return segments
	}
	if len(segments) > 0 {
		segments = append(segments, groupRowSegment{text: "  "})
	}
	for idx, badge := range badges {
		if idx > 0 {
			segments = append(segments, groupRowSegment{text: " "})
		}
		segments = append(segments, badge)
	}
	return segments
}

func appendSourceSegment(segments []groupRowSegment, sources []string, options settings.GroupHeaderOptions) []groupRowSegment {
	if sourceLabel := buildSourceLabel(sources, options); sourceLabel != "" {
		return appendSegmentWithGap(segments, groupRowSegment{text: sourceLabel}, "  ")
	}
	return segments
}

func appendSegmentWithGap(segments []groupRowSegment, addition groupRowSegment, gap string) []groupRowSegment {
	if addition.text == "" {
		return segments
	}
	if gap != "" && len(segments) > 0 {
		segments = append(segments, groupRowSegment{text: gap})
	}
	return append(segments, addition)
}

func groupBaseStyle(row GroupRow, styles *GroupRowStyles) lipgloss.Style {
	if row.Node != nil && row.Node.UnreadCount > 0 {
		unreadStyles := stylesWithUnread()
		return unreadStyles.Base
	}
	return styles.Base
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

func stylesWithUnread() GroupRowStyles {
	base := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ansiColorNumber(colors.Yellow)))
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

func resolveGroupRowOptions(options settings.GroupHeaderOptions) settings.GroupHeaderOptions {
	if options.BadgeColors == nil && !options.ShowTimeRange && !options.ShowLevelBadges && !options.ShowSourceAggregation {
		return settings.DefaultGroupHeaderOptions()
	}
	return options.Clone()
}

func buildTimeRangeLabel(row GroupRow, options settings.GroupHeaderOptions) string {
	if !options.ShowTimeRange {
		return ""
	}
	earliest := calculateAge(row.EarliestTimestamp, row.Now)
	latest := calculateAge(row.LatestTimestamp, row.Now)
	if earliest == "" && latest == "" {
		return ""
	}
	if earliest == "" {
		return latest
	}
	if latest == "" || earliest == latest {
		return earliest
	}
	return fmt.Sprintf("%s – %s", earliest, latest)
}

var severityDisplayOrder = []string{settings.LevelFilterCritical, settings.LevelFilterError, settings.LevelFilterWarning, settings.LevelFilterInfo}

func buildBadgeSegments(row GroupRow, options settings.GroupHeaderOptions) []groupRowSegment {
	if !options.ShowLevelBadges || len(row.LevelCounts) == 0 {
		return nil
	}
	segments := make([]groupRowSegment, 0, len(row.LevelCounts))
	for _, level := range severityDisplayOrder {
		count := row.LevelCounts[level]
		if count == 0 {
			continue
		}
		style := lipgloss.NewStyle().Bold(true)
		if color := options.BadgeColors[level]; color != "" {
			style = style.Foreground(lipgloss.Color(ansiColorNumber(color)))
		}
		segments = append(segments, groupRowSegment{
			text:  fmt.Sprintf("%s%d", badgeIconForLevel(level), count),
			style: &style,
		})
	}
	return segments
}

func buildSourceLabel(sources []string, options settings.GroupHeaderOptions) string {
	if !options.ShowSourceAggregation || len(sources) == 0 {
		return ""
	}
	return fmt.Sprintf("src: %s", strings.Join(sources, ","))
}

func badgeIconForLevel(level string) string {
	switch level {
	case settings.LevelFilterCritical:
		return "‼"
	case settings.LevelFilterError:
		return "❌"
	case settings.LevelFilterWarning:
		return "⚠"
	default:
		return "ℹ"
	}
}

func clampGroupRowSegments(segments []groupRowSegment, width int) []groupRowSegment {
	if width <= 0 {
		return segments
	}
	remaining := width
	result := make([]groupRowSegment, 0, len(segments))
	for _, segment := range segments {
		segWidth := segment.width()
		if segWidth == 0 {
			result = append(result, segment)
			continue
		}
		if remaining <= 0 {
			break
		}
		if segWidth <= remaining {
			result = append(result, segment)
			remaining -= segWidth
			continue
		}
		runes := []rune(segment.text)
		if len(runes) > remaining {
			runes = runes[:remaining]
		}
		result = append(result, groupRowSegment{text: string(runes), style: segment.style})
		break
	}
	return result
}

func plainTextFromSegments(segments []groupRowSegment) string {
	var builder strings.Builder
	for _, segment := range segments {
		builder.WriteString(segment.text)
	}
	return builder.String()
}

func renderSegments(segments []groupRowSegment, base lipgloss.Style) string {
	var builder strings.Builder
	for _, segment := range segments {
		if segment.style != nil {
			builder.WriteString(segment.style.Render(segment.text))
			continue
		}
		builder.WriteString(base.Render(segment.text))
	}
	return builder.String()
}
