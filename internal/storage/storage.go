// Package storage provides file-based TSV storage with locking.
package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/hooks"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
)

// File permission constants
const (
	// FileModeDir is the permission for directories (rwxr-xr-x)
	// Owner: read/write/execute, Group/others: read/execute
	FileModeDir os.FileMode = 0755
	// FileModeFile is the permission for data files (rw-r--r--)
	// Owner: read/write, Group/others: read only
	FileModeFile os.FileMode = 0644
	// FileModeScript is the permission for executable scripts (rwxr-xr-x)
	// Owner: read/write/execute, Group/others: read/execute
	FileModeScript os.FileMode = 0755

	// FileExtTSV is the file extension for TSV (Tab-Separated Values) files.
	// Used for notifications storage.
	FileExtTSV = ".tsv"
)

// Valid notification levels
var (
	validLevels = map[string]bool{
		"info":     true,
		"warning":  true,
		"error":    true,
		"critical": true,
	}

	// Valid notification states
	validStates = map[string]bool{
		"active":    true,
		"dismissed": true,
		"all":       true,
	}
)

// Custom error types for storage operations.
var (
	// ErrStorageNotInitialized is returned when storage operations are called before Init().
	ErrStorageNotInitialized = errors.New("storage not initialized")

	// ErrInvalidNotificationID is returned when a notification ID is invalid (empty or malformed).
	ErrInvalidNotificationID = errors.New("invalid notification ID")

	// ErrInvalidTSVFormat is returned when a notification line has an invalid TSV format.
	ErrInvalidTSVFormat = errors.New("invalid TSV format")

	// ErrNotificationNotFound is returned when a notification ID cannot be found.
	ErrNotificationNotFound = errors.New("notification not found")

	// ErrNotificationAlreadyDismissed is returned when attempting to dismiss an already-dismissed notification.
	ErrNotificationAlreadyDismissed = errors.New("notification already dismissed")
)

var (
	notificationsFile string
	lockDir           string
	stateDir          string
	initOnce          = &sync.Once{}
	initialized       bool
	initMu            sync.RWMutex
	initErr           error
	tmuxClient        tmux.TmuxClient = tmux.NewDefaultClient()
)

// Init initializes storage directories and files.
// Returns an error if initialization fails. Safe for concurrent calls.
func Init() error {
	var err error
	initOnce.Do(func() {
		// Load configuration
		config.Load()

		// Prefer environment variable directly (should match config.Load but ensure it works)
		stateDir = os.Getenv("TMUX_INTRAY_STATE_DIR")
		if stateDir == "" {
			stateDir = config.Get("state_dir", "")
		}
		colors.Debug("state_dir: " + stateDir)
		if stateDir == "" {
			err = fmt.Errorf("storage initialization failed: TMUX_INTRAY_STATE_DIR not configured")
			return
		}
		notificationsFile = filepath.Join(stateDir, "notifications"+FileExtTSV)
		lockDir = filepath.Join(stateDir, "lock")

		// Ensure directories exist
		if err = os.MkdirAll(stateDir, FileModeDir); err != nil {
			err = fmt.Errorf("failed to create state directory: %w", err)
			return
		}

		// Ensure notifications file exists
		var f *os.File
		f, err = os.OpenFile(notificationsFile, os.O_RDONLY|os.O_CREATE, FileModeFile)
		if err != nil {
			err = fmt.Errorf("failed to create notifications file: %w", err)
			return
		}
		if cerr := f.Close(); cerr != nil {
			err = fmt.Errorf("failed to close notifications file: %w", cerr)
			return
		}

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

// SetTmuxClient sets the tmux client for the storage package.
// This is primarily used for testing with mock implementations.
// Preconditions: client must be non-nil.
func SetTmuxClient(client tmux.TmuxClient) {
	tmuxClient = client
}

// validateNotificationInputs validates all parameters for AddNotification.
// Returns an error if validation fails, nil otherwise.
func validateNotificationInputs(message, timestamp, session, window, pane, paneCreated, level string) error {
	// Validate message is non-empty
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("validation error: message cannot be empty")
	}

	// Validate level (must be non-empty and one of valid levels)
	if level == "" {
		return fmt.Errorf("validation error: level cannot be empty")
	}
	if !validLevels[level] {
		return fmt.Errorf("validation error: invalid level '%s', must be one of: info, warning, error, critical", level)
	}

	// Validate timestamp format if provided
	if timestamp != "" {
		// Try to parse timestamp with RFC3339 format
		_, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return fmt.Errorf("validation error: invalid timestamp format '%s', expected RFC3339 format (e.g., 2006-01-02T15:04:05Z or 2006-01-02T15:04:05.123Z)", timestamp)
		}
	}

	// Validate session, window, pane are non-empty if provided (not just whitespace)
	// These are optional fields, but if provided they should contain actual content
	if session != "" && strings.TrimSpace(session) == "" {
		return fmt.Errorf("validation error: session cannot be whitespace only")
	}
	if window != "" && strings.TrimSpace(window) == "" {
		return fmt.Errorf("validation error: window cannot be whitespace only")
	}
	if pane != "" && strings.TrimSpace(pane) == "" {
		return fmt.Errorf("validation error: pane cannot be whitespace only")
	}

	return nil
}

// validateListInputs validates all parameters for ListNotifications.
// Empty string filters are ignored (except stateFilter which defaults to "all").
// Returns an error if validation fails, nil otherwise.
func validateListInputs(stateFilter, levelFilter, olderThanCutoff, newerThanCutoff string) error {
	// Validate state filter (if provided)
	// Valid values: "active", "dismissed", "all", or "" (defaults to "all" in filtering)
	if stateFilter != "" && !validStates[stateFilter] {
		return fmt.Errorf("invalid state '%s', must be one of: active, dismissed, all, or empty", stateFilter)
	}

	// Validate level filter (if provided)
	// Valid values: "info", "warning", "error", "critical", or "" (no filter)
	if levelFilter != "" && !validLevels[levelFilter] {
		return fmt.Errorf("invalid level '%s', must be one of: info, warning, error, critical, or empty", levelFilter)
	}

	// Validate olderThanCutoff timestamp format if provided
	if olderThanCutoff != "" {
		_, err := time.Parse(time.RFC3339, olderThanCutoff)
		if err != nil {
			return fmt.Errorf("invalid olderThanCutoff format '%s', expected RFC3339 format (e.g., 2006-01-02T15:04:05Z)", olderThanCutoff)
		}
	}

	// Validate newerThanCutoff timestamp format if provided
	if newerThanCutoff != "" {
		_, err := time.Parse(time.RFC3339, newerThanCutoff)
		if err != nil {
			return fmt.Errorf("invalid newerThanCutoff format '%s', expected RFC3339 format (e.g., 2006-01-02T15:04:05Z)", newerThanCutoff)
		}
	}

	return nil
}

// AddNotification adds a notification and returns its ID.
// Returns an error if validation fails or initialization fails.
// Preconditions: message must be non-empty; level must be one of "info", "warning", "error", or "critical";
// timestamp must be RFC3339 format if provided.
func AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	// Validate inputs first (Fail-Fast)
	if err := validateNotificationInputs(message, timestamp, session, window, pane, paneCreated, level); err != nil {
		return "", err
	}

	// Initialize storage
	if err := Init(); err != nil {
		return "", fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Generate ID
	id, err := getNextID()
	if err != nil {
		return "", fmt.Errorf("failed to generate id: %w", err)
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
		return "", fmt.Errorf("pre-add hook aborted: %w", err)
	}

	// Append line with lock
	if err := WithLock(lockDir, func() error {
		return appendLine(id, timestamp, "active", session, window, pane, escapedMessage, paneCreated, level, "")
	}); err != nil {
		colors.Error(fmt.Sprintf("failed to add notification: %v", err))
		return "", fmt.Errorf("failed to add notification: %w", err)
	}

	// Update tmux status option outside lock to avoid deadlock (updateTmuxStatusOption also acquires a lock) and keep lock duration short
	// Calculate active count after adding (this notification is now active)
	activeCount := 0
	latest, err2 := getLatestNotifications()
	if err2 == nil {
		for _, line := range latest {
			fields := strings.Split(line, "\t")
			state, err := getField(fields, fieldState)
			if err == nil && state == "active" {
				activeCount++
			}
		}
	}
	if err := updateTmuxStatusOption(activeCount); err != nil {
		colors.Error(fmt.Sprintf("failed to update tmux status: %v", err))
	}

	// Run post-add hooks
	if err := hooks.Run("post-add", envVars...); err != nil {
		colors.Error(fmt.Sprintf("post-add hook failed: %v", err))
		// Return error because post-processing failed in abort mode
		// The notification was added but post-add hooks are critical for cleanup/state
		return strconv.Itoa(id), fmt.Errorf("post-add hook failed: %w", err)
	}

	// Return ID as string
	return strconv.Itoa(id), nil
}

// ListNotifications returns TSV lines for notifications matching the specified filters.
// Filters that are empty strings are ignored (except stateFilter which defaults to "all").
// Valid state values: "active", "dismissed", "all", or "" (defaults to "all")
// Valid level values: "info", "warning", "error", "critical", or "" (no filter)
// Valid timestamp formats for olderThanCutoff and newerThanCutoff: RFC3339 (e.g., "2006-01-02T15:04:05Z")
// Valid readFilter values: "read" (has read_timestamp), "unread" (no read_timestamp), or "" (no filter)
// Returns TSV lines as a string and an error if validation fails.
// Preconditions: if stateFilter is non-empty, it must be one of "active", "dismissed", or "all";
// if levelFilter is non-empty, it must be one of "info", "warning", "error", or "critical";
// if olderThanCutoff or newerThanCutoff are non-empty, they must be RFC3339 format.
func ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	// Validate inputs first (Fail-Fast)
	if err := validateListInputs(stateFilter, levelFilter, olderThanCutoff, newerThanCutoff); err != nil {
		return "", err
	}

	if err := Init(); err != nil {
		colors.Error(fmt.Sprintf("failed to initialize storage: %v", err))
		return "", fmt.Errorf("failed to initialize storage: %w", err)
	}
	var lines []string
	err := WithLock(lockDir, func() error {
		latest, err := getLatestNotifications()
		if err != nil {
			return err
		}
		// Apply filters
		filtered := filterNotifications(latest, stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter)
		lines = filtered
		return nil
	})
	if err != nil {
		colors.Error(fmt.Sprintf("failed to list notifications: %v", err))
		return "", fmt.Errorf("failed to list notifications: %w", err)
	}
	return strings.Join(lines, "\n"), nil
}

// GetNotificationByID retrieves a single notification by its ID.
// This is an optimized version that avoids reading all notifications when possible.
// Returns the notification line as a TSV string or an error if not found.
// Preconditions: id must be non-empty.
func GetNotificationByID(id string) (string, error) {
	if err := Init(); err != nil {
		return "", fmt.Errorf("get notification by id: %w", err)
	}

	// Validate ID format
	if id == "" {
		return "", fmt.Errorf("get notification by id: %w", ErrInvalidNotificationID)
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
		return "", fmt.Errorf("get notification by id: %w: id %s", ErrNotificationNotFound, id)
	}

	return result, nil
}

// DismissNotification dismisses a notification by ID.
// Preconditions: id must be non-empty and reference an existing notification.
func DismissNotification(id string) error {
	if err := Init(); err != nil {
		return fmt.Errorf("dismiss notification: %w", err)
	}
	colors.Debug("DismissNotification called for ID:", id)
	err := WithLock(lockDir, func() error {
		latest, err := getLatestNotifications()
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to read notifications: %w", err)
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
			return fmt.Errorf("dismiss notification: %w: id %s", ErrNotificationNotFound, id)
		}
		fields := strings.Split(targetLine, "\t")
		fields, err = normalizeFields(fields)
		if err != nil {
			return fmt.Errorf("dismiss notification: %w: %s", ErrInvalidTSVFormat, err)
		}
		state, err := getField(fields, fieldState)
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to get state field: %w", err)
		}
		if state == "dismissed" {
			return fmt.Errorf("dismiss notification: %w: id %s", ErrNotificationAlreadyDismissed, id)
		}
		level, err := getField(fields, fieldLevel)
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to get level field: %w", err)
		}
		message, err := getField(fields, fieldMessage)
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to get message field: %w", err)
		}
		timestamp, err := getField(fields, fieldTimestamp)
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to get timestamp field: %w", err)
		}
		session, err := getField(fields, fieldSession)
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to get session field: %w", err)
		}
		window, err := getField(fields, fieldWindow)
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to get window field: %w", err)
		}
		pane, err := getField(fields, fieldPane)
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to get pane field: %w", err)
		}
		paneCreated, err := getField(fields, fieldPaneCreated)
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to get pane created field: %w", err)
		}
		readTimestamp, err := getField(fields, fieldReadTimestamp)
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to get read timestamp field: %w", err)
		}
		idField, err := getField(fields, fieldID)
		if err != nil {
			return fmt.Errorf("dismiss notification: failed to get id field: %w", err)
		}
		envVars := []string{
			fmt.Sprintf("NOTIFICATION_ID=%s", id),
			fmt.Sprintf("LEVEL=%s", level),
			fmt.Sprintf("MESSAGE=%s", message),
			fmt.Sprintf("ESCAPED_MESSAGE=%s", message),
			fmt.Sprintf("TIMESTAMP=%s", timestamp),
			fmt.Sprintf("SESSION=%s", session),
			fmt.Sprintf("WINDOW=%s", window),
			fmt.Sprintf("PANE=%s", pane),
			fmt.Sprintf("PANE_CREATED=%s", paneCreated),
		}
		if err := hooks.Run("pre-dismiss", envVars...); err != nil {
			return err
		}
		idInt, err := strToInt(idField)
		if err != nil {
			return fmt.Errorf("invalid id %s: %w", idField, err)
		}
		if err := appendLine(
			idInt,
			timestamp,
			"dismissed",
			session,
			window,
			pane,
			message,
			paneCreated,
			level,
			readTimestamp,
		); err != nil {
			return err
		}
		if err := hooks.Run("post-dismiss", envVars...); err != nil {
			return err
		}
		// Calculate active count after dismissing
		activeCount := 0
		latest, err2 := getLatestNotifications()
		if err2 == nil {
			for _, line := range latest {
				fields := strings.Split(line, "\t")
				if len(fields) > fieldState && fields[fieldState] == "active" {
					activeCount++
				}
			}
		}
		if err := updateTmuxStatusOption(activeCount); err != nil {
			colors.Error(fmt.Sprintf("failed to update tmux status: %v", err))
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// MarkNotificationRead marks a notification as read by setting read_timestamp.
// Preconditions: id must be non-empty and reference an existing notification.
func MarkNotificationRead(id string) error {
	return markNotificationReadState(id, time.Now().UTC().Format(time.RFC3339))
}

// MarkNotificationUnread marks a notification as unread by clearing read_timestamp.
// Preconditions: id must be non-empty and reference an existing notification.
func MarkNotificationUnread(id string) error {
	return markNotificationReadState(id, "")
}

// MarkNotificationReadWithTimestamp marks a notification as read by setting read_timestamp to the provided timestamp.
// Preconditions: id must be non-empty and reference an existing notification; timestamp must be RFC3339 format.
func MarkNotificationReadWithTimestamp(id, timestamp string) error {
	return markNotificationReadState(id, timestamp)
}

// MarkNotificationUnreadWithTimestamp marks a notification as unread by clearing read_timestamp.
// Preconditions: id must be non-empty and reference an existing notification.
func MarkNotificationUnreadWithTimestamp(id, timestamp string) error {
	return markNotificationReadState(id, timestamp)
}

func markNotificationReadState(id, readTimestamp string) error {
	if err := Init(); err != nil {
		return fmt.Errorf("markNotificationReadState: %w", err)
	}
	return WithLock(lockDir, func() error {
		latest, err := getLatestNotifications()
		if err != nil {
			return fmt.Errorf("markNotificationReadState: failed to read notifications: %w", err)
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
			return fmt.Errorf("markNotificationReadState: %w: id %s", ErrNotificationNotFound, id)
		}
		fields := strings.Split(targetLine, "\t")
		fields, err = normalizeFields(fields)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: %w: %s", ErrInvalidTSVFormat, err)
		}
		state, err := getField(fields, fieldState)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: failed to get state field: %w", err)
		}
		timestamp, err := getField(fields, fieldTimestamp)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: failed to get timestamp field: %w", err)
		}
		session, err := getField(fields, fieldSession)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: failed to get session field: %w", err)
		}
		window, err := getField(fields, fieldWindow)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: failed to get window field: %w", err)
		}
		pane, err := getField(fields, fieldPane)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: failed to get pane field: %w", err)
		}
		message, err := getField(fields, fieldMessage)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: failed to get message field: %w", err)
		}
		paneCreated, err := getField(fields, fieldPaneCreated)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: failed to get pane created field: %w", err)
		}
		level, err := getField(fields, fieldLevel)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: failed to get level field: %w", err)
		}
		idField, err := getField(fields, fieldID)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: failed to get id field: %w", err)
		}
		idInt, err := strToInt(idField)
		if err != nil {
			return fmt.Errorf("markNotificationReadState: invalid id %s: %w", idField, err)
		}
		return appendLine(
			idInt,
			timestamp,
			state,
			session,
			window,
			pane,
			message,
			paneCreated,
			level,
			readTimestamp,
		)
	})
}

// DismissAll dismisses all active notifications.
func DismissAll() error {
	if err := Init(); err != nil {
		return fmt.Errorf("dismiss all: %w", err)
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
			if err := dismissOneNotification(line); err != nil {
				return err
			}
		}
		// Calculate active count after dismissing all
		activeCount := 0
		latestAfter, err := getLatestNotifications()
		if err == nil {
			activeCount = countActiveNotifications(latestAfter)
		}
		if err := updateTmuxStatusOption(activeCount); err != nil {
			colors.Error(fmt.Sprintf("failed to update tmux status: %v", err))
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// CleanupOldNotifications cleans up notifications older than the threshold.
// Preconditions: daysThreshold must be >= 0; if 0, all dismissed notifications are cleaned up.
func CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	if err := Init(); err != nil {
		return fmt.Errorf("cleanup old notifications: %w", err)
	}
	return WithLock(lockDir, func() error {
		return cleanupOld(daysThreshold, dryRun)
	})
}

// updateTmuxStatusOption updates the tmux status option with the given active count.
func updateTmuxStatusOption(count int) error {
	// Only update if tmux is running
	running, err := tmuxClient.HasSession()
	if err != nil {
		return fmt.Errorf("updateTmuxStatusOption: tmux not available: %w", err)
	}
	if !running {
		return fmt.Errorf("updateTmuxStatusOption: tmux not running")
	}
	if err := tmuxClient.SetStatusOption("@tmux_intray_active_count", fmt.Sprintf("%d", count)); err != nil {
		return fmt.Errorf("updateTmuxStatusOption: failed to set @tmux_intray_active_count to %d: %w", count, err)
	}
	return nil
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
			state, err := getField(fields, fieldState)
			if err == nil && state == "active" {
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

// NormalizeFields ensures a TSV line has the correct number of fields.
// Pads with empty strings if fewer than expected, returns error if below minimum.
func NormalizeFields(fields []string) ([]string, error) {
	if len(fields) < minFields {
		return nil, fmt.Errorf("expected at least %d fields, got %d", minFields, len(fields))
	}
	if len(fields) < numFields {
		for len(fields) < numFields {
			fields = append(fields, "")
		}
	}
	return fields, nil
}

// normalizeFields is the internal version for backward compatibility within the package.
func normalizeFields(fields []string) ([]string, error) {
	return NormalizeFields(fields)
}

// getField safely retrieves a field from a TSV line with bounds checking.
// Returns an error if the field index is out of bounds or fields is nil.
func getField(fields []string, index int) (string, error) {
	if fields == nil {
		return "", fmt.Errorf("fields array is nil")
	}
	if index < 0 || index >= len(fields) {
		return "", fmt.Errorf("field index %d out of bounds (len=%d)", index, len(fields))
	}
	return fields[index], nil
}

// parseLineID parses the ID from a TSV line.
// Returns the parsed ID and a success flag. Returns (0, false) on parse failure.
func parseLineID(line string) (int, bool) {
	fields := strings.Split(line, "\t")
	if len(fields) <= fieldID {
		return 0, false
	}
	id, err := strconv.Atoi(fields[fieldID])
	if err != nil {
		return 0, false
	}
	return id, true
}

// findMaxID finds the maximum ID among TSV lines.
// Returns 0 if no valid IDs are found.
func findMaxID(lines []string) int {
	maxID := 0
	for _, line := range lines {
		id, ok := parseLineID(line)
		if ok && id > maxID {
			maxID = id
		}
	}
	return maxID
}

// verifyNewID verifies that newID is greater than all existing IDs in the lines.
// Logs assertion failures if newID is not strictly greater than existing IDs.
func verifyNewID(newID int, lines []string) {
	for _, line := range lines {
		id, ok := parseLineID(line)
		if ok && newID <= id {
			colors.Debug(fmt.Sprintf("ASSERTION FAILED: getNextID returned ID %d which is not greater than existing ID %d", newID, id))
		}
	}
}

// getNextID generates the next unique notification ID.
// Invariants:
//   - Returned ID must always be > 0
//   - Returned ID must be strictly greater than all existing IDs in storage
//   - IDs are monotonically increasing across calls
func getNextID() (int, error) {
	latest, err := getLatestNotifications()
	if err != nil {
		return 0, err
	}
	maxID := findMaxID(latest)
	newID := maxID + 1

	// Assertions: verify invariants
	if newID <= 0 {
		colors.Debug(fmt.Sprintf("ASSERTION FAILED: getNextID returned ID <= 0: %d", newID))
	} else {
		colors.Debug(fmt.Sprintf("getNextID assertion passed: ID > 0 (got %d)", newID))
	}

	// Verify ID is strictly greater than all existing IDs
	verifyNewID(newID, latest)
	if len(latest) > 0 {
		colors.Debug(fmt.Sprintf("getNextID monotonic increase assertion passed: ID %d > max existing ID %d", newID, maxID))
	}

	return newID, nil
}

// EscapeMessage escapes special characters in a message for TSV storage.
// Escapes backslashes, tabs, and newlines to preserve message formatting.
func EscapeMessage(msg string) string {
	// Escape backslashes first
	msg = strings.ReplaceAll(msg, "\\", "\\\\")
	// Escape tabs
	msg = strings.ReplaceAll(msg, "\t", "\\t")
	// Escape newlines
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	return msg
}

// UnescapeMessage unescapes special characters from a TSV-stored message.
// Restores newlines, tabs, and backslashes to their original values.
func UnescapeMessage(msg string) string {
	// Unescape newlines first
	msg = strings.ReplaceAll(msg, "\\n", "\n")
	// Unescape tabs
	msg = strings.ReplaceAll(msg, "\\t", "\t")
	// Unescape backslashes
	msg = strings.ReplaceAll(msg, "\\\\", "\\")
	return msg
}

// escapeMessage is the internal version for backward compatibility within the package.
func escapeMessage(msg string) string {
	return EscapeMessage(msg)
}

// unescapeMessage is the internal version for backward compatibility within the package.
func unescapeMessage(msg string) string {
	return UnescapeMessage(msg)
}

func appendLine(id int, timestamp, state, session, window, pane, message, paneCreated, level, readTimestamp string) error {
	line := fmt.Sprintf("%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		id, timestamp, state, session, window, pane, message, paneCreated, level, readTimestamp)
	f, err := os.OpenFile(notificationsFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, FileModeFile)
	if err != nil {
		return fmt.Errorf("appendLine: failed to open notifications file %s: %w", notificationsFile, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("appendLine: failed to close notifications file %s: %w", notificationsFile, cerr)
		}
	}()

	if _, err = f.WriteString(line); err != nil {
		return fmt.Errorf("appendLine: failed to write to notifications file %s: %w", notificationsFile, err)
	}

	if err = f.Sync(); err != nil {
		return fmt.Errorf("appendLine: failed to sync notifications file %s: %w", notificationsFile, err)
	}

	return nil
}

func readAllLines() ([]string, error) {
	data, err := os.ReadFile(notificationsFile)
	if err != nil {
		return nil, fmt.Errorf("readAllLines: failed to read notifications file %s: %w", notificationsFile, err)
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
		idField, err := getField(fields, fieldID)
		if err != nil {
			continue
		}
		id, err := strconv.Atoi(idField)
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

// predicate is a function that returns true if fields satisfy a condition.
type predicate func(fields []string) bool

// makeStatePredicate returns a predicate that filters by state.
// If stateFilter is empty or "all", returns nil (no predicate).
func makeStatePredicate(stateFilter string) predicate {
	if stateFilter == "" || stateFilter == "all" {
		return nil
	}
	return func(fields []string) bool {
		state, err := getField(fields, fieldState)
		if err != nil {
			return false
		}
		return state == stateFilter
	}
}

// makeExactMatchPredicate returns a predicate that filters by exact match of a field.
// If filterValue is empty, returns nil.
func makeExactMatchPredicate(fieldIndex int, filterValue string) predicate {
	if filterValue == "" {
		return nil
	}
	return func(fields []string) bool {
		value, err := getField(fields, fieldIndex)
		if err != nil {
			return false
		}
		return value == filterValue
	}
}

// makeReadStatusPredicate returns a predicate that filters by read status.
// Valid readFilter values: "read" (has read_timestamp), "unread" (no read_timestamp), or "" (no filter).
// If readFilter is empty, returns nil.
func makeReadStatusPredicate(readFilter string) predicate {
	if readFilter == "" {
		return nil
	}
	return func(fields []string) bool {
		readTimestamp, err := getField(fields, fieldReadTimestamp)
		if err != nil {
			return false
		}
		if readFilter == "read" {
			return readTimestamp != ""
		}
		if readFilter == "unread" {
			return readTimestamp == ""
		}
		// Invalid readFilter value (should not happen), treat as no filter
		return true
	}
}

// makeOlderThanPredicate returns a predicate that filters timestamps older than cutoff.
// If cutoff is empty, returns nil.
func makeOlderThanPredicate(cutoff string) predicate {
	if cutoff == "" {
		return nil
	}
	return func(fields []string) bool {
		timestamp, err := getField(fields, fieldTimestamp)
		if err != nil {
			return false
		}
		return timestamp < cutoff
	}
}

// makeNewerThanPredicate returns a predicate that filters timestamps newer than cutoff.
// If cutoff is empty, returns nil.
func makeNewerThanPredicate(cutoff string) predicate {
	if cutoff == "" {
		return nil
	}
	return func(fields []string) bool {
		timestamp, err := getField(fields, fieldTimestamp)
		if err != nil {
			return false
		}
		return timestamp > cutoff
	}
}

func filterNotifications(lines []string, stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) []string {
	var filtered []string
	// Build predicates for non-empty filters
	var predicates []predicate
	if p := makeStatePredicate(stateFilter); p != nil {
		predicates = append(predicates, p)
	}
	if p := makeExactMatchPredicate(fieldLevel, levelFilter); p != nil {
		predicates = append(predicates, p)
	}
	if p := makeExactMatchPredicate(fieldSession, sessionFilter); p != nil {
		predicates = append(predicates, p)
	}
	if p := makeExactMatchPredicate(fieldWindow, windowFilter); p != nil {
		predicates = append(predicates, p)
	}
	if p := makeExactMatchPredicate(fieldPane, paneFilter); p != nil {
		predicates = append(predicates, p)
	}
	if p := makeReadStatusPredicate(readFilter); p != nil {
		predicates = append(predicates, p)
	}
	if p := makeOlderThanPredicate(olderThanCutoff); p != nil {
		predicates = append(predicates, p)
	}
	if p := makeNewerThanPredicate(newerThanCutoff); p != nil {
		predicates = append(predicates, p)
	}
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) < numFields {
			// Pad missing fields
			for len(fields) < numFields {
				fields = append(fields, "")
			}
		}
		keep := true
		for _, p := range predicates {
			if !p(fields) {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, line)
		}
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
		return fmt.Errorf("dismissByID: %w: id %s", ErrNotificationNotFound, id)
	}
	fields := strings.Split(targetLine, "\t")
	fields, err = normalizeFields(fields)
	if err != nil {
		return fmt.Errorf("dismissByID: %w: %s", ErrInvalidTSVFormat, err)
	}
	// Ensure state is active
	state, err := getField(fields, fieldState)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to get state field: %w", err)
	}
	if state == "dismissed" {
		return fmt.Errorf("dismissByID: %w: id %s", ErrNotificationAlreadyDismissed, id)
	}
	idField, err := getField(fields, fieldID)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to get id field: %w", err)
	}
	timestamp, err := getField(fields, fieldTimestamp)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to get timestamp field: %w", err)
	}
	session, err := getField(fields, fieldSession)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to get session field: %w", err)
	}
	window, err := getField(fields, fieldWindow)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to get window field: %w", err)
	}
	pane, err := getField(fields, fieldPane)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to get pane field: %w", err)
	}
	message, err := getField(fields, fieldMessage)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to get message field: %w", err)
	}
	paneCreated, err := getField(fields, fieldPaneCreated)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to get pane created field: %w", err)
	}
	level, err := getField(fields, fieldLevel)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to get level field: %w", err)
	}
	readTimestamp, err := getField(fields, fieldReadTimestamp)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to get read timestamp field: %w", err)
	}
	idInt, err := strToInt(idField)
	if err != nil {
		return fmt.Errorf("dismissByID: failed to parse id field '%s': %w", idField, err)
	}
	// Write new line with state dismissed, preserving other fields
	return appendLine(
		idInt,
		timestamp,
		"dismissed",
		session,
		window,
		pane,
		message,
		paneCreated,
		level,
		readTimestamp,
	)
}

func dismissAllActive() error {
	latest, err := getLatestNotifications()
	if err != nil {
		return err
	}
	for _, line := range latest {
		fields := strings.Split(line, "\t")
		fields, err = normalizeFields(fields)
		if err != nil {
			continue
		}
		state, err := getField(fields, fieldState)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to get state field: %w", err)
		}
		if state != "active" {
			continue
		}
		idField, err := getField(fields, fieldID)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to get id field: %w", err)
		}
		timestamp, err := getField(fields, fieldTimestamp)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to get timestamp field: %w", err)
		}
		session, err := getField(fields, fieldSession)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to get session field: %w", err)
		}
		window, err := getField(fields, fieldWindow)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to get window field: %w", err)
		}
		pane, err := getField(fields, fieldPane)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to get pane field: %w", err)
		}
		message, err := getField(fields, fieldMessage)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to get message field: %w", err)
		}
		paneCreated, err := getField(fields, fieldPaneCreated)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to get pane created field: %w", err)
		}
		level, err := getField(fields, fieldLevel)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to get level field: %w", err)
		}
		readTimestamp, err := getField(fields, fieldReadTimestamp)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to get read timestamp field: %w", err)
		}
		idInt, err := strToInt(idField)
		if err != nil {
			return fmt.Errorf("dismissAllActive: failed to parse id field '%s': %w", idField, err)
		}
		// Write dismissed line
		err = appendLine(
			idInt,
			timestamp,
			"dismissed",
			session,
			window,
			pane,
			message,
			paneCreated,
			level,
			readTimestamp,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// buildHookEnvVars builds environment variable array for hooks.
func buildHookEnvVars(id, level, message, timestamp, session, window, pane, paneCreated, readTimestamp string) []string {
	return []string{
		fmt.Sprintf("NOTIFICATION_ID=%s", id),
		fmt.Sprintf("LEVEL=%s", level),
		fmt.Sprintf("MESSAGE=%s", message),
		fmt.Sprintf("ESCAPED_MESSAGE=%s", message),
		fmt.Sprintf("TIMESTAMP=%s", timestamp),
		fmt.Sprintf("SESSION=%s", session),
		fmt.Sprintf("WINDOW=%s", window),
		fmt.Sprintf("PANE=%s", pane),
		fmt.Sprintf("PANE_CREATED=%s", paneCreated),
	}
}

// countActiveNotifications counts active notifications from TSV lines.
func countActiveNotifications(latestLines []string) int {
	activeCount := 0
	for _, line := range latestLines {
		fields := strings.Split(line, "\t")
		state, err := getField(fields, fieldState)
		if err != nil {
			continue
		}
		if state == "active" {
			activeCount++
		}
	}
	return activeCount
}

// parseDismissalFields parses all required fields from a dismissal line at once.
// Returns the parsed fields or an error if any field cannot be extracted.
func parseDismissalFields(line string) (id, level, message, timestamp, session, window, pane, paneCreated, readTimestamp string, err error) {
	fields := strings.Split(line, "\t")
	if len(fields) < numFields {
		for len(fields) < numFields {
			fields = append(fields, "")
		}
	}

	id, err = getField(fields, fieldID)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", fmt.Errorf("dismissOneNotification: failed to get id field: %w", err)
	}

	state, err := getField(fields, fieldState)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", fmt.Errorf("dismissOneNotification: failed to get state field: %w", err)
	}

	if state != "active" {
		// Not active, nothing to dismiss - return empty strings with nil error
		return "", "", "", "", "", "", "", "", "", nil
	}

	level, err = getField(fields, fieldLevel)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", fmt.Errorf("dismissOneNotification: failed to get level field: %w", err)
	}

	message, err = getField(fields, fieldMessage)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", fmt.Errorf("dismissOneNotification: failed to get message field: %w", err)
	}

	timestamp, err = getField(fields, fieldTimestamp)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", fmt.Errorf("dismissOneNotification: failed to get timestamp field: %w", err)
	}

	session, err = getField(fields, fieldSession)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", fmt.Errorf("dismissOneNotification: failed to get session field: %w", err)
	}

	window, err = getField(fields, fieldWindow)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", fmt.Errorf("dismissOneNotification: failed to get window field: %w", err)
	}

	pane, err = getField(fields, fieldPane)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", fmt.Errorf("dismissOneNotification: failed to get pane field: %w", err)
	}

	paneCreated, err = getField(fields, fieldPaneCreated)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", fmt.Errorf("dismissOneNotification: failed to get pane created field: %w", err)
	}

	readTimestamp, err = getField(fields, fieldReadTimestamp)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", fmt.Errorf("dismissOneNotification: failed to get read timestamp field: %w", err)
	}

	return id, level, message, timestamp, session, window, pane, paneCreated, readTimestamp, nil
}

// dismissOneNotification dismisses a single notification with hooks (pre-dismiss → appendLine → post-dismiss).
func dismissOneNotification(line string) error {
	id, level, message, timestamp, session, window, pane, paneCreated, readTimestamp, err := parseDismissalFields(line)
	if err != nil {
		return err
	}

	// If all fields are empty, the notification was not active
	if id == "" {
		// Not active, nothing to dismiss
		return nil
	}

	// Run pre-dismiss hook
	envVars := buildHookEnvVars(id, level, message, timestamp, session, window, pane, paneCreated, readTimestamp)
	if err := hooks.Run("pre-dismiss", envVars...); err != nil {
		return err
	}

	idInt, err := strToInt(id)
	if err != nil {
		return fmt.Errorf("dismissOneNotification: invalid id %s: %w", id, err)
	}

	// Append dismissed line
	if err := appendLine(
		idInt,
		timestamp,
		"dismissed",
		session,
		window,
		pane,
		message,
		paneCreated,
		level,
		readTimestamp,
	); err != nil {
		return err
	}

	// Run post-dismiss hook
	if err := hooks.Run("post-dismiss", envVars...); err != nil {
		return err
	}
	return nil
}

// findNotificationsToDelete collects IDs of dismissed notifications older than cutoff.
// Returns a slice of notification IDs to delete.
func findNotificationsToDelete(latestLines []string, allDismissed bool, cutoffStr string) []int {
	var idsToDelete []int
	for _, line := range latestLines {
		fields := strings.Split(line, "\t")
		state, err := getField(fields, fieldState)
		if err != nil {
			continue
		}
		if state != "dismissed" {
			continue
		}
		timestamp, err := getField(fields, fieldTimestamp)
		if err != nil {
			continue
		}
		if !allDismissed && timestamp >= cutoffStr {
			continue
		}
		idField, err := getField(fields, fieldID)
		if err != nil {
			continue
		}
		id, err := strconv.Atoi(idField)
		if err != nil {
			continue
		}
		idsToDelete = append(idsToDelete, id)
	}
	return idsToDelete
}

// filterLinesByIDs removes lines whose ID is in idsToDelete.
// Returns the filtered lines.
func filterLinesByIDs(lines []string, idsToDelete []int) []string {
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
	return filtered
}

// writeNotifications writes lines to the notifications file.
func writeNotifications(lines []string) error {
	data := strings.Join(lines, "\n")
	if len(lines) > 0 {
		data += "\n"
	}
	if err := os.WriteFile(notificationsFile, []byte(data), FileModeFile); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// runPostCleanupHook runs the post-cleanup hook with the given environment variables and deleted count.
func runPostCleanupHook(envVars []string, deletedCount int, dryRun bool) error {
	// Ensure DRY_RUN is correctly set in envVars (caller should have set it)
	postEnv := append(envVars, fmt.Sprintf("DELETED_COUNT=%d", deletedCount))
	if err := hooks.Run("post-cleanup", postEnv...); err != nil {
		return fmt.Errorf("post-cleanup hook failed: %w", err)
	}
	return nil
}

// deleteNotificationsByIDs removes all lines with the given IDs from the notifications file.
func deleteNotificationsByIDs(idsToDelete []int) error {
	lines, err := readAllLines()
	if err != nil {
		return fmt.Errorf("failed to read all lines: %w", err)
	}
	filtered := filterLinesByIDs(lines, idsToDelete)
	if err := writeNotifications(filtered); err != nil {
		return fmt.Errorf("failed to write filtered lines: %w", err)
	}
	return nil
}

// cleanupOld performs the actual cleanup of old notifications.
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

	// Find notifications to delete
	idsToDelete := findNotificationsToDelete(latestLines, allDismissed, cutoffStr)
	deletedCount := len(idsToDelete)

	if deletedCount == 0 {
		colors.Info("No old dismissed notifications to clean up")
		// Run post-cleanup hooks with zero count
		if err := runPostCleanupHook(envVars, 0, dryRun); err != nil {
			return err
		}
		return nil
	}

	colors.Info(fmt.Sprintf("Found %d notification(s) to clean up", deletedCount))

	if dryRun {
		colors.Info(fmt.Sprintf("Dry run: would delete notifications with IDs: %v", idsToDelete))
		// Run post-cleanup hooks with dry run
		if err := runPostCleanupHook(envVars, deletedCount, dryRun); err != nil {
			return err
		}
		return nil
	}

	// Delete notifications by IDs
	if err := deleteNotificationsByIDs(idsToDelete); err != nil {
		return fmt.Errorf("failed to delete notifications: %w", err)
	}

	colors.Info(fmt.Sprintf("Successfully cleaned up %d notification(s)", deletedCount))

	// Run post-cleanup hooks
	if err := runPostCleanupHook(envVars, deletedCount, dryRun); err != nil {
		return err
	}
	return nil
}

func strToInt(s string) (int, error) {
	if strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("strToInt: negative value not allowed: %s", s)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("strToInt: failed to convert '%s' to int: %w", s, err)
	}
	return n, nil
}

// Reset resets the storage package state for testing.
func Reset() {
	initMu.Lock()
	defer initMu.Unlock()

	notificationsFile = ""
	lockDir = ""
	stateDir = ""
	initialized = false
	initErr = nil

	// Reset sync.Once by creating a new one
	// This is safe because Reset() should only be called in tests
	initOnce = &sync.Once{}
}

// GetStateDir returns the state directory path.
func GetStateDir() string {
	if stateDir != "" {
		return stateDir
	}
	if dir := os.Getenv("TMUX_INTRAY_STATE_DIR"); dir != "" {
		return dir
	}
	config.Load()
	return config.Get("state_dir", "")
}
