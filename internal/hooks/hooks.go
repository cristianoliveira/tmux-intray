// Package hooks provides a hook subsystem for extensibility.
package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
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
func Init() {
	m := getManager()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.initialized {
		return
	}
	// Ensure configuration is loaded
	config.Load()
	// Ensure hooks directory exists
	dir := getHooksDir()
	os.MkdirAll(dir, 0755)
	// Start signal handler
	go m.signalHandler()
	m.initialized = true
	colors.Debug("hooks subsystem initialized")
}

// getHooksDir returns the hooks directory path.
func getHooksDir() string {
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
	if mode := config.Get("hooks_failure_mode", ""); mode != "" {
		return mode
	}
	return "warn"
}

// isHookEnabled returns true if hooks are enabled globally and for the specific hook point.
func isHookEnabled(hookPoint string) bool {
	// Global enable
	if !config.GetBool("hooks_enabled", true) {
		return false
	}
	// Per-hook point enable
	key := "hooks_enabled_" + strings.ReplaceAll(hookPoint, "-", "_")
	return config.GetBool(key, true)
}

// isAsyncEnabled returns true if async execution is enabled.
func isAsyncEnabled() bool {
	return config.GetBool("hooks_async", false)
}

// getAsyncTimeout returns the timeout in seconds for async hooks.
func getAsyncTimeout() time.Duration {
	sec := config.GetInt("hooks_async_timeout", 30)
	return time.Duration(sec) * time.Second
}

// getMaxHooks returns the maximum number of concurrent async hooks.
func getMaxHooks() int {
	return config.GetInt("max_hooks", 10)
}

// Run executes hooks for a hook point with environment variables.
func Run(hookPoint string, envVars ...string) error {
	m := getManager()
	if !isHookEnabled(hookPoint) {
		colors.Debug("hooks disabled for " + hookPoint)
		return nil
	}
	hookDir := filepath.Join(getHooksDir(), hookPoint)
	colors.Debug(fmt.Sprintf("hook directory: %s", hookDir))
	files, err := os.ReadDir(hookDir)
	if err != nil {
		// Directory doesn't exist -> no hooks
		return nil
	}
	// Build environment map
	envMap := make(map[string]string)
	envMap["HOOK_POINT"] = hookPoint
	envMap["TMUX_INTRAY_HOOKS_FAILURE_MODE"] = getFailureMode()
	for _, v := range envVars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	failureMode := getFailureMode()
	async := isAsyncEnabled()
	maxHooks := getMaxHooks()
	timeout := getAsyncTimeout()
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if !strings.HasSuffix(f.Name(), ".sh") {
			continue
		}
		scriptPath := filepath.Join(hookDir, f.Name())
		info, err := os.Stat(scriptPath)
		if err != nil || info.Mode()&0111 == 0 {
			// Not executable
			continue
		}
		colors.Debug(fmt.Sprintf("executing hook script: %s", scriptPath))
		if async {
			// Check pending count
			m.mu.Lock()
			pendingCount := len(m.pending)
			m.mu.Unlock()
			if pendingCount >= maxHooks {
				colors.Warning("too many async hooks pending (max: " + strconv.Itoa(maxHooks) + "), skipping " + f.Name())
				continue
			}
			// Execute asynchronously
			err = m.runAsync(scriptPath, envMap, timeout)
			if err != nil {
				switch failureMode {
				case "abort":
					return fmt.Errorf("async hook %s failed to start: %v", f.Name(), err)
				case "warn":
					colors.Warning(fmt.Sprintf("async hook %s failed to start: %v", f.Name(), err))
				case "ignore":
					// do nothing
				}
			}
		} else {
			// Synchronous execution
			err = runSync(scriptPath, envMap)
			if err != nil {
				switch failureMode {
				case "abort":
					return fmt.Errorf("hook %s failed: %v", f.Name(), err)
				case "warn":
					colors.Warning(fmt.Sprintf("hook %s failed: %v", f.Name(), err))
				case "ignore":
					// do nothing
				}
			}
		}
	}
	return nil
}

// runSync executes a hook script synchronously.
func runSync(scriptPath string, envMap map[string]string) error {
	start := time.Now()
	cmd := exec.Command(scriptPath)
	// Build environment
	cmd.Env = os.Environ()
	for k, v := range envMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	output, err := cmd.CombinedOutput()
	if cmd.ProcessState != nil {
		colors.Debug(fmt.Sprintf("hook exit code: %v", cmd.ProcessState.ExitCode()))
	}
	if len(output) > 0 {
		colors.Debug(fmt.Sprintf("hook output: %s", string(output)))
	}
	colors.Debug(fmt.Sprintf("hook err: %v", err))
	duration := time.Since(start)
	if err != nil {
		colors.Debug(fmt.Sprintf("hook %s failed after %v: %v, output: %s", filepath.Base(scriptPath), duration, err, output))
		exitCode := -1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		return fmt.Errorf("exit %v: %s", exitCode, output)
	}
	colors.Debug(fmt.Sprintf("hook %s completed in %v", filepath.Base(scriptPath), duration))
	if len(output) > 0 {
		colors.Debug(fmt.Sprintf("hook output: %s", string(output)))
	}
	return nil
}

// runAsync starts a hook script asynchronously with timeout.
func (m *hookManager) runAsync(scriptPath string, envMap map[string]string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, scriptPath)
	// Build environment
	cmd.Env = os.Environ()
	for k, v := range envMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	// Redirect stdout/stderr to stderr (like bash)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	hook := &pendingHook{
		cmd:    cmd,
		cancel: cancel,
		start:  time.Now(),
	}
	m.mu.Lock()
	m.pending[pid] = hook
	m.mu.Unlock()
	m.wg.Add(1)
	go m.waitAsync(pid, hook)
	colors.Debug(fmt.Sprintf("started async hook %s with PID %d (timeout: %v)", filepath.Base(scriptPath), pid, timeout))
	return nil
}

// waitAsync waits for an async hook to finish and cleans up.
func (m *hookManager) waitAsync(pid int, hook *pendingHook) {
	defer m.wg.Done()
	err := hook.cmd.Wait()
	duration := time.Since(hook.start)
	m.mu.Lock()
	delete(m.pending, pid)
	m.mu.Unlock()
	if err != nil {
		colors.Debug(fmt.Sprintf("async hook PID %d failed after %v: %v", pid, duration, err))
	} else {
		colors.Debug(fmt.Sprintf("async hook PID %d completed in %v", pid, duration))
	}
}

// signalHandler sets up signal handling for graceful shutdown.
func (m *hookManager) signalHandler() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGCHLD)
	for {
		select {
		case sig := <-sigCh:
			switch sig {
			case os.Interrupt, syscall.SIGTERM:
				colors.Debug(fmt.Sprintf("received signal %v, shutting down hooks", sig))
				m.shutdownHooks()
				return
			case syscall.SIGCHLD:
				// Reap child processes (though we already wait)
				// We can ignore because we call Wait in goroutines.
			}
		case <-m.shutdown:
			m.shutdownHooks()
			return
		}
	}
}

// shutdownHooks waits for pending async hooks with a timeout.
func (m *hookManager) shutdownHooks() {
	colors.Debug("waiting for pending async hooks")
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		colors.Debug("all async hooks completed")
	case <-time.After(30 * time.Second):
		colors.Warning("timeout waiting for async hooks, forcing exit")
		// Kill remaining processes
		m.mu.Lock()
		for pid, hook := range m.pending {
			if hook.cmd.Process != nil {
				hook.cmd.Process.Kill()
			}
			delete(m.pending, pid)
		}
		m.mu.Unlock()
	}
}

// Shutdown gracefully shuts down the hooks subsystem.
func Shutdown() {
	m := getManager()
	close(m.shutdown)
}

// Reset resets the hooks subsystem for testing.
func Reset() {
	manager = nil
	once = sync.Once{}
}
