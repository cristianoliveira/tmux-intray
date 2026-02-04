package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// statusMockLines returns a fixed TSV string for testing.
func statusMockLines() string {
	return `1	2025-01-01T10:00:00Z	active	sess1	win1	pane1	message one	123	info
2	2025-01-01T11:00:00Z	active	sess1	win1	pane2	message two	124	warning
3	2025-01-01T12:00:00Z	dismissed	sess2	win2	pane3	message three	125	error
4	2025-01-01T13:00:00Z	active	sess2	win2	pane4	message four	126	info
5	2025-01-01T14:00:00Z	active	sess3	win3	pane5	message five	127	critical`
}

func statusSetupMock() {
	statusListFunc = func(state, level, session, window, pane, olderThan, newerThan string) string {
		// Simple filtering for testing
		lines := strings.Split(statusMockLines(), "\n")
		var filtered []string
		for _, line := range lines {
			if line == "" {
				continue
			}
			fields := strings.Split(line, "\t")
			if state != "" && state != "all" && fields[2] != state {
				continue
			}
			if level != "" && fields[8] != level {
				continue
			}
			// ignore other filters for simplicity
			filtered = append(filtered, line)
		}
		return strings.Join(filtered, "\n")
	}
	statusActiveCountFunc = func() int {
		// Count active lines
		count := 0
		for _, line := range strings.Split(statusMockLines(), "\n") {
			if line == "" {
				continue
			}
			fields := strings.Split(line, "\t")
			if fields[2] == "active" {
				count++
			}
		}
		return count
	}
}

func statusRestoreMock() {
	statusListFunc = func(state, level, session, window, pane, olderThan, newerThan string) string {
		// default to real storage (not used in tests)
		return ""
	}
	statusActiveCountFunc = func() int {
		return 0
	}
}

func TestPrintStatusEmpty(t *testing.T) {
	statusListFunc = func(state, level, session, window, pane, olderThan, newerThan string) string {
		return ""
	}
	defer statusRestoreMock()

	var buf bytes.Buffer
	statusOutputWriter = &buf
	defer func() { statusOutputWriter = nil }()

	PrintStatus("summary")
	output := buf.String()
	expected := "No notifications"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain %q, got %q", expected, output)
	}
}

func TestPrintStatusSummary(t *testing.T) {
	statusSetupMock()
	defer statusRestoreMock()

	var buf bytes.Buffer
	statusOutputWriter = &buf
	defer func() { statusOutputWriter = nil }()

	PrintStatus("summary")
	output := buf.String()
	// Should contain active, dismissed, total counts
	if !strings.Contains(output, "Active notifications: 4") {
		t.Errorf("Expected active count 4, got %q", output)
	}
	if !strings.Contains(output, "Dismissed notifications: 1") {
		t.Errorf("Expected dismissed count 1, got %q", output)
	}
	if !strings.Contains(output, "Total notifications: 5") {
		t.Errorf("Expected total count 5, got %q", output)
	}
	// Should contain level breakdown (only active notifications)
	if !strings.Contains(output, "info: 2") {
		t.Errorf("Expected info count 2, got %q", output)
	}
	if !strings.Contains(output, "warning: 1") {
		t.Errorf("Expected warning count 1, got %q", output)
	}
	if !strings.Contains(output, "error: 0") {
		t.Errorf("Expected error count 0, got %q", output)
	}
	if !strings.Contains(output, "critical: 1") {
		t.Errorf("Expected critical count 1, got %q", output)
	}
}

func TestPrintStatusLevels(t *testing.T) {
	statusSetupMock()
	defer statusRestoreMock()

	var buf bytes.Buffer
	statusOutputWriter = &buf
	defer func() { statusOutputWriter = nil }()

	PrintStatus("levels")
	output := buf.String()
	// Should contain each level line (only active notifications)
	expectedLines := []string{
		"info:2",
		"warning:1",
		"error:0",
		"critical:1",
	}
	for _, exp := range expectedLines {
		if !strings.Contains(output, exp) {
			t.Errorf("Missing line %q in output %q", exp, output)
		}
	}
}

func TestPrintStatusPanes(t *testing.T) {
	statusSetupMock()
	defer statusRestoreMock()

	var buf bytes.Buffer
	statusOutputWriter = &buf
	defer func() { statusOutputWriter = nil }()

	PrintStatus("panes")
	output := buf.String()
	// Should contain pane keys and counts
	// pane keys are sess1:win1:pane1, sess1:win1:pane2, sess2:win2:pane3, sess2:win2:pane4, sess3:win3:pane5
	// counts: pane1 1, pane2 1, pane3 1, pane4 1, pane5 1 (but pane3 is dismissed, filtered out)
	// active only, so dismissed pane3 excluded
	expectedCounts := map[string]int{
		"sess1:win1:pane1": 1,
		"sess1:win1:pane2": 1,
		"sess2:win2:pane4": 1,
		"sess3:win3:pane5": 1,
	}
	for pane, count := range expectedCounts {
		line := fmt.Sprintf("%s:%d", pane, count)
		if !strings.Contains(output, line) {
			t.Errorf("Missing pane line %q in output %q", line, output)
		}
	}
	// Ensure dismissed pane not present
	if strings.Contains(output, "sess2:win2:pane3") {
		t.Errorf("Dismissed pane should not appear in panes output")
	}
}

func TestPrintStatusJSON(t *testing.T) {
	statusSetupMock()
	defer statusRestoreMock()

	var buf bytes.Buffer
	statusOutputWriter = &buf
	defer func() { statusOutputWriter = nil }()

	PrintStatus("json")
	output := buf.String()
	expected := "JSON format not yet implemented"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected JSON placeholder, got %q", output)
	}
}

func TestPrintStatusUnknownFormat(t *testing.T) {
	statusSetupMock()
	defer statusRestoreMock()

	var buf bytes.Buffer
	statusOutputWriter = &buf
	defer func() { statusOutputWriter = nil }()

	PrintStatus("unknown")
	output := buf.String()
	expected := "Unknown format: unknown"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected unknown format error, got %q", output)
	}
}
