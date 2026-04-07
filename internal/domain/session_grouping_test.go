package domain

import (
	"testing"
	"time"
)

func TestGroupBySessionKeepMostRecent(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	earlier := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	muchEarlier := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name     string
		input    []Notification
		expected []string // Expected session order (most recent first)
		wantLen  int
	}{
		{
			name:     "empty input returns nil",
			input:    []Notification{},
			expected: nil,
			wantLen:  0,
		},
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: nil,
			wantLen:  0,
		},
		{
			name: "single session returns one",
			input: []Notification{
				{Session: "dev", Timestamp: now, Message: "msg1"},
			},
			expected: []string{"dev"},
			wantLen:  1,
		},
		{
			name: "multiple sessions returns most recent per session",
			input: []Notification{
				{Session: "dev", Timestamp: now, Message: "dev newest"},
				{Session: "dev", Timestamp: earlier, Message: "dev older"},
				{Session: "prod", Timestamp: now, Message: "prod msg"},
			},
			expected: []string{"dev", "prod"},
			wantLen:  2,
		},
		{
			name: "skips notifications without session",
			input: []Notification{
				{Session: "dev", Timestamp: now, Message: "dev msg"},
				{Session: "", Timestamp: now, Message: "no session"},
				{Session: "prod", Timestamp: earlier, Message: "prod msg"},
			},
			expected: []string{"dev", "prod"},
			wantLen:  2,
		},
		{
			name: "sorts by timestamp descending",
			input: []Notification{
				{Session: "z-session", Timestamp: muchEarlier, Message: "oldest"},
				{Session: "a-session", Timestamp: now, Message: "newest"},
				{Session: "m-session", Timestamp: earlier, Message: "middle"},
			},
			expected: []string{"a-session", "m-session", "z-session"},
			wantLen:  3,
		},
		{
			name: "overwrites when newer timestamp found",
			input: []Notification{
				{Session: "dev", Timestamp: earlier, Message: "old"},
				{Session: "dev", Timestamp: muchEarlier, Message: "older"},
				{Session: "dev", Timestamp: now, Message: "newest"},
			},
			expected: []string{"dev"},
			wantLen:  1,
		},
		{
			name: "equal timestamps keeps first seen",
			input: []Notification{
				{Session: "dev", Timestamp: now, Message: "first"},
				{Session: "dev", Timestamp: now, Message: "second"},
			},
			expected: []string{"dev"},
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GroupBySessionKeepMostRecent(tt.input)

			if tt.wantLen == 0 {
				if len(result) > 0 {
					t.Errorf("expected nil or empty, got %d items", len(result))
				}
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("expected %d sessions, got %d", tt.wantLen, len(result))
				return
			}

			for i, expectedSession := range tt.expected {
				if result[i].Session != expectedSession {
					t.Errorf("position %d: expected session %q, got %q", i, expectedSession, result[i].Session)
				}
			}
		})
	}
}

func TestGroupBySessionKeepMostRecent_MostRecentPerSession(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	earlier := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)

	// Setup: dev session has newer notification, prod has older
	input := []Notification{
		{Session: "dev", Timestamp: now, Message: "dev latest"},
		{Session: "dev", Timestamp: earlier, Message: "dev old"},
		{Session: "prod", Timestamp: earlier, Message: "prod only"},
	}

	result := GroupBySessionKeepMostRecent(input)

	if len(result) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(result))
	}

	// dev should have the newest message
	devResult := findBySession(result, "dev")
	if devResult == nil {
		t.Fatal("dev session not found")
	}
	if devResult.Notification.Message != "dev latest" {
		t.Errorf("dev: expected 'dev latest', got %q", devResult.Notification.Message)
	}

	// prod should have its only message
	prodResult := findBySession(result, "prod")
	if prodResult == nil {
		t.Fatal("prod session not found")
	}
	if prodResult.Notification.Message != "prod only" {
		t.Errorf("prod: expected 'prod only', got %q", prodResult.Notification.Message)
	}
}

func findBySession(sessions []SessionNotification, session string) *SessionNotification {
	for i := range sessions {
		if sessions[i].Session == session {
			return &sessions[i]
		}
	}
	return nil
}
