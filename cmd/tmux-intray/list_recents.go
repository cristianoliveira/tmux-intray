package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

// RecentsOptions holds options for the recents command.
type RecentsOptions struct {
	Client       listClient
	Hours        int
	Format       string // "simple" or "table"
	Session      string
	Level        string
	Window       string
	Pane         string
	OlderThan    string
	NewerThan    string
	ReadFilter   string
	DisplayNames appcore.DisplayNames
	RawIDs       bool
	ShowStale    bool
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
		_, _ = fmt.Fprintln(w, "No recent unread notifications found")
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
		formatRecentsUsingListFormatter(sessionBest, format.FormatterTypeJSON, opts.DisplayNames, opts.RawIDs, opts.ShowStale, w)
	case "table":
		formatRecentsUsingListFormatter(sessionBest, format.FormatterTypeTable, opts.DisplayNames, opts.RawIDs, opts.ShowStale, w)
	case "legacy":
		formatRecentsUsingListFormatter(sessionBest, format.FormatterTypeLegacy, opts.DisplayNames, opts.RawIDs, opts.ShowStale, w)
	case "compact":
		formatRecentsUsingListFormatter(sessionBest, format.FormatterTypeCompact, opts.DisplayNames, opts.RawIDs, opts.ShowStale, w)
	default:
		// Keep default aligned with `tmux-intray list` (simple formatter)
		formatRecentsUsingListFormatter(sessionBest, format.FormatterTypeSimple, opts.DisplayNames, opts.RawIDs, opts.ShowStale, w)
	}
}

func formatRecentsUsingListFormatter(notifs []notification.Notification, ftype format.FormatterType, displayNames appcore.DisplayNames, rawIDs, showStale bool, w io.Writer) {
	// Use the same formatter implementation as `tmux-intray list`.
	formatter := format.NewFormatter(ftype)

	// Convert cmd-level notifications to domain notifications.
	domainNotifs := make([]*domain.Notification, 0, len(notifs))
	for i := range notifs {
		n := notifs[i]
		domainNotifs = append(domainNotifs, &domain.Notification{
			ID:            n.ID,
			Timestamp:     n.Timestamp,
			State:         domain.NotificationState(n.State),
			Session:       n.Session,
			Window:        n.Window,
			Pane:          n.Pane,
			Message:       n.Message,
			PaneCreated:   n.PaneCreated,
			Level:         domain.NotificationLevel(n.Level),
			ReadTimestamp: n.ReadTimestamp,
		})
	}

	domainNotifs = keepOnlyResolvableTmuxRows(domainNotifs, ftype, displayNames, rawIDs, showStale)
	if !rawIDs && ftype == format.FormatterTypeSimple {
		domainNotifs = displayNames.EnrichNotifications(domainNotifs)
	}

	_ = formatter.FormatNotifications(domainNotifs, w)
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

// NOTE: recents output formatting is intentionally shared with `tmux-intray list`
// via internal/format. Older custom renderers were removed to keep output consistent.

// TabOptions holds options for the tab flag.
type TabOptions struct {
	Client       listClient
	Tab          string // "recents" or "sessions" or "all"
	Format       string
	Session      string
	Level        string
	Window       string
	Pane         string
	OlderThan    string
	NewerThan    string
	ReadFilter   string
	DisplayNames appcore.DisplayNames
	RawIDs       bool
	ShowStale    bool
}

// PrintTab prints the specified tab view.
func PrintTab(opts TabOptions) {
	switch opts.Tab {
	case "recents":
		PrintRecents(RecentsOptions{
			Client:       opts.Client,
			Hours:        1,
			Format:       opts.Format,
			Session:      opts.Session,
			Level:        opts.Level,
			Window:       opts.Window,
			Pane:         opts.Pane,
			OlderThan:    opts.OlderThan,
			NewerThan:    opts.NewerThan,
			ReadFilter:   opts.ReadFilter,
			DisplayNames: opts.DisplayNames,
			RawIDs:       opts.RawIDs,
			ShowStale:    opts.ShowStale,
		})
	case "sessions":
		PrintTabs(TabsOptions{
			Client:       opts.Client,
			All:          false,
			Format:       opts.Format,
			Session:      opts.Session,
			Level:        opts.Level,
			Window:       opts.Window,
			Pane:         opts.Pane,
			OlderThan:    opts.OlderThan,
			NewerThan:    opts.NewerThan,
			ReadFilter:   opts.ReadFilter,
			DisplayNames: opts.DisplayNames,
			RawIDs:       opts.RawIDs,
			ShowStale:    opts.ShowStale,
		})
	case "all":
		PrintList(FilterOptions{
			Client:       opts.Client,
			State:        "all",
			Format:       opts.Format,
			Session:      opts.Session,
			Level:        opts.Level,
			Window:       opts.Window,
			Pane:         opts.Pane,
			DisplayNames: opts.DisplayNames,
			RawIDs:       opts.RawIDs,
			ShowStale:    opts.ShowStale,
		})
	}
}
