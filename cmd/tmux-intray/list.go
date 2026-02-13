/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/cmd"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/spf13/cobra"
)

type listClient interface {
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
}

// NewListCmd creates the list command with explicit dependencies.
func NewListCmd(client listClient) *cobra.Command {
	if client == nil {
		panic("NewListCmd: client dependency cannot be nil")
	}

	var listPane string
	var listLevel string
	var listSession string
	var listWindow string
	var listOlderThan int
	var listNewerThan int
	var listSearch string
	var listRegex bool
	var listGroupBy string
	var listGroupCount bool
	var listFormat string
	var listFilter string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List notifications with filters and formats",
		Long: `List notifications with filters and formats.

USAGE:
    tmux-intray list [OPTIONS]

OPTIONS:
    --active             Show active notifications (default)
    --dismissed          Show dismissed notifications
    --all                Show all notifications
    --pane <id>          Filter notifications by pane ID (e.g., %0)
    --level <level>      Filter notifications by level: info, warning, error, critical
    --session <id>       Filter notifications by session ID
    --window <id>        Filter notifications by window ID
    --older-than <days>  Show notifications older than N days
    --newer-than <days>  Show notifications newer than N days
    --search <pattern>   Search messages (substring match)
    --regex              Use regex search with --search
    --group-by <field>   Group notifications by field (session, window, pane, level)
    --group-count        Show only group counts (requires --group-by)
    --filter <status>    Filter notifications by read status: read, unread
    --format=<format>    Output format: simple (default), legacy, table, compact, json

ORDERING:
    Unread notifications are listed first, then read notifications.
    Relative order remains unchanged within each group.
    -h, --help           Show this help`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine state filter based on flags (default active)
			state := "active"
			if cmd.Flag("dismissed").Changed {
				state = "dismissed"
			}
			if cmd.Flag("all").Changed {
				state = "all"
			}

			// Compute cutoff timestamps
			var olderCutoff, newerCutoff string
			if listOlderThan > 0 {
				t := time.Now().UTC().AddDate(0, 0, -listOlderThan)
				olderCutoff = t.Format("2006-01-02T15:04:05Z")
			}
			if listNewerThan > 0 {
				t := time.Now().UTC().AddDate(0, 0, -listNewerThan)
				newerCutoff = t.Format("2006-01-02T15:04:05Z")
			}

			// Validate group-by field
			if listGroupBy != "" && listGroupBy != "session" && listGroupBy != "window" && listGroupBy != "pane" && listGroupBy != "level" {
				return fmt.Errorf("invalid group-by field: %s (must be session, window, pane, level)", listGroupBy)
			}

			// Validate read filter
			if listFilter != "" && listFilter != "read" && listFilter != "unread" {
				return fmt.Errorf("invalid filter value: %s (must be read or unread)", listFilter)
			}

			opts := FilterOptions{
				Client:     client,
				State:      state,
				Level:      listLevel,
				Session:    listSession,
				Window:     listWindow,
				Pane:       listPane,
				OlderThan:  olderCutoff,
				NewerThan:  newerCutoff,
				Search:     listSearch,
				Regex:      listRegex,
				GroupBy:    listGroupBy,
				GroupCount: listGroupCount,
				Format:     listFormat,
				ReadFilter: listFilter,
			}
			PrintList(opts)
			return nil
		},
	}

	// Flags
	listCmd.Flags().Bool("active", false, "Show active notifications (default)")
	listCmd.Flags().Bool("dismissed", false, "Show dismissed notifications")
	listCmd.Flags().Bool("all", false, "Show all notifications")
	listCmd.Flags().StringVar(&listPane, "pane", "", "Filter notifications by pane ID (e.g., %0)")
	listCmd.Flags().StringVar(&listLevel, "level", "", "Filter notifications by level: info, warning, error, critical")
	listCmd.Flags().StringVar(&listSession, "session", "", "Filter notifications by session ID")
	listCmd.Flags().StringVar(&listWindow, "window", "", "Filter notifications by window ID")
	listCmd.Flags().IntVar(&listOlderThan, "older-than", 0, "Show notifications older than N days")
	listCmd.Flags().IntVar(&listNewerThan, "newer-than", 0, "Show notifications newer than N days")
	listCmd.Flags().StringVar(&listSearch, "search", "", "Search messages (substring match)")
	listCmd.Flags().BoolVar(&listRegex, "regex", false, "Use regex search with --search")
	listCmd.Flags().StringVar(&listGroupBy, "group-by", "", "Group notifications by field (session, window, pane, level)")
	listCmd.Flags().BoolVar(&listGroupCount, "group-count", false, "Show only group counts (requires --group-by)")
	listCmd.Flags().StringVar(&listFormat, "format", "simple", "Output format: simple (default), legacy, table, compact, json")
	listCmd.Flags().StringVar(&listFilter, "filter", "", "Filter notifications by read status: read, unread")

	return listCmd
}

// listCmd represents the list command
var listCmd = NewListCmd(coreClient)

// listOutputWriter is the writer used by PrintList. Can be changed for testing.
var listOutputWriter io.Writer = os.Stdout

// listListFunc is the function used to retrieve notifications. Can be changed for testing.
var listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
	result, _ := coreClient.ListNotifications(state, level, session, window, pane, olderThan, newerThan, readFilter)
	return result
}

// FilterOptions holds all filter parameters for listing notifications.
type FilterOptions struct {
	Client         listClient
	State          string
	Level          string
	Session        string
	Window         string
	Pane           string
	OlderThan      string // timestamp cutoff (>=)
	NewerThan      string // timestamp cutoff (<=)
	Search         string
	Regex          bool
	GroupBy        string
	GroupCount     bool
	Format         string          // legacy, table, compact, json
	SearchProvider search.Provider // Optional custom search provider (for testing/extension)
	ReadFilter     string          // read status filter: "read", "unread", or "" (no filter)
}

// PrintList prints notifications according to the provided filter options.
func PrintList(opts FilterOptions) {
	if listOutputWriter == nil {
		listOutputWriter = os.Stdout
	}
	printList(opts, listOutputWriter)
}

func printList(opts FilterOptions, w io.Writer) {
	// Get filtered notifications from storage
	var lines string
	if opts.Client != nil {
		var err error
		lines, err = opts.Client.ListNotifications(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, opts.OlderThan, opts.NewerThan, opts.ReadFilter)
		if err != nil {
			fmt.Fprintf(w, "list: failed to list notifications: %v\n", err)
			return
		}
	} else {
		lines = listListFunc(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, opts.OlderThan, opts.NewerThan, opts.ReadFilter)
	}
	if lines == "" {
		fmt.Fprintln(w, "No notifications found")
		return
	}

	// Determine search provider
	var searchProvider search.Provider
	if opts.SearchProvider != nil {
		// Use custom provider if provided
		searchProvider = opts.SearchProvider
	} else if opts.Search != "" {
		// Fetch name maps for transparent name-based search
		client := tmux.NewDefaultClient()
		sessionNames, err := client.ListSessions()
		if err != nil {
			sessionNames = make(map[string]string)
		}
		windowNames, err := client.ListWindows()
		if err != nil {
			windowNames = make(map[string]string)
		}
		paneNames, err := client.ListPanes()
		if err != nil {
			paneNames = make(map[string]string)
		}

		// Create default provider based on Regex flag
		if opts.Regex {
			searchProvider = search.NewRegexProvider(
				search.WithCaseInsensitive(false),
				search.WithSessionNames(sessionNames),
				search.WithWindowNames(windowNames),
				search.WithPaneNames(paneNames),
			)
		} else {
			searchProvider = search.NewSubstringProvider(
				search.WithCaseInsensitive(false),
				search.WithSessionNames(sessionNames),
				search.WithWindowNames(windowNames),
				search.WithPaneNames(paneNames),
			)
		}
	}

	// Parse lines into notifications
	var notifications []notification.Notification
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, err := notification.ParseNotification(line)
		if err != nil {
			continue
		}
		// Apply search filter using search provider
		if searchProvider != nil {
			if !searchProvider.Match(notif, opts.Search) {
				continue
			}
		}
		notifications = append(notifications, notif)
	}

	if len(notifications) == 0 {
		fmt.Fprintln(w, "No notifications found")
		return
	}

	notifications = orderUnreadFirst(notifications)

	// Apply grouping if requested
	if opts.GroupBy != "" {
		grouped := groupNotifications(notifications, opts.GroupBy)
		if opts.GroupCount {
			printGroupCounts(grouped, w, opts.Format)
		} else {
			printGrouped(grouped, w, opts.Format)
		}
		return
	}

	// Print based on format
	switch opts.Format {
	case "simple":
		printSimple(notifications, w)
	case "legacy":
		printLegacy(notifications, w)
	case "table":
		printTable(notifications, w)
	case "compact":
		printCompact(notifications, w)
	case "json":
		fmt.Fprintln(w, "JSON format not yet implemented")
	default:
		fmt.Fprintf(w, "list: unknown format: %s\n", opts.Format)
	}
}

// orderUnreadFirst places unread notifications before read notifications.
// It keeps the existing relative order within each bucket (stable).
func orderUnreadFirst(notifs []notification.Notification) []notification.Notification {
	if len(notifs) == 0 {
		return notifs
	}

	ordered := make([]notification.Notification, len(notifs))
	copy(ordered, notifs)

	sort.SliceStable(ordered, func(i, j int) bool {
		iUnread := !ordered[i].IsRead()
		jUnread := !ordered[j].IsRead()
		if iUnread == jUnread {
			return false
		}
		return iUnread && !jUnread
	})

	return ordered
}

// groupNotifications groups notifications by field.
func groupNotifications(notifs []notification.Notification, field string) map[string][]notification.Notification {
	groups := make(map[string][]notification.Notification)
	for _, n := range notifs {
		var key string
		switch field {
		case "session":
			key = n.Session
		case "window":
			key = n.Window
		case "pane":
			key = n.Pane
		case "level":
			key = n.Level
		default:
			key = ""
		}
		groups[key] = append(groups[key], n)
	}
	return groups
}

// printGroupCounts prints only group counts.
func printGroupCounts(groups map[string][]notification.Notification, w io.Writer, format string) {
	// Sort keys for consistent output
	var keys []string
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "Group: %s (%d)\n", k, len(groups[k]))
	}
}

// printGrouped prints grouped notifications with headers.
func printGrouped(groups map[string][]notification.Notification, w io.Writer, format string) {
	// Sort keys for consistent output
	var keys []string
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "=== %s (%d) ===\n", k, len(groups[k]))
		switch format {
		case "simple":
			printSimple(groups[k], w)
		case "legacy":
			printLegacy(groups[k], w)
		case "table":
			printTable(groups[k], w)
		case "compact":
			printCompact(groups[k], w)
		default:
			printLegacy(groups[k], w)
		}
	}
}

// printLegacy prints only messages (one per line).
func printLegacy(notifs []notification.Notification, w io.Writer) {
	for _, n := range notifs {
		fmt.Fprintln(w, n.Message)
	}
}

// printSimple prints a simple format: ID DATE - Message.
// Optimized for quick scanning with ID, timestamp, and message on one line.
func printSimple(notifs []notification.Notification, w io.Writer) {
	for _, n := range notifs {
		// Truncate message for display (50 chars max)
		displayMsg := n.Message
		if len(displayMsg) > 50 {
			displayMsg = displayMsg[:47] + "..."
		}
		fmt.Fprintf(w, "%-4d  %-25s  - %s\n", n.ID, n.Timestamp, displayMsg)
	}
}

// printTable prints a formatted table with ID, Timestamp, Message, and optional context (Session Window Pane).
// Format: ID DATE - Message (Session Window Pane)
// Optimized for readability with ID first for easy copying.
func printTable(notifs []notification.Notification, w io.Writer) {
	if len(notifs) == 0 {
		return
	}
	headerColor := colors.Blue
	reset := colors.Reset
	fmt.Fprintf(w, "%sID    DATE                   - Message%s\n", headerColor, reset)
	fmt.Fprintf(w, "%s----  ---------------------  - --------------------------------%s\n", headerColor, reset)
	for _, n := range notifs {
		// Truncate message for display (32 chars max)
		displayMsg := n.Message
		if len(displayMsg) > 32 {
			displayMsg = displayMsg[:29] + "..."
		}
		fmt.Fprintf(w, "%-4d  %-23s  - %s\n", n.ID, n.Timestamp, displayMsg)
	}
}

// printCompact prints a compact format with Message only.
func printCompact(notifs []notification.Notification, w io.Writer) {
	for _, n := range notifs {
		// Truncate message for display
		displayMsg := n.Message
		if len(displayMsg) > 60 {
			displayMsg = displayMsg[:57] + "..."
		}
		fmt.Fprintln(w, displayMsg)
	}
}

func init() {
	cmd.RootCmd.AddCommand(listCmd)
}
