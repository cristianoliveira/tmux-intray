package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
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
		_, _ = fmt.Fprintf(w, "%sNo notifications found%s\n", colors.Blue, colors.Reset)
		return
	}

	// Group by session and get most recent per session
	sessionGroups := groupBySession(notifications)

	if len(sessionGroups) == 0 {
		_, _ = fmt.Fprintf(w, "%sNo sessions with notifications found%s\n", colors.Blue, colors.Reset)
		return
	}

	switch opts.Format {
	case "table":
		printTabsTable(sessionGroups, w)
	case "json":
		printTabsJSON(sessionGroups, w)
	default:
		printTabsSimple(sessionGroups, w)
	}
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

// resolveSessionName resolves a session ID to its display name using tmux.
// Returns the resolved name, or the original ID if resolution fails.
func resolveSessionName(sessionID string, sessionNames map[string]string) string {
	if sessionID == "" {
		return ""
	}
	if name, ok := sessionNames[sessionID]; ok && name != "" {
		return name
	}
	return sessionID
}

// getSessionNamesForTabs returns a map of session IDs to names from tmux.
func getSessionNamesForTabs() map[string]string {
	client := tmux.NewDefaultClient()
	sessionNames, err := client.ListSessions()
	if err != nil || sessionNames == nil {
		return make(map[string]string)
	}
	return sessionNames
}

// printTabsSimple prints sessions in simple format.
func printTabsSimple(groups []domain.SessionNotification, w io.Writer) {
	sessionNames := getSessionNamesForTabs()

	header := fmt.Sprintf("%sSessions (%d)%s\n", colors.Bold, len(groups), colors.Reset)
	_, _ = fmt.Fprint(w, header)
	_, _ = fmt.Fprint(w, strings.Repeat("─", 60)+"\n")

	for i, sg := range groups {
		num := i + 1
		sessionDisplay := resolveSessionName(sg.Session, sessionNames)
		if sg.Notification.Session != "" {
			sessionDisplay = resolveSessionName(sg.Notification.Session, sessionNames)
		}

		level := string(sg.Notification.Level)
		levelColor := levelColorCode(level)

		_, _ = fmt.Fprintf(w, "%s%d.%s %s%s%s %s\n",
			colors.Bold, num, colors.Reset,
			colors.Yellow, sessionDisplay, colors.Reset,
			formatAge(sg.Notification.Timestamp),
		)
		_, _ = fmt.Fprintf(w, "   %s[%s]%s %s\n",
			levelColor, level, colors.Reset,
			truncateMessage(sg.Notification.Message, 50),
		)
		if i < len(groups)-1 {
			_, _ = fmt.Fprint(w, "\n")
		}
	}
}

// printTabsTable prints sessions in table format.
func printTabsTable(groups []domain.SessionNotification, w io.Writer) {
	sessionNames := getSessionNamesForTabs()

	header := fmt.Sprintf("%sSessions (%d)%s\n", colors.Bold, len(groups), colors.Reset)
	_, _ = fmt.Fprint(w, header)
	_, _ = fmt.Fprint(w, strings.Repeat("─", 80)+"\n")
	_, _ = fmt.Fprintf(w, "%-4s %-20s %-8s %-10s %s\n",
		colors.Bold+"Num"+colors.Reset,
		colors.Bold+"Session"+colors.Reset,
		colors.Bold+"Level"+colors.Reset,
		colors.Bold+"Age"+colors.Reset,
		colors.Bold+"Message"+colors.Reset,
	)
	_, _ = fmt.Fprint(w, strings.Repeat("─", 80)+"\n")

	for i, sg := range groups {
		num := i + 1
		sessionDisplay := resolveSessionName(sg.Session, sessionNames)
		if len(sessionDisplay) > 18 {
			sessionDisplay = sessionDisplay[:15] + "..."
		}

		level := string(sg.Notification.Level)
		levelColor := levelColorCode(level)

		age := formatAge(sg.Notification.Timestamp)
		msg := truncateMessage(sg.Notification.Message, 30)

		_, _ = fmt.Fprintf(w, "%-4d %-20s %s%-8s%s %-10s %s\n",
			num,
			sessionDisplay,
			levelColor, level, colors.Reset,
			age,
			msg,
		)
	}
}

// tabSessionJSON represents a session in JSON output for tabs.
type tabSessionJSON struct {
	Num       int    `json:"num"`
	Session   string `json:"session"`
	Level     string `json:"level"`
	Timestamp string `json:"timestamp"`
	Age       string `json:"age"`
	Message   string `json:"message"`
	Window    string `json:"window,omitempty"`
	Pane      string `json:"pane,omitempty"`
	Unread    bool   `json:"unread"`
	SessionID string `json:"session_id,omitempty"` // Raw session ID for debugging
}

// printTabsJSON prints sessions in JSON format.
func printTabsJSON(groups []domain.SessionNotification, w io.Writer) {
	sessionNames := getSessionNamesForTabs()

	sessions := make([]tabSessionJSON, 0, len(groups))
	for i, sg := range groups {
		sessions = append(sessions, tabSessionJSON{
			Num:       i + 1,
			Session:   resolveSessionName(sg.Session, sessionNames),
			Level:     string(sg.Notification.Level),
			Timestamp: sg.Notification.Timestamp,
			Age:       formatAge(sg.Notification.Timestamp),
			Message:   sg.Notification.Message,
			Window:    sg.Notification.Window,
			Pane:      sg.Notification.Pane,
			Unread:    !sg.Notification.IsRead(),
			SessionID: sg.Session, // Include raw session ID for debugging
		})
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(sessions); err != nil {
		_, _ = fmt.Fprintf(w, "tabs: failed to encode JSON: %v\n", err)
	}
}

// levelColorCode returns ANSI color code for notification level.
func levelColorCode(level string) string {
	switch level {
	case "error":
		return colors.Red
	case "warning":
		return colors.Yellow
	case "critical":
		return colors.Bold + colors.Red
	default:
		return colors.Reset
	}
}

// formatAge formats a timestamp as relative age (e.g., "2h").
func formatAge(timestamp string) string {
	if timestamp == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}

	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd", int(duration.Hours()/24))
}

// truncateMessage truncates a message to maxLen characters.
func truncateMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen-3] + "..."
}
