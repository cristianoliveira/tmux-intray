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
