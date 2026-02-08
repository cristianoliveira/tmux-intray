package notification

import (
	"testing"
	"time"
)

func TestParseNotification(t *testing.T) {
	line := "1\t2025-01-01T12:00:00Z\tactive\tsess\tsessname\twin\tpane\ttest\\tmessage\t123\tinfo\t2025-01-02T01:02:03Z"
	notif, err := ParseNotification(line)
	if err != nil {
		t.Fatal(err)
	}
	if notif.ID != 1 {
		t.Errorf("Expected ID 1, got %d", notif.ID)
	}
	if notif.Timestamp != "2025-01-01T12:00:00Z" {
		t.Errorf("Timestamp mismatch")
	}
	if notif.State != "active" {
		t.Errorf("State mismatch")
	}
	if notif.SessionName != "sessname" {
		t.Errorf("SessionName mismatch")
	}
	if notif.Message != "test\tmessage" {
		t.Errorf("Message not unescaped: %q", notif.Message)
	}
	if notif.Level != "info" {
		t.Errorf("Level mismatch")
	}
	if notif.ReadTimestamp != "2025-01-02T01:02:03Z" {
		t.Errorf("ReadTimestamp mismatch")
	}
}

func TestParseNotificationWithoutReadTimestamp(t *testing.T) {
	line := "1\t2025-01-01T12:00:00Z\tactive\tsess\twin\tpane\ttest\\tmessage\t123\tinfo"
	notif, err := ParseNotification(line)
	if err != nil {
		t.Fatal(err)
	}
	if notif.ReadTimestamp != "" {
		t.Errorf("Expected empty ReadTimestamp, got %q", notif.ReadTimestamp)
	}
}

func TestNotificationReadHelpers(t *testing.T) {
	n := Notification{}
	if n.IsRead() {
		t.Errorf("Expected IsRead false for empty ReadTimestamp")
	}

	n = n.MarkRead()
	if n.ReadTimestamp == "" {
		t.Errorf("Expected ReadTimestamp to be set")
	}
	if _, err := time.Parse(time.RFC3339, n.ReadTimestamp); err != nil {
		t.Fatalf("Expected RFC3339 timestamp, got %q", n.ReadTimestamp)
	}
	if !n.IsRead() {
		t.Errorf("Expected IsRead true after MarkRead")
	}

	n = n.MarkUnread()
	if n.ReadTimestamp != "" {
		t.Errorf("Expected empty ReadTimestamp after MarkUnread, got %q", n.ReadTimestamp)
	}
	if n.IsRead() {
		t.Errorf("Expected IsRead false after MarkUnread")
	}
}

func TestUnescapeMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plain", "plain"},
		{"test\\nnewline", "test\nnewline"},
		{"test\\ttab", "test\ttab"},
		{"back\\\\slash", "back\\slash"},
		{"mixed\\n\\t\\", "mixed\n\t\\"},
	}
	for _, tt := range tests {
		got := unescapeMessage(tt.input)
		if got != tt.expected {
			t.Errorf("unescapeMessage(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
