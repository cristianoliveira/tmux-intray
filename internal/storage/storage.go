// Package storage provides file-based TSV storage with locking.
package storage

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
)

// Default state directory follows XDG_STATE_HOME specification.
func getStateDir() string {
	if dir := os.Getenv("TMUX_INTRAY_STATE_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "tmux-intray")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "tmux-intray")
}

func getNotificationsFile() string {
	return filepath.Join(getStateDir(), "notifications.tsv")
}

func getLockDir() string {
	return filepath.Join(getStateDir(), "lock")
}

// Init initializes storage.
func Init() {
	dir := getStateDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		colors.Warning("failed to create state directory:", err.Error())
	}
	notificationsFile := getNotificationsFile()
	if _, err := os.Stat(notificationsFile); os.IsNotExist(err) {
		if err := os.WriteFile(notificationsFile, []byte{}, 0644); err != nil {
			colors.Warning("failed to create notifications file:", err.Error())
		}
	}
	dismissedFile := filepath.Join(dir, "dismissed.tsv")
	if _, err := os.Stat(dismissedFile); os.IsNotExist(err) {
		if err := os.WriteFile(dismissedFile, []byte{}, 0644); err != nil {
			colors.Warning("failed to create dismissed file:", err.Error())
		}
	}
}

// AddNotification adds a notification and returns its ID.
func AddNotification(message, timestamp, session, window, pane, paneCreated, level string) string {
	_ = message
	_ = timestamp
	_ = session
	_ = window
	_ = pane
	_ = paneCreated
	_ = level
	return ""
}

// ListNotifications returns TSV lines for notifications.
func ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) string {
	_ = stateFilter
	_ = levelFilter
	_ = sessionFilter
	_ = windowFilter
	_ = paneFilter
	_ = olderThanCutoff
	_ = newerThanCutoff
	return ""
}

// DismissNotification dismisses a notification by ID.
func DismissNotification(id string) error {
	colors.Debug("DismissNotification called for ID:", id)
	// Run pre-clear hooks? No, only per-notification hooks.
	return withLock(func() error {
		line, err := getLatestLineForID(id)
		if err != nil {
			return fmt.Errorf("failed to read notifications: %w", err)
		}
		if line == "" {
			return fmt.Errorf("Notification with ID %s not found", id)
		}
		lineID, timestamp, state, session, window, pane, message, paneCreated, level := parseLine(line)
		if lineID != id {
			// Should never happen
			return errors.New("internal error: ID mismatch")
		}
		if state == "dismissed" {
			return fmt.Errorf("Notification %s is already dismissed", id)
		}
		// Run pre-dismiss hooks
		envVars := []string{
			fmt.Sprintf("NOTIFICATION_ID=%s", id),
			fmt.Sprintf("LEVEL=%s", level),
			fmt.Sprintf("MESSAGE=%s", message),
			fmt.Sprintf("ESCAPED_MESSAGE=%s", message), // same as raw for now
			fmt.Sprintf("TIMESTAMP=%s", timestamp),
			fmt.Sprintf("SESSION=%s", session),
			fmt.Sprintf("WINDOW=%s", window),
			fmt.Sprintf("PANE=%s", pane),
			fmt.Sprintf("PANE_CREATED=%s", paneCreated),
		}
		if err := hooks.Run("pre-dismiss", envVars...); err != nil {
			return err
		}
		// Append dismissed version
		err = appendLine(id, timestamp, "dismissed", session, window, pane, message, paneCreated, level)
		if err != nil {
			return err
		}
		// Run post-dismiss hooks
		if err := hooks.Run("post-dismiss", envVars...); err != nil {
			return err
		}
		updateTmuxStatusOption()
		return nil
	})
}

// withLock runs fn while holding a file system lock.
func withLock(fn func() error) error {
	lockDir := getLockDir()
	timeout := 10 * time.Second
	start := time.Now()
	for {
		err := os.Mkdir(lockDir, 0755)
		if err == nil {
			break
		}
		if time.Since(start) > timeout {
			return errors.New("timeout acquiring lock")
		}
		time.Sleep(100 * time.Millisecond)
	}
	defer os.RemoveAll(lockDir)
	return fn()
}

// parseLine parses a TSV line into fields.
func parseLine(line string) (id, timestamp, state, session, window, pane, message, paneCreated, level string) {
	parts := strings.Split(line, "\t")
	if len(parts) > 0 {
		id = parts[0]
	}
	if len(parts) > 1 {
		timestamp = parts[1]
	}
	if len(parts) > 2 {
		state = parts[2]
	}
	if len(parts) > 3 {
		session = parts[3]
	}
	if len(parts) > 4 {
		window = parts[4]
	}
	if len(parts) > 5 {
		pane = parts[5]
	}
	if len(parts) > 6 {
		message = parts[6]
	}
	if len(parts) > 7 {
		paneCreated = parts[7]
	}
	if len(parts) > 8 {
		level = parts[8]
	}
	return
}

// getLatestActiveLines returns the latest active notification lines.
func getLatestActiveLines() ([]string, error) {
	file := getNotificationsFile()
	colors.Debug("reading notifications file:", file)
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	colors.Debug("file size:", fmt.Sprintf("%d bytes", len(data)))
	lines := strings.Split(string(data), "\n")
	colors.Debug("total lines:", fmt.Sprintf("%d", len(lines)))
	// Map from ID to latest line (keeping only last occurrence)
	latest := make(map[string]string)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		id, _, state, _, _, _, _, _, _ := parseLine(line)
		if state == "active" {
			if _, exists := latest[id]; !exists {
				latest[id] = line
			}
		}
	}
	var result []string
	for _, line := range latest {
		result = append(result, line)
	}
	colors.Debug("active lines found:", fmt.Sprintf("%d", len(result)))
	return result, nil
}

// getLatestLineForID returns the latest line for a notification ID.
func getLatestLineForID(id string) (string, error) {
	file := getNotificationsFile()
	colors.Debug("reading notifications file:", file)
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	// Iterate from end to find latest occurrence of ID
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		lineID, _, _, _, _, _, _, _, _ := parseLine(line)
		if lineID == id {
			return line, nil
		}
	}
	return "", nil // not found
}

// appendLine appends a line to the notifications file.
func appendLine(id, timestamp, state, session, window, pane, message, paneCreated, level string) error {
	file := getNotificationsFile()
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", id, timestamp, state, session, window, pane, message, paneCreated, level)
	_, err = f.WriteString(line)
	return err
}

// DismissAll dismisses all active notifications.
func DismissAll() error {
	colors.Debug("DismissAll called")
	// Run pre-clear hooks
	if err := hooks.Run("pre-clear"); err != nil {
		return err
	}
	return withLock(func() error {
		activeLines, err := getLatestActiveLines()
		if err != nil {
			return err
		}
		colors.Debug("processing", fmt.Sprintf("%d", len(activeLines)), "active notifications")
		for _, line := range activeLines {
			id, timestamp, _, session, window, pane, message, paneCreated, level := parseLine(line)
			colors.Debug("dismissing notification ID=" + id)
			// Run pre-dismiss hooks
			envVars := []string{
				fmt.Sprintf("NOTIFICATION_ID=%s", id),
				fmt.Sprintf("LEVEL=%s", level),
				fmt.Sprintf("MESSAGE=%s", message),
				fmt.Sprintf("ESCAPED_MESSAGE=%s", message), // same as raw for now
				fmt.Sprintf("TIMESTAMP=%s", timestamp),
				fmt.Sprintf("SESSION=%s", session),
				fmt.Sprintf("WINDOW=%s", window),
				fmt.Sprintf("PANE=%s", pane),
				fmt.Sprintf("PANE_CREATED=%s", paneCreated),
			}
			if err := hooks.Run("pre-dismiss", envVars...); err != nil {
				return err
			}
			// Append dismissed version
			err = appendLine(id, timestamp, "dismissed", session, window, pane, message, paneCreated, level)
			if err != nil {
				return err
			}
			// Run post-dismiss hooks
			if err := hooks.Run("post-dismiss", envVars...); err != nil {
				return err
			}
		}
		updateTmuxStatusOption()
		return nil
	})
}

// CleanupOldNotifications cleans up notifications older than the threshold.
func CleanupOldNotifications(daysThreshold int, dryRun bool) {
	_ = daysThreshold
	_ = dryRun
}

// updateTmuxStatusOption updates the tmux status option with the current active count.
func updateTmuxStatusOption() {
	// Only update if tmux is running
	cmd := exec.Command("tmux", "has-session")
	if err := cmd.Run(); err != nil {
		// tmux not running, skip
		return
	}
	count := GetActiveCount()
	cmd = exec.Command("tmux", "set", "-g", "@tmux_intray_active_count", fmt.Sprintf("%d", count))
	cmd.Run() // ignore error
}

// GetActiveCount returns the active notification count.
func GetActiveCount() int {
	activeLines, err := getLatestActiveLines()
	if err != nil {
		colors.Debug("failed to get active lines:", err.Error())
		return 0
	}
	return len(activeLines)
}
