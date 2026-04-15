package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
)

func TestPrintTabsSimple(t *testing.T) {
	tests := []struct {
		name         string
		groups       []domain.SessionNotification
		wantContains []string
	}{
		{
			name:   "empty groups shows only header",
			groups: []domain.SessionNotification{},
			wantContains: []string{
				"Sessions (0)",
			},
		},
		{
			name: "single session",
			groups: []domain.SessionNotification{
				{Session: "dev", Notification: domain.Notification{
					ID:        42,
					Message:   "Test message",
					Level:     domain.LevelInfo,
					Timestamp: "2024-01-01T10:00:00Z",
				}},
			},
			wantContains: []string{
				"Sessions (1)",
				"dev",
				"#42",
				"Test message",
			},
		},
		{
			name: "multiple sessions sorted by recency",
			groups: []domain.SessionNotification{
				{Session: "prod", Notification: domain.Notification{
					ID:        1,
					Message:   "prod message",
					Level:     domain.LevelError,
					Timestamp: "2024-01-01T12:00:00Z",
				}},
				{Session: "dev", Notification: domain.Notification{
					ID:        2,
					Message:   "dev message",
					Level:     domain.LevelWarning,
					Timestamp: "2024-01-01T11:00:00Z",
				}},
			},
			wantContains: []string{
				"Sessions (2)",
				"dev",
				"prod",
				"#1",
				"#2",
				"dev message",
				"prod message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printTabsSimple(tt.groups, &buf)
			output := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}

func TestPrintTabsTable(t *testing.T) {
	tests := []struct {
		name         string
		groups       []domain.SessionNotification
		wantContains []string
	}{
		{
			name:   "empty groups shows only header",
			groups: []domain.SessionNotification{},
			wantContains: []string{
				"Sessions (0)",
			},
		},
		{
			name: "single session in table format",
			groups: []domain.SessionNotification{
				{Session: "dev", Notification: domain.Notification{
					ID:        7,
					Message:   "Test message",
					Level:     domain.LevelInfo,
					Timestamp: "2024-01-01T10:00:00Z",
				}},
			},
			wantContains: []string{
				"Sessions (1)",
				"dev",
				"Test message",
				"Num",
				"ID",
				"Session",
				"Level",
				"7",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printTabsTable(tt.groups, &buf)
			output := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, output)
				}
			}
		})
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
