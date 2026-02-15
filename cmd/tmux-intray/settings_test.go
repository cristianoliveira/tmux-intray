package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type fakeSettingsClient struct {
	resetCalls  int
	resetErr    error
	resetResult *settings.Settings
	loadCalls   int
	loadErr     error
	loadResult  *settings.Settings
}

func (f *fakeSettingsClient) ResetSettings() (*settings.Settings, error) {
	f.resetCalls++
	return f.resetResult, f.resetErr
}

func (f *fakeSettingsClient) LoadSettings() (*settings.Settings, error) {
	f.loadCalls++
	return f.loadResult, f.loadErr
}

// captureStdout captures stdout during the execution of fn and returns the captured string.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	defer func() {
		os.Stdout = old
		_ = w.Close()
	}()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	// Regex to match ANSI escape sequences (CSI sequences)
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
}

func TestResetSettingsSuccess(t *testing.T) {
	client := &fakeSettingsClient{
		resetResult: settings.DefaultSettings(),
	}
	_, err := client.ResetSettings()
	require.NoError(t, err)
	require.Equal(t, 1, client.resetCalls)
}

func TestResetSettingsError(t *testing.T) {
	expectedErr := errors.New("storage error")
	client := &fakeSettingsClient{
		resetErr: expectedErr,
	}
	_, err := client.ResetSettings()
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Equal(t, 1, client.resetCalls)
}

func TestLoadSettingsSuccess(t *testing.T) {
	client := &fakeSettingsClient{
		loadResult: settings.DefaultSettings(),
	}
	_, err := client.LoadSettings()
	require.NoError(t, err)
	require.Equal(t, 1, client.loadCalls)
}

func TestLoadSettingsError(t *testing.T) {
	expectedErr := errors.New("load error")
	client := &fakeSettingsClient{
		loadErr: expectedErr,
	}
	_, err := client.LoadSettings()
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Equal(t, 1, client.loadCalls)
}

func TestResetSettingsMultipleCalls(t *testing.T) {
	client := &fakeSettingsClient{
		resetResult: settings.DefaultSettings(),
	}
	_, _ = client.ResetSettings()
	_, _ = client.ResetSettings()
	require.Equal(t, 2, client.resetCalls)
}

func TestLoadSettingsMultipleCalls(t *testing.T) {
	client := &fakeSettingsClient{
		loadResult: settings.DefaultSettings(),
	}
	_, _ = client.LoadSettings()
	_, _ = client.LoadSettings()
	require.Equal(t, 2, client.loadCalls)
}

func TestSettingsDefaults(t *testing.T) {
	defaults := settings.DefaultSettings()

	// Verify default columns are set
	require.NotNil(t, defaults.Columns)
	require.Greater(t, len(defaults.Columns), 0)

	// Verify default sort settings
	require.Equal(t, "timestamp", defaults.SortBy)
	require.Equal(t, "desc", defaults.SortOrder)

	// Verify default view mode
	require.Equal(t, settings.ViewModeGrouped, defaults.ViewMode)

	// Verify default grouping settings
	require.Equal(t, settings.GroupByNone, defaults.GroupBy)
	require.Equal(t, 1, defaults.DefaultExpandLevel)
	require.Equal(t, map[string]bool{}, defaults.ExpansionState)

	// Verify default filters are empty
	require.Equal(t, "", defaults.Filters.Level)
	require.Equal(t, "", defaults.Filters.State)
	require.Equal(t, "", defaults.Filters.Read)
	require.Equal(t, "", defaults.Filters.Session)
	require.Equal(t, "", defaults.Filters.Window)
	require.Equal(t, "", defaults.Filters.Pane)
}

func TestSettingsResetCommand(t *testing.T) {
	client := &fakeSettingsClient{
		resetResult: settings.DefaultSettings(),
	}
	cmd := NewSettingsCmd(client)
	// Find reset subcommand
	var resetCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "reset" {
			resetCmd = c
			break
		}
	}
	require.NotNil(t, resetCmd, "reset subcommand not found")
	// Set --force flag to skip confirmation
	err := resetCmd.Flags().Set("force", "true")
	require.NoError(t, err)
	// Run the command
	err = resetCmd.RunE(resetCmd, nil)
	require.NoError(t, err)
	require.Equal(t, 1, client.resetCalls)
}

func TestNewSettingsCmdPanicsWhenClientIsNil(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "expected panic, got nil")
		msg, ok := r.(string)
		require.True(t, ok, "panic message should be string")
		require.Contains(t, msg, "client dependency cannot be nil")
	}()

	NewSettingsCmd(nil)
}

func TestSettingsShowCommandSuccess(t *testing.T) {
	expectedSettings := settings.DefaultSettings()
	client := &fakeSettingsClient{
		loadResult: expectedSettings,
	}
	cmd := NewSettingsCmd(client)
	// Find show subcommand
	var showCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "show" {
			showCmd = c
			break
		}
	}
	require.NotNil(t, showCmd, "show subcommand not found")

	// Capture stdout during command execution
	var captured string
	runCmd := func() {
		err := showCmd.RunE(showCmd, nil)
		require.NoError(t, err)
	}
	captured = captureStdout(t, runCmd)

	// Strip ANSI color codes before parsing JSON
	captured = stripANSI(captured)
	captured = strings.TrimSpace(captured)
	// Verify output is valid JSON matching expected settings
	var actual settings.Settings
	err := json.Unmarshal([]byte(captured), &actual)
	require.NoError(t, err, "captured output should be valid JSON")
	require.Equal(t, *expectedSettings, actual)
	require.Equal(t, 1, client.loadCalls)
}

func TestSettingsShowCommandError(t *testing.T) {
	expectedErr := errors.New("load error")
	client := &fakeSettingsClient{
		loadErr: expectedErr,
	}
	cmd := NewSettingsCmd(client)
	var showCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "show" {
			showCmd = c
			break
		}
	}
	require.NotNil(t, showCmd, "show subcommand not found")

	err := showCmd.RunE(showCmd, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load settings")
	require.Equal(t, 1, client.loadCalls)
}

func TestSettingsResetCommandError(t *testing.T) {
	expectedErr := errors.New("storage error")
	client := &fakeSettingsClient{
		resetErr: expectedErr,
	}
	cmd := NewSettingsCmd(client)
	var resetCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "reset" {
			resetCmd = c
			break
		}
	}
	require.NotNil(t, resetCmd, "reset subcommand not found")
	// Set --force flag to skip confirmation
	err := resetCmd.Flags().Set("force", "true")
	require.NoError(t, err)
	// Run the command
	err = resetCmd.RunE(resetCmd, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to reset settings")
	require.Equal(t, 1, client.resetCalls)
}

func TestSettingsResetCommandWithEnvVarSkipsConfirmation(t *testing.T) {
	// Set CI environment variable to skip confirmation
	t.Setenv("CI", "true")
	client := &fakeSettingsClient{
		resetResult: settings.DefaultSettings(),
	}
	cmd := NewSettingsCmd(client)
	var resetCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "reset" {
			resetCmd = c
			break
		}
	}
	require.NotNil(t, resetCmd, "reset subcommand not found")
	// Do NOT set --force flag
	err := resetCmd.RunE(resetCmd, nil)
	require.NoError(t, err)
	require.Equal(t, 1, client.resetCalls)
}

func TestSettingsResetCommandWithBatsTmpdirSkipsConfirmation(t *testing.T) {
	// Set BATS_TMPDIR environment variable to skip confirmation
	t.Setenv("BATS_TMPDIR", "/tmp/bats")
	t.Setenv("CI", "") // Ensure CI is empty
	client := &fakeSettingsClient{
		resetResult: settings.DefaultSettings(),
	}
	cmd := NewSettingsCmd(client)
	var resetCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "reset" {
			resetCmd = c
			break
		}
	}
	require.NotNil(t, resetCmd, "reset subcommand not found")
	// Do NOT set --force flag
	err := resetCmd.RunE(resetCmd, nil)
	require.NoError(t, err)
	require.Equal(t, 1, client.resetCalls)
}

// Note: Testing interactive confirmation is complex and relies on stdin.
// We'll rely on integration tests for that.
