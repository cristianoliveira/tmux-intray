package status

import (
	"testing"
)

type fakeStatusPanelClient struct {
	ensureTmuxRunningResult bool
	activeCount             int
	listNotificationsResult string
	configBoolValues        map[string]bool
	configStringValues      map[string]string
}

func (f *fakeStatusPanelClient) EnsureTmuxRunning() bool {
	return f.ensureTmuxRunningResult
}

func (f *fakeStatusPanelClient) GetActiveCount() int {
	return f.activeCount
}

func (f *fakeStatusPanelClient) ListNotifications(stateFilter string) string {
	return f.listNotificationsResult
}

func (f *fakeStatusPanelClient) GetConfigBool(key string, defaultValue bool) bool {
	if f.configBoolValues != nil {
		if v, ok := f.configBoolValues[key]; ok {
			return v
		}
	}
	return defaultValue
}

func (f *fakeStatusPanelClient) GetConfigString(key, defaultValue string) string {
	if f.configStringValues != nil {
		if v, ok := f.configStringValues[key]; ok {
			return v
		}
	}
	return defaultValue
}

func TestRunStatusPanelDisabled(t *testing.T) {
	client := &fakeStatusPanelClient{
		ensureTmuxRunningResult: true,
		activeCount:             5,
	}

	opts := StatusPanelOptions{
		Format:  "compact",
		Enabled: false,
	}
	output, err := RunStatusPanel(client, opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if output != "" {
		t.Errorf("Expected empty output when disabled, got %q", output)
	}
}

func TestRunStatusPanelTmuxNotRunning(t *testing.T) {
	client := &fakeStatusPanelClient{
		ensureTmuxRunningResult: false,
	}

	opts := StatusPanelOptions{
		Format:  "compact",
		Enabled: true,
	}
	output, err := RunStatusPanel(client, opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if output != "" {
		t.Errorf("Expected empty output when tmux not running, got %q", output)
	}
}

func TestRunStatusPanelNoActiveNotifications(t *testing.T) {
	client := &fakeStatusPanelClient{
		ensureTmuxRunningResult: true,
		activeCount:             0,
	}

	opts := StatusPanelOptions{
		Format:  "compact",
		Enabled: true,
	}
	output, err := RunStatusPanel(client, opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if output != "" {
		t.Errorf("Expected empty output when no active notifications, got %q", output)
	}
}

func TestRunStatusPanelCompactFormat(t *testing.T) {
	client := &fakeStatusPanelClient{
		ensureTmuxRunningResult: true,
		activeCount:             3,
		listNotificationsResult: "1\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\tmsg1\t1234567890\tinfo\n" +
			"2\t2025-02-04T10:01:00Z\tactive\t$0\t%0\t:0.0\tmsg2\t1234567890\tinfo\n" +
			"3\t2025-02-04T10:02:00Z\tactive\t$0\t%0\t:0.0\tmsg3\t1234567890\twarning",
		configStringValues: map[string]string{
			"level_colors": "info:green,warning:yellow,error:red,critical:magenta",
		},
	}

	opts := StatusPanelOptions{
		Format:  "compact",
		Enabled: true,
	}
	output, err := RunStatusPanel(client, opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	// Highest severity is warning, color yellow
	expected := "#[fg=yellow]🔔 3#[default]"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestRunStatusPanelCompactFormatNoColor(t *testing.T) {
	client := &fakeStatusPanelClient{
		ensureTmuxRunningResult: true,
		activeCount:             1,
		listNotificationsResult: "1\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\tmsg1\t1234567890\tinfo",
		configStringValues: map[string]string{
			"level_colors": "", // Empty color config
		},
	}

	opts := StatusPanelOptions{
		Format:  "compact",
		Enabled: true,
	}
	output, err := RunStatusPanel(client, opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := "🔔 1"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestRunStatusPanelDetailedFormat(t *testing.T) {
	client := &fakeStatusPanelClient{
		ensureTmuxRunningResult: true,
		activeCount:             4,
		listNotificationsResult: "1\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\tmsg1\t1234567890\tinfo\n" +
			"2\t2025-02-04T10:01:00Z\tactive\t$0\t%0\t:0.0\tmsg2\t1234567890\twarning\n" +
			"3\t2025-02-04T10:02:00Z\tactive\t$0\t%0\t:0.0\tmsg3\t1234567890\terror\n" +
			"4\t2025-02-04T10:03:00Z\tactive\t$0\t%0\t:0.0\tmsg4\t1234567890\tcritical",
		configStringValues: map[string]string{
			"level_colors": "info:green,warning:yellow,error:red,critical:magenta",
		},
	}

	opts := StatusPanelOptions{
		Format:  "detailed",
		Enabled: true,
	}
	output, err := RunStatusPanel(client, opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	// Expect order: info, warning, error, critical
	expected := "#[fg=green]i:1#[default] #[fg=yellow]w:1#[default] #[fg=red]e:1#[default] #[fg=magenta]c:1#[default]"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestRunStatusPanelCountOnlyFormat(t *testing.T) {
	client := &fakeStatusPanelClient{
		ensureTmuxRunningResult: true,
		activeCount:             7,
	}

	opts := StatusPanelOptions{
		Format:  "count-only",
		Enabled: true,
	}
	output, err := RunStatusPanel(client, opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := "7"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestRunStatusPanelUnknownFormat(t *testing.T) {
	client := &fakeStatusPanelClient{
		ensureTmuxRunningResult: true,
		activeCount:             1, // Non-zero to proceed to format check
	}

	opts := StatusPanelOptions{
		Format:  "invalid",
		Enabled: true,
	}
	_, err := RunStatusPanel(client, opts)
	if err == nil {
		t.Error("Expected error for unknown format")
	}
	if err.Error() != "unknown format: invalid" {
		t.Errorf("Expected 'unknown format: invalid', got %v", err)
	}
}
