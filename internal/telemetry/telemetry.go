// Package telemetry provides feature usage tracking for tmux-intray.
// This is the client-facing API for logging feature usage events.
//
// Privacy guarantees:
// - No network calls (local storage only)
// - No personal identifiers in context
// - Local-only storage
// - Data only used for feature usage analysis
package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

const (
	// FeatureCategoryCLI is the category for CLI commands.
	FeatureCategoryCLI = "cli"
	// FeatureCategoryTUI is the category for TUI actions.
	FeatureCategoryTUI = "tui"

	// eventChannelSize is the buffer size for the async event channel.
	eventChannelSize = 100
)

var (
	// storageInstance holds the storage backend for telemetry logging.
	storageInstance storage.Storage

	// eventChannel is a buffered channel for async event logging.
	eventChannel chan *telemetryEvent

	// initOnce ensures Init() is only called once.
	initOnce = &sync.Once{}

	// initErr holds any initialization error.
	initErr error

	// initMu protects initOnce and initErr for concurrent access during Reset().
	initMu sync.Mutex

	// eventProcessorStarted indicates if the event processor goroutine is running.
	eventProcessorStarted bool

	// eventProcessorDone is used to wait for the event processor goroutine to finish.
	eventProcessorDone chan struct{}

	// eventProcessorMu protects access to event processor state.
	eventProcessorMu sync.Mutex
)

// telemetryEvent represents a single telemetry event to be logged.
type telemetryEvent struct {
	timestamp       string
	featureName     string
	featureCategory string
	contextData     string
}

// Init initializes the telemetry system with a storage backend.
// This should be called once during application startup.
// Returns an error if storage initialization fails.
func Init() error {
	initOnce.Do(func() {
		// Initialize storage if not already done
		if err := storage.Init(); err != nil {
			initErr = fmt.Errorf("telemetry: storage initialization failed: %w", err)
			return
		}

		// Get storage instance
		var err error
		storageInstance, err = storage.NewFromConfig()
		if err != nil {
			initErr = fmt.Errorf("telemetry: failed to create storage: %w", err)
			return
		}

		// Start event processor goroutine if not already running
		startEventProcessor()
	})

	return initErr
}

// IsEnabled returns true if telemetry is enabled via configuration.
func IsEnabled() bool {
	// Load config to ensure we have the latest value
	config.Load()
	return config.GetBool("telemetry_enabled", false)
}

// LogCLICommand logs a CLI command invocation.
// This is a no-op if telemetry is disabled.
// Errors are logged to stderr and not returned to the caller.
func LogCLICommand(command string, args []string, context map[string]interface{}) {
	if !IsEnabled() {
		return
	}

	// Build context with command and args
	mergedContext := make(map[string]interface{})
	for k, v := range context {
		mergedContext[k] = v
	}
	mergedContext["command"] = command
	mergedContext["args"] = args

	// Send event to channel (non-blocking)
	event := &telemetryEvent{
		timestamp:       time.Now().Format(time.RFC3339),
		featureName:     command,
		featureCategory: FeatureCategoryCLI,
	}
	if contextJSON, err := json.Marshal(mergedContext); err != nil {
		// If JSON marshaling fails, use empty object
		event.contextData = "{}"
		fmt.Fprintf(os.Stderr, "telemetry: failed to marshal context for CLI command '%s': %v\n", command, err)
	} else {
		event.contextData = string(contextJSON)
	}

	sendEvent(event)
}

// LogTUIAction logs a TUI action.
// This is a no-op if telemetry is disabled.
// Errors are logged to stderr and not returned to the caller.
func LogTUIAction(action string, context map[string]interface{}) {
	if !IsEnabled() {
		return
	}

	// Build context with action
	mergedContext := make(map[string]interface{})
	for k, v := range context {
		mergedContext[k] = v
	}
	mergedContext["action"] = action

	// Send event to channel (non-blocking)
	event := &telemetryEvent{
		timestamp:       time.Now().Format(time.RFC3339),
		featureName:     action,
		featureCategory: FeatureCategoryTUI,
	}
	if contextJSON, err := json.Marshal(mergedContext); err != nil {
		// If JSON marshaling fails, use empty object
		event.contextData = "{}"
		fmt.Fprintf(os.Stderr, "telemetry: failed to marshal context for TUI action '%s': %v\n", action, err)
	} else {
		event.contextData = string(contextJSON)
	}

	sendEvent(event)
}

// LogFeature logs a feature usage event.
// This is a no-op if telemetry is disabled.
// Errors are logged to stderr and not returned to the caller.
func LogFeature(feature, category string, context map[string]interface{}) {
	if !IsEnabled() {
		return
	}

	if feature == "" {
		fmt.Fprintf(os.Stderr, "telemetry: feature name cannot be empty, skipping log\n")
		return
	}
	if category == "" {
		fmt.Fprintf(os.Stderr, "telemetry: feature category cannot be empty, defaulting to 'cli'\n")
		category = FeatureCategoryCLI
	}

	// Send event to channel (non-blocking)
	event := &telemetryEvent{
		timestamp:       time.Now().Format(time.RFC3339),
		featureName:     feature,
		featureCategory: category,
	}
	if contextJSON, err := json.Marshal(context); err != nil {
		// If JSON marshaling fails, use empty object
		event.contextData = "{}"
		fmt.Fprintf(os.Stderr, "telemetry: failed to marshal context for feature '%s': %v\n", feature, err)
	} else {
		event.contextData = string(contextJSON)
	}

	sendEvent(event)
}

// sendEvent sends an event to the channel for async processing.
// This is non-blocking to avoid blocking the caller.
func sendEvent(event *telemetryEvent) {
	if eventChannel == nil {
		// Event channel not initialized, try to start it
		startEventProcessor()
	}

	select {
	case eventChannel <- event:
		// Event sent successfully
	default:
		// Channel full, drop event to avoid blocking
		fmt.Fprintf(os.Stderr, "telemetry: event channel full, dropping event for feature '%s'\n", event.featureName)
	}
}

// startEventProcessor starts the event processor goroutine if not already running.
// This is safe to call multiple times.
func startEventProcessor() {
	eventProcessorMu.Lock()
	defer eventProcessorMu.Unlock()

	if eventProcessorStarted {
		return
	}

	// Initialize event channel
	eventChannel = make(chan *telemetryEvent, eventChannelSize)

	// Initialize done channel
	eventProcessorDone = make(chan struct{})

	// Start event processor goroutine
	go eventProcessor()

	eventProcessorStarted = true
}

// eventProcessor processes telemetry events from the channel and writes them to storage.
// This runs in a goroutine and batches events for better performance.
func eventProcessor() {
	for event := range eventChannel {
		initMu.Lock()
		storage := storageInstance
		initMu.Unlock()

		if storage == nil {
			fmt.Fprintf(os.Stderr, "telemetry: storage not initialized, dropping event for feature '%s'\n", event.featureName)
			continue
		}

		// Log the event to storage using the interface
		if err := storage.LogTelemetryEvent(event.timestamp, event.featureName, event.featureCategory, event.contextData); err != nil {
			fmt.Fprintf(os.Stderr, "telemetry: failed to log event for feature '%s': %v\n", event.featureName, err)
		}
	}
	// Signal that the processor is done
	close(eventProcessorDone)
}

// Shutdown gracefully shuts down the telemetry system.
// This flushes any remaining events and closes the channel.
// This should be called during application shutdown.
func Shutdown() {
	eventProcessorMu.Lock()
	defer eventProcessorMu.Unlock()

	if !eventProcessorStarted {
		return
	}

	// Close the channel, which will cause eventProcessor to exit
	if eventChannel != nil {
		close(eventChannel)
	}

	// Wait for event processor to finish (with timeout)
	if eventProcessorDone != nil {
		select {
		case <-eventProcessorDone:
			// Event processor finished
		case <-time.After(1 * time.Second):
			// Timeout - event processor didn't finish in time
			fmt.Fprintf(os.Stderr, "telemetry: warning - event processor did not shut down gracefully\n")
		}
		eventProcessorDone = nil
	}

	eventProcessorStarted = false
}

// Reset resets the telemetry package state for testing.
// This should only be called in tests.
func Reset() {
	Shutdown()
	initMu.Lock()
	defer initMu.Unlock()
	initOnce = &sync.Once{}
	initErr = nil
	storageInstance = nil
}
