package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n"), f.listNotificationsErr
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

func TestStatusRunEDefaultCompactPreset(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "[4] message one\n", stdout.String())
	assert.Equal(t, 1, client.ensureCalls)
}

func TestStatusRunELegacySummaryAliasMapsToCompact(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "summary"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "Active notifications: 4\n  info: 2, warning: 1, error: 0, critical: 1\n", stdout.String())
}

func TestStatusRunEPresetFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
		assert func(t *testing.T, output string)
	}{
		{
			name:   "compact",
			format: "compact",
			assert: func(t *testing.T, output string) {
				assert.Equal(t, "[4] message one\n", output)
			},
		},
		{
			name:   "detailed",
			format: "detailed",
			assert: func(t *testing.T, output string) {
				assert.Equal(t, "4 unread, 1 read | Latest: message one\n", output)
			},
		},
		{
			name:   "json",
			format: "json",
			assert: func(t *testing.T, output string) {
				assert.Equal(t, "{\n  \"active\": 4,\n  \"info\": 2,\n  \"warning\": 1,\n  \"error\": 0,\n  \"critical\": 1,\n  \"panes\": {\n    \"sess1:win1:pane1\": 1,\n    \"sess1:win1:pane2\": 1,\n    \"sess2:win2:pane4\": 1,\n    \"sess3:win3:pane5\": 1\n  }\n}\n", output)
			},
		},
		{
			name:   "count-only",
			format: "count-only",
			assert: func(t *testing.T, output string) {
				assert.Equal(t, "4\n", output)
			},
		},
		{
			name:   "levels",
			format: "levels",
			assert: func(t *testing.T, output string) {
				assert.Equal(t, "info:2\nwarning:1\nerror:0\ncritical:1\n", output)
			},
		},
		{
			name:   "panes",
			format: "panes",
			assert: func(t *testing.T, output string) {
				assert.Equal(t, "sess1:win1:pane1:1\nsess1:win1:pane2:1\nsess2:win2:pane4:1\nsess3:win3:pane5:1\n", output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeStatusClient{
				ensureTmuxRunningResult: true,
				listNotificationsResult: statusMockLines(),
			}
			cmd := NewStatusCmd(client)
			require.NoError(t, cmd.Flags().Set("format", tt.format))
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			err := cmd.RunE(cmd, []string{})
			require.NoError(t, err)
			tt.assert(t, stdout.String())
		})
	}
}

func TestStatusRunECustomTemplateVariables(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "{{critical-count}}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "1\n", stdout.String())
}

func TestStatusRunEMixedCustomTemplateSyntax(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "critical={{critical-count}} unread={{.UnreadCount}}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "critical=1 unread={{.UnreadCount}}\n", stdout.String())
}

func TestStatusRunEResolvesAllTemplateVariables(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "{{unread-count}}|{{total-count}}|{{read-count}}|{{active-count}}|{{dismissed-count}}|{{latest-message}}|{{has-unread}}|{{has-active}}|{{has-dismissed}}|{{highest-severity}}|{{session-list}}|{{window-list}}|{{pane-list}}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "4|4|1|4|1|message one|true|true|true|1|||\n", stdout.String())
}

func TestStatusRunEBooleanVariablesRenderFalseLiterals(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "{{has-unread}}|{{has-active}}|{{has-dismissed}}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "false|false|false\n", stdout.String())
}

func TestStatusRunEPreservesTmuxColorCodes(t *testing.T) {
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "#[fg=red]{{critical-count}}#[default]"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "#[fg=red]1#[default]\n", stdout.String())
}

func TestStatusRunEInvalidVariableReturnsHelpfulError(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: true}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "{{unknown-var}}"))

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
}

func TestStatusRunEInvalidVariableNameReturnsError(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: true}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "{{critical_count}}"))

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
}

func TestStatusRunEUnknownPresetReturnsHelpfulError(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: true}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "not-a-preset"))

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
}

func TestStatusRunETmuxNotRunning(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: false}
	cmd := NewStatusCmd(client)

	err := cmd.RunE(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tmux not running")
	assert.Equal(t, 1, client.ensureCalls)
}

func TestStatusRunEEnvironmentFormatOverride(t *testing.T) {
	t.Setenv("TMUX_INTRAY_STATUS_FORMAT", "{{unread-count}}")
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "4\n", stdout.String())
}

func TestStatusRunEFlagTakesPrecedenceOverEnvironment(t *testing.T) {
	t.Setenv("TMUX_INTRAY_STATUS_FORMAT", "{{critical-count}}")
	client := &fakeStatusClient{
		ensureTmuxRunningResult: true,
		listNotificationsResult: statusMockLines(),
	}
	cmd := NewStatusCmd(client)
	require.NoError(t, cmd.Flags().Set("format", "{{unread-count}}"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "4\n", stdout.String())
}

func TestStatusHelpIncludesTemplateExamples(t *testing.T) {
	cmd := NewStatusCmd(&fakeStatusClient{ensureTmuxRunningResult: true})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Help()
	require.NoError(t, err)
	help := stdout.String()

	assert.Contains(t, help, "PRESETS (6):")
	assert.Contains(t, help, "{{unread-count}}")
	assert.Contains(t, help, "{{critical-count}}")
	assert.Contains(t, help, "tmux-intray status --format='{{unread-count}} new messages'")
}

func TestStatusHelperEdgeCases(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: true}

	client.listNotificationsResult = "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage one\t123\tunknown"
	info, warning, errCount, critical := countByLevel(client)
	require.Equal(t, 1, info)
	require.Equal(t, 0, warning)
	require.Equal(t, 0, errCount)
	require.Equal(t, 0, critical)

	client.listNotificationsResult = "1\t2025-01-01T10:00:00Z\tactive\tsess1\twin1\tpane1\tmessage one"
	info, warning, errCount, critical = countByLevel(client)
	require.Equal(t, 0, info)
	require.Equal(t, 0, warning)
	require.Equal(t, 0, errCount)
	require.Equal(t, 0, critical)

	client.listNotificationsResult = "1\t2025-01-01T10:00:00Z\tactive\tsess1"
	panes := paneCounts(client)
	require.Empty(t, panes)

	client.listNotificationsResult = "1\t2025-01-01T10:00:00Z\tactive\t\t\t\tmessage\t123\tinfo"
	panes = paneCounts(client)
	require.Len(t, panes, 1)
	count, ok := panes["::"]
	require.True(t, ok)
	require.Equal(t, 1, count)
}

func TestStatusMockLinesTimestampsAreISO8601(t *testing.T) {
	for _, line := range strings.Split(statusMockLines(), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		require.GreaterOrEqual(t, len(fields), 2)
		_, err := time.Parse(time.RFC3339, fields[1])
		require.NoError(t, err)
	}
}
