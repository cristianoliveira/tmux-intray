/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/cmd"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
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
    --format=<format>    Output format: legacy, table, compact, json
    -h, --help           Show this help`,
	Run: runList,
}

var (
	listState      string
	listPane       string
	listLevel      string
	listSession    string
	listWindow     string
	listOlderThan  int
	listNewerThan  int
	listSearch     string
	listRegex      bool
	listGroupBy    string
	listGroupCount bool
	listFormat     string
)

// listOutputWriter is the writer used by PrintList. Can be changed for testing.
var listOutputWriter io.Writer = os.Stdout

// listListFunc is the function used to retrieve notifications. Can be changed for testing.
var listListFunc = func(state, level, session, window, pane, olderThan, newerThan string) string {
	return storage.ListNotifications(state, level, session, window, pane, olderThan, newerThan)
}

// FilterOptions holds all filter parameters for listing notifications.
type FilterOptions struct {
	State      string
	Level      string
	Session    string
	Window     string
	Pane       string
	OlderThan  string // timestamp cutoff (>=)
	NewerThan  string // timestamp cutoff (<=)
	Search     string
	Regex      bool
	GroupBy    string
	GroupCount bool
	Format     string // legacy, table, compact, json
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
	lines := listListFunc(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, opts.OlderThan, opts.NewerThan)
	if lines == "" {
		fmt.Fprintln(w, "No notifications found")
		return
	}

	// Parse lines into notifications
	var notifications []Notification
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, err := parseNotification(line)
		if err != nil {
			continue
		}
		// Apply search filter
		if opts.Search != "" {
			if opts.Regex {
				re, err := regexp.Compile(opts.Search)
				if err != nil {
					// Invalid regex, treat as literal substring
					if !strings.Contains(notif.Message, opts.Search) {
						continue
					}
				} else {
					if !re.MatchString(notif.Message) {
						continue
					}
				}
			} else {
				if !strings.Contains(notif.Message, opts.Search) {
					continue
				}
			}
		}
		notifications = append(notifications, notif)
	}

	if len(notifications) == 0 {
		fmt.Fprintln(w, "No notifications found")
		return
	}

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
	case "legacy":
		printLegacy(notifications, w)
	case "table":
		printTable(notifications, w)
	case "compact":
		printCompact(notifications, w)
	case "json":
		fmt.Fprintln(w, "JSON format not yet implemented")
	default:
		fmt.Fprintf(w, "Unknown format: %s\n", opts.Format)
	}
}

// groupNotifications groups notifications by field.
func groupNotifications(notifs []Notification, field string) map[string][]Notification {
	groups := make(map[string][]Notification)
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
func printGroupCounts(groups map[string][]Notification, w io.Writer, format string) {
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
func printGrouped(groups map[string][]Notification, w io.Writer, format string) {
	// Sort keys for consistent output
	var keys []string
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "=== %s (%d) ===\n", k, len(groups[k]))
		switch format {
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
func printLegacy(notifs []Notification, w io.Writer) {
	for _, n := range notifs {
		fmt.Fprintln(w, n.Message)
	}
}

// printTable prints a formatted table with Timestamp, Session, Window, Pane, Level, Message, and ID.
// ID is positioned at the end for easy mouse selection and copying.
func printTable(notifs []Notification, w io.Writer) {
	if len(notifs) == 0 {
		return
	}
	headerColor := colors.Blue
	reset := colors.Reset
	fmt.Fprintf(w, "%sTimestamp                 Session Window    Pane    Level   Message                               ID%s\n", headerColor, reset)
	fmt.Fprintf(w, "%s------------------------  ------- ------  ------  ------  -------------------------------------- -----%s\n", headerColor, reset)
	for _, n := range notifs {
		// Truncate message for display (more space now that ID is at end)
		displayMsg := n.Message
		if len(displayMsg) > 37 {
			displayMsg = displayMsg[:34] + "..."
		}
		fmt.Fprintf(w, "%-25s  %-7s %-6s  %-6s  %-6s  %-38s %-5d\n", n.Timestamp, n.Session, n.Window, n.Pane, n.Level, displayMsg, n.ID)
	}
}

// printCompact prints a compact format with Message only.
func printCompact(notifs []Notification, w io.Writer) {
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

	// State filters (mutually exclusive)
	listCmd.Flags().Bool("active", false, "Show active notifications (default)")
	listCmd.Flags().Bool("dismissed", false, "Show dismissed notifications")
	listCmd.Flags().Bool("all", false, "Show all notifications")
	// Other filters
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
	listCmd.Flags().StringVar(&listFormat, "format", "legacy", "Output format: legacy, table, compact, json")
}

func runList(cmd *cobra.Command, args []string) {
	// Determine state filter based on flags (default active)
	state := "active"
	if cmd.Flag("dismissed").Changed {
		state = "dismissed"
	}
	if cmd.Flag("all").Changed {
		state = "all"
	}
	// If both active and dismissed? active is default, we'll ignore active flag.

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
		cmd.Printf("Invalid group-by field: %s (must be session, window, pane, level)\n", listGroupBy)
		return
	}

	opts := FilterOptions{
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
	}

	PrintList(opts)
}
