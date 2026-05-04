package service

import (
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
)

func TestGetMostRecentPerSession(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	earlier := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	muchEarlier := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name         string
		input        []domain.Notification
		wantLen      int
		wantSessions []string // Expected session order
	}{
		{
			name:    "empty input returns empty",
			input:   []domain.Notification{},
			wantLen: 0,
		},
		{
			name: "single session",
			input: []domain.Notification{
				{ID: 1, Session: "dev", Timestamp: now, Message: "msg"},
			},
			wantLen:      1,
			wantSessions: []string{"dev"},
		},
		{
			name: "multiple sessions keeps most recent per session",
			input: []domain.Notification{
				{ID: 1, Session: "dev", Timestamp: now, Message: "dev newest"},
				{ID: 2, Session: "dev", Timestamp: earlier, Message: "dev older"},
				{ID: 3, Session: "prod", Timestamp: muchEarlier, Message: "prod old"},
			},
			wantLen:      2,
			wantSessions: []string{"dev", "prod"}, // sorted by timestamp desc
		},
		{
			name: "skips notifications without session",
			input: []domain.Notification{
				{ID: 1, Session: "dev", Timestamp: now, Message: "dev msg"},
				{ID: 2, Session: "", Timestamp: now, Message: "no session"},
			},
			wantLen:      1,
			wantSessions: []string{"dev"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &DefaultNotificationService{}
			result := svc.getMostRecentPerSession(tt.input, "timestamp", "desc")

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

// TestGetMostRecentPerSession_UsesSharedDomainFunction verifies that the service
// method produces the same output as the domain function.
func TestGetMostRecentPerSession_UsesSharedDomainFunction(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	earlier := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)

	input := []domain.Notification{
		{ID: 1, Session: "dev", Timestamp: now, Message: "dev msg"},
		{ID: 2, Session: "prod", Timestamp: earlier, Message: "prod msg"},
	}

	// Convert to domain
	domainNotifs := convertToDomain(input)

	// Call shared domain function
	domainResult := domain.GroupBySessionKeepMostRecent(domainNotifs)

	// Call service method
	svc := &DefaultNotificationService{}
	serviceResult := svc.getMostRecentPerSession(input, "timestamp", "desc")

	// Both should have same length
	if len(domainResult) != len(serviceResult) {
		t.Errorf("domain result len=%d, service result len=%d", len(domainResult), len(serviceResult))
	}

	// Both should have same sessions (though order might differ)
	domainSessions := make(map[string]string)
	for _, d := range domainResult {
		domainSessions[d.Session] = d.Notification.Message
	}

	serviceSessions := make(map[string]string)
	for _, n := range serviceResult {
		serviceSessions[n.Session] = n.Message
	}

	for session, msg := range domainSessions {
		if svcMsg, ok := serviceSessions[session]; !ok {
			t.Errorf("service missing session %q", session)
		} else if svcMsg != msg {
			t.Errorf("session %q: domain msg=%q, service msg=%q", session, msg, svcMsg)
		}
	}
}

// convertToDomain is a test helper to convert domain.Notification to domain.Notification.
func convertToDomain(notifs []domain.Notification) []domain.Notification {
	result := make([]domain.Notification, len(notifs))
	for i, n := range notifs {
		level := domain.NotificationLevel(n.Level)
		if n.Level == "" {
			level = domain.LevelInfo
		}
		state := domain.NotificationState(n.State)
		if n.State == "" {
			state = domain.StateActive
		}
		result[i] = domain.Notification{
			ID:            n.ID,
			Timestamp:     n.Timestamp,
			State:         state,
			Session:       n.Session,
			Window:        n.Window,
			Pane:          n.Pane,
			Message:       n.Message,
			PaneCreated:   n.PaneCreated,
			Level:         level,
			ReadTimestamp: n.ReadTimestamp,
		}
	}
	return result
}
