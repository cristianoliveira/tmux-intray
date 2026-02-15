package dedup

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildKeysCriteria(t *testing.T) {
	records := []Record{
		{Message: "disk full", Level: "error", Session: "$0", Window: "@0", Pane: "%0"},
		{Message: "disk full", Level: "warning", Session: "$1", Window: "@1", Pane: "%1"},
	}

	keys := BuildKeys(records, Options{Criteria: CriteriaMessage})
	require.Equal(t, []string{"disk full", "disk full"}, keys)

	keys = BuildKeys(records, Options{Criteria: CriteriaMessageLevel})
	require.Equal(t, []string{"disk full\x00error", "disk full\x00warning"}, keys)

	keys = BuildKeys(records, Options{Criteria: CriteriaMessageSource})
	require.Equal(t, []string{"disk full\x00$0\x00@0\x00%0", "disk full\x00$1\x00@1\x00%1"}, keys)
}

func TestBuildKeysWindow(t *testing.T) {
	records := []Record{
		{Message: "alert", Timestamp: "2024-01-01T10:00:00Z"},
		{Message: "alert", Timestamp: "2024-01-01T09:55:00Z"},
		{Message: "alert", Timestamp: "2024-01-01T09:30:00Z"},
	}

	keys := BuildKeys(records, Options{Criteria: CriteriaMessage, Window: 15 * time.Minute})
	require.Len(t, keys, 3)
	require.Equal(t, "alert", keys[0])
	require.Equal(t, "alert", keys[1])
	require.Equal(t, -1, BucketFromKey(keys[0]))
	require.Equal(t, -1, BucketFromKey(keys[1]))
	require.True(t, strings.Contains(keys[2], "\u001f"))
	require.Equal(t, 1, BucketFromKey(keys[2]))
	require.Equal(t, "alert", StripBucketSuffix(keys[2]))

	keys = BuildKeys(records, Options{Criteria: CriteriaMessage, Window: 5 * time.Minute})
	require.Equal(t, -1, BucketFromKey(keys[0]))
	require.Equal(t, -1, BucketFromKey(keys[1]))
	require.Equal(t, 1, BucketFromKey(keys[2]))
}

func TestStripBucketSuffixWithoutSuffix(t *testing.T) {
	require.Equal(t, "hello", StripBucketSuffix("hello"))
	require.Equal(t, "hello", StripBucketSuffix("hello\u001f1"))
}
