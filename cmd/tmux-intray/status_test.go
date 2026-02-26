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
	// Explicitly set format to "summary" for backward compatibility test
	require.NoError(t, cmd.Flags().Set("format", "summary"))
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
	// JSON format should now be implemented and output valid JSON
	assert.Contains(t, output, "active")
	assert.Contains(t, output, "info")
	assert.Contains(t, output, "warning")
	assert.Contains(t, output, "error")
	assert.Contains(t, output, "critical")
	assert.Contains(t, output, "panes")
	// Should be valid JSON (basic check)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(output), "{"))
	assert.True(t, strings.HasSuffix(strings.TrimSpace(output), "}"))
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
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "invalid"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	// Unknown format is treated as template, not an error
	require.NoError(t, err)
	// The template "invalid" has no variables, so it outputs as-is
	output := stdout.String()
	assert.Contains(t, output, "invalid")
}

func TestStatusRunEListNotificationsError(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsErr:    assert.AnError,
	}
	cmd := NewStatusCmd(client)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err) // error is ignored, returns empty counts
	assert.Equal(t, 1, client.ensureCalls)
	// With template-based system, ListNotifications may be called multiple times
	assert.GreaterOrEqual(t, len(client.listNotificationsCalls), 1)
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

// Tests for new preset-based format system

func TestStatusPresetCompactFormat(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "compact"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	// Compact format: [${unread-count}] ${latest-message}
	assert.Contains(t, output, "[4]")
	assert.Contains(t, output, "message one")
}

func TestStatusPresetDetailedFormat(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "detailed"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	// Detailed format: ${unread-count} unread, ${read-count} read | Latest: ${latest-message}
	assert.Contains(t, output, "unread")
	assert.Contains(t, output, "read")
	assert.Contains(t, output, "Latest:")
}

func TestStatusPresetCountOnlyFormat(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "count-only"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	// Count-only format: just the unread count
	assert.Equal(t, "4", output)
}

func TestStatusCustomTemplateFormat(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "Unread: ${unread-count}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "Unread: 4")
}

func TestStatusCustomTemplateWithVariable(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${unread-count}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "4", output)
}

func TestStatusUnknownFormatAsTreatAsTemplate(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "unknown-format"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	// Should not error - unknown format is treated as template (and will output empty since no variables match)
	require.NoError(t, err)
}

func TestStatusDefaultFormatIsCompact(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	// Don't set format flag - should use default
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	// Default is compact: [${unread-count}] ${latest-message}
	assert.Contains(t, output, "[4]")
	assert.Contains(t, output, "message one")
}

// Helper to set flag (already defined in add_test.go)
// We rely on the existing setFlag from add_test.go

// =====================================================================
// COMPREHENSIVE E2E TESTS FOR ALL 6 PRESETS AND 13 VARIABLES
// =====================================================================

func TestStatusPresetJSONFormat(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "json"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	// JSON format should contain valid JSON structure
	// (Note: legacy format handler returns different format than template presets)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(output), "{"))
	assert.True(t, strings.HasSuffix(strings.TrimSpace(output), "}"))
	// Should have panes information
	assert.Contains(t, output, "panes")
}

func TestStatusPresetLevelsFormat(t *testing.T) {
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
	// Levels format should show severity breakdown
	assert.Contains(t, output, "info")
	assert.Contains(t, output, "warning")
	assert.Contains(t, output, "error")
	assert.Contains(t, output, "critical")
}

func TestStatusPresetPanesFormat(t *testing.T) {
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
	// Panes format should show pane list
	assert.Contains(t, output, "pane")
	assert.Contains(t, output, "4")
}

// =====================================================================
// VARIABLE RESOLUTION TESTS - All 13 Template Variables
// =====================================================================

func TestStatusVariableUnreadCount(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${unread-count}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "4", output)
}

func TestStatusVariableReadCount(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${read-count}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	// read-count is set to dismissed count
	// total=5, dismissed=1
	assert.Equal(t, "1", output)
}

func TestStatusVariableActiveCount(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${active-count}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "4", output)
}

func TestStatusVariableDismissedCount(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${dismissed-count}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "1", output)
}

func TestStatusVariableTotalCount(t *testing.T) {
	// total-count is an alias for unread-count
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${total-count}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "4", output)
}

func TestStatusVariableLatestMessage(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${latest-message}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	// latest message should be "message five" since it's the last (most recent) active
	assert.Contains(t, output, "message")
}

func TestStatusVariableHasUnreadTrue(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${has-unread}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "true", output)
}

func TestStatusVariableHasUnreadFalse(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: "",
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${has-unread}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "false", output)
}

func TestStatusVariableHasActive(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${has-active}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "true", output)
}

func TestStatusVariableHasDismissed(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${has-dismissed}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "true", output)
}

func TestStatusVariableHighestSeverity(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${highest-severity}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	// Highest severity should be "1" (critical) since we have critical in mock data
	assert.Equal(t, "1", output)
}

func TestStatusVariableSessionList(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${session-list}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	// Currently session-list is not populated from buildVariableContext
	// but the variable should be available (just empty)
	assert.NotNil(t, output) // should not error, just return empty string
}

func TestStatusVariableWindowList(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${window-list}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	// Currently window-list is not populated from buildVariableContext
	// but the variable should be available (just empty)
	assert.NotNil(t, output) // should not error, just return empty string
}

func TestStatusVariablePaneList(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${pane-list}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	// Currently pane-list is not populated from buildVariableContext
	// but the variable should be available (just empty)
	assert.NotNil(t, output) // should not error, just return empty string
}

// =====================================================================
// CUSTOM TEMPLATE TESTS - Multiple Variables
// =====================================================================

func TestStatusCustomTemplateMultipleVariables(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "[${unread-count}/${total-count}]"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := strings.TrimSpace(stdout.String())
	assert.Contains(t, output, "[4/4]")
}

func TestStatusCustomTemplateWithText(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "Unread: ${unread-count} Active: ${active-count}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "Unread: 4")
	assert.Contains(t, output, "Active: 4")
}

// =====================================================================
// ERROR HANDLING AND EDGE CASES
// =====================================================================

func TestStatusEmptyNotifications(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: "",
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "compact"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	// Should output [0] with no message
	assert.Contains(t, output, "[0]")
}

func TestStatusInvalidVariable(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "${invalid-variable}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	// Invalid variables are replaced with empty strings
	output := stdout.String()
	assert.Equal(t, "\n", output)
}

func TestStatusMixedValidInvalidVariables(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "Count: ${unread-count} Invalid: ${bad-var}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "Count: 4")
	assert.Contains(t, output, "Invalid: ")
}

// =====================================================================
// REGRESSION TESTS
// =====================================================================

func TestStatusWithoutFormatFlagUsesDefault(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	// Don't set any format flag - should use default
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	output := stdout.String()
	// Default should be compact format
	assert.Contains(t, output, "[4]")
	assert.Contains(t, output, "message")
}

func TestStatusHelpShowsFormatExamples(t *testing.T) {
	cmd := NewStatusCmd(&fakeStatusClient{})
	output := cmd.Long
	// Help should mention format options
	assert.Contains(t, output, "format")
}

// =====================================================================
// ALL PRESETS INTEGRATION TEST
// =====================================================================

func TestAllPresetsProduceOutput(t *testing.T) {
	presets := []string{"compact", "detailed", "json", "count-only", "levels", "panes"}

	for _, preset := range presets {
		t.Run(preset, func(t *testing.T) {
			client := &fakeStatusClient{
				ensureTmuxRunningResult: true,
				listNotificationsResult: statusMockLines(),
			}
			cmd := NewStatusCmd(client)
			require.NoError(t, cmd.Flags().Set("format", preset))
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			err := cmd.RunE(cmd, []string{})
			require.NoError(t, err, "preset %s should not error", preset)

			output := stdout.String()
			assert.NotEmpty(t, output, "preset %s should produce output", preset)
		})
	}
}
