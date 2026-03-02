package core

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCore_Default(t *testing.T) {
	t.Run("creates_instance_with_defaults", func(t *testing.T) {
		c := Default()
		require.NotNil(t, c)
		require.NotNil(t, c.client)
		require.NotNil(t, c.storage)
		require.NotNil(t, c.settings)
	})
}

func TestCore_Version(t *testing.T) {
	t.Run("returns_version_string", func(t *testing.T) {
		c := NewCore(nil, nil)
		version := c.Version()

		assert.NotEmpty(t, version)
		// Version should be in format like "1.2.3" or "dev"
		assert.NotContains(t, version, "<")
		assert.NotContains(t, version, ">")
	})
}

func TestCore_DismissNotification(t *testing.T) {
	setupStorage(t)

	t.Run("dismiss_existing_notification", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := tmpDir + "/notifications.db"
		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		require.NoError(t, err)
		defer sqliteStorage.Close()

		c := NewCore(nil, sqliteStorage)

		// Add a notification
		id, err := c.AddTrayItem("test message", "$1", "%1", "@1", "123", true, "info")
		require.NoError(t, err)

		// Dismiss it
		err = c.DismissNotification(id)
		require.NoError(t, err)

		// Verify it's dismissed
		line, err := c.GetNotificationByID(id)
		require.NoError(t, err)
		assert.Contains(t, line, "dismissed")
	})

	t.Run("dismiss_nonexistent_notification", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := tmpDir + "/notifications.db"
		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		require.NoError(t, err)
		defer sqliteStorage.Close()

		c := NewCore(nil, sqliteStorage)

		// Try to dismiss non-existent notification
		err = c.DismissNotification("nonexistent")
		require.Error(t, err)
	})
}

func TestCore_DismissAll(t *testing.T) {
	setupStorage(t)

	t.Run("dismiss_all_notifications", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := tmpDir + "/notifications.db"
		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		require.NoError(t, err)
		defer sqliteStorage.Close()

		c := NewCore(nil, sqliteStorage)

		// Add multiple notifications
		_, err = c.AddTrayItem("msg1", "$1", "%1", "@1", "123", true, "info")
		require.NoError(t, err)
		_, err = c.AddTrayItem("msg2", "$2", "%2", "@2", "456", true, "warning")
		require.NoError(t, err)
		_, err = c.AddTrayItem("msg3", "$3", "%3", "@3", "789", true, "error")
		require.NoError(t, err)

		// Verify active count
		count := c.GetActiveCount()
		assert.Equal(t, 3, count)

		// Dismiss all
		err = c.DismissAll()
		require.NoError(t, err)

		// Verify all dismissed
		count = c.GetActiveCount()
		assert.Equal(t, 0, count)
	})

	t.Run("dismiss_all_when_empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := tmpDir + "/notifications.db"
		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		require.NoError(t, err)
		defer sqliteStorage.Close()

		c := NewCore(nil, sqliteStorage)

		// Try to dismiss when no notifications
		err = c.DismissAll()
		require.NoError(t, err)

		// Verify still empty
		count := c.GetActiveCount()
		assert.Equal(t, 0, count)
	})
}

func TestCore_GetTrayItems_EdgeCases(t *testing.T) {
	setupStorage(t)

	t.Run("empty_result", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := tmpDir + "/notifications.db"
		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		require.NoError(t, err)
		defer sqliteStorage.Close()

		c := NewCore(nil, sqliteStorage)

		// Get tray items when empty
		items, err := c.GetTrayItems("active")
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("filter_by_dismissed_state", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := tmpDir + "/notifications.db"
		sqliteStorage, err := sqlite.NewSQLiteStorage(dbPath)
		require.NoError(t, err)
		defer sqliteStorage.Close()

		c := NewCore(nil, sqliteStorage)

		// Add notification and dismiss it
		id, err := c.AddTrayItem("test", "$1", "%1", "@1", "123", true, "info")
		require.NoError(t, err)
		err = c.DismissNotification(id)
		require.NoError(t, err)

		// Get active items should be empty
		items, err := c.GetTrayItems("active")
		require.NoError(t, err)
		assert.Empty(t, items)

		// Get dismissed items should have it
		items, err = c.GetTrayItems("dismissed")
		require.NoError(t, err)
		assert.NotEmpty(t, items)
		assert.Contains(t, items, "test")
	})
}

func TestCore_ResetSettings_LoadSettings_EdgeCases(t *testing.T) {
	t.Run("load_settings_with_uninitialized_settings_store", func(t *testing.T) {
		c := NewCore(nil, nil)
		// Force nil settings store
		c.settings = nil

		// Should create default settings store
		settings, err := c.LoadSettings()
		require.NoError(t, err)
		assert.NotNil(t, settings)
	})

	t.Run("reset_settings_with_uninitialized_settings_store", func(t *testing.T) {
		c := NewCore(nil, nil)
		// Force nil settings store
		c.settings = nil

		// Should create default settings store
		settings, err := c.ResetSettings()
		require.NoError(t, err)
		assert.NotNil(t, settings)
	})

	t.Run("load_settings_with_custom_store", func(t *testing.T) {
		customStore := &stubSettingsStore{
			loadResult: &settings.Settings{ShowHelp: true},
			loadErr:    nil,
		}

		c := NewCore(nil, nil)
		c.settings = customStore

		// Should use custom store
		result, err := c.LoadSettings()
		require.NoError(t, err)
		assert.True(t, result.ShowHelp)
	})

	t.Run("load_settings_with_error", func(t *testing.T) {
		customStore := &stubSettingsStore{
			loadResult: nil,
			loadErr:    assert.AnError,
		}

		c := NewCore(nil, nil)
		c.settings = customStore

		// Should return error
		_, err := c.LoadSettings()
		require.Error(t, err)
	})

	t.Run("reset_settings_with_custom_store", func(t *testing.T) {
		customStore := &stubSettingsStore{
			resetResult: &settings.Settings{ShowHelp: false},
			resetErr:    nil,
		}

		c := NewCore(nil, nil)
		c.settings = customStore

		// Should use custom store
		result, err := c.ResetSettings()
		require.NoError(t, err)
		assert.False(t, result.ShowHelp)
	})

	t.Run("load_settings_with_error", func(t *testing.T) {
		customStore := &stubSettingsStore{
			loadResult: nil,
			loadErr:    assert.AnError,
		}

		c := NewCore(nil, nil)
		c.settings = customStore

		// Should return error
		_, err := c.LoadSettings()
		require.Error(t, err)
	})

	t.Run("reset_settings_with_custom_store", func(t *testing.T) {
		customStore := &stubSettingsStore{
			resetResult: &settings.Settings{ShowHelp: false},
			resetErr:    nil,
		}

		c := NewCore(nil, nil)
		c.settings = customStore

		// Should use custom store
		result, err := c.ResetSettings()
		require.NoError(t, err)
		assert.False(t, result.ShowHelp)
	})

	t.Run("reset_settings_with_error", func(t *testing.T) {
		customStore := &stubSettingsStore{
			resetResult: nil,
			resetErr:    assert.AnError,
		}

		c := NewCore(nil, nil)
		c.settings = customStore

		// Should return error
		_, err := c.ResetSettings()
		require.Error(t, err)
	})
}

func TestCore_GetVisibility_EdgeCases(t *testing.T) {
	t.Run("falls_back_to_default_on_empty_visibility", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("GetEnvironment", "TMUX_INTRAY_VISIBLE").Return("", nil).Once()

		c := NewCore(mockClient, nil)
		visible, err := c.GetVisibility()

		require.NoError(t, err)
		assert.Equal(t, "0", visible) // Should fall back to "0"
		mockClient.AssertExpectations(t)
	})

	t.Run("tmux_error_returns_default", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("GetEnvironment", "TMUX_INTRAY_VISIBLE").Return("", tmux.ErrTmuxNotRunning).Once()

		c := NewCore(mockClient, nil)
		visible, err := c.GetVisibility()

		require.NoError(t, err)
		assert.Equal(t, "0", visible) // Should fall back to "0"
		mockClient.AssertExpectations(t)
	})
}

func TestCore_SetVisibility_EdgeCases(t *testing.T) {
	t.Run("set_to_true", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "1").Return(nil).Once()

		c := NewCore(mockClient, nil)
		err := c.SetVisibility(true)

		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("set_to_false", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "0").Return(nil).Once()

		c := NewCore(mockClient, nil)
		err := c.SetVisibility(false)

		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("tmux_error", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("SetEnvironment", "TMUX_INTRAY_VISIBLE", "1").Return(tmux.ErrTmuxNotRunning).Once()

		c := NewCore(mockClient, nil)
		err := c.SetVisibility(true)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "set tmux visibility")
		mockClient.AssertExpectations(t)
	})
}

func TestCore_NewCoreWithDeps(t *testing.T) {
	t.Run("with_all_dependencies", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		customStore := &stubSettingsStore{
			loadResult:  "settings",
			resetResult: "settings",
		}

		c := NewCoreWithDeps(mockClient, nil, customStore)
		require.NotNil(t, c)
		assert.Equal(t, mockClient, c.client)
		assert.Equal(t, customStore, c.settings)
	})

	t.Run("with_nil_client_creates_default", func(t *testing.T) {
		c := NewCoreWithDeps(nil, nil, nil)
		require.NotNil(t, c)
		require.NotNil(t, c.client)
	})

	t.Run("with_nil_storage_creates_default", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
		storage.Reset()

		c := NewCoreWithDeps(nil, nil, nil)
		require.NotNil(t, c)
		require.NotNil(t, c.storage)
	})

	t.Run("with_nil_settings_creates_default", func(t *testing.T) {
		c := NewCoreWithDeps(nil, nil, nil)
		require.NotNil(t, c)
		require.NotNil(t, c.settings)
	})
}

func setupStorage(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	storage.Reset()
}
