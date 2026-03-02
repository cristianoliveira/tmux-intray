// Package app provides TUI application adapters for command wiring.
package app

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/settings"
)

// mockSettingsLoader is a test double for SettingsLoader.
type mockSettingsLoader struct {
	settings *settings.Settings
	err      error
}

func (m *mockSettingsLoader) Load() (*settings.Settings, error) {
	return m.settings, m.err
}

// TestDefaultClient_LoadSettings_WithInjectedLoader verifies that DefaultClient
// uses the injected SettingsLoader instead of calling settings.Load directly.
func TestDefaultClient_LoadSettings_WithInjectedLoader(t *testing.T) {
	expectedSettings := &settings.Settings{
		SortBy:    "timestamp",
		SortOrder: "desc",
	}
	mockLoader := &mockSettingsLoader{
		settings: expectedSettings,
		err:      nil,
	}

	client := NewDefaultClient(nil, nil, mockLoader)

	result, err := client.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v, want nil", err)
	}

	if result.SortBy != expectedSettings.SortBy {
		t.Errorf("LoadSettings() SortBy = %v, want %v", result.SortBy, expectedSettings.SortBy)
	}
}

// TestDefaultClient_LoadSettings_WithLoaderError verifies that DefaultClient
// propagates errors from the injected SettingsLoader.
func TestDefaultClient_LoadSettings_WithLoaderError(t *testing.T) {
	expectedErr := errors.New("failed to load settings")
	mockLoader := &mockSettingsLoader{
		settings: nil,
		err:      expectedErr,
	}

	client := NewDefaultClient(nil, nil, mockLoader)

	_, err := client.LoadSettings()
	if err != expectedErr {
		t.Fatalf("LoadSettings() error = %v, want %v", err, expectedErr)
	}
}

// TestNewDefaultClient_BackwardCompatibility verifies that passing nil for
// settingsLoader results in a DefaultSettingsLoader being used, maintaining
// backward compatibility.
func TestNewDefaultClient_BackwardCompatibility(t *testing.T) {
	client := NewDefaultClient(nil, nil, nil)

	if client.settingsLoader == nil {
		t.Fatal("NewDefaultClient() settingsLoader is nil, want DefaultSettingsLoader")
	}

	// Verify it's a DefaultSettingsLoader by calling Load
	_, err := client.settingsLoader.Load()
	if err != nil {
		// This is expected if settings file doesn't exist, but it shouldn't panic
		// The important thing is that it doesn't panic and returns an error
	}
}
