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

func setupSettingsTest(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	config.Load()

	return filepath.Join(tmpDir, "tmux-intray")
}

func TestSettingsJSONRoundTrip(t *testing.T) {
	original := &Settings{
		Columns:   []string{ColumnID, ColumnMessage, ColumnLevel},
		SortBy:    SortByLevel,
		SortOrder: SortOrderAsc,
		Filters: Filter{
			Level:   LevelFilterWarning,
			State:   StateFilterActive,
			Read:    ReadFilterUnread,
			Session: "session-1",
			Window:  "@1",
			Pane:    "%1",
		},
		ViewMode:  ViewModeDetailed,
		ActiveTab: TabAll,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Settings
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original.Columns, decoded.Columns)
	assert.Equal(t, original.SortBy, decoded.SortBy)
	assert.Equal(t, original.SortOrder, decoded.SortOrder)
	assert.Equal(t, original.Filters, decoded.Filters)
	assert.Equal(t, original.ViewMode, decoded.ViewMode)
	assert.Equal(t, original.ActiveTab, decoded.ActiveTab)
}

func TestSettingsValidation(t *testing.T) {
	tests := []struct {
		name     string
		settings *Settings
		wantErr  bool
	}{
		{
			name:     "valid defaults",
			settings: DefaultSettings(),
			wantErr:  false,
		},
		{
			name: "invalid column",
			settings: &Settings{
				Columns: []string{"bad"},
			},
			wantErr: true,
		},
		{
			name: "invalid sort order",
			settings: &Settings{
				SortOrder: "sideways",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.settings)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestLoadDefaultFileMissing(t *testing.T) {
	configDir := setupSettingsTest(t)
	settingsPath := filepath.Join(configDir, tuiSettingsFilename)

	_, err := os.Stat(settingsPath)
	require.True(t, os.IsNotExist(err))

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, DefaultSettings(), loaded)
}

func TestLoadCorruptedTOML(t *testing.T) {
	configDir := setupSettingsTest(t)
	require.NoError(t, os.MkdirAll(configDir, FileModeDir))

	settingsPath := filepath.Join(configDir, tuiSettingsFilename)
	require.NoError(t, os.WriteFile(settingsPath, []byte("invalid = ["), FileModeFile))

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, DefaultSettings(), loaded)
}

func TestSaveLoad(t *testing.T) {
	setupSettingsTest(t)

	original := &Settings{
		Columns:   []string{ColumnID, ColumnMessage},
		SortBy:    SortByLevel,
		SortOrder: SortOrderAsc,
		Filters: Filter{
			Level: LevelFilterError,
		},
		ViewMode:  ViewModeDetailed,
		ActiveTab: TabAll,
	}

	require.NoError(t, Save(original))

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, original.Columns, loaded.Columns)
	assert.Equal(t, original.SortBy, loaded.SortBy)
	assert.Equal(t, original.SortOrder, loaded.SortOrder)
	assert.Equal(t, original.Filters, loaded.Filters)
	assert.Equal(t, original.ViewMode, loaded.ViewMode)
	assert.Equal(t, original.ActiveTab, loaded.ActiveTab)
}

func TestSaveWithLocking(t *testing.T) {
	configDir := setupSettingsTest(t)
	lockDir := configDir + ".lock"

	require.NoError(t, os.MkdirAll(lockDir, FileModeDir))

	require.NoError(t, Save(DefaultSettings()))

	_, err := os.Stat(lockDir)
	assert.True(t, os.IsNotExist(err))
}

func TestValidateSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings *Settings
		wantErr  bool
	}{
		{
			name:     "valid settings",
			settings: DefaultSettings(),
			wantErr:  false,
		},
		{
			name: "invalid filter level",
			settings: &Settings{
				Filters: Filter{Level: "invalid"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.settings)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
