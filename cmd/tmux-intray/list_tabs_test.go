package main

import (
	"bytes"
	"strings"
	"testing"

	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

func TestFormatTabsUsingListFormatterSimple(t *testing.T) {
	groups := []domain.SessionNotification{
		{Session: "dev", Notification: domain.Notification{ID: 42, Timestamp: "2024-01-01T10:00:00Z", Message: "Test message"}},
	}

	var buf bytes.Buffer
	formatTabsUsingListFormatter(groups, format.FormatterTypeSimple, appcore.DisplayNames{}, true, false, &buf)

	out := buf.String()
	if !strings.HasPrefix(out, "42") {
		t.Fatalf("expected output to start with notification id, got: %q", out)
	}

	cols := splitSimpleColumns(t, strings.TrimSpace(out))
	if got := cols[6]; got != "Test message" {
		t.Fatalf("expected message column to be %q, got: %q (full=%q)", "Test message", got, out)
	}
}

func TestFormatTabsUsingListFormatterTable(t *testing.T) {
	groups := []domain.SessionNotification{
		{Session: "dev", Notification: domain.Notification{ID: 7, Timestamp: "2024-01-01T10:00:00Z", Message: "Test message"}},
	}

	var buf bytes.Buffer
	formatTabsUsingListFormatter(groups, format.FormatterTypeTable, appcore.DisplayNames{}, false, false, &buf)

	out := buf.String()
	for _, want := range []string{"ID", "DATE", "7", "Test message"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestFormatTabsUsingListFormatterEmpty(t *testing.T) {
	var buf bytes.Buffer
	formatTabsUsingListFormatter(nil, format.FormatterTypeSimple, appcore.DisplayNames{}, false, false, &buf)
	if buf.Len() != 0 {
		t.Fatalf("expected no output for empty groups, got: %q", buf.String())
	}
}

func TestFormatTabsUsingListFormatterSimpleUsesResolvedNamesByDefault(t *testing.T) {
	groups := []domain.SessionNotification{{Session: "$1", Notification: domain.Notification{ID: 42, Timestamp: "2024-01-01T10:00:00Z", Session: "$1", Window: "@2", Pane: "%3", Message: "Test message"}}}

	var buf bytes.Buffer
	formatTabsUsingListFormatter(groups, format.FormatterTypeSimple, appcore.DisplayNames{
		Sessions: map[string]string{"$1": "work"},
		Windows:  map[string]string{"@2": "editor"},
		Panes:    map[string]string{"%3": "shell"},
	}, false, false, &buf)

	cols := splitSimpleColumns(t, strings.TrimSpace(buf.String()))
	if cols[2] != "work" || cols[3] != "editor" || cols[4] != "shell" {
		t.Fatalf("expected resolved names, got %v", cols)
	}
}

func TestFormatTabsUsingListFormatterSimpleOmitsRowsWhenNamesCannotBeResolved(t *testing.T) {
	groups := []domain.SessionNotification{{Session: "$1", Notification: domain.Notification{ID: 42, Timestamp: "2024-01-01T10:00:00Z", Session: "$1", Window: "@2", Pane: "%3", Message: "Test message"}}}

	var buf bytes.Buffer
	formatTabsUsingListFormatter(groups, format.FormatterTypeSimple, appcore.DisplayNames{
		Sessions: map[string]string{"$1": "work"},
		Windows:  map[string]string{},
		Panes:    map[string]string{},
	}, false, false, &buf)

	if strings.TrimSpace(buf.String()) != "" {
		t.Fatalf("expected unresolved row to be omitted, got %q", buf.String())
	}
}

func TestFormatTabsUsingListFormatterSimpleKeepsRawIDsWithExplicitFlag(t *testing.T) {
	groups := []domain.SessionNotification{{Session: "$1", Notification: domain.Notification{ID: 42, Timestamp: "2024-01-01T10:00:00Z", Session: "$1", Window: "@2", Pane: "%3", Message: "Test message"}}}

	var buf bytes.Buffer
	formatTabsUsingListFormatter(groups, format.FormatterTypeSimple, appcore.DisplayNames{
		Sessions: map[string]string{"$1": "work"},
		Windows:  map[string]string{"@2": "editor"},
		Panes:    map[string]string{"%3": "shell"},
	}, true, false, &buf)

	cols := splitSimpleColumns(t, strings.TrimSpace(buf.String()))
	if cols[2] != "$1" || cols[3] != "@2" || cols[4] != "%3" {
		t.Fatalf("expected raw ids, got %v", cols)
	}
}

func TestGroupBySession(t *testing.T) {
	// Test the wrapper function in CLI
	notifications := []notification.Notification{
		{Session: "dev", Timestamp: "2024-01-01T12:00:00Z", Message: "newer"},
		{Session: "dev", Timestamp: "2024-01-01T11:00:00Z", Message: "older"},
		{Session: "prod", Timestamp: "2024-01-01T10:00:00Z", Message: "prod msg"},
	}

	result := groupBySession(notifications)

	if len(result) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(result))
	}

	// Should be sorted by timestamp desc (dev first)
	if len(result) >= 1 && result[0].Session != "dev" {
		t.Errorf("expected dev first, got %s", result[0].Session)
	}
	if len(result) >= 2 && result[1].Session != "prod" {
		t.Errorf("expected prod second, got %s", result[1].Session)
	}
}
