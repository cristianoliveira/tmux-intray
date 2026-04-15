package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

func TestPrintRecentsSimpleIncludesID(t *testing.T) {
	notifs := []notification.Notification{
		{ID: 42, Session: "$1", Level: "error", Timestamp: time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339), Message: "boom"},
	}

	var buf bytes.Buffer
	printRecentsSimple(notifs, &buf)

	out := buf.String()
	if !strings.Contains(out, "#42") {
		t.Fatalf("expected output to include notification id, got:\n%s", out)
	}
}

func TestPrintRecentsTableIncludesIDColumn(t *testing.T) {
	notifs := []notification.Notification{
		{ID: 7, Session: "$1", Level: "info", Timestamp: time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339), Message: "hello"},
	}

	var buf bytes.Buffer
	printRecentsTable(notifs, &buf)
	out := buf.String()

	for _, want := range []string{"ID", "7"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPrintRecentsJSONIncludesIDField(t *testing.T) {
	notifs := []notification.Notification{
		{ID: 99, Session: "$1", Level: "warning", Timestamp: time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339), Message: "warn"},
	}

	var buf bytes.Buffer
	printRecentsJSON(notifs, &buf)

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("failed to parse json output: %v\nraw:\n%s", err, buf.String())
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 json item, got %d", len(got))
	}
	if got[0]["id"] != float64(99) { // encoding/json uses float64 for numbers in map[string]any
		t.Fatalf("expected json to include id=99, got: %#v", got[0])
	}
}
