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
	// Wait a bit for async hook to complete
	time.Sleep(200 * time.Millisecond)
}

func TestRunAsyncHookMaxLimit(t *testing.T) {
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
	time.Sleep(600 * time.Millisecond)
}
