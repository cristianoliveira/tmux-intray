package format

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCountsByLevel(t *testing.T) {
	tests := []struct {
		name     string
		lines    string
		info     int
		warning  int
		error    int
		critical int
	}{
		{
			name:     "empty",
			lines:    "",
			info:     0,
			warning:  0,
			error:    0,
			critical: 0,
		},
		{
			name:  "single info",
			lines: "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage\t123\tinfo",
			info:  1,
		},
		{
			name: "mixed levels",
			lines: `1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage\t123\tinfo
2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\tmessage\t124\twarning
3\t2025-01-01T12:00:00Z\tactive\tsess2\twin2\tpane3\tmessage\t125\terror
4\t2025-01-01T13:00:00Z\tactive\tsess2\twin2\tpane4\tmessage\t126\tcritical
5\t2025-01-01T14:00:00Z\tactive\tsess3\twin3\tpane5\tmessage\t127\tinfo`,
			info:     2,
			warning:  1,
			error:    1,
			critical: 1,
		},
		{
			name:  "unknown level defaults to info",
			lines: "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage\t123\tunknown",
			info:  1,
		},
		{
			name:  "missing level field skips",
			lines: "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage\t123",
			info:  0,
		},
		{
			name:    "multiple lines with empty lines",
			lines:   "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage\t123\tinfo\n\n2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\tmessage\t124\twarning",
			info:    1,
			warning: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replace escaped tabs with actual tabs
			lines := strings.ReplaceAll(tt.lines, "\\t", "\t")
			info, warning, err, critical, parseErr := ParseCountsByLevel(lines)
			require.NoError(t, parseErr)
			assert.Equal(t, tt.info, info)
			assert.Equal(t, tt.warning, warning)
			assert.Equal(t, tt.error, err)
			assert.Equal(t, tt.critical, critical)
		})
	}
}

func TestParsePaneCounts(t *testing.T) {
	tests := []struct {
		name  string
		lines string
		panes map[string]int
	}{
		{
			name:  "empty",
			lines: "",
			panes: map[string]int{},
		},
		{
			name:  "single pane",
			lines: "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage\t123\tinfo",
			panes: map[string]int{"sess1:win1:pane1": 1},
		},
		{
			name: "multiple panes",
			lines: `1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage\t123\tinfo
2\t2025-01-01T11:00:00Z\tactive\tsess1\twin1\tpane2\tmessage\t124\twarning
3\t2025-01-01T12:00:00Z\tactive\tsess2\twin2\tpane3\tmessage\t125\terror
4\t2025-01-01T13:00:00Z\tactive\tsess2\twin2\tpane4\tmessage\t126\tcritical
5\t2025-01-01T14:00:00Z\tactive\tsess3\twin3\tpane5\tmessage\t127\tinfo`,
			panes: map[string]int{
				"sess1:win1:pane1": 1,
				"sess1:win1:pane2": 1,
				"sess2:win2:pane3": 1,
				"sess2:win2:pane4": 1,
				"sess3:win3:pane5": 1,
			},
		},
		{
			name:  "missing pane fields skips",
			lines: "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\t\tmessage\t123\tinfo",
			panes: map[string]int{"sess1:win1:": 1},
		},
		{
			name:  "empty session/window/pane",
			lines: "1\t2025-01-01T10:00:00Z\tactive\t\t\t\tmessage\t123\tinfo",
			panes: map[string]int{"::": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.ReplaceAll(tt.lines, "\\t", "\t")
			panes := ParsePaneCounts(lines)
			assert.Equal(t, tt.panes, panes)
		})
	}
}

func TestFormatSummary(t *testing.T) {
	tests := []struct {
		name     string
		active   int
		info     int
		warning  int
		error    int
		critical int
		expected string
	}{
		{
			name:     "zero active",
			active:   0,
			expected: "No active notifications\n",
		},
		{
			name:     "active with all levels",
			active:   10,
			info:     2,
			warning:  3,
			error:    4,
			critical: 1,
			expected: "Active notifications: 10\n  info: 2, warning: 3, error: 4, critical: 1\n",
		},
		{
			name:     "active with some zero levels",
			active:   5,
			info:     5,
			warning:  0,
			error:    0,
			critical: 0,
			expected: "Active notifications: 5\n  info: 5, warning: 0, error: 0, critical: 0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := FormatSummary(&buf, tt.active, tt.info, tt.warning, tt.error, tt.critical)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestFormatLevels(t *testing.T) {
	tests := []struct {
		name     string
		info     int
		warning  int
		error    int
		critical int
		expected string
	}{
		{
			name:     "all zero",
			expected: "info:0\nwarning:0\nerror:0\ncritical:0\n",
		},
		{
			name:     "mixed",
			info:     2,
			warning:  3,
			error:    4,
			critical: 1,
			expected: "info:2\nwarning:3\nerror:4\ncritical:1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := FormatLevels(&buf, tt.info, tt.warning, tt.error, tt.critical)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestFormatPanes(t *testing.T) {
	tests := []struct {
		name     string
		counts   map[string]int
		expected string // lines sorted alphabetically
	}{
		{
			name:     "empty",
			counts:   map[string]int{},
			expected: "",
		},
		{
			name:     "single pane",
			counts:   map[string]int{"sess1:win1:pane1": 5},
			expected: "sess1:win1:pane1:5\n",
		},
		{
			name: "multiple panes sorted",
			counts: map[string]int{
				"sess2:win2:pane2": 2,
				"sess1:win1:pane1": 1,
				"sess3:win3:pane3": 3,
			},
			expected: "sess1:win1:pane1:1\nsess2:win2:pane2:2\nsess3:win3:pane3:3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := FormatPanes(&buf, tt.counts)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		name     string
		active   int
		info     int
		warning  int
		error    int
		critical int
		panes    map[string]int
		expected string
	}{
		{
			name:     "empty",
			active:   0,
			panes:    map[string]int{},
			expected: `{"active":0,"info":0,"warning":0,"error":0,"critical":0,"panes":{}}` + "\n",
		},
		{
			name:     "with data",
			active:   10,
			info:     2,
			warning:  3,
			error:    4,
			critical: 1,
			panes:    map[string]int{"sess1:win1:pane1": 5, "sess2:win2:pane2": 3},
			expected: `{"active":10,"info":2,"warning":3,"error":4,"critical":1,"panes":{"sess1:win1:pane1":5,"sess2:win2:pane2":3}}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := FormatJSON(&buf, tt.active, tt.info, tt.warning, tt.error, tt.critical, tt.panes)
			require.NoError(t, err)
			// JSON output order of map keys is nondeterministic; we need to parse and compare
			var got, expected map[string]interface{}
			err = json.Unmarshal(buf.Bytes(), &got)
			require.NoError(t, err)
			err = json.Unmarshal([]byte(tt.expected), &expected)
			require.NoError(t, err)
			assert.Equal(t, expected, got)
		})
	}
}
