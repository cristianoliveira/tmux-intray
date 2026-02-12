package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/pelletier/go-toml/v2"
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
	assert.Equal(t, ViewModeGrouped, s.ViewMode)

	// Check grouping settings
	assert.Equal(t, GroupByNone, s.GroupBy)
	assert.Equal(t, 1, s.DefaultExpandLevel)
	assert.Equal(t, map[string]bool{}, s.ExpansionState)
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
	assert.Equal(t, expected.GroupBy, settings.GroupBy)
	assert.Equal(t, expected.DefaultExpandLevel, settings.DefaultExpandLevel)
	assert.Equal(t, expected.ExpansionState, settings.ExpansionState)
}

func TestLoadFromExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create settings file with custom values
	configDir := filepath.Join(tmpDir, "tmux-intray")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	settingsPath := filepath.Join(configDir, "settings.toml")
	customSettings := &Settings{
		Columns:   []string{ColumnID, ColumnMessage, ColumnLevel},
		SortBy:    SortByLevel,
		SortOrder: SortOrderAsc,
		Filters: Filter{
			Level:   LevelFilterWarning,
			State:   StateFilterActive,
			Session: "my-session",
		},
		ViewMode:           ViewModeDetailed,
		GroupBy:            GroupByWindow,
		DefaultExpandLevel: 2,
		ExpansionState: map[string]bool{
			"window:@1": true,
		},
	}

	data, err := toml.Marshal(customSettings)
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
	assert.Equal(t, GroupByWindow, settings.GroupBy)
	assert.Equal(t, 2, settings.DefaultExpandLevel)
	assert.Equal(t, map[string]bool{"window:@1": true}, settings.ExpansionState)
}

func TestLoadPartialSettings(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create settings file with only some fields
	configDir := filepath.Join(tmpDir, "tmux-intray")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	settingsPath := filepath.Join(configDir, "settings.toml")
	partialTOML := `sortBy = "level"
viewMode = "detailed"
`
	require.NoError(t, os.WriteFile(settingsPath, []byte(partialTOML), 0644))
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
	assert.Equal(t, GroupByNone, settings.GroupBy)
	assert.Equal(t, 1, settings.DefaultExpandLevel)
	assert.Equal(t, map[string]bool{}, settings.ExpansionState)
}

func TestLoadInvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create settings file with invalid TOML
	configDir := filepath.Join(tmpDir, "tmux-intray")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	settingsPath := filepath.Join(configDir, "settings.toml")
	require.NoError(t, os.WriteFile(settingsPath, []byte("invalid toml [unclosed"), 0644))
	defer os.Remove(settingsPath)

	// Load should succeed with defaults (not error) - corrupted TOML is handled gracefully
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
	assert.Equal(t, expected.GroupBy, settings.GroupBy)
	assert.Equal(t, expected.DefaultExpandLevel, settings.DefaultExpandLevel)
	assert.Equal(t, expected.ExpansionState, settings.ExpansionState)
}

func TestLoadInvalidExpansionStateType(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, "tmux-intray")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	settingsPath := filepath.Join(configDir, "settings.toml")
	invalidTOML := `viewMode = "grouped"
groupBy = "window"
[expansionState]
"window:$1:@1" = "collapsed"
`
	require.NoError(t, os.WriteFile(settingsPath, []byte(invalidTOML), 0644))
	defer os.Remove(settingsPath)

	settings, err := Load()
	require.NoError(t, err)
	require.NotNil(t, settings)

	expected := DefaultSettings()
	assert.Equal(t, expected.ViewMode, settings.ViewMode)
	assert.Equal(t, expected.GroupBy, settings.GroupBy)
	assert.Equal(t, expected.ExpansionState, settings.ExpansionState)
}

func TestLoadInvalidColumn(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create settings file with invalid column name
	configDir := filepath.Join(tmpDir, "tmux-intray")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	settingsPath := filepath.Join(configDir, "settings.toml")
	invalidTOML := `columns = ["id", "invalid_column"]
`
	require.NoError(t, os.WriteFile(settingsPath, []byte(invalidTOML), 0644))
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
		ViewMode:           ViewModeDetailed,
		GroupBy:            GroupBySession,
		DefaultExpandLevel: 2,
		ExpansionState: map[string]bool{
			"session:$1": true,
		},
	}

	// Save settings
	err := Save(settings)
	require.NoError(t, err)

	// Verify file exists
	configDir := filepath.Join(tmpDir, "tmux-intray")
	settingsPath := filepath.Join(configDir, "settings.toml")
	require.FileExists(t, settingsPath)

	// Verify file contents
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var loaded Settings
	err = toml.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, settings.Columns, loaded.Columns)
	assert.Equal(t, settings.SortBy, loaded.SortBy)
	assert.Equal(t, settings.SortOrder, loaded.SortOrder)
	assert.Equal(t, settings.Filters, loaded.Filters)
	assert.Equal(t, settings.ViewMode, loaded.ViewMode)
	assert.Equal(t, settings.GroupBy, loaded.GroupBy)
	assert.Equal(t, settings.DefaultExpandLevel, loaded.DefaultExpandLevel)
	assert.Equal(t, settings.ExpansionState, loaded.ExpansionState)
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
		Columns:            []string{ColumnID},
		SortBy:             SortByTimestamp,
		SortOrder:          SortOrderDesc,
		ViewMode:           ViewModeCompact,
		GroupBy:            GroupByNone,
		DefaultExpandLevel: 1,
	}
	err := Save(settings1)
	require.NoError(t, err)

	// Save different settings
	settings2 := &Settings{
		Columns:            []string{ColumnID, ColumnMessage},
		SortBy:             SortByLevel,
		SortOrder:          SortOrderAsc,
		ViewMode:           ViewModeDetailed,
		GroupBy:            GroupByWindow,
		DefaultExpandLevel: 2,
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
	assert.Equal(t, GroupByWindow, loaded.GroupBy)
	assert.Equal(t, 2, loaded.DefaultExpandLevel)
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
				Columns:            []string{ColumnID, ColumnMessage, ColumnLevel},
				SortBy:             SortByLevel,
				SortOrder:          SortOrderAsc,
				ViewMode:           ViewModeDetailed,
				GroupBy:            GroupBySession,
				DefaultExpandLevel: 2,
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
				ViewMode:           ViewModeCompact,
				GroupBy:            GroupByPane,
				DefaultExpandLevel: 3,
			},
		},
		{
			name: "grouped view mode",
			settings: &Settings{
				Columns:            DefaultColumns,
				SortBy:             SortByTimestamp,
				SortOrder:          SortOrderDesc,
				ViewMode:           ViewModeGrouped,
				GroupBy:            GroupBySession,
				DefaultExpandLevel: 1,
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
				GroupBy:   "",
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
		{
			name: "invalid groupBy",
			settings: &Settings{
				GroupBy: "invalid",
			},
			wantErr: "invalid groupBy value",
		},
		{
			name: "invalid defaultExpandLevel",
			settings: &Settings{
				DefaultExpandLevel: 5,
			},
			wantErr: "invalid defaultExpandLevel value",
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
	expected := filepath.Join(tmpDir, "tmux-intray", "settings.toml")
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
	expected := filepath.Join(tmpDir, ".config", "tmux-intray", "settings.toml")
	assert.Equal(t, expected, path)
}

func TestSettingsTOMLMarshaling(t *testing.T) {
	settings := DefaultSettings()

	data, err := toml.Marshal(settings)
	require.NoError(t, err)

	// Verify TOML can be unmarshaled back
	var loaded Settings
	err = toml.Unmarshal(data, &loaded)
	require.NoError(t, err)

	// Verify all fields are preserved
	assert.Equal(t, settings.Columns, loaded.Columns)
	assert.Equal(t, settings.SortBy, loaded.SortBy)
	assert.Equal(t, settings.SortOrder, loaded.SortOrder)
	assert.Equal(t, settings.Filters, loaded.Filters)
	assert.Equal(t, settings.ViewMode, loaded.ViewMode)
	assert.Equal(t, settings.GroupBy, loaded.GroupBy)
	assert.Equal(t, settings.DefaultExpandLevel, loaded.DefaultExpandLevel)
	// ExpansionState may be nil after unmarshal (empty map doesn't serialize)
	if loaded.ExpansionState == nil {
		loaded.ExpansionState = map[string]bool{}
	}
	assert.Equal(t, settings.ExpansionState, loaded.ExpansionState)
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
	assert.Equal(t, GroupByNone, settings.GroupBy)
	assert.Equal(t, 1, settings.DefaultExpandLevel)
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

	settingsPath := filepath.Join(configDir, "settings.toml")
	customSettings := &Settings{
		SortBy:             SortByLevel,
		SortOrder:          SortOrderAsc,
		ViewMode:           ViewModeDetailed,
		GroupBy:            GroupByWindow,
		DefaultExpandLevel: 2,
	}

	data, err := toml.Marshal(customSettings)
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
	assert.Equal(t, GroupByWindow, settings.GroupBy)
	assert.Equal(t, 2, settings.DefaultExpandLevel)
}

func TestReset(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Create and save custom settings
	settingsPath := filepath.Join(tmpDir, "tmux-intray", "settings.toml")
	customSettings := &Settings{
		SortBy:             SortByLevel,
		SortOrder:          SortOrderAsc,
		ViewMode:           ViewModeDetailed,
		GroupBy:            GroupByWindow,
		DefaultExpandLevel: 2,
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
	assert.Equal(t, ViewModeGrouped, defaults.ViewMode)
	assert.Equal(t, GroupByNone, defaults.GroupBy)
	assert.Equal(t, 1, defaults.DefaultExpandLevel)
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
	assert.Equal(t, ViewModeGrouped, defaults.ViewMode)
	assert.Equal(t, GroupByNone, defaults.GroupBy)
	assert.Equal(t, 1, defaults.DefaultExpandLevel)
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
