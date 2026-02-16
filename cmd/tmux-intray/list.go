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
	"github.com/cristianoliveira/tmux-intray/internal/dedupconfig"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/spf13/cobra"
)

type listClient interface {
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
}

// notificationsToValues converts a slice of notification pointers to values.
func notificationsToValues(notifs []*domain.Notification) []domain.Notification {
	values := make([]domain.Notification, len(notifs))
	for i, n := range notifs {
		values[i] = *n
	}
	return values
}

const listCommandLong = `List notifications with filters and formats.

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
    --group-by <field>   Group notifications by field (session, window, pane, level, message)
    --group-count        Show only group counts (requires --group-by)
    --filter <status>    Filter notifications by read status: read, unread
    --format=<format>    Output format: simple (default), legacy, table, compact, json

ORDERING:
    Unread notifications are listed first, then read notifications.
    Relative order remains unchanged within each group.
    -h, --help           Show this help`

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
		Long:  listCommandLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			state := determineListState(cmd)
			olderCutoff, newerCutoff := computeCutoffTimestamps(listOlderThan, listNewerThan)
			if err := validateListOptions(listGroupBy, listFilter); err != nil {
				return err
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

	registerListFlags(listCmd, &listPane, &listLevel, &listSession, &listWindow, &listOlderThan, &listNewerThan, &listSearch, &listRegex, &listGroupBy, &listGroupCount, &listFormat, &listFilter)

	return listCmd
}

// registerListFlags registers all flags for the list command.
func registerListFlags(cmd *cobra.Command, listPane, listLevel, listSession, listWindow *string, listOlderThan, listNewerThan *int, listSearch *string, listRegex *bool, listGroupBy *string, listGroupCount *bool, listFormat, listFilter *string) {
	cmd.Flags().Bool("active", false, "Show active notifications (default)")
	cmd.Flags().Bool("dismissed", false, "Show dismissed notifications")
	cmd.Flags().Bool("all", false, "Show all notifications")
	cmd.Flags().StringVar(listPane, "pane", "", "Filter notifications by pane ID (e.g., %0)")
	cmd.Flags().StringVar(listLevel, "level", "", "Filter notifications by level: info, warning, error, critical")
	cmd.Flags().StringVar(listSession, "session", "", "Filter notifications by session ID")
	cmd.Flags().StringVar(listWindow, "window", "", "Filter notifications by window ID")
	cmd.Flags().IntVar(listOlderThan, "older-than", 0, "Show notifications older than N days")
	cmd.Flags().IntVar(listNewerThan, "newer-than", 0, "Show notifications newer than N days")
	cmd.Flags().StringVar(listSearch, "search", "", "Search messages (substring match)")
	cmd.Flags().BoolVar(listRegex, "regex", false, "Use regex search with --search")
	cmd.Flags().StringVar(listGroupBy, "group-by", "", "Group notifications by field (session, window, pane, level, message)")
	cmd.Flags().BoolVar(listGroupCount, "group-count", false, "Show only group counts (requires --group-by)")
	cmd.Flags().StringVar(listFormat, "format", "simple", "Output format: simple (default), legacy, table, compact, json")
	cmd.Flags().StringVar(listFilter, "filter", "", "Filter notifications by read status: read, unread")
}

// determineListState determines the state filter based on flags.
func determineListState(cmd *cobra.Command) string {
	state := "active"
	if cmd.Flag("dismissed").Changed {
		state = "dismissed"
	}
	if cmd.Flag("all").Changed {
		state = "all"
	}
	return state
}

// computeCutoffTimestamps computes timestamp cutoffs for older/newer-than filters.
func computeCutoffTimestamps(olderThan, newerThan int) (olderCutoff, newerCutoff string) {
	if olderThan > 0 {
		t := time.Now().UTC().AddDate(0, 0, -olderThan)
		olderCutoff = t.Format("2006-01-02T15:04:05Z")
	}
	if newerThan > 0 {
		t := time.Now().UTC().AddDate(0, 0, -newerThan)
		newerCutoff = t.Format("2006-01-02T15:04:05Z")
	}
	return
}

// validateListOptions validates list command options.
func validateListOptions(groupBy, filter string) error {
	// Validate group-by field
	if groupBy != "" && groupBy != "session" && groupBy != "window" && groupBy != "pane" && groupBy != "level" && groupBy != "message" {
		return fmt.Errorf("invalid group-by field: %s (must be session, window, pane, level, message)", groupBy)
	}

	// Validate read filter
	if filter != "" && filter != "read" && filter != "unread" {
		return fmt.Errorf("invalid filter value: %s (must be read or unread)", filter)
	}

	return nil
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
	lines, err := fetchNotifications(opts)
	if err != nil {
		_, _ = fmt.Fprintf(w, "list: failed to list notifications: %v\n", err)
		return
	}
	if lines == "" {
		fmt.Fprintf(w, "%s%s%s\n", colors.Blue, "No notifications found", colors.Reset)
		return
	}

	searchProvider := getSearchProvider(opts)
	notifications := parseAndFilterNotifications(lines, searchProvider, opts.Search)
	if len(notifications) == 0 {
		fmt.Fprintf(w, "%s%s%s\n", colors.Blue, "No notifications found", colors.Reset)
		return
	}

	notifications = orderUnreadFirst(notifications)
	printNotifications(notifications, opts, w)
}

// fetchNotifications retrieves notifications from storage.
func fetchNotifications(opts FilterOptions) (string, error) {
	if opts.Client != nil {
		return opts.Client.ListNotifications(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, opts.OlderThan, opts.NewerThan, opts.ReadFilter)
	}
	return listListFunc(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, opts.OlderThan, opts.NewerThan, opts.ReadFilter), nil
}

// getSearchProvider returns the appropriate search provider based on options.
func getSearchProvider(opts FilterOptions) search.Provider {
	if opts.SearchProvider != nil {
		return opts.SearchProvider
	}
	if opts.Search == "" {
		return nil
	}

	// Fetch name maps for transparent name-based search
	client := tmux.NewDefaultClient()
	sessionNames, _ := client.ListSessions()
	if sessionNames == nil {
		sessionNames = make(map[string]string)
	}
	windowNames, _ := client.ListWindows()
	if windowNames == nil {
		windowNames = make(map[string]string)
	}
	paneNames, _ := client.ListPanes()
	if paneNames == nil {
		paneNames = make(map[string]string)
	}

	// Create default provider based on Regex flag
	if opts.Regex {
		return search.NewRegexProvider(
			search.WithCaseInsensitive(false),
			search.WithSessionNames(sessionNames),
			search.WithWindowNames(windowNames),
			search.WithPaneNames(paneNames),
		)
	}
	return search.NewSubstringProvider(
		search.WithCaseInsensitive(false),
		search.WithSessionNames(sessionNames),
		search.WithWindowNames(windowNames),
		search.WithPaneNames(paneNames),
	)
}

// parseAndFilterNotifications parses and filters notification lines.
func parseAndFilterNotifications(lines string, searchProvider search.Provider, searchQuery string) []*domain.Notification {
	var notifications []*domain.Notification
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
			if !searchProvider.Match(notif, searchQuery) {
				continue
			}
		}
		notifications = append(notifications, notification.ToDomainUnsafe(notif))
	}
	return notifications
}

// printNotifications prints notifications based on options.
func printNotifications(notifications []*domain.Notification, opts FilterOptions, w io.Writer) {
	// Apply grouping if requested
	if opts.GroupBy != "" {
		notificationsValues := notificationsToValues(notifications)
		var groupResult domain.GroupResult
		if opts.GroupBy == domain.GroupByMessage.String() {
			groupResult = domain.GroupNotificationsWithDedup(notificationsValues, domain.GroupByMode(opts.GroupBy), dedupconfig.Load())
		} else {
			groupResult = domain.GroupNotifications(notificationsValues, domain.GroupByMode(opts.GroupBy))
		}
		formatter := format.GetFormatter(opts.Format, opts.GroupCount)
		err := formatter.FormatGroups(groupResult, w)
		if err != nil {
			_, _ = fmt.Fprintf(w, "list: formatting error: %v\n", err)
		}
		return
	}

	// No grouping, use appropriate formatter
	formatter := format.GetFormatter(opts.Format, false)
	err := formatter.FormatNotifications(notifications, w)
	if err != nil {
		_, _ = fmt.Fprintf(w, "list: formatting error: %v\n", err)
	}
}

// orderUnreadFirst places unread notifications before read notifications.
// It keeps the existing relative order within each bucket (stable).
func orderUnreadFirst(notifs []*domain.Notification) []*domain.Notification {
	if len(notifs) == 0 {
		return notifs
	}

	ordered := make([]*domain.Notification, len(notifs))
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

func init() {
	cmd.RootCmd.AddCommand(listCmd)
}
