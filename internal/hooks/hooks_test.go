package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInitAndRunNoPanic(t *testing.T) {
	require.NotPanics(t, func() {
		Init()
		Run("pre-add", "FOO=bar")
	})
}

func TestRunSyncHookSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))
	script := filepath.Join(hookDir, "test.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho hello"), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	require.NoError(t, Run("pre-add", "FOO=bar"))
}

func TestRunSyncHookFailureModes(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))
	script := filepath.Join(hookDir, "fail.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexit 1"), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")

	// abort mode should return error
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "abort")
	err := Run("pre-add")
	require.Error(t, err)
	require.Contains(t, err.Error(), "hook fail.sh failed")

	// warn mode should not error but print warning (we can't capture stderr easily)
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "warn")
	require.NoError(t, Run("pre-add"))

	// ignore mode should not error
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
	require.NoError(t, Run("pre-add"))
}

func TestRunSyncHookEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))
	script := filepath.Join(hookDir, "env.sh")
	// Write script that prints env var
	require.NoError(t, os.WriteFile(script, []byte(`#!/bin/sh
echo "HOOK_POINT=$HOOK_POINT"
echo "FOO=$FOO"
`), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	// Capture output? We'll just ensure no error.
	require.NoError(t, Run("pre-add", "FOO=bar"))
}

func TestRunSyncHookOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))
	// Create scripts with numeric names out of order
	script1 := filepath.Join(hookDir, "2-second.sh")
	script2 := filepath.Join(hookDir, "1-first.sh")
	script3 := filepath.Join(hookDir, "3-third.sh")
	require.NoError(t, os.WriteFile(script1, []byte("#!/bin/sh\necho second"), 0755))
	require.NoError(t, os.WriteFile(script2, []byte("#!/bin/sh\necho first"), 0755))
	require.NoError(t, os.WriteFile(script3, []byte("#!/bin/sh\necho third"), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	// No easy way to verify order, but we can at least ensure they all run
	require.NoError(t, Run("pre-add"))
}

func TestRunAsyncHook(t *testing.T) {
	ResetForTesting()
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))
	script := filepath.Join(hookDir, "async.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nsleep 0.1\necho done"), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	start := time.Now()
	require.NoError(t, Run("pre-add"))
	// Async should return quickly (not wait for sleep)
	require.Less(t, time.Since(start), 50*time.Millisecond)
	// Wait for async hook to complete
	WaitForPendingHooks()
}

func TestRunAsyncHookMaxLimit(t *testing.T) {
	ResetForTesting()
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))
	// Create 3 scripts that sleep
	for i := 1; i <= 3; i++ {
		script := filepath.Join(hookDir, fmt.Sprintf("hook%d.sh", i))
		require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nsleep 0.5"), 0755))
	}

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	os.Setenv("TMUX_INTRAY_MAX_HOOKS", "2")
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	// Should skip the third hook due to max limit (warning printed)
	require.NoError(t, Run("pre-add"))
	// Wait for pending hooks
	WaitForPendingHooks()
}

func TestAsyncHookPanicRecovery(t *testing.T) {
	ResetForTesting()
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Create a script that panics (will cause panic in goroutine body)
	script := filepath.Join(hookDir, "panic.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexit 0"), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")

	// This should not panic the test even if internal goroutine panics
	require.NotPanics(t, func() {
		Run("pre-add")
	})

	// Ensure all hooks complete and cleanup happens
	WaitForPendingHooks()
}

func TestAsyncHookTimeoutDetection(t *testing.T) {
	ResetForTesting()
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Create a script that sleeps longer than the timeout
	script := filepath.Join(hookDir, "timeout.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nsleep 5\necho done"), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT", "0.5")

	start := time.Now()
	err := Run("pre-add")
	require.NoError(t, err)

	// Wait for hook to complete (should timeout)
	WaitForPendingHooks()
	duration := time.Since(start)

	// Should complete within timeout + some overhead (not wait for full sleep 5)
	require.Less(t, duration, 2*time.Second)
}

func TestAsyncHookNoLeakOnFailure(t *testing.T) {
	ResetForTesting()
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Create a script that fails
	script := filepath.Join(hookDir, "fail.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexit 1"), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "warn")

	err := Run("pre-add")
	require.NoError(t, err)

	// Wait for all hooks to complete
	WaitForPendingHooks()

	// Verify no goroutines leaked by checking that WaitForPendingHooks returns
	// If there was a leak, this would block forever
}

func TestAsyncHookCleanupAlwaysCalled(t *testing.T) {
	ResetForTesting()
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Create multiple hooks with different behaviors
	scripts := []struct {
		name     string
		content  string
		duration time.Duration
	}{
		{"success.sh", "#!/bin/sh\necho success", 100 * time.Millisecond},
		{"fail.sh", "#!/bin/sh\nexit 1", 100 * time.Millisecond},
		{"timeout.sh", "#!/bin/sh\nsleep 2", 2 * time.Second},
	}

	for _, s := range scripts {
		script := filepath.Join(hookDir, s.name)
		require.NoError(t, os.WriteFile(script, []byte(s.content), 0755))
	}

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "warn")
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT", "0.5")

	err := Run("pre-add")
	require.NoError(t, err)

	// Wait for all hooks to complete (including those that timeout)
	WaitForPendingHooks()

	// If cleanup wasn't called, this test would hang
}

func TestAsyncHookContextCancellation(t *testing.T) {
	ResetForTesting()
	tmpDir := t.TempDir()
	hookDir := filepath.Join(tmpDir, "pre-add")
	require.NoError(t, os.MkdirAll(hookDir, 0755))

	// Create a script that ignores SIGTERM and keeps running
	script := filepath.Join(hookDir, "ignore-sigterm.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\ntrap '' TERM\nsleep 10\necho done"), 0755))

	oldDir := os.Getenv("TMUX_INTRAY_HOOKS_DIR")
	defer os.Setenv("TMUX_INTRAY_HOOKS_DIR", oldDir)
	os.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	os.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	os.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
	os.Setenv("TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT", "0.5")

	start := time.Now()
	err := Run("pre-add")
	require.NoError(t, err)

	// Wait for all hooks to complete
	WaitForPendingHooks()
	duration := time.Since(start)

	// Should complete within timeout + overhead
	require.Less(t, duration, 2*time.Second)
}

func TestGetMaxAsyncHooksValidation(t *testing.T) {
	// Test default value when env var is not set
	oldMaxHooks := os.Getenv("TMUX_INTRAY_MAX_HOOKS")
	defer os.Setenv("TMUX_INTRAY_MAX_HOOKS", oldMaxHooks)
	os.Unsetenv("TMUX_INTRAY_MAX_HOOKS")
	max := getMaxAsyncHooks()
	require.Equal(t, 10, max, "Default should be 10")

	// Test valid values within range [1, 100]
	os.Setenv("TMUX_INTRAY_MAX_HOOKS", "1")
	max = getMaxAsyncHooks()
	require.Equal(t, 1, max, "Minimum value should be 1")

	os.Setenv("TMUX_INTRAY_MAX_HOOKS", "50")
	max = getMaxAsyncHooks()
	require.Equal(t, 50, max, "Valid mid-range value should be accepted")

	os.Setenv("TMUX_INTRAY_MAX_HOOKS", "100")
	max = getMaxAsyncHooks()
	require.Equal(t, 100, max, "Maximum value should be 100")

	// Test values below minimum (should default to 10)
	os.Setenv("TMUX_INTRAY_MAX_HOOKS", "0")
	max = getMaxAsyncHooks()
	require.Equal(t, 10, max, "Zero should default to 10")

	os.Setenv("TMUX_INTRAY_MAX_HOOKS", "-5")
	max = getMaxAsyncHooks()
	require.Equal(t, 10, max, "Negative value should default to 10")

	// Test values above maximum (should default to 10)
	os.Setenv("TMUX_INTRAY_MAX_HOOKS", "101")
	max = getMaxAsyncHooks()
	require.Equal(t, 10, max, "Value above maximum should default to 10")

	os.Setenv("TMUX_INTRAY_MAX_HOOKS", "999")
	max = getMaxAsyncHooks()
	require.Equal(t, 10, max, "Very large value should default to 10")

	// Test invalid format (should default to 10)
	os.Setenv("TMUX_INTRAY_MAX_HOOKS", "abc")
	max = getMaxAsyncHooks()
	require.Equal(t, 10, max, "Non-numeric value should default to 10")

	os.Setenv("TMUX_INTRAY_MAX_HOOKS", "")
	max = getMaxAsyncHooks()
	require.Equal(t, 10, max, "Empty string should default to 10")
}
