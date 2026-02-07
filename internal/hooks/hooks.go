// Package hooks provides a hook subsystem for extensibility.
package hooks

import (
	"context"
	"fmt"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// async tracking
	asyncPending      sync.WaitGroup
	asyncPendingMu    sync.Mutex
	asyncPendingCount int
)

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
	if err := os.MkdirAll(dir, 0755); err != nil {
		colors.Error(fmt.Sprintf("failed to create hooks directory %s: %v", dir, err))
		return fmt.Errorf("failed to create hooks directory %s: %w", dir, err)
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
		os.Stderr.Write(output)
	}
	if err != nil {
		switch failureMode {
		case "abort":
			return fmt.Errorf("hook %s failed: %v, output: %s", scriptName, err, output)
		case "warn":
			fmt.Fprintf(os.Stderr, "warning: hook %s failed: %v, output: %s\n", scriptName, err, output)
		case "ignore":
			// do nothing
		}
	} else {
		// Success: log duration
		fmt.Fprintf(os.Stderr, "  Hook completed in %.2fs\n", duration.Seconds())
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
		if failureMode != "ignore" {
			fmt.Fprintf(os.Stderr, "warning: async hook %s failed to start: %v\n", scriptName, err)
		}
		// Decrement pending count on start failure
		asyncPendingMu.Lock()
		asyncPendingCount--
		asyncPendingMu.Unlock()
		asyncPending.Done()
		return
	}
	// Track start time for hung hook detection
	startTime := time.Now()

	// Wait for completion in goroutine, then decrement count
	go func() {
		// Ensure we always clean up, even on panic
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "error: async hook %s panicked: %v\n", scriptName, r)
			}
			// Always decrement count, even on panic
			asyncPendingMu.Lock()
			asyncPendingCount--
			asyncPendingMu.Unlock()
			// Always signal completion, even on panic
			asyncPending.Done()
			cancel() // ensure cancel is called after wait returns
		}()

		// Wait for command completion
		err := cmd.Wait()
		duration := time.Since(startTime)

		// Check if hook exceeded timeout (context was canceled)
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Fprintf(os.Stderr, "warning: async hook %s timed out after %.2fs\n", scriptName, duration.Seconds())
		}

		// Log hook execution result
		if err != nil && failureMode != "ignore" {
			fmt.Fprintf(os.Stderr, "warning: async hook %s failed: %v (duration: %.2fs)\n", scriptName, err, duration.Seconds())
		} else if err == nil {
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
	// Build environment map
	envMap := make(map[string]string)
	envMap["HOOK_POINT"] = hookPoint
	envMap["TMUX_INTRAY_HOOKS_FAILURE_MODE"] = getFailureMode()
	envMap["HOOK_TIMESTAMP"] = time.Now().Format(time.RFC3339)
	// Add the tmux-intray binary path to help hooks find it
	// Try multiple methods to find the binary
	var tmuxIntrayPath string
	// Method 1: Use os.Executable()
	if exe, err := os.Executable(); err == nil {
		tmuxIntrayPath = exe
	}
	// Method 2: Check if we're being called directly and use argv[0]
	if len(os.Args) > 0 && os.Args[0] != "" {
		// Resolve relative paths
		if filepath.IsAbs(os.Args[0]) {
			tmuxIntrayPath = os.Args[0]
		} else {
			// Try to find the command in PATH
			if path, err := exec.LookPath(os.Args[0]); err == nil {
				tmuxIntrayPath = path
			}
		}
	}
	// Method 3: Try common installation paths if still not found
	if tmuxIntrayPath == "" {
		// Check common installation directories
		home, _ := os.UserHomeDir()
		commonPaths := []string{
			filepath.Join(home, ".local", "bin", "tmux-intray"),
			"/usr/local/bin/tmux-intray",
			"/usr/bin/tmux-intray",
		}
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				tmuxIntrayPath = path
				break
			}
		}
	}
	if tmuxIntrayPath != "" {
		envMap["TMUX_INTRAY_BINARY"] = tmuxIntrayPath
	}
	for _, v := range envVars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Collect executable hook scripts
	type scriptInfo struct {
		path string
		name string
	}
	scripts := []scriptInfo{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		scriptPath := filepath.Join(hookDir, f.Name())
		info, err := os.Stat(scriptPath)
		if err != nil || info.Mode()&0111 == 0 {
			// Not executable
			continue
		}
		scripts = append(scripts, scriptInfo{path: scriptPath, name: f.Name()})
	}
	// Sort by name (ascending)
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].name < scripts[j].name
	})

	if len(scripts) == 0 {
		return nil
	}

	// Log hook execution (similar to Bash)
	fmt.Fprintf(os.Stderr, "Running %s hooks (%d script(s))\n", hookPoint, len(scripts))

	failureMode := getFailureMode()
	asyncEnabled := getAsyncEnabled()
	maxAsync := getMaxAsyncHooks()

	for _, script := range scripts {
		fmt.Fprintf(os.Stderr, "  Executing hook: %s\n", script.name)
		if asyncEnabled {
			// Check if we can start another async hook
			asyncPendingMu.Lock()
			if asyncPendingCount >= maxAsync {
				asyncPendingMu.Unlock()
				fmt.Fprintf(os.Stderr, "warning: Too many async hooks pending (max: %d), skipping %s\n", maxAsync, script.name)
				continue
			}
			asyncPendingCount++
			asyncPending.Add(1)
			asyncPendingMu.Unlock()
			// Start async hook
			fmt.Fprintf(os.Stderr, "  Starting hook asynchronously: %s\n", script.name)
			go runAsyncHook(script.path, script.name, envMap, failureMode)
		} else {
			// Synchronous execution
			if err := runSyncHook(script.path, script.name, envMap, failureMode); err != nil {
				if failureMode == "abort" {
					return err
				}

				// warn or ignore: continue
			}
		}
	}
	return nil
}

// ResetForTesting resets internal state for testing.
// Precondition: All async hooks must have completed before calling this.
// Violating this precondition will cause a panic (fail-fast).
func ResetForTesting() {
	asyncPendingMu.Lock()
	defer asyncPendingMu.Unlock()
	if asyncPendingCount > 0 {
		panic(fmt.Sprintf("ResetForTesting called with %d pending hooks. Call WaitForPendingHooks() first.", asyncPendingCount))
	}
	asyncPendingCount = 0
	asyncPending = sync.WaitGroup{}
}

// WaitForPendingHooks waits for all pending async hooks to complete.
func WaitForPendingHooks() {
	asyncPending.Wait()
}

// Shutdown gracefully shuts down the hooks subsystem.
func Shutdown() {
	WaitForPendingHooks()
}
