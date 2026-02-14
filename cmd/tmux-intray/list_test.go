package main

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
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

func setupMock() {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		// We'll apply basic filters here for simplicity.
		// In real tests we can filter based on parameters.
		return mockLines()
	}
}

func restoreMock() {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		// default to real storage (not used in tests)
		return ""
	}
}

func TestPrintListEmpty(t *testing.T) {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return ""
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{})
	output := buf.String()
	if output != "No notifications found\n" {
		t.Errorf("Expected 'No notifications found', got %q", output)
	}
}

func TestPrintListLegacyFormat(t *testing.T) {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{Format: "legacy"})
	output := buf.String()
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
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{Format: "simple"})
	output := buf.String()
	// Should contain ID, DATE, and message separator dash
	if !strings.Contains(output, "1") || !strings.Contains(output, "2025-01-01") {
		t.Error("Simple format missing ID or timestamp")
	}
	// Should contain separator dash
	if !strings.Contains(output, "-") {
		t.Error("Simple format missing separator dash")
	}
	// Should contain messages
	if !strings.Contains(output, "message one") || !strings.Contains(output, "message two") {
		t.Error("Simple format missing messages")
	}
	// Check line structure: should have one per notification
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}
}

func TestPrintListUnreadFirstOrdering(t *testing.T) {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{Format: "simple"})
	output := strings.TrimSpace(buf.String())

	var ids []int
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			t.Fatalf("failed to parse ID from line %q: %v", line, err)
		}
		ids = append(ids, id)
	}

	assert.Equal(t, []int{2, 4, 1, 3, 5}, ids)
}

func TestPrintListTableFormat(t *testing.T) {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{Format: "table"})
	output := buf.String()
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
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{Format: "compact"})
	output := buf.String()
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
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	// Substring search
	PrintList(FilterOptions{Search: "three", Format: "legacy"})
	output := buf.String()
	if !strings.Contains(output, "message three") {
		t.Error("Search filter didn't find 'message three'")
	}
	if strings.Contains(output, "message one") {
		t.Error("Search filter incorrectly included 'message one'")
	}
}

func TestPrintListRegexSearch(t *testing.T) {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	// Regex search for messages ending with 'e'
	PrintList(FilterOptions{Search: "e$", Regex: true, Format: "legacy"})
	output := buf.String()
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
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{GroupBy: "level", Format: "simple"})
	output := buf.String()
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
	// Should contain ID and timestamps in simple format
	if !strings.Contains(output, "-") {
		t.Error("Simple format missing separator dash")
	}
}

func TestPrintListGroupByLevel(t *testing.T) {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{GroupBy: "level", Format: "legacy"})
	output := buf.String()
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

func TestPrintListGroupCount(t *testing.T) {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{GroupBy: "level", GroupCount: true})
	output := buf.String()
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
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\terror message\t123\terror\n" +
			"2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\twarning message\t124\twarning\n"
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	// Test with custom search provider that only matches "error"
	mockProvider := search.NewSubstringProvider(search.WithFields([]string{"level"}))

	opts := FilterOptions{
		Search:         "error",
		SearchProvider: mockProvider,
		Format:         "legacy",
	}
	PrintList(opts)
	output := buf.String()

	// Should only match the first notification (has error in level)
	if !strings.Contains(output, "error message") {
		t.Error("Missing error message")
	}
	if strings.Contains(output, "warning message") {
		t.Error("Should not include warning message")
	}

	buf.Reset()

	// Test with provider that matches warning
	opts.Search = "warning"
	PrintList(opts)
	output = buf.String()
	if strings.Contains(output, "error message") {
		t.Error("Should not include error message")
	}
	if !strings.Contains(output, "warning message") {
		t.Error("Missing warning message")
	}
}

func TestPrintListBackwardCompatibility(t *testing.T) {
	// Verify that existing behavior is preserved when no custom provider is set

	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tHello World\t123\tinfo\n"
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	// Test substring search (default, no regex flag)
	opts := FilterOptions{
		Search: "Hello",
		Regex:  false,
		Format: "legacy",
	}
	PrintList(opts)
	if !strings.Contains(buf.String(), "Hello World") {
		t.Error("Substring search should find Hello World")
	}

	buf.Reset()

	// Test regex search (with regex flag)
	opts.Regex = true
	PrintList(opts)
	if !strings.Contains(buf.String(), "Hello World") {
		t.Error("Regex search should find Hello World")
	}

	buf.Reset()

	// Test no search (should show all)
	opts.Search = ""
	PrintList(opts)
	if !strings.Contains(buf.String(), "Hello World") {
		t.Error("Empty search should show all notifications")
	}
}

func TestPrintListFilterRead(t *testing.T) {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		// Only return read notifications (those with non-empty read_timestamp)
		// Using mockLines which has read timestamps for IDs 1, 3, 5
		lines := ""
		if readFilter == "read" {
			lines = "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage one\t123\tinfo\t2025-01-01T10:05:00Z\n" +
				"3\t2025-01-01T12:00:00Z\tdismissed\tsess2\twin2\tpane3\tmessage three\t125\terror\t2025-01-01T12:05:00Z\n" +
				"5\t2025-01-01T14:00:00Z\tactive\tsess3\twin3\tpane5\tmessage five\t127\tinfo\t2025-01-01T14:05:00Z"
		}
		return lines
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	// Test --filter read
	PrintList(FilterOptions{ReadFilter: "read", Format: "simple"})
	output := buf.String()

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
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		// Only return unread notifications (those with empty read_timestamp)
		// Using mockLines which has no read timestamps for IDs 2, 4
		lines := ""
		if readFilter == "unread" {
			lines = "2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\tmessage two\t124\twarning\t\n" +
				"4\t2025-01-01T13:00:00Z\tactive\tsess2\twin2\tpane4\tmessage four\t126\tinfo\t"
		}
		return lines
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	// Test --filter unread
	PrintList(FilterOptions{ReadFilter: "unread", Format: "simple"})
	output := buf.String()

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
	empty := []notification.Notification{}
	result := orderUnreadFirst(notification.ToDomainSliceUnsafe(empty))
	assert.Equal(t, 0, len(result))

	// Single unread notification (no read timestamp)
	n1 := notification.Notification{ID: 1, ReadTimestamp: ""}
	single := []notification.Notification{n1}
	result = orderUnreadFirst(notification.ToDomainSliceUnsafe(single))
	assert.Equal(t, notification.ToDomainSliceUnsafe(single), result)

	// Single read notification
	n2 := notification.Notification{ID: 2, ReadTimestamp: "2025-01-01T10:00:00Z"}
	singleRead := []notification.Notification{n2}
	result = orderUnreadFirst(notification.ToDomainSliceUnsafe(singleRead))
	assert.Equal(t, notification.ToDomainSliceUnsafe(singleRead), result)

	// Mixed: unread should come before read
	notifs := []notification.Notification{
		{ID: 1, ReadTimestamp: "2025-01-01T10:00:00Z"}, // read
		{ID: 2, ReadTimestamp: ""},                     // unread
		{ID: 3, ReadTimestamp: "2025-01-01T11:00:00Z"}, // read
		{ID: 4, ReadTimestamp: ""},                     // unread
	}
	result = orderUnreadFirst(notification.ToDomainSliceUnsafe(notifs))
	// Expect unread IDs 2,4 first, then read IDs 1,3 preserving relative order
	expected := []notification.Notification{notifs[1], notifs[3], notifs[0], notifs[2]}
	assert.Equal(t, notification.ToDomainSliceUnsafe(expected), result)
}

func TestGroupNotifications(t *testing.T) {
	notifs := []notification.Notification{
		{ID: 1, Session: "sess1", Window: "win1", Pane: "pane1", Level: "info"},
		{ID: 2, Session: "sess1", Window: "win2", Pane: "pane2", Level: "warning"},
		{ID: 3, Session: "sess2", Window: "win1", Pane: "pane1", Level: "info"},
	}
	domainNotifs := notification.ToDomainSliceUnsafe(notifs)
	// Convert to values for domain.GroupNotifications
	values := notificationsToValues(domainNotifs)

	// Group by session
	result := domain.GroupNotifications(values, domain.GroupBySession)
	assert.Equal(t, 2, len(result.Groups))
	// Find groups by display name (should be sess1, sess2)
	var sess1Group, sess2Group domain.Group
	for _, g := range result.Groups {
		if g.DisplayName == "sess1" {
			sess1Group = g
		} else if g.DisplayName == "sess2" {
			sess2Group = g
		}
	}
	assert.Equal(t, 2, sess1Group.Count)
	assert.Equal(t, 1, sess2Group.Count)

	// Group by window
	result = domain.GroupNotifications(values, domain.GroupByWindow)
	assert.Equal(t, 3, len(result.Groups)) // sess1/win1, sess1/win2, sess2/win1
	// Check total count
	assert.Equal(t, 3, result.TotalCount)

	// Group by pane
	result = domain.GroupNotifications(values, domain.GroupByPane)
	assert.Equal(t, 3, len(result.Groups)) // sess1/win1/pane1, sess1/win2/pane2, sess2/win1/pane1
	assert.Equal(t, 3, result.TotalCount)

	// Group by level
	result = domain.GroupNotifications(values, domain.GroupByLevel)
	assert.Equal(t, 2, len(result.Groups)) // info, warning
	var infoGroup, warningGroup domain.Group
	for _, g := range result.Groups {
		if g.DisplayName == "info" {
			infoGroup = g
		} else if g.DisplayName == "warning" {
			warningGroup = g
		}
	}
	assert.Equal(t, 2, infoGroup.Count)
	assert.Equal(t, 1, warningGroup.Count)

	// Unknown field defaults to GroupByNone (empty groups)
	result = domain.GroupNotifications(values, domain.GroupByMode("unknown"))
	assert.Equal(t, 0, len(result.Groups))
	assert.Equal(t, 3, result.TotalCount)
}

func TestPrintFunctionsWithEmptySlice(t *testing.T) {
	var buf bytes.Buffer
	// printSimple
	printSimple(notification.ToDomainSliceUnsafe([]notification.Notification{}), &buf)
	assert.Equal(t, "", buf.String())
	buf.Reset()
	// printTable
	printTable(notification.ToDomainSliceUnsafe([]notification.Notification{}), &buf)
	assert.Equal(t, "", buf.String())
	buf.Reset()
	// printCompact
	printCompact(notification.ToDomainSliceUnsafe([]notification.Notification{}), &buf)
	assert.Equal(t, "", buf.String())
	buf.Reset()
	// printLegacy
	printLegacy(notification.ToDomainSliceUnsafe([]notification.Notification{}), &buf)
	assert.Equal(t, "", buf.String())
}

func TestPrintFunctionsWithSingleNotification(t *testing.T) {
	notif := notification.Notification{
		ID:        42,
		Timestamp: "2025-01-01T10:00:00Z",
		Message:   "test message",
	}
	var buf bytes.Buffer
	printSimple(notification.ToDomainSliceUnsafe([]notification.Notification{notif}), &buf)
	assert.Contains(t, buf.String(), "42")
	assert.Contains(t, buf.String(), "test message")
	buf.Reset()
	printTable(notification.ToDomainSliceUnsafe([]notification.Notification{notif}), &buf)
	assert.Contains(t, buf.String(), "42")
	buf.Reset()
	printCompact(notification.ToDomainSliceUnsafe([]notification.Notification{notif}), &buf)
	assert.Contains(t, buf.String(), "test message")
	buf.Reset()
	printLegacy(notification.ToDomainSliceUnsafe([]notification.Notification{notif}), &buf)
	assert.Equal(t, "test message\n", buf.String())
}

func TestPrintListJSONFormat(t *testing.T) {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{Format: "json"})
	output := buf.String()
	if !strings.Contains(output, "JSON format not yet implemented") {
		t.Errorf("Expected JSON not implemented message, got %q", output)
	}
}

func TestPrintListUnknownFormat(t *testing.T) {
	listListFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) string {
		return mockLines()
	}
	defer restoreMock()

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

	PrintList(FilterOptions{Format: "unknown"})
	output := buf.String()
	if !strings.Contains(output, "list: unknown format") {
		t.Errorf("Expected unknown format error, got %q", output)
	}
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

	NewListCmd(nil)
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
			cmd := NewListCmd(client)
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

func TestListCmdRunEFlagCombinations(t *testing.T) {
	now := time.Now().UTC()
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeListClient{
				listNotificationsResult: "",
			}
			cmd := NewListCmd(client)
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
	cmd := NewListCmd(client)

	var buf bytes.Buffer
	listOutputWriter = &buf
	defer func() { listOutputWriter = nil }()

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
