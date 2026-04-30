package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
	"github.com/cristianoliveira/tmux-intray/internal/format"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

func TestFormatRecentsUsingListFormatterSimpleMatchesListStyle(t *testing.T) {
	notifs := []notification.Notification{
		{ID: 42, Level: "error", Timestamp: time.Now().UTC().Format(time.RFC3339), Message: "boom"},
	}

	var buf bytes.Buffer
	formatRecentsUsingListFormatter(notifs, format.FormatterTypeSimple, appcore.DisplayNames{}, true, false, &buf)

	out := buf.String()
	// simple format starts with the numeric ID
	if !strings.HasPrefix(out, "42") {
		t.Fatalf("expected output to start with notification id, got: %q", out)
	}

	// Output is column-based (same as internal/format SimpleFormatter)
	cols := splitSimpleColumns(t, strings.TrimSpace(out))
	if got := cols[6]; got != "boom" {
		t.Fatalf("expected message column to be %q, got: %q (full=%q)", "boom", got, out)
	}
}

func TestFormatRecentsUsingListFormatterSimpleUsesResolvedNamesByDefault(t *testing.T) {
	notifs := []notification.Notification{{ID: 42, Level: "error", Timestamp: time.Now().UTC().Format(time.RFC3339), Session: "$1", Window: "@2", Pane: "%3", Message: "boom"}}

	var buf bytes.Buffer
	formatRecentsUsingListFormatter(notifs, format.FormatterTypeSimple, appcore.DisplayNames{
		Sessions: map[string]string{"$1": "work"},
		Windows:  map[string]string{"@2": "editor"},
		Panes:    map[string]string{"%3": "shell"},
	}, false, false, &buf)

	cols := splitSimpleColumns(t, strings.TrimSpace(buf.String()))
	if cols[2] != "work" || cols[3] != "editor" || cols[4] != "shell" {
		t.Fatalf("expected resolved names, got %v", cols)
	}
}

func TestFormatRecentsUsingListFormatterSimpleOmitsRowsWhenNamesCannotBeResolved(t *testing.T) {
	notifs := []notification.Notification{{ID: 42, Level: "error", Timestamp: time.Now().UTC().Format(time.RFC3339), Session: "$1", Window: "@2", Pane: "%3", Message: "boom"}}

	var buf bytes.Buffer
	formatRecentsUsingListFormatter(notifs, format.FormatterTypeSimple, appcore.DisplayNames{
		Sessions: map[string]string{"$1": "work"},
		Windows:  map[string]string{},
		Panes:    map[string]string{},
	}, false, false, &buf)

	if strings.TrimSpace(buf.String()) != "" {
		t.Fatalf("expected unresolved row to be omitted, got: %q", buf.String())
	}
}

func TestFormatRecentsUsingListFormatterSimpleKeepsRowsWithRawIDsFlag(t *testing.T) {
	notifs := []notification.Notification{{ID: 42, Level: "error", Timestamp: time.Now().UTC().Format(time.RFC3339), Session: "$1", Window: "@2", Pane: "%3", Message: "boom"}}

	var buf bytes.Buffer
	formatRecentsUsingListFormatter(notifs, format.FormatterTypeSimple, appcore.DisplayNames{
		Sessions: map[string]string{"$1": "work"},
	}, true, false, &buf)

	cols := splitSimpleColumns(t, strings.TrimSpace(buf.String()))
	if cols[2] != "$1" || cols[3] != "@2" || cols[4] != "%3" {
		t.Fatalf("expected raw ids when --ids is set, got %v", cols)
	}
}

func TestFormatRecentsUsingListFormatterJSONIncludesID(t *testing.T) {
	notifs := []notification.Notification{
		{ID: 99, Level: "warning", Timestamp: time.Now().UTC().Format(time.RFC3339), Message: "warn"},
	}

	var buf bytes.Buffer
	formatRecentsUsingListFormatter(notifs, format.FormatterTypeJSON, appcore.DisplayNames{}, false, false, &buf)

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("failed to parse json output: %v\nraw:\n%s", err, buf.String())
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 json item, got %d", len(got))
	}
	if got[0]["ID"] != float64(99) {
		t.Fatalf("expected json to include ID=99, got: %#v", got[0])
	}
}
