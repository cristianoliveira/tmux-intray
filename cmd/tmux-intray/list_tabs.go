package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// TabsOptions holds options for the tabs command.
type TabsOptions struct {
	Client     listClient
	All        bool
	Format     string // "simple", "table", or "json"
	Session    string
	Level      string
	Window     string
	Pane       string
	OlderThan  string
	NewerThan  string
	ReadFilter string
}

// tabsOutputWriter is the writer used by PrintTabs. Can be changed for testing.
var tabsOutputWriter io.Writer = os.Stdout

// PrintTabs prints sessions with their most recent notification.
func PrintTabs(opts TabsOptions) {
	if tabsOutputWriter == nil {
		tabsOutputWriter = os.Stdout
	}
	printTabs(opts, tabsOutputWriter)
}

func printTabs(opts TabsOptions, w io.Writer) {
	state := "active"
	if opts.All {
		state = "all"
	}

	lines, err := opts.Client.ListNotifications(
		state,
		opts.Level,
		opts.Session,
		opts.Window,
		opts.Pane,
		opts.OlderThan,
		opts.NewerThan,
		opts.ReadFilter,
	)
	if err != nil {
		_, _ = fmt.Fprintf(w, "tabs: failed to list notifications: %v\n", err)
		return
	}

	notifications := parseTabsNotifications(lines)
	if len(notifications) == 0 {
		_, _ = fmt.Fprintln(w, "No notifications found")
		return
	}

	// Group by session and get most recent per session
	sessionGroups := groupBySession(notifications)

	if len(sessionGroups) == 0 {
		_, _ = fmt.Fprintln(w, "No sessions with notifications found")
		return
	}

	// Render using the same formatter implementation as `tmux-intray list`.
	switch opts.Format {
	case "json":
		formatTabsUsingListFormatter(sessionGroups, format.FormatterTypeJSON, w)
	case "table":
		formatTabsUsingListFormatter(sessionGroups, format.FormatterTypeTable, w)
	case "legacy":
		formatTabsUsingListFormatter(sessionGroups, format.FormatterTypeLegacy, w)
	case "compact":
		formatTabsUsingListFormatter(sessionGroups, format.FormatterTypeCompact, w)
	default:
		formatTabsUsingListFormatter(sessionGroups, format.FormatterTypeSimple, w)
	}
}

func formatTabsUsingListFormatter(groups []domain.SessionNotification, ftype format.FormatterType, w io.Writer) {
	formatter := format.NewFormatter(ftype)

	notifs := make([]*domain.Notification, 0, len(groups))
	for i := range groups {
		n := groups[i].Notification
		notifs = append(notifs, &n)
	}

	_ = formatter.FormatNotifications(notifs, w)
}

// parseTabsNotifications parses notification lines.
func parseTabsNotifications(lines string) []notification.Notification {
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
	return notifications
}

// groupBySession groups notifications by session, keeping only the most recent.
func groupBySession(notifications []notification.Notification) []domain.SessionNotification {
	// Convert to domain notifications
	domainNotifs := notificationsToDomain(notifs(notifications))
	return domain.GroupBySessionKeepMostRecent(domainNotifs)
}

// notifs converts []notification.Notification to []domain.Notification.
func notifs(n []notification.Notification) []*domain.Notification {
	result := make([]*domain.Notification, len(n))
	for i := range n {
		result[i] = domainNotificationToPointer(&n[i])
	}
	return result
}

// notificationsToDomain converts notification.Notification to domain.Notification.
func notificationsToDomain(n []*domain.Notification) []domain.Notification {
	result := make([]domain.Notification, len(n))
	for i := range n {
		result[i] = *n[i]
	}
	return result
}

// domainNotificationToPointer converts notification.Notification to *domain.Notification.
func domainNotificationToPointer(n *notification.Notification) *domain.Notification {
	level := domain.NotificationLevel(n.Level)
	if n.Level == "" {
		level = domain.LevelInfo
	}
	state := domain.NotificationState(n.State)
	if n.State == "" {
		state = domain.StateActive
	}
	return &domain.Notification{
		ID:            n.ID,
		Timestamp:     n.Timestamp,
		State:         state,
		Session:       n.Session,
		Window:        n.Window,
		Pane:          n.Pane,
		Message:       n.Message,
		PaneCreated:   n.PaneCreated,
		Level:         level,
		ReadTimestamp: n.ReadTimestamp,
	}
}

// NOTE: sessions output formatting is intentionally shared with `tmux-intray list`
// via internal/format. Older custom renderers were removed to keep output consistent.
