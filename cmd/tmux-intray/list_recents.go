package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// RecentsOptions holds options for the recents command.
type RecentsOptions struct {
	Client     listClient
	Hours      int
	Format     string // "simple" or "table"
	Session    string
	Level      string
	Window     string
	Pane       string
	OlderThan  string
	NewerThan  string
	ReadFilter string
}

// recentsOutputWriter is the writer used by PrintRecents. Can be changed for testing.
var recentsOutputWriter io.Writer = os.Stdout

// PrintRecents prints recent unread notifications.
func PrintRecents(opts RecentsOptions) {
	if recentsOutputWriter == nil {
		recentsOutputWriter = os.Stdout
	}
	printRecents(opts, recentsOutputWriter)
}

func printRecents(opts RecentsOptions, w io.Writer) {
	// Calculate time cutoff (only if not already set)
	cutoffStr := opts.OlderThan
	if cutoffStr == "" && opts.Hours > 0 {
		cutoff := time.Now().UTC().Add(-time.Duration(opts.Hours) * time.Hour)
		cutoffStr = cutoff.Format("2006-01-02T15:04:05Z")
	}

	// Build read filter - recents always wants unread, but allow override
	readFilter := opts.ReadFilter
	if readFilter == "" {
		readFilter = "unread"
	}

	lines, err := opts.Client.ListNotifications(
		"active",
		opts.Level,
		opts.Session,
		opts.Window,
		opts.Pane,
		cutoffStr,
		opts.NewerThan,
		readFilter,
	)
	if err != nil {
		_, _ = fmt.Fprintf(w, "recents: failed to list notifications: %v\n", err)
		return
	}

	notifications := parseTabsNotifications(lines)
	if len(notifications) == 0 {
		_, _ = fmt.Fprintf(w, "%sNo recent unread notifications found%s\n", colors.Blue, colors.Reset)
		return
	}

	// Smart selection: max 1 per session, prioritizing errors/warnings
	sessionBest := selectBestPerSession(notifications)

	// Sort by severity (errors first), then recency
	sort.Slice(sessionBest, func(i, j int) bool {
		sevI := severityWeight(sessionBest[i].Level)
		sevJ := severityWeight(sessionBest[j].Level)
		if sevI != sevJ {
			return sevI > sevJ
		}
		return sessionBest[i].Timestamp > sessionBest[j].Timestamp
	})

	switch opts.Format {
	case "json":
		printRecentsJSON(sessionBest, w)
	case "table":
		printRecentsTable(sessionBest, w)
	default:
		printRecentsSimple(sessionBest, w)
	}
}

// recentsJSON represents a notification in JSON output for recents.
type recentsJSON struct {
	Num       int    `json:"num"`
	ID        int    `json:"id"`
	Session   string `json:"session"`
	Level     string `json:"level"`
	Timestamp string `json:"timestamp"`
	Age       string `json:"age"`
	Message   string `json:"message"`
	Window    string `json:"window,omitempty"`
	Pane      string `json:"pane,omitempty"`
	Unread    bool   `json:"unread"`
}

// printRecentsJSON prints recents in JSON format.
func printRecentsJSON(notifs []notification.Notification, w io.Writer) {
	sessionNames := getSessionNamesForTabs()

	sessions := make([]recentsJSON, 0, len(notifs))
	for i, notif := range notifs {
		sessions = append(sessions, recentsJSON{
			Num:       i + 1,
			ID:        notif.ID,
			Session:   resolveSessionName(notif.Session, sessionNames),
			Level:     notif.Level,
			Timestamp: notif.Timestamp,
			Age:       formatAge(notif.Timestamp),
			Message:   notif.Message,
			Window:    notif.Window,
			Pane:      notif.Pane,
			Unread:    !notif.IsRead(),
		})
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(sessions); err != nil {
		_, _ = fmt.Fprintf(w, "recents: failed to encode JSON: %v\n", err)
	}
}

// selectBestPerSession selects the best notification per session.
func selectBestPerSession(notifications []notification.Notification) []notification.Notification {
	best := make(map[string]notification.Notification)
	for _, notif := range notifications {
		session := notif.Session
		if session == "" {
			session = "__no_session__" // Group notifications without session
		}
		existing, ok := best[session]
		if !ok || isBetterNotification(notif, existing) {
			best[session] = notif
		}
	}

	result := make([]notification.Notification, 0, len(best))
	for _, notif := range best {
		result = append(result, notif)
	}
	return result
}

// isBetterNotification returns true if a is a better notification than b.
func isBetterNotification(a, b notification.Notification) bool {
	sevA := severityWeight(a.Level)
	sevB := severityWeight(b.Level)
	if sevA != sevB {
		return sevA > sevB
	}
	// Same severity, prefer more recent
	return a.Timestamp > b.Timestamp
}

// severityWeight returns a weight for notification level (higher = more severe).
func severityWeight(level string) int {
	switch level {
	case "critical":
		return 4
	case "error":
		return 3
	case "warning":
		return 2
	default:
		return 1
	}
}

// printRecentsSimple prints recents in simple format.
func printRecentsSimple(notifs []notification.Notification, w io.Writer) {
	sessionNames := getSessionNamesForTabs()

	header := fmt.Sprintf("%sRecent Notifications (%d)%s\n", colors.Bold, len(notifs), colors.Reset)
	_, _ = fmt.Fprint(w, header)
	_, _ = fmt.Fprint(w, strings.Repeat("─", 60)+"\n")

	for i, notif := range notifs {
		num := i + 1
		sessionDisplay := resolveSessionName(notif.Session, sessionNames)
		if sessionDisplay == "" {
			sessionDisplay = "(no session)"
		}

		levelColor := levelColorCode(notif.Level)
		age := formatAge(notif.Timestamp)

		_, _ = fmt.Fprintf(w, "%s%d.%s %s%s%s %s #%d\n",
			colors.Bold, num, colors.Reset,
			colors.Yellow, sessionDisplay, colors.Reset,
			age,
			notif.ID,
		)
		_, _ = fmt.Fprintf(w, "   %s[%s]%s %s\n",
			levelColor, notif.Level, colors.Reset,
			truncateMessage(notif.Message, 50),
		)
		if i < len(notifs)-1 {
			_, _ = fmt.Fprint(w, "\n")
		}
	}
}

// printRecentsTable prints recents in table format.
func printRecentsTable(notifs []notification.Notification, w io.Writer) {
	sessionNames := getSessionNamesForTabs()

	header := fmt.Sprintf("%sRecent Notifications (%d)%s\n", colors.Bold, len(notifs), colors.Reset)
	_, _ = fmt.Fprint(w, header)
	_, _ = fmt.Fprint(w, strings.Repeat("─", 80)+"\n")
	_, _ = fmt.Fprintf(w, "%-4s %-7s %-20s %-10s %-8s %s\n",
		colors.Bold+"Num"+colors.Reset,
		colors.Bold+"ID"+colors.Reset,
		colors.Bold+"Session"+colors.Reset,
		colors.Bold+"Age"+colors.Reset,
		colors.Bold+"Level"+colors.Reset,
		colors.Bold+"Message"+colors.Reset,
	)
	_, _ = fmt.Fprint(w, strings.Repeat("─", 80)+"\n")

	for i, notif := range notifs {
		num := i + 1
		sessionDisplay := resolveSessionName(notif.Session, sessionNames)
		if sessionDisplay == "" {
			sessionDisplay = "(no session)"
		}
		if len(sessionDisplay) > 18 {
			sessionDisplay = sessionDisplay[:15] + "..."
		}

		levelColor := levelColorCode(notif.Level)
		age := formatAge(notif.Timestamp)
		msg := truncateMessage(notif.Message, 30)

		_, _ = fmt.Fprintf(w, "%-4d %-7d %-20s %-10s %s%-8s%s %s\n",
			num,
			notif.ID,
			sessionDisplay,
			age,
			levelColor, notif.Level, colors.Reset,
			msg,
		)
	}
}

// TabOptions holds options for the tab flag.
type TabOptions struct {
	Client     listClient
	Tab        string // "recents" or "sessions" or "all"
	Format     string
	Session    string
	Level      string
	Window     string
	Pane       string
	OlderThan  string
	NewerThan  string
	ReadFilter string
}

// PrintTab prints the specified tab view.
func PrintTab(opts TabOptions) {
	switch opts.Tab {
	case "recents":
		PrintRecents(RecentsOptions{
			Client:     opts.Client,
			Hours:      1,
			Format:     opts.Format,
			Session:    opts.Session,
			Level:      opts.Level,
			Window:     opts.Window,
			Pane:       opts.Pane,
			OlderThan:  opts.OlderThan,
			NewerThan:  opts.NewerThan,
			ReadFilter: opts.ReadFilter,
		})
	case "sessions":
		PrintTabs(TabsOptions{
			Client:     opts.Client,
			All:        false,
			Format:     opts.Format,
			Session:    opts.Session,
			Level:      opts.Level,
			Window:     opts.Window,
			Pane:       opts.Pane,
			OlderThan:  opts.OlderThan,
			NewerThan:  opts.NewerThan,
			ReadFilter: opts.ReadFilter,
		})
	case "all":
		PrintList(FilterOptions{
			Client:  opts.Client,
			State:   "all",
			Format:  opts.Format,
			Session: opts.Session,
			Level:   opts.Level,
			Window:  opts.Window,
			Pane:    opts.Pane,
		})
	}
}
