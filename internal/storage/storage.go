// Package storage provides file-based TSV storage with locking.
package storage

import (
	"errors"
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
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
)

const (
	fieldID          = 0
	fieldTimestamp   = 1
	fieldState       = 2
	fieldSession     = 3
	fieldWindow      = 4
	fieldPane        = 5
	fieldMessage     = 6
	fieldPaneCreated = 7
	fieldLevel       = 8
	numFields        = 9
)

var (
	notificationsFile string
	lockDir           string
	initOnce          = &sync.Once{}
	initialized       bool
	initMu            sync.RWMutex
	initErr           error
)

// Init initializes storage directories and files.
// Returns an error if initialization fails. Safe for concurrent calls.
func Init() error {
	var err error
	initOnce.Do(func() {
		// Load configuration
		config.Load()

		// Prefer environment variable directly (should match config.Load but ensure it works)
		stateDir := os.Getenv("TMUX_INTRAY_STATE_DIR")
		if stateDir == "" {
			stateDir = config.Get("state_dir", "")
		}
		colors.Debug("state_dir: " + stateDir)
		if stateDir == "" {
			err = fmt.Errorf("state_dir not configured")
			return
		}
		notificationsFile = filepath.Join(stateDir, "notifications.tsv")
		lockDir = filepath.Join(stateDir, "lock")

		// Ensure directories exist
		if err = os.MkdirAll(stateDir, 0755); err != nil {
			err = fmt.Errorf("failed to create state directory: %w", err)
			return
		}

		// Ensure notifications file exists
		var f *os.File
		f, err = os.OpenFile(notificationsFile, os.O_RDONLY|os.O_CREATE, 0644)
		if err != nil {
			err = fmt.Errorf("failed to create notifications file: %w", err)
			return
		}
		f.Close()

		// Mark initialized only if all steps succeeded
		initMu.Lock()
		initialized = true
		initErr = nil
		initMu.Unlock()

		colors.Debug("storage initialized")
	})

	// Return any initialization error from first call
	if err != nil {
		return err
	}

	// Check if there was an error from a previous initialization attempt
	initMu.RLock()
	err = initErr
	initMu.RUnlock()
	return err
}

// AddNotification adds a notification and returns its ID.
func AddNotification(message, timestamp, session, window, pane, paneCreated, level string) string {
	if err := Init(); err != nil {
		colors.Error(fmt.Sprintf("failed to initialize storage: %v", err))
		return ""
	}
	// Generate ID
	id, err := getNextID()
	if err != nil {
		colors.Error(fmt.Sprintf("failed to generate ID: %v", err))
		return ""
	}
	// Use provided timestamp or generate current UTC
	if timestamp == "" {
		timestamp = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	}
	// Escape message
	escapedMessage := escapeMessage(message)

	// Run pre-add hooks
	envVars := []string{
		fmt.Sprintf("NOTIFICATION_ID=%d", id),
		fmt.Sprintf("LEVEL=%s", level),
		fmt.Sprintf("MESSAGE=%s", message),
		fmt.Sprintf("ESCAPED_MESSAGE=%s", escapedMessage),
		fmt.Sprintf("TIMESTAMP=%s", timestamp),
		fmt.Sprintf("SESSION=%s", session),
		fmt.Sprintf("WINDOW=%s", window),
		fmt.Sprintf("PANE=%s", pane),
		fmt.Sprintf("PANE_CREATED=%s", paneCreated),
	}
	if err := hooks.Run("pre-add", envVars...); err != nil {
		colors.Error(fmt.Sprintf("pre-add hook aborted: %v", err))
		return ""
	}

	// Append line with lock
	err = WithLock(lockDir, func() error {
		return appendLine(id, timestamp, "active", session, window, pane, escapedMessage, paneCreated, level)
	})
	if err != nil {
		colors.Error(fmt.Sprintf("failed to add notification: %v", err))
		return ""
	}
	// Update tmux status option outside lock to avoid deadlock (updateTmuxStatusOption also acquires a lock) and keep lock duration short
	updateTmuxStatusOption()

	// Run post-add hooks
	if err := hooks.Run("post-add", envVars...); err != nil {
		colors.Error(fmt.Sprintf("post-add hook aborted: %v", err))
		// Still return ID because notification was added
	}

	// Return ID as string
	return strconv.Itoa(id)
}

// ListNotifications returns TSV lines for notifications.
func ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) string {
	if err := Init(); err != nil {
		colors.Error(fmt.Sprintf("failed to initialize storage: %v", err))
		return ""
	}
	var lines []string
	err := WithLock(lockDir, func() error {
		latest, err := getLatestNotifications()
		if err != nil {
			return err
		}
		// Apply filters
		filtered := filterNotifications(latest, stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
		lines = filtered
		return nil
	})
	if err != nil {
		colors.Error(fmt.Sprintf("failed to list notifications: %v", err))
		return ""
	}
	return strings.Join(lines, "\n")
}

// GetNotificationByID retrieves a single notification by its ID.
// This is an optimized version that avoids reading all notifications when possible.
// Returns the notification line as a TSV string or an error if not found.
func GetNotificationByID(id string) (string, error) {
	if err := Init(); err != nil {
		return "", fmt.Errorf("storage not initialized: %w", err)
	}

	// Validate ID format
	if id == "" {
		return "", errors.New("notification ID cannot be empty")
	}

	var result string
	err := WithLock(lockDir, func() error {
		// Get all notifications to find the latest version of the requested ID
		// Note: This is necessary to ensure we get the latest state (active/dismissed)
		latest, err := getLatestNotifications()
		if err != nil {
			return fmt.Errorf("failed to read notifications: %w", err)
		}

		// Find the notification with matching ID
		for _, line := range latest {
			fields := strings.Split(line, "\t")
			if len(fields) > fieldID && fields[fieldID] == id {
				result = line
				return nil
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if result == "" {
		return "", fmt.Errorf("notification with ID %s not found", id)
	}

	return result, nil
}

// DismissNotification dismisses a notification by ID.
func DismissNotification(id string) error {
	if err := Init(); err != nil {
		return fmt.Errorf("storage not initialized: %w", err)
	}
	colors.Debug("DismissNotification called for ID:", id)
	err := WithLock(lockDir, func() error {
		latest, err := getLatestNotifications()
		if err != nil {
			return fmt.Errorf("failed to read notifications: %w", err)
		}
		var targetLine string
		for _, line := range latest {
			fields := strings.Split(line, "\t")
			if len(fields) > fieldID && fields[fieldID] == id {
				targetLine = line
				break
			}
		}
		if targetLine == "" {
			return fmt.Errorf("notification %s not found", id)
		}
		fields := strings.Split(targetLine, "\t")
		if len(fields) < numFields {
			return fmt.Errorf("invalid line format")
		}
		if fields[fieldState] == "dismissed" {
			return fmt.Errorf("notification %s is already dismissed", id)
		}
		envVars := []string{
			fmt.Sprintf("NOTIFICATION_ID=%s", id),
			fmt.Sprintf("LEVEL=%s", fields[fieldLevel]),
			fmt.Sprintf("MESSAGE=%s", fields[fieldMessage]),
			fmt.Sprintf("ESCAPED_MESSAGE=%s", fields[fieldMessage]),
			fmt.Sprintf("TIMESTAMP=%s", fields[fieldTimestamp]),
			fmt.Sprintf("SESSION=%s", fields[fieldSession]),
			fmt.Sprintf("WINDOW=%s", fields[fieldWindow]),
			fmt.Sprintf("PANE=%s", fields[fieldPane]),
			fmt.Sprintf("PANE_CREATED=%s", fields[fieldPaneCreated]),
		}
		if err := hooks.Run("pre-dismiss", envVars...); err != nil {
			return err
		}
		idInt, err := strToInt(fields[fieldID])
		if err != nil {
			return fmt.Errorf("invalid ID %s: %w", fields[fieldID], err)
		}
		if err := appendLine(
			idInt,
			fields[fieldTimestamp],
			"dismissed",
			fields[fieldSession],
			fields[fieldWindow],
			fields[fieldPane],
			fields[fieldMessage],
			fields[fieldPaneCreated],
			fields[fieldLevel],
		); err != nil {
			return err
		}
		if err := hooks.Run("post-dismiss", envVars...); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	updateTmuxStatusOption()
	return nil
}

// DismissAll dismisses all active notifications.
func DismissAll() error {
	if err := Init(); err != nil {
		return fmt.Errorf("storage not initialized: %w", err)
	}
	colors.Debug("DismissAll called")
	if err := hooks.Run("pre-clear"); err != nil {
		return err
	}
	err := WithLock(lockDir, func() error {
		latest, err := getLatestNotifications()
		if err != nil {
			return err
		}
		for _, line := range latest {
			fields := strings.Split(line, "\t")
			if len(fields) < numFields {
				for len(fields) < numFields {
					fields = append(fields, "")
				}
			}
			if fields[fieldState] != "active" {
				continue
			}
			id := fields[fieldID]
			envVars := []string{
				fmt.Sprintf("NOTIFICATION_ID=%s", id),
				fmt.Sprintf("LEVEL=%s", fields[fieldLevel]),
				fmt.Sprintf("MESSAGE=%s", fields[fieldMessage]),
				fmt.Sprintf("ESCAPED_MESSAGE=%s", fields[fieldMessage]),
				fmt.Sprintf("TIMESTAMP=%s", fields[fieldTimestamp]),
				fmt.Sprintf("SESSION=%s", fields[fieldSession]),
				fmt.Sprintf("WINDOW=%s", fields[fieldWindow]),
				fmt.Sprintf("PANE=%s", fields[fieldPane]),
				fmt.Sprintf("PANE_CREATED=%s", fields[fieldPaneCreated]),
			}
			if err := hooks.Run("pre-dismiss", envVars...); err != nil {
				return err
			}
			idInt, err := strToInt(fields[fieldID])
			if err != nil {
				return fmt.Errorf("invalid ID %s: %w", fields[fieldID], err)
			}
			if err := appendLine(
				idInt,
				fields[fieldTimestamp],
				"dismissed",
				fields[fieldSession],
				fields[fieldWindow],
				fields[fieldPane],
				fields[fieldMessage],
				fields[fieldPaneCreated],
				fields[fieldLevel],
			); err != nil {
				return err
			}
			if err := hooks.Run("post-dismiss", envVars...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	updateTmuxStatusOption()
	return nil
}

// CleanupOldNotifications cleans up notifications older than the threshold.
func CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	if err := Init(); err != nil {
		return fmt.Errorf("storage not initialized: %w", err)
	}
	return WithLock(lockDir, func() error {
		return cleanupOld(daysThreshold, dryRun)
	})
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
	if err := Init(); err != nil {
		colors.Error(fmt.Sprintf("failed to initialize storage: %v", err))
		return 0
	}
	var count int
	err := WithLock(lockDir, func() error {
		latest, err := getLatestNotifications()
		if err != nil {
			return err
		}
		for _, line := range latest {
			fields := strings.Split(line, "\t")
			if len(fields) > fieldState && fields[fieldState] == "active" {
				count++
			}
		}
		return nil
	})
	if err != nil {
		colors.Error(fmt.Sprintf("failed to get active count: %v", err))
		return 0
	}
	return count
}

// Helper functions

func getNextID() (int, error) {
	latest, err := getLatestNotifications()
	if err != nil {
		return 0, err
	}
	maxID := 0
	for _, line := range latest {
		fields := strings.Split(line, "\t")
		if len(fields) <= fieldID {
			continue
		}
		id, err := strconv.Atoi(fields[fieldID])
		if err != nil {
			continue
		}
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1, nil
}

func escapeMessage(msg string) string {
	// Escape backslashes first
	msg = strings.ReplaceAll(msg, "\\", "\\\\")
	// Escape tabs
	msg = strings.ReplaceAll(msg, "\t", "\\t")
	// Escape newlines
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	return msg
}

func unescapeMessage(msg string) string {
	// Unescape newlines first
	msg = strings.ReplaceAll(msg, "\\n", "\n")
	// Unescape tabs
	msg = strings.ReplaceAll(msg, "\\t", "\t")
	// Unescape backslashes
	msg = strings.ReplaceAll(msg, "\\\\", "\\")
	return msg
}

func appendLine(id int, timestamp, state, session, window, pane, message, paneCreated, level string) error {
	line := fmt.Sprintf("%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		id, timestamp, state, session, window, pane, message, paneCreated, level)
	f, err := os.OpenFile(notificationsFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close file: %w", cerr)
		}
	}()

	if _, err = f.WriteString(line); err != nil {
		return fmt.Errorf("write line: %w", err)
	}

	if err = f.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}

	return nil
}

func readAllLines() ([]string, error) {
	data, err := os.ReadFile(notificationsFile)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	// Remove empty trailing line
	var result []string
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

func getLatestNotifications() ([]string, error) {
	lines, err := readAllLines()
	if err != nil {
		return nil, err
	}
	// Map from ID to latest line (last occurrence)
	latestMap := make(map[int]string)
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) <= fieldID {
			continue
		}
		id, err := strconv.Atoi(fields[fieldID])
		if err != nil {
			continue
		}
		latestMap[id] = line
	}
	// Convert to slice and sort by ID
	var ids []int
	for id := range latestMap {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var result []string
	for _, id := range ids {
		result = append(result, latestMap[id])
	}
	return result, nil
}

func filterNotifications(lines []string, stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) []string {
	var filtered []string
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) < numFields {
			// Pad missing fields
			for len(fields) < numFields {
				fields = append(fields, "")
			}
		}
		// State filter
		if stateFilter != "" && stateFilter != "all" {
			if fields[fieldState] != stateFilter {
				continue
			}
		}
		// Level filter
		if levelFilter != "" && fields[fieldLevel] != levelFilter {
			continue
		}
		// Session filter
		if sessionFilter != "" && fields[fieldSession] != sessionFilter {
			continue
		}
		// Window filter
		if windowFilter != "" && fields[fieldWindow] != windowFilter {
			continue
		}
		// Pane filter
		if paneFilter != "" && fields[fieldPane] != paneFilter {
			continue
		}
		// Older than cutoff
		if olderThanCutoff != "" && fields[fieldTimestamp] >= olderThanCutoff {
			continue
		}
		// Newer than cutoff
		if newerThanCutoff != "" && fields[fieldTimestamp] <= newerThanCutoff {
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered
}

func dismissByID(id string) error {
	latest, err := getLatestNotifications()
	if err != nil {
		return err
	}
	var targetLine string
	for _, line := range latest {
		fields := strings.Split(line, "\t")
		if len(fields) > fieldID && fields[fieldID] == id {
			targetLine = line
			break
		}
	}
	if targetLine == "" {
		return fmt.Errorf("notification %s not found", id)
	}
	fields := strings.Split(targetLine, "\t")
	if len(fields) < numFields {
		return fmt.Errorf("invalid line format")
	}
	// Ensure state is active
	if fields[fieldState] == "dismissed" {
		return fmt.Errorf("already dismissed")
	}
	idInt, err := strToInt(fields[fieldID])
	if err != nil {
		return fmt.Errorf("invalid ID %s: %w", fields[fieldID], err)
	}
	// Write new line with state dismissed, preserving other fields
	return appendLine(
		idInt,
		fields[fieldTimestamp],
		"dismissed",
		fields[fieldSession],
		fields[fieldWindow],
		fields[fieldPane],
		fields[fieldMessage],
		fields[fieldPaneCreated],
		fields[fieldLevel],
	)
}

func dismissAllActive() error {
	latest, err := getLatestNotifications()
	if err != nil {
		return err
	}
	for _, line := range latest {
		fields := strings.Split(line, "\t")
		if len(fields) <= fieldState {
			continue
		}
		if fields[fieldState] != "active" {
			continue
		}
		idInt, err := strToInt(fields[fieldID])
		if err != nil {
			return fmt.Errorf("invalid ID %s: %w", fields[fieldID], err)
		}
		// Write dismissed line
		err = appendLine(
			idInt,
			fields[fieldTimestamp],
			"dismissed",
			fields[fieldSession],
			fields[fieldWindow],
			fields[fieldPane],
			fields[fieldMessage],
			fields[fieldPaneCreated],
			fields[fieldLevel],
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func cleanupOld(daysThreshold int, dryRun bool) error {
	allDismissed := daysThreshold == 0
	cutoff := time.Now().UTC().AddDate(0, 0, -daysThreshold)
	cutoffStr := cutoff.Format("2006-01-02T15:04:05Z")

	if allDismissed {
		colors.Info("Cleaning up all dismissed notifications")
	} else {
		colors.Info(fmt.Sprintf("Cleaning up notifications dismissed before %s", cutoffStr))
	}

	// Run pre-cleanup hooks
	envVars := []string{
		fmt.Sprintf("CLEANUP_DAYS=%d", daysThreshold),
		fmt.Sprintf("CUTOFF_TIMESTAMP=%s", cutoffStr),
		fmt.Sprintf("DRY_RUN=%t", dryRun),
	}
	if err := hooks.Run("cleanup", envVars...); err != nil {
		return fmt.Errorf("pre-cleanup hook failed: %w", err)
	}

	// Get latest version of each notification
	latestLines, err := getLatestNotifications()
	if err != nil {
		return fmt.Errorf("failed to read notifications: %w", err)
	}

	// Collect IDs of dismissed notifications older than cutoff (or all dismissed if daysThreshold == 0)
	var idsToDelete []int
	for _, line := range latestLines {
		fields := strings.Split(line, "\t")
		if len(fields) <= fieldState {
			continue
		}
		if fields[fieldState] != "dismissed" {
			continue
		}
		if !allDismissed && fields[fieldTimestamp] >= cutoffStr {
			continue
		}
		id, err := strconv.Atoi(fields[fieldID])
		if err != nil {
			continue
		}
		idsToDelete = append(idsToDelete, id)
	}

	deletedCount := len(idsToDelete)
	if deletedCount == 0 {
		colors.Info("No old dismissed notifications to clean up")
		// Run post-cleanup hooks with zero count
		postEnv := append(envVars, "DELETED_COUNT=0")
		if err := hooks.Run("post-cleanup", postEnv...); err != nil {
			return fmt.Errorf("post-cleanup hook failed: %w", err)
		}
		return nil
	}

	colors.Info(fmt.Sprintf("Found %d notification(s) to clean up", deletedCount))

	if dryRun {
		colors.Info(fmt.Sprintf("Dry run: would delete notifications with IDs: %v", idsToDelete))
		// Run post-cleanup hooks with dry run
		postEnv := append(envVars, "DRY_RUN=true", fmt.Sprintf("DELETED_COUNT=%d", deletedCount))
		if err := hooks.Run("post-cleanup", postEnv...); err != nil {
			return fmt.Errorf("post-cleanup hook failed: %w", err)
		}
		return nil
	}

	// Filter out all lines whose ID is in idsToDelete
	lines, err := readAllLines()
	if err != nil {
		return fmt.Errorf("failed to read all lines: %w", err)
	}
	var filtered []string
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) <= fieldID {
			continue
		}
		id, err := strconv.Atoi(fields[fieldID])
		if err != nil {
			continue
		}
		keep := true
		for _, delID := range idsToDelete {
			if id == delID {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, line)
		}
	}
	// Write back filtered lines
	data := strings.Join(filtered, "\n")
	if len(filtered) > 0 {
		data += "\n"
	}
	if err := os.WriteFile(notificationsFile, []byte(data), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	colors.Info(fmt.Sprintf("Successfully cleaned up %d notification(s)", deletedCount))

	// Run post-cleanup hooks
	postEnv := append(envVars, fmt.Sprintf("DELETED_COUNT=%d", deletedCount))
	if err := hooks.Run("post-cleanup", postEnv...); err != nil {
		return fmt.Errorf("post-cleanup hook failed: %w", err)
	}
	return nil
}

func strToInt(s string) (int, error) {
	if strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("negative value not allowed: %s", s)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("convert string to int: %w", err)
	}
	return n, nil
}

// Reset resets the storage package state for testing.
func Reset() {
	initMu.Lock()
	defer initMu.Unlock()

	notificationsFile = ""
	lockDir = ""
	initialized = false
	initErr = nil

	// Reset sync.Once by creating a new one
	// This is safe because Reset() should only be called in tests
	initOnce = &sync.Once{}
}
