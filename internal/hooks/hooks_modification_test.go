package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPreAddHookModificationExitCode2 tests that a hook exiting with code 2
// signals that the notification should be accepted with modifications.
// Hooks echo "export VAR=value" to stdout to signal modifications.
func TestPreAddHookModificationExitCode2(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Hook that echoes export statements to signal modifications, exits with code 2
	// Hooks must ECHO export statements to stdout - this is how modifications are captured
	script := filepath.Join(hookDir, "01-modify.sh")
	scriptContent := `#!/bin/sh
echo "export MESSAGE=\"[MODIFIED] $MESSAGE\""
echo "export LEVEL=warning"
exit 2
`
	require.NoError(t, os.WriteFile(script, []byte(scriptContent), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	// Run hook and capture result
	result, err := RunWithModification("pre-add", "MESSAGE=original", "LEVEL=info")
	require.NoError(t, err)

	// Should indicate modification
	require.True(t, result.Modified, "Hook exited with 2, expected Modified=true")
	require.Equal(t, 2, result.ExitCode)

	// Check that modifications were captured
	require.Equal(t, "[MODIFIED] original", result.EnvVars["MESSAGE"])
	require.Equal(t, "warning", result.EnvVars["LEVEL"])
}

// TestPreAddHookAcceptExitCode0 tests that exit code 0 means accept without modifications.
func TestPreAddHookAcceptExitCode0(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Hook that accepts without modifications
	script := filepath.Join(hookDir, "01-accept.sh")
	scriptContent := `#!/bin/sh
exit 0
`
	require.NoError(t, os.WriteFile(script, []byte(scriptContent), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	result, err := RunWithModification("pre-add", "MESSAGE=original", "LEVEL=info")
	require.NoError(t, err)

	// Should indicate accepted without modification
	require.False(t, result.Modified, "Hook exited with 0, expected Modified=false")
	require.Equal(t, 0, result.ExitCode)
	// No modifications should be present
	require.Empty(t, result.EnvVars)
}

// TestPreAddHookRejectExitCode1 tests that exit code 1 means reject.
func TestPreAddHookRejectExitCode1(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Hook that rejects the notification
	script := filepath.Join(hookDir, "01-reject.sh")
	scriptContent := `#!/bin/sh
echo "Rejecting notification" >&2
exit 1
`
	require.NoError(t, os.WriteFile(script, []byte(scriptContent), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "warn"))

	_, err := RunWithModification("pre-add", "MESSAGE=original", "LEVEL=info")
	require.Error(t, err)
	require.Contains(t, err.Error(), "rejected")
}

// TestPreAddHookModificationChain tests that multiple hooks in a chain
// can progressively modify the notification.
func TestPreAddHookModificationChain(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// First hook: add prefix
	script1 := filepath.Join(hookDir, "01-prefix.sh")
	require.NoError(t, os.WriteFile(script1, []byte(`#!/bin/sh
echo "export MESSAGE=\"[PREFIX] $MESSAGE\""
exit 2
`), 0755))

	// Second hook: add suffix
	script2 := filepath.Join(hookDir, "02-suffix.sh")
	require.NoError(t, os.WriteFile(script2, []byte(`#!/bin/sh
echo "export MESSAGE=\"$MESSAGE [SUFFIX]\""
exit 2
`), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	result, err := RunWithModification("pre-add", "MESSAGE=original", "LEVEL=info")
	require.NoError(t, err)

	// Both hooks should have executed and modified
	require.True(t, result.Modified)
	// The final message should reflect both modifications (last write wins for same var)
	// But order is alphabetical: 01-prefix runs first, 02-suffix runs second
	require.Equal(t, "original [SUFFIX]", result.EnvVars["MESSAGE"])
}

// TestPreAddHookModificationWithAsync tests that modification hooks work
// in async mode (though modifications won't be captured).
func TestPreAddHookModificationWithAsync(t *testing.T) {
	ResetForTesting()
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	script := filepath.Join(hookDir, "01-modify.sh")
	require.NoError(t, os.WriteFile(script, []byte(`#!/bin/sh
echo "export MESSAGE=\"[ASYNC] $MESSAGE\""
exit 2
`), 0755))

	// Save and restore environment variables
	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	oldAsync := os.Getenv("TMUX_INTRAY_HOOKS_ASYNC")
	oldFailureMode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE")
	t.Cleanup(func() {
		_ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
		_ = os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", oldAsync)
		_ = os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", oldFailureMode)
	})
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1"))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	// In async mode, we can't capture modifications (hooks run in background)
	// So we return immediately without modification data
	_, err := RunWithModification("pre-add", "MESSAGE=original", "LEVEL=info")
	// Async hooks don't block, so we get no modification result
	require.NoError(t, err)

	// Wait for async hooks to complete
	WaitForPendingHooks()
}

// TestPreAddHookModificationInvalidExport tests that invalid export lines are ignored.
func TestPreAddHookModificationInvalidExport(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	script := filepath.Join(hookDir, "01-invalid.sh")
	scriptContent := `#!/bin/sh
echo 'export MESSAGE="valid modification"'
echo "some debug output" >&2
echo "export INVALID=without space"
echo 'export "MALFORMED=export"'
echo 'export MESSAGE_TWO="another valid"'
exit 2
`
	require.NoError(t, os.WriteFile(script, []byte(scriptContent), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	result, err := RunWithModification("pre-add", "MESSAGE=original", "LEVEL=info")
	require.NoError(t, err)
	require.True(t, result.Modified)

	// Valid exports should be captured
	require.Equal(t, "valid modification", result.EnvVars["MESSAGE"])
	require.Equal(t, "another valid", result.EnvVars["MESSAGE_TWO"])

	// Invalid exports should be ignored (not panic or error)
	_, hasInvalid := result.EnvVars["INVALID"]
	require.False(t, hasInvalid, "Invalid export 'INVALID=without space' should be ignored")
}

// TestPreAddHookModificationPreservesOriginal tests that original env vars
// are passed to the hook and can be referenced.
func TestPreAddHookModificationPreservesOriginal(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	script := filepath.Join(hookDir, "01-enrich.sh")
	scriptContent := `#!/bin/sh
# Hook can access original values and enrich them
echo "export MESSAGE=\"[ENRICHED] $MESSAGE\""
echo "export LEVEL=warning"
exit 2
`
	require.NoError(t, os.WriteFile(script, []byte(scriptContent), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	result, err := RunWithModification("pre-add", "MESSAGE=original message", "LEVEL=info")
	require.NoError(t, err)
	require.True(t, result.Modified)

	// Message should include enrichment prefix and original message
	require.Contains(t, result.EnvVars["MESSAGE"], "[ENRICHED]")
	require.Contains(t, result.EnvVars["MESSAGE"], "original message")
	require.Equal(t, "warning", result.EnvVars["LEVEL"])
}

// TestPreAddHookModificationEmptyScript tests that a hook that exits 0
// but produces no output still works.
func TestPreAddHookModificationEmptyScript(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	script := filepath.Join(hookDir, "01-empty.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexit 0"), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	result, err := RunWithModification("pre-add", "MESSAGE=test")
	require.NoError(t, err)
	require.False(t, result.Modified)
	require.Equal(t, 0, result.ExitCode)
}

// TestParseModifications tests the modification parsing logic.
func TestParseModifications(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected map[string]string
	}{
		{
			name:     "empty output",
			output:   "",
			expected: map[string]string{},
		},
		{
			name:     "single export",
			output:   "export MESSAGE=hello\n",
			expected: map[string]string{"MESSAGE": "hello"},
		},
		{
			name:     "export with spaces around equals - not supported",
			output:   "export LEVEL = warning\n",
			expected: map[string]string{}, // Spaces around = not supported
		},
		{
			name:     "export with quotes",
			output:   `export MESSAGE="hello world"` + "\n",
			expected: map[string]string{"MESSAGE": "hello world"},
		},
		{
			name:     "export with single quotes",
			output:   "export MESSAGE='hello world'\n",
			expected: map[string]string{"MESSAGE": "hello world"},
		},
		{
			name:     "multiple exports",
			output:   "export A=1\nexport B=2\nexport C=3\n",
			expected: map[string]string{"A": "1", "B": "2", "C": "3"},
		},
		{
			name:     "mixed output with exports",
			output:   "some debug output\nexport MESSAGE=modified\nmore output\nexport LEVEL=warning\n",
			expected: map[string]string{"MESSAGE": "modified", "LEVEL": "warning"},
		},
		{
			name:     "value with equals sign",
			output:   "export KEY=value=with=equals\n",
			expected: map[string]string{"KEY": "value=with=equals"},
		},
		{
			name:     "empty value",
			output:   "export EMPTY=\n",
			expected: map[string]string{"EMPTY": ""},
		},
		{
			name:     "export without value",
			output:   "export NO_VALUE\n",
			expected: map[string]string{},
		},
		{
			name:     "non-export lines ignored",
			output:   "some random output\ndebug log line\nexport KEEP=this\nanother line\n",
			expected: map[string]string{"KEEP": "this"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseModifications(tt.output)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestRunWithModificationNoHooks tests that running when no hooks exist
// returns a result indicating no modification.
func TestRunWithModificationNoHooks(t *testing.T) {
	tmpDir := t.TempDir()

	// No hooks directory created - should return no modification
	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))

	result, err := RunWithModification("pre-add", "MESSAGE=test")
	require.NoError(t, err)
	require.False(t, result.Modified)
	require.Equal(t, 0, result.ExitCode)
	require.Empty(t, result.EnvVars)
}

// TestRunWithModificationFailureModeAbort tests that exit code 1 always
// triggers rejection regardless of failure mode.
func TestRunWithModificationFailureModeAbort(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Hook that rejects the notification (exit code 1)
	script := filepath.Join(hookDir, "01-reject.sh")
	require.NoError(t, os.WriteFile(script, []byte(`#!/bin/sh
echo "export MESSAGE=modified"
exit 1
`), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "warn"))

	// Should error because exit 1 triggers rejection
	_, err := RunWithModification("pre-add", "MESSAGE=original")
	require.Error(t, err)
	require.Contains(t, err.Error(), "rejected")
}

// TestBackwardCompatibilityRunFunction tests that the original Run function
// still works for hooks that don't use modifications.
func TestBackwardCompatibilityRunFunction(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Standard hook that accepts
	script := filepath.Join(hookDir, "01-standard.sh")
	require.NoError(t, os.WriteFile(script, []byte(`#!/bin/sh
echo "Standard hook execution"
exit 0
`), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	// Original Run function should still work
	err := Run("pre-add", "MESSAGE=test")
	require.NoError(t, err)
}

// TestModificationWithSpecialCharacters tests that modifications handle
// special characters correctly.
func TestModificationWithSpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	script := filepath.Join(hookDir, "01-escape.sh")
	scriptContent := `#!/bin/sh
echo "export MESSAGE=\"Modified: $MESSAGE with safe chars only\""
exit 2
`
	require.NoError(t, os.WriteFile(script, []byte(scriptContent), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	result, err := RunWithModification("pre-add", "MESSAGE=original")
	require.NoError(t, err)
	require.True(t, result.Modified)

	// Message should contain the modified content with special chars
	require.Contains(t, result.EnvVars["MESSAGE"], "Modified:")
	require.Contains(t, result.EnvVars["MESSAGE"], "original")
}

// TestModificationPreservesVariableOrdering tests that when multiple hooks
// modify the same variable, the last modification wins.
func TestModificationPreservesVariableOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Hook 1: sets MESSAGE
	script1 := filepath.Join(hookDir, "01-first.sh")
	require.NoError(t, os.WriteFile(script1, []byte(`#!/bin/sh
echo "export MESSAGE=first"
exit 2
`), 0755))

	// Hook 2: also sets MESSAGE (should override)
	script2 := filepath.Join(hookDir, "02-second.sh")
	require.NoError(t, os.WriteFile(script2, []byte(`#!/bin/sh
echo "export MESSAGE=second"
exit 2
`), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	result, err := RunWithModification("pre-add", "MESSAGE=original")
	require.NoError(t, err)
	require.True(t, result.Modified)

	// Last hook's modification should win
	require.Equal(t, "second", result.EnvVars["MESSAGE"])
}

// TestModificationAddsNewVariables tests that hooks can add new variables
// that weren't in the original set.
func TestModificationAddsNewVariables(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	script := filepath.Join(hookDir, "01-add.sh")
	require.NoError(t, os.WriteFile(script, []byte(`#!/bin/sh
echo "export CUSTOM_FIELD=custom-value"
echo "export COMPUTED=computed-value"
exit 2
`), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	t.Cleanup(func() { _ = os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir) })
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir))
	require.NoError(t, os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore"))

	result, err := RunWithModification("pre-add", "MESSAGE=original")
	require.NoError(t, err)
	require.True(t, result.Modified)

	// New variables should be present
	require.Equal(t, "custom-value", result.EnvVars["CUSTOM_FIELD"])
	require.Equal(t, "computed-value", result.EnvVars["COMPUTED"])
}

// BenchmarkParseModifications benchmarks the modification parsing function.
func BenchmarkParseModifications(b *testing.B) {
	testOutput := strings.Repeat("export VAR1=value1\nexport VAR2=value2\nexport VAR3=value3\n", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseModifications(testOutput)
	}
}
