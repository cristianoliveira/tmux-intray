// Package storage provides file-based TSV storage with locking.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
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
	initialized       bool
)

// Init initializes storage directories and files.
func Init() {
	if initialized {
		return
	}
	// Load configuration
	config.Load()
	// Prefer environment variable directly (should match config.Load but ensure it works)
	stateDir := os.Getenv("TMUX_INTRAY_STATE_DIR")
	if stateDir == "" {
		stateDir = config.Get("state_dir", "")
	}
	colors.Debug("state_dir: " + stateDir)
	if stateDir == "" {
		colors.Error("state_dir not configured")
		return
	}
	notificationsFile = filepath.Join(stateDir, "notifications.tsv")
	lockDir = filepath.Join(stateDir, "lock")

	// Ensure directories exist
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		colors.Error(fmt.Sprintf("failed to create state directory: %v", err))
		return
	}

	// Ensure notifications file exists
	f, err := os.OpenFile(notificationsFile, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		colors.Error(fmt.Sprintf("failed to create notifications file: %v", err))
		return
	}
	f.Close()

	initialized = true
	colors.Debug("storage initialized")
}

// AddNotification adds a notification and returns its ID.
func AddNotification(message, timestamp, session, window, pane, paneCreated, level string) string {
	Init()
	if !initialized {
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
	// Append line with lock
	err = WithLock(lockDir, func() error {
		return appendLine(id, timestamp, "active", session, window, pane, escapedMessage, paneCreated, level)
	})
	if err != nil {
		colors.Error(fmt.Sprintf("failed to add notification: %v", err))
		return ""
	}
	// Update tmux status (not implemented yet)
	// Return ID as string
	return strconv.Itoa(id)
}

// ListNotifications returns TSV lines for notifications.
func ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) string {
	Init()
	if !initialized {
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

// DismissNotification dismisses a notification by ID.
func DismissNotification(id string) {
	Init()
	if !initialized {
		return
	}
	err := WithLock(lockDir, func() error {
		return dismissByID(id)
	})
	if err != nil {
		colors.Error(fmt.Sprintf("failed to dismiss notification %s: %v", id, err))
	}
}

// DismissAll dismisses all active notifications.
func DismissAll() {
	Init()
	if !initialized {
		return
	}
	err := WithLock(lockDir, func() error {
		return dismissAllActive()
	})
	if err != nil {
		colors.Error(fmt.Sprintf("failed to dismiss all notifications: %v", err))
	}
}

// CleanupOldNotifications cleans up notifications older than the threshold.
func CleanupOldNotifications(daysThreshold int, dryRun bool) {
	Init()
	if !initialized {
		return
	}
	err := WithLock(lockDir, func() error {
		return cleanupOld(daysThreshold, dryRun)
	})
	if err != nil {
		colors.Error(fmt.Sprintf("cleanup failed: %v", err))
	}
}

// GetActiveCount returns the active notification count.
func GetActiveCount() int {
	Init()
	if !initialized {
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
	defer f.Close()
	_, err = f.WriteString(line)
	return err
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
	// Write new line with state dismissed, preserving other fields
	return appendLine(
		strToInt(fields[fieldID]),
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
		// Write dismissed line
		err = appendLine(
			strToInt(fields[fieldID]),
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
	cutoff := time.Now().UTC().AddDate(0, 0, -daysThreshold)
	cutoffStr := cutoff.Format("2006-01-02T15:04:05Z")
	lines, err := readAllLines()
	if err != nil {
		return err
	}
	// Collect IDs of dismissed notifications older than cutoff
	idSet := make(map[int]bool)
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) <= fieldState {
			continue
		}
		if fields[fieldState] != "dismissed" {
			continue
		}
		if fields[fieldTimestamp] >= cutoffStr {
			continue
		}
		id, err := strconv.Atoi(fields[fieldID])
		if err != nil {
			continue
		}
		idSet[id] = true
	}
	if len(idSet) == 0 {
		colors.Info("no old dismissed notifications to clean up")
		return nil
	}
	if dryRun {
		var ids []int
		for id := range idSet {
			ids = append(ids, id)
		}
		colors.Info(fmt.Sprintf("dry run: would delete notifications with IDs: %v", ids))
		return nil
	}
	// Filter out lines whose ID is in idSet
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
		if !idSet[id] {
			filtered = append(filtered, line)
		}
	}
	// Write back filtered lines
	data := strings.Join(filtered, "\n")
	if len(filtered) > 0 {
		data += "\n"
	}
	err = os.WriteFile(notificationsFile, []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	colors.Info(fmt.Sprintf("cleaned up %d notification(s)", len(idSet)))
	return nil
}

func strToInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
