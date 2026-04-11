// Package hooks provides a hook subsystem for extensibility.
package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
)

// HookResult contains the result of running hooks with modification support.
type HookResult struct {
	// Modified indicates whether any hook modified the notification.
	Modified bool
	// ExitCode is the exit code of the last hook that executed.
	ExitCode int
	// EnvVars contains modified environment variables from hooks that exited with code 2.
	// Keys are variable names, values are the modified values.
	EnvVars map[string]string
	// Rejected indicates whether a hook rejected the notification.
	Rejected bool
}

// Exit codes for hook modification semantics:
// 0 = Accept notification as-is (no modification)
// 1 = Reject notification (don't store)
// 2 = Accept with modifications (read modified env vars from output)
// 3 = Route to alternative action (store + external action) - treated as accept with mod
// 4 = Defer/delay processing - treated as accept (future enhancement)
const (
	ExitCodeAccept = 0
	ExitCodeReject = 1
	ExitCodeModify = 2
	ExitCodeRoute  = 3
	ExitCodeDefer  = 4
)

// File permission constants
const (
	// FileModeDir is the permission for directories (rwxr-xr-x)
	// Owner: read/write/execute, Group/others: read/execute
	FileModeDir os.FileMode = 0755
	// FileModeScript is the permission for executable scripts (rwxr-xr-x)
	// Owner: read/write/execute, Group/others: read/execute
	FileModeScript os.FileMode = 0755
)

var (
	// pendingHooks tracks async hook execution state
	pendingHooks     sync.WaitGroup
	pendingHooksMu   sync.Mutex
	pendingHookCount int
)

// isHooksVerbose checks if verbose mode is enabled via environment variable
func isHooksVerbose() bool {
	return os.Getenv("TMUX_INTRAY_HOOKS_VERBOSE") == "1"
}

var (
	manager *hookManager
	once    sync.Once
)

type hookManager struct {
	mu          sync.Mutex
	shutdown    chan struct{}
	initialized bool
}

func getManager() *hookManager {
	once.Do(func() {
		manager = &hookManager{
			shutdown: make(chan struct{}),
		}
	})
	return manager
}

// Init initializes the hooks subsystem.
func Init() error {
	config.Load()
	m := getManager()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.initialized {
		return nil
	}
	// Ensure hooks directory exists
	dir := getHooksDir()
	if err := os.MkdirAll(dir, FileModeDir); err != nil {
		colors.Error(fmt.Sprintf("hooks.Init: failed to create hooks directory %s: %v", dir, err))
		return fmt.Errorf("hooks.Init: failed to create hooks directory %s: %w", dir, err)
	}
	m.initialized = true
	return nil
}

// getHooksDir returns the hooks directory path.
func getHooksDir() string {
	config.Load()
	// First check environment variable (highest precedence)
	if dir := os.Getenv("TMUX_INTRAY_HOOKS_DIR"); dir != "" {
		return dir
	}
	// Then check config
	if dir := config.Get("hooks_dir", ""); dir != "" {
		colors.Debug(fmt.Sprintf("hooks_dir from config: %s", dir))
		return dir
	}
	// Default: $XDG_CONFIG_HOME/tmux-intray/hooks
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return filepath.Join(configDir, "tmux-intray", "hooks")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tmux-intray", "hooks")
}

// getFailureMode returns the failure mode (abort, warn, ignore).
func getFailureMode() string {
	// Environment variable takes precedence
	if mode := os.Getenv("TMUX_INTRAY_HOOKS_FAILURE_MODE"); mode != "" {
		return mode
	}
	return "warn"
}

// getAsyncEnabled returns true if async hooks are enabled.
func getAsyncEnabled() bool {
	if async := os.Getenv("TMUX_INTRAY_HOOKS_ASYNC"); async != "" {
		return async == "1" || async == "true" || async == "yes" || async == "on"
	}
	return false
}

// getAsyncTimeout returns the timeout in seconds for async hooks.
func getAsyncTimeout() time.Duration {
	if timeoutStr := os.Getenv("TMUX_INTRAY_HOOKS_ASYNC_TIMEOUT"); timeoutStr != "" {
		if seconds, err := time.ParseDuration(timeoutStr + "s"); err == nil {
			return seconds
		}
	}
	return 30 * time.Second
}

// getMaxAsyncHooks returns maximum number of concurrent async hooks.
func getMaxAsyncHooks() int {
	if maxStr := os.Getenv("TMUX_INTRAY_MAX_HOOKS"); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil && max > 0 {
			return max
		}
	}
	return 10
}

// Run executes hooks for a hook point with environment variables.
// This is the original function for backward compatibility.
// For enhanced modification support, use RunWithModification instead.
func Run(hookPoint string, envVars ...string) error {
	hookDir := filepath.Join(getHooksDir(), hookPoint)
	files, err := os.ReadDir(hookDir)
	if err != nil {
		// Directory doesn't exist -> no hooks
		return nil
	}

	envMap := buildHookEnv(hookPoint, envVars)
	scripts := collectHookScripts(hookDir, files)
	if len(scripts) == 0 {
		return nil
	}

	// Log hook execution (similar to Bash)
	if isHooksVerbose() {
		fmt.Fprintf(os.Stderr, "Running %s hooks (%d script(s))\n", hookPoint, len(scripts))
	}

	failureMode := getFailureMode()
	return executeHooks(scripts, envMap, failureMode, getAsyncEnabled(), getMaxAsyncHooks())
}

// RunWithModification executes hooks for a hook point and returns modification results.
// This function supports the enhanced hook semantics where hooks can modify notification
// content by exporting environment variables and exiting with code 2.
//
// Return codes:
//   - 0: Accept notification as-is
//   - 1: Reject notification (don't store)
//   - 2: Accept with modifications (read export statements from output)
//   - 3: Route to alternative action (treated as accept with modifications)
//   - 4: Defer processing (treated as accept)
//
// When async mode is enabled, modifications cannot be captured (hooks run in background).
// In this case, the function returns immediately with Modified=false.
func RunWithModification(hookPoint string, envVars ...string) (HookResult, error) {
	// Async hooks can't capture modifications
	if getAsyncEnabled() {
		// Run in async mode - no modification capture possible
		hookDir := filepath.Join(getHooksDir(), hookPoint)
		files, err := os.ReadDir(hookDir)
		if err != nil {
			return HookResult{ExitCode: 0, EnvVars: make(map[string]string)}, nil
		}

		envMap := buildHookEnv(hookPoint, envVars)
		scripts := collectHookScripts(hookDir, files)
		if len(scripts) == 0 {
			return HookResult{ExitCode: 0, EnvVars: make(map[string]string)}, nil
		}

		failureMode := getFailureMode()
		if isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "Running %s hooks (async, %d script(s))\n", hookPoint, len(scripts))
		}

		// Fire and forget - async hooks run in background
		_ = executeHooks(scripts, envMap, failureMode, true, getMaxAsyncHooks())
		return HookResult{ExitCode: 0, Modified: false, EnvVars: make(map[string]string)}, nil
	}

	// Sync mode - we can capture modifications
	hookDir := filepath.Join(getHooksDir(), hookPoint)
	files, err := os.ReadDir(hookDir)
	if err != nil {
		// Directory doesn't exist -> no hooks
		return HookResult{ExitCode: 0, EnvVars: make(map[string]string)}, nil
	}

	envMap := buildHookEnv(hookPoint, envVars)
	scripts := collectHookScripts(hookDir, files)
	if len(scripts) == 0 {
		return HookResult{ExitCode: 0, EnvVars: make(map[string]string)}, nil
	}

	// Log hook execution
	if isHooksVerbose() {
		fmt.Fprintf(os.Stderr, "Running %s hooks (%d script(s))\n", hookPoint, len(scripts))
	}

	failureMode := getFailureMode()

	// Execute hooks synchronously and track modifications
	result := HookResult{
		EnvVars:  make(map[string]string),
		ExitCode: 0,
		Modified: false,
	}

	for _, script := range scripts {
		if isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "  Executing hook: %s\n", script.name)
		}

		execResult := runSyncHookWithResult(script.path, script.name, envMap, failureMode)

		// Track the last exit code
		result.ExitCode = execResult.ExitCode

		// Check for modification semantics (exit code 2, 3, or 4)
		if execResult.ExitCode == ExitCodeModify || execResult.ExitCode == ExitCodeRoute || execResult.ExitCode == ExitCodeDefer {
			// Parse modifications from output
			mods := parseModifications(execResult.Output)
			for k, v := range mods {
				result.EnvVars[k] = v
			}
			result.Modified = true
		}

		// Check for rejection (exit code 1) - always return error for rejection
		// This is a semantic signal from the hook, not just a failure
		if execResult.ExitCode == ExitCodeReject {
			result.Rejected = true
			return result, fmt.Errorf("hook '%s' rejected notification", script.name)
		}

		// Handle other errors (non-rejection) based on failure mode
		if execResult.Error != nil {
			if failureMode == "abort" {
				return result, execResult.Error
			}
			// warn or ignore: continue silently
		}
	}

	return result, nil
}

func executeHooks(scripts []hookScript, envMap map[string]string, failureMode string, asyncEnabled bool, maxAsync int) error {
	for _, script := range scripts {
		if isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "  Executing hook: %s\n", script.name)
		}
		if asyncEnabled {
			tryStartAsyncHook(script, envMap, failureMode, maxAsync)
			continue
		}

		if err := runSyncHookAndCheckAbort(script, envMap, failureMode); err != nil {
			return err
		}
	}
	return nil
}

// runSyncHookAndCheckAbort runs a sync hook and returns error only if abort mode.
func runSyncHookAndCheckAbort(script hookScript, envMap map[string]string, failureMode string) error {
	if err := runSyncHook(script.path, script.name, envMap, failureMode); err != nil {
		if failureMode == "abort" {
			return err
		}
		// warn or ignore: continue
	}
	return nil
}

// tryStartAsyncHook attempts to start an async hook, returns false if limit reached.
func tryStartAsyncHook(script hookScript, envMap map[string]string, failureMode string, maxAsync int) bool {
	// Check if we can start another async hook
	pendingHooksMu.Lock()
	defer pendingHooksMu.Unlock()

	if pendingHookCount >= maxAsync {
		if isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "warning: Too many async hooks pending (max: %d), skipping %s\n", maxAsync, script.name)
		}
		return false
	}
	pendingHookCount++
	pendingHooks.Add(1)
	// Start async hook
	if isHooksVerbose() {
		fmt.Fprintf(os.Stderr, "  Starting hook asynchronously: %s\n", script.name)
	}
	go runAsyncHook(script.path, script.name, envMap, failureMode)
	return true
}

// ResetForTesting resets internal state for testing.
// Precondition: All async hooks must have completed before calling this.
// Violating this precondition will cause a panic (fail-fast).
func ResetForTesting() {
	pendingHooksMu.Lock()
	defer pendingHooksMu.Unlock()
	if pendingHookCount > 0 {
		panic(fmt.Sprintf("hooks.ResetForTesting: called with %d pending hooks. Call hooks.WaitForPendingHooks() first.", pendingHookCount))
	}
	pendingHookCount = 0
	pendingHooks = sync.WaitGroup{}
}

// WaitForPendingHooks waits for all pending async hooks to complete.
func WaitForPendingHooks() {
	pendingHooks.Wait()
}

// Shutdown gracefully shuts down the hooks subsystem.
func Shutdown() {
	WaitForPendingHooks()
}
