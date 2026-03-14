package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	// Test successful initialization
	err := Init()
	require.NoError(t, err)

	// Test that second call returns without error
	err = Init()
	require.NoError(t, err)

	// Test that event processor is started
	eventProcessorMu.Lock()
	started := eventProcessorStarted
	eventProcessorMu.Unlock()
	require.True(t, started, "event processor should be started after Init")
}

func TestIsEnabled(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	// Test when telemetry is disabled (default)
	enabled := IsEnabled()
	require.False(t, enabled, "telemetry should be disabled by default")

	// Test with environment variable set
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)

	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")
	enabled = IsEnabled()
	require.True(t, enabled, "telemetry should be enabled when env var is true")

	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "1")
	enabled = IsEnabled()
	require.True(t, enabled, "telemetry should be enabled when env var is 1")

	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "false")
	enabled = IsEnabled()
	require.False(t, enabled, "telemetry should be disabled when env var is false")
}

func TestLogCLICommand(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	tests := []struct {
		name      string
		command   string
		args      []string
		context   map[string]interface{}
		shouldLog bool
	}{
		{
			name:      "disabled telemetry",
			command:   "add",
			args:      []string{"test"},
			context:   nil,
			shouldLog: false,
		},
		{
			name:      "simple command",
			command:   "list",
			args:      []string{},
			context:   nil,
			shouldLog: true,
		},
		{
			name:      "command with args",
			command:   "add",
			args:      []string{"--level", "error", "test message"},
			context:   nil,
			shouldLog: true,
		},
		{
			name:      "command with context",
			command:   "dismiss",
			args:      []string{"1"},
			context:   map[string]interface{}{"session": "test-session"},
			shouldLog: true,
		},
		{
			name:      "complex context",
			command:   "jump",
			args:      []string{"1"},
			context:   map[string]interface{}{"session": "test", "window": "1", "pane": "0"},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset for each test
			Reset()

			// Set telemetry enabled based on shouldLog
			if tt.shouldLog {
				oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
				defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
				os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

				// Initialize
				err := Init()
				require.NoError(t, err)
			}

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			defer func() { os.Stderr = oldStderr }()

			// Log CLI command
			LogCLICommand(tt.command, tt.args, tt.context)

			// Close pipe and read output
			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.shouldLog {
				// Should not have errors
				require.NotContains(t, output, "telemetry: failed")
			} else {
				// Should be a no-op
				require.Empty(t, output)
			}
		})
	}
}

func TestLogTUIAction(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	tests := []struct {
		name      string
		action    string
		context   map[string]interface{}
		shouldLog bool
	}{
		{
			name:      "disabled telemetry",
			action:    "open",
			context:   nil,
			shouldLog: false,
		},
		{
			name:      "simple action",
			action:    "open",
			context:   nil,
			shouldLog: true,
		},
		{
			name:      "action with context",
			action:    "filter",
			context:   map[string]interface{}{"filter_type": "level"},
			shouldLog: true,
		},
		{
			name:      "complex context",
			action:    "jump",
			context:   map[string]interface{}{"notification_id": "123", "target": "pane"},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset for each test
			Reset()

			// Set telemetry enabled based on shouldLog
			if tt.shouldLog {
				oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
				defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
				os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

				// Initialize
				err := Init()
				require.NoError(t, err)
			}

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			defer func() { os.Stderr = oldStderr }()

			// Log TUI action
			LogTUIAction(tt.action, tt.context)

			// Close pipe and read output
			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.shouldLog {
				// Should not have errors
				require.NotContains(t, output, "telemetry: failed")
			} else {
				// Should be a no-op
				require.Empty(t, output)
			}
		})
	}
}

func TestLogFeature(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	tests := []struct {
		name        string
		feature     string
		category    string
		context     map[string]interface{}
		shouldLog   bool
		expectError bool
	}{
		{
			name:      "disabled telemetry",
			feature:   "test-feature",
			category:  "cli",
			context:   nil,
			shouldLog: false,
		},
		{
			name:      "simple feature",
			feature:   "add-notification",
			category:  "cli",
			context:   nil,
			shouldLog: true,
		},
		{
			name:      "feature with context",
			feature:   "dismiss-notification",
			category:  "tui",
			context:   map[string]interface{}{"level": "error"},
			shouldLog: true,
		},
		{
			name:        "empty feature name",
			feature:     "",
			category:    "cli",
			context:     nil,
			shouldLog:   false,
			expectError: true,
		},
		{
			name:      "empty category defaults to cli",
			feature:   "test-feature",
			category:  "",
			context:   nil,
			shouldLog: true,
		},
		{
			name:      "complex context",
			feature:   "jump-to-pane",
			category:  "tui",
			context:   map[string]interface{}{"session": "test", "window": "1", "pane": "0", "duration": "1.5s"},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset for each test
			Reset()

			// Set telemetry enabled based on shouldLog
			if tt.shouldLog || tt.expectError {
				oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
				defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
				os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

				// Initialize
				err := Init()
				require.NoError(t, err)
			}

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			defer func() { os.Stderr = oldStderr }()

			// Log feature
			LogFeature(tt.feature, tt.category, tt.context)

			// Close pipe and read output
			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.expectError {
				// Should have error about empty feature name
				require.Contains(t, output, "telemetry: feature name cannot be empty")
			} else if tt.shouldLog {
				// Should not have errors (except maybe empty category warning)
				if tt.category == "" {
					require.Contains(t, output, "telemetry: feature category cannot be empty")
				}
				require.NotContains(t, output, "telemetry: failed to marshal context")
			} else {
				// Should be a no-op
				require.Empty(t, output)
			}
		})
	}
}

func TestContextJSONMarshaling(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	// Set telemetry enabled
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	require.NoError(t, err)

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	// Log with context that should marshal successfully
	context := map[string]interface{}{
		"string": "test",
		"number": 42,
		"bool":   true,
		"array":  []string{"a", "b", "c"},
		"nested": map[string]interface{}{"key": "value"},
	}

	LogFeature("test-feature", "cli", context)

	// Close pipe and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should not have marshaling errors
	require.NotContains(t, output, "telemetry: failed to marshal context")
}

func TestChannelFull(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	// Set telemetry enabled
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize storage manually to control event channel
	err := storage.Init()
	require.NoError(t, err)

	// Get storage instance
	storageInstance, err = storage.NewFromConfig()
	require.NoError(t, err)

	// Create event channel with very small buffer
	eventChannel = make(chan *telemetryEvent, 1)

	// Don't start event processor - we want to fill the channel

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	// Send two events - first should succeed, second should drop
	LogFeature("feature1", "cli", nil)
	LogFeature("feature2", "cli", nil)

	// Close pipe and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should have channel full warning
	require.Contains(t, output, "telemetry: event channel full")
	require.Contains(t, output, "dropping event")
}

func TestAsyncLogging(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	// Set telemetry enabled
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	require.NoError(t, err)

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	// Log multiple events quickly
	startTime := time.Now()
	for i := 0; i < 10; i++ {
		LogFeature(fmt.Sprintf("feature%d", i), "cli", nil)
	}
	elapsed := time.Since(startTime)

	// Close pipe and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Logging should be fast (async)
	require.Less(t, elapsed, 100*time.Millisecond, "async logging should not block")

	// Should not have errors
	require.NotContains(t, output, "telemetry: failed")
}

func TestTimestampFormat(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	// Set telemetry enabled
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize storage manually to control event channel
	err := storage.Init()
	require.NoError(t, err)

	// Get storage instance
	storageInstance, err = storage.NewFromConfig()
	require.NoError(t, err)

	// Create event channel but don't start event processor
	eventChannel = make(chan *telemetryEvent, 10)

	// Log feature
	before := time.Now()
	LogFeature("test-feature", "cli", nil)
	time.Sleep(10 * time.Millisecond) // Give time for event to be sent

	// Read event directly from channel
	var capturedEvent *telemetryEvent
	select {
	case event := <-eventChannel:
		capturedEvent = event
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
	after := time.Now()

	// Parse and validate timestamp format
	parsedTime, err := time.Parse(time.RFC3339, capturedEvent.timestamp)
	require.NoError(t, err, "timestamp should be in RFC3339 format")
	require.True(t, parsedTime.After(before.Add(-time.Second)), "timestamp should be close to current time")
	require.True(t, parsedTime.Before(after.Add(time.Second)), "timestamp should be close to current time")
}

func TestLogCLICommandContext(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	// Set telemetry enabled
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize storage manually to control event channel
	err := storage.Init()
	require.NoError(t, err)

	// Get storage instance
	storageInstance, err = storage.NewFromConfig()
	require.NoError(t, err)

	// Create event channel but don't start event processor
	eventChannel = make(chan *telemetryEvent, 10)

	// Log CLI command with context
	command := "add"
	args := []string{"--level", "error", "test message"}
	context := map[string]interface{}{
		"session": "test-session",
		"window":  "1",
	}

	LogCLICommand(command, args, context)
	time.Sleep(10 * time.Millisecond)

	// Read event directly from channel
	var capturedEvent *telemetryEvent
	select {
	case event := <-eventChannel:
		capturedEvent = event
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}

	// Parse and validate context
	var parsedContext map[string]interface{}
	err = json.Unmarshal([]byte(capturedEvent.contextData), &parsedContext)
	require.NoError(t, err, "context should be valid JSON")

	// Verify command and args are in context
	require.Equal(t, command, parsedContext["command"])
	// Note: JSON unmarshaling converts arrays to []interface{}
	require.ElementsMatch(t, args, parsedContext["args"])
	require.Equal(t, "test-session", parsedContext["session"])
	require.Equal(t, "1", parsedContext["window"])
}

func TestLogTUIActionContext(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	// Set telemetry enabled
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize storage manually to control event channel
	err := storage.Init()
	require.NoError(t, err)

	// Get storage instance
	storageInstance, err = storage.NewFromConfig()
	require.NoError(t, err)

	// Create event channel but don't start event processor
	eventChannel = make(chan *telemetryEvent, 10)

	// Log TUI action with context
	action := "filter"
	context := map[string]interface{}{
		"filter_type":  "level",
		"filter_value": "error",
	}

	LogTUIAction(action, context)
	time.Sleep(10 * time.Millisecond)

	// Read event directly from channel
	var capturedEvent *telemetryEvent
	select {
	case event := <-eventChannel:
		capturedEvent = event
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}

	// Parse and validate context
	var parsedContext map[string]interface{}
	err = json.Unmarshal([]byte(capturedEvent.contextData), &parsedContext)
	require.NoError(t, err, "context should be valid JSON")

	// Verify action is in context
	require.Equal(t, action, parsedContext["action"])
	require.Equal(t, "level", parsedContext["filter_type"])
	require.Equal(t, "error", parsedContext["filter_value"])
}

func TestShutdown(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	// Set telemetry enabled
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	require.NoError(t, err)

	// Verify event processor is running
	eventProcessorMu.Lock()
	started := eventProcessorStarted
	eventProcessorMu.Unlock()
	require.True(t, started)

	// Shutdown
	Shutdown()

	// Verify event processor is stopped
	eventProcessorMu.Lock()
	started = eventProcessorStarted
	eventProcessorMu.Unlock()
	require.False(t, started)

	// Shutdown should be idempotent
	Shutdown()
}

func TestReset(t *testing.T) {
	// Set telemetry enabled
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	require.NoError(t, err)

	// Call Reset
	Reset()

	// Verify state is reset
	require.Nil(t, storageInstance)

	// Verify can reinitialize after Reset
	err = Init()
	require.NoError(t, err)
}

func TestMultipleLogCalls(t *testing.T) {
	// Reset before test
	Reset()
	defer Reset()

	// Set telemetry enabled
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	require.NoError(t, err)

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	// Log multiple events of different types
	LogCLICommand("add", []string{"test"}, nil)
	LogTUIAction("open", nil)
	LogFeature("test-feature", "cli", nil)
	LogCLICommand("list", []string{}, nil)
	LogTUIAction("close", nil)

	// Close pipe and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should not have errors
	require.NotContains(t, output, "telemetry: failed")
}
