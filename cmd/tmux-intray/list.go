/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
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
    --tab <tab>          Show special tab view: recents, sessions, all
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

TAB VIEWS:
    --tab=recents        Show recent unread notifications (max 1 per session, last hour)
    --tab=sessions       Show unique sessions with notifications
    --tab=all            Show all notifications (same as --all)

ORDERING:
    Unread notifications are listed first, then read notifications.
    Relative order remains unchanged within each group.
    -h, --help           Show this help`

// NewListCmd creates the list command with explicit dependencies.
func NewListCmd(client listClient) *cobra.Command {
	if client == nil {
		panic("NewListCmd: client dependency cannot be nil")
	}

	// Create the main list command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List notifications with filters and formats",
		Long:  listCommandLong,
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
	var listTab string
	var listJSON bool

	listCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Handle --json flag
		if listJSON {
			listFormat = "json"
		}

		// Handle --tab flag
		if listTab != "" {
			validTabs := []string{"recents", "sessions", "all"}
			if !isValidTab(listTab, validTabs) {
				return fmt.Errorf("invalid --tab value: %s (available: %s)", listTab, strings.Join(validTabs, ", "))
			}

			olderCutoff, newerCutoff := computeCutoffTimestamps(listOlderThan, listNewerThan)

			tabOpts := TabOptions{
				Client:     client,
				Tab:        listTab,
				Format:     listFormat,
				Session:    listSession,
				Level:      listLevel,
				Window:     listWindow,
				Pane:       listPane,
				OlderThan:  olderCutoff,
				NewerThan:  newerCutoff,
				ReadFilter: listFilter,
			}
			PrintTab(tabOpts)
			return nil
		}

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
	}

	registerListFlags(listCmd, &listPane, &listLevel, &listSession, &listWindow, &listOlderThan, &listNewerThan, &listSearch, &listRegex, &listGroupBy, &listGroupCount, &listFormat, &listFilter)

	// Add --tab flag
	listCmd.Flags().StringVar(&listTab, "tab", "", "Show special tab view: recents, sessions, all")

	// Add --json flag
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")

	return listCmd
}

// isValidTab checks if a tab value is valid.
func isValidTab(tab string, validTabs []string) bool {
	for _, t := range validTabs {
		if t == tab {
			return true
		}
	}
	return false
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

// listNow is the time source used for list timestamp filters.
var listNow = time.Now

// computeCutoffTimestamps computes timestamp cutoffs for older/newer-than filters.
func computeCutoffTimestamps(olderThan, newerThan int) (olderCutoff, newerCutoff string) {
	base := listNow().UTC().Truncate(time.Second)
	if olderThan > 0 {
		olderCutoff = base.AddDate(0, 0, -olderThan).Format("2006-01-02T15:04:05Z")
	}
	if newerThan > 0 {
		newerCutoff = base.AddDate(0, 0, -newerThan).Format("2006-01-02T15:04:05Z")
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

// listOutputWriter is the writer used by PrintList. Can be changed for testing.
var listOutputWriter io.Writer = os.Stdout

// listListFunc is the function used to retrieve notifications. Can be changed for testing.
var listListFunc func(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error)

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
		_, _ = fmt.Fprintf(w, "%s%s%s\n", colors.Blue, "No notifications found", colors.Reset)
		return
	}

	searchProvider := getSearchProvider(opts)
	notifications := parseAndFilterNotifications(lines, searchProvider, opts.Search)
	if len(notifications) == 0 {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", colors.Blue, "No notifications found", colors.Reset)
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
	if listListFunc != nil {
		return listListFunc(opts.State, opts.Level, opts.Session, opts.Window, opts.Pane, opts.OlderThan, opts.NewerThan, opts.ReadFilter)
	}
	return "", fmt.Errorf("list: missing client")
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

	if opts.Format == "table" {
		printTabsTable(sessionGroups, w)
	} else if opts.Format == "json" {
		printTabsJSON(sessionGroups, w)
	} else {
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

	if opts.Format == "json" {
		printRecentsJSON(sessionBest, w)
	} else if opts.Format == "table" {
		printRecentsTable(sessionBest, w)
	} else {
		printRecentsSimple(sessionBest, w)
	}
}

// recentsJSON represents a notification in JSON output for recents.
type recentsJSON struct {
	Num       int    `json:"num"`
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

		_, _ = fmt.Fprintf(w, "%s%d.%s %s%s%s %s\n",
			colors.Bold, num, colors.Reset,
			colors.Yellow, sessionDisplay, colors.Reset,
			age,
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
	_, _ = fmt.Fprintf(w, "%-4s %-20s %-10s %-8s %s\n",
		colors.Bold+"Num"+colors.Reset,
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

		_, _ = fmt.Fprintf(w, "%-4d %-20s %-10s %s%-8s%s %s\n",
			num,
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
