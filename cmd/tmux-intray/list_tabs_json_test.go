package main

import (
	"bytes"
	"encoding/json"
	"testing"

	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/format"
)

func TestFormatTabsUsingListFormatterJSONIncludesIDField(t *testing.T) {
	groups := []domain.SessionNotification{
		{Session: "$1", Notification: domain.Notification{ID: 123, Message: "m", Level: domain.LevelInfo, Timestamp: "2024-01-01T10:00:00Z"}},
	}

	var buf bytes.Buffer
	formatTabsUsingListFormatter(groups, format.FormatterTypeJSON, appcore.DisplayNames{}, false, false, &buf)

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("failed to parse json output: %v\nraw:\n%s", err, buf.String())
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 json item, got %d", len(got))
	}
	if got[0]["ID"] != float64(123) {
		t.Fatalf("expected json to include ID=123, got: %#v", got[0])
	}
}
