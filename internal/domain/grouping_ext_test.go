package domain

import (
	"testing"
	"time"
)

func TestGroupingCriteria_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		criteria GroupingCriteria
		want     bool
	}{
		{"Valid exact match", GroupByExactMatch, true},
		{"Valid message and level", GroupByMessageAndLevel, true},
		{"Valid message and source", GroupByMessageAndSource, true},
		{"Valid time window", GroupByTimeWindow, true},
		{"Invalid criteria", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.criteria.IsValid(); got != tt.want {
				t.Errorf("GroupingCriteria.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotificationGroup_GetDisplayName(t *testing.T) {
	notif := &Notification{
		Message: "Test message",
		Level:   LevelInfo,
		Session: "session1",
		Window:  "window1",
		Pane:    "pane1",
	}

	tests := []struct {
		name     string
		criteria GroupingCriteria
		want     string
	}{
		{"Exact match", GroupByExactMatch, "Exact Match: Test message"},
		{"Message and level", GroupByMessageAndLevel, "Test message (info)"},
		{"Message and source", GroupByMessageAndSource, "Test message (session1/window1/pane1)"},
		{"Time window", GroupByTimeWindow, "Test message (Time Window)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := NewNotificationGroup(tt.criteria, notif)
			if got := group.GetDisplayName(); got != tt.want {
				t.Errorf("NotificationGroup.GetDisplayName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDuplicate(t *testing.T) {
	existing := &Notification{
		Message: "Test message",
		Level:   LevelInfo,
		Session: "session1",
		Window:  "window1",
		Pane:    "pane1",
	}

	tests := []struct {
		name     string
		criteria GroupingCriteria
		new      *Notification
		want     bool
	}{
		{"Exact match - same", GroupByExactMatch, &Notification{Message: "Test message"}, true},
		{"Exact match - different", GroupByExactMatch, &Notification{Message: "Different message"}, false},
		{"Message and level - same", GroupByMessageAndLevel, &Notification{Message: "Test message", Level: LevelInfo}, true},
		{"Message and level - different level", GroupByMessageAndLevel, &Notification{Message: "Test message", Level: LevelError}, false},
		{"Message and source - same", GroupByMessageAndSource, &Notification{Message: "Test message", Session: "session1", Window: "window1", Pane: "pane1"}, true},
		{"Message and source - different source", GroupByMessageAndSource, &Notification{Message: "Test message", Session: "session2"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDuplicate(tt.criteria, existing, tt.new); got != tt.want {
				t.Errorf("IsDuplicate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupNotificationsByCriteria(t *testing.T) {
	notifications := []*Notification{
		{
			ID:        1,
			Timestamp: time.Now().Format(time.RFC3339),
			Message:   "Test message",
			Level:     LevelInfo,
			Session:   "session1",
			Window:    "window1",
			Pane:      "pane1",
		},
		{
			ID:        2,
			Timestamp: time.Now().Add(time.Minute).Format(time.RFC3339),
			Message:   "Test message",
			Level:     LevelInfo,
			Session:   "session1",
			Window:    "window1",
			Pane:      "pane1",
		},
		{
			ID:        3,
			Timestamp: time.Now().Add(2 * time.Minute).Format(time.RFC3339),
			Message:   "Different message",
			Level:     LevelWarning,
			Session:   "session1",
			Window:    "window1",
			Pane:      "pane1",
		},
	}

	config := GroupingConfig{
		Criteria:          GroupByExactMatch,
		TimeWindow:        5 * time.Minute,
		MaxGroupSize:      10,
		EnableAggregation: true,
	}

	result, err := GroupNotificationsByCriteria(notifications, config)
	if err != nil {
		t.Fatalf("GroupNotificationsByCriteria() error = %v", err)
	}

	if len(result.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(result.Groups))
	}

	// Check group counts - the order might be different, so check both ways
	if (result.Groups[0].Count == 2 && result.Groups[1].Count == 1) ||
		(result.Groups[0].Count == 1 && result.Groups[1].Count == 2) {
		// This is correct
	} else {
		t.Errorf("Expected groups to have counts 2 and 1, got %d and %d",
			result.Groups[0].Count, result.Groups[1].Count)
	}
}

func TestGetGroupedNotificationsByCriteria(t *testing.T) {
	notifications := []*Notification{
		{
			ID:        1,
			Timestamp: time.Now().Format(time.RFC3339),
			Message:   "Test message",
			Level:     LevelInfo,
			Session:   "session1",
			Window:    "window1",
			Pane:      "pane1",
		},
		{
			ID:        2,
			Timestamp: time.Now().Add(time.Minute).Format(time.RFC3339),
			Message:   "Test message",
			Level:     LevelInfo,
			Session:   "session1",
			Window:    "window1",
			Pane:      "pane1",
		},
	}

	groups, err := GetGroupedNotificationsByCriteria(notifications, GroupByExactMatch)
	if err != nil {
		t.Fatalf("GetGroupedNotificationsByCriteria() error = %v", err)
	}

	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}

	if groups[0].Count != 2 {
		t.Errorf("Group should have 2 notifications, got %d", groups[0].Count)
	}
}
