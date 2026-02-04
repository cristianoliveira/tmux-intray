package cmd

import (
	"os/exec"
	"strings"
	"testing"
)

func TestToggleCommand(t *testing.T) {
	// Skip if tmux not running
	if err := exec.Command("tmux", "has-session").Run(); err != nil {
		t.Skip("tmux not running, skipping toggle test")
	}

	// Use the Go binary built in the project root
	binary := "../tmux-intray-go"
	if _, err := exec.LookPath(binary); err != nil {
		// Try to build it
		cmd := exec.Command("go", "build", "-o", binary)
		cmd.Dir = ".."
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to build binary: %v, output: %s", err, out)
		}
	}

	// Helper to get current visibility
	getVisibility := func() string {
		cmd := exec.Command("tmux", "show-environment", "-g", "TMUX_INTRAY_VISIBLE")
		out, err := cmd.CombinedOutput()
		if err != nil {
			// Variable not set, default to "0"
			return "0"
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "TMUX_INTRAY_VISIBLE=") {
				return strings.TrimPrefix(line, "TMUX_INTRAY_VISIBLE=")
			}
		}
		return "0"
	}

	// Helper to run toggle command
	runToggle := func() (string, error) {
		cmd := exec.Command(binary, "toggle")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// Record initial visibility
	initial := getVisibility()
	t.Logf("initial visibility: %s", initial)

	// Toggle twice and verify behavior
	for i := 0; i < 2; i++ {
		visibleBefore := getVisibility()
		out, err := runToggle()
		if err != nil {
			t.Fatalf("toggle %d failed: %v, output: %s", i+1, err, out)
		}
		// Determine expected message based on visibility before toggle
		// Message indicates new state after toggle
		var expected string
		if visibleBefore == "1" {
			expected = "Tray hidden"
		} else {
			expected = "Tray visible"
		}
		if !strings.Contains(out, expected) {
			t.Errorf("toggle %d: expected output to contain %q, got %q", i+1, expected, out)
		}
		// Verify visibility changed
		visibleAfter := getVisibility()
		if visibleAfter == visibleBefore {
			t.Errorf("toggle %d: visibility did not change (still %s)", i+1, visibleAfter)
		}
		t.Logf("toggle %d: %s -> %s", i+1, visibleBefore, visibleAfter)
	}

	// Final visibility should match initial (after two toggles)
	final := getVisibility()
	if final != initial {
		t.Errorf("final visibility %s does not match initial %s", final, initial)
	}
}
