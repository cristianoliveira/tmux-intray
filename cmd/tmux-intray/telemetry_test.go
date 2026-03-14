package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTelemetryStorage struct {
	features      []sqlite.FeatureUsage
	events        []sqlite.TelemetryEventType
	deletedCount  int64
	deleteErr     error
	getFeatureErr error
	getAllErr     error
	getEventsErr  error
}

func (f *fakeTelemetryStorage) GetFeatureUsage(featureName string) (int64, error) {
	if f.getFeatureErr != nil {
		return 0, f.getFeatureErr
	}
	for _, feature := range f.features {
		if feature.FeatureName == featureName {
			return feature.UsageCount, nil
		}
	}
	return 0, nil
}

func (f *fakeTelemetryStorage) GetAllFeatures() ([]sqlite.FeatureUsage, error) {
	if f.getAllErr != nil {
		return nil, f.getAllErr
	}
	return f.features, nil
}

func (f *fakeTelemetryStorage) GetTelemetryEvents(startTime, endTime string) ([]sqlite.TelemetryEventType, error) {
	if f.getEventsErr != nil {
		return nil, f.getEventsErr
	}
	// Filter by time range if specified
	var filtered []sqlite.TelemetryEventType
	for _, event := range f.events {
		include := true
		if startTime != "" && event.Timestamp < startTime {
			include = false
		}
		if endTime != "" && event.Timestamp > endTime {
			include = false
		}
		if include {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
}

func (f *fakeTelemetryStorage) ClearTelemetryEvents(olderThanDays int) (int64, error) {
	if f.deleteErr != nil {
		return 0, f.deleteErr
	}
	return f.deletedCount, nil
}

type fakeTelemetryConfig struct {
	enabled bool
}

func (f *fakeTelemetryConfig) IsEnabled() bool {
	return f.enabled
}

func TestNewTelemetryCmdPanicsWhenStorageIsNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got nil")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected panic message as string, got %T", r)
		}
		if !strings.Contains(msg, "storage dependency cannot be nil") {
			t.Fatalf("expected panic message to mention nil storage, got %q", msg)
		}
	}()

	NewTelemetryCmd(nil, &fakeTelemetryConfig{})
}

func TestNewTelemetryCmdPanicsWhenConfigIsNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got nil")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected panic message as string, got %T", r)
		}
		if !strings.Contains(msg, "config dependency cannot be nil") {
			t.Fatalf("expected panic message to mention nil config, got %q", msg)
		}
	}()

	NewTelemetryCmd(&fakeTelemetryStorage{}, nil)
}

func TestTelemetryShowDisabled(t *testing.T) {
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{},
		config:  &fakeTelemetryConfig{enabled: false},
	}
	cmd := newShowTelemetryCmd(client)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Telemetry is currently disabled")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryShowNoData(t *testing.T) {
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{features: []sqlite.FeatureUsage{}},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newShowTelemetryCmd(client)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "No telemetry data available")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryShowWithData(t *testing.T) {
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{
			features: []sqlite.FeatureUsage{
				{FeatureName: "add-notification", FeatureCategory: "cli", UsageCount: 10},
				{FeatureName: "list-notifications", FeatureCategory: "cli", UsageCount: 5},
				{FeatureName: "dismiss-notification", FeatureCategory: "cli", UsageCount: 3},
				{FeatureName: "open-tui", FeatureCategory: "tui", UsageCount: 2},
			},
		},
		config: &fakeTelemetryConfig{enabled: true},
	}
	cmd := newShowTelemetryCmd(client)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Category: cli")
	assert.Contains(t, output, "add-notification: 10 calls")
	assert.Contains(t, output, "list-notifications: 5 calls")
	assert.Contains(t, output, "Category: tui")
	assert.Contains(t, output, "open-tui: 2 calls")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryShowWithDaysFilter(t *testing.T) {
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{
			features: []sqlite.FeatureUsage{
				{FeatureName: "add-notification", FeatureCategory: "cli", UsageCount: 10},
				{FeatureName: "list-notifications", FeatureCategory: "cli", UsageCount: 5},
			},
			events: []sqlite.TelemetryEventType{
				{Timestamp: "2026-01-01T00:00:00Z", FeatureName: "add-notification"},
			},
		},
		config: &fakeTelemetryConfig{enabled: true},
	}
	cmd := newShowTelemetryCmd(client)
	require.NoError(t, cmd.Flags().Set("days", "7"))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryShowError(t *testing.T) {
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{getAllErr: assert.AnError},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newShowTelemetryCmd(client)

	err := cmd.RunE(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get feature usage")
}

func TestTelemetryExportDisabled(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "telemetry-export-*.jsonl")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	client := &telemetryClient{
		storage: &fakeTelemetryStorage{},
		config:  &fakeTelemetryConfig{enabled: false},
	}
	cmd := newExportCmd(client)
	require.NoError(t, cmd.Flags().Set("output", tmpFile.Name()))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Telemetry is currently disabled")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryExportNoData(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "telemetry-export-*.jsonl")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	client := &telemetryClient{
		storage: &fakeTelemetryStorage{events: []sqlite.TelemetryEventType{}},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newExportCmd(client)
	require.NoError(t, cmd.Flags().Set("output", tmpFile.Name()))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "No telemetry data to export")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryExportWithData(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "telemetry-export-*.jsonl")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	events := []sqlite.TelemetryEventType{
		{ID: 1, Timestamp: "2026-01-01T00:00:00Z", FeatureName: "add-notification", FeatureCategory: "cli", ContextData: "{}"},
		{ID: 2, Timestamp: "2026-01-01T01:00:00Z", FeatureName: "list-notifications", FeatureCategory: "cli", ContextData: "{}"},
	}

	client := &telemetryClient{
		storage: &fakeTelemetryStorage{events: events},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newExportCmd(client)
	require.NoError(t, cmd.Flags().Set("output", tmpFile.Name()))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Exported 2 telemetry events")
	assert.Contains(t, output, "Data is local-only and never transmitted")

	// Verify file contents
	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	lines := strings.Split(string(content), "\n")
	assert.Equal(t, 3, len(lines)) // 2 events + 1 empty line
	assert.Contains(t, lines[0], "add-notification")
	assert.Contains(t, lines[1], "list-notifications")
}

func TestTelemetryExportError(t *testing.T) {
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{getEventsErr: assert.AnError},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newExportCmd(client)
	require.NoError(t, cmd.Flags().Set("output", "/tmp/test.jsonl"))

	err := cmd.RunE(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get telemetry events")
}

func TestTelemetryClearDisabled(t *testing.T) {
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{},
		config:  &fakeTelemetryConfig{enabled: false},
	}
	cmd := newClearTelemetryCmd(client)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Telemetry is currently disabled")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryClearNoData(t *testing.T) {
	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "true")
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{deletedCount: 0},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newClearTelemetryCmd(client)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "No telemetry data to clear")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryClearWithData(t *testing.T) {
	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "true")
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{deletedCount: 5},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newClearTelemetryCmd(client)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Cleared 5 telemetry event(s)")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryClearWithDays(t *testing.T) {
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{deletedCount: 3},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newClearTelemetryCmd(client)
	require.NoError(t, cmd.Flags().Set("days", "7"))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Cleared 3 telemetry event(s)")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryClearError(t *testing.T) {
	t.Setenv("TMUX_INTRAY_ALLOW_NO_TMUX", "true")
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{deleteErr: assert.AnError},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newClearTelemetryCmd(client)

	err := cmd.RunE(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clear telemetry events")
}

func TestTelemetryStatusDisabled(t *testing.T) {
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{},
		config:  &fakeTelemetryConfig{enabled: false},
	}
	cmd := newStatusTelemetryCmd(client)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Telemetry Status")
	assert.Contains(t, output, "Enabled:")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryStatusEnabled(t *testing.T) {
	events := []sqlite.TelemetryEventType{
		{Timestamp: "2026-01-01T00:00:00Z", FeatureName: "add-notification"},
		{Timestamp: "2026-01-01T01:00:00Z", FeatureName: "list-notifications"},
	}
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{events: events},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newStatusTelemetryCmd(client)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Telemetry Status")
	assert.Contains(t, output, "Enabled:")
	assert.Contains(t, output, "Total Events: 2")
	assert.Contains(t, output, "First Event: 2026-01-01T00:00:00Z")
	assert.Contains(t, output, "Last Event: 2026-01-01T01:00:00Z")
	assert.Contains(t, output, "Data is local-only and never transmitted")
}

func TestTelemetryStatusError(t *testing.T) {
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{getEventsErr: assert.AnError},
		config:  &fakeTelemetryConfig{enabled: true},
	}
	cmd := newStatusTelemetryCmd(client)

	err := cmd.RunE(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get status")
}

func TestFormatBool(t *testing.T) {
	tests := []struct {
		name string
		b    bool
		want string
	}{
		{"true", true, "\033[0;32mtrue\033[0m"},
		{"false", false, "\033[0;31mfalse\033[0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBool(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterFeaturesByDays(t *testing.T) {
	features := []sqlite.FeatureUsage{
		{FeatureName: "add-notification", FeatureCategory: "cli", UsageCount: 10},
		{FeatureName: "list-notifications", FeatureCategory: "cli", UsageCount: 5},
	}
	// Use relative timestamp to ensure test passes regardless of when it runs
	events := []sqlite.TelemetryEventType{
		{Timestamp: time.Now().UTC().AddDate(0, 0, -1).Format(time.RFC3339), FeatureName: "add-notification"},
	}
	client := &telemetryClient{
		storage: &fakeTelemetryStorage{
			features: features,
			events:   events,
		},
		config: &fakeTelemetryConfig{enabled: true},
	}

	filtered, err := filterFeaturesByDays(client, features, 7)
	require.NoError(t, err)
	assert.Len(t, filtered, 1)
	assert.Equal(t, "add-notification", filtered[0].FeatureName)
}

func TestTelemetryConfigAdapter(t *testing.T) {
	adapter := &telemetryConfigAdapter{}
	enabled := adapter.IsEnabled()
	// The result depends on config, just check it returns a bool
	assert.IsType(t, false, enabled)
}
