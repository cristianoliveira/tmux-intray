package cmd

import (
	"errors"
	"testing"
)

func TestRunDisabled(t *testing.T) {
	originalEnsureTmuxRunningFunc := statusPanelEnsureTmuxRunningFunc
	originalGetActiveCountFunc := statusPanelGetActiveCountFunc
	defer func() {
		statusPanelEnsureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		statusPanelGetActiveCountFunc = originalGetActiveCountFunc
	}()

	statusPanelEnsureTmuxRunningFunc = func() bool { return true }
	statusPanelGetActiveCountFunc = func() int { return 5 }

	opts := StatusPanelOptions{
		Format:  "compact",
		Enabled: false,
	}
	output, err := Run(opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if output != "" {
		t.Errorf("Expected empty output when disabled, got %q", output)
	}
}

func TestRunTmuxNotRunning(t *testing.T) {
	originalEnsureTmuxRunningFunc := statusPanelEnsureTmuxRunningFunc
	defer func() { statusPanelEnsureTmuxRunningFunc = originalEnsureTmuxRunningFunc }()

	statusPanelEnsureTmuxRunningFunc = func() bool { return false }

	opts := StatusPanelOptions{
		Format:  "compact",
		Enabled: true,
	}
	output, err := Run(opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if output != "" {
		t.Errorf("Expected empty output when tmux not running, got %q", output)
	}
}

func TestRunNoActiveNotifications(t *testing.T) {
	originalEnsureTmuxRunningFunc := statusPanelEnsureTmuxRunningFunc
	originalGetActiveCountFunc := statusPanelGetActiveCountFunc
	defer func() {
		statusPanelEnsureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		statusPanelGetActiveCountFunc = originalGetActiveCountFunc
	}()

	statusPanelEnsureTmuxRunningFunc = func() bool { return true }
	statusPanelGetActiveCountFunc = func() int { return 0 }

	opts := StatusPanelOptions{
		Format:  "compact",
		Enabled: true,
	}
	output, err := Run(opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if output != "" {
		t.Errorf("Expected empty output when no active notifications, got %q", output)
	}
}

func TestRunCompactFormat(t *testing.T) {
	originalEnsureTmuxRunningFunc := statusPanelEnsureTmuxRunningFunc
	originalGetActiveCountFunc := statusPanelGetActiveCountFunc
	originalListNotificationsFunc := statusPanelListNotificationsFunc
	originalGetConfigStringFunc := statusPanelGetConfigStringFunc
	defer func() {
		statusPanelEnsureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		statusPanelGetActiveCountFunc = originalGetActiveCountFunc
		statusPanelListNotificationsFunc = originalListNotificationsFunc
		statusPanelGetConfigStringFunc = originalGetConfigStringFunc
	}()

	statusPanelEnsureTmuxRunningFunc = func() bool { return true }
	statusPanelGetActiveCountFunc = func() int { return 3 }
	statusPanelListNotificationsFunc = func(stateFilter string) string {
		// Simulate 2 info, 1 warning
		return "1\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\tmsg1\t1234567890\tinfo\n" +
			"2\t2025-02-04T10:01:00Z\tactive\t$0\t%0\t:0.0\tmsg2\t1234567890\tinfo\n" +
			"3\t2025-02-04T10:02:00Z\tactive\t$0\t%0\t:0.0\tmsg3\t1234567890\twarning"
	}
	statusPanelGetConfigStringFunc = func(key, defaultValue string) string {
		if key == "level_colors" {
			return "info:green,warning:yellow,error:red,critical:magenta"
		}
		return defaultValue
	}

	opts := StatusPanelOptions{
		Format:  "compact",
		Enabled: true,
	}
	output, err := Run(opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	// Highest severity is warning, color yellow
	expected := "#[fg=yellow]🔔 3#[default]"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestRunCompactFormatNoColor(t *testing.T) {
	originalEnsureTmuxRunningFunc := statusPanelEnsureTmuxRunningFunc
	originalGetActiveCountFunc := statusPanelGetActiveCountFunc
	originalListNotificationsFunc := statusPanelListNotificationsFunc
	originalGetConfigStringFunc := statusPanelGetConfigStringFunc
	defer func() {
		statusPanelEnsureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		statusPanelGetActiveCountFunc = originalGetActiveCountFunc
		statusPanelListNotificationsFunc = originalListNotificationsFunc
		statusPanelGetConfigStringFunc = originalGetConfigStringFunc
	}()

	statusPanelEnsureTmuxRunningFunc = func() bool { return true }
	statusPanelGetActiveCountFunc = func() int { return 1 }
	statusPanelListNotificationsFunc = func(stateFilter string) string {
		return "1\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\tmsg1\t1234567890\tinfo"
	}
	statusPanelGetConfigStringFunc = func(key, defaultValue string) string {
		// No color mapping for info
		return ""
	}

	opts := StatusPanelOptions{
		Format:  "compact",
		Enabled: true,
	}
	output, err := Run(opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := "🔔 1"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestRunDetailedFormat(t *testing.T) {
	originalEnsureTmuxRunningFunc := statusPanelEnsureTmuxRunningFunc
	originalGetActiveCountFunc := statusPanelGetActiveCountFunc
	originalListNotificationsFunc := statusPanelListNotificationsFunc
	originalGetConfigStringFunc := statusPanelGetConfigStringFunc
	defer func() {
		statusPanelEnsureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		statusPanelGetActiveCountFunc = originalGetActiveCountFunc
		statusPanelListNotificationsFunc = originalListNotificationsFunc
		statusPanelGetConfigStringFunc = originalGetConfigStringFunc
	}()

	statusPanelEnsureTmuxRunningFunc = func() bool { return true }
	statusPanelGetActiveCountFunc = func() int { return 4 }
	statusPanelListNotificationsFunc = func(stateFilter string) string {
		return "1\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\tmsg1\t1234567890\tinfo\n" +
			"2\t2025-02-04T10:01:00Z\tactive\t$0\t%0\t:0.0\tmsg2\t1234567890\twarning\n" +
			"3\t2025-02-04T10:02:00Z\tactive\t$0\t%0\t:0.0\tmsg3\t1234567890\terror\n" +
			"4\t2025-02-04T10:03:00Z\tactive\t$0\t%0\t:0.0\tmsg4\t1234567890\tcritical"
	}
	statusPanelGetConfigStringFunc = func(key, defaultValue string) string {
		if key == "level_colors" {
			return "info:green,warning:yellow,error:red,critical:magenta"
		}
		return defaultValue
	}

	opts := StatusPanelOptions{
		Format:  "detailed",
		Enabled: true,
	}
	output, err := Run(opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	// Expect order: info, warning, error, critical
	expected := "#[fg=green]i:1#[default] #[fg=yellow]w:1#[default] #[fg=red]e:1#[default] #[fg=magenta]c:1#[default]"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestRunDetailedFormatSomeZero(t *testing.T) {
	originalEnsureTmuxRunningFunc := statusPanelEnsureTmuxRunningFunc
	originalGetActiveCountFunc := statusPanelGetActiveCountFunc
	originalListNotificationsFunc := statusPanelListNotificationsFunc
	originalGetConfigStringFunc := statusPanelGetConfigStringFunc
	defer func() {
		statusPanelEnsureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		statusPanelGetActiveCountFunc = originalGetActiveCountFunc
		statusPanelListNotificationsFunc = originalListNotificationsFunc
		statusPanelGetConfigStringFunc = originalGetConfigStringFunc
	}()

	statusPanelEnsureTmuxRunningFunc = func() bool { return true }
	statusPanelGetActiveCountFunc = func() int { return 2 }
	statusPanelListNotificationsFunc = func(stateFilter string) string {
		return "1\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t:0.0\tmsg1\t1234567890\tinfo\n" +
			"2\t2025-02-04T10:01:00Z\tactive\t$0\t%0\t:0.0\tmsg2\t1234567890\terror"
	}
	statusPanelGetConfigStringFunc = func(key, defaultValue string) string {
		if key == "level_colors" {
			return "info:green,warning:yellow,error:red,critical:magenta"
		}
		return defaultValue
	}

	opts := StatusPanelOptions{
		Format:  "detailed",
		Enabled: true,
	}
	output, err := Run(opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	// Expect only info and error
	expected := "#[fg=green]i:1#[default] #[fg=red]e:1#[default]"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestRunCountOnlyFormat(t *testing.T) {
	originalEnsureTmuxRunningFunc := statusPanelEnsureTmuxRunningFunc
	originalGetActiveCountFunc := statusPanelGetActiveCountFunc
	defer func() {
		statusPanelEnsureTmuxRunningFunc = originalEnsureTmuxRunningFunc
		statusPanelGetActiveCountFunc = originalGetActiveCountFunc
	}()

	statusPanelEnsureTmuxRunningFunc = func() bool { return true }
	statusPanelGetActiveCountFunc = func() int { return 7 }

	opts := StatusPanelOptions{
		Format:  "count-only",
		Enabled: true,
	}
	output, err := Run(opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := "7"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestRunUnknownFormat(t *testing.T) {
	opts := StatusPanelOptions{
		Format:  "invalid",
		Enabled: true,
	}
	_, err := Run(opts)
	if err == nil {
		t.Error("Expected error for unknown format")
	}
	if !errors.Is(err, err) && err.Error() != "unknown format: invalid" {
		t.Errorf("Expected 'unknown format: invalid', got %v", err)
	}
}

func TestGetCountsByLevelError(t *testing.T) {
	originalListNotificationsFunc := statusPanelListNotificationsFunc
	defer func() { statusPanelListNotificationsFunc = originalListNotificationsFunc }()

	statusPanelListNotificationsFunc = func(stateFilter string) string {
		// Simulate empty lines
		return ""
	}
	info, warning, error, critical, err := getCountsByLevel()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if info != 0 || warning != 0 || error != 0 || critical != 0 {
		t.Errorf("Expected all zero counts, got %d %d %d %d", info, warning, error, critical)
	}
}

func TestParseLevelColors(t *testing.T) {
	originalGetConfigStringFunc := statusPanelGetConfigStringFunc
	defer func() { statusPanelGetConfigStringFunc = originalGetConfigStringFunc }()

	statusPanelGetConfigStringFunc = func(key, defaultValue string) string {
		return "info:green,warning:yellow,error:red,critical:magenta"
	}
	m := parseLevelColors()
	if m["info"] != "green" {
		t.Errorf("Expected info->green, got %s", m["info"])
	}
	if m["warning"] != "yellow" {
		t.Errorf("Expected warning->yellow, got %s", m["warning"])
	}
	if m["error"] != "red" {
		t.Errorf("Expected error->red, got %s", m["error"])
	}
	if m["critical"] != "magenta" {
		t.Errorf("Expected critical->magenta, got %s", m["critical"])
	}
}

func TestParseLevelColorsEmpty(t *testing.T) {
	originalGetConfigStringFunc := statusPanelGetConfigStringFunc
	defer func() { statusPanelGetConfigStringFunc = originalGetConfigStringFunc }()

	statusPanelGetConfigStringFunc = func(key, defaultValue string) string {
		return ""
	}
	m := parseLevelColors()
	if len(m) != 0 {
		t.Errorf("Expected empty map, got %v", m)
	}
}

func TestParseLevelColorsMalformed(t *testing.T) {
	originalGetConfigStringFunc := statusPanelGetConfigStringFunc
	defer func() { statusPanelGetConfigStringFunc = originalGetConfigStringFunc }()

	statusPanelGetConfigStringFunc = func(key, defaultValue string) string {
		return "info:green,warning"
	}
	m := parseLevelColors()
	// Only first pair parsed
	if m["info"] != "green" {
		t.Errorf("Expected info->green, got %s", m["info"])
	}
	if _, ok := m["warning"]; ok {
		t.Errorf("Expected warning not in map")
	}
}

func TestGetLevelColor(t *testing.T) {
	originalGetConfigStringFunc := statusPanelGetConfigStringFunc
	defer func() { statusPanelGetConfigStringFunc = originalGetConfigStringFunc }()

	statusPanelGetConfigStringFunc = func(key, defaultValue string) string {
		return "info:green,warning:yellow"
	}
	color := getLevelColor("info")
	if color != "green" {
		t.Errorf("Expected green, got %s", color)
	}
	color = getLevelColor("warning")
	if color != "yellow" {
		t.Errorf("Expected yellow, got %s", color)
	}
	color = getLevelColor("error")
	if color != "" {
		t.Errorf("Expected empty color for missing level, got %s", color)
	}
}
