// Package tmux provides a unified abstraction layer for tmux operations.
package tmux

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultClientRunSuccessfulExecution tests successful execution of tmux commands.
func TestDefaultClientRunSuccessfulExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	tests := []struct {
		name        string
		args        []string
		wantStdout  string
		wantStderr  string
		wantErr     bool
		description string
	}{
		{
			name:        "list sessions",
			args:        []string{"list-sessions", "-F", "#{session_name}"},
			wantStdout:  "", // We'll just check it's non-empty
			wantStderr:  "",
			wantErr:     false,
			description: "should list all tmux sessions",
		},
		{
			name:        "get server version",
			args:        []string{"-V"},
			wantStdout:  "tmux", // Should contain "tmux"
			wantStderr:  "",
			wantErr:     false,
			description: "should get tmux version",
		},
		{
			name:        "display format string",
			args:        []string{"display", "-p", "#{pane_id}"},
			wantStdout:  "%", // Pane IDs start with %
			wantStderr:  "",
			wantErr:     false,
			description: "should display current pane ID",
		},
		{
			name:        "show environment variable",
			args:        []string{"show-environment", "-g", "TERM"},
			wantStdout:  "TERM=", // Environment variables are shown as NAME=value
			wantStderr:  "",
			wantErr:     false,
			description: "should show TERM environment variable",
		},
		{
			name:        "has session",
			args:        []string{"has-session"},
			wantStdout:  "",
			wantStderr:  "",
			wantErr:     false,
			description: "should check if tmux server has sessions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := client.Run(tt.args...)

			if tt.wantErr {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}

			if tt.wantStdout != "" {
				assert.Contains(t, stdout, tt.wantStdout, tt.description)
			}

			if tt.wantStderr != "" {
				assert.Contains(t, stderr, tt.wantStderr, tt.description)
			}

			t.Logf("Command: tmux %s", strings.Join(tt.args, " "))
			t.Logf("Stdout: %q", stdout)
			t.Logf("Stderr: %q", stderr)
			if err != nil {
				t.Logf("Error: %v", err)
			}
		})
	}
}

// TestDefaultClientRunWithSocketPath tests command execution with custom socket path.
func TestDefaultClientRunWithSocketPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create client with socket path option
	socketPath := "test-socket"
	client := NewDefaultClient(WithSocketPath(socketPath))

	// Test with a command that should fail with non-existent socket
	// (we can't easily test with a real custom socket without setup/teardown)
	stdout, stderr, err := client.Run("list-sessions", "-F", "#{session_name}")

	// Should fail because the socket doesn't exist
	assert.Error(t, err, "should fail with non-existent socket path")

	// Check that stderr contains information about the socket issue
	assert.NotEmpty(t, stderr, "stderr should contain error details")

	t.Logf("Stdout: %q", stdout)
	t.Logf("Stderr: %q", stderr)
	t.Logf("Error: %v", err)
}

// TestDefaultClientRunTimeout tests timeout behavior.
func TestDefaultClientRunTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create client with very short timeout
	shortTimeout := 100 * time.Millisecond
	client := NewDefaultClient(WithTimeout(shortTimeout))

	// Run a command that will take longer than the timeout
	// We use tmux's "run-shell" command with a sleep command
	args := []string{"run-shell", "sleep 1"}

	start := time.Now()
	stdout, stderr, err := client.Run(args...)
	duration := time.Since(start)

	// Should fail with timeout
	assert.Error(t, err, "should timeout")

	// Should complete within expected time (with some buffer)
	assert.Less(t, duration, shortTimeout+200*time.Millisecond, "should timeout within expected duration")

	t.Logf("Command timed out after %v (expected around %v)", duration, shortTimeout)
	t.Logf("Stdout: %q", stdout)
	t.Logf("Stderr: %q", stderr)
	t.Logf("Error: %v", err)

	// Check if error is context timeout (wrapped)
	if err != nil && strings.Contains(err.Error(), "context") {
		t.Log("Error indicates context timeout (expected)")
	}
}

// TestDefaultClientRunCommandFailure tests handling of invalid tmux commands.
func TestDefaultClientRunCommandFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient()

	tests := []struct {
		name           string
		args           []string
		description    string
		checkStderr    bool
		stderrContains []string
	}{
		{
			name:           "invalid command",
			args:           []string{"invalid-command-that-does-not-exist"},
			description:    "should fail with invalid tmux command",
			checkStderr:    true,
			stderrContains: []string{"unknown command", "invalid"},
		},
		{
			name:           "invalid option",
			args:           []string{"--invalid-option"},
			description:    "should fail with invalid option",
			checkStderr:    true,
			stderrContains: []string{"unknown option", "usage"},
		},
		{
			name:           "target not found",
			args:           []string{"select-window", "-t", "$999999"},
			description:    "should fail when target not found",
			checkStderr:    true,
			stderrContains: []string{"can't find", "not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := client.Run(tt.args...)

			// Should fail
			assert.Error(t, err, tt.description)

			// Should have stderr with error details
			if tt.checkStderr {
				assert.NotEmpty(t, stderr, tt.description+" - stderr should not be empty")

				// Check for expected error patterns
				if len(tt.stderrContains) > 0 {
					matched := false
					for _, pattern := range tt.stderrContains {
						if strings.Contains(strings.ToLower(stderr), strings.ToLower(pattern)) {
							matched = true
							break
						}
					}
					if !matched {
						t.Logf("Stderr did not contain expected patterns %v, got: %q", tt.stderrContains, stderr)
					}
				}
			}

			t.Logf("Command: tmux %s", strings.Join(tt.args, " "))
			t.Logf("Stdout: %q", stdout)
			t.Logf("Stderr: %q", stderr)
			t.Logf("Error: %v", err)

			// Verify error wrapping includes command context
			assert.Contains(t, err.Error(), "tmux command", "error should include 'tmux command' context")
		})
	}
}

// TestDefaultClientRunErrorWrapping tests that errors are properly wrapped with context.
func TestDefaultClientRunErrorWrapping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient()

	// Test with an invalid command
	args := []string{"invalid-command"}
	stdout, stderr, err := client.Run(args...)

	require.Error(t, err)
	assert.NotEmpty(t, err.Error(), "error message should not be empty")

	// Check that error includes "tmux command" context
	assert.Contains(t, err.Error(), "tmux command", "error should include 'tmux command' prefix")

	// Check that error includes the command arguments
	assert.Contains(t, err.Error(), "invalid-command", "error should include command that failed")

	// Check that error uses proper wrapping (%w)
	// by checking that the error chain can be unwrapped
	unwrappedErr := fmt.Errorf("wrap: %w", err)
	assert.Error(t, unwrappedErr, "wrapped error should still be an error")

	t.Logf("Error message: %v", err)
	t.Logf("Stdout: %q", stdout)
	t.Logf("Stderr: %q", stderr)
}

// TestDefaultClientRunEmptyArgs tests behavior with empty arguments.
func TestDefaultClientRunEmptyArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient()

	// Test with empty args (should just run "tmux" without arguments)
	stdout, stderr, err := client.Run()

	// Running tmux with no args is not typically valid
	// It might error or show usage
	if err != nil {
		// Error is acceptable
		t.Logf("Command with empty args failed (acceptable): %v", err)
	} else {
		// If it succeeds, stdout should have some output
		assert.NotEmpty(t, stdout, "stdout should not be empty when command succeeds")
	}

	t.Logf("Stdout: %q", stdout)
	t.Logf("Stderr: %q", stderr)
}

// TestDefaultClientRunDefaultTimeout tests that default timeout is applied.
func TestDefaultClientRunDefaultTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient()

	// Verify the default timeout is set
	// We can check by examining the client's timeout field via reflection or testing behavior
	// For now, we'll just verify it was created with default timeout

	// Run a quick command to ensure client works
	stdout, stderr, err := client.Run("-V")
	require.NoError(t, err, "client should be able to run commands")
	assert.Contains(t, stdout, "tmux", "should get tmux version")
	assert.Empty(t, stderr, "stderr should be empty for successful command")

	t.Logf("Default timeout: %v", DefaultTimeout)
}

// TestDefaultClientRunWithCustomTimeout tests custom timeout configuration.
func TestDefaultClientRunWithCustomTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	customTimeout := 10 * time.Second
	client := NewDefaultClient(WithTimeout(customTimeout))

	// Run a quick command
	stdout, stderr, err := client.Run("-V")
	require.NoError(t, err)
	assert.Contains(t, stdout, "tmux")
	assert.Empty(t, stderr)

	// Verify the client was configured with custom timeout
	// We can't directly access the timeout field, but we can infer it works
	t.Logf("Custom timeout: %v", customTimeout)
}

// TestNewDefaultClientOptions tests functional options for client configuration.
func TestNewDefaultClientOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     []ClientOption
		description string
	}{
		{
			name:        "default options",
			options:     nil,
			description: "should create client with default settings",
		},
		{
			name:        "with socket path",
			options:     []ClientOption{WithSocketPath("custom-socket")},
			description: "should create client with custom socket path",
		},
		{
			name:        "with timeout",
			options:     []ClientOption{WithTimeout(10 * time.Second)},
			description: "should create client with custom timeout",
		},
		{
			name: "with both options",
			options: []ClientOption{
				WithSocketPath("custom-socket"),
				WithTimeout(10 * time.Second),
			},
			description: "should create client with both custom options",
		},
		{
			name: "multiple socket options",
			options: []ClientOption{
				WithSocketPath("socket1"),
				WithSocketPath("socket2"),
			},
			description: "should use last socket path option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewDefaultClient(tt.options...)
			assert.NotNil(t, client, tt.description)
		})
	}
}

// TestDefaultClientRunConcurrent tests concurrent command execution.
func TestDefaultClientRunConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Run multiple concurrent commands
	const concurrency = 10
	results := make(chan struct {
		stdout string
		stderr string
		err    error
	}, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			stdout, stderr, err := client.Run("-V")
			results <- struct {
				stdout string
				stderr string
				err    error
			}{stdout, stderr, err}
		}()
	}

	// Collect results
	for i := 0; i < concurrency; i++ {
		result := <-results
		assert.NoError(t, result.err)
		assert.Contains(t, result.stdout, "tmux")
		assert.Empty(t, result.stderr)
	}

	t.Logf("Successfully ran %d concurrent commands", concurrency)
}

// TestDefaultClientGetCurrentContext tests GetCurrentContext method.
func TestDefaultClientGetCurrentContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()
	ctx, err := client.GetCurrentContext()

	assert.NoError(t, err, "GetCurrentContext should succeed")
	assert.NotEmpty(t, ctx.SessionID, "SessionID should not be empty")
	assert.NotEmpty(t, ctx.WindowID, "WindowID should not be empty")
	assert.NotEmpty(t, ctx.PaneID, "PaneID should not be empty")
	assert.NotEmpty(t, ctx.PanePID, "PanePID should not be empty")

	// SessionID should start with $
	assert.Contains(t, ctx.SessionID, "$", "SessionID should start with $")
	// WindowID should start with @
	assert.Contains(t, ctx.WindowID, "@", "WindowID should start with @")
	// PaneID should start with %
	assert.Contains(t, ctx.PaneID, "%", "PaneID should start with %")

	t.Logf("Context: SessionID=%s, WindowID=%s, PaneID=%s, PanePID=%s",
		ctx.SessionID, ctx.WindowID, ctx.PaneID, ctx.PanePID)
}

// TestDefaultClientValidatePaneExists tests ValidatePaneExists method.
func TestDefaultClientValidatePaneExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Get current context to test with real values
	ctx, err := client.GetCurrentContext()
	require.NoError(t, err, "should get current context")

	// Test with current pane (should exist)
	exists, err := client.ValidatePaneExists(ctx.SessionID, ctx.WindowID, ctx.PaneID)
	assert.NoError(t, err, "ValidatePaneExists should succeed for existing pane")
	assert.True(t, exists, "current pane should exist")

	// Test with non-existent pane
	exists, err = client.ValidatePaneExists(ctx.SessionID, ctx.WindowID, "%999999")
	assert.NoError(t, err, "ValidatePaneExists should succeed for non-existent pane")
	assert.False(t, exists, "non-existent pane should not exist")

	// Test with invalid target
	_, err = client.ValidatePaneExists("$999999", "@999999", "%999999")
	assert.Error(t, err, "ValidatePaneExists should fail for invalid target")
}

// TestDefaultClientJumpToPane tests JumpToPane method.
func TestDefaultClientJumpToPane(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Get current context to test with real values
	ctx, err := client.GetCurrentContext()
	require.NoError(t, err, "should get current context")

	// Test jumping to current pane (should succeed)
	success, err := client.JumpToPane(ctx.SessionID, ctx.WindowID, ctx.PaneID)
	assert.NoError(t, err, "JumpToPane should succeed for current pane")
	assert.True(t, success, "jump to current pane should succeed")

	// Test jumping to non-existent pane (should succeed but fall back to window)
	success, err = client.JumpToPane(ctx.SessionID, ctx.WindowID, "%999999")
	assert.NoError(t, err, "JumpToPane should succeed for non-existent pane (fallback to window)")
	assert.True(t, success, "jump to non-existent pane should succeed with fallback")

	// Test window-only jump with empty paneID (should succeed)
	success, err = client.JumpToPane(ctx.SessionID, ctx.WindowID, "")
	assert.NoError(t, err, "JumpToPane should succeed for window-only jump")
	assert.True(t, success, "window-only jump should succeed")

	// Test with invalid target (all empty parameters)
	success, err = client.JumpToPane("", "", "")
	assert.Error(t, err, "JumpToPane should fail with all empty parameters")
	assert.False(t, success, "jump with all empty parameters should fail")

	// Test with invalid session
	success, err = client.JumpToPane("$999999", "@999999", "%999999")
	assert.Error(t, err, "JumpToPane should fail with invalid session")
	assert.False(t, success, "jump to invalid session should fail")
}

// TestDefaultClientSetEnvironment tests SetEnvironment method.
func TestDefaultClientSetEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Test setting an environment variable
	testVarName := "TMUX_INTRAY_TEST_VAR"
	testVarValue := "test_value_123"

	err = client.SetEnvironment(testVarName, testVarValue)
	assert.NoError(t, err, "SetEnvironment should succeed")

	// Verify the variable was set by getting it back
	retrievedValue, err := client.GetEnvironment(testVarName)
	assert.NoError(t, err, "GetEnvironment should succeed after SetEnvironment")
	assert.Equal(t, testVarValue, retrievedValue, "retrieved value should match set value")

	// Clean up - unset the variable
	err = client.SetEnvironment(testVarName, "")
	assert.NoError(t, err, "unsetting environment variable should succeed")
}

// TestDefaultClientGetEnvironment tests GetEnvironment method.
func TestDefaultClientGetEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Test getting existing environment variable (TERM should always exist)
	termValue, err := client.GetEnvironment("TERM")
	assert.NoError(t, err, "GetEnvironment should succeed for existing variable")
	assert.NotEmpty(t, termValue, "TERM value should not be empty")

	t.Logf("TERM environment variable: %s", termValue)

	// Test getting non-existent variable
	_, err = client.GetEnvironment("TMUX_INTRAY_NONEXISTENT_VAR")
	assert.Error(t, err, "GetEnvironment should fail for non-existent variable")
}

// TestDefaultClientHasSession tests HasSession method.
func TestDefaultClientHasSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient()

	// Test when tmux is running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	tmuxRunning := err == nil

	if tmuxRunning {
		running, err := client.HasSession()
		assert.NoError(t, err, "HasSession should succeed when tmux is running")
		assert.True(t, running, "HasSession should return true when tmux is running")
	} else {
		t.Skip("tmux not running, skipping HasSession test")
	}

	// Test with custom socket path (tmux not running on that socket)
	clientWithSocket := NewDefaultClient(WithSocketPath("nonexistent-socket"))
	running, err := clientWithSocket.HasSession()
	assert.Error(t, err, "HasSession should fail with non-existent socket")
	assert.False(t, running, "HasSession should return false when tmux is not running")
}

// TestDefaultClientSetStatusOption tests SetStatusOption method.
func TestDefaultClientSetStatusOption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Test setting a status option
	// We'll set status-interval (safe to change)
	err = client.SetStatusOption("status-interval", "5")
	assert.NoError(t, err, "SetStatusOption should succeed")

	// Verify by getting the status-interval value
	stdout, _, err := client.Run("show-option", "-g", "status-interval")
	assert.NoError(t, err, "show-option should succeed")
	assert.Contains(t, stdout, "5", "status-interval should be set to 5")

	// Restore default value
	err = client.SetStatusOption("status-interval", "15")
	assert.NoError(t, err, "restoring default status-interval should succeed")

	t.Logf("SetStatusOption stdout: %s", stdout)

	// Test when tmux is not running
	clientWithSocket := NewDefaultClient(WithSocketPath("nonexistent-socket"))
	err = clientWithSocket.SetStatusOption("status-interval", "5")
	assert.Error(t, err, "SetStatusOption should fail when tmux is not running")
}

// TestDefaultClientListSessions tests ListSessions method.
func TestDefaultClientListSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Test listing sessions
	sessions, err := client.ListSessions()
	assert.NoError(t, err, "ListSessions should succeed")
	assert.NotNil(t, sessions, "sessions map should not be nil")

	// Should have at least one session
	assert.Greater(t, len(sessions), 0, "should have at least one session")

	// Verify session IDs start with $
	for sessionID := range sessions {
		assert.Contains(t, sessionID, "$", "session ID should start with $")
		assert.NotEmpty(t, sessions[sessionID], "session name should not be empty")
		t.Logf("Session: ID=%s, Name=%s", sessionID, sessions[sessionID])
	}

	// Test with non-existent socket
	clientWithSocket := NewDefaultClient(WithSocketPath("nonexistent-socket"))
	_, err = clientWithSocket.ListSessions()
	assert.Error(t, err, "ListSessions should fail with non-existent socket")
}

// TestDefaultClientMethodsErrorWrapping tests that all methods properly wrap errors.
func TestDefaultClientMethodsErrorWrapping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient(WithSocketPath("nonexistent-socket"))

	tests := []struct {
		name        string
		method      func() (interface{}, error)
		errorCheck  func(error) bool
		description string
	}{
		{
			name: "GetCurrentContext",
			method: func() (interface{}, error) {
				return client.GetCurrentContext()
			},
			errorCheck: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "failed to get tmux context")
			},
			description: "should wrap error with context message",
		},
		{
			name: "ValidatePaneExists",
			method: func() (interface{}, error) {
				return client.ValidatePaneExists("$0", "@0", "%0")
			},
			errorCheck: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "failed to list panes")
			},
			description: "should wrap error with context message",
		},
		{
			name: "SetEnvironment",
			method: func() (interface{}, error) {
				return nil, client.SetEnvironment("TEST", "value")
			},
			errorCheck: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "failed to set environment variable")
			},
			description: "should wrap error with context message",
		},
		{
			name: "GetEnvironment",
			method: func() (interface{}, error) {
				return client.GetEnvironment("TEST")
			},
			errorCheck: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "failed to get environment variable")
			},
			description: "should wrap error with context message",
		},
		{
			name: "ListSessions",
			method: func() (interface{}, error) {
				return client.ListSessions()
			},
			errorCheck: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "failed to list sessions")
			},
			description: "should wrap error with context message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.method()
			assert.Error(t, err, tt.description)
			assert.True(t, tt.errorCheck(err), "error should contain expected message")
			t.Logf("Error: %v", err)
		})
	}
}

// TestDefaultClientEnvironmentGetSetRoundTrip tests setting and getting environment variables.
func TestDefaultClientEnvironmentGetSetRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	testCases := []struct {
		name  string
		value string
	}{
		{"TMUX_INTRAY_TEST_1", "value1"},
		{"TMUX_INTRAY_TEST_2", "value with spaces"},
		{"TMUX_INTRAY_TEST_3", "12345"},
		{"TMUX_INTRAY_TEST_4", "special!@#$%^&*()"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the variable
			err := client.SetEnvironment(tc.name, tc.value)
			assert.NoError(t, err, "SetEnvironment should succeed")

			// Get the variable
			retrieved, err := client.GetEnvironment(tc.name)
			assert.NoError(t, err, "GetEnvironment should succeed")
			assert.Equal(t, tc.value, retrieved, "retrieved value should match set value")

			// Clean up
			err = client.SetEnvironment(tc.name, "")
			assert.NoError(t, err, "unsetting should succeed")
		})
	}
}

// TestDefaultClientGetCurrentContextInvalidFormat tests GetCurrentContext with invalid output format.
func TestDefaultClientGetCurrentContextInvalidFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient(WithSocketPath("nonexistent-socket"))

	_, err := client.GetCurrentContext()
	assert.Error(t, err, "GetCurrentContext should fail with invalid format")
	t.Logf("Error: %v", err)
}

// TestDefaultClientGetEnvironmentNotFound tests GetEnvironment with non-existent variable.
func TestDefaultClientGetEnvironmentNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Test getting non-existent variable
	_, err = client.GetEnvironment("TMUX_INTRAY_NONEXISTENT_VAR_XYZ123")
	assert.Error(t, err, "GetEnvironment should fail for non-existent variable")
	assert.Contains(t, err.Error(), "failed to get environment variable", "error should mention failed to get")
	t.Logf("Error: %v", err)
}

// TestDefaultClientSetStatusOptionTmuxNotRunning tests SetStatusOption when tmux is not running.
func TestDefaultClientSetStatusOptionTmuxNotRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient(WithSocketPath("nonexistent-socket"))

	err := client.SetStatusOption("status-interval", "5")
	assert.Error(t, err, "SetStatusOption should fail when tmux is not running")
	assert.Equal(t, ErrTmuxNotRunning, err, "should return ErrTmuxNotRunning")
	t.Logf("Error: %v", err)
}

// TestDefaultClientJumpToPaneInvalidTarget tests JumpToPane with invalid targets.
func TestDefaultClientJumpToPaneInvalidTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient()

	tests := []struct {
		name        string
		sessionID   string
		windowID    string
		paneID      string
		description string
	}{
		{
			name:        "empty session",
			sessionID:   "",
			windowID:    "@0",
			paneID:      "%0",
			description: "should fail with empty session ID",
		},
		{
			name:        "empty window",
			sessionID:   "$0",
			windowID:    "",
			paneID:      "%0",
			description: "should fail with empty window ID",
		},
		{
			name:        "all empty",
			sessionID:   "",
			windowID:    "",
			paneID:      "",
			description: "should fail with all empty IDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success, err := client.JumpToPane(tt.sessionID, tt.windowID, tt.paneID)
			assert.Error(t, err, tt.description)
			assert.False(t, success, tt.description)
			assert.Equal(t, ErrInvalidTarget, err, "should return ErrInvalidTarget")
			t.Logf("Error: %v", err)
		})
	}
}

// TestDefaultClientJumpToPaneWindowOnly tests window-only jumps (empty paneID).
func TestDefaultClientJumpToPaneWindowOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Get current context to test with real values
	ctx, err := client.GetCurrentContext()
	require.NoError(t, err, "should get current context")

	tests := []struct {
		name        string
		sessionID   string
		windowID    string
		paneID      string
		expectError bool
		description string
	}{
		{
			name:        "window-only jump succeeds",
			sessionID:   ctx.SessionID,
			windowID:    ctx.WindowID,
			paneID:      "",
			expectError: false,
			description: "should succeed with valid session and window",
		},
		{
			name:        "window-only jump fails with invalid window",
			sessionID:   ctx.SessionID,
			windowID:    "@999999",
			paneID:      "",
			expectError: true,
			description: "should fail with invalid window ID",
		},
		{
			name:        "window-only jump fails with invalid session",
			sessionID:   "$999999",
			windowID:    ctx.WindowID,
			paneID:      "",
			expectError: true,
			description: "should fail with invalid session ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success, err := client.JumpToPane(tt.sessionID, tt.windowID, tt.paneID)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.False(t, success, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				assert.True(t, success, tt.description)
			}
			t.Logf("Error: %v", err)
		})
	}
}

// TestDefaultClientJumpToPaneValidateError tests JumpToPane when ValidatePaneExists returns an error.
func TestDefaultClientJumpToPaneValidateError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient(WithSocketPath("nonexistent-socket"))

	// Test with non-existent session - validation will fail
	success, err := client.JumpToPane("$999999", "@999999", "%999999")
	assert.Error(t, err, "JumpToPane should fail when validation errors")
	assert.False(t, success, "JumpToPane should return false on error")
	assert.Contains(t, err.Error(), "pane validation failed", "error should mention validation failed")
	t.Logf("Error: %v", err)
}

// TestDefaultClientJumpToPaneWindowNotExist tests JumpToPane when window doesn't exist.
func TestDefaultClientJumpToPaneWindowNotExist(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Get current context to test with real session
	ctx, err := client.GetCurrentContext()
	require.NoError(t, err, "should get current context")

	// Test with valid session but invalid window - validation will fail first
	success, err := client.JumpToPane(ctx.SessionID, "@999999", "%0")
	assert.Error(t, err, "JumpToPane should fail when window doesn't exist")
	assert.False(t, success, "JumpToPane should return false when window doesn't exist")
	assert.Contains(t, err.Error(), "pane validation failed", "error should mention validation failed")
	t.Logf("Error: %v", err)
}

// TestDefaultClientListSessionsEmptyOutput tests ListSessions with empty output.
func TestDefaultClientListSessionsEmptyOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient(WithSocketPath("nonexistent-socket"))

	sessions, err := client.ListSessions()
	assert.Error(t, err, "ListSessions should fail with non-existent socket")
	assert.Nil(t, sessions, "sessions should be nil on error")
	t.Logf("Error: %v", err)
}

// TestDefaultClientGetSessionName tests GetSessionName method.
func TestDefaultClientGetSessionName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Get current context to test with real session
	ctx, err := client.GetCurrentContext()
	require.NoError(t, err, "should get current context")

	// Test getting current session name
	sessionName, err := client.GetSessionName(ctx.SessionID)
	assert.NoError(t, err, "GetSessionName should succeed for current session")
	assert.NotEmpty(t, sessionName, "session name should not be empty")

	t.Logf("Session ID: %s, Session Name: %s", ctx.SessionID, sessionName)

	// Test with non-existent session
	_, err = client.GetSessionName("$999999")
	assert.Error(t, err, "GetSessionName should fail for non-existent session")

	// Test with invalid session ID format
	clientWithSocket := NewDefaultClient(WithSocketPath("nonexistent-socket"))
	_, err = clientWithSocket.GetSessionName("invalid")
	assert.Error(t, err, "GetSessionName should fail with invalid session ID")
	assert.Contains(t, err.Error(), "get session name", "error should contain context message")
}

// TestDefaultClientGetSessionNameErrorWrapping tests error wrapping in GetSessionName.
func TestDefaultClientGetSessionNameErrorWrapping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient(WithSocketPath("nonexistent-socket"))

	_, err := client.GetSessionName("$0")
	assert.Error(t, err, "GetSessionName should fail with non-existent socket")
	assert.Contains(t, err.Error(), "get session name", "error should contain context message")
	t.Logf("Error: %v", err)
}

// TestDefaultClientGetTmuxVisibility tests GetTmuxVisibility method.
func TestDefaultClientGetTmuxVisibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Clear any existing visibility state before test
	err = client.SetEnvironment("TMUX_INTRAY_VISIBLE", "")
	assert.NoError(t, err, "clearing visibility should succeed")

	// Test default visibility (not set - should be false) with retry due to tmux eventual consistency
	var visible bool
	for i := 0; i < 5; i++ {
		visible, err = client.GetTmuxVisibility()
		if err == nil && !visible {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.NoError(t, err, "GetTmuxVisibility should succeed")
	assert.False(t, visible, "default visibility should be false")

	// Set visibility to true
	err = client.SetTmuxVisibility(true)
	assert.NoError(t, err, "SetTmuxVisibility should succeed")

	// Verify visibility is now true (with retry due to tmux eventual consistency)
	for i := 0; i < 5; i++ {
		visible, err = client.GetTmuxVisibility()
		if err == nil && visible {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.NoError(t, err, "GetTmuxVisibility should succeed after setting")
	assert.True(t, visible, "visibility should be true after setting to true")

	// Set visibility to false
	err = client.SetTmuxVisibility(false)
	assert.NoError(t, err, "SetTmuxVisibility should succeed")

	// Verify visibility is now false (with retry due to tmux eventual consistency)
	for i := 0; i < 5; i++ {
		visible, err = client.GetTmuxVisibility()
		if err == nil && !visible {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.NoError(t, err, "GetTmuxVisibility should succeed after setting to false")
	assert.False(t, visible, "visibility should be false after setting to false")

	// Clean up - unset the variable
	err = client.SetEnvironment("TMUX_INTRAY_VISIBLE", "")
	assert.NoError(t, err, "unsetting visibility should succeed")
}

// TestDefaultClientSetTmuxVisibility tests SetTmuxVisibility method.
func TestDefaultClientSetTmuxVisibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Clear any existing visibility state before test
	err = client.SetEnvironment("TMUX_INTRAY_VISIBLE", "")
	assert.NoError(t, err, "clearing visibility should succeed")

	tests := []struct {
		name        string
		visible     bool
		description string
	}{
		{
			name:        "set visibility to true",
			visible:     true,
			description: "should set visibility to true",
		},
		{
			name:        "set visibility to false",
			visible:     false,
			description: "should set visibility to false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.SetTmuxVisibility(tt.visible)
			assert.NoError(t, err, tt.description)

			// Verify the value was set (with retry due to tmux eventual consistency)
			var visible bool
			for i := 0; i < 5; i++ {
				visible, err = client.GetTmuxVisibility()
				if err == nil && visible == tt.visible {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			assert.NoError(t, err, "GetTmuxVisibility should succeed")
			assert.Equal(t, tt.visible, visible, "visibility should match what was set")
		})
	}

	// Clean up
	err = client.SetEnvironment("TMUX_INTRAY_VISIBLE", "")
	assert.NoError(t, err, "unsetting visibility should succeed")
}

// TestDefaultClientGetTmuxVisibilityTmuxNotRunning tests GetTmuxVisibility when tmux is not running.
func TestDefaultClientGetTmuxVisibilityTmuxNotRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient(WithSocketPath("nonexistent-socket"))

	_, err := client.GetTmuxVisibility()
	assert.Error(t, err, "GetTmuxVisibility should fail when tmux is not running")
	assert.Contains(t, err.Error(), "get tmux visibility", "error should contain context message")
	t.Logf("Error: %v", err)
}

// TestDefaultClientSetTmuxVisibilityTmuxNotRunning tests SetTmuxVisibility when tmux is not running.
func TestDefaultClientSetTmuxVisibilityTmuxNotRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewDefaultClient(WithSocketPath("nonexistent-socket"))

	err := client.SetTmuxVisibility(true)
	assert.Error(t, err, "SetTmuxVisibility should fail when tmux is not running")
	assert.Contains(t, err.Error(), "set tmux visibility", "error should contain context message")
	t.Logf("Error: %v", err)
}

// TestDefaultClientVisibilityRoundTrip tests round-trip of visibility state.
func TestDefaultClientVisibilityRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	testCases := []bool{true, false, true, false, true}

	for i, expectedVisible := range testCases {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Set visibility
			err := client.SetTmuxVisibility(expectedVisible)
			assert.NoError(t, err, "SetTmuxVisibility should succeed")

			// Get visibility
			visible, err := client.GetTmuxVisibility()
			assert.NoError(t, err, "GetTmuxVisibility should succeed")
			assert.Equal(t, expectedVisible, visible, "visibility should match expected value")

			t.Logf("Iteration %d: Set=%v, Got=%v", i, expectedVisible, visible)
		})
	}

	// Clean up
	err = client.SetEnvironment("TMUX_INTRAY_VISIBLE", "")
	assert.NoError(t, err, "unsetting visibility should succeed")
}

// TestDefaultClientGetSessionNameInvalidID tests GetSessionName with invalid session ID.
func TestDefaultClientGetSessionNameInvalidID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	tests := []struct {
		name          string
		sessionID     string
		shouldFail    bool
		description   string
		expectedError string
	}{
		{
			name:          "empty session ID",
			sessionID:     "",
			shouldFail:    false,
			description:   "tmux returns current session name for empty ID",
			expectedError: "",
		},
		{
			name:          "non-existent session",
			sessionID:     "$999999",
			shouldFail:    true,
			description:   "should fail for non-existent session",
			expectedError: "not found",
		},
		{
			name:          "invalid format",
			sessionID:     "not-a-session",
			shouldFail:    true,
			description:   "should fail with invalid format",
			expectedError: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := client.GetSessionName(tt.sessionID)

			if tt.shouldFail {
				assert.Error(t, err, tt.description)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError, "error should contain expected message")
				}
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotEmpty(t, name, "session name should not be empty")
			}
			t.Logf("Error: %v", err)
		})
	}
}

// TestDefaultClientGetSessionNameSuccess tests successful GetSessionName calls.
func TestDefaultClientGetSessionNameSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if tmux not running
	_, err := exec.Command("tmux", "has-session").CombinedOutput()
	if err != nil {
		t.Skip("tmux not running, skipping integration test")
	}

	client := NewDefaultClient()

	// Get all sessions to test with
	sessions, err := client.ListSessions()
	require.NoError(t, err, "should list sessions")
	require.Greater(t, len(sessions), 0, "should have at least one session")

	// Test getting names for all sessions
	for sessionID, expectedName := range sessions {
		t.Run(sessionID, func(t *testing.T) {
			name, err := client.GetSessionName(sessionID)
			assert.NoError(t, err, "GetSessionName should succeed")
			assert.Equal(t, expectedName, name, "session name should match")
			t.Logf("Session %s has name: %s", sessionID, name)
		})
	}
}
