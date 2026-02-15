// Package dedup provides helpers for building deduplication keys.
package dedup

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Criteria defines how notification duplicates are detected.
type Criteria string

const (
	CriteriaMessage       Criteria = "message"
	CriteriaMessageLevel  Criteria = "message_level"
	CriteriaMessageSource Criteria = "message_source"
	CriteriaExact         Criteria = "exact"

	bucketSeparator = "" // Unit Separator to avoid conflicts with message text
)

// Options configure deduplication behavior when building grouping keys.
type Options struct {
	Criteria Criteria
	Window   time.Duration
}

// Record captures the fields needed to compute deduplication keys.
type Record struct {
	Message   string
	Level     string
	Session   string
	Window    string
	Pane      string
	State     string
	Timestamp string
}

// ParseCriteria converts user-provided strings into a Criteria value.
func ParseCriteria(value string) Criteria {
	switch strings.ToLower(value) {
	case string(CriteriaMessageLevel):
		return CriteriaMessageLevel
	case string(CriteriaMessageSource):
		return CriteriaMessageSource
	case string(CriteriaExact):
		return CriteriaExact
	default:
		return CriteriaMessage
	}
}

// String returns the string value for Criteria.
func (c Criteria) String() string {
	return string(c)
}

// BuildKeys returns a deduplicated key for each record based on the provided options.
// The output slice has the same order and length as the input slice.
func BuildKeys(records []Record, opts Options) []string {
	criteria := opts.Criteria
	if criteria == "" {
		criteria = CriteriaMessage
	}
	keys := make([]string, len(records))
	for i := range records {
		keys[i] = buildBaseKey(records[i], criteria)
	}
	if opts.Window <= 0 {
		return keys
	}
	buckets := assignWindowBuckets(records, keys, opts.Window)
	for i, bucket := range buckets {
		if bucket > 0 {
			keys[i] = appendBucketSuffix(keys[i], bucket)
		}
	}
	return keys
}

// StripBucketSuffix removes the internal window-based suffix from a dedup key.
func StripBucketSuffix(key string) string {
	if idx := strings.Index(key, bucketSeparator); idx >= 0 {
		return key[:idx]
	}
	return key
}

func buildBaseKey(record Record, criteria Criteria) string {
	switch criteria {
	case CriteriaMessageLevel:
		return joinParts(record.Message, record.Level)
	case CriteriaMessageSource:
		return joinParts(record.Message, record.Session, record.Window, record.Pane)
	case CriteriaExact:
		return joinParts(record.Message, record.Level, record.Session, record.Window, record.Pane, record.State)
	case CriteriaMessage:
		fallthrough
	default:
		return record.Message
	}
}

func joinParts(parts ...string) string {
	return strings.Join(parts, "\x00")
}

func appendBucketSuffix(base string, bucket int) string {
	return fmt.Sprintf("%s%s%d", base, bucketSeparator, bucket)
}

func assignWindowBuckets(records []Record, keys []string, window time.Duration) []int {
	assignments := make([]int, len(records))
	type entry struct {
		idx       int
		timestamp time.Time
	}
	grouped := make(map[string][]entry)
	for i, key := range keys {
		grouped[key] = append(grouped[key], entry{idx: i, timestamp: parseTimestamp(records[i].Timestamp)})
	}
	for _, entries := range grouped {
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].timestamp.After(entries[j].timestamp)
		})
		bucketIndex := -1
		var bucketLatest time.Time
		for _, entry := range entries {
			if entry.timestamp.IsZero() {
				if bucketIndex == -1 {
					bucketIndex = 0
				}
				assignments[entry.idx] = bucketIndex
				continue
			}
			if bucketIndex == -1 {
				bucketIndex = 0
				bucketLatest = entry.timestamp
				assignments[entry.idx] = bucketIndex
				continue
			}
			diff := bucketLatest.Sub(entry.timestamp)
			if diff <= window {
				assignments[entry.idx] = bucketIndex
				continue
			}
			bucketIndex++
			bucketLatest = entry.timestamp
			assignments[entry.idx] = bucketIndex
		}
	}
	return assignments
}

func parseTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

// BucketFromKey returns the window bucket index encoded in the key, or -1 if none.
func BucketFromKey(key string) int {
	idx := strings.Index(key, bucketSeparator)
	if idx < 0 {
		return -1
	}
	value := key[idx+len(bucketSeparator):]
	bucket, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return bucket
}
