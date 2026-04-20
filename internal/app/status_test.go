package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStatusPresetLookup struct {
	calls []string
}

func (f *fakeStatusPresetLookup) Lookup(name string) (string, bool) {
	f.calls = append(f.calls, name)
	if name == "compact" {
		return "[{{unread-count}}] {{latest-message}}", true
	}
	return "", false
}

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
	listNotificationsResult string
	listNotificationsErr    error
}

func (f *fakeStatusClient) EnsureTmuxRunning() bool {
	f.ensureCalls++
	return f.ensureTmuxRunningResult
}

func (f *fakeStatusClient) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
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

func TestDetermineStatusFormat(t *testing.T) {
	tests := []struct {
		name        string
		formatFlag  string
		envFormat   string
		flagChanged bool
		want        string
	}{
		{name: "flag wins when changed", formatFlag: "compact", envFormat: "json", flagChanged: true, want: "compact"},
		{name: "env used when flag unchanged", formatFlag: "compact", envFormat: "json", flagChanged: false, want: "json"},
		{name: "default compact when empty", formatFlag: "", envFormat: "", flagChanged: false, want: "compact"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineStatusFormat(tt.formatFlag, tt.envFormat, tt.flagChanged)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStatusUseCaseExecuteCompactPreset(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: true, listNotificationsResult: statusMockLines()}
	useCase := NewStatusUseCase(client, nil)
	var buf bytes.Buffer

	err := useCase.Execute("compact", &buf)
	require.NoError(t, err)
	assert.Equal(t, "[4] message one\n", buf.String())
}

func TestStatusUseCaseExecuteDetailedPreset(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: true, listNotificationsResult: statusMockLines()}
	useCase := NewStatusUseCase(client, nil)
	var buf bytes.Buffer

	err := useCase.Execute("detailed", &buf)
	require.NoError(t, err)
	assert.Equal(t, "4 unread, 1 read | Latest: message one\n", buf.String())
}

func TestStatusUseCaseExecuteLegacyFormats(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{format: "summary", want: "Active notifications: 4\n  info: 2, warning: 1, error: 0, critical: 1\n"},
		{format: "levels", want: "info:2\nwarning:1\nerror:0\ncritical:1\n"},
		{format: "panes", want: "sess1:win1:pane1:1\nsess1:win1:pane2:1\nsess2:win2:pane4:1\nsess3:win3:pane5:1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			client := &fakeStatusClient{ensureTmuxRunningResult: true, listNotificationsResult: statusMockLines()}
			useCase := NewStatusUseCase(client, nil)
			var buf bytes.Buffer

			err := useCase.Execute(tt.format, &buf)
			require.NoError(t, err)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestStatusUseCaseExecuteCustomTemplate(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: true, listNotificationsResult: statusMockLines()}
	useCase := NewStatusUseCase(client, nil)
	var buf bytes.Buffer

	err := useCase.Execute("{{critical-count}}|{{unread-count}}|{{latest-message}}", &buf)
	require.NoError(t, err)
	assert.Equal(t, "1|4|message one\n", buf.String())
}

func TestStatusUseCaseExecuteTmuxNotRunning(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: false}
	useCase := NewStatusUseCase(client, nil)
	var buf bytes.Buffer

	err := useCase.Execute("compact", &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tmux not running")
	assert.Equal(t, 1, client.ensureCalls)
}

func TestStatusUseCaseExecuteUsesInjectedPresetLookup(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: true, listNotificationsResult: statusMockLines()}
	lookup := &fakeStatusPresetLookup{}
	useCase := NewStatusUseCase(client, lookup.Lookup)
	var buf bytes.Buffer

	err := useCase.Execute("compact", &buf)
	require.NoError(t, err)
	assert.Equal(t, []string{"compact"}, lookup.calls)
	assert.Equal(t, "[4] message one\n", buf.String())
}

func TestStatusCountHelpers(t *testing.T) {
	client := &fakeStatusClient{ensureTmuxRunningResult: true, listNotificationsResult: statusMockLines()}

	assert.Equal(t, 4, CountByState(client, "active"))
	assert.Equal(t, 1, CountByState(client, "dismissed"))

	info, warning, errCount, critical := CountByLevel(client)
	assert.Equal(t, 2, info)
	assert.Equal(t, 1, warning)
	assert.Equal(t, 0, errCount)
	assert.Equal(t, 1, critical)

	panes := PaneCounts(client)
	assert.Len(t, panes, 4)
	assert.Equal(t, 1, panes["sess1:win1:pane1"])
	assert.Equal(t, 1, panes["sess3:win3:pane5"])
}
