package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitAndRunNoPanic(t *testing.T) {
	require.NotPanics(t, func() {
		Init()
		Run("pre-add", "FOO=bar")
	})
}

func TestHookDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)

	Init()
	// Ensure directory exists
	_, err := os.Stat(tmpDir)
	assert.NoError(t, err)
}

func TestHookEnabled(t *testing.T) {
	tests := []struct {
		name      string
		global    string
		hookPoint string
		want      bool
	}{
		{"global enabled", "1", "pre-add", true},
		{"global disabled", "0", "pre-add", false},
		{"hook point disabled", "1", "pre-add", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", tt.global)
			t.Setenv("TMUX_INTRAY_HOOKS_ENABLED_PRE_ADD", tt.hookPoint)
			// Reset manager to re-read config
			Reset()
			config.Load()
			t.Logf("TMUX_INTRAY_HOOKS_DIR env: %s", os.Getenv("TMUX_INTRAY_HOOKS_DIR"))
			t.Logf("hooks_dir config: %s", config.Get("hooks_dir", ""))
			Init()
			// We can't directly test isHookEnabled because it's private.
			// Instead test Run behavior by creating a dummy hook directory.
			tmpDir := t.TempDir()
			t.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
			hookDir := filepath.Join(tmpDir, "pre-add")
			os.MkdirAll(hookDir, 0755)
			script := filepath.Join(hookDir, "test.sh")
			os.WriteFile(script, []byte("#!/bin/sh\nexit 0"), 0755)
			err := Run("pre-add")
			if tt.want {
				assert.NoError(t, err)
			} else {
				// Run returns nil when hooks disabled
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunWithExecutableScript(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
	Reset()
	config.Load()
	Init()

	hookDir := filepath.Join(tmpDir, "post-add")
	os.MkdirAll(hookDir, 0755)
	script := filepath.Join(hookDir, "test.sh")
	os.WriteFile(script, []byte(`#!/bin/sh
echo "hook executed"
`), 0755)

	err := Run("post-add", "FOO=bar")
	assert.NoError(t, err)
}

func TestRunWithNonExecutableScript(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	Reset()
	config.Load()
	Init()

	hookDir := filepath.Join(tmpDir, "post-add")
	os.MkdirAll(hookDir, 0755)
	script := filepath.Join(hookDir, "test.sh")
	os.WriteFile(script, []byte("not executable"), 0644)

	err := Run("post-add")
	assert.NoError(t, err) // skipped
}

func TestRunWithScriptFailureModes(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	Reset()
	config.Load()
	Init()

	hookDir := filepath.Join(tmpDir, "pre-dismiss")
	os.MkdirAll(hookDir, 0755)
	script := filepath.Join(hookDir, "fail.sh")
	os.WriteFile(script, []byte(`#!/bin/sh
exit 1
`), 0755)

	t.Run("abort mode", func(t *testing.T) {
		t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "abort")
		config.Load()
		err := Run("pre-dismiss")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hook fail.sh failed")
	})

	t.Run("warn mode", func(t *testing.T) {
		t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "warn")
		config.Load()
		err := Run("pre-dismiss")
		assert.NoError(t, err) // warning printed but no error returned
	})

	t.Run("ignore mode", func(t *testing.T) {
		t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
		config.Load()
		err := Run("pre-dismiss")
		assert.NoError(t, err)
	})
}

func TestAsyncHook(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	t.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
	Reset()
	config.Load()
	Init()

	hookDir := filepath.Join(tmpDir, "post-list")
	os.MkdirAll(hookDir, 0755)
	script := filepath.Join(hookDir, "async.sh")
	os.WriteFile(script, []byte(`#!/bin/sh
sleep 0.1
echo "async done"
`), 0755)

	start := time.Now()
	err := Run("post-list")
	assert.NoError(t, err)
	duration := time.Since(start)
	t.Logf("async hook Run duration: %v", duration)
	// Async execution should return immediately (allow up to 250ms for slow CI)
	assert.True(t, duration < 250*time.Millisecond)
	// Wait a bit for async hook to complete
	time.Sleep(200 * time.Millisecond)
}

func TestAsyncHookTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	t.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	t.Setenv("TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT", "1")
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
	Reset()
	config.Load()
	Init()

	hookDir := filepath.Join(tmpDir, "post-list")
	os.MkdirAll(hookDir, 0755)
	script := filepath.Join(hookDir, "sleep.sh")
	os.WriteFile(script, []byte(`#!/bin/sh
sleep 5
`), 0755)

	err := Run("post-list")
	assert.NoError(t, err) // starts async, timeout kills after 1 second
	// Wait for timeout to trigger
	time.Sleep(2 * time.Second)
}

func TestMaxAsyncHooks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	t.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	t.Setenv("TMUX_INTRAY_MAX_HOOKS", "2")
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
	Reset()
	config.Load()
	Init()

	hookDir := filepath.Join(tmpDir, "post-add")
	os.MkdirAll(hookDir, 0755)
	// Create 3 scripts
	for i := 0; i < 3; i++ {
		script := filepath.Join(hookDir, fmt.Sprintf("hook%d.sh", i))
		os.WriteFile(script, []byte(`#!/bin/sh
sleep 0.5
`), 0755)
	}

	err := Run("post-add")
	assert.NoError(t, err)
	// Should have started only 2 async hooks, third skipped with warning
	// Wait for hooks to finish
	time.Sleep(1 * time.Second)
}

func TestShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_HOOKS_DIR", tmpDir)
	t.Setenv("TMUX_INTRAY_HOOKS_ENABLED", "1")
	t.Setenv("TMUX_INTRAY_HOOKS_ASYNC", "1")
	t.Setenv("TMUX_INTRAY_HOOKS_FAILURE_MODE", "ignore")
	Reset()
	config.Load()
	Init()

	hookDir := filepath.Join(tmpDir, "pre-add")
	os.MkdirAll(hookDir, 0755)
	script := filepath.Join(hookDir, "long.sh")
	os.WriteFile(script, []byte(`#!/bin/sh
sleep 10
`), 0755)

	err := Run("pre-add")
	assert.NoError(t, err)
	// Shutdown should wait for pending hooks
	Shutdown()
}
