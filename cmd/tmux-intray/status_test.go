package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statusMockLines returns a fixed TSV string for testing.
func statusMockLines() string {
	return `1	2025-01-01T10:00:00Z	active	sess1	win1	pane1	message one	123	info
2	2025-01-01T11:00:00Z	active	sess1	win1	pane2	message two	124	warning
3	2025-01-01T12:00:00Z	dismissed	sess2	win2	pane3	message three	125	error
4	2025-01-01T13:00:00Z	active	sess2	win2	pane4	message four	126	info
5	2025-01-01T14:00:00Z	active	sess3	win3	pane5	message five	127	critical`
}

type fakeStatusClient struct {
	ensureTmuxRunningResult bool
	ensureCalls             int
	listNotificationsCalls  []struct {
		stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter string
		olderThanCutoff, newerThanCutoff, readFilter                      string
	}
	listNotificationsResult string
	listNotificationsErr    error
	getActiveCountCalls     int
	getActiveCountResult    int
}

func (f *fakeStatusClient) EnsureTmuxRunning() bool {
	f.ensureCalls++
	return f.ensureTmuxRunningResult
}

func (f *fakeStatusClient) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	f.listNotificationsCalls = append(f.listNotificationsCalls, struct {
		stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter string
		olderThanCutoff, newerThanCutoff, readFilter                      string
	}{stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter})
	// Filter lines based on stateFilter (simple implementation for tests)
	lines := strings.Split(f.listNotificationsResult, "\n")
	var filtered []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if stateFilter != "" && stateFilter != "all" && len(fields) > 2 && fields[2] != stateFilter {
			continue
		}
		// ignore other filters for simplicity
		filtered = append(filtered, line)
	}
	result := strings.Join(filtered, "\n")
	return result, f.listNotificationsErr
}

func (f *fakeStatusClient) GetActiveCount() int {
	f.getActiveCountCalls++
	return f.getActiveCountResult
}

func TestNewStatusCmdPanicsWhenClientIsNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got nil")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected panic message as string, got %T", r)
		}
		if !strings.Contains(msg, "client dependency cannot be nil") {
			t.Fatalf("expected panic message to mention nil dependency, got %q", msg)
		}
	}()

	NewStatusCmd(nil)
}

func TestStatusRunESummaryFormat(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, 1, client.ensureCalls)
	// ListNotifications should be called twice: once for countByState, once for countByLevel
	assert.GreaterOrEqual(t, len(client.listNotificationsCalls), 2)
	// GetActiveCount not used in summary format
	assert.Equal(t, 0, client.getActiveCountCalls)

	output := stdout.String()
	assert.Contains(t, output, "Active notifications: 4")
	assert.Contains(t, output, "info: 2, warning: 1, error: 0, critical: 1")
}

func TestStatusRunELevelsFormat(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "levels"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	expected := "info:2\nwarning:1\nerror:0\ncritical:1\n"
	assert.Equal(t, expected, output)
}

func TestStatusRunEPanesFormat(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "panes"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 4) // four unique pane keys
	assert.Contains(t, output, "sess1:win1:pane1:1")
	assert.Contains(t, output, "sess1:win1:pane2:1")
	assert.Contains(t, output, "sess2:win2:pane4:1")
	assert.Contains(t, output, "sess3:win3:pane5:1")
}

func TestStatusRunEJSONFormatNotImplemented(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "json"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "JSON format not yet implemented")
}

func TestStatusRunETmuxNotRunning(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: false,
	}
	cmd := NewStatusCmd(client)

	err := cmd.RunE(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tmux not running")
	assert.Equal(t, 1, client.ensureCalls)
}

func TestStatusRunEEnvironmentFormatOverride(t *testing.T) {
	t.Setenv("TMUX_INTRAY_STATUS_FORMAT", "levels")
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "info:2")
	// flag not changed, environment used
}

func TestStatusRunEInvalidFormat(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "invalid"))

	err := cmd.RunE(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown format")
}

func TestStatusRunEListNotificationsError(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsErr:    assert.AnError,
	}
	cmd := NewStatusCmd(client)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err) // error is ignored, returns empty counts
	assert.Equal(t, 1, client.ensureCalls)
	assert.Len(t, client.listNotificationsCalls, 1) // called once (countByState returns 0, early exit)
}

func TestStatusHelperEdgeCases(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
	}
	// Test countByLevel with unknown level
	client.listNotificationsResult = "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage one\t123\tunknown"
	info, warning, err, critical := countByLevel(client)
	require.Equal(t, 1, info) // default case increments info
	require.Equal(t, 0, warning)
	require.Equal(t, 0, err)
	require.Equal(t, 0, critical)

	// Test countByLevel with fields length <=8 (skip)
	client.listNotificationsResult = "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage one"
	info, warning, err, critical = countByLevel(client)
	require.Equal(t, 0, info)
	require.Equal(t, 0, warning)
	require.Equal(t, 0, err)
	require.Equal(t, 0, critical)

	// Test paneCounts with fields length <=5 (skip)
	client.listNotificationsResult = "1\t2025-01-01T10:00:00Z\tactive\tsess1"
	panes := paneCounts(client)
	require.Empty(t, panes)

	// Test paneCounts with empty session/window/pane (still counts)
	client.listNotificationsResult = "1\t2025-01-01T10:00:00Z\tactive\t\t\t\tmessage\t123\tinfo"
	panes = paneCounts(client)
	require.Len(t, panes, 1)
	key := "::"
	count, ok := panes[key]
	require.True(t, ok)
	require.Equal(t, 1, count)
}

func TestStatusRunEGetActiveCountError(t *testing.T) {
	// GetActiveCount returns int, cannot error; ignore
}

// Helper to set flag (already defined in add_test.go)
// We rely on the existing setFlag from add_test.go
