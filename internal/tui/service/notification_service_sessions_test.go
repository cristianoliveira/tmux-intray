package service

import (
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
)

// TestSelectDataset_SessionsTab_IncludesAllNotifications verifies that the Sessions tab
// lists unique sessions from ALL notifications (including dismissed/read), not just active ones.
// This is a regression test for the bug where dismissed sessions were not shown.
func TestSelectDataset_SessionsTab_IncludesAllNotifications(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	earlier := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name          string
		notifications []notification.Notification
		wantLen       int
		wantSessions  []string
	}{
		{
			name: "includes dismissed notifications in session list",
			notifications: []notification.Notification{
				{ID: 1, Session: "dev", State: "active", Timestamp: now, Message: "active msg"},
				{ID: 2, Session: "prod", State: "dismissed", Timestamp: earlier, Message: "dismissed msg"},
			},
			wantLen:      2,
			wantSessions: []string{"dev", "prod"},
		},
		{
			name: "includes read notifications in session list",
			notifications: []notification.Notification{
				{ID: 1, Session: "dev", State: "active", Timestamp: now, Message: "active msg", ReadTimestamp: now},
				{ID: 2, Session: "prod", State: "active", Timestamp: earlier, Message: "read msg", ReadTimestamp: earlier},
			},
			wantLen:      2,
			wantSessions: []string{"dev", "prod"},
		},
		{
			name: "mix of active, dismissed, and read shows all unique sessions",
			notifications: []notification.Notification{
				{ID: 1, Session: "dev", State: "active", Timestamp: now, Message: "dev active"},
				{ID: 2, Session: "staging", State: "dismissed", Timestamp: earlier, Message: "staging dismissed"},
				{ID: 3, Session: "prod", State: "active", Timestamp: earlier, Message: "prod read", ReadTimestamp: earlier},
			},
			wantLen:      3,
			wantSessions: []string{"dev", "staging", "prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &DefaultNotificationService{
				notifications: tt.notifications,
			}

			result := svc.selectDataset(settings.TabSessions, "timestamp", "desc")

			if len(result) != tt.wantLen {
				t.Errorf("expected %d sessions, got %d", tt.wantLen, len(result))
				return
			}

			for i, wantSession := range tt.wantSessions {
				if i >= len(result) {
					break
				}
				if result[i].Session != wantSession {
					t.Errorf("position %d: expected session %q, got %q", i, wantSession, result[i].Session)
				}
			}
		})
	}
}

// TestSelectDataset_SessionsTab_EdgeCases tests edge cases for the Sessions tab.
func TestSelectDataset_SessionsTab_EdgeCases(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	earlier := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name          string
		notifications []notification.Notification
		wantLen       int
	}{
		{
			name:          "empty notifications returns empty",
			notifications: []notification.Notification{},
			wantLen:       0,
		},
		{
			name: "skips notifications without session",
			notifications: []notification.Notification{
				{ID: 1, Session: "", State: "active", Timestamp: now, Message: "no session"},
				{ID: 2, Session: "dev", State: "active", Timestamp: earlier, Message: "dev msg"},
			},
			wantLen: 1,
		},
		{
			name: "multiple notifications same session only shows one",
			notifications: []notification.Notification{
				{ID: 1, Session: "dev", State: "active", Timestamp: earlier, Message: "first"},
				{ID: 2, Session: "dev", State: "active", Timestamp: now, Message: "second"},
				{ID: 3, Session: "dev", State: "dismissed", Timestamp: earlier, Message: "third"},
			},
			wantLen: 1,
		},
		{
			name: "most recent notification wins when multiple per session",
			notifications: []notification.Notification{
				{ID: 1, Session: "dev", State: "active", Timestamp: earlier, Message: "older"},
				{ID: 2, Session: "dev", State: "dismissed", Timestamp: now, Message: "newest dismissed"},
				{ID: 3, Session: "dev", State: "active", Timestamp: earlier, Message: "old active"},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &DefaultNotificationService{
				notifications: tt.notifications,
			}

			result := svc.selectDataset(settings.TabSessions, "timestamp", "desc")

			if len(result) != tt.wantLen {
				t.Errorf("expected %d sessions, got %d", tt.wantLen, len(result))
			}
		})
	}
}

// TestSelectDataset_SessionsTab_NoTimeLimit verifies that Sessions tab shows
// unique sessions with no time limit (all notifications from storage).
func TestSelectDataset_SessionsTab_NoTimeLimit(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	oneYearAgo := time.Now().Add(-365 * 24 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name          string
		notifications []notification.Notification
		wantLen       int
		wantSessions  []string
	}{
		{
			name: "recent notification is included",
			notifications: []notification.Notification{
				{ID: 1, Session: "dev", State: "active", Timestamp: now, Message: "recent"},
			},
			wantLen:      1,
			wantSessions: []string{"dev"},
		},
		{
			name: "old notification from 1 year ago is still included (no time limit)",
			notifications: []notification.Notification{
				{ID: 1, Session: "old-session", State: "active", Timestamp: oneYearAgo, Message: "old"},
			},
			wantLen:      1,
			wantSessions: []string{"old-session"},
		},
		{
			name: "both recent and old sessions shown with no time limit",
			notifications: []notification.Notification{
				{ID: 1, Session: "dev", State: "active", Timestamp: now, Message: "recent"},
				{ID: 2, Session: "archive", State: "active", Timestamp: oneYearAgo, Message: "old"},
			},
			wantLen:      2,
			wantSessions: []string{"dev", "archive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &DefaultNotificationService{
				notifications: tt.notifications,
			}

			result := svc.selectDataset(settings.TabSessions, "timestamp", "desc")

			if len(result) != tt.wantLen {
				t.Errorf("expected %d sessions, got %d", tt.wantLen, len(result))
				return
			}

			for i, wantSession := range tt.wantSessions {
				if i >= len(result) {
					break
				}
				if result[i].Session != wantSession {
					t.Errorf("position %d: expected session %q, got %q", i, wantSession, result[i].Session)
				}
			}
		})
	}
}
