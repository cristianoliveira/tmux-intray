package views

import (
	"fmt"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRecentsPerSessionSelection_Orchestrator verifies that the orchestrator
// replicates the TUI Recents per-session semantics: one representative per
// session, chosen by severity first (error > warning > info) then recency.
func TestActiveNotificationTimelineFiltersDismissed(t *testing.T) {
	orchestrator := NewOrchestrator()
	notifs := []domain.Notification{
		{ID: 1, State: domain.StateActive, Message: "active"},
		{ID: 2, State: domain.StateDismissed, Message: "dismissed"},
		{ID: 3, State: "", Message: "implicit active"},
	}

	result := orchestrator.Build(Options{Kind: KindActiveNotificationTimeline}, notifs)
	require.Len(t, result.Notifications, 2)
	assert.Equal(t, []int{1, 3}, []int{result.Notifications[0].ID, result.Notifications[1].ID})
}

func TestRecentsPerSessionSelection_Orchestrator(t *testing.T) {
	orchestrator := NewOrchestrator()
	now := time.Now().UTC()

	notifs := []domain.Notification{
		// Session 1: multiple notifications
		{
			ID:        1,
			Message:   "session1 info",
			Timestamp: now.Add(-30 * time.Minute).Format(time.RFC3339),
			State:     domain.StateActive,
			Level:     domain.LevelInfo,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		{
			ID:        2,
			Message:   "session1 warning",
			Timestamp: now.Add(-25 * time.Minute).Format(time.RFC3339),
			State:     domain.StateActive,
			Level:     domain.LevelWarning,
			Session:   "$1",
			Window:    "@1",
			Pane:      "%1",
		},
		// Session 2: multiple notifications
		{
			ID:        3,
			Message:   "session2 error",
			Timestamp: now.Add(-28 * time.Minute).Format(time.RFC3339),
			State:     domain.StateActive,
			Level:     domain.LevelError,
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
		},
		{
			ID:        4,
			Message:   "session2 info",
			Timestamp: now.Add(-22 * time.Minute).Format(time.RFC3339),
			State:     domain.StateActive,
			Level:     domain.LevelInfo,
			Session:   "$2",
			Window:    "@2",
			Pane:      "%2",
		},
		// Session 3: single notification
		{
			ID:        5,
			Message:   "session3 warning",
			Timestamp: now.Add(-15 * time.Minute).Format(time.RFC3339),
			State:     domain.StateActive,
			Level:     domain.LevelWarning,
			Session:   "$3",
			Window:    "@3",
			Pane:      "%3",
		},
	}

	opts := Options{
		Kind:   KindRecentUnreadSessionHighlights,
		SortBy: "timestamp",
		Order:  "desc",
	}

	result := orchestrator.Build(opts, notifs)

	require.Len(t, result.Notifications, 3, "one representative per session expected")

	bySession := make(map[string]domain.Notification)
	for _, n := range result.Notifications {
		bySession[n.Session] = n
	}

	assert.Equal(t, domain.LevelWarning, bySession["$1"].Level)
	assert.Equal(t, 2, bySession["$1"].ID)

	assert.Equal(t, domain.LevelError, bySession["$2"].Level)
	assert.Equal(t, 3, bySession["$2"].ID)

	assert.Equal(t, domain.LevelWarning, bySession["$3"].Level)
	assert.Equal(t, 5, bySession["$3"].ID)

	// Ensure ordering by most recent representative activity
	require.Equal(t, 3, len(result.Notifications))

	// Collect timestamps for debug if ordering fails
	var ids []int
	for _, n := range result.Notifications {
		ids = append(ids, n.ID)
	}

	// We expect most recent session first. With the timestamps above, that is:
	// session3 (ID 5), session1 (ID 2), session2 (ID 3).
	if !assert.Equal(t, []int{5, 2, 3}, ids) {
		for i, n := range result.Notifications {
			parsed, _ := time.Parse(time.RFC3339, n.Timestamp)
			t.Logf("idx=%d id=%d ts=%s", i, n.ID, parsed.Format(time.RFC3339))
		}
	}
}

// TestRecentsPerSessionLimit_Orchestrator verifies that we respect the 20-session
// dataset limit, returning the most recent 20 representatives.
func TestRecentsPerSessionLimit_Orchestrator(t *testing.T) {
	orchestrator := NewOrchestrator()
	now := time.Now().UTC()

	notifs := make([]domain.Notification, 0, 30)
	for i := 1; i <= 30; i++ {
		notifs = append(notifs, domain.Notification{
			ID:        i,
			Message:   fmt.Sprintf("msg%d", i),
			Timestamp: now.Add(-time.Duration(30-i) * time.Minute).Format(time.RFC3339),
			State:     domain.StateActive,
			Level:     domain.LevelInfo,
			Session:   fmt.Sprintf("$%d", i),
			Window:    "@1",
			Pane:      "%1",
		})
	}

	opts := Options{
		Kind:   KindRecentUnreadSessionHighlights,
		SortBy: "timestamp",
		Order:  "desc",
	}

	result := orchestrator.Build(opts, notifs)

	require.Len(t, result.Notifications, 20, "should limit to 20 sessions")

	// Expect the most recent 20 IDs: 30 down to 11
	var ids []int
	for _, n := range result.Notifications {
		ids = append(ids, n.ID)
	}
	assert.Equal(t, []int{30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11}, ids)
}

func TestRecentsPerSessionFiltersActiveUnreadAndWindow_Orchestrator(t *testing.T) {
	orchestrator := NewOrchestrator()
	now := time.Now().UTC()

	notifs := []domain.Notification{
		{ID: 1, Timestamp: now.Add(-20 * time.Minute).Format(time.RFC3339), State: domain.StateActive, Session: "$1", Level: domain.LevelInfo},
		{ID: 2, Timestamp: now.Add(-15 * time.Minute).Format(time.RFC3339), State: "", Session: "$2", Level: domain.LevelInfo}, // blank state treated as active
		{ID: 3, Timestamp: now.Add(-10 * time.Minute).Format(time.RFC3339), State: domain.StateDismissed, Session: "$3", Level: domain.LevelInfo},
		{ID: 4, Timestamp: now.Add(-5 * time.Minute).Format(time.RFC3339), State: domain.StateActive, Session: "$4", Level: domain.LevelInfo, ReadTimestamp: now.Format(time.RFC3339)},
		{ID: 5, Timestamp: now.Add(-2 * time.Hour).Format(time.RFC3339), State: domain.StateActive, Session: "$5", Level: domain.LevelInfo},
	}

	result := orchestrator.Build(Options{Kind: KindRecentUnreadSessionHighlights}, notifs)

	require.Len(t, result.Notifications, 2)
	assert.Equal(t, []int{2, 1}, []int{result.Notifications[0].ID, result.Notifications[1].ID})
}

func TestSessionsPerSessionIgnoresTimeWindowAndReadState_Orchestrator(t *testing.T) {
	orchestrator := NewOrchestrator()
	now := time.Now().UTC()

	notifs := []domain.Notification{
		{ID: 1, Timestamp: now.Add(-3 * time.Hour).Format(time.RFC3339), State: domain.StateActive, Session: "$1", Level: domain.LevelInfo, ReadTimestamp: now.Add(-2 * time.Hour).Format(time.RFC3339)},
		{ID: 2, Timestamp: now.Add(-2 * time.Hour).Format(time.RFC3339), State: domain.StateActive, Session: "$2", Level: domain.LevelWarning},
		{ID: 3, Timestamp: now.Add(-90 * time.Minute).Format(time.RFC3339), State: domain.StateDismissed, Session: "$3", Level: domain.LevelError},
		{ID: 4, Timestamp: now.Add(-30 * time.Minute).Format(time.RFC3339), State: domain.StateActive, Session: "$1", Level: domain.LevelError},
	}

	result := orchestrator.Build(Options{Kind: KindSessionHistory}, notifs)

	require.Len(t, result.Notifications, 2, "sessions view should include all active sessions regardless of time/read state")
	assert.Equal(t, []int{4, 2}, []int{result.Notifications[0].ID, result.Notifications[1].ID})
}
