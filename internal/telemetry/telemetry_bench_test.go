package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

// BenchmarkLogCLICommand measures the overhead of logging a CLI command.
// This includes JSON marshaling, event channel send, and any synchronous overhead.
func BenchmarkLogCLICommand(b *testing.B) {
	// Setup
	Reset()
	defer Reset()

	// Enable telemetry
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	if err != nil {
		b.Fatalf("failed to initialize telemetry: %v", err)
	}

	// Suppress stderr for benchmarks
	devNull, _ := os.Open(os.DevNull)
	oldStderr := os.Stderr
	os.Stderr = devNull
	defer func() {
		os.Stderr = oldStderr
		devNull.Close()
	}()

	command := "add"
	args := []string{"--level", "error", "test message"}
	context := map[string]interface{}{
		"session": "test-session",
		"window":  "1",
		"pane":    "0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogCLICommand(command, args, context)
	}
}

// BenchmarkLogTUIAction measures the overhead of logging a TUI action.
func BenchmarkLogTUIAction(b *testing.B) {
	// Setup
	Reset()
	defer Reset()

	// Enable telemetry
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	if err != nil {
		b.Fatalf("failed to initialize telemetry: %v", err)
	}

	// Suppress stderr for benchmarks
	devNull, _ := os.Open(os.DevNull)
	oldStderr := os.Stderr
	os.Stderr = devNull
	defer func() {
		os.Stderr = oldStderr
		devNull.Close()
	}()

	action := "jump"
	context := map[string]interface{}{
		"notification_id": "123",
		"target":          "pane",
		"duration_ms":     150,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogTUIAction(action, context)
	}
}

// BenchmarkLogFeature measures the overhead of logging a generic feature.
func BenchmarkLogFeature(b *testing.B) {
	// Setup
	Reset()
	defer Reset()

	// Enable telemetry
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	if err != nil {
		b.Fatalf("failed to initialize telemetry: %v", err)
	}

	// Suppress stderr for benchmarks
	devNull, _ := os.Open(os.DevNull)
	oldStderr := os.Stderr
	os.Stderr = devNull
	defer func() {
		os.Stderr = oldStderr
		devNull.Close()
	}()

	feature := "test-feature"
	category := "cli"
	context := map[string]interface{}{
		"status": "success",
		"value":  42,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogFeature(feature, category, context)
	}
}

// BenchmarkLogFeatureDisabled measures the overhead when telemetry is disabled.
// This should be near-zero (just a config check).
func BenchmarkLogFeatureDisabled(b *testing.B) {
	// Setup
	Reset()
	defer Reset()

	// Telemetry disabled (default)

	feature := "test-feature"
	category := "cli"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogFeature(feature, category, nil)
	}
}

// BenchmarkStorageWrite measures the actual database write performance.
func BenchmarkStorageWrite(b *testing.B) {
	// Setup
	Reset()
	defer Reset()

	// Initialize storage
	err := storage.Init()
	if err != nil {
		b.Fatalf("failed to initialize storage: %v", err)
	}

	st, err := storage.NewFromConfig()
	if err != nil {
		b.Fatalf("failed to create storage: %v", err)
	}

	// Set storage instance for telemetry
	storageInstance = st

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timestamp := time.Now().Format(time.RFC3339)
		featureName := fmt.Sprintf("feature-%d", i)
		err := st.LogTelemetryEvent(timestamp, featureName, "cli", `{"test":"value"}`)
		if err != nil {
			b.Fatalf("failed to log event: %v", err)
		}
	}
}

// BenchmarkConcurrentLogCLICommand measures thread-safe concurrent logging.
func BenchmarkConcurrentLogCLICommand(b *testing.B) {
	// Setup
	Reset()
	defer Reset()

	// Enable telemetry
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	if err != nil {
		b.Fatalf("failed to initialize telemetry: %v", err)
	}

	// Suppress stderr for benchmarks
	devNull, _ := os.Open(os.DevNull)
	oldStderr := os.Stderr
	os.Stderr = devNull
	defer func() {
		os.Stderr = oldStderr
		devNull.Close()
	}()

	command := "add"
	args := []string{"--level", "error", "test"}
	context := map[string]interface{}{"session": "test"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			LogCLICommand(command, args, context)
		}
	})
}

// BenchmarkConcurrentLogTUIAction measures concurrent TUI logging.
func BenchmarkConcurrentLogTUIAction(b *testing.B) {
	// Setup
	Reset()
	defer Reset()

	// Enable telemetry
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	if err != nil {
		b.Fatalf("failed to initialize telemetry: %v", err)
	}

	// Suppress stderr for benchmarks
	devNull, _ := os.Open(os.DevNull)
	oldStderr := os.Stderr
	os.Stderr = devNull
	defer func() {
		os.Stderr = oldStderr
		devNull.Close()
	}()

	action := "jump"
	context := map[string]interface{}{"target": "pane"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			LogTUIAction(action, context)
		}
	})
}

// BenchmarkComplexContext measures overhead with complex nested context data.
func BenchmarkComplexContext(b *testing.B) {
	// Setup
	Reset()
	defer Reset()

	// Enable telemetry
	oldEnv := os.Getenv("TMUX_INTRAY_TELEMETRY_ENABLED")
	defer os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", oldEnv)
	os.Setenv("TMUX_INTRAY_TELEMETRY_ENABLED", "true")

	// Initialize
	err := Init()
	if err != nil {
		b.Fatalf("failed to initialize telemetry: %v", err)
	}

	// Suppress stderr for benchmarks
	devNull, _ := os.Open(os.DevNull)
	oldStderr := os.Stderr
	os.Stderr = devNull
	defer func() {
		os.Stderr = oldStderr
		devNull.Close()
	}()

	// Complex context with nested structures
	context := map[string]interface{}{
		"session": "main",
		"window":  "1",
		"pane":    "0",
		"notification": map[string]interface{}{
			"id":      "abc123",
			"level":   "error",
			"message": "something went wrong",
			"tags":    []string{"critical", "urgent"},
			"metadata": map[string]interface{}{
				"timestamp": "2026-03-15T10:30:00Z",
				"source":    "api",
				"retry":     3,
			},
		},
		"performance": map[string]interface{}{
			"duration_ms": 150,
			"memory_mb":   25,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogFeature("complex-operation", "cli", context)
	}
}

// BenchmarkChannelOperation measures just the channel send operation.
func BenchmarkChannelOperation(b *testing.B) {
	// Setup
	Reset()
	defer Reset()

	// Initialize storage manually
	err := storage.Init()
	if err != nil {
		b.Fatalf("failed to initialize storage: %v", err)
	}

	st, err := storage.NewFromConfig()
	if err != nil {
		b.Fatalf("failed to create storage: %v", err)
	}

	storageInstance = st

	// Create event channel
	eventChannel = make(chan *telemetryEvent, 100)

	event := &telemetryEvent{
		timestamp:       time.Now().Format(time.RFC3339),
		featureName:     "test",
		featureCategory: "cli",
		contextData:     `{"test":"value"}`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case eventChannel <- event:
			// Event sent
		default:
			// Channel full
		}
	}
}

// BenchmarkJSONMarshaling measures just the JSON marshaling overhead.
func BenchmarkJSONMarshaling(b *testing.B) {
	context := map[string]interface{}{
		"session": "test",
		"window":  "1",
		"pane":    "0",
		"level":   "error",
		"message": "test message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(context)
	}
}
