/*
Copyright © 2026 Cristian Oliveira <license@cristianoliveira.dev>
*/
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/ports"
	"github.com/spf13/cobra"
)

const checkmark = "✓"

// TelemetryStorage is an alias to the ports interface.
type TelemetryStorage = ports.TelemetryStorage

// TelemetryStatus represents the current status of telemetry.
type TelemetryStatus struct {
	Enabled      bool
	TotalEvents  int64
	FirstEvent   string
	LastEvent    string
	DatabaseSize int64
}

type telemetryClient struct {
	storage TelemetryStorage
	config  TelemetryConfig
}

// TelemetryConfig defines the interface for checking telemetry configuration.
type TelemetryConfig interface {
	IsEnabled() bool
}

// NewTelemetryCmd creates the telemetry command with explicit dependencies.
func NewTelemetryCmd(storage TelemetryStorage, config TelemetryConfig) *cobra.Command {
	if storage == nil {
		panic("NewTelemetryCmd: storage dependency cannot be nil")
	}
	if config == nil {
		panic("NewTelemetryCmd: config dependency cannot be nil")
	}

	client := &telemetryClient{
		storage: storage,
		config:  config,
	}

	const telemetryCommandLong = `Manage telemetry data and settings.

USAGE:
    tmux-intray telemetry <subcommand>

SUBCOMMANDS:
    show      Display feature usage summary
    export    Export telemetry data to JSONL format
    clear     Clear old telemetry data
    status    Show telemetry status

PRIVACY:
    Data is local-only and never transmitted`

	telemetryCmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Manage telemetry data and settings",
		Long:  telemetryCommandLong,
	}

	showCmd := newShowTelemetryCmd(client)
	exportCmd := newExportCmd(client)
	clearCmd := newClearTelemetryCmd(client)
	statusCmd := newStatusTelemetryCmd(client)

	telemetryCmd.AddCommand(showCmd)
	telemetryCmd.AddCommand(exportCmd)
	telemetryCmd.AddCommand(clearCmd)
	telemetryCmd.AddCommand(statusCmd)

	return telemetryCmd
}

// newShowTelemetryCmd creates the show subcommand.
func newShowTelemetryCmd(client *telemetryClient) *cobra.Command {
	var showDays int

	showCmd := &cobra.Command{
		Use:   "show [--days N]",
		Short: "Display feature usage summary",
		Long: `Display feature usage summary grouped by feature name and category.

USAGE:
    tmux-intray telemetry show [--days N]

OPTIONS:
    --days N    Show usage from the last N days (default: all time)

PRIVACY:
    Data is local-only and never transmitted`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShowTelemetryCmd(cmd, client, showDays)
		},
	}
	showCmd.Flags().IntVar(&showDays, "days", 0, "Show usage from the last N days (default: all time)")

	return showCmd
}

// runShowTelemetryCmd executes the show subcommand.
func runShowTelemetryCmd(cmd *cobra.Command, client *telemetryClient, days int) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	if !client.config.IsEnabled() {
		_, _ = fmt.Fprintln(out, colors.Blue, "Telemetry is currently disabled", colors.Reset)
		_, _ = fmt.Fprintln(errOut, colors.Yellow, "Warning:", colors.Reset, "Data is local-only and never transmitted")
		return nil
	}

	// Get all features with usage stats
	features, err := client.storage.GetAllFeatures()
	if err != nil {
		return fmt.Errorf("telemetry show: failed to get feature usage: %w", err)
	}

	if len(features) == 0 {
		_, _ = fmt.Fprintln(out, colors.Blue, "No telemetry data available", colors.Reset)
		_, _ = fmt.Fprintln(errOut, colors.Yellow, "Warning:", colors.Reset, "Data is local-only and never transmitted")
		return nil
	}

	// Filter by days if specified
	if days > 0 {
		features, err = filterFeaturesByDays(client, features, days)
		if err != nil {
			return fmt.Errorf("telemetry show: failed to filter by days: %w", err)
		}
		if len(features) == 0 {
			_, _ = fmt.Fprintln(out, colors.Blue, "No telemetry data available in the specified time range", colors.Reset)
			_, _ = fmt.Fprintln(errOut, colors.Yellow, "Warning:", colors.Reset, "Data is local-only and never transmitted")
			return nil
		}
	}

	// Group by category
	categoryMap := make(map[string][]ports.FeatureUsage)
	for _, feature := range features {
		categoryMap[feature.FeatureCategory] = append(categoryMap[feature.FeatureCategory], feature)
	}

	// Sort categories alphabetically
	categories := make([]string, 0, len(categoryMap))
	for category := range categoryMap {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	// Display results
	_, _ = fmt.Fprintln(out)
	for _, category := range categories {
		_, _ = fmt.Fprintf(out, "%sCategory: %s%s\n", colors.Yellow, category, colors.Reset)
		categoryFeatures := categoryMap[category]
		// Sort by usage count (descending)
		sort.Slice(categoryFeatures, func(i, j int) bool {
			return categoryFeatures[i].UsageCount > categoryFeatures[j].UsageCount
		})
		for _, feature := range categoryFeatures {
			_, _ = fmt.Fprintf(out, "  %s: %d calls\n", feature.FeatureName, feature.UsageCount)
		}
		_, _ = fmt.Fprintln(out)
	}
	_, _ = fmt.Fprintln(errOut, colors.Yellow, "Warning:", colors.Reset, "Data is local-only and never transmitted")

	return nil
}

// filterFeaturesByDays filters features to only include those with usage in the specified time range.
func filterFeaturesByDays(client *telemetryClient, features []ports.FeatureUsage, days int) ([]ports.FeatureUsage, error) {
	startTime := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02T15:04:05Z")

	// Get events in the time range
	events, err := client.storage.GetTelemetryEvents(startTime, "")
	if err != nil {
		return nil, err
	}

	// Create a set of feature names that have events in the time range
	featureSet := make(map[string]bool)
	for _, event := range events {
		featureSet[event.FeatureName] = true
	}

	// Filter features
	var filtered []ports.FeatureUsage
	for _, feature := range features {
		if featureSet[feature.FeatureName] {
			filtered = append(filtered, feature)
		}
	}

	return filtered, nil
}

// newExportCmd creates the export subcommand.
func newExportCmd(client *telemetryClient) *cobra.Command {
	var exportOutput string

	exportCmd := &cobra.Command{
		Use:   "export --output FILE",
		Short: "Export telemetry data to JSONL format",
		Long: `Export telemetry data to JSONL format (one JSON object per line).

USAGE:
    tmux-intray telemetry export --output FILE

OPTIONS:
    --output FILE    Output file path (required)

PRIVACY:
    Data is local-only and never transmitted`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExportCmd(cmd, client, exportOutput)
		},
	}
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (required)")
	_ = exportCmd.MarkFlagRequired("output")

	return exportCmd
}

// runExportCmd executes the export subcommand.
func runExportCmd(cmd *cobra.Command, client *telemetryClient, outputPath string) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	if !client.config.IsEnabled() {
		_, _ = fmt.Fprintln(out, colors.Blue, "Telemetry is currently disabled", colors.Reset)
		_, _ = fmt.Fprintln(errOut, colors.Yellow, "Warning:", colors.Reset, "Data is local-only and never transmitted")
		return nil
	}

	// Get all events
	events, err := client.storage.GetTelemetryEvents("", "")
	if err != nil {
		return fmt.Errorf("telemetry export: failed to get telemetry events: %w", err)
	}

	if len(events) == 0 {
		_, _ = fmt.Fprintln(out, colors.Blue, "No telemetry data to export", colors.Reset)
		_, _ = fmt.Fprintln(errOut, colors.Yellow, "Warning:", colors.Reset, "Data is local-only and never transmitted")
		return nil
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("telemetry export: failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	writer := bufio.NewWriter(file)
	defer func() { _ = writer.Flush() }()

	// Write each event as a JSON line
	for _, event := range events {
		line, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("telemetry export: failed to marshal event: %w", err)
		}
		if _, err := writer.Write(append(line, '\n')); err != nil {
			return fmt.Errorf("telemetry export: failed to write event: %w", err)
		}
	}

	_, _ = fmt.Fprintf(out, "%s%s %s%s%s\n", colors.Green, checkmark, colors.Reset, fmt.Sprintf("Exported %d telemetry events to %s", len(events), outputPath), colors.Reset)
	_, _ = fmt.Fprintln(errOut, colors.Yellow, "Warning:", colors.Reset, "Data is local-only and never transmitted")

	return nil
}

// newClearTelemetryCmd creates the clear subcommand.
func newClearTelemetryCmd(client *telemetryClient) *cobra.Command {
	var clearDays int

	clearCmd := &cobra.Command{
		Use:   "clear [--days N]",
		Short: "Clear old telemetry data",
		Long: `Clear telemetry data older than N days.

USAGE:
    tmux-intray telemetry clear [--days N]

OPTIONS:
    --days N    Clear events older than N days (default: 90)

PRIVACY:
    Data is local-only and never transmitted`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClearTelemetryCmd(cmd, client, clearDays)
		},
	}
	clearCmd.Flags().IntVar(&clearDays, "days", 90, "Clear events older than N days (default: 90)")

	return clearCmd
}

// runClearTelemetryCmd executes the clear subcommand.
func runClearTelemetryCmd(cmd *cobra.Command, client *telemetryClient, days int) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	if !client.config.IsEnabled() {
		_, _ = fmt.Fprintln(out, colors.Blue, "Telemetry is currently disabled", colors.Reset)
		_, _ = fmt.Fprintln(errOut, colors.Yellow, "Warning:", colors.Reset, "Data is local-only and never transmitted")
		return nil
	}

	// If --days is 0, we need to confirm clearing all data
	if days == 0 {
		// Skip confirmation in CI/test environment
		if !allowTmuxlessMode() {
			if !confirmClearTelemetry() {
				_, _ = fmt.Fprintln(out, colors.Blue, "Operation cancelled", colors.Reset)
				return nil
			}
		}
	}

	// Clear telemetry events
	deleted, err := client.storage.ClearTelemetryEvents(days)
	if err != nil {
		return fmt.Errorf("telemetry clear: failed to clear telemetry events: %w", err)
	}

	if deleted == 0 {
		_, _ = fmt.Fprintln(out, colors.Blue, "No telemetry data to clear", colors.Reset)
	} else {
		_, _ = fmt.Fprintf(out, "%s%s %s%s%s\n", colors.Green, checkmark, colors.Reset, fmt.Sprintf("Cleared %d telemetry event(s)", deleted), colors.Reset)
	}
	_, _ = fmt.Fprintln(errOut, colors.Yellow, "Warning:", colors.Reset, "Data is local-only and never transmitted")

	return nil
}

// confirmClearTelemetry asks the user for confirmation before clearing telemetry data.
func confirmClearTelemetry() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Are you sure you want to clear all telemetry data? (y/N): ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read, assume no
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

// newStatusTelemetryCmd creates the status subcommand.
func newStatusTelemetryCmd(client *telemetryClient) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show telemetry status",
		Long: `Show telemetry status including enabled/disabled state, total events, timestamps, and database size.

USAGE:
    tmux-intray telemetry status

PRIVACY:
    Data is local-only and never transmitted`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusTelemetryCmd(cmd, client)
		},
	}
}

// runStatusTelemetryCmd executes the status subcommand.
func runStatusTelemetryCmd(cmd *cobra.Command, client *telemetryClient) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	status, err := getTelemetryStatus(client)
	if err != nil {
		return fmt.Errorf("telemetry status: failed to get status: %w", err)
	}

	// Display status
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Telemetry Status")
	_, _ = fmt.Fprintln(out, "------------------")
	_, _ = fmt.Fprintf(out, "Enabled: %s\n", formatBool(status.Enabled))
	_, _ = fmt.Fprintf(out, "Total Events: %d\n", status.TotalEvents)
	if status.TotalEvents > 0 {
		_, _ = fmt.Fprintf(out, "First Event: %s\n", status.FirstEvent)
		_, _ = fmt.Fprintf(out, "Last Event: %s\n", status.LastEvent)
	}
	if status.DatabaseSize > 0 {
		_, _ = fmt.Fprintf(out, "Database Size: %d bytes\n", status.DatabaseSize)
	}
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(errOut, colors.Yellow, "Warning:", colors.Reset, "Data is local-only and never transmitted")

	return nil
}

// getTelemetryStatus retrieves the current telemetry status.
func getTelemetryStatus(client *telemetryClient) (*TelemetryStatus, error) {
	status := &TelemetryStatus{
		Enabled: client.config.IsEnabled(),
	}

	if !status.Enabled {
		return status, nil
	}

	// Get all events to count and find timestamps
	events, err := client.storage.GetTelemetryEvents("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get telemetry events: %w", err)
	}

	status.TotalEvents = int64(len(events))

	if len(events) > 0 {
		// Find first and last events
		// Events are returned in timestamp order
		status.FirstEvent = events[0].Timestamp
		status.LastEvent = events[len(events)-1].Timestamp
	}

	// Get database size (simplified - in a real implementation, we'd query the file size)
	status.DatabaseSize = 0 // Placeholder for now

	return status, nil
}

// formatBool returns a colored string representation of a boolean.
func formatBool(b bool) string {
	if b {
		return fmt.Sprintf("%strue%s", colors.Green, colors.Reset)
	}
	return fmt.Sprintf("%sfalse%s", colors.Red, colors.Reset)
}

// telemetryConfigAdapter adapts the config package to TelemetryConfig interface.
type telemetryConfigAdapter struct{}

func (t *telemetryConfigAdapter) IsEnabled() bool {
	config.Load()
	return config.GetBool("telemetry_enabled", false)
}
