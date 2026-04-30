package main

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/stretchr/testify/assert"
)

// mockLines returns a fixed TSV string for testing.
func mockLines() string {
	return `1	2025-01-01T10:00:00Z	active	sess1	win1	pane1	message one	123	info	2025-01-01T10:05:00Z
2	2025-01-01T11:00:00Z	active	sess1	win1	pane2	message two	124	warning	
3	2025-01-01T12:00:00Z	dismissed	sess2	win2	pane3	message three	125	error	2025-01-01T12:05:00Z
4	2025-01-01T13:00:00Z	active	sess2	win2	pane4	message four	126	info	
5	2025-01-01T14:00:00Z	active	sess3	win3	pane5	message five	127	info	2025-01-01T14:05:00Z`
}

func runPrintList(t *testing.T, lines string, err error, opts FilterOptions) string {
	t.Helper()
	client := &fakeListClient{
		listNotificationsResult: lines,
		listNotificationsError:  err,
	}
	if opts.Client == nil {
		opts.Client = client
	}
	var buf bytes.Buffer
	PrintListTo(opts, &buf, defaultListSearchProvider)
	return buf.String()
}

var simpleColumnsSeparator = regexp.MustCompile(`\s{2,}`)

func splitSimpleColumns(t *testing.T, line string) []string {
	t.Helper()

	cols := simpleColumnsSeparator.Split(strings.TrimSpace(line), 7)
	if len(cols) != 7 {
		t.Fatalf("expected 7 columns, got %d from line %q", len(cols), line)
	}

	return cols
}

func TestPrintListEmpty(t *testing.T) {
	output := runPrintList(t, "", nil, FilterOptions{})
	if output != "\033[0;34mNo notifications found\033[0m\n" {
		t.Errorf("Expected colored 'No notifications found', got %q", output)
	}
}

func TestPrintListLegacyFormat(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{Format: "legacy"})
	// Should contain only messages, one per line
	if !strings.Contains(output, "message one") {
		t.Error("Missing message one")
	}
	if !strings.Contains(output, "message two") {
		t.Error("Missing message two")
	}
	// Ensure no IDs or timestamps appear
	if strings.Contains(output, "1") && strings.Contains(output, "2025-01-01") {
		t.Error("Legacy format should not show IDs or timestamps")
	}
}

func TestPrintListSimpleFormat(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{Format: "simple"})
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}

	first := splitSimpleColumns(t, lines[0])
	assert.Equal(t, "2", first[0])
	assert.Equal(t, "2025-01-01T11:00:00Z", first[1])
	assert.Equal(t, "sess1", first[2])
	assert.Equal(t, "win1", first[3])
	assert.Equal(t, "pane2", first[4])
	assert.Equal(t, "warning", first[5])
	assert.Equal(t, "message two", first[6])

	third := splitSimpleColumns(t, lines[2])
	assert.Equal(t, "1", third[0])
	assert.Equal(t, "2025-01-01T10:00:00Z", third[1])
	assert.Equal(t, "sess1", third[2])
	assert.Equal(t, "win1", third[3])
	assert.Equal(t, "pane1", third[4])
	assert.Equal(t, "info", third[5])
	assert.Equal(t, "message one", third[6])
}

func TestPrintListUnreadFirstOrdering(t *testing.T) {
	output := strings.TrimSpace(runPrintList(t, mockLines(), nil, FilterOptions{Format: "simple"}))

	var ids []int
	for _, line := range strings.Split(output, "\n") {
		cols := splitSimpleColumns(t, line)
		id, err := strconv.Atoi(cols[0])
		if err != nil {
			t.Fatalf("failed to parse ID from line %q: %v", line, err)
		}
		ids = append(ids, id)
	}

	assert.Equal(t, []int{2, 4, 1, 3, 5}, ids)
}

func TestPrintListTableFormat(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{Format: "table"})
	// Should contain header with ID and DATE
	if !strings.Contains(output, "ID") || !strings.Contains(output, "DATE") {
		t.Error("Table missing header")
	}
	// Should contain messages separator dash
	if !strings.Contains(output, "-") {
		t.Error("Table missing separator dash")
	}
	// Should contain IDs at the beginning of each row
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines, got %d", len(lines))
	}
	// Check that messages appear in output
	if !strings.Contains(output, "message one") || !strings.Contains(output, "message two") {
		t.Error("Table missing messages")
	}
}

func TestPrintListCompactFormat(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{Format: "compact"})
	// Should contain messages only, no extra whitespace
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if !strings.HasPrefix(line, "message ") {
			t.Errorf("Line %d doesn't start with 'message ': %q", i, line)
		}
	}
}

func TestPrintListSearchFilter(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{Search: "three", Format: "legacy"})
	if !strings.Contains(output, "message three") {
		t.Error("Search filter didn't find 'message three'")
	}
	if strings.Contains(output, "message one") {
		t.Error("Search filter incorrectly included 'message one'")
	}
}

func TestPrintListRegexSearch(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{Search: "e$", Regex: true, Format: "legacy"})
	// message one, three, five end with 'e'
	if !strings.Contains(output, "message one") {
		t.Error("Regex missing message one")
	}
	if !strings.Contains(output, "message three") {
		t.Error("Regex missing message three")
	}
	if !strings.Contains(output, "message five") {
		t.Error("Regex missing message five")
	}
	if strings.Contains(output, "message two") {
		t.Error("Regex incorrectly included message two")
	}
}

func TestPrintListGroupByLevelSimple(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{GroupBy: "level", Format: "simple"})
	// Should contain group headers
	if !strings.Contains(output, "=== info (3) ===") {
		t.Error("Missing info group header")
	}
	if !strings.Contains(output, "=== warning (1) ===") {
		t.Error("Missing warning group header")
	}
	if !strings.Contains(output, "=== error (1) ===") {
		t.Error("Missing error group header")
	}
	// Should contain ID, timestamps, routing fields, level, and message in simple format
	if !strings.Contains(output, "sess1") || !strings.Contains(output, "win1") || !strings.Contains(output, "pane1") || !strings.Contains(output, "info") {
		t.Error("Simple grouped format missing session, window, pane, or level")
	}
}

func TestPrintListGroupByLevel(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{GroupBy: "level", Format: "legacy"})
	// Should contain group headers
	if !strings.Contains(output, "=== info (3) ===") {
		t.Error("Missing info group header")
	}
	if !strings.Contains(output, "=== warning (1) ===") {
		t.Error("Missing warning group header")
	}
	if !strings.Contains(output, "=== error (1) ===") {
		t.Error("Missing error group header")
	}
}

func TestPrintListGroupByMessage(t *testing.T) {
	t.Setenv("TMUX_INTRAY_DEDUP__CRITERIA", "message")
	t.Setenv("TMUX_INTRAY_DEDUP__WINDOW", "")
	output := runPrintList(t,
		"1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\trepeated message\t123\tinfo\t\n"+
			"2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\trepeated message\t124\twarning\t\n"+
			"3\t2025-01-01T12:00:00Z\tactive\tsess2\twin2\tpane3\tunique message\t125\terror\t\n",
		nil,
		FilterOptions{GroupBy: "message", Format: "legacy"},
	)
	if !strings.Contains(output, "=== repeated message (2) ===") {
		t.Error("Missing repeated message group header")
	}
	if !strings.Contains(output, "=== unique message (1) ===") {
		t.Error("Missing unique message group header")
	}
}

func TestPrintListGroupCount(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{GroupBy: "level", GroupCount: true})
	// Should contain group counts only
	if !strings.Contains(output, "Group: info (3)") {
		t.Error("Missing group count for info")
	}
	if !strings.Contains(output, "Group: warning (1)") {
		t.Error("Missing group count for warning")
	}
	if !strings.Contains(output, "Group: error (1)") {
		t.Error("Missing group count for error")
	}
	// Should not contain messages
	if strings.Contains(output, "message") {
		t.Error("Group count should not list messages")
	}
}

func TestPrintListWithCustomSearchProvider(t *testing.T) {
	// Test with custom search provider that only matches "error"
	mockProvider := search.NewSubstringProvider(search.WithFields([]string{"level"}))

	opts := FilterOptions{
		Search:         "error",
		SearchProvider: mockProvider,
		Format:         "legacy",
	}
	output := runPrintList(t,
		"1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\terror message\t123\terror\n"+
			"2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\twarning message\t124\twarning\n",
		nil,
		opts,
	)

	// Should only match the first notification (has error in level)
	if !strings.Contains(output, "error message") {
		t.Error("Missing error message")
	}
	if strings.Contains(output, "warning message") {
		t.Error("Should not include warning message")
	}

	// Test with provider that matches warning
	opts.Search = "warning"
	output = runPrintList(t,
		"1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\terror message\t123\terror\n"+
			"2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\twarning message\t124\twarning\n",
		nil,
		opts,
	)
	if strings.Contains(output, "error message") {
		t.Error("Should not include error message")
	}
	if !strings.Contains(output, "warning message") {
		t.Error("Missing warning message")
	}
}

func TestPrintListBackwardCompatibility(t *testing.T) {
	// Verify that existing behavior is preserved when no custom provider is set

	// Test substring search (default, no regex flag)
	opts := FilterOptions{
		Search: "Hello",
		Regex:  false,
		Format: "legacy",
	}
	output := runPrintList(t, "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tHello World\t123\tinfo\n", nil, opts)
	if !strings.Contains(output, "Hello World") {
		t.Error("Substring search should find Hello World")
	}

	// Test regex search (with regex flag)
	opts.Regex = true
	output = runPrintList(t, "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tHello World\t123\tinfo\n", nil, opts)
	if !strings.Contains(output, "Hello World") {
		t.Error("Regex search should find Hello World")
	}

	// Test no search (should show all)
	opts.Search = ""
	output = runPrintList(t, "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tHello World\t123\tinfo\n", nil, opts)
	if !strings.Contains(output, "Hello World") {
		t.Error("Empty search should show all notifications")
	}
}

func TestPrintListFilterRead(t *testing.T) {
	// Test --filter read
	output := runPrintList(t,
		"1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage one\t123\tinfo\t2025-01-01T10:05:00Z\n"+
			"3\t2025-01-01T12:00:00Z\tdismissed\tsess2\twin2\tpane3\tmessage three\t125\terror\t2025-01-01T12:05:00Z\n"+
			"5\t2025-01-01T14:00:00Z\tactive\tsess3\twin3\tpane5\tmessage five\t127\tinfo\t2025-01-01T14:05:00Z",
		nil,
		FilterOptions{ReadFilter: "read", Format: "simple"},
	)

	// Should contain read notifications
	if !strings.Contains(output, "message one") {
		t.Error("Filter read should include message one")
	}
	if !strings.Contains(output, "message three") {
		t.Error("Filter read should include message three")
	}
	if !strings.Contains(output, "message five") {
		t.Error("Filter read should include message five")
	}

	// Should not contain unread notifications
	if strings.Contains(output, "message two") {
		t.Error("Filter read should not include message two (unread)")
	}
	if strings.Contains(output, "message four") {
		t.Error("Filter read should not include message four (unread)")
	}
}

func TestPrintListFilterUnread(t *testing.T) {
	// Test --filter unread
	output := runPrintList(t,
		"2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\tmessage two\t124\twarning\t\n"+
			"4\t2025-01-01T13:00:00Z\tactive\tsess2\twin2\tpane4\tmessage four\t126\tinfo\t",
		nil,
		FilterOptions{ReadFilter: "unread", Format: "simple"},
	)

	// Should contain unread notifications
	if !strings.Contains(output, "message two") {
		t.Error("Filter unread should include message two")
	}
	if !strings.Contains(output, "message four") {
		t.Error("Filter unread should include message four")
	}

	// Should not contain read notifications
	if strings.Contains(output, "message one") {
		t.Error("Filter unread should not include message one (read)")
	}
	if strings.Contains(output, "message three") {
		t.Error("Filter unread should not include message three (read)")
	}
	if strings.Contains(output, "message five") {
		t.Error("Filter unread should not include message five (read)")
	}
}

// fakeListClient implements listClient for testing
type fakeListClient struct {
	listNotificationsCalls []struct {
		stateFilter   string
		levelFilter   string
		sessionFilter string
		windowFilter  string
		paneFilter    string
		olderThan     string
		newerThan     string
		readFilter    string
	}
	listNotificationsResult string
	listNotificationsError  error
}

func (f *fakeListClient) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	f.listNotificationsCalls = append(f.listNotificationsCalls, struct {
		stateFilter   string
		levelFilter   string
		sessionFilter string
		windowFilter  string
		paneFilter    string
		olderThan     string
		newerThan     string
		readFilter    string
	}{
		stateFilter:   stateFilter,
		levelFilter:   levelFilter,
		sessionFilter: sessionFilter,
		windowFilter:  windowFilter,
		paneFilter:    paneFilter,
		olderThan:     olderThanCutoff,
		newerThan:     newerThanCutoff,
		readFilter:    readFilter,
	})
	return f.listNotificationsResult, f.listNotificationsError
}

func TestOrderUnreadFirstEdgeCases(t *testing.T) {
	// Empty slice
	empty := []*domain.Notification{}
	result := orderUnreadFirst(empty)
	assert.Equal(t, 0, len(result))

	// Single unread notification (no read timestamp)
	n1 := &domain.Notification{ID: 1, ReadTimestamp: "", State: domain.StateActive, Level: domain.LevelInfo, Message: "test", Timestamp: "2025-01-01T10:00:00Z"}
	single := []*domain.Notification{n1}
	result = orderUnreadFirst(single)
	assert.Equal(t, single, result)

	// Single read notification
	n2 := &domain.Notification{ID: 2, ReadTimestamp: "2025-01-01T10:00:00Z", State: domain.StateActive, Level: domain.LevelInfo, Message: "test", Timestamp: "2025-01-01T10:00:00Z"}
	singleRead := []*domain.Notification{n2}
	result = orderUnreadFirst(singleRead)
	assert.Equal(t, singleRead, result)

	// Mixed: unread should come before read
	notifs := []*domain.Notification{
		{ID: 1, ReadTimestamp: "2025-01-01T10:00:00Z", State: domain.StateActive, Level: domain.LevelInfo, Message: "test1", Timestamp: "2025-01-01T10:00:00Z"}, // read
		{ID: 2, ReadTimestamp: "", State: domain.StateActive, Level: domain.LevelInfo, Message: "test2", Timestamp: "2025-01-01T11:00:00Z"},                     // unread
		{ID: 3, ReadTimestamp: "2025-01-01T11:00:00Z", State: domain.StateActive, Level: domain.LevelInfo, Message: "test3", Timestamp: "2025-01-01T12:00:00Z"}, // read
		{ID: 4, ReadTimestamp: "", State: domain.StateActive, Level: domain.LevelInfo, Message: "test4", Timestamp: "2025-01-01T13:00:00Z"},                     // unread
	}
	result = orderUnreadFirst(notifs)
	// Expect unread IDs 2,4 first, then read IDs 1,3 preserving relative order
	assert.Equal(t, []*domain.Notification{notifs[1], notifs[3], notifs[0], notifs[2]}, result)
}

func TestGroupNotifications(t *testing.T) {
	notifs := []domain.Notification{
		{ID: 1, Session: "sess1", Window: "win1", Pane: "pane1", Level: domain.LevelInfo, State: domain.StateActive, Message: "test1", Timestamp: "2025-01-01T10:00:00Z"},
		{ID: 2, Session: "sess1", Window: "win2", Pane: "pane2", Level: domain.LevelWarning, State: domain.StateActive, Message: "test2", Timestamp: "2025-01-01T11:00:00Z"},
		{ID: 3, Session: "sess2", Window: "win1", Pane: "pane1", Level: domain.LevelInfo, State: domain.StateActive, Message: "test3", Timestamp: "2025-01-01T12:00:00Z"},
	}

	// Group by session
	result := domain.GroupNotifications(notifs, domain.GroupBySession)
	assert.Equal(t, 2, len(result.Groups))
	// Find groups by display name (should be sess1, sess2)
	var sess1Group, sess2Group domain.Group
	for _, g := range result.Groups {
		switch g.DisplayName {
		case "sess1":
			sess1Group = g
		case "sess2":
			sess2Group = g
		}
	}
	assert.Equal(t, 2, sess1Group.Count)
	assert.Equal(t, 1, sess2Group.Count)

	// Group by window
	result = domain.GroupNotifications(notifs, domain.GroupByWindow)
	assert.Equal(t, 3, len(result.Groups)) // sess1/win1, sess1/win2, sess2/win1
	// Check total count
	assert.Equal(t, 3, result.TotalCount)

	// Group by pane
	result = domain.GroupNotifications(notifs, domain.GroupByPane)
	assert.Equal(t, 3, len(result.Groups)) // sess1/win1/pane1, sess1/win2/pane2, sess2/win1/pane1
	assert.Equal(t, 3, result.TotalCount)

	// Group by level
	result = domain.GroupNotifications(notifs, domain.GroupByLevel)
	assert.Equal(t, 2, len(result.Groups)) // info, warning
	var infoGroup, warningGroup domain.Group
	for _, g := range result.Groups {
		switch g.DisplayName {
		case "info":
			infoGroup = g
		case "warning":
			warningGroup = g
		}
	}
	assert.Equal(t, 2, infoGroup.Count)
	assert.Equal(t, 1, warningGroup.Count)

	// Unknown field defaults to GroupByNone (empty groups)
	result = domain.GroupNotifications(notifs, domain.GroupByMode("unknown"))
	assert.Equal(t, 0, len(result.Groups))
	assert.Equal(t, 3, result.TotalCount)
}

func TestPrintFunctionsWithEmptySlice(t *testing.T) {
	var buf bytes.Buffer

	// Create formatters
	simpleFormatter := format.NewSimpleFormatter()
	tableFormatter := format.NewTableFormatter()
	compactFormatter := format.NewCompactFormatter()
	legacyFormatter := format.NewLegacyFormatter()

	// Test with empty slice
	var emptyNotifs []*domain.Notification

	simpleFormatter.FormatNotifications(emptyNotifs, &buf)
	assert.Equal(t, "", buf.String())
	buf.Reset()

	tableFormatter.FormatNotifications(emptyNotifs, &buf)
	assert.Equal(t, "", buf.String())
	buf.Reset()

	compactFormatter.FormatNotifications(emptyNotifs, &buf)
	assert.Equal(t, "", buf.String())
	buf.Reset()

	legacyFormatter.FormatNotifications(emptyNotifs, &buf)
	assert.Equal(t, "", buf.String())
}

func TestPrintFunctionsWithSingleNotification(t *testing.T) {
	notif := &domain.Notification{
		ID:        42,
		Timestamp: "2025-01-01T10:00:00Z",
		Message:   "test message",
		State:     domain.StateActive,
		Level:     domain.LevelInfo,
	}
	var buf bytes.Buffer

	// Create formatters
	simpleFormatter := format.NewSimpleFormatter()
	tableFormatter := format.NewTableFormatter()
	compactFormatter := format.NewCompactFormatter()
	legacyFormatter := format.NewLegacyFormatter()

	simpleFormatter.FormatNotifications([]*domain.Notification{notif}, &buf)
	assert.Contains(t, buf.String(), "42")
	assert.Contains(t, buf.String(), "test message")
	buf.Reset()

	tableFormatter.FormatNotifications([]*domain.Notification{notif}, &buf)
	assert.Contains(t, buf.String(), "42")
	buf.Reset()

	compactFormatter.FormatNotifications([]*domain.Notification{notif}, &buf)
	assert.Contains(t, buf.String(), "test message")
	buf.Reset()

	legacyFormatter.FormatNotifications([]*domain.Notification{notif}, &buf)
	assert.Equal(t, "test message\n", buf.String())
}

func TestPrintListJSONFormat(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{Format: "json"})
	// JSON format is now implemented, check for JSON structure
	if !strings.Contains(output, `"ID"`) || !strings.Contains(output, `"Message"`) {
		t.Errorf("Expected JSON output with ID and Message fields, got %q", output)
	}
}

func TestPrintListUnknownFormat(t *testing.T) {
	output := runPrintList(t, mockLines(), nil, FilterOptions{Format: "unknown"})
	// Unknown format should fall back to simple format (default)
	if !strings.Contains(output, "1") || !strings.Contains(output, "message") {
		t.Errorf("Expected simple format output with ID and message, got %q", output)
	}
}

func TestPrintListSimpleFormatUsesResolvedNamesByDefault(t *testing.T) {
	output := runPrintList(t,
		"1\t2025-01-01T10:00:00Z\tactive\t$1\t@2\t%3\tmessage one\t123\tinfo\t\n",
		nil,
		FilterOptions{
			Format: "simple",
			DisplayNames: appcore.DisplayNames{
				Sessions: map[string]string{"$1": "work"},
				Windows:  map[string]string{"@2": "editor"},
				Panes:    map[string]string{"%3": "shell"},
			},
		},
	)

	cols := splitSimpleColumns(t, strings.TrimSpace(output))
	assert.Equal(t, "work", cols[2])
	assert.Equal(t, "editor", cols[3])
	assert.Equal(t, "shell", cols[4])
}

func TestPrintListSimpleFormatUsesReadableFallbackWhenNamesUnavailable(t *testing.T) {
	output := runPrintList(t,
		"1\t2025-01-01T10:00:00Z\tactive\t$1\t@2\t%3\tmessage one\t123\tinfo\t\n",
		nil,
		FilterOptions{
			Format: "simple",
			DisplayNames: appcore.DisplayNames{
				Sessions: map[string]string{"$1": "work"},
			},
		},
	)

	cols := splitSimpleColumns(t, strings.TrimSpace(output))
	assert.Equal(t, "work", cols[2])
	assert.Equal(t, "stale-window:@2", cols[3])
	assert.Equal(t, "stale-pane:%3", cols[4])
}

func TestPrintListGroupCountUsesResolvedGroupNamesByDefault(t *testing.T) {
	output := runPrintList(t,
		"1\t2025-01-01T10:00:00Z\tactive\t$1\t@2\t%3\tone\t123\tinfo\t\n2\t2025-01-01T10:01:00Z\tactive\t$1\t@4\t%5\ttwo\t124\tinfo\t\n",
		nil,
		FilterOptions{
			Format:       "simple",
			GroupBy:      "session",
			GroupCount:   true,
			DisplayNames: appcore.DisplayNames{Sessions: map[string]string{"$1": "work"}},
		},
	)

	assert.Contains(t, output, "Group: work (2)")
	assert.NotContains(t, output, "Group: $1 (2)")
}

func TestPrintListGroupCountKeepsDistinctRawSessionGroupsWhenDisplayNamesCollide(t *testing.T) {
	output := runPrintList(t,
		"1\t2025-01-01T10:00:00Z\tactive\t$1\t@2\t%3\tone\t123\tinfo\t\n2\t2025-01-01T10:01:00Z\tactive\t$2\t@4\t%5\ttwo\t124\tinfo\t\n",
		nil,
		FilterOptions{
			Format:     "simple",
			GroupBy:    "session",
			GroupCount: true,
			DisplayNames: appcore.DisplayNames{
				Sessions: map[string]string{"$1": "work", "$2": "work"},
			},
		},
	)

	assert.Equal(t, 2, strings.Count(output, "Group: work (1)"))
}

func TestPrintListGroupCountKeepsDistinctRawWindowGroupsWhenDisplayNamesCollide(t *testing.T) {
	output := runPrintList(t,
		"1\t2025-01-01T10:00:00Z\tactive\t$1\t@2\t%3\tone\t123\tinfo\t\n2\t2025-01-01T10:01:00Z\tactive\t$9\t@4\t%5\ttwo\t124\tinfo\t\n",
		nil,
		FilterOptions{
			Format:     "simple",
			GroupBy:    "window",
			GroupCount: true,
			DisplayNames: appcore.DisplayNames{
				Windows: map[string]string{"@2": "editor", "@4": "editor"},
			},
		},
	)

	assert.Equal(t, 2, strings.Count(output, "Group: editor (1)"))
}

func TestPrintListJSONFormatRemainsRawIDs(t *testing.T) {
	output := runPrintList(t,
		"1\t2025-01-01T10:00:00Z\tactive\t$1\t@2\t%3\tmessage one\t123\tinfo\t\n",
		nil,
		FilterOptions{
			Format: "json",
			DisplayNames: appcore.DisplayNames{
				Sessions: map[string]string{"$1": "work"},
				Windows:  map[string]string{"@2": "editor"},
				Panes:    map[string]string{"%3": "shell"},
			},
		},
	)
	assert.Contains(t, output, `"Session": "$1"`)
	assert.Contains(t, output, `"Window": "@2"`)
	assert.Contains(t, output, `"Pane": "%3"`)
	assert.NotContains(t, output, `"Session": "work"`)
}

func TestNewListCmdPanicsWhenClientIsNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got nil")
		}

		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected panic message as string, got %T", r)
		}
		if !strings.Contains(msg, "client dependency cannot be nil") {
			t.Fatalf("expected panic message to mention nil dependency, got %q", msg)
		}
	}()

	NewListCmd(nil, defaultListSearchProvider, loadTmuxDisplayNames)
}

func TestListCmdFlagValidation(t *testing.T) {
	tests := []struct {
		name       string
		flagName   string
		flagValue  string
		wantErrMsg string
	}{
		{
			name:       "invalid group-by field",
			flagName:   "group-by",
			flagValue:  "invalid",
			wantErrMsg: "invalid group-by field",
		},
		{
			name:       "invalid filter value",
			flagName:   "filter",
			flagValue:  "invalid",
			wantErrMsg: "invalid filter value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeListClient{}
			cmd := NewListCmd(client, defaultListSearchProvider, func() appcore.DisplayNames { return appcore.DisplayNames{} })
			setFlag(t, cmd, tt.flagName, tt.flagValue)
			err := cmd.RunE(cmd, []string{})
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErrMsg, err.Error())
			}
		})
	}
}

func TestListCmdReusesLoadedDisplayNamesForSearchAndRendering(t *testing.T) {
	client := &fakeListClient{listNotificationsResult: "1\t2025-01-01T10:00:00Z\tactive\t$1\t@2\t%3\tmessage one\t123\tinfo\t\n"}
	loaderCalls := 0
	factoryCalls := 0
	cmd := NewListCmd(
		client,
		func(regex bool) search.Provider {
			factoryCalls++
			return search.NewSubstringProvider()
		},
		func() appcore.DisplayNames {
			loaderCalls++
			return appcore.DisplayNames{
				Sessions: map[string]string{"$1": "work"},
				Windows:  map[string]string{"@2": "editor"},
				Panes:    map[string]string{"%3": "shell"},
			}
		},
	)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	setFlag(t, cmd, "search", "work")
	setFlag(t, cmd, "format", "simple")

	err := cmd.RunE(cmd, []string{})
	assert.NoError(t, err)
	assert.Equal(t, 1, loaderCalls)
	assert.Equal(t, 0, factoryCalls)
	assert.Contains(t, buf.String(), "work")
	assert.Contains(t, buf.String(), "editor")
	assert.Contains(t, buf.String(), "shell")
}

func TestListCmdResolvesSessionFilterFromSessionName(t *testing.T) {
	client := &fakeListClient{listNotificationsResult: ""}
	loaderCalls := 0
	cmd := NewListCmd(
		client,
		defaultListSearchProvider,
		func() appcore.DisplayNames {
			loaderCalls++
			return appcore.DisplayNames{
				Sessions: map[string]string{"$1": "work"},
			}
		},
	)

	setFlag(t, cmd, "session", "work")
	err := cmd.RunE(cmd, []string{})
	assert.NoError(t, err)
	assert.Equal(t, 1, loaderCalls)
	if assert.Len(t, client.listNotificationsCalls, 1) {
		assert.Equal(t, "$1", client.listNotificationsCalls[0].sessionFilter)
	}
}

func TestListCmdResolvesWindowAndPaneFiltersFromNames(t *testing.T) {
	client := &fakeListClient{listNotificationsResult: ""}
	cmd := NewListCmd(
		client,
		defaultListSearchProvider,
		func() appcore.DisplayNames {
			return appcore.DisplayNames{
				Windows: map[string]string{"@2": "editor"},
				Panes:   map[string]string{"%3": "shell"},
			}
		},
	)

	setFlag(t, cmd, "window", "editor")
	setFlag(t, cmd, "pane", "shell")
	err := cmd.RunE(cmd, []string{})
	assert.NoError(t, err)
	if assert.Len(t, client.listNotificationsCalls, 1) {
		assert.Equal(t, "@2", client.listNotificationsCalls[0].windowFilter)
		assert.Equal(t, "%3", client.listNotificationsCalls[0].paneFilter)
	}
}

func TestListCmdRunEFlagCombinations(t *testing.T) {
	now := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	previousListNow := listNow
	listNow = func() time.Time { return now }
	defer func() { listNow = previousListNow }()
	tests := []struct {
		name           string
		flags          map[string]string
		wantState      string
		wantLevel      string
		wantSession    string
		wantWindow     string
		wantPane       string
		wantOlderThan  string // empty if not set
		wantNewerThan  string // empty if not set
		wantReadFilter string
	}{
		{
			name:      "default state active",
			flags:     map[string]string{},
			wantState: "active",
		},
		{
			name:      "flag dismissed",
			flags:     map[string]string{"dismissed": "true"},
			wantState: "dismissed",
		},
		{
			name:      "flag all",
			flags:     map[string]string{"all": "true"},
			wantState: "all",
		},
		{
			name:      "level filter",
			flags:     map[string]string{"level": "warning"},
			wantState: "active",
			wantLevel: "warning",
		},
		{
			name:        "session filter",
			flags:       map[string]string{"session": "sess1"},
			wantState:   "active",
			wantSession: "sess1",
		},
		{
			name:       "window filter",
			flags:      map[string]string{"window": "win2"},
			wantState:  "active",
			wantWindow: "win2",
		},
		{
			name:      "pane filter",
			flags:     map[string]string{"pane": "%0"},
			wantState: "active",
			wantPane:  "%0",
		},
		{
			name:          "older-than 7 days",
			flags:         map[string]string{"older-than": "7"},
			wantState:     "active",
			wantOlderThan: now.AddDate(0, 0, -7).Format("2006-01-02T15:04:05Z"),
		},
		{
			name:          "newer-than 2 days",
			flags:         map[string]string{"newer-than": "2"},
			wantState:     "active",
			wantNewerThan: now.AddDate(0, 0, -2).Format("2006-01-02T15:04:05Z"),
		},
		{
			name:           "filter read",
			flags:          map[string]string{"filter": "read"},
			wantState:      "active",
			wantReadFilter: "read",
		},
		{
			name:           "filter unread",
			flags:          map[string]string{"filter": "unread"},
			wantState:      "active",
			wantReadFilter: "unread",
		},
		{
			name:      "group-by message",
			flags:     map[string]string{"group-by": "message"},
			wantState: "active",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeListClient{
				listNotificationsResult: "",
			}
			cmd := NewListCmd(client, defaultListSearchProvider, func() appcore.DisplayNames { return appcore.DisplayNames{} })
			for flag, value := range tt.flags {
				setFlag(t, cmd, flag, value)
			}
			err := cmd.RunE(cmd, []string{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Verify exactly one call
			if len(client.listNotificationsCalls) != 1 {
				t.Fatalf("expected 1 call to ListNotifications, got %d", len(client.listNotificationsCalls))
			}
			call := client.listNotificationsCalls[0]
			assert.Equal(t, tt.wantState, call.stateFilter)
			assert.Equal(t, tt.wantLevel, call.levelFilter)
			assert.Equal(t, tt.wantSession, call.sessionFilter)
			assert.Equal(t, tt.wantWindow, call.windowFilter)
			assert.Equal(t, tt.wantPane, call.paneFilter)
			assert.Equal(t, tt.wantOlderThan, call.olderThan)
			assert.Equal(t, tt.wantNewerThan, call.newerThan)
			assert.Equal(t, tt.wantReadFilter, call.readFilter)
		})
	}
}

func TestListCmdRunEClientError(t *testing.T) {
	client := &fakeListClient{
		listNotificationsError: errors.New("storage error"),
	}
	cmd := NewListCmd(client, defaultListSearchProvider, func() appcore.DisplayNames { return appcore.DisplayNames{} })

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	// RunE returns nil because PrintList prints error to writer
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "list: failed to list notifications") {
		t.Fatalf("expected error message in output, got %q", output)
	}
}

func TestListCmdTabWithFilters(t *testing.T) {
	tests := []struct {
		name           string
		tab            string
		flags          map[string]string
		wantSession    string
		wantLevel      string
		wantWindow     string
		wantPane       string
		wantOlderThan  string
		wantNewerThan  string
		wantReadFilter string
	}{
		{
			name:        "sessions with session filter",
			tab:         "sessions",
			flags:       map[string]string{"session": "$1"},
			wantSession: "$1",
		},
		{
			name:      "sessions with level filter",
			tab:       "sessions",
			flags:     map[string]string{"level": "error"},
			wantLevel: "error",
		},
		{
			name:        "sessions with multiple filters",
			tab:         "sessions",
			flags:       map[string]string{"session": "$1", "level": "warning", "window": "win1"},
			wantSession: "$1",
			wantLevel:   "warning",
			wantWindow:  "win1",
		},
		{
			name:           "recents with session filter",
			tab:            "recents",
			flags:          map[string]string{"session": "$2"},
			wantSession:    "$2",
			wantReadFilter: "unread", // recents defaults to unread
		},
		{
			name:           "recents with custom read filter",
			tab:            "recents",
			flags:          map[string]string{"filter": "read"},
			wantReadFilter: "read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeListClient{
				listNotificationsResult: "",
			}
			cmd := NewListCmd(client, defaultListSearchProvider, func() appcore.DisplayNames { return appcore.DisplayNames{} })

			// Set tab flag
			setFlag(t, cmd, "tab", tt.tab)

			// Set filter flags
			for flagName, flagValue := range tt.flags {
				setFlag(t, cmd, flagName, flagValue)
			}

			err := cmd.RunE(cmd, []string{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify exactly one call
			if len(client.listNotificationsCalls) != 1 {
				t.Fatalf("expected 1 call to ListNotifications, got %d", len(client.listNotificationsCalls))
			}
			call := client.listNotificationsCalls[0]
			assert.Equal(t, tt.wantSession, call.sessionFilter, "session filter mismatch")
			assert.Equal(t, tt.wantLevel, call.levelFilter, "level filter mismatch")
			assert.Equal(t, tt.wantWindow, call.windowFilter, "window filter mismatch")
			assert.Equal(t, tt.wantPane, call.paneFilter, "pane filter mismatch")
			assert.Equal(t, tt.wantReadFilter, call.readFilter, "read filter mismatch")
		})
	}
}
