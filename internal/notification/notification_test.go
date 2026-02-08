package notification

import "testing"

func TestParseNotification(t *testing.T) {
	line := "1\t2025-01-01T12:00:00Z\tactive\tsess\twin\tpane\ttest\\tmessage\t123\tinfo\t2025-01-02T01:02:03Z"
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
