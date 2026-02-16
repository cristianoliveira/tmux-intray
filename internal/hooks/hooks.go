// Package hooks provides a hook subsystem for extensibility.
package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
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

type pendingHook struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	start  time.Time
}

type hookManager struct {
	mu          sync.Mutex
	pending     map[int]*pendingHook // key is PID
	shutdown    chan struct{}
	wg          sync.WaitGroup
	initialized bool
}

func getManager() *hookManager {
	once.Do(func() {
		manager = &hookManager{
			pending:  make(map[int]*pendingHook),
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

// runSyncHook executes a hook script synchronously.
func runSyncHook(scriptPath, scriptName string, envMap map[string]string, failureMode string) error {
	start := time.Now()
	cmd := exec.Command(scriptPath)
	cmd.Env = os.Environ()
	for k, v := range envMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	// Print hook output to stderr (so it appears in logs)
	if len(output) > 0 {
		_, _ = os.Stderr.Write(output)
	}
	if err != nil {
		switch failureMode {
		case "abort":
			return fmt.Errorf("hooks.Run: hook '%s' failed after %.2fs: %v, output: %s", scriptName, duration.Seconds(), err, output)
		case "warn":
			if isHooksVerbose() {
				fmt.Fprintf(os.Stderr, "warning: hook %s failed after %.2fs: %v, output: %s\n", scriptName, duration.Seconds(), err, output)
			}
		case "ignore":
			// do nothing
		}
	} else {
		// Success: log duration
		if isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "  Hook completed in %.2fs\n", duration.Seconds())
		}
	}
	return nil
}

// runAsyncHook executes a hook script asynchronously with timeout.
func runAsyncHook(scriptPath, scriptName string, envMap map[string]string, failureMode string) {
	timeout := getAsyncTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Env = os.Environ()
	for k, v := range envMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	// Redirect stdout to stderr as Bash does
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	// Start the command
	if err := cmd.Start(); err != nil {
		cancel() // release context resources
		if failureMode != "ignore" && isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "warning: async hook %s failed to start: %v\n", scriptName, err)
		}
		// Decrement pending count on start failure
		pendingHooksMu.Lock()
		pendingHookCount--
		pendingHooksMu.Unlock()
		pendingHooks.Done()
		return
	}
	// Track start time for hung hook detection
	startTime := time.Now()

	// Wait for completion in goroutine, then decrement count
	go func() {
		// Ensure we always clean up, even on panic
		defer func() {
			if r := recover(); r != nil {
				if isHooksVerbose() {
					fmt.Fprintf(os.Stderr, "error: async hook %s panicked: %v\n", scriptName, r)
				}
			}
			// Always decrement count, even on panic
			pendingHooksMu.Lock()
			pendingHookCount--
			pendingHooksMu.Unlock()
			// Always signal completion, even on panic
			pendingHooks.Done()
			cancel() // ensure cancel is called after wait returns
		}()

		// Wait for command completion
		err := cmd.Wait()
		duration := time.Since(startTime)

		// Check if hook exceeded timeout (context was canceled)
		if ctx.Err() == context.DeadlineExceeded && isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "warning: async hook %s timed out after %.2fs\n", scriptName, duration.Seconds())
		}

		// Log hook execution result
		if err != nil && failureMode != "ignore" && isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "warning: async hook %s failed: %v (duration: %.2fs)\n", scriptName, err, duration.Seconds())
		} else if err == nil && isHooksVerbose() {
			fmt.Fprintf(os.Stderr, "  async hook %s completed in %.2fs\n", scriptName, duration.Seconds())
		}
	}()
}

// Run executes hooks for a hook point with environment variables.
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

func buildHookEnv(hookPoint string, envVars []string) map[string]string {
	envMap := make(map[string]string)
	envMap["HOOK_POINT"] = hookPoint
	envMap["TMUX_INTRAY_HOOKS_FAILURE_MODE"] = getFailureMode()
	envMap["HOOK_TIMESTAMP"] = time.Now().Format(time.RFC3339)

	if tmuxIntrayPath := resolveTmuxIntrayPath(); tmuxIntrayPath != "" {
		envMap["TMUX_INTRAY_BINARY"] = tmuxIntrayPath
	}

	for _, v := range envVars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	return envMap
}

func resolveTmuxIntrayPath() string {
	var tmuxIntrayPath string
	if exe, err := os.Executable(); err == nil {
		tmuxIntrayPath = exe
	}

	if len(os.Args) > 0 && os.Args[0] != "" {
		if filepath.IsAbs(os.Args[0]) {
			tmuxIntrayPath = os.Args[0]
		} else if path, err := exec.LookPath(os.Args[0]); err == nil {
			tmuxIntrayPath = path
		}
	}

	if tmuxIntrayPath != "" {
		return tmuxIntrayPath
	}

	home, _ := os.UserHomeDir()
	commonPaths := []string{
		filepath.Join(home, ".local", "bin", "tmux-intray"),
		"/usr/local/bin/tmux-intray",
		"/usr/bin/tmux-intray",
	}
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func collectHookScripts(hookDir string, files []os.DirEntry) []hookScript {
	scripts := []hookScript{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		scriptPath := filepath.Join(hookDir, f.Name())
		info, err := os.Stat(scriptPath)
		if err != nil || info.Mode()&0111 == 0 {
			continue
		}
		scripts = append(scripts, hookScript{path: scriptPath, name: f.Name()})
	}

	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].name < scripts[j].name
	})

	return scripts
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

// hookScript holds information about a hook script
type hookScript struct {
	path string
	name string
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
