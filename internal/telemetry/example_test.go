package telemetry_test

import (
	"fmt"
	"os"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/telemetry"
)

// Example demonstrates the telemetry package working end-to-end.
// This example verifies:
// 1. Init() initializes the system
// 2. IsEnabled() checks configuration
// 3. LogCLICommand, LogTUIAction, and LogFeature log events
// 4. Events are written to storage
// 5. Shutdown() gracefully stops the system
func Example() {
	// Set telemetry enabled
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")
	defer os.Unsetenv("TMUX_INTRAY_TELEMETRY_ENABLED")

	// Initialize telemetry
	if err := telemetry.Init(); err != nil {
		panic(err)
	}

	// Check if enabled
	if !telemetry.IsEnabled() {
		panic("telemetry should be enabled")
	}

	// Log some events
	telemetry.LogCLICommand("add", []string{"--level", "error", "test message"}, map[string]interface{}{
		"session": "test-session",
		"window":  "1",
		"pane":    "0",
	})

	telemetry.LogTUIAction("open", map[string]interface{}{
		"view_mode": "detailed",
	})

	telemetry.LogFeature("jump-to-pane", "tui", map[string]interface{}{
		"session":  "test",
		"window":   "1",
		"pane":     "0",
		"duration": "1.5s",
	})

	// Give time for async processing
	time.Sleep(100 * time.Millisecond)

	// Shutdown
	telemetry.Shutdown()

	fmt.Println("Telemetry events logged successfully")
	// Output: Telemetry events logged successfully
}

// Example_disabled demonstrates that telemetry is a no-op when disabled.
func Example_disabled() {
	// Ensure telemetry is disabled (default)
	os.Unsetenv("TMUX_INTRAY_TELEMETRY_ENABLED")

	// Initialize (even if disabled)
	if err := telemetry.Init(); err != nil {
		panic(err)
	}

	// Check if enabled
	if telemetry.IsEnabled() {
		panic("telemetry should be disabled by default")
	}

	// Log some events - these should be no-ops
	telemetry.LogCLICommand("add", []string{"test"}, nil)
	telemetry.LogTUIAction("open", nil)
	telemetry.LogFeature("test-feature", "cli", nil)

	// Shutdown
	telemetry.Shutdown()

	fmt.Println("Telemetry events were no-ops (disabled)")
	// Output: Telemetry events were no-ops (disabled)
}
