// Package storage provides storage backend selection and implementations.
package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite/sqlcgen"
)

const (
	// ReadBackendTSV makes read operations use the TSV backend.
	ReadBackendTSV = "tsv"
	// ReadBackendSQLite makes read operations use the SQLite backend.
	ReadBackendSQLite = "sqlite"
)

var fieldNames = []string{
	"id",
	"timestamp",
	"state",
	"session",
	"window",
	"pane",
	"message",
	"pane_created",
	"level",
	"read_timestamp",
}

// DualWriterOptions controls DualWriter behavior.
type DualWriterOptions struct {
	ReadBackend string
	VerifyOnly  bool
	SampleSize  int
}

// WriteMetrics tracks basic write performance and reliability counters.
type WriteMetrics struct {
	WriteOperations    int64
	TSVWriteFailures   int64
	SQLiteWriteFailure int64
	TotalWriteLatency  time.Duration
	MaxWriteLatency    time.Duration
}

// AverageWriteLatency returns the mean latency for write operations.
func (m WriteMetrics) AverageWriteLatency() time.Duration {
	if m.WriteOperations == 0 {
		return 0
	}
	return m.TotalWriteLatency / time.Duration(m.WriteOperations)
}

// ConsistencyFieldDiff represents a mismatch in a specific notification field.
type ConsistencyFieldDiff struct {
	Field       string
	TSVValue    string
	SQLiteValue string
}

// ConsistencyRecordDiff represents all field mismatches for one notification ID.
type ConsistencyRecordDiff struct {
	ID         string
	FieldDiffs []ConsistencyFieldDiff
}

// ConsistencyReport contains cross-backend consistency results.
type ConsistencyReport struct {
	TSVCount          int
	SQLiteCount       int
	TSVActiveCount    int
	SQLiteActiveCount int
	SampledRecords    int
	MissingInTSV      []string
	MissingInSQLite   []string
	RecordDiffs       []ConsistencyRecordDiff
	Consistent        bool
}

// DualWriter writes to TSV then SQLite and can read from a selected backend.
type DualWriter struct {
	tsv         Storage
	sqlite      Storage
	readBackend string
	verifyOnly  bool
	sampleSize  int

	backendMu sync.RWMutex

	metricsMu sync.Mutex
	metrics   WriteMetrics
}

var _ Storage = (*DualWriter)(nil)

// NewDualWriter creates a DualWriter using default TSV and SQLite backends.
func NewDualWriter(opts DualWriterOptions) (*DualWriter, error) {
	tsvStorage, err := NewFileStorage()
	if err != nil {
		return nil, fmt.Errorf("dual writer: initialize tsv backend: %w", err)
	}

	dbPath := filepath.Join(GetStateDir(), "notifications.db")
	sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
	if err != nil {
		return nil, fmt.Errorf("dual writer: initialize sqlite backend: %w", err)
	}

	dw, err := NewDualWriterWithBackends(tsvStorage, sqliteStorage, opts)
	if err != nil {
		return nil, err
	}

	// Perform automatic migration if SQLite is empty and TSV has data
	if err := dw.migrateTSVToSQLiteIfNeeded(); err != nil {
		colors.Warning(fmt.Sprintf("dual writer: automatic tsv to sqlite migration failed: %v", err))
	}

	return dw, nil
}

// NewDualWriterWithBackends creates a DualWriter from explicit backend instances.
func NewDualWriterWithBackends(tsvStorage, sqliteStorage Storage, opts DualWriterOptions) (*DualWriter, error) {
	if tsvStorage == nil {
		return nil, fmt.Errorf("dual writer: tsv backend is required")
	}
	if sqliteStorage == nil {
		return nil, fmt.Errorf("dual writer: sqlite backend is required")
	}

	readBackend := strings.ToLower(strings.TrimSpace(opts.ReadBackend))
	if readBackend == "" {
		readBackend = ReadBackendSQLite
	}
	if readBackend != ReadBackendTSV && readBackend != ReadBackendSQLite {
		colors.Warning(fmt.Sprintf("invalid dual read backend '%s', defaulting to sqlite", opts.ReadBackend))
		readBackend = ReadBackendSQLite
	}

	sampleSize := opts.SampleSize
	if sampleSize <= 0 {
		sampleSize = 25
	}

	return &DualWriter{
		tsv:         tsvStorage,
		sqlite:      sqliteStorage,
		readBackend: readBackend,
		verifyOnly:  opts.VerifyOnly,
		sampleSize:  sampleSize,
	}, nil
}

// AddNotification writes to TSV then SQLite.
func (d *DualWriter) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	start := time.Now()
	id, tsvErr := d.tsv.AddNotification(message, timestamp, session, window, pane, paneCreated, level)
	if tsvErr != nil {
		d.recordWrite(start, true, false)
		return "", tsvErr
	}

	sqliteFailed := false
	if !d.verifyOnly {
		sqliteID, sqliteErr := d.sqlite.AddNotification(message, timestamp, session, window, pane, paneCreated, level)
		if sqliteErr != nil {
			sqliteFailed = true
			d.handleSQLiteWriteFailure(fmt.Sprintf("add failed for id %s", id), sqliteErr)
		} else if sqliteID != id {
			colors.Warning(fmt.Sprintf("dual write: id mismatch after add (tsv=%s sqlite=%s)", id, sqliteID))
		}
	}

	d.recordWrite(start, false, sqliteFailed)
	return id, nil
}

// ListNotifications delegates reads to the configured read backend.
func (d *DualWriter) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff string) (string, error) {
	return d.readStorage().ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff)
}

// GetNotificationByID delegates reads to the configured read backend.
func (d *DualWriter) GetNotificationByID(id string) (string, error) {
	return d.readStorage().GetNotificationByID(id)
}

// DismissNotification writes to TSV then SQLite.
func (d *DualWriter) DismissNotification(id string) error {
	start := time.Now()
	if err := d.tsv.DismissNotification(id); err != nil {
		d.recordWrite(start, true, false)
		return err
	}

	sqliteFailed := false
	if !d.verifyOnly {
		if err := d.sqlite.DismissNotification(id); err != nil {
			sqliteFailed = true
			d.handleSQLiteWriteFailure(fmt.Sprintf("dismiss failed for id %s", id), err)
		}
	}

	d.recordWrite(start, false, sqliteFailed)
	return nil
}

// DismissAll writes to TSV then SQLite.
func (d *DualWriter) DismissAll() error {
	start := time.Now()
	if err := d.tsv.DismissAll(); err != nil {
		d.recordWrite(start, true, false)
		return err
	}

	sqliteFailed := false
	if !d.verifyOnly {
		if err := d.sqlite.DismissAll(); err != nil {
			sqliteFailed = true
			d.handleSQLiteWriteFailure("dismiss all failed", err)
		}
	}

	d.recordWrite(start, false, sqliteFailed)
	return nil
}

// MarkNotificationRead writes to TSV then SQLite.
func (d *DualWriter) MarkNotificationRead(id string) error {
	start := time.Now()
	if err := d.tsv.MarkNotificationRead(id); err != nil {
		d.recordWrite(start, true, false)
		return err
	}

	sqliteFailed := false
	if !d.verifyOnly {
		if err := d.sqlite.MarkNotificationRead(id); err != nil {
			sqliteFailed = true
			d.handleSQLiteWriteFailure(fmt.Sprintf("mark-read failed for id %s", id), err)
		}
	}

	d.recordWrite(start, false, sqliteFailed)
	return nil
}

// MarkNotificationUnread writes to TSV then SQLite.
func (d *DualWriter) MarkNotificationUnread(id string) error {
	start := time.Now()
	if err := d.tsv.MarkNotificationUnread(id); err != nil {
		d.recordWrite(start, true, false)
		return err
	}

	sqliteFailed := false
	if !d.verifyOnly {
		if err := d.sqlite.MarkNotificationUnread(id); err != nil {
			sqliteFailed = true
			d.handleSQLiteWriteFailure(fmt.Sprintf("mark-unread failed for id %s", id), err)
		}
	}

	d.recordWrite(start, false, sqliteFailed)
	return nil
}

// CleanupOldNotifications writes to TSV then SQLite.
func (d *DualWriter) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	start := time.Now()
	if err := d.tsv.CleanupOldNotifications(daysThreshold, dryRun); err != nil {
		d.recordWrite(start, true, false)
		return err
	}

	sqliteFailed := false
	if !d.verifyOnly {
		if err := d.sqlite.CleanupOldNotifications(daysThreshold, dryRun); err != nil {
			sqliteFailed = true
			d.handleSQLiteWriteFailure("cleanup failed", err)
		}
	}

	d.recordWrite(start, false, sqliteFailed)
	return nil
}

// GetActiveCount delegates reads to the configured read backend.
func (d *DualWriter) GetActiveCount() int {
	return d.readStorage().GetActiveCount()
}

// migrateTSVToSQLiteIfNeeded performs automatic migration if SQLite is empty and TSV has data.
// This ensures existing TSV data is visible in dual mode with SQLite read preference.
func (d *DualWriter) migrateTSVToSQLiteIfNeeded() error {
	// Check if SQLite is empty
	sqliteCount := d.sqlite.GetActiveCount()
	tsvCount := d.tsv.GetActiveCount()

	colors.Debug(fmt.Sprintf("dual writer migration check: sqlite_count=%d tsv_count=%d", sqliteCount, tsvCount))

	// No migration needed if SQLite has data or TSV is empty
	if sqliteCount > 0 || tsvCount == 0 {
		colors.Debug("dual writer: skipping migration (sqlite has data or tsv is empty)")
		return nil
	}

	colors.Info(fmt.Sprintf("dual writer: migrating %d notifications from tsv to sqlite", tsvCount))

	// Get all notifications from TSV (including dismissed ones)
	tsvLines, err := d.tsv.ListNotifications("all", "", "", "", "", "", "")
	if err != nil {
		return fmt.Errorf("dual writer: list tsv notifications: %w", err)
	}

	// Parse TSV lines and migrate to SQLite
	return d.migrateNotificationLines(tsvLines)
}

// migrateNotificationLines parses TSV lines and upserts them to SQLite using sqlc.
func (d *DualWriter) migrateNotificationLines(tsvContent string) error {
	if strings.TrimSpace(tsvContent) == "" {
		return nil
	}

	// Type assertion to access SQLite storage methods
	sqliteStorage, ok := d.sqlite.(*sqlite.SQLiteStorage)
	if !ok {
		return fmt.Errorf("dual writer: sqlite backend is not sqlite.SQLiteStorage type")
	}

	// Parse TSV lines
	records := parseNotificationLines(tsvContent)

	// Collect IDs for sorted migration
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, id)
	}
	sortIDs(ids)

	// Migrate each notification using sqlc-backed UpsertNotification
	for _, idStr := range ids {
		fields := records[idStr]
		if len(fields) < NumFields {
			continue
		}

		// Parse ID as int64 for sqlc
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			colors.Warning(fmt.Sprintf("dual writer: skipping invalid id %s during migration", idStr))
			continue
		}

		// Build parameters for UpsertNotification
		// TSV messages are stored in escaped form in the TSV file.
		// We store them as-is (still escaped) in SQLite for consistency with the source of truth.
		// This ensures that messages migrated from TSV appear identical to those added directly via SQLite.
		params := sqlcgen.UpsertNotificationParams{
			ID:            id,
			Timestamp:     fields[FieldTimestamp],
			State:         fields[FieldState],
			Session:       fields[FieldSession],
			Window:        fields[FieldWindow],
			Pane:          fields[FieldPane],
			Message:       fields[FieldMessage],
			PaneCreated:   fields[FieldPaneCreated],
			Level:         fields[FieldLevel],
			ReadTimestamp: fields[FieldReadTimestamp],
			UpdatedAt:     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		}

		// Validate state and level before upsert
		if params.State != "active" && params.State != "dismissed" {
			colors.Warning(fmt.Sprintf("dual writer: skipping notification id %s with invalid state '%s'", idStr, params.State))
			continue
		}
		if params.Level != "info" && params.Level != "warning" && params.Level != "error" && params.Level != "critical" {
			colors.Warning(fmt.Sprintf("dual writer: skipping notification id %s with invalid level '%s'", idStr, params.Level))
			continue
		}

		if err := sqliteStorage.UpsertNotification(context.Background(), params); err != nil {
			colors.Warning(fmt.Sprintf("dual writer: failed to migrate notification id %s: %v", idStr, err))
			continue
		}
	}

	colors.Success(fmt.Sprintf("dual writer: successfully migrated notifications from tsv to sqlite"))
	return nil
}

// Metrics returns a snapshot of write metrics.
func (d *DualWriter) Metrics() WriteMetrics {
	d.metricsMu.Lock()
	defer d.metricsMu.Unlock()
	return d.metrics
}

// VerifyConsistency compares TSV and SQLite data and reports discrepancies.
func (d *DualWriter) VerifyConsistency(sampleSize int) (ConsistencyReport, error) {
	tsvLines, err := d.tsv.ListNotifications("all", "", "", "", "", "", "")
	if err != nil {
		return ConsistencyReport{}, fmt.Errorf("dual writer consistency: list tsv notifications: %w", err)
	}
	sqliteLines, err := d.sqlite.ListNotifications("all", "", "", "", "", "", "")
	if err != nil {
		return ConsistencyReport{}, fmt.Errorf("dual writer consistency: list sqlite notifications: %w", err)
	}

	tsvRecords := parseNotificationLines(tsvLines)
	sqliteRecords := parseNotificationLines(sqliteLines)

	report := ConsistencyReport{
		TSVCount:          len(tsvRecords),
		SQLiteCount:       len(sqliteRecords),
		TSVActiveCount:    countActive(tsvRecords),
		SQLiteActiveCount: countActive(sqliteRecords),
	}

	sampleLimit := sampleSize
	if sampleLimit <= 0 {
		sampleLimit = d.sampleSize
	}

	report.MissingInTSV, report.MissingInSQLite = findMissingIDs(tsvRecords, sqliteRecords)
	for _, id := range report.MissingInTSV {
		colors.Warning(fmt.Sprintf("dual writer consistency discrepancy: id %s missing in tsv", id))
	}
	for _, id := range report.MissingInSQLite {
		colors.Warning(fmt.Sprintf("dual writer consistency discrepancy: id %s missing in sqlite", id))
	}

	sampledIDs := sampledIntersectionIDs(tsvRecords, sqliteRecords, sampleLimit)
	report.SampledRecords = len(sampledIDs)

	for _, id := range sampledIDs {
		tsvFields := tsvRecords[id]
		sqliteFields := sqliteRecords[id]
		diffs := diffFields(tsvFields, sqliteFields)
		if len(diffs) == 0 {
			continue
		}
		report.RecordDiffs = append(report.RecordDiffs, ConsistencyRecordDiff{
			ID:         id,
			FieldDiffs: diffs,
		})
		for _, diff := range diffs {
			colors.Warning(fmt.Sprintf("dual writer consistency discrepancy: id %s field %s differs (tsv=%q sqlite=%q)", id, diff.Field, diff.TSVValue, diff.SQLiteValue))
		}
	}

	report.Consistent =
		report.TSVCount == report.SQLiteCount &&
			report.TSVActiveCount == report.SQLiteActiveCount &&
			len(report.MissingInTSV) == 0 &&
			len(report.MissingInSQLite) == 0 &&
			len(report.RecordDiffs) == 0

	return report, nil
}

func (d *DualWriter) recordWrite(start time.Time, tsvFailed, sqliteFailed bool) {
	d.metricsMu.Lock()
	defer d.metricsMu.Unlock()

	latency := time.Since(start)
	d.metrics.WriteOperations++
	d.metrics.TotalWriteLatency += latency
	if latency > d.metrics.MaxWriteLatency {
		d.metrics.MaxWriteLatency = latency
	}
	if tsvFailed {
		d.metrics.TSVWriteFailures++
	}
	if sqliteFailed {
		d.metrics.SQLiteWriteFailure++
	}
}

func (d *DualWriter) readStorage() Storage {
	d.backendMu.RLock()
	defer d.backendMu.RUnlock()

	if d.readBackend == ReadBackendTSV {
		return d.tsv
	}
	return d.sqlite
}

func (d *DualWriter) handleSQLiteWriteFailure(operation string, err error) {
	d.backendMu.Lock()
	defer d.backendMu.Unlock()

	if d.readBackend == ReadBackendSQLite {
		d.readBackend = ReadBackendTSV
		colors.Warning(fmt.Sprintf("dual write: sqlite %s; switching reads to tsv: %v", operation, err))
		return
	}

	colors.Warning(fmt.Sprintf("dual write: sqlite %s, continuing with tsv only: %v", operation, err))
}

func parseNotificationLines(content string) map[string][]string {
	records := make(map[string][]string)
	if strings.TrimSpace(content) == "" {
		return records
	}

	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < NumFields {
			for len(fields) < NumFields {
				fields = append(fields, "")
			}
		}
		if len(fields) <= FieldID {
			continue
		}
		records[fields[FieldID]] = fields
	}
	return records
}

func countActive(records map[string][]string) int {
	count := 0
	for _, fields := range records {
		if len(fields) > FieldState && fields[FieldState] == "active" {
			count++
		}
	}
	return count
}

func findMissingIDs(tsvRecords, sqliteRecords map[string][]string) ([]string, []string) {
	missingInTSV := make([]string, 0)
	missingInSQLite := make([]string, 0)

	for id := range sqliteRecords {
		if _, ok := tsvRecords[id]; !ok {
			missingInTSV = append(missingInTSV, id)
		}
	}
	for id := range tsvRecords {
		if _, ok := sqliteRecords[id]; !ok {
			missingInSQLite = append(missingInSQLite, id)
		}
	}

	sortIDs(missingInTSV)
	sortIDs(missingInSQLite)
	return missingInTSV, missingInSQLite
}

func sampledIntersectionIDs(tsvRecords, sqliteRecords map[string][]string, sampleSize int) []string {
	ids := make([]string, 0)
	for id := range tsvRecords {
		if _, ok := sqliteRecords[id]; ok {
			ids = append(ids, id)
		}
	}
	sortIDs(ids)
	if sampleSize > 0 && len(ids) > sampleSize {
		return ids[:sampleSize]
	}
	return ids
}

func diffFields(tsvFields, sqliteFields []string) []ConsistencyFieldDiff {
	diffs := make([]ConsistencyFieldDiff, 0)
	for i := 0; i < NumFields; i++ {
		tsvValue := ""
		sqliteValue := ""
		if i < len(tsvFields) {
			tsvValue = tsvFields[i]
		}
		if i < len(sqliteFields) {
			sqliteValue = sqliteFields[i]
		}
		if tsvValue != sqliteValue {
			diffs = append(diffs, ConsistencyFieldDiff{
				Field:       fieldNames[i],
				TSVValue:    tsvValue,
				SQLiteValue: sqliteValue,
			})
		}
	}
	return diffs
}

func sortIDs(ids []string) {
	sort.Slice(ids, func(i, j int) bool {
		left, leftErr := strconv.Atoi(ids[i])
		right, rightErr := strconv.Atoi(ids[j])
		if leftErr == nil && rightErr == nil {
			return left < right
		}
		return ids[i] < ids[j]
	})
}
