package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/search"
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
