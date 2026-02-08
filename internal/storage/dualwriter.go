// Package storage provides storage backend selection and implementations.
package storage

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
)

const (
	defaultConsistencySampleSize = 10
	defaultVerifyEveryNWrites    = 25
	notificationFieldCount       = 10
)

// DualWriterOptions controls runtime behavior for DualWriter.
type DualWriterOptions struct {
	// ReadFromSQLite controls which backend serves reads. Defaults to true.
	ReadFromSQLite bool
	// ReadOnlyVerificationMode writes to TSV only while still allowing SQLite reads.
	ReadOnlyVerificationMode bool
	// ConsistencySampleSize controls how many shared IDs are compared per verify call.
	ConsistencySampleSize int
	// VerifyEveryNWrites runs VerifyConsistency every N write operations.
	// Set to 0 to disable periodic verification.
	VerifyEveryNWrites int
}

// WriteMetrics captures aggregate write latency and count information.
type WriteMetrics struct {
	WriteOperations    int64
	TSVWriteLatency    time.Duration
	SQLiteWriteLatency time.Duration
}

// ConsistencyReport captures parity checks between TSV and SQLite backends.
type ConsistencyReport struct {
	TSVRecordCount    int
	SQLiteRecordCount int
	TSVActiveCount    int
	SQLiteActiveCount int
	MissingInTSV      []string
	MissingInSQLite   []string
	MismatchedIDs     []string
}

// HasCriticalDifferences returns true when the report indicates divergence.
func (r ConsistencyReport) HasCriticalDifferences() bool {
	return r.TSVRecordCount != r.SQLiteRecordCount ||
		r.TSVActiveCount != r.SQLiteActiveCount ||
		len(r.MissingInTSV) > 0 ||
		len(r.MissingInSQLite) > 0 ||
		len(r.MismatchedIDs) > 0
}

// DualWriter writes to TSV and SQLite during rollout, then verifies parity.
type DualWriter struct {
	tsv    Storage
	sqlite Storage
	opts   DualWriterOptions

	mu                  sync.Mutex
	metrics             WriteMetrics
	sqliteWriteDisabled bool
	rand                *rand.Rand
}

var _ Storage = (*DualWriter)(nil)

// NewDualWriter creates a dual-writer storage wrapper.
func NewDualWriter(tsvStore, sqliteStore Storage, opts DualWriterOptions) (*DualWriter, error) {
	if tsvStore == nil {
		return nil, fmt.Errorf("dual writer: tsv store is required")
	}
	if sqliteStore == nil {
		return nil, fmt.Errorf("dual writer: sqlite store is required")
	}

	if opts.ConsistencySampleSize <= 0 {
		opts.ConsistencySampleSize = defaultConsistencySampleSize
	}
	if opts.VerifyEveryNWrites < 0 {
		opts.VerifyEveryNWrites = 0
	}

	return &DualWriter{
		tsv:    tsvStore,
		sqlite: sqliteStore,
		opts:   opts,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// AddNotification writes to TSV first, then SQLite.
func (d *DualWriter) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	var tsvID, sqliteID string
	tsvErr := d.withTSVWriteMetric(func() error {
		var err error
		tsvID, err = d.tsv.AddNotification(message, timestamp, session, window, pane, paneCreated, level)
		return err
	})
	if tsvErr != nil {
		colors.Warning(fmt.Sprintf("dual writer: tsv add failed: %v", tsvErr))
	}

	var sqliteErr error
	if !d.skipSQLiteWrites() {
		sqliteErr = d.withSQLiteWriteMetric(func() error {
			var err error
			sqliteID, err = d.sqlite.AddNotification(message, timestamp, session, window, pane, paneCreated, level)
			return err
		})
		if sqliteErr != nil {
			d.disableSQLiteWrites(sqliteErr)
		}
	}

	if tsvErr != nil && sqliteErr != nil {
		return "", fmt.Errorf("dual writer: add failed on both backends: tsv=%v sqlite=%v", tsvErr, sqliteErr)
	}

	if tsvErr == nil && sqliteErr == nil && tsvID != sqliteID {
		colors.Warning(fmt.Sprintf("dual writer: id mismatch after add (tsv=%s sqlite=%s)", tsvID, sqliteID))
	}

	d.afterWrite()
	if d.opts.ReadFromSQLite && sqliteErr == nil && sqliteID != "" {
		return sqliteID, nil
	}
	if tsvID != "" {
		return tsvID, nil
	}
	return sqliteID, nil
}

// ListNotifications reads from configured primary backend and falls back on error.
func (d *DualWriter) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error) {
	if d.opts.ReadFromSQLite {
		result, err := d.sqlite.ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
		if err == nil {
			return result, nil
		}
		colors.Warning(fmt.Sprintf("dual writer: sqlite list failed, falling back to tsv: %v", err))
	}
	return d.tsv.ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
}

// GetNotificationByID reads from configured primary backend and falls back on error.
func (d *DualWriter) GetNotificationByID(id string) (string, error) {
	if d.opts.ReadFromSQLite {
		result, err := d.sqlite.GetNotificationByID(id)
		if err == nil {
			return result, nil
		}
		colors.Warning(fmt.Sprintf("dual writer: sqlite get by id failed, falling back to tsv: %v", err))
	}
	return d.tsv.GetNotificationByID(id)
}

// DismissNotification writes to TSV first, then SQLite.
func (d *DualWriter) DismissNotification(id string) error {
	tsvErr := d.withTSVWriteMetric(func() error {
		return d.tsv.DismissNotification(id)
	})
	if tsvErr != nil {
		colors.Warning(fmt.Sprintf("dual writer: tsv dismiss failed for id %s: %v", id, tsvErr))
	}

	var sqliteErr error
	if !d.skipSQLiteWrites() {
		sqliteErr = d.withSQLiteWriteMetric(func() error {
			return d.sqlite.DismissNotification(id)
		})
		if sqliteErr != nil {
			d.disableSQLiteWrites(sqliteErr)
		}
	}

	if tsvErr != nil && sqliteErr != nil {
		return fmt.Errorf("dual writer: dismiss failed on both backends: tsv=%v sqlite=%v", tsvErr, sqliteErr)
	}

	d.afterWrite()
	if d.opts.ReadFromSQLite && sqliteErr == nil {
		return nil
	}
	if tsvErr == nil {
		return nil
	}
	return sqliteErr
}

// DismissAll writes to TSV first, then SQLite.
func (d *DualWriter) DismissAll() error {
	tsvErr := d.withTSVWriteMetric(d.tsv.DismissAll)
	if tsvErr != nil {
		colors.Warning(fmt.Sprintf("dual writer: tsv dismiss all failed: %v", tsvErr))
	}

	var sqliteErr error
	if !d.skipSQLiteWrites() {
		sqliteErr = d.withSQLiteWriteMetric(d.sqlite.DismissAll)
		if sqliteErr != nil {
			d.disableSQLiteWrites(sqliteErr)
		}
	}

	if tsvErr != nil && sqliteErr != nil {
		return fmt.Errorf("dual writer: dismiss all failed on both backends: tsv=%v sqlite=%v", tsvErr, sqliteErr)
	}

	d.afterWrite()
	if tsvErr == nil || sqliteErr == nil {
		return nil
	}
	return tsvErr
}

// MarkNotificationRead writes to TSV first, then SQLite.
func (d *DualWriter) MarkNotificationRead(id string) error {
	return d.markNotificationReadState(id, true)
}

// MarkNotificationUnread writes to TSV first, then SQLite.
func (d *DualWriter) MarkNotificationUnread(id string) error {
	return d.markNotificationReadState(id, false)
}

func (d *DualWriter) markNotificationReadState(id string, read bool) error {
	tsvErr := d.withTSVWriteMetric(func() error {
		if read {
			return d.tsv.MarkNotificationRead(id)
		}
		return d.tsv.MarkNotificationUnread(id)
	})
	if tsvErr != nil {
		colors.Warning(fmt.Sprintf("dual writer: tsv mark read state failed for id %s: %v", id, tsvErr))
	}

	var sqliteErr error
	if !d.skipSQLiteWrites() {
		sqliteErr = d.withSQLiteWriteMetric(func() error {
			if read {
				return d.sqlite.MarkNotificationRead(id)
			}
			return d.sqlite.MarkNotificationUnread(id)
		})
		if sqliteErr != nil {
			d.disableSQLiteWrites(sqliteErr)
		}
	}

	if tsvErr != nil && sqliteErr != nil {
		return fmt.Errorf("dual writer: mark read state failed on both backends: tsv=%v sqlite=%v", tsvErr, sqliteErr)
	}

	d.afterWrite()
	if tsvErr == nil || sqliteErr == nil {
		return nil
	}
	return tsvErr
}

// CleanupOldNotifications writes to TSV first, then SQLite.
func (d *DualWriter) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	tsvErr := d.withTSVWriteMetric(func() error {
		return d.tsv.CleanupOldNotifications(daysThreshold, dryRun)
	})
	if tsvErr != nil {
		colors.Warning(fmt.Sprintf("dual writer: tsv cleanup failed: %v", tsvErr))
	}

	var sqliteErr error
	if !d.skipSQLiteWrites() {
		sqliteErr = d.withSQLiteWriteMetric(func() error {
			return d.sqlite.CleanupOldNotifications(daysThreshold, dryRun)
		})
		if sqliteErr != nil {
			d.disableSQLiteWrites(sqliteErr)
		}
	}

	if tsvErr != nil && sqliteErr != nil {
		return fmt.Errorf("dual writer: cleanup failed on both backends: tsv=%v sqlite=%v", tsvErr, sqliteErr)
	}

	d.afterWrite()
	if tsvErr == nil || sqliteErr == nil {
		return nil
	}
	return tsvErr
}

// GetActiveCount reads from configured primary backend and falls back on error.
func (d *DualWriter) GetActiveCount() int {
	if d.opts.ReadFromSQLite {
		return d.sqlite.GetActiveCount()
	}
	return d.tsv.GetActiveCount()
}

// VerifyConsistency compares both backends and returns divergence details.
func (d *DualWriter) VerifyConsistency() (ConsistencyReport, error) {
	tsvLines, err := d.tsv.ListNotifications("all", "", "", "", "", "", "")
	if err != nil {
		return ConsistencyReport{}, fmt.Errorf("dual writer: list tsv notifications: %w", err)
	}
	sqliteLines, err := d.sqlite.ListNotifications("all", "", "", "", "", "", "")
	if err != nil {
		return ConsistencyReport{}, fmt.Errorf("dual writer: list sqlite notifications: %w", err)
	}

	tsvMap := parseNotificationMap(tsvLines)
	sqliteMap := parseNotificationMap(sqliteLines)

	report := ConsistencyReport{
		TSVRecordCount:    len(tsvMap),
		SQLiteRecordCount: len(sqliteMap),
		TSVActiveCount:    d.tsv.GetActiveCount(),
		SQLiteActiveCount: d.sqlite.GetActiveCount(),
	}

	for id := range tsvMap {
		if _, ok := sqliteMap[id]; !ok {
			report.MissingInSQLite = append(report.MissingInSQLite, id)
		}
	}
	for id := range sqliteMap {
		if _, ok := tsvMap[id]; !ok {
			report.MissingInTSV = append(report.MissingInTSV, id)
		}
	}

	sort.Strings(report.MissingInSQLite)
	sort.Strings(report.MissingInTSV)

	sharedIDs := intersectSortedIDs(tsvMap, sqliteMap)
	for _, id := range d.sampleIDs(sharedIDs, d.opts.ConsistencySampleSize) {
		if tsvMap[id] != sqliteMap[id] {
			report.MismatchedIDs = append(report.MismatchedIDs, id)
		}
	}

	if report.HasCriticalDifferences() {
		colors.Warning(fmt.Sprintf("dual writer consistency mismatch: tsv_count=%d sqlite_count=%d tsv_active=%d sqlite_active=%d missing_in_sqlite=%v missing_in_tsv=%v mismatched_ids=%v",
			report.TSVRecordCount,
			report.SQLiteRecordCount,
			report.TSVActiveCount,
			report.SQLiteActiveCount,
			report.MissingInSQLite,
			report.MissingInTSV,
			report.MismatchedIDs,
		))
		return report, fmt.Errorf("dual writer: critical consistency differences detected")
	}

	return report, nil
}

// Metrics returns a point-in-time copy of write metrics.
func (d *DualWriter) Metrics() WriteMetrics {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.metrics
}

func (d *DualWriter) withTSVWriteMetric(fn func() error) error {
	start := time.Now()
	err := fn()
	d.mu.Lock()
	d.metrics.TSVWriteLatency += time.Since(start)
	d.mu.Unlock()
	return err
}

func (d *DualWriter) withSQLiteWriteMetric(fn func() error) error {
	start := time.Now()
	err := fn()
	d.mu.Lock()
	d.metrics.SQLiteWriteLatency += time.Since(start)
	d.mu.Unlock()
	return err
}

func (d *DualWriter) afterWrite() {
	d.mu.Lock()
	d.metrics.WriteOperations++
	shouldVerify := d.opts.VerifyEveryNWrites > 0 && d.metrics.WriteOperations%int64(d.opts.VerifyEveryNWrites) == 0
	d.mu.Unlock()

	if !shouldVerify {
		return
	}
	if _, err := d.VerifyConsistency(); err != nil {
		colors.Warning(fmt.Sprintf("dual writer periodic consistency verification failed: %v", err))
	}
}

func (d *DualWriter) skipSQLiteWrites() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.opts.ReadOnlyVerificationMode || d.sqliteWriteDisabled
}

func (d *DualWriter) disableSQLiteWrites(err error) {
	d.mu.Lock()
	d.sqliteWriteDisabled = true
	d.mu.Unlock()
	colors.Warning(fmt.Sprintf("dual writer: sqlite write failed, falling back to tsv-only mode: %v", err))
}

func parseNotificationMap(content string) map[string]string {
	result := make(map[string]string)
	if strings.TrimSpace(content) == "" {
		return result
	}

	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		normalized, ok := normalizeNotificationLine(line)
		if !ok {
			continue
		}
		id := strings.SplitN(normalized, "\t", 2)[0]
		result[id] = normalized
	}
	return result
}

func normalizeNotificationLine(line string) (string, bool) {
	parts := strings.Split(line, "\t")
	if len(parts) == 0 {
		return "", false
	}
	id := strings.TrimSpace(parts[0])
	if id == "" {
		return "", false
	}
	if _, err := strconv.Atoi(id); err != nil {
		return "", false
	}
	for len(parts) < notificationFieldCount {
		parts = append(parts, "")
	}
	return strings.Join(parts[:notificationFieldCount], "\t"), true
}

func intersectSortedIDs(a, b map[string]string) []string {
	ids := make([]string, 0, len(a))
	for id := range a {
		if _, ok := b[id]; ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (d *DualWriter) sampleIDs(ids []string, sampleSize int) []string {
	if len(ids) <= sampleSize {
		return ids
	}
	if sampleSize <= 0 {
		return nil
	}

	indices := d.rand.Perm(len(ids))[:sampleSize]
	selection := make([]string, 0, sampleSize)
	for _, idx := range indices {
		selection = append(selection, ids[idx])
	}
	sort.Strings(selection)
	return selection
}
