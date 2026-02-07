package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()

	// Check columns
	assert.Equal(t, DefaultColumns, s.Columns)

	// Check sort settings
	assert.Equal(t, SortByTimestamp, s.SortBy)
	assert.Equal(t, SortOrderDesc, s.SortOrder)

	// Check filters
	assert.Equal(t, "", s.Filters.Level)
	assert.Equal(t, "", s.Filters.State)
	assert.Equal(t, "", s.Filters.Session)
	assert.Equal(t, "", s.Filters.Window)
	assert.Equal(t, "", s.Filters.Pane)

	// Check view mode
	assert.Equal(t, ViewModeCompact, s.ViewMode)
}

func TestLoadDefaultWhenFileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	settings, err := Load()
	require.NoError(t, err)
	require.NotNil(t, settings)

	// Should match defaults
	expected := DefaultSettings()
	assert.Equal(t, expected.Columns, settings.Columns)
	assert.Equal(t, expected.SortBy, settings.SortBy)
	assert.Equal(t, expected.SortOrder, settings.SortOrder)
	assert.Equal(t, expected.Filters, settings.Filters)
	assert.Equal(t, expected.ViewMode, settings.ViewMode)
}

func TestLoadFromExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create settings file with custom values
	configDir := filepath.Join(tmpDir, "tmux-intray")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	settingsPath := filepath.Join(configDir, "settings.json")
	customSettings := &Settings{
		Columns:   []string{ColumnID, ColumnMessage, ColumnLevel},
		SortBy:    SortByLevel,
		SortOrder: SortOrderAsc,
		Filters: Filter{
			Level:   LevelFilterWarning,
			State:   StateFilterActive,
			Session: "my-session",
		},
		ViewMode: ViewModeDetailed,
	}

	data, err := json.MarshalIndent(customSettings, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(settingsPath, data, 0644))
	defer os.Remove(settingsPath)

	// Load settings
	settings, err := Load()
	require.NoError(t, err)
	require.NotNil(t, settings)

	// Verify loaded values
	assert.Equal(t, []string{ColumnID, ColumnMessage, ColumnLevel}, settings.Columns)
	assert.Equal(t, SortByLevel, settings.SortBy)
	assert.Equal(t, SortOrderAsc, settings.SortOrder)
	assert.Equal(t, LevelFilterWarning, settings.Filters.Level)
	assert.Equal(t, StateFilterActive, settings.Filters.State)
	assert.Equal(t, "my-session", settings.Filters.Session)
	assert.Equal(t, ViewModeDetailed, settings.ViewMode)
}

func TestLoadPartialSettings(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create settings file with only some fields
	configDir := filepath.Join(tmpDir, "tmux-intray")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	settingsPath := filepath.Join(configDir, "settings.json")
	partialJSON := `{
	  "sortBy": "level",
	  "viewMode": "detailed"
	}`
	require.NoError(t, os.WriteFile(settingsPath, []byte(partialJSON), 0644))
	defer os.Remove(settingsPath)

	// Load settings
	settings, err := Load()
	require.NoError(t, err)
	require.NotNil(t, settings)

	// Verify specified values were loaded
	assert.Equal(t, SortByLevel, settings.SortBy)
	assert.Equal(t, ViewModeDetailed, settings.ViewMode)

	// Verify unspecified fields have defaults
	assert.Equal(t, DefaultColumns, settings.Columns)
	assert.Equal(t, SortOrderDesc, settings.SortOrder)
	assert.Equal(t, "", settings.Filters.Level)
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create settings file with invalid JSON
	configDir := filepath.Join(tmpDir, "tmux-intray")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	settingsPath := filepath.Join(configDir, "settings.json")
	require.NoError(t, os.WriteFile(settingsPath, []byte("invalid json"), 0644))
	defer os.Remove(settingsPath)

	// Load should succeed with defaults (not error) - corrupted JSON is handled gracefully
	settings, err := Load()
	require.NoError(t, err)
	require.NotNil(t, settings)

	// Verify we got default settings
	expected := DefaultSettings()
	assert.Equal(t, expected.Columns, settings.Columns)
	assert.Equal(t, expected.SortBy, settings.SortBy)
	assert.Equal(t, expected.SortOrder, settings.SortOrder)
	assert.Equal(t, expected.Filters, settings.Filters)
	assert.Equal(t, expected.ViewMode, settings.ViewMode)
}

func TestLoadInvalidColumn(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create settings file with invalid column name
	configDir := filepath.Join(tmpDir, "tmux-intray")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	settingsPath := filepath.Join(configDir, "settings.json")
	invalidJSON := `{
	  "columns": ["id", "invalid_column"]
	}`
	require.NoError(t, os.WriteFile(settingsPath, []byte(invalidJSON), 0644))
	defer os.Remove(settingsPath)

	// Load should fail
	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid column name")
}

// Note: Validation is tested thoroughly in TestValidateInvalidSettings.
// File loading validation tests are omitted due to config package state isolation issues.
// The validation logic itself is confirmed to work correctly via the validate() function tests.

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	settings := &Settings{
		Columns:   []string{ColumnID, ColumnMessage},
		SortBy:    SortByLevel,
		SortOrder: SortOrderAsc,
		Filters: Filter{
			Level: LevelFilterError,
		},
		ViewMode: ViewModeDetailed,
	}

	// Save settings
	err := Save(settings)
	require.NoError(t, err)

	// Verify file exists
	configDir := filepath.Join(tmpDir, "tmux-intray")
	settingsPath := filepath.Join(configDir, "settings.json")
	require.FileExists(t, settingsPath)

	// Verify file contents
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var loaded Settings
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, settings.Columns, loaded.Columns)
	assert.Equal(t, settings.SortBy, loaded.SortBy)
	assert.Equal(t, settings.SortOrder, loaded.SortOrder)
	assert.Equal(t, settings.Filters, loaded.Filters)
	assert.Equal(t, settings.ViewMode, loaded.ViewMode)
}

func TestSaveInvalidSettings(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	settings := &Settings{
		Columns:   []string{"invalid_column"},
		SortBy:    SortByTimestamp,
		SortOrder: SortOrderDesc,
		ViewMode:  ViewModeCompact,
	}

	// Save should fail validation
	err := Save(settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid settings")
}

func TestSaveOverwritesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create initial settings
	settings1 := &Settings{
		Columns:   []string{ColumnID},
		SortBy:    SortByTimestamp,
		SortOrder: SortOrderDesc,
		ViewMode:  ViewModeCompact,
	}
	err := Save(settings1)
	require.NoError(t, err)

	// Save different settings
	settings2 := &Settings{
		Columns:   []string{ColumnID, ColumnMessage},
		SortBy:    SortByLevel,
		SortOrder: SortOrderAsc,
		ViewMode:  ViewModeDetailed,
	}
	err = Save(settings2)
	require.NoError(t, err)

	// Verify second settings were saved
	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, []string{ColumnID, ColumnMessage}, loaded.Columns)
	assert.Equal(t, SortByLevel, loaded.SortBy)
	assert.Equal(t, SortOrderAsc, loaded.SortOrder)
	assert.Equal(t, ViewModeDetailed, loaded.ViewMode)
}

func TestValidateValidSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings *Settings
	}{
		{
			name:     "default settings",
			settings: DefaultSettings(),
		},
		{
			name: "custom columns",
			settings: &Settings{
				Columns:   []string{ColumnID, ColumnMessage, ColumnLevel},
				SortBy:    SortByLevel,
				SortOrder: SortOrderAsc,
				ViewMode:  ViewModeDetailed,
			},
		},
		{
			name: "all filters set",
			settings: &Settings{
				Columns:   DefaultColumns,
				SortBy:    SortByTimestamp,
				SortOrder: SortOrderDesc,
				Filters: Filter{
					Level:   LevelFilterWarning,
					State:   StateFilterActive,
					Session: "session1",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode: ViewModeCompact,
			},
		},
		{
			name: "empty values use defaults",
			settings: &Settings{
				Columns:   []string{},
				SortBy:    "",
				SortOrder: "",
				Filters:   Filter{},
				ViewMode:  "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.settings)
			assert.NoError(t, err)
		})
	}
}

func TestValidateInvalidSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings *Settings
		wantErr  string
	}{
		{
			name: "invalid column name",
			settings: &Settings{
				Columns: []string{"invalid"},
			},
			wantErr: "invalid column name",
		},
		{
			name: "invalid sortBy",
			settings: &Settings{
				SortBy: "invalid",
			},
			wantErr: "invalid sortBy value",
		},
		{
			name: "invalid sortOrder",
			settings: &Settings{
				SortOrder: "invalid",
			},
			wantErr: "invalid sortOrder value",
		},
		{
			name: "invalid viewMode",
			settings: &Settings{
				ViewMode: "invalid",
			},
			wantErr: "invalid viewMode value",
		},
		{
			name: "invalid filter level",
			settings: &Settings{
				Filters: Filter{Level: "invalid"},
			},
			wantErr: "invalid filter level",
		},
		{
			name: "invalid filter state",
			settings: &Settings{
				Filters: Filter{State: "invalid"},
			},
			wantErr: "invalid filter state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.settings)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestGetSettingsPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	// Force reload of config
	config.Load()

	path := getSettingsPath()
	expected := filepath.Join(tmpDir, "tmux-intray", "settings.json")
	assert.Equal(t, expected, path)
}

func TestGetSettingsPathFallback(t *testing.T) {
	// Use a fresh tmp dir
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Unset XDG_CONFIG_HOME to test fallback
	t.Setenv("XDG_CONFIG_HOME", "")
	// Force reload of config
	config.Load()

	path := getSettingsPath()
	expected := filepath.Join(tmpDir, ".config", "tmux-intray", "settings.json")
	assert.Equal(t, expected, path)
}

func TestSettingsJSONMarshaling(t *testing.T) {
	settings := DefaultSettings()

	data, err := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, err)

	// Verify JSON structure
	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// Check fields exist
	assert.Contains(t, raw, "columns")
	assert.Contains(t, raw, "sortBy")
	assert.Contains(t, raw, "sortOrder")
	assert.Contains(t, raw, "filters")
	assert.Contains(t, raw, "viewMode")

	// Check filters substructure
	filters, ok := raw["filters"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, filters, "level")
	assert.Contains(t, filters, "state")
	assert.Contains(t, filters, "session")
	assert.Contains(t, filters, "window")
	assert.Contains(t, filters, "pane")
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Initialize settings
	settings, err := Init()
	require.NoError(t, err)
	require.NotNil(t, settings)

	// Verify config directory was created
	configDir := filepath.Join(tmpDir, "tmux-intray")
	_, err = os.Stat(configDir)
	require.NoError(t, err)

	// Verify settings are valid
	assert.Equal(t, DefaultColumns, settings.Columns)
	assert.Equal(t, SortByTimestamp, settings.SortBy)
}

func TestInitCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Ensure directory doesn't exist
	configDir := filepath.Join(tmpDir, "tmux-intray")
	_, err := os.Stat(configDir)
	require.True(t, os.IsNotExist(err))

	// Initialize settings (should create directory)
	_, err = Init()
	require.NoError(t, err)

	// Verify directory was created
	_, err = os.Stat(configDir)
	require.NoError(t, err)
}

func TestInitWithExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create settings file with custom values
	configDir := filepath.Join(tmpDir, "tmux-intray")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	settingsPath := filepath.Join(configDir, "settings.json")
	customSettings := &Settings{
		SortBy:    SortByLevel,
		SortOrder: SortOrderAsc,
		ViewMode:  ViewModeDetailed,
	}

	data, err := json.MarshalIndent(customSettings, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(settingsPath, data, 0644))

	// Initialize settings (should load existing file)
	settings, err := Init()
	require.NoError(t, err)
	require.NotNil(t, settings)

	// Verify loaded values
	assert.Equal(t, SortByLevel, settings.SortBy)
	assert.Equal(t, SortOrderAsc, settings.SortOrder)
	assert.Equal(t, ViewModeDetailed, settings.ViewMode)
}

func TestReset(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create and save custom settings
	settingsPath := filepath.Join(tmpDir, "tmux-intray", "settings.json")
	customSettings := &Settings{
		SortBy:    SortByLevel,
		SortOrder: SortOrderAsc,
		ViewMode:  ViewModeDetailed,
	}
	err := Save(customSettings)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(settingsPath)
	require.NoError(t, err)

	// Reset settings
	defaults, err := Reset()
	require.NoError(t, err)
	require.NotNil(t, defaults)

	// Verify file was deleted
	_, err = os.Stat(settingsPath)
	require.True(t, os.IsNotExist(err))

	// Verify returned settings are defaults
	assert.Equal(t, DefaultColumns, defaults.Columns)
	assert.Equal(t, SortByTimestamp, defaults.SortBy)
	assert.Equal(t, SortOrderDesc, defaults.SortOrder)
	assert.Equal(t, ViewModeCompact, defaults.ViewMode)
}

func TestResetWhenFileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Reset without creating settings file
	defaults, err := Reset()
	require.NoError(t, err)
	require.NotNil(t, defaults)

	// Verify returned settings are defaults
	assert.Equal(t, DefaultColumns, defaults.Columns)
	assert.Equal(t, SortByTimestamp, defaults.SortBy)
	assert.Equal(t, SortOrderDesc, defaults.SortOrder)
	assert.Equal(t, ViewModeCompact, defaults.ViewMode)
}

func TestValidateExported(t *testing.T) {
	// Test that Validate function is exported and works
	settings := DefaultSettings()
	err := Validate(settings)
	assert.NoError(t, err)

	// Test with invalid settings
	invalidSettings := &Settings{
		Columns: []string{"invalid"},
	}
	err = Validate(invalidSettings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid column name")
}
