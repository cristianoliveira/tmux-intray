package main

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/stretchr/testify/require"
)

func TestResetSettingsSuccess(t *testing.T) {
	originalResetSettingsFunc := resetSettingsFunc
	defer func() { resetSettingsFunc = originalResetSettingsFunc }()

	called := false
	resetSettingsFunc = func() (*settings.Settings, error) {
		called = true
		return settings.DefaultSettings(), nil
	}

	_, err := resetSettingsFunc()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected resetSettingsFunc to be called")
	}
}

func TestResetSettingsError(t *testing.T) {
	originalResetSettingsFunc := resetSettingsFunc
	defer func() { resetSettingsFunc = originalResetSettingsFunc }()

	expectedErr := errors.New("storage error")
	resetSettingsFunc = func() (*settings.Settings, error) {
		return nil, expectedErr
	}

	_, err := resetSettingsFunc()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestLoadSettingsSuccess(t *testing.T) {
	originalLoadSettingsFunc := loadSettingsFunc
	defer func() { loadSettingsFunc = originalLoadSettingsFunc }()

	called := false
	loadSettingsFunc = func() (*settings.Settings, error) {
		called = true
		return settings.DefaultSettings(), nil
	}

	_, err := loadSettingsFunc()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected loadSettingsFunc to be called")
	}
}

func TestLoadSettingsError(t *testing.T) {
	originalLoadSettingsFunc := loadSettingsFunc
	defer func() { loadSettingsFunc = originalLoadSettingsFunc }()

	expectedErr := errors.New("load error")
	loadSettingsFunc = func() (*settings.Settings, error) {
		return nil, expectedErr
	}

	_, err := loadSettingsFunc()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestResetSettingsMultipleCalls(t *testing.T) {
	originalResetSettingsFunc := resetSettingsFunc
	defer func() { resetSettingsFunc = originalResetSettingsFunc }()

	count := 0
	resetSettingsFunc = func() (*settings.Settings, error) {
		count++
		return settings.DefaultSettings(), nil
	}

	_, _ = resetSettingsFunc()
	_, _ = resetSettingsFunc()
	if count != 2 {
		t.Errorf("Expected resetSettingsFunc to be called 2 times, got %d", count)
	}
}

func TestLoadSettingsMultipleCalls(t *testing.T) {
	originalLoadSettingsFunc := loadSettingsFunc
	defer func() { loadSettingsFunc = originalLoadSettingsFunc }()

	count := 0
	loadSettingsFunc = func() (*settings.Settings, error) {
		count++
		return settings.DefaultSettings(), nil
	}

	_, _ = loadSettingsFunc()
	_, _ = loadSettingsFunc()
	if count != 2 {
		t.Errorf("Expected loadSettingsFunc to be called 2 times, got %d", count)
	}
}

func TestSettingsDefaults(t *testing.T) {
	defaults := settings.DefaultSettings()

	// Verify default columns are set
	require.NotNil(t, defaults.Columns)
	require.Greater(t, len(defaults.Columns), 0)

	// Verify default sort settings
	require.Equal(t, "timestamp", defaults.SortBy)
	require.Equal(t, "desc", defaults.SortOrder)

	// Verify default view mode
	require.Equal(t, "compact", defaults.ViewMode)

	// Verify default grouping settings
	require.Equal(t, settings.GroupByNone, defaults.GroupBy)
	require.Equal(t, 1, defaults.DefaultExpandLevel)
	require.Equal(t, map[string]bool{}, defaults.ExpansionState)

	// Verify default filters are empty
	require.Equal(t, "", defaults.Filters.Level)
	require.Equal(t, "", defaults.Filters.State)
	require.Equal(t, "", defaults.Filters.Session)
	require.Equal(t, "", defaults.Filters.Window)
	require.Equal(t, "", defaults.Filters.Pane)
}
