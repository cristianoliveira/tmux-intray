package settings

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
)

const (
	legacySettingsFilename = "settings" + FileExtTOML
	tuiSettingsFilename    = "tui" + FileExtTOML
)

// getSettingsPath returns the filesystem path for the TUI settings file.
// It respects the optional tui_settings_path override and migrates legacy
// settings.toml files to tui.toml automatically.
func getSettingsPath() string {
	if override := config.Get("tui_settings_path", ""); override != "" {
		return override
	}

	configDir := resolveConfigDir()
	settingsPath := filepath.Join(configDir, tuiSettingsFilename)
	migrateLegacySettings(configDir, settingsPath)
	return settingsPath
}

// resolveConfigDir returns the configured tmux-intray config directory,
// falling back to the XDG default if needed.
func resolveConfigDir() string {
	configDir := config.Get("config_dir", "")
	if configDir != "" {
		return configDir
	}
	home, _ := os.UserHomeDir()
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(home, ".config")
	}
	return filepath.Join(xdgConfigHome, "tmux-intray")
}

// migrateLegacySettings copies ~/.config/tmux-intray/settings.toml to
// tui.toml and removes the legacy file when successful.
func migrateLegacySettings(configDir, targetPath string) {
	legacyPath := filepath.Join(configDir, legacySettingsFilename)
	if legacyPath == targetPath {
		return
	}
	legacyInfo, err := os.Stat(legacyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			colors.Warning(fmt.Sprintf("Unable to inspect legacy settings path %s: %v", legacyPath, err))
		}
		return
	}
	if legacyInfo.IsDir() {
		colors.Warning(fmt.Sprintf("Legacy settings path %s is a directory, skipping migration", legacyPath))
		return
	}

	if _, err := os.Stat(targetPath); err == nil {
		colors.Warning("Found both settings.toml and tui.toml; leaving legacy settings.toml untouched")
		return
	} else if err != nil && !os.IsNotExist(err) {
		colors.Warning(fmt.Sprintf("Unable to inspect tui settings path %s: %v", targetPath, err))
		return
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), FileModeDir); err != nil {
		colors.Warning(fmt.Sprintf("Unable to create directory for %s: %v", targetPath, err))
		return
	}

	if err := copySettingsFile(legacyPath, targetPath, legacyInfo.Mode()); err != nil {
		colors.Warning(fmt.Sprintf("Failed to migrate legacy settings to %s: %v", targetPath, err))
		return
	}

	if err := os.Remove(legacyPath); err != nil {
		colors.Warning(fmt.Sprintf("Migrated settings to %s but could not remove %s: %v", targetPath, legacyPath, err))
		return
	}

	colors.Info(fmt.Sprintf("Migrated settings from %s to %s", legacyPath, targetPath))
}

func copySettingsFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	fileMode := FileModeFile
	if mode != 0 {
		fileMode = mode
	}
	return os.WriteFile(dst, data, fileMode)
}
