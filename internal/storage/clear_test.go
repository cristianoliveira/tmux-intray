package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDismissAll(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")
	os.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	// Create notifications.tsv with sample data
	notificationsFile := filepath.Join(stateDir, "notifications.tsv")
	os.MkdirAll(stateDir, 0755)
	sampleData := `1	2026-02-01T19:37:23Z	active	$23	@137		[2026-02-01 20:37:22] Warning message main		warning
2	2026-02-01T19:37:26Z	active	$23	@138		[2026-02-01 20:37:26] Info message alt window		info
3	2026-02-01T19:37:28Z	active	$24	@139		[2026-02-01 20:37:28] Error message other session		error
`
	err := os.WriteFile(notificationsFile, []byte(sampleData), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Run DismissAll
	err = DismissAll()
	if err != nil {
		t.Fatalf("DismissAll failed: %v", err)
	}

	// Read file after operation
	data, err := os.ReadFile(notificationsFile)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 6 {
		t.Errorf("expected 6 lines (3 original + 3 dismissed), got %d", len(lines))
		for i, line := range lines {
			t.Logf("line %d: %s", i, line)
		}
	}

	// Verify that the last three lines have state "dismissed"
	for i := 3; i < 6; i++ {
		fields := strings.Split(lines[i], "\t")
		if len(fields) < 3 {
			t.Errorf("line %d has insufficient fields", i)
			continue
		}
		if fields[2] != "dismissed" {
			t.Errorf("line %d state = %s, want dismissed", i, fields[2])
		}
	}
}
