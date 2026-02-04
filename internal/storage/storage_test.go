package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorageStubs(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")
	os.Setenv("TMUX_INTRAY_STATE_DIR", stateDir)
	defer os.Unsetenv("TMUX_INTRAY_STATE_DIR")
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "0")

	Init()

	require.Equal(t, "", AddNotification("msg", "", "", "", "", "", "info"))
	require.Equal(t, "", ListNotifications("active", "", "", "", "", "", ""))

	_ = DismissNotification("1")
	DismissAll()
	CleanupOldNotifications(30, true)

	require.Equal(t, 0, GetActiveCount())
}

func TestDismissNotification(t *testing.T) {
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
`
	err := os.WriteFile(notificationsFile, []byte(sampleData), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Run DismissNotification
	err = DismissNotification("1")
	if err != nil {
		t.Fatalf("DismissNotification failed: %v", err)
	}

	// Read file after operation
	data, err := os.ReadFile(notificationsFile)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (original + dismissed), got %d", len(lines))
		for i, line := range lines {
			t.Logf("line %d: %s", i, line)
		}
	}

	// Verify that the second line has state "dismissed"
	fields := strings.Split(lines[1], "\t")
	if len(fields) < 3 {
		t.Errorf("line 1 has insufficient fields")
	} else if fields[2] != "dismissed" {
		t.Errorf("line 1 state = %s, want dismissed", fields[2])
	}

	// Verify that dismissing again returns error
	err = DismissNotification("1")
	if err == nil {
		t.Error("expected error when dismissing already dismissed notification")
	}
}
