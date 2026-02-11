//go:build integration
// +build integration

package hooks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHookExecutionCompatibility verifies that Go hook system can execute
// Bash hook scripts with the same environment variables as Bash implementation.
func TestHookExecutionCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	hookOutputFile := filepath.Join(tmpDir, "hook-output.txt")

	// Create pre-add hook that writes all env vars to output file
	preAddDir := filepath.Join(hooksDir, "pre-add")
	require.NoError(t, os.MkdirAll(preAddDir, 0755))

	hookScriptContent := `#!/bin/bash
# Capture all environment variables that hooks should receive
env | sort > "$HOOK_OUTPUT_FILE"
`
	hookScript := filepath.Join(preAddDir, "01-capture-env.sh")
	require.NoError(t, os.WriteFile(hookScript, []byte(hookScriptContent), 0755))

	// Set up environment for Go hooks
	oldHookDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldHookDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", hooksDir)

	oldAsync := os.Getenv("TMUX_INTRAY_HOOKS_ASYNC")
	defer os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", oldAsync)
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "0")

	oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
	defer os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	// Run hook via Go (pass HOOK_OUTPUT_FILE via env var to the hook)
	require.NoError(t, Run("pre-add",
		"NOTIFICATION_ID=1",
		"LEVEL=info",
		"MESSAGE=test message",
		"TIMESTAMP=2025-01-01T00:00:00Z",
		"SESSION=sess1",
		"WINDOW=win1",
		"PANE=pane1",
		"PANE_CREATED=123456",
		fmt.Sprintf("HOOK_OUTPUT_FILE=%s", hookOutputFile),
	))

	// Read captured env vars from Go hook execution
	goOutput, err := os.ReadFile(hookOutputFile)
	require.NoError(t, err)

	// Clear output file for Bash test
	require.NoError(t, os.WriteFile(hookOutputFile, []byte{}, 0644))

	// Now run via Bash directly for comparison
	bashScriptContent := fmt.Sprintf(`
export HOOK_POINT="pre-add"
export NOTIFICATION_ID="1"
export LEVEL="info"
export MESSAGE="test message"
export TIMESTAMP="2025-01-01T00:00:00Z"
export SESSION="sess1"
export WINDOW="win1"
export PANE="pane1"
export PANE_CREATED="123456"
export HOOK_OUTPUT_FILE="%s"

# Execute the hook script
bash "%s"
`, hookOutputFile, hookScript)

	bashScriptPath := filepath.Join(tmpDir, "run-hook.sh")
	require.NoError(t, os.WriteFile(bashScriptPath, []byte(bashScriptContent), 0755))

	cmd := exec.Command("bash", bashScriptPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Bash hook execution failed: %s", output)

	// Read captured env vars from Bash hook execution
	bashOutput, err := os.ReadFile(hookOutputFile)
	require.NoError(t, err)

	// Both should have captured similar environment variables
	goEnvVars := strings.TrimSpace(string(goOutput))
	bashEnvVars := strings.TrimSpace(string(bashOutput))

	// Check that key environment variables are present in both
	// Note: grep -E pattern matches keys with their values
	for _, key := range []string{
		"HOOK_POINT=pre-add",
		"NOTIFICATION_ID=1",
		"LEVEL=info",
		"MESSAGE=test message",
		"TIMESTAMP=2025-01-01T00:00:00Z",
		"SESSION=sess1",
		"WINDOW=win1",
		"PANE=pane1",
		"PANE_CREATED=123456",
	} {
		require.Contains(t, goEnvVars, key, "Go hook missing env var: %s\nGo env vars:\n%s", key, goEnvVars)
		require.Contains(t, bashEnvVars, key, "Bash hook missing env var: %s\nBash env vars:\n%s", key, bashEnvVars)
	}

	// Note: Go and Bash environments may have different numbers of env vars due to
	// test environment differences, so we don't check exact count
}

// TestHookEnvironmentVariablePassing verifies that all expected environment
// variables are passed from Go hooks to Bash hook scripts.
func TestHookEnvironmentVariablePassing(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	hookLogFile := filepath.Join(tmpDir, "hook.log")

	preAddDir := filepath.Join(hooksDir, "pre-add")
	require.NoError(t, os.MkdirAll(preAddDir, 0755))

	// Hook that writes all env vars to a log file
	hookScriptContent := `#!/bin/bash
echo "=== Hook Environment ===" >> "$HOOK_LOG"
echo "HOOK_POINT=$HOOK_POINT" >> "$HOOK_LOG"
echo "HOOK_TIMESTAMP=$HOOK_TIMESTAMP" >> "$HOOK_LOG"
echo "TMUX_INTRAY_HOOKS_FAILURE_MODE=$TMUX_INTRAY_HOOKS_FAILURE_MODE" >> "$HOOK_LOG"
echo "NOTIFICATION_ID=$NOTIFICATION_ID" >> "$HOOK_LOG"
echo "LEVEL=$LEVEL" >> "$HOOK_LOG"
echo "MESSAGE=$MESSAGE" >> "$HOOK_LOG"
echo "TIMESTAMP=$TIMESTAMP" >> "$HOOK_LOG"
echo "SESSION=$SESSION" >> "$HOOK_LOG"
echo "WINDOW=$WINDOW" >> "$HOOK_LOG"
echo "PANE=$PANE" >> "$HOOK_LOG"
echo "PANE_CREATED=$PANE_CREATED" >> "$HOOK_LOG"
`
	hookScript := filepath.Join(preAddDir, "01-log-env.sh")
	require.NoError(t, os.WriteFile(hookScript, []byte(hookScriptContent), 0755))

	oldHookDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldHookDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", hooksDir)

	oldAsync := os.Getenv("TMUX_INTRAY_HOOKS_ASYNC")
	defer os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", oldAsync)
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "0")

	oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
	defer os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	// Run hook via Go with custom env vars
	require.NoError(t, Run("pre-add",
		"NOTIFICATION_ID=42",
		"LEVEL=warning",
		"MESSAGE=test message with special chars: !@#$%",
		"TIMESTAMP=2025-02-11T12:34:56Z",
		"SESSION=my-session",
		"WINDOW=my-window",
		"PANE=my-pane",
		"PANE_CREATED=999888",
		fmt.Sprintf("HOOK_LOG=%s", hookLogFile),
	))

	// Verify log file was created and contains expected values
	logContent, err := os.ReadFile(hookLogFile)
	require.NoError(t, err)

	logStr := string(logContent)
	require.Contains(t, logStr, "HOOK_POINT=pre-add")
	require.Contains(t, logStr, "NOTIFICATION_ID=42")
	require.Contains(t, logStr, "LEVEL=warning")
	require.Contains(t, logStr, "MESSAGE=test message with special chars: !@#$%")
	require.Contains(t, logStr, "TIMESTAMP=2025-02-11T12:34:56Z")
	require.Contains(t, logStr, "SESSION=my-session")
	require.Contains(t, logStr, "WINDOW=my-window")
	require.Contains(t, logStr, "PANE=my-pane")
	require.Contains(t, logStr, "PANE_CREATED=999888")
}

// TestHookOutputHandlingConsistency verifies that hook output handling
// is consistent between Go and Bash implementations.
func TestHookOutputHandlingConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	outputFile := filepath.Join(tmpDir, "output.txt")

	preAddDir := filepath.Join(hooksDir, "pre-add")
	require.NoError(t, os.MkdirAll(preAddDir, 0755))

	// Hook that writes to stdout and stderr
	hookScriptContent := `#!/bin/bash
echo "stdout message"
echo "stderr message" >&2
echo "another stdout"
echo "another stderr" >&2
echo "result=$?" > "%s"
`
	hookScript := filepath.Join(preAddDir, "01-output.sh")
	require.NoError(t, os.WriteFile(hookScript, []byte(fmt.Sprintf(hookScriptContent, outputFile)), 0755))

	oldHookDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldHookDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", hooksDir)

	oldAsync := os.Getenv("TMUX_INTRAY_HOOKS_ASYNC")
	defer os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", oldAsync)
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "0")

	oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
	defer os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "warn")

	// Capture stderr during hook execution
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Run hook via Go
	err := Run("pre-add", "NOTIFICATION_ID=1")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderrOutput := buf.String()

	// Hook should succeed
	require.NoError(t, err)

	// Verify hook script was executed (result file should exist with exit code 0)
	resultContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.Contains(t, string(resultContent), "result=0")

	// Stderr should contain hook output (Go redirects to stderr as Bash does)
	// Note: Go hooks write both stdout and stderr to stderr
	hasOutput := strings.Contains(stderrOutput, "stdout message") || strings.Contains(stderrOutput, "stderr message")
	require.True(t, hasOutput, "stderrOutput should contain hook output")
}

// TestMultipleHooksExecutionOrder verifies that multiple hooks are executed
// in the correct alphabetical order (as Bash does).
func TestMultipleHooksExecutionOrder(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	orderFile := filepath.Join(tmpDir, "execution-order.txt")

	preAddDir := filepath.Join(hooksDir, "pre-add")
	require.NoError(t, os.MkdirAll(preAddDir, 0755))

	// Create multiple hooks that write their names in order
	hooks := []struct {
		name     string
		filename string
		content  string
	}{
		{"03-third", "03-third.sh", "echo 'third' >> \"$ORDER_FILE\""},
		{"01-first", "01-first.sh", "echo 'first' >> \"$ORDER_FILE\""},
		{"02-second", "02-second.sh", "echo 'second' >> \"$ORDER_FILE\""},
	}

	for _, h := range hooks {
		scriptContent := fmt.Sprintf(`#!/bin/bash
%s
`, h.content)
		scriptPath := filepath.Join(preAddDir, h.filename)
		require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0755))
	}

	oldHookDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldHookDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", hooksDir)

	oldAsync := os.Getenv("TMUX_INTRAY_HOOKS_ASYNC")
	defer os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", oldAsync)
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "0")

	oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
	defer os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	// Run hook via Go
	require.NoError(t, Run("pre-add", fmt.Sprintf("ORDER_FILE=%s", orderFile)))

	// Verify execution order
	orderContent, err := os.ReadFile(orderFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(orderContent)), "\n")
	require.Len(t, lines, 3)
	require.Equal(t, "first", lines[0])
	require.Equal(t, "second", lines[1])
	require.Equal(t, "third", lines[2])
}

// TestHookFailureModesCompatibility verifies that hook failure modes work
// consistently between Go and Bash implementations.
func TestHookFailureModesCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")

	preAddDir := filepath.Join(hooksDir, "pre-add")
	require.NoError(t, os.MkdirAll(preAddDir, 0755))

	// Hook that always fails
	hookScriptContent := `#!/bin/bash
exit 1
`
	hookScript := filepath.Join(preAddDir, "01-fail.sh")
	require.NoError(t, os.WriteFile(hookScript, []byte(hookScriptContent), 0755))

	oldHookDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldHookDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", hooksDir)

	oldAsync := os.Getenv("TMUX_INTRAY_HOOKS_ASYNC")
	defer os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", oldAsync)
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "0")

	t.Run("abort_mode_returns_error", func(t *testing.T) {
		oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
		defer os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
		os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "abort")

		err := Run("pre-add", "NOTIFICATION_ID=1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "hook '01-fail.sh' failed")
	})

	t.Run("warn_mode_logs_warning", func(t *testing.T) {
		oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
		defer os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
		os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "warn")

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		err := Run("pre-add", "NOTIFICATION_ID=1")

		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		buf.ReadFrom(r)
		stderrOutput := buf.String()

		// Should not error
		require.NoError(t, err)
		// Should log warning
		require.Contains(t, stderrOutput, "warning")
	})

	t.Run("ignore_mode_silent", func(t *testing.T) {
		oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
		defer os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
		os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		err := Run("pre-add", "NOTIFICATION_ID=1")

		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		buf.ReadFrom(r)
		stderrOutput := buf.String()

		// Should not error
		require.NoError(t, err)
		// Should not show hook failure warning (in ignore mode)
		// Note: There may still be other warnings from config loading
		require.NotContains(t, stderrOutput, "hook '01-fail.sh' failed")
	})
}

// TestPostAddHookWithNotificationID verifies that post-add hook receives
// the actual notification ID after it's been assigned.
func TestPostAddHookWithNotificationID(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")
	idFile := filepath.Join(tmpDir, "notification-id.txt")

	postAddDir := filepath.Join(hooksDir, "post-add")
	require.NoError(t, os.MkdirAll(postAddDir, 0755))

	hookScriptContent := `#!/bin/bash
echo "NOTIFICATION_ID=$NOTIFICATION_ID" > "$ID_FILE"
`
	hookScript := filepath.Join(postAddDir, "01-capture-id.sh")
	require.NoError(t, os.WriteFile(hookScript, []byte(hookScriptContent), 0755))

	oldHookDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldHookDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", hooksDir)

	oldAsync := os.Getenv("TMUX_INTRAY_HOOKS_ASYNC")
	defer os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", oldAsync)
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "0")

	oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
	defer os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	// Run post-add hook with a specific notification ID
	require.NoError(t, Run("post-add", "NOTIFICATION_ID=123", fmt.Sprintf("ID_FILE=%s", idFile)))

	// Verify the hook received the notification ID
	idContent, err := os.ReadFile(idFile)
	require.NoError(t, err)
	require.Contains(t, string(idContent), "NOTIFICATION_ID=123")
}
